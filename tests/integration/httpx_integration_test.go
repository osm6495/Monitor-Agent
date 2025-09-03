package integration

import (
	"context"
	"testing"
	"time"

	"github.com/monitor-agent/internal/config"
	"github.com/monitor-agent/internal/discovery/httpx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPXIntegrationWithChaosDB(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test that HTTPX client can be created with the configuration
	httpxClient := httpx.NewClient(&httpx.ProbeConfig{
		Timeout:         10 * time.Second,
		TotalTimeout:    5 * time.Minute,
		Concurrency:     20,
		RateLimit:       50,
		FollowRedirects: true,
		MaxRedirects:    3,
	})
	assert.NotNil(t, httpxClient, "HTTPX client should be created successfully")
}

func TestHTTPXProbeWithMockDomains(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create mock HTTPX client with known existing domains
	existingDomains := []string{"google.com", "github.com", "stackoverflow.com"}
	client := httpx.NewMockClient(existingDomains, 0)

	ctx := context.Background()

	// Test with a mix of domains that should exist and fake ones that shouldn't
	testDomains := []string{
		"google.com", // Should exist
		"this-domain-definitely-does-not-exist-12345.com", // Should not exist
		"github.com",                    // Should exist
		"stackoverflow.com",             // Should exist
		"another-fake-domain-67890.org", // Should not exist
	}

	// Probe the domains
	results, err := client.ProbeDomains(ctx, testDomains)
	require.NoError(t, err)
	assert.Len(t, results, len(testDomains))

	// Analyze results
	existingDomainsResult := make(map[string]bool)
	nonExistingDomains := make(map[string]bool)

	for _, result := range results {
		domain := client.ExtractDomainFromURL(result.URL)
		if result.Exists {
			existingDomainsResult[domain] = true
			assert.Equal(t, 200, result.StatusCode, "Existing domain %s should have status code 200", domain)
			assert.Empty(t, result.Error, "Existing domain %s should not have error", domain)
		} else {
			nonExistingDomains[domain] = true
			assert.Equal(t, 0, result.StatusCode, "Non-existing domain %s should have no status code", domain)
			assert.NotEmpty(t, result.Error, "Non-existing domain %s should have error", domain)
		}
	}

	// Verify expected results
	expectedExisting := []string{"google.com", "github.com", "stackoverflow.com"}
	expectedNonExisting := []string{"this-domain-definitely-does-not-exist-12345.com", "another-fake-domain-67890.org"}

	for _, domain := range expectedExisting {
		assert.True(t, existingDomainsResult[domain], "Domain %s should exist", domain)
	}

	for _, domain := range expectedNonExisting {
		assert.True(t, nonExistingDomains[domain], "Domain %s should not exist", domain)
	}

	// Test filtering functionality
	filteredDomains, err := client.FilterExistingDomains(ctx, testDomains)
	require.NoError(t, err)

	// Should only contain existing domains
	assert.Len(t, filteredDomains, 3)
	assert.Contains(t, filteredDomains, "google.com")
	assert.Contains(t, filteredDomains, "github.com")
	assert.Contains(t, filteredDomains, "stackoverflow.com")
}

func TestHTTPXProbePerformance(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create mock HTTPX client with performance testing
	existingDomains := []string{"google.com", "github.com", "stackoverflow.com"}
	client := httpx.NewMockClient(existingDomains, 10*time.Millisecond) // Small delay to simulate processing

	ctx := context.Background()

	// Create a larger list of domains for performance testing
	var testDomains []string
	for i := 0; i < 20; i++ {
		testDomains = append(testDomains, "google.com")
		testDomains = append(testDomains, "github.com")
		testDomains = append(testDomains, "stackoverflow.com")
	}

	// Measure performance
	start := time.Now()
	results, err := client.ProbeDomains(ctx, testDomains)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Len(t, results, len(testDomains))

	// Performance assertions - should complete quickly with mock
	assert.Less(t, duration, 5*time.Second, "Mock probe should complete within 5 seconds")

	// Calculate throughput
	throughput := float64(len(testDomains)) / duration.Seconds()
	t.Logf("Probed %d domains in %v (%.2f domains/second)", len(testDomains), duration, throughput)

	// Should achieve reasonable throughput even with mock delay
	assert.Greater(t, throughput, 1.0, "Should achieve at least 1 domain/second")
}

