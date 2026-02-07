use std::sync::Arc;
use metrics::{describe_counter, describe_gauge, describe_histogram};
use metrics_exporter_prometheus::PrometheusBuilder;
use tracing::{info, error};
use tokio::sync::RwLock;

#[derive(Clone)]
pub struct BrokerMetrics {
    inner: Arc<BrokerMetricsInner>,
}

struct BrokerMetricsInner {
    // Incoming messages
    messages_received_total: metrics::Counter,
    messages_invalid_total: metrics::Counter,
    messages_dropped_total: metrics::Counter,
    
    // Outgoing messages
    messages_sent_total: metrics::Counter,
    messages_failed_total: metrics::Counter,
    messages_queued_total: metrics::Counter,
    
    // Fanout metrics
    fanout_operations_total: metrics::Counter,
    fanout_latency_seconds: metrics::Histogram,
    fanout_recipients_per_message: metrics::Histogram,
    
    // Routing metrics
    routing_cache_hits: metrics::Counter,
    routing_cache_misses: metrics::Counter,
    routing_shard_operations: metrics::CounterVec,
    
    // NATS metrics
    nats_published_total: metrics::Counter,
    nats_consumed_total: metrics::Counter,
    nats_errors_total: metrics::Counter,
    
    // System metrics
    active_connections: metrics::Gauge,
    active_topics: metrics::Gauge,
    memory_usage_bytes: metrics::Gauge,
    cpu_usage_percent: metrics::Gauge,
    
    // Latency histograms
    ingress_latency_seconds: metrics::Histogram,
    egress_latency_seconds: metrics::Histogram,
    processing_latency_seconds: metrics::Histogram,
    
    // Rate limiting
    rate_limit_hits_total: metrics::Counter,
    backpressure_events_total: metrics::Counter,
}

impl BrokerMetrics {
    pub fn new() -> anyhow::Result<Self> {
        // Describe metrics for Prometheus
        describe_counter!(
            "broker_messages_received_total",
            "Total number of messages received"
        );
        describe_counter!(
            "broker_messages_invalid_total",
            "Total number of invalid messages rejected"
        );
        describe_counter!(
            "broker_messages_dropped_total",
            "Total number of messages dropped due to backpressure"
        );
        
        describe_counter!(
            "broker_messages_sent_total",
            "Total number of messages sent to recipients"
        );
        describe_counter!(
            "broker_messages_failed_total",
            "Total number of messages that failed to send"
        );
        describe_counter!(
            "broker_messages_queued_total",
            "Total number of messages queued for offline users"
        );
        
        describe_counter!(
            "broker_fanout_operations_total",
            "Total number of fanout operations"
        );
        describe_histogram!(
            "broker_fanout_latency_seconds",
            "Fanout operation latency in seconds",
            unit: metrics::Unit::Seconds,
            buckets: [0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0]
        );
        describe_histogram!(
            "broker_fanout_recipients_per_message",
            "Number of recipients per fanout operation",
            buckets: [1.0, 10.0, 100.0, 1000.0, 10000.0, 100000.0]
        );
        
        describe_counter!(
            "broker_routing_cache_hits",
            "Routing cache hits"
        );
        describe_counter!(
            "broker_routing_cache_misses",
            "Routing cache misses"
        );
        
        describe_counter!(
            "broker_nats_published_total",
            "Total messages published to NATS"
        );
        describe_counter!(
            "broker_nats_consumed_total",
            "Total messages consumed from NATS"
        );
        describe_counter!(
            "broker_nats_errors_total",
            "Total NATS communication errors"
        );
        
        describe_gauge!(
            "broker_active_connections",
            "Number of active connections to gateways"
        );
        describe_gauge!(
            "broker_active_topics",
            "Number of active routing topics"
        );
        describe_gauge!(
            "broker_memory_usage_bytes",
            "Memory usage in bytes"
        );
        describe_gauge!(
            "broker_cpu_usage_percent",
            "CPU usage percentage"
        );
        
        describe_histogram!(
            "broker_ingress_latency_seconds",
            "Ingress processing latency",
            unit: metrics::Unit::Seconds,
            buckets: [0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5]
        );
        describe_histogram!(
            "broker_egress_latency_seconds",
            "Egress processing latency",
            unit: metrics::Unit::Seconds,
            buckets: [0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5]
        );
        
        describe_counter!(
            "broker_rate_limit_hits_total",
            "Total rate limit hits"
        );
        describe_counter!(
            "broker_backpressure_events_total",
            "Total backpressure events"
        );
        
        let inner = BrokerMetricsInner {
            messages_received_total: metrics::counter!("broker_messages_received_total"),
            messages_invalid_total: metrics::counter!("broker_messages_invalid_total"),
            messages_dropped_total: metrics::counter!("broker_messages_dropped_total"),
            
            messages_sent_total: metrics::counter!("broker_messages_sent_total"),
            messages_failed_total: metrics::counter!("broker_messages_failed_total"),
            messages_queued_total: metrics::counter!("broker_messages_queued_total"),
            
            fanout_operations_total: metrics::counter!("broker_fanout_operations_total"),
            fanout_latency_seconds: metrics::histogram!("broker_fanout_latency_seconds"),
            fanout_recipients_per_message: metrics::histogram!("broker_fanout_recipients_per_message"),
            
            routing_cache_hits: metrics::counter!("broker_routing_cache_hits"),
            routing_cache_misses: metrics::counter!("broker_routing_cache_misses"),
            routing_shard_operations: metrics::counter_vec!("broker_routing_shard_operations", &["shard_id"]),
            
            nats_published_total: metrics::counter!("broker_nats_published_total"),
            nats_consumed_total: metrics::counter!("broker_nats_consumed_total"),
            nats_errors_total: metrics::counter!("broker_nats_errors_total"),
            
            active_connections: metrics::gauge!("broker_active_connections"),
            active_topics: metrics::gauge!("broker_active_topics"),
            memory_usage_bytes: metrics::gauge!("broker_memory_usage_bytes"),
            cpu_usage_percent: metrics::gauge!("broker_cpu_usage_percent"),
            
            ingress_latency_seconds: metrics::histogram!("broker_ingress_latency_seconds"),
            egress_latency_seconds: metrics::histogram!("broker_egress_latency_seconds"),
            processing_latency_seconds: metrics::histogram!("broker_processing_latency_seconds"),
            
            rate_limit_hits_total: metrics::counter!("broker_rate_limit_hits_total"),
            backpressure_events_total: metrics::counter!("broker_backpressure_events_total"),
        };
        
        Ok(Self {
            inner: Arc::new(inner),
        })
    }
    
