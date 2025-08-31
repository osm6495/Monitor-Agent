package httpx

import (
	"context"
	"strings"
	"time"
)

// MockClient is a mock implementation of the HTTPX client for testing
type MockClient struct {
	existingDomains map[string]bool
	delay           time.Duration
}

// NewMockClient creates a new mock HTTPX client
func NewMockClient(existingDomains []string, delay time.Duration) *MockClient {
	domainMap := make(map[string]bool)
	for _, domain := range existingDomains {
		domainMap[domain] = true
	}

	return &MockClient{
		existingDomains: domainMap,
		delay:           delay,
	}
}

// ProbeDomains simulates probing domains without making real network requests
func (m *MockClient) ProbeDomains(ctx context.Context, domains []string) ([]ProbeResult, error) {
	if len(domains) == 0 {
		return []ProbeResult{}, nil
	}

	// Simulate some processing time
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	var results []ProbeResult
	for _, domain := range domains {
		// Clean domain and add protocol
		cleanDomain := strings.TrimSpace(domain)
		if cleanDomain == "" {
			continue
		}

		// Add protocol if not present
		url := cleanDomain
		if !strings.HasPrefix(cleanDomain, "http://") && !strings.HasPrefix(cleanDomain, "https://") {
			url = "https://" + cleanDomain
		}

		// Check if domain exists in our mock data
		exists := m.existingDomains[cleanDomain]
		if !exists {
			// Also check without protocol
			cleanDomainNoProtocol := strings.TrimPrefix(cleanDomain, "https://")
			cleanDomainNoProtocol = strings.TrimPrefix(cleanDomainNoProtocol, "http://")
			exists = m.existingDomains[cleanDomainNoProtocol]
		}

		result := ProbeResult{
			URL:    url,
			Exists: exists,
		}

		if exists {
			result.StatusCode = 200 // Mock successful response
		} else {
			result.Error = "Domain does not exist or is unreachable"
		}

		results = append(results, result)
	}

	return results, nil
}

// FilterExistingDomains filters domains using the mock probe
func (m *MockClient) FilterExistingDomains(ctx context.Context, domains []string) ([]string, error) {
	probeResults, err := m.ProbeDomains(ctx, domains)
	if err != nil {
		return nil, err
	}

	var existingDomains []string
	for _, result := range probeResults {
		if result.Exists {
			// Extract domain from URL
			domain := m.ExtractDomainFromURL(result.URL)
			if domain != "" {
				existingDomains = append(existingDomains, domain)
			}
		}
	}

	return existingDomains, nil
}

// ExtractDomainFromURL extracts the domain from a URL
func (m *MockClient) ExtractDomainFromURL(urlStr string) string {
	// Remove protocol
	urlStr = strings.TrimPrefix(urlStr, "https://")
	urlStr = strings.TrimPrefix(urlStr, "http://")

	// Remove path and query parameters
	if idx := strings.Index(urlStr, "/"); idx != -1 {
		urlStr = urlStr[:idx]
	}

	// Remove query parameters (after ?)
	if idx := strings.Index(urlStr, "?"); idx != -1 {
		urlStr = urlStr[:idx]
	}

	// Remove port if present
	if idx := strings.Index(urlStr, ":"); idx != -1 {
		urlStr = urlStr[:idx]
	}

	return strings.TrimSpace(urlStr)
}
