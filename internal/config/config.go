package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
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
	Username  string
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
	LogLevel    string
	Environment string
}

// HTTPConfig holds HTTP client configuration
type HTTPConfig struct {
	Timeout       time.Duration
	RetryAttempts int
	RetryDelay    time.Duration
}

// DiscoveryConfig holds discovery configuration
type DiscoveryConfig struct {
	BulkSize int
	HTTPX    HTTPXConfig
	Timeouts TimeoutConfig
}

// HTTPXConfig holds HTTPX probe configuration
type HTTPXConfig struct {
	Enabled         bool
	Timeout         time.Duration
	Concurrency     int
	RateLimit       int
	FollowRedirects bool
	MaxRedirects    int
	Debug           bool // Enable debug logging for HTTPX probes
}

// TimeoutConfig holds program-level timeouts
type TimeoutConfig struct {
	ProgramProcess time.Duration
	ChaosDiscovery time.Duration
}

// Load loads configuration from YAML config file and environment variables
func Load() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		logrus.Debug("No .env file found, using environment variables")
	}

	// Try to load from config file first
	config, err := loadFromConfigFile()
	if err != nil {
		logrus.Debugf("Failed to load from config file, using environment variables: %v", err)
		config = &Config{}
	} else {
		logrus.Info("Configuration loaded from config file")
		// Still load sensitive values from environment variables
		loadSensitiveFromEnv(config)
		return config, nil
	}

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

	connectTimeout, err := time.ParseDuration(getEnv("DB_CONNECT_TIMEOUT", "60s"))
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
			Username:  getEnv("HACKERONE_USERNAME", ""),
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
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		Environment: getEnv("ENVIRONMENT", "development"),
	}

	// HTTP configuration
	timeout, err := time.ParseDuration(getEnv("HTTP_TIMEOUT", "60s"))
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

	// HTTPX configuration
	httpxEnabled := getEnv("HTTPX_ENABLED", "true") == "true"

	httpxTimeout, err := time.ParseDuration(getEnv("HTTPX_TIMEOUT", "30s"))
	if err != nil {
		return nil, fmt.Errorf("invalid HTTPX_TIMEOUT: %w", err)
	}

	httpxConcurrency, err := strconv.Atoi(getEnv("HTTPX_CONCURRENCY", "25"))
	if err != nil {
		return nil, fmt.Errorf("invalid HTTPX_CONCURRENCY: %w", err)
	}

	httpxRateLimit, err := strconv.Atoi(getEnv("HTTPX_RATE_LIMIT", "50"))
	if err != nil {
		return nil, fmt.Errorf("invalid HTTPX_RATE_LIMIT: %w", err)
	}

	httpxFollowRedirects := getEnv("HTTPX_FOLLOW_REDIRECTS", "true") == "true"

	httpxMaxRedirects, err := strconv.Atoi(getEnv("HTTPX_MAX_REDIRECTS", "3"))
	if err != nil {
		return nil, fmt.Errorf("invalid HTTPX_MAX_REDIRECTS: %w", err)
	}

	httpxDebug := getEnv("HTTPX_DEBUG", "false") == "true"

	programProcessTimeout, err := time.ParseDuration(getEnv("PROGRAM_PROCESS_TIMEOUT", "45m"))
	if err != nil {
		return nil, fmt.Errorf("invalid PROGRAM_PROCESS_TIMEOUT: %w", err)
	}

	chaosDiscoveryTimeout, err := time.ParseDuration(getEnv("CHAOS_DISCOVERY_TIMEOUT", "30m"))
	if err != nil {
		return nil, fmt.Errorf("invalid CHAOS_DISCOVERY_TIMEOUT: %w", err)
	}

	config.Discovery = DiscoveryConfig{
		BulkSize: bulkSize,
		HTTPX: HTTPXConfig{
			Enabled:         httpxEnabled,
			Timeout:         httpxTimeout,
			Concurrency:     httpxConcurrency,
			RateLimit:       httpxRateLimit,
			FollowRedirects: httpxFollowRedirects,
			MaxRedirects:    httpxMaxRedirects,
			Debug:           httpxDebug,
		},
		Timeouts: TimeoutConfig{
			ProgramProcess: programProcessTimeout,
			ChaosDiscovery: chaosDiscoveryTimeout,
		},
	}

	return config, nil
}

