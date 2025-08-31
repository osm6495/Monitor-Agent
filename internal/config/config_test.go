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
				"DB_HOST":             "test-host",
				"DB_PORT":             "5432",
				"DB_NAME":             "test_db",
				"DB_USER":             "test_user",
				"DB_PASSWORD":         "test_password",
				"DB_SSL_MODE":         "require",
				"HACKERONE_USERNAME":  "h1_user",
				"HACKERONE_API_KEY":   "h1_key",
				"BUGCROWD_API_KEY":    "bc_key",
				"CHAOSDB_API_KEY":     "cd_key",
				"LOG_LEVEL":           "debug",
				"ENVIRONMENT":         "production",
				"HTTP_TIMEOUT":        "60s",
				"HTTP_RETRY_ATTEMPTS": "5",
				"HTTP_RETRY_DELAY":    "2s",
				"CHAOSDB_BULK_SIZE":   "200",
			},
			want: &Config{
				Database: DatabaseConfig{
					Host:            "test-host",
					Port:            5432,
					Name:            "test_db",
					User:            "test_user",
					Password:        "test_password",
					SSLMode:         "require",
					SSLCert:         "",
					SSLKey:          "",
					SSLRootCert:     "",
					ConnectTimeout:  60 * time.Second,
					MaxOpenConns:    25,
					MaxIdleConns:    5,
					ConnMaxLifetime: 5 * time.Minute,
				},
				APIs: APIConfig{
					HackerOne: HackerOneConfig{
						APIKey:    "h1_key",
						Username:  "h1_user",
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
					LogLevel:    "debug",
					Environment: "production",
				},
				HTTP: HTTPConfig{
					Timeout:       60 * time.Second,
					RetryAttempts: 5,
					RetryDelay:    2 * time.Second,
				},
				Discovery: DiscoveryConfig{
					BulkSize: 200,
					HTTPX: HTTPXConfig{
						Enabled:         true,
						Timeout:         15 * time.Second,
						Concurrency:     25,
						RateLimit:       100,
						FollowRedirects: true,
						MaxRedirects:    3,
					},
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
				"DB_PASSWORD":        "test_password",
				"HACKERONE_USERNAME": "h1_user",
				"HACKERONE_API_KEY":  "h1_key",
				"BUGCROWD_API_KEY":   "bc_key",
				"CHAOSDB_API_KEY":    "cd_key",
			},
			want: &Config{
				Database: DatabaseConfig{
					Host:            "localhost",
					Port:            5432,
					Name:            "monitor_agent",
					User:            "monitor_agent",
					Password:        "test_password",
					SSLMode:         "disable",
					SSLCert:         "",
					SSLKey:          "",
					SSLRootCert:     "",
					ConnectTimeout:  60 * time.Second,
					MaxOpenConns:    25,
					MaxIdleConns:    5,
					ConnMaxLifetime: 5 * time.Minute,
				},
				APIs: APIConfig{
					HackerOne: HackerOneConfig{
						APIKey:    "h1_key",
						Username:  "h1_user",
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
					LogLevel:    "info",
					Environment: "development",
				},
				HTTP: HTTPConfig{
					Timeout:       60 * time.Second,
					RetryAttempts: 3,
					RetryDelay:    1 * time.Second,
				},
				Discovery: DiscoveryConfig{
					BulkSize: 100,
					HTTPX: HTTPXConfig{
						Enabled:         true,
						Timeout:         15 * time.Second,
						Concurrency:     25,
						RateLimit:       100,
						FollowRedirects: true,
						MaxRedirects:    3,
					},
				},
			},
			wantErr: false,
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
						Username:  "h1_user",
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
					LogLevel:    "info",
					Environment: "development",
				},
				HTTP: HTTPConfig{
					Timeout:       30 * time.Second,
					RetryAttempts: 3,
					RetryDelay:    1 * time.Second,
				},
				Discovery: DiscoveryConfig{
					BulkSize: 100,
					HTTPX: HTTPXConfig{
						Enabled:         true,
						Timeout:         30 * time.Second,
						Concurrency:     25,
						RateLimit:       100,
						FollowRedirects: true,
						MaxRedirects:    3,
					},
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
						Username:  "h1_user",
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
					LogLevel:    "info",
					Environment: "development",
				},
				HTTP: HTTPConfig{
					Timeout:       30 * time.Second,
					RetryAttempts: 3,
					RetryDelay:    1 * time.Second,
				},
				Discovery: DiscoveryConfig{
					BulkSize: 100,
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
						Username:  "",
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
					LogLevel:    "info",
					Environment: "development",
				},
				HTTP: HTTPConfig{
					Timeout:       30 * time.Second,
					RetryAttempts: 3,
					RetryDelay:    1 * time.Second,
				},
				Discovery: DiscoveryConfig{
					BulkSize: 100,
				},
			},
			wantErr: false,
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
						Username:  "h1_user",
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
					LogLevel:    "info",
					Environment: "development",
				},
				HTTP: HTTPConfig{
					Timeout:       30 * time.Second,
					RetryAttempts: 3,
					RetryDelay:    1 * time.Second,
				},
				Discovery: DiscoveryConfig{
					BulkSize: 100,
				},
			},
			wantErr: false,
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
						Username:  "h1_user",
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
					LogLevel:    "info",
					Environment: "development",
				},
				HTTP: HTTPConfig{
					Timeout:       30 * time.Second,
					RetryAttempts: 3,
					RetryDelay:    1 * time.Second,
				},
				Discovery: DiscoveryConfig{
					BulkSize: 100,
				},
			},
			wantErr: false,
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

func TestConfig_PlatformConfiguration(t *testing.T) {
	config := &Config{
		APIs: APIConfig{
			HackerOne: HackerOneConfig{
				APIKey:   "h1_key",
				Username: "h1_user",
			},
			BugCrowd: BugCrowdConfig{
				APIKey: "",
			},
			ChaosDB: ChaosDBConfig{
				APIKey: "cd_key",
			},
		},
	}

	// Test HasHackerOneConfig
	assert.True(t, config.HasHackerOneConfig())
	assert.False(t, config.HasBugCrowdConfig())
	assert.True(t, config.HasChaosDBConfig())

	// Test GetConfiguredPlatforms
	platforms := config.GetConfiguredPlatforms()
	assert.Contains(t, platforms, "hackerone")
	assert.NotContains(t, platforms, "bugcrowd")
	assert.Contains(t, platforms, "chaosdb")
	assert.Len(t, platforms, 2)
}

func TestConfig_NoPlatformsConfigured(t *testing.T) {
	config := &Config{
		APIs: APIConfig{
			HackerOne: HackerOneConfig{
				APIKey:   "",
				Username: "",
			},
			BugCrowd: BugCrowdConfig{
				APIKey: "",
			},
			ChaosDB: ChaosDBConfig{
				APIKey: "",
			},
		},
	}

	// Test Has*Config methods
	assert.False(t, config.HasHackerOneConfig())
	assert.False(t, config.HasBugCrowdConfig())
	assert.False(t, config.HasChaosDBConfig())

	// Test GetConfiguredPlatforms
	platforms := config.GetConfiguredPlatforms()
	assert.Empty(t, platforms)
}
