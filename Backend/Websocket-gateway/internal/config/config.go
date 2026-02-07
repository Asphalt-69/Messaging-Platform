package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server struct {
		Host                 string        `mapstructure:"host"`
		Port                 int           `mapstructure:"port"`
		ReadBufferSize       int           `mapstructure:"read_buffer_size"`
		WriteBufferSize      int           `mapstructure:"write_buffer_size"`
		MaxMessageSize       int64         `mapstructure:"max_message_size"`
		WriteWait            time.Duration `mapstructure:"write_wait"`
		PongWait             time.Duration `mapstructure:"pong_wait"`
		PingPeriod           time.Duration `mapstructure:"ping_period"`
		MaxConnsPerIP        int           `mapstructure:"max_conns_per_ip"`
		GracefulShutdownWait time.Duration `mapstructure:"graceful_shutdown_wait"`
	} `mapstructure:"server"`

	Auth struct {
		JWTSecret           string        `mapstructure:"jwt_secret"`
		TokenExpiry         time.Duration `mapstructure:"token_expiry"`
		AuthTimeout         time.Duration `mapstructure:"auth_timeout"`
		RequireAuthOnConnect bool         `mapstructure:"require_auth_on_connect"`
	} `mapstructure:"auth"`

	Cluster struct {
		NodeID              string        `mapstructure:"node_id"`
		ServiceDiscoveryURL string        `mapstructure:"service_discovery_url"`
		HeartbeatInterval   time.Duration `mapstructure:"heartbeat_interval"`
		StickySessionSecret string        `mapstructure:"sticky_session_secret"`
	} `mapstructure:"cluster"`

	Redis struct {
		Addresses           []string      `mapstructure:"addresses"`
		Password            string        `mapstructure:"password"`
		DB                  int           `mapstructure:"db"`
		PoolSize            int           `mapstructure:"pool_size"`
		MinIdleConns        int           `mapstructure:"min_idle_conns"`
		PubSubChannelPrefix string        `mapstructure:"pubsub_channel_prefix"`
	} `mapstructure:"redis"`

	NATS struct {
		URLs                []string      `mapstructure:"urls"`
		StreamName          string        `mapstructure:"stream_name"`
		ConsumerName        string        `mapstructure:"consumer_name"`
		DurableConsumer     bool          `mapstructure:"durable_consumer"`
	} `mapstructure:"nats"`

	RateLimit struct {
		MessagesPerSecond   int           `mapstructure:"messages_per_second"`
		Burst               int           `mapstructure:"burst"`
		ConnectionsPerUser  int           `mapstructure:"connections_per_user"`
		GlobalConnections   int           `mapstructure:"global_connections"`
	} `mapstructure:"rate_limit"`

	Observability struct {
		MetricsPort         int           `mapstructure:"metrics_port"`
		LogLevel            string        `mapstructure:"log_level"`
		EnableTracing       bool          `mapstructure:"enable_tracing"`
		OtelEndpoint        string        `mapstructure:"otel_endpoint"`
	} `mapstructure:"observability"`

	Sharding struct {
		ShardCount          int           `mapstructure:"shard_count"`
		ShardKey            string        `mapstructure:"shard_key"`
	} `mapstructure:"sharding"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/websocket-gateway")

	// Set defaults
	setDefaults()

	// Read from environment variables
	viper.AutomaticEnv()

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate config
	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func setDefaults() {
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.read_buffer_size", 4096)
	viper.SetDefault("server.write_buffer_size", 4096)
	viper.SetDefault("server.max_message_size", 512*1024) // 512KB
	viper.SetDefault("server.write_wait", 10*time.Second)
	viper.SetDefault("server.pong_wait", 60*time.Second)
	viper.SetDefault("server.ping_period", 54*time.Second) // 90% of pong_wait
	viper.SetDefault("server.max_conns_per_ip", 10)
	viper.SetDefault("server.graceful_shutdown_wait", 30*time.Second)

	viper.SetDefault("auth.auth_timeout", 5*time.Second)
	viper.SetDefault("auth.require_auth_on_connect", true)

	viper.SetDefault("cluster.node_id", generateNodeID())
	viper.SetDefault("cluster.heartbeat_interval", 5*time.Second)

	viper.SetDefault("redis.addresses", []string{"localhost:6379"})
	viper.SetDefault("redis.pool_size", 100)
	viper.SetDefault("redis.min_idle_conns", 10)
	viper.SetDefault("redis.pubsub_channel_prefix", "ws-gateway")

	viper.SetDefault("nats.stream_name", "messaging")
	viper.SetDefault("nats.consumer_name", "websocket-gateway")
	viper.SetDefault("nats.durable_consumer", true)

	viper.SetDefault("rate_limit.messages_per_second", 100)
	viper.SetDefault("rate_limit.burst", 150)
	viper.SetDefault("rate_limit.connections_per_user", 5)
	viper.SetDefault("rate_limit.global_connections", 1000000)

	viper.SetDefault("observability.metrics_port", 9090)
	viper.SetDefault("observability.log_level", "info")
	viper.SetDefault("observability.enable_tracing", false)

	viper.SetDefault("sharding.shard_count", 64)
	viper.SetDefault("sharding.shard_key", "user_id")
}

func generateNodeID() string {
	hostname, _ := os.Hostname()
	return hostname + "-" + strconv.Itoa(os.Getpid())
}

func validateConfig(cfg *Config) error {
	if cfg.Auth.JWTSecret == "" {
		return fmt.Errorf("jwt_secret is required")
	}
	
	if cfg.Server.MaxMessageSize > 10*1024*1024 {
		return fmt.Errorf("max_message_size cannot exceed 10MB")
	}
	
	if cfg.Sharding.ShardCount <= 0 {
		return fmt.Errorf("shard_count must be positive")
	}
	
	if cfg.RateLimit.GlobalConnections <= 0 {
		return fmt.Errorf("global_connections limit must be positive")
	}
	
	return nil
}
