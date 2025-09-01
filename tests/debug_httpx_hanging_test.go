package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/monitor-agent/internal/discovery/chaosdb"
	"github.com/monitor-agent/internal/discovery/httpx"
	"github.com/stretchr/testify/assert"
)

// TestHTTPXProbeWithRealisticChaosDBData tests HTTPX probes using realistic ChaosDB output
func TestHTTPXProbeWithRealisticChaosDBData(t *testing.T) {
	// Create realistic ChaosDB output based on the Slack example from the test data
	realisticChaosDBOutput := &chaosdb.BulkDiscoveryResult{
		Results: []chaosdb.DiscoveryResult{
			{
				Domain:       "slack.com",
				Subdomains:   []string{"api", "app", "status", "edgeapi", "hooks", "files", "rtm"},
				Count:        7,
				DiscoveredAt: time.Now(),
			},
			{
				Domain:       "slack-status.com",
				Subdomains:   []string{"www", "status", "api"},
				Count:        3,
				DiscoveredAt: time.Now(),
			},
			{
				Domain:       "slackhq.com",
				Subdomains:   []string{"www", "brand", "campaign", "investor", "blog"},
				Count:        5,
				DiscoveredAt: time.Now(),
			},
		},
		TotalCount:   15,
		ErrorCount:   0,
		DiscoveredAt: time.Now(),
	}

	// Extract all subdomains for HTTPX probing
	var allSubdomains []string
	for _, result := range realisticChaosDBOutput.Results {
		for _, subdomain := range result.Subdomains {
			fullDomain := fmt.Sprintf("%s.%s", subdomain, result.Domain)
			allSubdomains = append(allSubdomains, fullDomain)
		}
	}

	t.Logf("Testing HTTPX probe with %d realistic subdomains from ChaosDB", len(allSubdomains))
	t.Logf("Subdomains: %v", allSubdomains)

	// Test with 30 second timeout (as requested)
	testHTTPXProbeWithTimeout(t, allSubdomains, 30*time.Second)
}

// testHTTPXProbeWithTimeout tests HTTPX probe with a specific timeout
func testHTTPXProbeWithTimeout(t *testing.T, domains []string, timeout time.Duration) {
	t.Logf("Testing HTTPX probe with %v timeout for %d domains", timeout, len(domains))

	// Create HTTPX client with the specified timeout
	httpxClient := httpx.NewClient(&httpx.ProbeConfig{
		Timeout:         timeout,
		Concurrency:     25,
		RateLimit:       100,
		FollowRedirects: true,
		MaxRedirects:    3,
		Debug:           false, // Disable debug logging for cleaner test output
	})

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout+5*time.Second)
	defer cancel()

	// Start timing
	start := time.Now()

	// Run the probe
	results, err := httpxClient.ProbeDomains(ctx, domains)
	duration := time.Since(start)

	// Log results
	t.Logf("HTTPX probe completed in %v", duration)
	t.Logf("Expected %d results, got %d results", len(domains), len(results))

	if err != nil {
		t.Logf("HTTPX probe error: %v", err)
	}

	// Analyze results
	existingCount := 0
	nonExistingCount := 0
	errorCount := 0

	for _, result := range results {
		if result.Exists {
			existingCount++
		} else if result.Error != "" {
			errorCount++
		} else {
			nonExistingCount++
		}
	}

	t.Logf("Results breakdown: %d existing, %d non-existing, %d errors", existingCount, nonExistingCount, errorCount)

	// Key assertions to identify hanging issues
	assert.LessOrEqual(t, duration, timeout+10*time.Second, "HTTPX probe should complete within timeout + buffer")
	assert.Greater(t, len(results), 0, "Should get some results")

	// Check if we're getting stuck waiting for results
	if duration > timeout {
		t.Logf("WARNING: HTTPX probe took longer than expected timeout (%v > %v)", duration, timeout)
		t.Logf("This may indicate hanging behavior")
	}
}

// TestHTTPXProbeWithLargeDataset tests HTTPX probe with a larger dataset to stress test
func TestHTTPXProbeWithLargeDataset(t *testing.T) {
	// Create a larger dataset to stress test the HTTPX probe
	largeDataset := generateLargeTestDataset(100) // 100 domains

	t.Logf("Testing HTTPX probe with large dataset: %d domains", len(largeDataset))

	// Test with different concurrency levels
	concurrencyLevels := []int{10, 25, 50, 100}

	for _, concurrency := range concurrencyLevels {
		t.Run(fmt.Sprintf("Concurrency_%d", concurrency), func(t *testing.T) {
			testHTTPXProbeWithConcurrency(t, largeDataset, concurrency)
		})
	}
}

// testHTTPXProbeWithConcurrency tests HTTPX probe with different concurrency levels
func testHTTPXProbeWithConcurrency(t *testing.T, domains []string, concurrency int) {
	t.Logf("Testing HTTPX probe with concurrency %d for %d domains", concurrency, len(domains))

	// Create HTTPX client with specific concurrency
	httpxClient := httpx.NewClient(&httpx.ProbeConfig{
		Timeout:         30 * time.Second,
		Concurrency:     concurrency,
		RateLimit:       100,
		FollowRedirects: true,
		MaxRedirects:    3,
		Debug:           false, // Disable debug logging for cleaner test output
	})

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Start timing
	start := time.Now()

	// Run the probe
	results, err := httpxClient.ProbeDomains(ctx, domains)
	duration := time.Since(start)

	// Log results
	t.Logf("HTTPX probe with concurrency %d completed in %v", concurrency, duration)
	t.Logf("Expected %d results, got %d results", len(domains), len(results))

	if err != nil {
		t.Logf("HTTPX probe error: %v", err)
	}

	// Analyze performance
	throughput := float64(len(results)) / duration.Seconds()
	t.Logf("Throughput: %.2f domains/second", throughput)

	// Assertions - now we expect some results since we're using real domains
	assert.LessOrEqual(t, duration, 60*time.Second, "Should complete within 60 seconds")
	assert.Greater(t, len(results), 0, "Should get some results from real domains")

	// Check for hanging behavior
	if duration > 45*time.Second {
		t.Logf("WARNING: HTTPX probe with concurrency %d took very long (%v)", concurrency, duration)
		t.Logf("This may indicate hanging behavior at high concurrency")
	}
}

// generateLargeTestDataset generates a large dataset for stress testing
func generateLargeTestDataset(count int) []string {
	var domains []string

	// Use a mix of real domains and fake domains for realistic testing
	realDomains := []string{
		"google.com", "github.com", "stackoverflow.com", "reddit.com", "wikipedia.org",
		"amazon.com", "microsoft.com", "apple.com", "netflix.com", "youtube.com",
		"twitter.com", "facebook.com", "linkedin.com", "instagram.com", "discord.com",
		"slack.com", "zoom.us", "dropbox.com", "spotify.com", "twitch.tv",
	}

	// Generate some fake domains to test error handling
	fakeDomains := []string{
		"this-domain-definitely-does-not-exist-12345.com",
		"another-fake-domain-67890.org",
		"fake-subdomain.test-domain.net",
		"nonexistent.example.io",
		"invalid-domain-name-12345.co.uk",
	}

	// Mix real and fake domains
	for i := 0; i < count; i++ {
		if i < len(realDomains) {
			// Use real domains for the first batch
			domains = append(domains, realDomains[i%len(realDomains)])
		} else {
			// Use fake domains for the rest
			fakeIndex := (i - len(realDomains)) % len(fakeDomains)
			domains = append(domains, fakeDomains[fakeIndex])
		}
	}

	return domains
}
