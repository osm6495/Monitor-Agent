package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// Config holds all configuration for the application
type Config struct {
	Database  DatabaseConfig
	APIs      APIConfig
	App       AppConfig
	HTTP      HTTPConfig
	Discovery DiscoveryConfig
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host            string
	Port            int
	Name            string
	User            string
	Password        string
	SSLMode         string
	SSLCert         string
	SSLKey          string
	SSLRootCert     string
	ConnectTimeout  time.Duration
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// APIConfig holds API configuration
type APIConfig struct {
	HackerOne HackerOneConfig
	BugCrowd  BugCrowdConfig
	ChaosDB   ChaosDBConfig
}

// HackerOneConfig holds HackerOne API configuration
type HackerOneConfig struct {
	APIKey    string
	RateLimit int
}

// BugCrowdConfig holds BugCrowd API configuration
type BugCrowdConfig struct {
	APIKey    string
	RateLimit int
}

// ChaosDBConfig holds ChaosDB API configuration
type ChaosDBConfig struct {
	APIKey    string
	RateLimit int
}

// AppConfig holds application configuration
type AppConfig struct {
	LogLevel     string
	Environment  string
	CronSchedule string
}

// HTTPConfig holds HTTP client configuration
type HTTPConfig struct {
	Timeout       time.Duration
	RetryAttempts int
	RetryDelay    time.Duration
}

// DiscoveryConfig holds discovery configuration
type DiscoveryConfig struct {
	BulkSize        int
	ConcurrentLimit int
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		logrus.Debug("No .env file found, using environment variables")
	}

	config := &Config{}

	// Database configuration
	dbPort, err := strconv.Atoi(getEnv("DB_PORT", "5432"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_PORT: %w", err)
	}

	// Database connection pool settings
	maxOpenConns, err := strconv.Atoi(getEnv("DB_MAX_OPEN_CONNS", "25"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_MAX_OPEN_CONNS: %w", err)
	}

	maxIdleConns, err := strconv.Atoi(getEnv("DB_MAX_IDLE_CONNS", "5"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_MAX_IDLE_CONNS: %w", err)
	}

	connectTimeout, err := time.ParseDuration(getEnv("DB_CONNECT_TIMEOUT", "30s"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_CONNECT_TIMEOUT: %w", err)
	}

	connMaxLifetime, err := time.ParseDuration(getEnv("DB_CONN_MAX_LIFETIME", "5m"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_CONN_MAX_LIFETIME: %w", err)
	}

	config.Database = DatabaseConfig{
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            dbPort,
		Name:            getEnv("DB_NAME", "monitor_agent"),
		User:            getEnv("DB_USER", "monitor_agent"),
		Password:        getEnv("DB_PASSWORD", ""),
		SSLMode:         getEnv("DB_SSL_MODE", "disable"),
		SSLCert:         getEnv("DB_SSL_CERT", ""),
		SSLKey:          getEnv("DB_SSL_KEY", ""),
		SSLRootCert:     getEnv("DB_SSL_ROOT_CERT", ""),
		ConnectTimeout:  connectTimeout,
		MaxOpenConns:    maxOpenConns,
		MaxIdleConns:    maxIdleConns,
		ConnMaxLifetime: connMaxLifetime,
	}

	// API configuration
	hackerOneRateLimit, err := strconv.Atoi(getEnv("HACKERONE_RATE_LIMIT", "550"))
	if err != nil {
		return nil, fmt.Errorf("invalid HACKERONE_RATE_LIMIT: %w", err)
	}

	bugCrowdRateLimit, err := strconv.Atoi(getEnv("BUGCROWD_RATE_LIMIT", "55"))
	if err != nil {
		return nil, fmt.Errorf("invalid BUGCROWD_RATE_LIMIT: %w", err)
	}

	chaosDBRateLimit, err := strconv.Atoi(getEnv("CHAOSDB_RATE_LIMIT", "55"))
	if err != nil {
		return nil, fmt.Errorf("invalid CHAOSDB_RATE_LIMIT: %w", err)
	}

	config.APIs = APIConfig{
		HackerOne: HackerOneConfig{
			APIKey:    getEnv("HACKERONE_API_KEY", ""),
			RateLimit: hackerOneRateLimit,
		},
		BugCrowd: BugCrowdConfig{
			APIKey:    getEnv("BUGCROWD_API_KEY", ""),
			RateLimit: bugCrowdRateLimit,
		},
		ChaosDB: ChaosDBConfig{
			APIKey:    getEnv("CHAOSDB_API_KEY", ""),
			RateLimit: chaosDBRateLimit,
		},
	}

	// Application configuration
	config.App = AppConfig{
		LogLevel:     getEnv("LOG_LEVEL", "info"),
		Environment:  getEnv("ENVIRONMENT", "development"),
		CronSchedule: getEnv("CRON_SCHEDULE", "0 */6 * * *"),
	}

	// HTTP configuration
	timeout, err := time.ParseDuration(getEnv("HTTP_TIMEOUT", "30s"))
	if err != nil {
		return nil, fmt.Errorf("invalid HTTP_TIMEOUT: %w", err)
	}

	retryDelay, err := time.ParseDuration(getEnv("HTTP_RETRY_DELAY", "1s"))
	if err != nil {
		return nil, fmt.Errorf("invalid HTTP_RETRY_DELAY: %w", err)
	}

	retryAttempts, err := strconv.Atoi(getEnv("HTTP_RETRY_ATTEMPTS", "3"))
	if err != nil {
		return nil, fmt.Errorf("invalid HTTP_RETRY_ATTEMPTS: %w", err)
	}

	config.HTTP = HTTPConfig{
		Timeout:       timeout,
		RetryAttempts: retryAttempts,
		RetryDelay:    retryDelay,
	}

	// Discovery configuration
	bulkSize, err := strconv.Atoi(getEnv("CHAOSDB_BULK_SIZE", "100"))
	if err != nil {
		return nil, fmt.Errorf("invalid CHAOSDB_BULK_SIZE: %w", err)
	}

	concurrentLimit, err := strconv.Atoi(getEnv("DISCOVERY_CONCURRENT_LIMIT", "10"))
	if err != nil {
		return nil, fmt.Errorf("invalid DISCOVERY_CONCURRENT_LIMIT: %w", err)
	}

	config.Discovery = DiscoveryConfig{
		BulkSize:        bulkSize,
		ConcurrentLimit: concurrentLimit,
	}

	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	var errors []string

	// Database validation
	if err := c.validateDatabase(); err != nil {
		errors = append(errors, fmt.Sprintf("database: %v", err))
	}

	// API validation
	if err := c.validateAPIs(); err != nil {
		errors = append(errors, fmt.Sprintf("APIs: %v", err))
	}

	// Application validation
	if err := c.validateApp(); err != nil {
		errors = append(errors, fmt.Sprintf("application: %v", err))
	}

	// HTTP validation
	if err := c.validateHTTP(); err != nil {
		errors = append(errors, fmt.Sprintf("HTTP: %v", err))
	}

	// Discovery validation
	if err := c.validateDiscovery(); err != nil {
		errors = append(errors, fmt.Sprintf("discovery: %v", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// validateDatabase validates database configuration
func (c *Config) validateDatabase() error {
	if c.Database.Password == "" {
		return fmt.Errorf("DB_PASSWORD is required")
	}

	if c.Database.Host == "" {
		return fmt.Errorf("DB_HOST is required")
	}

	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		return fmt.Errorf("DB_PORT must be between 1 and 65535")
	}

	if c.Database.Name == "" {
		return fmt.Errorf("DB_NAME is required")
	}

	if c.Database.User == "" {
		return fmt.Errorf("DB_USER is required")
	}

	// Validate SSL mode
	validSSLModes := []string{"disable", "require", "verify-ca", "verify-full"}
	validSSLMode := false
	for _, mode := range validSSLModes {
		if c.Database.SSLMode == mode {
			validSSLMode = true
			break
		}
	}
	if !validSSLMode {
		return fmt.Errorf("DB_SSL_MODE must be one of: %s", strings.Join(validSSLModes, ", "))
	}

	// Validate SSL certificates if using verify-full
	if c.Database.SSLMode == "verify-full" {
		if c.Database.SSLCert == "" {
			return fmt.Errorf("DB_SSL_CERT is required when DB_SSL_MODE is verify-full")
		}
		if c.Database.SSLKey == "" {
			return fmt.Errorf("DB_SSL_KEY is required when DB_SSL_MODE is verify-full")
		}
		if c.Database.SSLRootCert == "" {
			return fmt.Errorf("DB_SSL_ROOT_CERT is required when DB_SSL_MODE is verify-full")
		}

		// Check if SSL certificate files exist
		if _, err := os.Stat(c.Database.SSLCert); os.IsNotExist(err) {
			return fmt.Errorf("SSL certificate file does not exist: %s", c.Database.SSLCert)
		}
		if _, err := os.Stat(c.Database.SSLKey); os.IsNotExist(err) {
			return fmt.Errorf("SSL key file does not exist: %s", c.Database.SSLKey)
		}
		if _, err := os.Stat(c.Database.SSLRootCert); os.IsNotExist(err) {
			return fmt.Errorf("SSL root certificate file does not exist: %s", c.Database.SSLRootCert)
		}
	}

	// Validate connection pool settings
	if c.Database.MaxOpenConns <= 0 {
		return fmt.Errorf("DB_MAX_OPEN_CONNS must be greater than 0")
	}
	if c.Database.MaxIdleConns <= 0 {
		return fmt.Errorf("DB_MAX_IDLE_CONNS must be greater than 0")
	}
	if c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		return fmt.Errorf("DB_MAX_IDLE_CONNS cannot be greater than DB_MAX_OPEN_CONNS")
	}
	if c.Database.ConnMaxLifetime <= 0 {
		return fmt.Errorf("DB_CONN_MAX_LIFETIME must be greater than 0")
	}
	if c.Database.ConnectTimeout <= 0 {
		return fmt.Errorf("DB_CONNECT_TIMEOUT must be greater than 0")
	}

	return nil
}

// validateAPIs validates API configuration
func (c *Config) validateAPIs() error {
	// Validate HackerOne configuration
	if c.APIs.HackerOne.APIKey == "" {
		return fmt.Errorf("HACKERONE_API_KEY is required")
	}
	if c.APIs.HackerOne.RateLimit <= 0 || c.APIs.HackerOne.RateLimit > 600 {
		return fmt.Errorf("HACKERONE_RATE_LIMIT must be between 1 and 600")
	}

	// Validate BugCrowd configuration
	if c.APIs.BugCrowd.APIKey == "" {
		return fmt.Errorf("BUGCROWD_API_KEY is required")
	}
	if c.APIs.BugCrowd.RateLimit <= 0 || c.APIs.BugCrowd.RateLimit > 60 {
		return fmt.Errorf("BUGCROWD_RATE_LIMIT must be between 1 and 60")
	}

	// Validate ChaosDB configuration
	if c.APIs.ChaosDB.APIKey == "" {
		return fmt.Errorf("CHAOSDB_API_KEY is required")
	}
	if c.APIs.ChaosDB.RateLimit <= 0 || c.APIs.ChaosDB.RateLimit > 60 {
		return fmt.Errorf("CHAOSDB_RATE_LIMIT must be between 1 and 60")
	}

	return nil
}

// validateApp validates application configuration
func (c *Config) validateApp() error {
	// Validate log level
	validLogLevels := []string{"debug", "info", "warn", "error", "fatal"}
	validLogLevel := false
	for _, level := range validLogLevels {
		if c.App.LogLevel == level {
			validLogLevel = true
			break
		}
	}
	if !validLogLevel {
		return fmt.Errorf("LOG_LEVEL must be one of: %s", strings.Join(validLogLevels, ", "))
	}

	// Validate environment
	validEnvironments := []string{"development", "staging", "production"}
	validEnvironment := false
	for _, env := range validEnvironments {
		if c.App.Environment == env {
			validEnvironment = true
			break
		}
	}
	if !validEnvironment {
		return fmt.Errorf("ENVIRONMENT must be one of: %s", strings.Join(validEnvironments, ", "))
	}

	// Validate cron schedule
	if c.App.CronSchedule == "" {
		return fmt.Errorf("CRON_SCHEDULE is required")
	}

	// Basic cron schedule validation (simple check for common patterns)
	if !strings.Contains(c.App.CronSchedule, "*") && !strings.Contains(c.App.CronSchedule, "/") {
		return fmt.Errorf("CRON_SCHEDULE appears to be invalid")
	}

	return nil
}

// validateHTTP validates HTTP configuration
func (c *Config) validateHTTP() error {
	if c.HTTP.Timeout <= 0 {
		return fmt.Errorf("HTTP_TIMEOUT must be greater than 0")
	}
	if c.HTTP.RetryAttempts < 0 || c.HTTP.RetryAttempts > 10 {
		return fmt.Errorf("HTTP_RETRY_ATTEMPTS must be between 0 and 10")
	}
	if c.HTTP.RetryDelay <= 0 {
		return fmt.Errorf("HTTP_RETRY_DELAY must be greater than 0")
	}

	return nil
}

// validateDiscovery validates discovery configuration
func (c *Config) validateDiscovery() error {
	if c.Discovery.BulkSize <= 0 || c.Discovery.BulkSize > 1000 {
		return fmt.Errorf("CHAOSDB_BULK_SIZE must be between 1 and 1000")
	}
	if c.Discovery.ConcurrentLimit <= 0 || c.Discovery.ConcurrentLimit > 100 {
		return fmt.Errorf("DISCOVERY_CONCURRENT_LIMIT must be between 1 and 100")
	}

	return nil
}

// GetDSN returns the database connection string
func (c *Config) GetDSN() string {
	dsn := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s connect_timeout=%d",
		c.Database.Host, c.Database.Port, c.Database.Name, c.Database.User, c.Database.Password,
		c.Database.SSLMode, int(c.Database.ConnectTimeout.Seconds()))

	// Add SSL certificate parameters if provided
	if c.Database.SSLCert != "" {
		dsn += fmt.Sprintf(" sslcert=%s", c.Database.SSLCert)
	}
	if c.Database.SSLKey != "" {
		dsn += fmt.Sprintf(" sslkey=%s", c.Database.SSLKey)
	}
	if c.Database.SSLRootCert != "" {
		dsn += fmt.Sprintf(" sslrootcert=%s", c.Database.SSLRootCert)
	}

	return dsn
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
