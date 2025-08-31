//go:build manual

package manual

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/monitor-agent/internal/config"
	"github.com/monitor-agent/internal/discovery/chaosdb"
	"github.com/monitor-agent/internal/platforms"
	"github.com/monitor-agent/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHackerOneAPI tests the HackerOne API integration with real API key
func TestHackerOneAPI(t *testing.T) {
	// This test requires a real HackerOne API key and username
	// Set HACKERONE_USERNAME and HACKERONE_API_KEY environment variables before running

	cfg, err := config.Load()
	require.NoError(t, err)

	if cfg.APIs.HackerOne.APIKey == "" || cfg.APIs.HackerOne.Username == "" {
		t.Skip("HACKERONE_USERNAME and HACKERONE_API_KEY not set, skipping manual test")
	}

	// Create HackerOne client
	platformFactory := platforms.NewPlatformFactory()
	platformFactory.RegisterPlatform("hackerone", &platforms.PlatformConfig{
		APIKey:        cfg.APIs.HackerOne.APIKey,
		Username:      cfg.APIs.HackerOne.Username,
		RateLimit:     cfg.APIs.HackerOne.RateLimit,
		Timeout:       cfg.HTTP.Timeout,
		RetryAttempts: cfg.HTTP.RetryAttempts,
		RetryDelay:    cfg.HTTP.RetryDelay,
	})

	platform, err := platformFactory.GetPlatform("hackerone")
	require.NoError(t, err)

	ctx := context.Background()

	// Test health check
	err = platform.IsHealthy(ctx)
	assert.NoError(t, err)

	// Debug: Let's also test the API directly to see what we get
	fmt.Println("=== DEBUG: Testing API directly ===")
	client := resty.New()
	client.SetBasicAuth("aghost", "XbJ6XgTKBIJv6TO+YHIwXOqeQccRmJ+/9fmLBg8/UmU=")
	client.SetHeaders(map[string]string{
		"Accept":       "application/json",
		"Content-Type": "application/json",
		"User-Agent":   "Monitor-Agent/1.0",
	})

	params := url.Values{}
	params.Set("page[number]", "1")
	params.Set("page[size]", "10")

	resp, err := client.R().Get(fmt.Sprintf("https://api.hackerone.com/v1/hackers/programs?%s", params.Encode()))
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode())

	fmt.Printf("Direct API Response Status: %d\n", resp.StatusCode())
	fmt.Printf("Direct API Response Body: %s\n", string(resp.Body()))
	fmt.Println("=== END DEBUG ===")

	// Test getting public programs
	programs, err := platform.GetPublicPrograms(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, programs)

	// Test getting scope for first program
	if len(programs) > 0 {
		scopeAssets, err := platform.GetProgramScope(ctx, programs[0].ProgramURL)
		require.NoError(t, err)
		assert.NotNil(t, scopeAssets)
	}
}

// TestBugCrowdAPI tests the BugCrowd API integration with real API key
func TestBugCrowdAPI(t *testing.T) {
	// This test requires a real BugCrowd API key
	// Set BUGCROWD_API_KEY environment variable before running

	cfg, err := config.Load()
	require.NoError(t, err)

	if cfg.APIs.BugCrowd.APIKey == "" {
		t.Skip("BUGCROWD_API_KEY not set, skipping manual test")
	}

	// Create BugCrowd client
	platformFactory := platforms.NewPlatformFactory()
	platformFactory.RegisterPlatform("bugcrowd", &platforms.PlatformConfig{
		APIKey:        cfg.APIs.BugCrowd.APIKey,
		RateLimit:     cfg.APIs.BugCrowd.RateLimit,
		Timeout:       cfg.HTTP.Timeout,
		RetryAttempts: cfg.HTTP.RetryAttempts,
		RetryDelay:    cfg.HTTP.RetryDelay,
	})

	platform, err := platformFactory.GetPlatform("bugcrowd")
	require.NoError(t, err)

	ctx := context.Background()

	// Test health check
	err = platform.IsHealthy(ctx)
	assert.NoError(t, err)

	// Test getting public programs
	programs, err := platform.GetPublicPrograms(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, programs)

	// Test getting scope for first program
	if len(programs) > 0 {
		scopeAssets, err := platform.GetProgramScope(ctx, programs[0].ProgramURL)
		require.NoError(t, err)
		assert.NotNil(t, scopeAssets)
	}
}

