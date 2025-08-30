package integration

import (
	"testing"
	"time"

	"github.com/monitor-agent/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestConfigurationValidationIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("InvalidDatabaseConfiguration", func(t *testing.T) {
		invalidConfig := &config.Config{
			Database: config.DatabaseConfig{
				Host:            "localhost",
				Port:            99999, // Invalid port
				Name:            "test_db",
				User:            "test_user",
				Password:        "password",
				SSLMode:         "disable",
				ConnectTimeout:  30 * time.Second,
				MaxOpenConns:    25,
				MaxIdleConns:    5,
				ConnMaxLifetime: 5 * time.Minute,
			},
			APIs: config.APIConfig{
				HackerOne: config.HackerOneConfig{
					APIKey:    "h1_key",
					RateLimit: 550,
				},
				BugCrowd: config.BugCrowdConfig{
					APIKey:    "bc_key",
					RateLimit: 55,
				},
				ChaosDB: config.ChaosDBConfig{
					APIKey:    "cd_key",
					RateLimit: 55,
				},
			},
			App: config.AppConfig{
				LogLevel:    "info",
				Environment: "development",
			},
			HTTP: config.HTTPConfig{
				Timeout:       30 * time.Second,
				RetryAttempts: 3,
				RetryDelay:    1 * time.Second,
			},
			Discovery: config.DiscoveryConfig{
				BulkSize: 100,
			},
		}

		err := invalidConfig.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DB_PORT must be between 1 and 65535")
	})

	t.Run("InvalidAPIConfiguration", func(t *testing.T) {
		invalidConfig := &config.Config{
			Database: config.DatabaseConfig{
				Host:            "localhost",
				Port:            5432,
				Name:            "test_db",
				User:            "test_user",
				Password:        "password",
				SSLMode:         "disable",
				ConnectTimeout:  30 * time.Second,
				MaxOpenConns:    25,
				MaxIdleConns:    5,
				ConnMaxLifetime: 5 * time.Minute,
			},
			APIs: config.APIConfig{
				HackerOne: config.HackerOneConfig{
					APIKey:    "h1_key",
					RateLimit: 1000, // Exceeds limit
				},
				BugCrowd: config.BugCrowdConfig{
					APIKey:    "bc_key",
					RateLimit: 55,
				},
				ChaosDB: config.ChaosDBConfig{
					APIKey:    "cd_key",
					RateLimit: 55,
				},
			},
			App: config.AppConfig{
				LogLevel:    "info",
				Environment: "development",
			},
			HTTP: config.HTTPConfig{
				Timeout:       30 * time.Second,
				RetryAttempts: 3,
				RetryDelay:    1 * time.Second,
			},
			Discovery: config.DiscoveryConfig{
				BulkSize: 100,
			},
		}

		err := invalidConfig.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "HACKERONE_RATE_LIMIT must be between 1 and 600")
	})

	t.Run("InvalidHTTPConfiguration", func(t *testing.T) {
		invalidConfig := &config.Config{
			Database: config.DatabaseConfig{
				Host:            "localhost",
				Port:            5432,
				Name:            "test_db",
				User:            "test_user",
				Password:        "password",
				SSLMode:         "disable",
				ConnectTimeout:  30 * time.Second,
				MaxOpenConns:    25,
				MaxIdleConns:    5,
				ConnMaxLifetime: 5 * time.Minute,
			},
			APIs: config.APIConfig{
				HackerOne: config.HackerOneConfig{
					APIKey:    "h1_key",
					RateLimit: 550,
				},
				BugCrowd: config.BugCrowdConfig{
					APIKey:    "bc_key",
					RateLimit: 55,
				},
				ChaosDB: config.ChaosDBConfig{
					APIKey:    "cd_key",
					RateLimit: 55,
				},
			},
			App: config.AppConfig{
				LogLevel:    "info",
				Environment: "development",
			},
			HTTP: config.HTTPConfig{
				Timeout:       -1 * time.Second, // Invalid timeout
				RetryAttempts: 3,
				RetryDelay:    1 * time.Second,
			},
			Discovery: config.DiscoveryConfig{
				BulkSize: 100,
			},
		}

		err := invalidConfig.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "HTTP_TIMEOUT must be greater than 0")
	})

	t.Run("InvalidDiscoveryConfiguration", func(t *testing.T) {
		invalidConfig := &config.Config{
			Database: config.DatabaseConfig{
				Host:            "localhost",
				Port:            5432,
				Name:            "test_db",
				User:            "test_user",
				Password:        "password",
				SSLMode:         "disable",
				ConnectTimeout:  30 * time.Second,
				MaxOpenConns:    25,
				MaxIdleConns:    5,
				ConnMaxLifetime: 5 * time.Minute,
			},
			APIs: config.APIConfig{
				HackerOne: config.HackerOneConfig{
					APIKey:    "h1_key",
					RateLimit: 550,
				},
				BugCrowd: config.BugCrowdConfig{
					APIKey:    "bc_key",
					RateLimit: 55,
				},
				ChaosDB: config.ChaosDBConfig{
					APIKey:    "cd_key",
					RateLimit: 55,
				},
			},
			App: config.AppConfig{
				LogLevel:    "info",
				Environment: "development",
			},
			HTTP: config.HTTPConfig{
				Timeout:       30 * time.Second,
				RetryAttempts: 3,
				RetryDelay:    1 * time.Second,
			},
			Discovery: config.DiscoveryConfig{
				BulkSize: 2000, // Exceeds limit
			},
		}

		err := invalidConfig.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "CHAOSDB_BULK_SIZE must be between 1 and 1000")
	})
}