    pub fn record_message_received(&self) {
        self.inner.messages_received_total.increment(1);
    }
    
    pub fn record_message_invalid(&self) {
        self.inner.messages_invalid_total.increment(1);
    }
    
    pub fn record_message_dropped(&self, reason: &str) {
        self.inner.messages_dropped_total.increment(1);
        metrics::counter!("broker_messages_dropped_reason", "reason" => reason.to_string()).increment(1);
    }
    
    pub fn record_message_sent(&self, recipient_count: u64) {
        self.inner.messages_sent_total.increment(recipient_count);
    }
    
    pub fn record_message_failed(&self, reason: &str) {
        self.inner.messages_failed_total.increment(1);
        metrics::counter!("broker_messages_failed_reason", "reason" => reason.to_string()).increment(1);
    }
    
    pub fn record_fanout_operation(&self, recipient_count: u64, latency: f64) {
        self.inner.fanout_operations_total.increment(1);
        self.inner.fanout_latency_seconds.record(latency);
        self.inner.fanout_recipients_per_message.record(recipient_count as f64);
    }
    
    pub fn record_routing_cache_hit(&self) {
        self.inner.routing_cache_hits.increment(1);
    }
    
    pub fn record_routing_cache_miss(&self) {
        self.inner.routing_cache_misses.increment(1);
    }
    
    pub fn record_nats_published(&self, count: u64) {
        self.inner.nats_published_total.increment(count);
    }
    
    pub fn record_nats_error(&self, error: &str) {
        self.inner.nats_errors_total.increment(1);
        metrics::counter!("broker_nats_error_types", "error" => error.to_string()).increment(1);
    }
    
    pub fn update_active_connections(&self, count: i64) {
        self.inner.active_connections.set(count as f64);
    }
    
    pub fn update_active_topics(&self, count: i64) {
        self.inner.active_topics.set(count as f64);
    }
    
    pub fn record_rate_limit_hit(&self, user_id: &str) {
        self.inner.rate_limit_hits_total.increment(1);
        metrics::counter!("broker_rate_limit_hits_user", "user_id" => user_id.to_string()).increment(1);
    }
    
    pub fn record_backpressure_event(&self) {
        self.inner.backpressure_events_total.increment(1);
    }
    
    pub fn record_ingress_latency(&self, latency: f64) {
        self.inner.ingress_latency_seconds.record(latency);
    }
    
    pub fn record_egress_latency(&self, latency: f64) {
        self.inner.egress_latency_seconds.record(latency);
    }
    
    pub fn start_processing_timer(&self) -> ProcessingTimer {
        ProcessingTimer::new()
    }
}

pub struct ProcessingTimer {
    start: std::time::Instant,
}

impl ProcessingTimer {
    fn new() -> Self {
        Self {
            start: std::time::Instant::now(),
        }
    }
    
    pub fn record(self) {
        let elapsed = self.start.elapsed();
        metrics::histogram!("broker_processing_latency_seconds")
            .record(elapsed.as_secs_f64());
    }
}

pub fn start_metrics_server(addr: std::net::SocketAddr) -> anyhow::Result<()> {
    let builder = PrometheusBuilder::new();
    
    tokio::spawn(async move {
        match builder.with_http_listener(addr).install() {
            Ok(_) => info!("Prometheus metrics server started on {}", addr),
            Err(e) => error!("Failed to start metrics server: {}", e),
        }
        
        // Keep the task alive
        std::future::pending::<()>().await;
    });
    
    Ok(())
                      }
