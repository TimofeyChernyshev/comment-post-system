package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ConfigTestSuite struct {
	suite.Suite
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}

func (s *ConfigTestSuite) clearEnv() {
	vars := []string{
		"PORT",
		"SHUTDOWN_TIMEOUT",
		"STORAGE_TYPE",
		"DATABASE_URL",
		"DOMAIN_MAX_COMMENT_LENGTH",
		"DOMAIN_MAX_POST_TITLE_LENGTH",
		"DOMAIN_MAX_POST_CONTENT_LENGTH",
		"DOMAIN_MAX_COMMENT_PAGE_SIZE",
		"DOMAIN_MAX_POST_PAGE_SIZE",
		"SUBSCRIPTION_BUFFER_SIZE",
		"GRAPHQL_MAX_COMPLEXITY",
		"GRAPHQL_QUERY_CACHE_SIZE",
		"GRAPHQL_APQ_CACHE_SIZE",
		"GRAPHQL_HANDSHAKE_TIMEOUT",
		"GRAPHQL_KEEP_ALIVE_PING_INTERVAL",
		"HTTP_SERVER_READ_TIMEOUT",
		"HTTP_SERVER_WRITE_TIMEOUT",
		"HTTP_SERVER_IDLE_TIMEOUT",
		"LOG_LEVEL",
		"LOG_FORMAT",
	}
	for _, v := range vars {
		os.Unsetenv(v)
	}
}

func (s *ConfigTestSuite) SetupTest() {
	s.clearEnv()
}

func (s *ConfigTestSuite) TestLoad_Defaults() {
	s.T().Setenv("PORT", "8080")

	cfg, err := Load()
	require.NoError(s.T(), err)

	assert.Equal(s.T(), "8080", cfg.Port)
	assert.Equal(s.T(), 30*time.Second, cfg.ShutdownTimeout)
	assert.Equal(s.T(), MemoryStorage, cfg.StorageType)
	assert.Equal(s.T(), "", cfg.DatabaseURL)
	assert.Equal(s.T(), 2000, cfg.DomainLimits.MaxCommentLength)
	assert.Equal(s.T(), 200, cfg.DomainLimits.MaxPostTitleLength)
	assert.Equal(s.T(), 10000, cfg.DomainLimits.MaxPostContentLength)
	assert.Equal(s.T(), 100, cfg.DomainLimits.MaxCommentPageSize)
	assert.Equal(s.T(), 100, cfg.DomainLimits.MaxPostPageSize)
	assert.Equal(s.T(), 100, cfg.SubscriptionConfig.BufferSize)
	assert.Equal(s.T(), 7, cfg.GraphQLConfig.MaxComplexity)
	assert.Equal(s.T(), 100, cfg.GraphQLConfig.QueryCacheSize)
	assert.Equal(s.T(), 1000, cfg.GraphQLConfig.APQCacheSize)
	assert.Equal(s.T(), 10*time.Second, cfg.GraphQLConfig.HandshakeTimeout)
	assert.Equal(s.T(), 30*time.Second, cfg.GraphQLConfig.KeepAlivePingInterval)
	assert.Equal(s.T(), 20*time.Second, cfg.HTTPServerConfig.ReadTimeout)
	assert.Equal(s.T(), 20*time.Second, cfg.HTTPServerConfig.WriteTimeout)
	assert.Equal(s.T(), 20*time.Second, cfg.HTTPServerConfig.IdleTimeout)
	assert.Equal(s.T(), "info", cfg.Logging.Level)
	assert.Equal(s.T(), "text", cfg.Logging.Format)
}

func (s *ConfigTestSuite) TestLoad_Overrides() {
	s.T().Setenv("PORT", "9090")
	s.T().Setenv("SHUTDOWN_TIMEOUT", "10s")
	s.T().Setenv("STORAGE_TYPE", "postgres")
	s.T().Setenv("DATABASE_URL", "postgres://localhost/test")
	s.T().Setenv("DOMAIN_MAX_COMMENT_LENGTH", "500")
	s.T().Setenv("SUBSCRIPTION_BUFFER_SIZE", "200")
	s.T().Setenv("LOG_LEVEL", "debug")
	s.T().Setenv("LOG_FORMAT", "json")

	cfg, err := Load()
	s.Require().NoError(err)

	s.Equal("9090", cfg.Port)
	s.Equal(10*time.Second, cfg.ShutdownTimeout)
	s.Equal(PostgresStorage, cfg.StorageType)
	s.Equal("postgres://localhost/test", cfg.DatabaseURL)
	s.Equal(500, cfg.DomainLimits.MaxCommentLength)
	s.Equal(200, cfg.SubscriptionConfig.BufferSize)
	s.Equal("debug", cfg.Logging.Level)
	s.Equal("json", cfg.Logging.Format)
}

func (s *ConfigTestSuite) TestLoad_MissingPort() {
	_, err := Load()
	s.Require().Error(err)
	s.ErrorContains(err, "failed to parse config")
}

func (s *ConfigTestSuite) TestLoad_InvalidStorageType() {
	s.T().Setenv("PORT", "8080")
	s.T().Setenv("STORAGE_TYPE", "mysql")

	_, err := Load()
	s.Require().Error(err)
	s.ErrorContains(err, "invalid STORAGE_TYPE")
}

func (s *ConfigTestSuite) TestLoad_PostgresMissingURL() {
	s.T().Setenv("PORT", "8080")
	s.T().Setenv("STORAGE_TYPE", "postgres")

	_, err := Load()
	s.Require().Error(err)
	s.ErrorContains(err, "postgres storage requires DATABASE_URL")
}

func (s *ConfigTestSuite) TestLoad_NegativeMaxCommentLength() {
	s.T().Setenv("PORT", "8080")
	s.T().Setenv("DOMAIN_MAX_COMMENT_LENGTH", "-1")

	_, err := Load()
	s.Require().Error(err)
	s.ErrorContains(err, "DOMAIN_MAX_COMMENT_LENGTH must be positive")
}

func (s *ConfigTestSuite) TestLoad_InvalidShutdownTimeout() {
	s.T().Setenv("PORT", "8080")
	s.T().Setenv("SHUTDOWN_TIMEOUT", "abc")

	_, err := Load()
	s.Require().Error(err)
	s.ErrorContains(err, "failed to parse config")
}

func (s *ConfigTestSuite) TestValidate_Success() {
	cfg := &Config{
		Port:        "8080",
		StorageType: "memory",
		DomainLimits: DomainLimits{
			MaxCommentLength: 2000,
		},
	}
	err := cfg.Validate()
	s.Require().NoError(err)
}

func (s *ConfigTestSuite) TestValidate_InvalidStorageType() {
	cfg := &Config{
		Port:        "8080",
		StorageType: "invalid",
	}
	err := cfg.Validate()
	s.Require().Error(err)
	s.ErrorContains(err, "invalid STORAGE_TYPE")
}

func (s *ConfigTestSuite) TestValidate_PostgresMissingURL() {
	cfg := &Config{
		Port:        "8080",
		StorageType: "postgres",
	}
	err := cfg.Validate()
	s.Require().Error(err)
	s.ErrorContains(err, "postgres storage requires DATABASE_URL")
}

func (s *ConfigTestSuite) TestValidate_NegativeMaxCommentLength() {
	cfg := &Config{
		Port:        "8080",
		StorageType: "memory",
		DomainLimits: DomainLimits{
			MaxCommentLength: 0,
		},
	}
	err := cfg.Validate()
	s.Require().Error(err)
	s.ErrorContains(err, "DOMAIN_MAX_COMMENT_LENGTH must be positive")
}