// loadFromConfigFile loads configuration from YAML config file
func loadFromConfigFile() (*Config, error) {
	configPath := getConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// getConfigPath returns the path to the config file
func getConfigPath() string {
	// Check for config file in current directory first
	if _, err := os.Stat("configs/config.yaml"); err == nil {
		return "configs/config.yaml"
	}

	// Check for config file in parent directory
	if _, err := os.Stat("../configs/config.yaml"); err == nil {
		return "../configs/config.yaml"
	}

	// Default to current directory
	return "configs/config.yaml"
}

// loadSensitiveFromEnv loads sensitive configuration values from environment variables
func loadSensitiveFromEnv(config *Config) {
	// Database password
	if password := os.Getenv("DB_PASSWORD"); password != "" {
		config.Database.Password = password
	}

	// API keys
	if apiKey := os.Getenv("HACKERONE_API_KEY"); apiKey != "" {
		config.APIs.HackerOne.APIKey = apiKey
	}
	if username := os.Getenv("HACKERONE_USERNAME"); username != "" {
		config.APIs.HackerOne.Username = username
	}
	if apiKey := os.Getenv("BUGCROWD_API_KEY"); apiKey != "" {
		config.APIs.BugCrowd.APIKey = apiKey
	}
	if apiKey := os.Getenv("CHAOSDB_API_KEY"); apiKey != "" {
		config.APIs.ChaosDB.APIKey = apiKey
	}
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
	// Validate HackerOne configuration (only if API key is provided)
	if c.APIs.HackerOne.APIKey != "" {
		if c.APIs.HackerOne.RateLimit <= 0 || c.APIs.HackerOne.RateLimit > 600 {
			return fmt.Errorf("HACKERONE_RATE_LIMIT must be between 1 and 600")
		}
	}

	// Validate BugCrowd configuration (only if API key is provided)
	if c.APIs.BugCrowd.APIKey != "" {
		if c.APIs.BugCrowd.RateLimit <= 0 || c.APIs.BugCrowd.RateLimit > 60 {
			return fmt.Errorf("BUGCROWD_RATE_LIMIT must be between 1 and 60")
		}
	}

	// Validate ChaosDB configuration (only if API key is provided)
	if c.APIs.ChaosDB.APIKey != "" {
		if c.APIs.ChaosDB.RateLimit <= 0 || c.APIs.ChaosDB.RateLimit > 60 {
			return fmt.Errorf("CHAOSDB_RATE_LIMIT must be between 1 and 60")
		}
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

	// Validate HTTPX configuration if enabled
	if c.Discovery.HTTPX.Enabled {
		if c.Discovery.HTTPX.Timeout <= 0 {
			return fmt.Errorf("HTTPX_TIMEOUT must be greater than 0")
		}
		if c.Discovery.HTTPX.Concurrency <= 0 || c.Discovery.HTTPX.Concurrency > 100 {
			return fmt.Errorf("HTTPX_CONCURRENCY must be between 1 and 100")
		}
		if c.Discovery.HTTPX.RateLimit <= 0 {
			return fmt.Errorf("HTTPX_RATE_LIMIT must be greater than 0")
		}
		if c.Discovery.HTTPX.MaxRedirects < 0 || c.Discovery.HTTPX.MaxRedirects > 10 {
			return fmt.Errorf("HTTPX_MAX_REDIRECTS must be between 0 and 10")
		}
	}

	// Validate timeouts
	if c.Discovery.Timeouts.ProgramProcess <= 0 {
		return fmt.Errorf("PROGRAM_PROCESS_TIMEOUT must be greater than 0")
	}
	if c.Discovery.Timeouts.ChaosDiscovery <= 0 {
		return fmt.Errorf("CHAOS_DISCOVERY_TIMEOUT must be greater than 0")
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

// HasHackerOneConfig returns true if HackerOne is configured with an API key and username
func (c *Config) HasHackerOneConfig() bool {
	return c.APIs.HackerOne.APIKey != "" && c.APIs.HackerOne.Username != ""
}

// HasBugCrowdConfig returns true if BugCrowd is configured with an API key
func (c *Config) HasBugCrowdConfig() bool {
	return c.APIs.BugCrowd.APIKey != ""
}

// HasChaosDBConfig returns true if ChaosDB is configured with an API key
func (c *Config) HasChaosDBConfig() bool {
	return c.APIs.ChaosDB.APIKey != ""
}

// GetConfiguredPlatforms returns a list of platform names that have API keys configured
func (c *Config) GetConfiguredPlatforms() []string {
	var platforms []string
	if c.HasHackerOneConfig() {
		platforms = append(platforms, "hackerone")
	}
	if c.HasBugCrowdConfig() {
		platforms = append(platforms, "bugcrowd")
	}
	if c.HasChaosDBConfig() {
		platforms = append(platforms, "chaosdb")
	}
	return platforms
}
