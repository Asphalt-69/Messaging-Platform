package metrics

import (
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/yourcompany/websocket-gateway/internal/connection"
)

// MetricsCollector collects and exposes metrics
type MetricsCollector struct {
	// Prometheus metrics
	activeConnections     prometheus.Gauge
	totalConnections      prometheus.Counter
	messagesReceived      prometheus.Counter
	messagesSent          prometheus.Counter
	messageLatency        prometheus.Histogram
	authAttempts          prometheus.Counter
	authFailures          prometheus.Counter
	rateLimitHits         prometheus.Counter
	shardConnections      *prometheus.GaugeVec
	errorCount            *prometheus.CounterVec
	
	// Internal state
	logger     *zap.Logger
	connMgr    *connection.Manager
	server     *http.Server
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(port int, connMgr *connection.Manager, logger *zap.Logger) *MetricsCollector {
	collector := &MetricsCollector{
		logger:  logger,
		connMgr: connMgr,
	}
	
	// Register metrics
	collector.registerMetrics()
	
	// Start HTTP server for metrics
	collector.startMetricsServer(port)
	
	return collector
}

func (m *MetricsCollector) registerMetrics() {
	m.activeConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "websocket_gateway_active_connections",
		Help: "Number of active WebSocket connections",
	})
	
	m.totalConnections = promauto.NewCounter(prometheus.CounterOpts{
		Name: "websocket_gateway_total_connections",
		Help: "Total number of WebSocket connections since startup",
	})
	
	m.messagesReceived = promauto.NewCounter(prometheus.CounterOpts{
		Name: "websocket_gateway_messages_received_total",
		Help: "Total number of messages received",
	})
	
	m.messagesSent = promauto.NewCounter(prometheus.CounterOpts{
		Name: "websocket_gateway_messages_sent_total",
		Help: "Total number of messages sent",
	})
	
	m.messageLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "websocket_gateway_message_latency_seconds",
		Help:    "Message processing latency in seconds",
		Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
	})
	
	m.authAttempts = promauto.NewCounter(prometheus.CounterOpts{
		Name: "websocket_gateway_auth_attempts_total",
		Help: "Total number of authentication attempts",
	})
	
	m.authFailures = promauto.NewCounter(prometheus.CounterOpts{
		Name: "websocket_gateway_auth_failures_total",
		Help: "Total number of authentication failures",
	})
	
	m.rateLimitHits = promauto.NewCounter(prometheus.CounterOpts{
		Name: "websocket_gateway_rate_limit_hits_total",
		Help: "Total number of rate limit hits",
	})
	
	m.shardConnections = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "websocket_gateway_shard_connections",
		Help: "Number of connections per shard",
	}, []string{"shard_id"})
	
	m.errorCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "websocket_gateway_errors_total",
		Help: "Total number of errors by type",
	}, []string{"error_type"})
}

// startMetricsServer starts the Prometheus metrics server
func (m *MetricsCollector) startMetricsServer(port int) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/health", http.HandlerFunc(m.healthHandler))
	mux.Handle("/stats", http.HandlerFunc(m.statsHandler))
	
	m.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	
	go func() {
		m.logger.Info("starting metrics server", zap.Int("port", port))
		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			m.logger.Error("metrics server failed", zap.Error(err))
		}
	}()
}

// UpdateConnectionMetrics updates connection-related metrics
func (m *MetricsCollector) UpdateConnectionMetrics(stats map[uint32]connection.ShardStats) {
	totalActive := int64(0)
	
	for shardID, shardStats := range stats {
		m.shardConnections.WithLabelValues(fmt.Sprintf("%d", shardID)).
			Set(float64(shardStats.ActiveConnections))
		totalActive += shardStats.ActiveConnections
	}
	
	m.activeConnections.Set(float64(totalActive))
}

// RecordMessageReceived records a received message
func (m *MetricsCollector) RecordMessageReceived() {
	m.messagesReceived.Inc()
}

// RecordMessageSent records a sent message
func (m *MetricsCollector) RecordMessageSent() {
	m.messagesSent.Inc()
}

// RecordAuthAttempt records an authentication attempt
func (m *MetricsCollector) RecordAuthAttempt(success bool) {
	m.authAttempts.Inc()
	if !success {
		m.authFailures.Inc()
	}
}

// RecordRateLimitHit records a rate limit hit
func (m *MetricsCollector) RecordRateLimitHit() {
	m.rateLimitHits.Inc()
}

// RecordError records an error
func (m *MetricsCollector) RecordError(errorType string) {
	m.errorCount.WithLabelValues(errorType).Inc()
}

// RecordLatency records message processing latency
func (m *MetricsCollector) RecordLatency(duration time.Duration) {
	m.messageLatency.Observe(duration.Seconds())
}

// healthHandler handles health checks
func (m *MetricsCollector) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "healthy", "timestamp": "` + time.Now().Format(time.RFC3339) + `"}`))
}

// statsHandler returns gateway statistics
func (m *MetricsCollector) statsHandler(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"active_connections":   m.activeConnections,
		"total_connections":    m.totalConnections,
		"messages_received":    m.messagesReceived,
		"messages_sent":        m.messagesSent,
		"uptime":               time.Since(startTime).String(),
		"timestamp":            time.Now().Unix(),
	}
	
	json.NewEncoder(w).Encode(stats)
}

// Shutdown gracefully shuts down the metrics server
func (m *MetricsCollector) Shutdown() {
	if m.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if err := m.server.Shutdown(ctx); err != nil {
			m.logger.Error("failed to shutdown metrics server", zap.Error(err))
		}
	}
}

var startTime = time.Now()
