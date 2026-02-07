package connection

import (
	"context"
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/yourcompany/websocket-gateway/internal/config"
)

// Manager manages all WebSocket connections
type Manager struct {
	shards      []*Shard
	shardCount  uint32
	shardMask   uint32
	config      *config.Config
	logger      *zap.Logger
	
	// Global connection counter with atomic operations
	globalConns int64
	maxConns    int64
	
	// Rate limiters
	ipLimiter   *IPRateLimiter
	userLimiter *UserRateLimiter
	
	// Context for cleanup
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	
	// Event handlers
	onConnect    func(*Client)
	onDisconnect func(*Client, string)
	onMessage    func(*Client, []byte) error
}

// NewManager creates a new connection manager
func NewManager(cfg *config.Config, logger *zap.Logger) *Manager {
	// Ensure shard count is power of two for efficient masking
	shardCount := uint32(cfg.Sharding.ShardCount)
	if shardCount == 0 {
		shardCount = 64
	}
	
	// Find next power of two
	shardCount = nextPowerOfTwo(shardCount)
	shardMask := shardCount - 1
	
	manager := &Manager{
		shards:     make([]*Shard, shardCount),
		shardCount: shardCount,
		shardMask:  shardMask,
		config:     cfg,
		logger:     logger,
		maxConns:   int64(cfg.RateLimit.GlobalConnections),
	}
	
	// Initialize shards
	for i := uint32(0); i < shardCount; i++ {
		manager.shards[i] = NewShard(i, logger)
	}
	
	// Initialize rate limiters
	manager.ipLimiter = NewIPRateLimiter(
		rate.Limit(cfg.RateLimit.ConnectionsPerUser),
		cfg.RateLimit.Burst,
	)
	
	manager.userLimiter = NewUserRateLimiter(
		rate.Limit(cfg.RateLimit.ConnectionsPerUser),
		cfg.RateLimit.ConnectionsPerUser,
	)
	
	manager.ctx, manager.cancel = context.WithCancel(context.Background())
	
	// Start background cleanup
	manager.startCleanupRoutine()
	
	return manager
}

// RegisterHandlers registers event handlers
func (m *Manager) RegisterHandlers(
	onConnect func(*Client),
	onDisconnect func(*Client, string),
	onMessage func(*Client, []byte) error,
) {
	m.onConnect = onConnect
	m.onDisconnect = onDisconnect
	m.onMessage = onMessage
}

// AddConnection adds a new WebSocket connection
func (m *Manager) AddConnection(
	conn *websocket.Conn,
	ip string,
	rateLimit rate.Limit,
	burst int,
) (*Client, error) {
	// Check global connection limit
	if atomic.LoadInt64(&m.globalConns) >= m.maxConns {
		return nil, fmt.Errorf("global connection limit reached")
	}
	
	// Check IP rate limit
	if !m.ipLimiter.Allow(ip) {
		return nil, fmt.Errorf("ip rate limit exceeded")
	}
	
	// Generate client ID
	clientID := uuid.New().String()
	
	// Determine shard based on client ID
	shardID := m.getShardID(clientID)
	shard := m.shards[shardID]
	
	// Create client
	client := NewClient(conn, clientID, ip, shardID, rateLimit, burst, m.logger)
	
	// Add to shard
	shard.AddClient(client)
	atomic.AddInt64(&m.globalConns, 1)
	
	m.logger.Info("new connection established",
		zap.String("client_id", clientID),
		zap.String("ip", ip),
		zap.Uint32("shard_id", shardID))
	
	// Start client goroutines
	m.wg.Add(2)
	go func() {
		defer m.wg.Done()
		client.ReadPump(m.onMessage, m.handleDisconnect)
	}()
	
	go func() {
		defer m.wg.Done()
		client.WritePump()
	}()
	
	// Call connect handler
	if m.onConnect != nil {
		m.onConnect(client)
	}
	
	return client, nil
}