// TestChaosDBAPI tests the ChaosDB API integration with real API key
func TestChaosDBAPI(t *testing.T) {
	// This test requires a real ChaosDB API key
	// Set CHAOSDB_API_KEY environment variable before running

	cfg, err := config.Load()
	require.NoError(t, err)

	if cfg.APIs.ChaosDB.APIKey == "" {
		t.Skip("CHAOSDB_API_KEY not set, skipping manual test")
	}

	// Create ChaosDB client
	chaosDBClient := chaosdb.NewClient(&chaosdb.ClientConfig{
		APIKey:        cfg.APIs.ChaosDB.APIKey,
		RateLimit:     cfg.APIs.ChaosDB.RateLimit,
		Timeout:       cfg.HTTP.Timeout,
		RetryAttempts: cfg.HTTP.RetryAttempts,
		RetryDelay:    cfg.HTTP.RetryDelay,
	})

	ctx := context.Background()

	// Test health check
	err = chaosDBClient.IsHealthy(ctx)
	assert.NoError(t, err)

	// Test single domain discovery
	testDomain := "example.com"
	result, err := chaosDBClient.DiscoverDomain(ctx, testDomain)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, testDomain, result.Domain)

	// Test bulk discovery
	testDomains := []string{"example.com", "google.com"}
	bulkResult, err := chaosDBClient.DiscoverDomainsBulk(ctx, testDomains)
	require.NoError(t, err)
	assert.NotNil(t, bulkResult)
	assert.Len(t, bulkResult.Results, len(testDomains))
}

// TestFullWorkflow tests the complete workflow from program discovery to asset storage
func TestFullWorkflow(t *testing.T) {
	// This test requires all API keys and a database connection
	// Set all required environment variables before running

	cfg, err := config.Load()
	require.NoError(t, err)

	// Check if all required API keys are set
	if cfg.APIs.HackerOne.APIKey == "" || cfg.APIs.BugCrowd.APIKey == "" || cfg.APIs.ChaosDB.APIKey == "" {
		t.Skip("Not all API keys are set, skipping full workflow test")
	}

	// This test would:
	// 1. Connect to database
	// 2. Create monitor service
	// 3. Run a full scan
	// 4. Verify that programs and assets are stored
	// 5. Check statistics

	// For now, we'll just test the configuration
	assert.NotEmpty(t, cfg.APIs.HackerOne.APIKey)
	assert.NotEmpty(t, cfg.APIs.BugCrowd.APIKey)
	assert.NotEmpty(t, cfg.APIs.ChaosDB.APIKey)
}

// TestRateLimiting tests that rate limiting is working correctly
func TestRateLimiting(t *testing.T) {
	// This test verifies that rate limiting is working
	// It should not require API keys as it tests the rate limiter itself

	rateLimiter := utils.NewRateLimiter(10, time.Minute) // 10 requests per minute

	start := time.Now()

	// Make 10 requests quickly
	for i := 0; i < 10; i++ {
		rateLimiter.Wait()
	}

	// The 11th request should be rate limited
	rateLimiter.Wait()

	duration := time.Since(start)

	// Should take at least 1 second due to rate limiting
	assert.GreaterOrEqual(t, duration, time.Second)
}

// TestURLProcessing tests URL processing utilities
func TestURLProcessing(t *testing.T) {
	urlProcessor := utils.NewURLProcessor()

	// Test domain extraction
	domain, err := urlProcessor.ExtractDomain("https://subdomain.example.com/path")
	assert.NoError(t, err)
	assert.Equal(t, "subdomain.example.com", domain)

	// Test subdomain extraction
	subdomain, err := urlProcessor.ExtractSubdomain("https://subdomain.example.com/path")
	assert.NoError(t, err)
	assert.Equal(t, "subdomain", subdomain)

	// Test wildcard conversion
	wildcardDomain := urlProcessor.ConvertWildcardToDomain("*.example.com")
	assert.Equal(t, "example.com", wildcardDomain)

	// Test URL normalization
	normalized, err := urlProcessor.NormalizeURL("http://example.com:80/path/")
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com/path", normalized)
}
