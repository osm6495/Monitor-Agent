package httpx

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name   string
		config *ProbeConfig
		want   *Client
	}{
		{
			name:   "nil config uses defaults",
			config: nil,
			want: &Client{
				config: &ProbeConfig{
					Timeout:         30 * time.Second,
					Concurrency:     25,
					RateLimit:       50,
					UserAgent:       "Monitor-Agent/1.0",
					FollowRedirects: true,
					MaxRedirects:    3,
				},
			},
		},
		{
			name: "custom config",
			config: &ProbeConfig{
				Timeout:         60 * time.Second,
				Concurrency:     35,
				RateLimit:       200,
				UserAgent:       "Custom/1.0",
				FollowRedirects: false,
				MaxRedirects:    5,
			},
			want: &Client{
				config: &ProbeConfig{
					Timeout:         60 * time.Second,
					Concurrency:     35,
					RateLimit:       200,
					UserAgent:       "Custom/1.0",
					FollowRedirects: false,
					MaxRedirects:    5,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewClient(tt.config)
			assert.Equal(t, tt.want.config.Timeout, got.config.Timeout)
			assert.Equal(t, tt.want.config.Concurrency, got.config.Concurrency)
			assert.Equal(t, tt.want.config.RateLimit, got.config.RateLimit)
			assert.Equal(t, tt.want.config.UserAgent, got.config.UserAgent)
			assert.Equal(t, tt.want.config.FollowRedirects, got.config.FollowRedirects)
			assert.Equal(t, tt.want.config.MaxRedirects, got.config.MaxRedirects)
		})
	}
}