// AuthenticateClient authenticates a client
func (m *Manager) AuthenticateClient(clientID, userID, deviceID string) error {
	shardID := m.getShardID(clientID)
	shard := m.shards[shardID]
	
	client := shard.GetClient(clientID)
	if client == nil {
		return fmt.Errorf("client not found")
	}
	
	// Check user connection limit
	if !m.userLimiter.Allow(userID) {
		return fmt.Errorf("user connection limit exceeded")
	}
	
	shard.RegisterAuthenticatedClient(client, userID, deviceID)
	
	m.logger.Info("client authenticated",
		zap.String("client_id", clientID),
		zap.String("user_id", userID),
		zap.String("device_id", deviceID))
	
	return nil
}

// SendToClient sends a message to a specific client
func (m *Manager) SendToClient(clientID string, message []byte) error {
	shardID := m.getShardID(clientID)
	shard := m.shards[shardID]
	
	client := shard.GetClient(clientID)
	if client == nil {
		return fmt.Errorf("client not found")
	}
	
	return client.WriteMessage(message)
}

// SendToUser sends a message to all devices of a user
func (m *Manager) SendToUser(userID string, message []byte) (int, error) {
	totalSent := 0
	
	// Iterate through all shards (user might be connected to multiple shards)
	for _, shard := range m.shards {
		if sent, err := shard.BroadcastToUser(userID, message); err == nil {
			totalSent += sent
		}
	}
	
	return totalSent, nil
}

// GetClient retrieves a client by ID
func (m *Manager) GetClient(clientID string) *Client {
	shardID := m.getShardID(clientID)
	return m.shards[shardID].GetClient(clientID)
}

// GetUserClients retrieves all clients for a user
func (m *Manager) GetUserClients(userID string) []*Client {
	var allClients []*Client
	
	for _, shard := range m.shards {
		clients := shard.GetUserClients(userID)
		allClients = append(allClients, clients...)
	}
	
	return allClients
}

// RemoveClient removes a client
func (m *Manager) RemoveClient(clientID, reason string) {
	shardID := m.getShardID(clientID)
	shard := m.shards[shardID]
	
	client := shard.RemoveClient(clientID)
	if client != nil {
		atomic.AddInt64(&m.globalConns, -1)
		client.Close(reason)
		
		// Call disconnect handler
		if m.onDisconnect != nil {
			m.onDisconnect(client, reason)
		}
	}
}

// handleDisconnect handles client disconnection
func (m *Manager) handleDisconnect(client *Client) {
	m.RemoveClient(client.ID, "client_disconnected")
}

// getShardID determines which shard a key belongs to
func (m *Manager) getShardID(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32() & m.shardMask
}

func nextPowerOfTwo(n uint32) uint32 {
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n++
	return n
}

// GetStats returns manager statistics
func (m *Manager) GetStats() map[uint32]ShardStats {
	stats := make(map[uint32]ShardStats)
	
	for i, shard := range m.shards {
		stats[uint32(i)] = shard.GetStats()
	}
	
	return stats
}

// Shutdown gracefully shuts down the connection manager
func (m *Manager) Shutdown() {
	m.logger.Info("shutting down connection manager")
	
	// Cancel context to stop all goroutines
	m.cancel()
	
	// Wait for all goroutines to finish
	m.wg.Wait()
	
	m.logger.Info("connection manager shutdown complete")
}

// startCleanupRoutine starts the background cleanup goroutine
func (m *Manager) startCleanupRoutine() {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		
		for {
			select {
			case <-m.ctx.Done():
				return
			case <-ticker.C:
				m.cleanupInactiveConnections()
			}
		}
	}()
}

func (m *Manager) cleanupInactiveConnections() {
	timeout := m.config.Server.PongWait * 2
	
	for _, shard := range m.shards {
		removed := shard.CleanupInactive(timeout)
		
		for _, clientID := range removed {
			atomic.AddInt64(&m.globalConns, -1)
			
			// Call disconnect handler
			if m.onDisconnect != nil {
				if client := shard.GetClient(clientID); client != nil {
					m.onDisconnect(client, "inactive_timeout")
				}
			}
		}
	}
}
