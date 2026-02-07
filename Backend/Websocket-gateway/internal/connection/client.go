package connection

import (
	"context"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/yourcompany/websocket-gateway/pkg/protocol"
)

// Client represents a single WebSocket connection
type Client struct {
	ID           string
	UserID       string
	DeviceID     string
	IP           string
	Conn         *websocket.Conn
	Send         chan []byte
	RateLimiter  *rate.Limiter
	ConnectedAt  time.Time
	LastActivity time.Time
	ShardID      uint32
	
	// Context for cancellation
	ctx        context.Context
	cancel     context.CancelFunc
	
	// Mutex for thread-safe operations
	mu         sync.RWMutex
	
	// State
	authenticated bool
	closing       bool
	
	// Metrics
	metrics     *ClientMetrics
	
	logger      *zap.Logger
}

type ClientMetrics struct {
	MessagesSent     int64
	MessagesReceived int64
	BytesSent        int64
	BytesReceived    int64
	LastPingTime     time.Time
}

// NewClient creates a new client connection
func NewClient(
	conn *websocket.Conn,
	clientID string,
	ip string,
	shardID uint32,
	rateLimit rate.Limit,
	burst int,
	logger *zap.Logger,
) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Client{
		ID:           clientID,
		IP:           ip,
		Conn:         conn,
		Send:         make(chan []byte, 256), // Buffered channel
		RateLimiter:  rate.NewLimiter(rateLimit, burst),
		ConnectedAt:  time.Now(),
		LastActivity: time.Now(),
		ShardID:      shardID,
		ctx:          ctx,
		cancel:       cancel,
		metrics:      &ClientMetrics{},
		logger:       logger.With(zap.String("client_id", clientID)),
	}
}

// SetAuthenticated marks the client as authenticated
func (c *Client) SetAuthenticated(userID, deviceID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.UserID = userID
	c.DeviceID = deviceID
	c.authenticated = true
	c.logger = c.logger.With(zap.String("user_id", userID))
}

// IsAuthenticated checks if client is authenticated
func (c *Client) IsAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.authenticated
}

// WriteMessage sends a message to the client
func (c *Client) WriteMessage(message []byte) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if c.closing {
		return ErrClientClosed
	}
	
	select {
	case c.Send <- message:
		c.metrics.MessagesSent++
		c.metrics.BytesSent += int64(len(message))
		return nil
	default:
		// Channel is full - client is too slow
		c.logger.Warn("client send channel full, dropping message",
			zap.Int("channel_size", len(c.Send)))
		return ErrClientSlow
	}
}

// ReadPump handles incoming messages from the client
func (c *Client) ReadPump(
	messageHandler func(*Client, []byte) error,
	closeHandler func(*Client),
) {
	defer func() {
		closeHandler(c)
		c.cleanup()
	}()
	
	c.Conn.SetReadLimit(1024 * 1024) // 1MB max message size
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		c.metrics.LastPingTime = time.Now()
		return nil
	})
	
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			_, message, err := c.Conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, 
					websocket.CloseGoingAway, 
					websocket.CloseAbnormalClosure,
					websocket.CloseNormalClosure) {
					c.logger.Debug("websocket read error", zap.Error(err))
				}
				return
			}
			
			c.LastActivity = time.Now()
			c.metrics.MessagesReceived++
			c.metrics.BytesReceived += int64(len(message))
			
			// Apply rate limiting
			if !c.RateLimiter.Allow() {
				c.logger.Warn("rate limit exceeded",
					zap.String("ip", c.IP))
				errMsg := protocol.NewErrorMessage(
					"RATE_LIMIT_EXCEEDED",
					"Too many messages",
					"Please slow down",
				)
				msgBytes, _ := json.Marshal(errMsg)
				c.WriteMessage(msgBytes)
				continue
			}
			
			// Process message
			if err := messageHandler(c, message); err != nil {
				c.logger.Error("error processing message", zap.Error(err))
			}
		}
	}
}

// WritePump handles outgoing messages to the client
func (c *Client) WritePump() {
	ticker := time.NewTicker(54 * time.Second) // 90% of pong wait
	defer func() {
		ticker.Stop()
		c.cleanup()
	}()
	
	for {
		select {
		case <-c.ctx.Done():
			return
			
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			
			if !ok {
				// Channel closed
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			
			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			
			w.Write(message)
			
			// Drain any pending messages
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}
			
			if err := w.Close(); err != nil {
				return
			}
			
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Close gracefully closes the client connection
func (c *Client) Close(reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.closing {
		return
	}
	
	c.closing = true
	c.logger.Info("closing client connection", 
		zap.String("reason", reason),
		zap.Duration("duration", time.Since(c.ConnectedAt)))
	
	c.cancel()
	close(c.Send)
	c.Conn.Close()
}

func (c *Client) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if !c.closing {
		c.closing = true
		c.cancel()
		close(c.Send)
		c.Conn.Close()
	}
}

// Errors
var (
	ErrClientClosed = errors.New("client connection closed")
	ErrClientSlow   = errors.New("client send channel full")
)
