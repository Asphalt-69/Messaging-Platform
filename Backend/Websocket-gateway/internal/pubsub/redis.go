package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/yourcompany/websocket-gateway/internal/config"
	"github.com/yourcompany/websocket-gateway/pkg/protocol"
)

// RedisPubSub implements Pub/Sub using Redis
type RedisPubSub struct {
	client     redis.UniversalClient
	channel    string
	nodeID     string
	subscriber *redis.PubSub
	logger     *zap.Logger
	
	// Message handlers
	handlers   map[string]MessageHandler
	mu         sync.RWMutex
	
	// Context for shutdown
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

type MessageHandler func(ctx context.Context, msg protocol.BaseMessage) error

// NewRedisPubSub creates a new Redis Pub/Sub instance
func NewRedisPubSub(cfg *config.Config, logger *zap.Logger) (*RedisPubSub, error) {
	// Create Redis client
	var client redis.UniversalClient
	
	if len(cfg.Redis.Addresses) > 1 {
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        cfg.Redis.Addresses,
			Password:     cfg.Redis.Password,
			PoolSize:     cfg.Redis.PoolSize,
			MinIdleConns: cfg.Redis.MinIdleConns,
		})
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:         cfg.Redis.Addresses[0],
			Password:     cfg.Redis.Password,
			DB:           cfg.Redis.DB,
			PoolSize:     cfg.Redis.PoolSize,
			MinIdleConns: cfg.Redis.MinIdleConns,
		})
	}
	
	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}
	
	ps := &RedisPubSub{
		client:   client,
		channel:  fmt.Sprintf("%s:messages", cfg.Redis.PubSubChannelPrefix),
		nodeID:   cfg.Cluster.NodeID,
		logger:   logger,
		handlers: make(map[string]MessageHandler),
	}
	
	ps.ctx, ps.cancel = context.WithCancel(context.Background())
	
	return ps, nil
}

// Publish publishes a message to Redis Pub/Sub
func (r *RedisPubSub) Publish(ctx context.Context, msg interface{}) error {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	
	// Create envelope with node ID to avoid loops
	envelope := map[string]interface{}{
		"node_id":    r.nodeID,
		"timestamp":  time.Now().UnixMilli(),
		"message":    msgBytes,
	}
	
	envelopeBytes, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("failed to marshal envelope: %w", err)
	}
	
	// Publish to Redis
	if err := r.client.Publish(ctx, r.channel, envelopeBytes).Err(); err != nil {
		return fmt.Errorf("redis publish failed: %w", err)
	}
	
	return nil
}

// Subscribe starts listening for messages
func (r *RedisPubSub) Subscribe() error {
	r.subscriber = r.client.Subscribe(r.ctx, r.channel)
	
	// Start listening for messages
	r.wg.Add(1)
	go r.listen()
	
	r.logger.Info("redis pub/sub subscribed", 
		zap.String("channel", r.channel),
		zap.String("node_id", r.nodeID))
	
	return nil
}

// RegisterHandler registers a message handler
func (r *RedisPubSub) RegisterHandler(msgType string, handler MessageHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.handlers[msgType] = handler
	r.logger.Debug("registered pub/sub handler", zap.String("message_type", msgType))
}

// listen listens for incoming messages
func (r *RedisPubSub) listen() {
	defer r.wg.Done()
	
	ch := r.subscriber.Channel()
	
	for {
		select {
		case <-r.ctx.Done():
			return
		case msg := <-ch:
			r.processMessage(msg)
		}
	}
}

func (r *RedisPubSub) processMessage(redisMsg *redis.Message) {
	var envelope struct {
		NodeID    string          `json:"node_id"`
		Timestamp int64           `json:"timestamp"`
		Message   json.RawMessage `json:"message"`
	}
	
	// Parse envelope
	if err := json.Unmarshal([]byte(redisMsg.Payload), &envelope); err != nil {
		r.logger.Error("failed to parse message envelope", zap.Error(err))
		return
	}
	
	// Skip messages from ourselves
	if envelope.NodeID == r.nodeID {
		return
	}
	
	// Parse inner message
	var baseMsg protocol.BaseMessage
	if err := json.Unmarshal(envelope.Message, &baseMsg); err != nil {
		r.logger.Error("failed to parse inner message", zap.Error(err))
		return
	}
	
	// Find handler for message type
	r.mu.RLock()
	handler, exists := r.handlers[baseMsg.Type]
	r.mu.RUnlock()
	
	if !exists {
		r.logger.Debug("no handler for message type", 
			zap.String("type", baseMsg.Type))
		return
	}
	
	// Execute handler
	ctx, cancel := context.WithTimeout(r.ctx, 5*time.Second)
	defer cancel()
	
	if err := handler(ctx, baseMsg); err != nil {
		r.logger.Error("message handler failed",
			zap.String("type", baseMsg.Type),
			zap.Error(err))
	}
}

// PublishUserMessage publishes a message to a user's channel
func (r *RedisPubSub) PublishUserMessage(ctx context.Context, userID string, msg interface{}) error {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	
	channel := fmt.Sprintf("%s:user:%s", r.channel, userID)
	
	return r.client.Publish(ctx, channel, msgBytes).Err()
}

// SubscribeToUser subscribes to messages for a specific user
func (r *RedisPubSub) SubscribeToUser(userID string, handler MessageHandler) error {
	channel := fmt.Sprintf("%s:user:%s", r.channel, userID)
	
	subscriber := r.client.Subscribe(r.ctx, channel)
	
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer subscriber.Close()
		
		ch := subscriber.Channel()
		for {
			select {
			case <-r.ctx.Done():
				return
			case msg := <-ch:
				var baseMsg protocol.BaseMessage
				if err := json.Unmarshal([]byte(msg.Payload), &baseMsg); err == nil {
					ctx, cancel := context.WithTimeout(r.ctx, 5*time.Second)
					handler(ctx, baseMsg)
					cancel()
				}
			}
		}
	}()
	
	return nil
}

// Shutdown gracefully shuts down the Pub/Sub
func (r *RedisPubSub) Shutdown() {
	r.logger.Info("shutting down redis pub/sub")
	
	r.cancel()
	
	if r.subscriber != nil {
		r.subscriber.Close()
	}
	
	r.wg.Wait()
	
	if err := r.client.Close(); err != nil {
		r.logger.Error("failed to close redis client", zap.Error(err))
	}
	
	r.logger.Info("redis pub/sub shutdown complete")
}