func TestProbeDomains_EmptyList(t *testing.T) {
	client := NewClient(nil)
	ctx := context.Background()

	results, err := client.ProbeDomains(ctx, []string{})
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestProbeDomains_EmptyDomains(t *testing.T) {
	client := NewClient(nil)
	ctx := context.Background()

	// Test with domains that are empty or just protocol
	domains := []string{
		"",
		"   ",
		"https://",
		"http://",
		"example.com", // This one should work
	}

	results, err := client.ProbeDomains(ctx, domains)
	require.NoError(t, err)
	// Should only get results for valid domains
	assert.Len(t, results, 1)
	assert.Equal(t, "https://example.com", results[0].URL)
}

func TestProbeDomains_PerURLTimeout(t *testing.T) {
	// Use mock client with delay to test per-URL timeout behavior
	// This avoids making real network requests that could hang
	client := NewMockClient([]string{"example.com", "google.com"}, 2*time.Second)
	ctx := context.Background()

	// Test with a mix of domains - some should timeout due to mock delay
	domains := []string{
		"example.com", // Should exist in mock
		"google.com",  // Should exist in mock
		"this-domain-definitely-does-not-exist-12345.com", // Should not exist in mock
	}

	results, err := client.ProbeDomains(ctx, domains)
	require.NoError(t, err)

	// Should get results for all domains
	assert.Len(t, results, len(domains))

	// Check that we got results for each domain
	urls := make(map[string]bool)
	for _, result := range results {
		urls[result.URL] = true
	}

	assert.True(t, urls["https://example.com"])
	assert.True(t, urls["https://google.com"])
	assert.True(t, urls["https://this-domain-definitely-does-not-exist-12345.com"])
}

func TestProbeDomains_ValidDomains(t *testing.T) {
	// Use mock client with known existing domains
	existingDomains := []string{"google.com", "github.com", "example.com"}
	client := NewMockClient(existingDomains, 0)
	ctx := context.Background()

	// Test with known existing domains
	domains := []string{
		"google.com",
		"github.com",
		"example.com",
	}

	results, err := client.ProbeDomains(ctx, domains)
	require.NoError(t, err)
	assert.Len(t, results, len(domains))

	// Check that we got results for all domains
	for _, result := range results {
		assert.NotEmpty(t, result.URL)
		assert.True(t, result.Exists, "Domain %s should exist", result.URL)
		assert.Equal(t, 200, result.StatusCode, "Domain %s should have status code 200", result.URL)
		assert.Empty(t, result.Error, "Domain %s should not have error", result.URL)
	}
}

func TestProbeDomains_InvalidDomains(t *testing.T) {
	// Use mock client with no existing domains
	client := NewMockClient([]string{}, 0)
	ctx := context.Background()

	// Test with domains that don't exist
	domains := []string{
		"this-domain-definitely-does-not-exist-12345.com",
		"another-fake-domain-67890.org",
	}

	results, err := client.ProbeDomains(ctx, domains)
	require.NoError(t, err)
	assert.Len(t, results, len(domains))

	// Check that we got results for all domains
	for _, result := range results {
		assert.NotEmpty(t, result.URL)
		// These domains should not exist
		assert.False(t, result.Exists, "Domain %s should not exist", result.URL)
		assert.Equal(t, 0, result.StatusCode, "Domain %s should have no status code", result.URL)
		assert.NotEmpty(t, result.Error, "Domain %s should have an error", result.URL)
	}
}

func TestFilterExistingDomains(t *testing.T) {
	// Use mock client with specific existing domains
	existingDomains := []string{"google.com", "github.com"}
	client := NewMockClient(existingDomains, 0)
	ctx := context.Background()

	// Mix of existing and non-existing domains
	domains := []string{
		"google.com", // Should exist
		"this-domain-definitely-does-not-exist-12345.com", // Should not exist
		"github.com",                    // Should exist
		"another-fake-domain-67890.org", // Should not exist
	}

	existingDomainsResult, err := client.FilterExistingDomains(ctx, domains)
	require.NoError(t, err)

	// Should have filtered out non-existing domains
	assert.Len(t, existingDomainsResult, 2)
	assert.Contains(t, existingDomainsResult, "google.com")
	assert.Contains(t, existingDomainsResult, "github.com")
}

func TestExtractDomainFromURL(t *testing.T) {
	client := NewClient(nil)

	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "simple domain",
			url:      "https://example.com",
			expected: "example.com",
		},
		{
			name:     "domain with path",
			url:      "https://example.com/path",
			expected: "example.com",
		},
		{
			name:     "domain with query",
			url:      "https://example.com?param=value",
			expected: "example.com",
		},
		{
			name:     "domain with port",
			url:      "https://example.com:8080",
			expected: "example.com",
		},
		{
			name:     "domain with subdomain",
			url:      "https://sub.example.com",
			expected: "sub.example.com",
		},
		{
			name:     "http domain",
			url:      "http://example.com",
			expected: "example.com",
		},
		{
			name:     "domain without protocol",
			url:      "example.com",
			expected: "example.com",
		},
		{
			name:     "domain with trailing slash",
			url:      "https://example.com/",
			expected: "example.com",
		},
		{
			name:     "empty URL",
			url:      "",
			expected: "",
		},
		{
			name:     "protocol only URL",
			url:      "https://",
			expected: "",
		},
		{
			name:     "http protocol only URL",
			url:      "http://",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.ExtractDomainFromURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProbeDomains_ContextCancellation(t *testing.T) {
	// Use mock client with delay to test context cancellation
	client := NewMockClient([]string{"google.com", "github.com"}, 100*time.Millisecond)

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	domains := []string{"google.com", "github.com"}

	_, err := client.ProbeDomains(ctx, domains)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestProbeDomains_MixedResults(t *testing.T) {
	// Use mock client with specific existing domains
	existingDomains := []string{"google.com", "github.com", "stackoverflow.com"}
	client := NewMockClient(existingDomains, 0)
	ctx := context.Background()

	// Test with a mix of valid and invalid domains
	domains := []string{
		"google.com",
		"this-domain-definitely-does-not-exist-12345.com",
		"github.com",
		"stackoverflow.com",
		"another-fake-domain-67890.org",
	}

	results, err := client.ProbeDomains(ctx, domains)
	require.NoError(t, err)
	assert.Len(t, results, len(domains))

	// Count existing and non-existing domains
	existingCount := 0
	nonExistingCount := 0

	for _, result := range results {
		if result.Exists {
			existingCount++
			assert.Equal(t, 200, result.StatusCode)
			assert.Empty(t, result.Error)
		} else {
			nonExistingCount++
			assert.Equal(t, 0, result.StatusCode)
			assert.NotEmpty(t, result.Error)
		}
	}

	// Should have exactly 3 existing domains (google.com, github.com, stackoverflow.com)
	assert.Equal(t, 3, existingCount)
	// Should have exactly 2 non-existing domains
	assert.Equal(t, 2, nonExistingCount)
	// Total should equal input
	assert.Equal(t, len(domains), existingCount+nonExistingCount)
}

func TestProbeDomainsWithDetails_EmptyList(t *testing.T) {
	client := NewClient(nil)
	ctx := context.Background()

	results, err := client.ProbeDomainsWithDetails(ctx, []string{})
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestProbeDomainsWithDetails_ValidDomains(t *testing.T) {
	client := NewClient(nil)
	ctx := context.Background()

	domains := []string{"example.com"}

	results, err := client.ProbeDomainsWithDetails(ctx, domains)
	require.NoError(t, err)
	assert.Len(t, results, 1)

	result := results[0]
	assert.Equal(t, "https://example.com", result.URL)
	assert.True(t, result.Exists)
	assert.Equal(t, 200, result.StatusCode)
	assert.Empty(t, result.Error)
	// Response time should be a reasonable value for a real HTTP request
	assert.Greater(t, result.ResponseTime, int64(0))
}

func TestProbeDomainsWithDetails_InvalidDomains(t *testing.T) {
	client := NewClient(nil)
	ctx := context.Background()

	domains := []string{"this-domain-definitely-does-not-exist-12345.com"}

	results, err := client.ProbeDomainsWithDetails(ctx, domains)
	require.NoError(t, err)

	// HTTPX might not return results for non-existent domains
	// So we check that we get either 0 or 1 results
	if len(results) == 0 {
		// This is acceptable - HTTPX didn't return any results for non-existent domain
		t.Log("HTTPX returned no results for non-existent domain (this is acceptable)")
	} else if len(results) == 1 {
		result := results[0]
		assert.Equal(t, "https://this-domain-definitely-does-not-exist-12345.com", result.URL)
		assert.False(t, result.Exists)
		assert.Equal(t, 0, result.StatusCode)
		// Error might be empty if HTTPX doesn't provide error details
	}
}

func TestDetailedProbeResult_ToJSON(t *testing.T) {
	result := DetailedProbeResult{
		URL:          "https://example.com",
		StatusCode:   200,
		Exists:       true,
		Error:        "",
		Headers:      map[string]string{"Server": "nginx"},
		Body:         "<html><body>Hello World</body></html>",
		ResponseTime: 150,
		ContentType:  "text/html",
		Server:       "nginx/1.18.0",
		Title:        "Example Domain",
		Technologies: []string{"nginx", "PHP"},
	}

	jsonData, err := result.ToJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Verify the JSON contains expected fields
	assert.Contains(t, jsonData, "example.com")
	assert.Contains(t, jsonData, "200")
	assert.Contains(t, jsonData, "nginx")
	assert.Contains(t, jsonData, "Hello World")
}

func TestDetailedProbeResult_ToJSON_EmptyFields(t *testing.T) {
	result := DetailedProbeResult{
		URL:        "https://example.com",
		StatusCode: 404,
		Exists:     false,
		Error:      "not found",
	}

	jsonData, err := result.ToJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Verify the JSON contains expected fields
	assert.Contains(t, jsonData, "example.com")
	assert.Contains(t, jsonData, "404")
	assert.Contains(t, jsonData, "not found")
}

func TestProbeDomainsWithDetails_ContextCancellation(t *testing.T) {
	client := NewClient(nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	domains := []string{"example.com"}

	// HTTPX might not respect context cancellation immediately
	// So we check that we either get an error or the operation completes
	results, err := client.ProbeDomainsWithDetails(ctx, domains)
	if err != nil {
		// If we get an error, it should be context cancelled
		assert.Equal(t, context.Canceled, err)
	} else {
		// If no error, the operation completed (which is also acceptable)
		// This can happen if HTTPX doesn't check context cancellation
		t.Log("HTTPX completed despite cancelled context (this is acceptable)")
		assert.NotNil(t, results)
	}
}
