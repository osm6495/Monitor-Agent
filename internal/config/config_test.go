package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		want    *Config
		wantErr bool
	}{
		{
			name: "valid configuration",
			envVars: map[string]string{
				"DB_HOST":                    "test-host",
				"DB_PORT":                    "5433",
				"DB_NAME":                    "test_db",
				"DB_USER":                    "test_user",
				"DB_PASSWORD":                "test_password",
				"DB_SSL_MODE":                "require",
				"HACKERONE_API_KEY":          "h1_key",
				"HACKERONE_RATE_LIMIT":       "150",
				"BUGCROWD_API_KEY":           "bc_key",
				"BUGCROWD_RATE_LIMIT":        "200",
				"CHAOSDB_API_KEY":            "cd_key",
				"CHAOSDB_RATE_LIMIT":         "75",
				"LOG_LEVEL":                  "debug",
				"ENVIRONMENT":                "test",
				"CRON_SCHEDULE":              "0 */12 * * *",
				"HTTP_TIMEOUT":               "60s",
				"HTTP_RETRY_ATTEMPTS":        "5",
				"HTTP_RETRY_DELAY":           "2s",
				"CHAOSDB_BULK_SIZE":          "200",
				"DISCOVERY_CONCURRENT_LIMIT": "20",
			},
			want: &Config{
				Database: DatabaseConfig{
					Host:            "test-host",
					Port:            5433,
					Name:            "test_db",
					User:            "test_user",
					Password:        "test_password",
					SSLMode:         "require",
					SSLCert:         "",
					SSLKey:          "",
					SSLRootCert:     "",
					ConnectTimeout:  30000000000, // 30s in nanoseconds
					MaxOpenConns:    25,
					MaxIdleConns:    5,
					ConnMaxLifetime: 300000000000, // 5m in nanoseconds
				},
				APIs: APIConfig{
					HackerOne: HackerOneConfig{
						APIKey:    "h1_key",
						RateLimit: 150,
					},
					BugCrowd: BugCrowdConfig{
						APIKey:    "bc_key",
						RateLimit: 200,
					},
					ChaosDB: ChaosDBConfig{
						APIKey:    "cd_key",
						RateLimit: 75,
					},
				},
				App: AppConfig{
					LogLevel:     "debug",
					Environment:  "test",
					CronSchedule: "0 */12 * * *",
				},
				HTTP: HTTPConfig{
					Timeout:       60000000000, // 60s in nanoseconds
					RetryAttempts: 5,
					RetryDelay:    2000000000, // 2s in nanoseconds
				},
				Discovery: DiscoveryConfig{
					BulkSize:        200,
					ConcurrentLimit: 20,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid DB_PORT",
			envVars: map[string]string{
				"DB_PORT": "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid HTTP_TIMEOUT",
			envVars: map[string]string{
				"HTTP_TIMEOUT": "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid HTTP_RETRY_DELAY",
			envVars: map[string]string{
				"HTTP_RETRY_DELAY": "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid HTTP_RETRY_ATTEMPTS",
			envVars: map[string]string{
				"HTTP_RETRY_ATTEMPTS": "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid HACKERONE_RATE_LIMIT",
			envVars: map[string]string{
				"HACKERONE_RATE_LIMIT": "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid BUGCROWD_RATE_LIMIT",
			envVars: map[string]string{
				"BUGCROWD_RATE_LIMIT": "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid CHAOSDB_RATE_LIMIT",
			envVars: map[string]string{
				"CHAOSDB_RATE_LIMIT": "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid CHAOSDB_BULK_SIZE",
			envVars: map[string]string{
				"CHAOSDB_BULK_SIZE": "invalid",
			},
			wantErr: true,
		},
		{
			name: "default rate limits when not provided",
			envVars: map[string]string{
				"DB_HOST":           "test-host",
				"DB_PORT":           "5433",
				"DB_NAME":           "test_db",
				"DB_USER":           "test_user",
				"DB_PASSWORD":       "test_password",
				"HACKERONE_API_KEY": "h1_key",
				"BUGCROWD_API_KEY":  "bc_key",
				"CHAOSDB_API_KEY":   "cd_key",
			},
			want: &Config{
				Database: DatabaseConfig{
					Host:            "test-host",
					Port:            5433,
					Name:            "test_db",
					User:            "test_user",
					Password:        "test_password",
					SSLMode:         "disable",
					SSLCert:         "",
					SSLKey:          "",
					SSLRootCert:     "",
					ConnectTimeout:  30000000000, // 30s in nanoseconds
					MaxOpenConns:    25,
					MaxIdleConns:    5,
					ConnMaxLifetime: 300000000000, // 5m in nanoseconds
				},
				APIs: APIConfig{
					HackerOne: HackerOneConfig{
						APIKey:    "h1_key",
						RateLimit: 550, // Default value
					},
					BugCrowd: BugCrowdConfig{
						APIKey:    "bc_key",
						RateLimit: 55, // Default value
					},
					ChaosDB: ChaosDBConfig{
						APIKey:    "cd_key",
						RateLimit: 55, // Default value
					},
				},
				App: AppConfig{
					LogLevel:     "info",
					Environment:  "development",
					CronSchedule: "0 */6 * * *",
				},
				HTTP: HTTPConfig{
					Timeout:       30000000000, // 30s in nanoseconds
					RetryAttempts: 3,
					RetryDelay:    1000000000, // 1s in nanoseconds
				},
				Discovery: DiscoveryConfig{
					BulkSize:        100,
					ConcurrentLimit: 10,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid DISCOVERY_CONCURRENT_LIMIT",
			envVars: map[string]string{
				"DISCOVERY_CONCURRENT_LIMIT": "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			defer func() {
				// Clean up environment variables
				for key := range tt.envVars {
					os.Unsetenv(key)
				}
			}()

			got, err := Load()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid configuration",
			config: &Config{
				Database: DatabaseConfig{
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
				APIs: APIConfig{
					HackerOne: HackerOneConfig{
						APIKey:    "h1_key",
						RateLimit: 550,
					},
					BugCrowd: BugCrowdConfig{
						APIKey:    "bc_key",
						RateLimit: 55,
					},
					ChaosDB: ChaosDBConfig{
						APIKey:    "cd_key",
						RateLimit: 55,
					},
				},
				App: AppConfig{
					LogLevel:     "info",
					Environment:  "development",
					CronSchedule: "0 */6 * * *",
				},
				HTTP: HTTPConfig{
					Timeout:       30 * time.Second,
					RetryAttempts: 3,
					RetryDelay:    1 * time.Second,
				},
				Discovery: DiscoveryConfig{
					BulkSize:        100,
					ConcurrentLimit: 10,
				},
			},
			wantErr: false,
		},
		{
			name: "missing DB_PASSWORD",
			config: &Config{
				Database: DatabaseConfig{
					Host:            "localhost",
					Port:            5432,
					Name:            "test_db",
					User:            "test_user",
					Password:        "",
					SSLMode:         "disable",
					ConnectTimeout:  30 * time.Second,
					MaxOpenConns:    25,
					MaxIdleConns:    5,
					ConnMaxLifetime: 5 * time.Minute,
				},
				APIs: APIConfig{
					HackerOne: HackerOneConfig{
						APIKey:    "h1_key",
						RateLimit: 550,
					},
					BugCrowd: BugCrowdConfig{
						APIKey:    "bc_key",
						RateLimit: 55,
					},
					ChaosDB: ChaosDBConfig{
						APIKey:    "cd_key",
						RateLimit: 55,
					},
				},
				App: AppConfig{
					LogLevel:     "info",
					Environment:  "development",
					CronSchedule: "0 */6 * * *",
				},
				HTTP: HTTPConfig{
					Timeout:       30 * time.Second,
					RetryAttempts: 3,
					RetryDelay:    1 * time.Second,
				},
				Discovery: DiscoveryConfig{
					BulkSize:        100,
					ConcurrentLimit: 10,
				},
			},
			wantErr: true,
		},
		{
			name: "missing HACKERONE_API_KEY",
			config: &Config{
				Database: DatabaseConfig{
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
				APIs: APIConfig{
					HackerOne: HackerOneConfig{
						APIKey:    "",
						RateLimit: 550,
					},
					BugCrowd: BugCrowdConfig{
						APIKey:    "bc_key",
						RateLimit: 55,
					},
					ChaosDB: ChaosDBConfig{
						APIKey:    "cd_key",
						RateLimit: 55,
					},
				},
				App: AppConfig{
					LogLevel:     "info",
					Environment:  "development",
					CronSchedule: "0 */6 * * *",
				},
				HTTP: HTTPConfig{
					Timeout:       30 * time.Second,
					RetryAttempts: 3,
					RetryDelay:    1 * time.Second,
				},
				Discovery: DiscoveryConfig{
					BulkSize:        100,
					ConcurrentLimit: 10,
				},
			},
			wantErr: true,
		},
		{
			name: "missing BUGCROWD_API_KEY",
			config: &Config{
				Database: DatabaseConfig{
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
				APIs: APIConfig{
					HackerOne: HackerOneConfig{
						APIKey:    "h1_key",
						RateLimit: 550,
					},
					BugCrowd: BugCrowdConfig{
						APIKey:    "",
						RateLimit: 55,
					},
					ChaosDB: ChaosDBConfig{
						APIKey:    "cd_key",
						RateLimit: 55,
					},
				},
				App: AppConfig{
					LogLevel:     "info",
					Environment:  "development",
					CronSchedule: "0 */6 * * *",
				},
				HTTP: HTTPConfig{
					Timeout:       30 * time.Second,
					RetryAttempts: 3,
					RetryDelay:    1 * time.Second,
				},
				Discovery: DiscoveryConfig{
					BulkSize:        100,
					ConcurrentLimit: 10,
				},
			},
			wantErr: true,
		},
		{
			name: "missing CHAOSDB_API_KEY",
			config: &Config{
				Database: DatabaseConfig{
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
				APIs: APIConfig{
					HackerOne: HackerOneConfig{
						APIKey:    "h1_key",
						RateLimit: 550,
					},
					BugCrowd: BugCrowdConfig{
						APIKey:    "bc_key",
						RateLimit: 55,
					},
					ChaosDB: ChaosDBConfig{
						APIKey:    "",
						RateLimit: 55,
					},
				},
				App: AppConfig{
					LogLevel:     "info",
					Environment:  "development",
					CronSchedule: "0 */6 * * *",
				},
				HTTP: HTTPConfig{
					Timeout:       30 * time.Second,
					RetryAttempts: 3,
					RetryDelay:    1 * time.Second,
				},
				Discovery: DiscoveryConfig{
					BulkSize:        100,
					ConcurrentLimit: 10,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_GetDSN(t *testing.T) {
	config := &Config{
		Database: DatabaseConfig{
			Host:           "localhost",
			Port:           5432,
			Name:           "test_db",
			User:           "test_user",
			Password:       "test_password",
			SSLMode:        "disable",
			ConnectTimeout: 30 * time.Second,
		},
	}

	expected := "host=localhost port=5432 dbname=test_db user=test_user password=test_password sslmode=disable connect_timeout=30"
	assert.Equal(t, expected, config.GetDSN())
}

func TestGetEnv(t *testing.T) {
	// Test with existing environment variable
	os.Setenv("TEST_KEY", "test_value")
	defer os.Unsetenv("TEST_KEY")

	assert.Equal(t, "test_value", getEnv("TEST_KEY", "default"))

	// Test with non-existing environment variable
	assert.Equal(t, "default", getEnv("NON_EXISTENT_KEY", "default"))
}
