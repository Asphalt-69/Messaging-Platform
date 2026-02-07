package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/yourcompany/websocket-gateway/internal/config"
	"github.com/yourcompany/websocket-gateway/internal/connection"
	"github.com/yourcompany/websocket-gateway/internal/messaging"
	"github.com/yourcompany/websocket-gateway/internal/metrics"
	"github.com/yourcompany/websocket-gateway/internal/pubsub"
)

// WebSocketServer represents the WebSocket gateway server
type WebSocketServer struct {
	config        *config.Config
	logger        *zap.Logger
	upgrader      websocket.Upgrader
	connManager   *connection.Manager
	messageRouter *messaging.Router
	metrics       *metrics.MetricsCollector
	pubSub        pubsub.PubSub
	
	// HTTP server
	httpServer    *http.Server
	
	// Shutdown coordination
	shutdownOnce  sync.Once
	shutdownChan  chan struct{}
}

// NewWebSocketServer creates a new WebSocket server
func NewWebSocketServer(cfg *config.Config, logger *zap.Logger) (*WebSocketServer, error) {
	// Create connection manager
	connManager := connection.NewManager(cfg, logger)
	
	// Create Pub/Sub (Redis)
	pubSub, err := pubsub.NewRedisPubSub(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create pub/sub: %w", err)
	}
	
	// Subscribe to messages
	if err := pubSub.Subscribe(); err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}
	
	// Create message router
	router := messaging.NewRouter(connManager, pubSub, logger, cfg.Cluster.NodeID)
	
	// Create metrics collector
	metricsCollector := metrics.NewMetricsCollector(
		cfg.Observability.MetricsPort,
		connManager,
		logger,
	)
	
	// Configure WebSocket upgrader
	upgrader := websocket.Upgrader{
		ReadBufferSize:  cfg.Server.ReadBufferSize,
		WriteBufferSize: cfg.Server.WriteBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			// In production, implement proper origin checking
			return true
		},
		EnableCompression: true,
	}
	
	server := &WebSocketServer{
		config:        cfg,
		logger:        logger,
		upgrader:      upgrader,
		connManager:   connManager,
		messageRouter: router,
		metrics:       metricsCollector,
		pubSub:        pubSub,
		shutdownChan:  make(chan struct{}),
	}
	
	// Register handlers
	server.registerHandlers()
	
	return server, nil
}

// registerHandlers registers connection manager handlers
func (s *WebSocketServer) registerHandlers() {
	s.connManager.RegisterHandlers(
		s.handleConnect,
		s.handleDisconnect,
		s.handleMessage,
	)
}

// handleConnect handles new connections
func (s *WebSocketServer) handleConnect(client *connection.Client) {
	s.logger.Info("client connected",
		zap.String("client_id", client.ID),
		zap.String("ip", client.IP),
		zap.Uint32("shard_id", client.ShardID))
	
	s.metrics.RecordConnection(client.IP)
}

// handleDisconnect handles client disconnections
func (s *WebSocketServer) handleDisconnect(client *connection.Client, reason string) {
	s.logger.Info("client disconnected",
		zap.String("client_id", client.ID),
		zap.String("user_id", client.UserID),
		zap.String("reason", reason),
		zap.Duration("duration", time.Since(client.ConnectedAt)))
	
	if client.UserID != "" {
		// Publish offline status
		s.publishPresence(client.UserID, "offline", client.DeviceID)
	}
	
	s.metrics.RecordDisconnection(reason)
}

// handleMessage handles incoming messages
func (s *WebSocketServer) handleMessage(client *connection.Client, message []byte) error {
	return s.messageRouter.HandleMessage(client, message)
}

// ServeHTTP handles HTTP requests and upgrades to WebSocket
func (s *WebSocketServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract client IP
	ip := getClientIP(r)
	
	// Check connection limits per IP
	if !s.checkIPLimit(ip) {
		http.Error(w, "too many connections from this IP", http.StatusTooManyRequests)
		return
	}
	
	// Upgrade to WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("failed to upgrade connection", 
			zap.Error(err),
			zap.String("ip", ip))
		return
	}
	
	// Create rate limiter for this connection
	rateLimiter := rate.NewLimiter(
		rate.Limit(s.config.RateLimit.MessagesPerSecond),
		s.config.RateLimit.Burst,
	)
	
	// Add to connection manager
	_, err = s.connManager.AddConnection(conn, ip, rateLimiter, s.config.RateLimit.Burst)
	if err != nil {
		s.logger.Error("failed to add connection",
			zap.Error(err),
			zap.String("ip", ip))
		conn.Close()
		return
	}
}

// checkIPLimit checks if IP has exceeded connection limit
func (s *WebSocketServer) checkIPLimit(ip string) bool {
	// This would be implemented with a sliding window counter
	// For simplicity, we're using the connection manager's rate limiter
	return true
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (if behind proxy)
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return forwarded
	}
	
	// Fall back to remote address
	return r.RemoteAddr
}

// Start starts the WebSocket server
func (s *WebSocketServer) Start() error {
	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port),
		Handler:      s,
		ReadTimeout:  s.config.Server.WriteWait,
		WriteTimeout: s.config.Server.WriteWait,
	}
	
	s.logger.Info("starting WebSocket server",
		zap.String("host", s.config.Server.Host),
		zap.Int("port", s.config.Server.Port))
	
	// Start metrics updater
	go s.updateMetrics()
	
	// Start server
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}
	
	return nil
}

// updateMetrics periodically updates metrics
func (s *WebSocketServer) updateMetrics() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-s.shutdownChan:
			return
		case <-ticker.C:
			stats := s.connManager.GetStats()
			s.metrics.UpdateConnectionMetrics(stats)
		}
	}
}

// Shutdown gracefully shuts down the server
func (s *WebSocketServer) Shutdown() {
	s.shutdownOnce.Do(func() {
		s.logger.Info("initiating graceful shutdown")
		
		close(s.shutdownChan)
		
		// Shutdown HTTP server
		if s.httpServer != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 
				s.config.Server.GracefulShutdownWait)
			defer cancel()
			
			if err := s.httpServer.Shutdown(ctx); err != nil {
				s.logger.Error("failed to shutdown HTTP server", zap.Error(err))
			}
		}
		
		// Shutdown connection manager
		s.connManager.Shutdown()
		
		// Shutdown message router
		s.messageRouter.Shutdown()
		
		// Shutdown Pub/Sub
		s.pubSub.Shutdown()
		
		// Shutdown metrics
		s.metrics.Shutdown()
		
		s.logger.Info("graceful shutdown complete")
	})
}

// publishPresence publishes presence updates
func (s *WebSocketServer) publishPresence(userID, status, deviceID string) {
	presence := map[string]interface{}{
		"type":      "presence",
		"user_id":   userID,
		"status":    status,
		"device_id": deviceID,
		"timestamp": time.Now().UnixMilli(),
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := s.pubSub.Publish(ctx, presence); err != nil {
		s.logger.Error("failed to publish presence", 
			zap.Error(err),
			zap.String("user_id", userID))
	}
}
