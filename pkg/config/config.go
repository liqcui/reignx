package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Server        ServerConfig
	Database      DatabaseConfig
	NATS          NATSConfig
	ETCD          ETCDConfig
	Security      SecurityConfig
	Observability ObservabilityConfig
	Agent         AgentConfig
}

// ServerConfig contains server-specific configuration
type ServerConfig struct {
	GRPCPort      int
	HTTPPort      int
	MetricsPort   int
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	IdleTimeout   time.Duration
	MaxRecvMsgSize int
	MaxSendMsgSize int
}

// DatabaseConfig contains database connection configuration
type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// NATSConfig contains NATS connection configuration
type NATSConfig struct {
	URL                 string
	ClusterID           string
	ClientID            string
	MaxReconnects       int
	ReconnectWait       time.Duration
	ConnectionTimeout   time.Duration
	EnableJetStream     bool
	JetStreamMaxWait    time.Duration
}

// ETCDConfig contains etcd connection configuration
type ETCDConfig struct {
	Endpoints   []string
	DialTimeout time.Duration
	Username    string
	Password    string
}

// SecurityConfig contains security-related configuration
type SecurityConfig struct {
	TLSEnabled        bool
	CertFile          string
	KeyFile           string
	CAFile            string
	JWTSecret         string
	JWTExpiry         time.Duration
	JWTRefreshExpiry  time.Duration
	EnableMTLS        bool
	EncryptionKey     string // AES-256 key for encrypting sensitive data (32 bytes base64 encoded)
}

// ObservabilityConfig contains observability configuration
type ObservabilityConfig struct {
	Logging LoggingConfig
	Metrics MetricsConfig
	Tracing TracingConfig
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level      string
	Format     string // json or console
	OutputPath string
}

// MetricsConfig contains metrics configuration
type MetricsConfig struct {
	Enabled bool
	Port    int
	Path    string
}

// TracingConfig contains tracing configuration
type TracingConfig struct {
	Enabled      bool
	JaegerURL    string
	ServiceName  string
	SamplingRate float64
}

// AgentConfig contains agent-specific configuration
type AgentConfig struct {
	ServerID          string
	AgentID           string
	APIServerAddr     string
	HeartbeatInterval time.Duration
	ReconnectInterval time.Duration
	TaskConcurrency   int
	CacheDir          string
}

// Load loads configuration from files and environment variables
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Read config file if provided
	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Environment variables take precedence
	v.AutomaticEnv()
	v.SetEnvPrefix("BM")

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.grpcport", 50051)
	v.SetDefault("server.httpport", 8080)
	v.SetDefault("server.metricsport", 9090)
	v.SetDefault("server.readtimeout", 30*time.Second)
	v.SetDefault("server.writetimeout", 30*time.Second)
	v.SetDefault("server.idletimeout", 60*time.Second)
	v.SetDefault("server.maxrecvmsgsize", 16*1024*1024) // 16MB
	v.SetDefault("server.maxsendmsgsize", 16*1024*1024) // 16MB

	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "reignx")
	v.SetDefault("database.password", "") // Must be set via env var or config file
	v.SetDefault("database.database", "reignx")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.maxopenconns", 100)
	v.SetDefault("database.maxidleconns", 10)
	v.SetDefault("database.connmaxlifetime", 1*time.Hour)

	// NATS defaults
	v.SetDefault("nats.url", "nats://localhost:4222")
	v.SetDefault("nats.clusterid", "bm-cluster")
	v.SetDefault("nats.clientid", "bm-client")
	v.SetDefault("nats.maxreconnects", 10)
	v.SetDefault("nats.reconnectwait", 2*time.Second)
	v.SetDefault("nats.connectiontimeout", 10*time.Second)
	v.SetDefault("nats.enablejetstream", true)
	v.SetDefault("nats.jetstreammaxwait", 5*time.Second)

	// ETCD defaults
	v.SetDefault("etcd.endpoints", []string{"localhost:2379"})
	v.SetDefault("etcd.dialtimeout", 5*time.Second)

	// Security defaults
	v.SetDefault("security.tlsenabled", false)
	v.SetDefault("security.jwtexpiry", 1*time.Hour)
	v.SetDefault("security.jwtrefreshexpiry", 7*24*time.Hour)
	v.SetDefault("security.enablemtls", false)

	// Logging defaults
	v.SetDefault("observability.logging.level", "info")
	v.SetDefault("observability.logging.format", "json")
	v.SetDefault("observability.logging.outputpath", "stdout")

	// Metrics defaults
	v.SetDefault("observability.metrics.enabled", true)
	v.SetDefault("observability.metrics.port", 9090)
	v.SetDefault("observability.metrics.path", "/metrics")

	// Tracing defaults
	v.SetDefault("observability.tracing.enabled", false)
	v.SetDefault("observability.tracing.jaegerurl", "http://localhost:14268/api/traces")
	v.SetDefault("observability.tracing.servicename", "bm-solution")
	v.SetDefault("observability.tracing.samplingrate", 0.1)

	// Agent defaults
	v.SetDefault("agent.apiserveraddr", "localhost:50051")
	v.SetDefault("agent.heartbeatinterval", 30*time.Second)
	v.SetDefault("agent.reconnectinterval", 5*time.Second)
	v.SetDefault("agent.taskconcurrency", 5)
	v.SetDefault("agent.cachedir", "/var/lib/bm-agent")
}

// GetDatabaseDSN returns the PostgreSQL connection string

func (c *DatabaseConfig) GetDSN() string {
	// Use URI format for better compatibility
	if c.Password != "" {
		return fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?sslmode=%s",
			c.User, c.Password, c.Host, c.Port, c.Database, c.SSLMode,
		)
	}
	// No password
	return fmt.Sprintf(
		"postgres://%s@%s:%d/%s?sslmode=%s",
		c.User, c.Host, c.Port, c.Database, c.SSLMode,
	)
}