func TestHTTPXProbeWithContextTimeout(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create mock HTTPX client with delay
	client := httpx.NewMockClient([]string{"google.com", "github.com", "stackoverflow.com"}, 2*time.Second)

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	testDomains := []string{"google.com", "github.com", "stackoverflow.com"}

	// This should timeout due to the short context timeout
	_, err := client.ProbeDomains(ctx, testDomains)
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestHTTPXConfigurationValidation(t *testing.T) {
	// Helper function to create a valid base config
	createValidBaseConfig := func() *config.Config {
		return &config.Config{
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
					Username:  "h1_user",
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
				Timeouts: config.TimeoutConfig{
					ProgramProcess: 45 * time.Minute,
					ChaosDiscovery: 30 * time.Minute,
				},
			},
		}
	}

	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid HTTPX configuration",
			config: func() *config.Config {
				cfg := createValidBaseConfig()
				cfg.Discovery.HTTPX = config.HTTPXConfig{
					Enabled:         true,
					Timeout:         30 * time.Second,
					Concurrency:     25,
					RateLimit:       100,
					FollowRedirects: true,
					MaxRedirects:    3,
					Debug:           false,
				}
				return cfg
			}(),
			expectError: false,
		},
		{
			name: "invalid timeout",
			config: func() *config.Config {
				cfg := createValidBaseConfig()
				cfg.Discovery.HTTPX = config.HTTPXConfig{
					Enabled:         true,
					Timeout:         0, // Invalid
					Concurrency:     25,
					RateLimit:       100,
					FollowRedirects: true,
					MaxRedirects:    3,
					Debug:           false,
				}
				return cfg
			}(),
			expectError: true,
			errorMsg:    "HTTPX_TIMEOUT must be greater than 0",
		},
		{
			name: "invalid concurrency",
			config: func() *config.Config {
				cfg := createValidBaseConfig()
				cfg.Discovery.HTTPX = config.HTTPXConfig{
					Enabled:         true,
					Timeout:         30 * time.Second,
					Concurrency:     0, // Invalid
					RateLimit:       100,
					FollowRedirects: true,
					MaxRedirects:    3,
					Debug:           false,
				}
				return cfg
			}(),
			expectError: true,
			errorMsg:    "HTTPX_CONCURRENCY must be between 1 and 100",
		},
		{
			name: "invalid rate limit",
			config: func() *config.Config {
				cfg := createValidBaseConfig()
				cfg.Discovery.HTTPX = config.HTTPXConfig{
					Enabled:         true,
					Timeout:         30 * time.Second,
					Concurrency:     25,
					RateLimit:       0, // Invalid
					FollowRedirects: true,
					MaxRedirects:    3,
					Debug:           false,
				}
				return cfg
			}(),
			expectError: true,
			errorMsg:    "HTTPX_RATE_LIMIT must be greater than 0",
		},
		{
			name: "invalid max redirects",
			config: func() *config.Config {
				cfg := createValidBaseConfig()
				cfg.Discovery.HTTPX = config.HTTPXConfig{
					Enabled:         true,
					Timeout:         30 * time.Second,
					Concurrency:     25,
					RateLimit:       100,
					FollowRedirects: true,
					MaxRedirects:    15, // Invalid (> 10)
					Debug:           false,
				}
				return cfg
			}(),
			expectError: true,
			errorMsg:    "HTTPX_MAX_REDIRECTS must be between 0 and 10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
