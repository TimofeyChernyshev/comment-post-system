package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	// Server configuration
	Port            string        `env:"PORT,required"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"30s"`

	// Storage configuration
	StorageType          StorageType   `env:"STORAGE_TYPE" envDefault:"memory"` // "memory" or "postgres"
	DatabaseURL          string        `env:"DATABASE_URL"`
	DBConntectionTimeout time.Duration `env:"DATABASE_CONNECTION_TIMEOUT" envDefault:"30s"`

	// Domain business rules (configurable limits)
	DomainLimits DomainLimits `envPrefix:"DOMAIN_"`

	// Subscription configuration (optional)
	SubscriptionConfig SubscriptionConfig `envPrefix:"SUBSCRIPTION_"`

	// GraphQL configuration
	GraphQLConfig GraphQLConfig `envPrefix:"GRAPHQL_"`

	// HTTP server configuration
	HTTPServerConfig HTTPServerConfig `envPrefix:"HTTP_SERVER_"`

	// Logging configuration
	Logging LoggingConfig `envPrefix:"LOG_"`

	// Dataloader configuration
	Dataloader DataloaderConfig `envPrefix:"DATALOADER_"`
}

type StorageType string

const (
	MemoryStorage   StorageType = "memory"
	PostgresStorage StorageType = "postgres"
)

// DomainLimits contains configurable business rules
type DomainLimits struct {
	MaxCommentLength     int `env:"MAX_COMMENT_LENGTH" envDefault:"2000"`
	MaxPostTitleLength   int `env:"MAX_POST_TITLE_LENGTH" envDefault:"200"`
	MaxPostContentLength int `env:"MAX_POST_CONTENT_LENGTH" envDefault:"10000"`
	MaxCommentPageSize   int `env:"MAX_COMMENT_PAGE_SIZE" envDefault:"100"`
	MaxPostPageSize      int `env:"MAX_POST_PAGE_SIZE" envDefault:"100"`
}

// SubscriptionConfig for WebSocket/GraphQL subscriptions
type SubscriptionConfig struct {
	BufferSize int `env:"BUFFER_SIZE" envDefault:"100"`
}

// GraphQLConfig for GraphQL server
type GraphQLConfig struct {
	MaxComplexity         int           `env:"MAX_COMPLEXITY" envDefault:"7"`
	QueryCacheSize        int           `env:"QUERY_CACHE_SIZE" envDefault:"100"`
	APQCacheSize          int           `env:"APQ_CACHE_SIZE" envDefault:"1000"`
	HandshakeTimeout      time.Duration `env:"HANDSHAKE_TIMEOUT" envDefault:"10s"`
	KeepAlivePingInterval time.Duration `env:"KEEP_ALIVE_PING_INTERVAL" envDefault:"30s"`
}

type HTTPServerConfig struct {
	ReadTimeout  time.Duration `env:"READ_TIMEOUT" envDefault:"20s"`
	WriteTimeout time.Duration `env:"WRITE_TIMEOUT" envDefault:"20s"`
	IdleTimeout  time.Duration `env:"IDLE_TIMEOUT" envDefault:"20s"`
}

type LoggingConfig struct {
	Level  string `env:"LEVEL" envDefault:"info"`
	Format string `env:"FORMAT" envDefault:"text"`
}

type DataloaderConfig struct {
	BatchCapacity int           `env:"BATCH_CAPACITY" envDefault:"1000"`
	WaitTime      time.Duration `env:"WAIT_TIME" envDefault:"2ms"`
}

func Load() (*Config, error) {
	var cfg Config

	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.StorageType != MemoryStorage && c.StorageType != PostgresStorage {
		return errors.New("invalid STORAGE_TYPE: must be 'memory' or 'postgres'")
	}

	if c.StorageType == PostgresStorage && c.DatabaseURL == "" {
		return errors.New("postgres storage requires DATABASE_URL")
	}

	if c.DomainLimits.MaxCommentLength <= 0 {
		return errors.New("DOMAIN_MAX_COMMENT_LENGTH must be positive")
	}

	return nil
}
