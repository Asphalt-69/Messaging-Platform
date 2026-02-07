use std::{net::SocketAddr, time::Duration};
use serde::{Deserialize, Serialize};
use config::{Config, ConfigError, Environment, File};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BrokerConfig {
    pub broker_id: String,
    pub environment: String,
    
    pub nats: NatsConfig,
    pub api: ApiConfig,
    pub routing: RoutingConfig,
    pub metrics: MetricsConfig,
    pub limits: RateLimits,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NatsConfig {
    pub servers: Vec<String>,
    pub username: Option<String>,
    pub password: Option<String>,
    pub token: Option<String>,
    pub tls_cert: Option<String>,
    pub tls_key: Option<String>,
    pub tls_ca: Option<String>,
    
    // Topics for communication
    pub ingress_topic: String,
    pub egress_user_prefix: String,
    pub egress_group_prefix: String,
    pub control_topic: String,
    
    // JetStream for persistence
    pub stream_name: String,
    pub consumer_name: String,
    
    pub connect_timeout: Duration,
    pub reconnect_delay: Duration,
    pub max_reconnects: Option<usize>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApiConfig {
    pub grpc_addr: SocketAddr,
    pub rest_addr: SocketAddr,
    pub grpc_tls_cert: Option<String>,
    pub grpc_tls_key: Option<String>,
    
    pub max_concurrent_streams: u32,
    pub max_frame_size: usize,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RoutingConfig {
    pub shard_count: usize,
    pub fanout_batch_size: usize,
    pub fanout_parallelism: usize,
    
    pub presence_ttl: Duration,
    pub typing_ttl: Duration,
    
    pub cache_size: usize,
    pub bloom_filter_size: usize,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MetricsConfig {
    pub prometheus_addr: SocketAddr,
    pub log_level: String,
    pub enable_tracing: bool,
    pub otel_endpoint: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RateLimits {
    pub messages_per_second: u32,
    pub burst_size: u32,
    pub max_message_size: usize,
    pub max_recipients_per_message: usize,
    pub max_group_size: usize,
    
    pub user_message_limit: u32,
    pub user_message_window: Duration,
    pub connection_limit_per_user: u32,
}

impl BrokerConfig {
    pub fn load() -> Result<Self, ConfigError> {
        let env = std::env::var("ENVIRONMENT").unwrap_or_else(|_| "development".into());
        
        let config = Config::builder()
            .add_source(File::with_name("config/default").required(false))
            .add_source(File::with_name(&format!("config/{}", env)).required(false))
            .add_source(File::with_name("config/local").required(false))
            .add_source(Environment::with_prefix("BROKER").separator("__"))
            .set_default("broker_id", generate_broker_id())?
            .set_default("environment", env)?
            
            // NATS defaults
            .set_default("nats.servers", vec!["nats://localhost:4222"])?
            .set_default("nats.ingress_topic", "broker.ingress")?
            .set_default("nats.egress_user_prefix", "gateway.user")?
            .set_default("nats.egress_group_prefix", "gateway.group")?
            .set_default("nats.control_topic", "broker.control")?
            .set_default("nats.stream_name", "messages")?
            .set_default("nats.consumer_name", "broker-consumer")?
            .set_default("nats.connect_timeout", 5)? // seconds
            .set_default("nats.reconnect_delay", 2)? // seconds
            
            // API defaults
            .set_default("api.grpc_addr", "0.0.0.0:50051")?
            .set_default("api.rest_addr", "0.0.0.0:8080")?
            .set_default("api.max_concurrent_streams", 10000)?
            .set_default("api.max_frame_size", 1048576)? // 1MB
            
            // Routing defaults
            .set_default("routing.shard_count", 64)?
            .set_default("routing.fanout_batch_size", 100)?
            .set_default("routing.fanout_parallelism", 16)?
            .set_default("routing.presence_ttl", 300)? // 5 minutes
            .set_default("routing.typing_ttl", 10)? // 10 seconds
            .set_default("routing.cache_size", 10000)?
            .set_default("routing.bloom_filter_size", 100000)?
            
            // Metrics defaults
            .set_default("metrics.prometheus_addr", "0.0.0.0:9090")?
            .set_default("metrics.log_level", "info")?
            .set_default("metrics.enable_tracing", false)?
            
            // Rate limit defaults
            .set_default("limits.messages_per_second", 10000)?
            .set_default("limits.burst_size", 15000)?
            .set_default("limits.max_message_size", 65536)? // 64KB
            .set_default("limits.max_recipients_per_message", 1000)?
            .set_default("limits.max_group_size", 100000)? // 100K users max per group
            .set_default("limits.user_message_limit", 100)?
            .set_default("limits.user_message_window", 60)? // 1 minute
            .set_default("limits.connection_limit_per_user", 10)?
            
            .build()?;
        
        config.try_deserialize()
    }
    
    pub fn is_production(&self) -> bool {
        self.environment == "production"
    }
    
    pub fn require_tls(&self) -> bool {
        self.is_production()
    }
}

fn generate_broker_id() -> String {
    use std::time::{SystemTime, UNIX_EPOCH};
    
    let hostname = hostname::get()
        .map(|h| h.to_string_lossy().into_owned())
        .unwrap_or_else(|_| "unknown".into());
    
    let pid = std::process::id();
    let timestamp = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_secs();
    
    format!("{}-{}-{}", hostname, pid, timestamp)
  }
