package connection

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Shard represents a shard of connections
type Shard struct {
	ID           uint32
	clients      map[string]*Client
	userClients  map[string]map[string]*Client // user_id -> {device_id -> client}
	mu           sync.RWMutex
	stats        ShardStats
	logger       *zap.Logger
}

type ShardStats struct {
	TotalConnections     int64
	ActiveConnections    int64
	Disconnections       int64
	MessagesProcessed    int64
	BytesProcessed       int64
	LastCleanup          time.Time
}

// NewShard creates a new connection shard
func NewShard(id uint32, logger *zap.Logger) *Shard {
	return &Shard{
		ID:          id,
		clients:     make(map[string]*Client),
		userClients: make(map[string]map[string]*Client),
		logger:      logger.With(zap.Uint32("shard_id", id)),
	}
}

// AddClient adds a client to the shard
func (s *Shard) AddClient(client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.clients[client.ID] = client
	s.stats.TotalConnections++
	s.stats.ActiveConnections++
	
	s.logger.Debug("client added to shard",
		zap.String("client_id", client.ID),
		zap.Int("total_clients", len(s.clients)))
}

// RemoveClient removes a client from the shard
func (s *Shard) RemoveClient(clientID string) *Client {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	client, exists := s.clients[clientID]
	if !exists {
		return nil
	}
	
	delete(s.clients, clientID)
	
	// Remove from user mapping if authenticated
	if client.UserID != "" {
		if devices, ok := s.userClients[client.UserID]; ok {
			delete(devices, client.DeviceID)
			if len(devices) == 0 {
				delete(s.userClients, client.UserID)
			}
		}
	}
	
	s.stats.ActiveConnections--
	s.stats.Disconnections++
	
	s.logger.Debug("client removed from shard",
		zap.String("client_id", clientID),
		zap.Int("remaining_clients", len(s.clients)))
	
	return client
}

// GetClient retrieves a client by ID
func (s *Shard) GetClient(clientID string) *Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return s.clients[clientID]
}

// GetUserClients retrieves all clients for a user
func (s *Shard) GetUserClients(userID string) []*Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	devices, exists := s.userClients[userID]
	if !exists {
		return nil
	}
	
	clients := make([]*Client, 0, len(devices))
	for _, client := range devices {
		clients = append(clients, client)
	}
	
	return clients
}

// RegisterAuthenticatedClient registers an authenticated client
func (s *Shard) RegisterAuthenticatedClient(client *Client, userID, deviceID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	client.SetAuthenticated(userID, deviceID)
	
	// Add to user mapping
	if _, exists := s.userClients[userID]; !exists {
		s.userClients[userID] = make(map[string]*Client)
	}
	s.userClients[userID][deviceID] = client
}

// BroadcastToUser sends a message to all devices of a user
func (s *Shard) BroadcastToUser(userID string, message []byte) (int, error) {
	s.mu.RLock()
	clients := s.GetUserClients(userID)
	s.mu.RUnlock()
	
	sent := 0
	for _, client := range clients {
		if err := client.WriteMessage(message); err == nil {
			sent++
		}
	}
	
	return sent, nil
}

// CleanupInactive removes inactive clients
func (s *Shard) CleanupInactive(timeout time.Duration) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	now := time.Now()
	removed := []string{}
	
	for id, client := range s.clients {
		if now.Sub(client.LastActivity) > timeout {
			removed = append(removed, id)
			delete(s.clients, id)
			
			// Clean up user mapping
			if client.UserID != "" {
				if devices, ok := s.userClients[client.UserID]; ok {
					delete(devices, client.DeviceID)
					if len(devices) == 0 {
						delete(s.userClients, client.UserID)
					}
				}
			}
		}
	}
	
	s.stats.ActiveConnections -= int64(len(removed))
	s.stats.LastCleanup = now
	
	if len(removed) > 0 {
		s.logger.Info("cleaned up inactive clients",
			zap.Int("count", len(removed)),
			zap.Int("remaining", len(s.clients)))
	}
	
	return removed
}

// GetStats returns shard statistics
func (s *Shard) GetStats() ShardStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	stats := s.stats
	stats.ActiveConnections = int64(len(s.clients))
	
	return stats
}
