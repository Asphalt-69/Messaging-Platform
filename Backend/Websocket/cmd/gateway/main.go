package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourorg/websocket-gateway/internal/server"
	"github.com/yourorg/websocket-gateway/config"
	"github.com/yourorg/websocket-gateway/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger.Init(cfg.LogLevel)
	zap.L().Info("Starting WebSocket Gateway",
		zap.String("node_id", cfg.NodeID),
		zap.String("environment", cfg.Environment),
	)

	// Create server instance
	wsServer, err := server.NewWebSocketServer(cfg)
	if err != nil {
		zap.L().Fatal("Failed to create WebSocket server", zap.Error(err))
	}

	// Start metrics server in background
	go func() {
		metricsAddr := fmt.Sprintf(":%d", cfg.MetricsPort)
		zap.L().Info("Starting metrics server", zap.String("addr", metricsAddr))
		if err := wsServer.StartMetricsServer(metricsAddr); err != nil && err != http.ErrServerClosed {
			zap.L().Error("Metrics server failed", zap.Error(err))
		}
	}()

	// Start WebSocket server
	go func() {
		wsAddr := fmt.Sprintf(":%d", cfg.WebSocketPort)
		zap.L().Info("Starting WebSocket server", zap.String("addr", wsAddr))
		if err := wsServer.Start(wsAddr); err != nil && err != http.ErrServerClosed {
			zap.L().Fatal("WebSocket server failed", zap.Error(err))
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zap.L().Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := wsServer.Shutdown(ctx); err != nil {
		zap.L().Error("Server shutdown failed", zap.Error(err))
	}

	zap.L().Info("Server exited gracefully")
}
