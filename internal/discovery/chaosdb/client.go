package chaosdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/monitor-agent/internal/utils"
	"github.com/sirupsen/logrus"
)

const (
	baseURL = "https://dns.projectdiscovery.io/dns"
)

// Client represents a ChaosDB API client
type Client struct {
	httpClient   *resty.Client
	apiKey       string
	rateLimiter  *utils.RateLimiter
	urlProcessor *utils.URLProcessor
}

// ClientConfig holds configuration for the ChaosDB client
type ClientConfig struct {
	APIKey        string
	RateLimit     int
	Timeout       time.Duration
	RetryAttempts int
	RetryDelay    time.Duration
}

// NewClient creates a new ChaosDB client
func NewClient(config *ClientConfig) *Client {
	client := resty.New()
	client.SetTimeout(config.Timeout)
	client.SetRetryCount(config.RetryAttempts)
	client.SetRetryWaitTime(config.RetryDelay)
	client.SetRetryMaxWaitTime(config.RetryDelay * 2)

	// Set default headers
	client.SetHeaders(map[string]string{
		"Accept":       "application/json",
		"Content-Type": "application/json",
		"User-Agent":   "Monitor-Agent/1.0",
	})

	// Add API key if provided
	if config.APIKey != "" {
		client.SetHeader("Authorization", fmt.Sprintf("Bearer %s", config.APIKey))
	}

	return &Client{
		httpClient:   client,
		apiKey:       config.APIKey,
		rateLimiter:  utils.NewRateLimiter(config.RateLimit, time.Minute),
		urlProcessor: utils.NewURLProcessor(),
	}
}

// DiscoverDomain discovers subdomains for a single domain
func (c *Client) DiscoverDomain(ctx context.Context, domain string) (*DiscoveryResult, error) {
	c.rateLimiter.Wait()

	// Clean and normalize domain
	cleanDomain, err := c.urlProcessor.ExtractDomain(domain)
	if err != nil {
		return nil, fmt.Errorf("failed to extract domain from %s: %w", domain, err)
	}

	// Remove protocol and path
	cleanDomain = strings.TrimPrefix(cleanDomain, "http://")
	cleanDomain = strings.TrimPrefix(cleanDomain, "https://")
	if idx := strings.Index(cleanDomain, "/"); idx != -1 {
		cleanDomain = cleanDomain[:idx]
	}

	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetQueryParam("domain", cleanDomain).
		Get(baseURL)

	if err != nil {
		return nil, fmt.Errorf("failed to make request for domain %s: %w", cleanDomain, err)
	}

	if resp.StatusCode() != http.StatusOK {
		var errorResp ChaosDBError
		if err := json.Unmarshal(resp.Body(), &errorResp); err == nil {
			return nil, fmt.Errorf("ChaosDB API error for domain %s: %s", cleanDomain, errorResp.Message)
		}
		return nil, fmt.Errorf("ChaosDB API returned status %d for domain %s", resp.StatusCode(), cleanDomain)
	}

	var chaosResp ChaosDBResponse
	if err := json.Unmarshal(resp.Body(), &chaosResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response for domain %s: %w", cleanDomain, err)
	}

	// Check if there was an error in the response
	if chaosResp.Error != "" {
		return &DiscoveryResult{
			Domain:       cleanDomain,
			Subdomains:   []string{},
			Count:        0,
			DiscoveredAt: time.Now(),
			Error:        chaosResp.Error,
		}, nil
	}

	result := &DiscoveryResult{
		Domain:       cleanDomain,
		Subdomains:   chaosResp.Subdomains,
		Count:        chaosResp.Count,
		DiscoveredAt: time.Now(),
	}

	logrus.Infof("Discovered %d subdomains for domain %s", result.Count, cleanDomain)
	return result, nil
}

// DiscoverDomainsBulk discovers subdomains for multiple domains in bulk
// This uses the bulk endpoint for efficiency when available
func (c *Client) DiscoverDomainsBulk(ctx context.Context, domains []string) (*BulkDiscoveryResult, error) {
	if len(domains) == 0 {
		return &BulkDiscoveryResult{
			Results:      []DiscoveryResult{},
			TotalCount:   0,
			ErrorCount:   0,
			DiscoveredAt: time.Now(),
		}, nil
	}

	// Clean and normalize domains
	var cleanDomains []string
	for _, domain := range domains {
		cleanDomain, err := c.urlProcessor.ExtractDomain(domain)
		if err != nil {
			logrus.Warnf("Failed to extract domain from %s: %v", domain, err)
			continue
		}

		// Remove protocol and path
		cleanDomain = strings.TrimPrefix(cleanDomain, "http://")
		cleanDomain = strings.TrimPrefix(cleanDomain, "https://")
		if idx := strings.Index(cleanDomain, "/"); idx != -1 {
			cleanDomain = cleanDomain[:idx]
		}

		cleanDomains = append(cleanDomains, cleanDomain)
	}

	if len(cleanDomains) == 0 {
		return &BulkDiscoveryResult{
			Results:      []DiscoveryResult{},
			TotalCount:   0,
			ErrorCount:   0,
			DiscoveredAt: time.Now(),
		}, nil
	}

	// Try bulk endpoint first, fallback to concurrent if not available
	bulkResult, err := c.tryBulkEndpoint(ctx, cleanDomains)
	if err != nil {
		logrus.Warnf("Bulk endpoint failed, falling back to concurrent requests: %v", err)
		return c.DiscoverDomainsConcurrent(ctx, cleanDomains, 10)
	}

	return bulkResult, nil
}

// tryBulkEndpoint attempts to use the bulk endpoint for efficiency
func (c *Client) tryBulkEndpoint(ctx context.Context, domains []string) (*BulkDiscoveryResult, error) {
	c.rateLimiter.Wait()

	// Prepare bulk request
	bulkRequest := ChaosDBBulkRequest{
		Domains: domains,
	}

	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetBody(bulkRequest).
		Post(baseURL + "/bulk")

	if err != nil {
		return nil, fmt.Errorf("failed to make bulk request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		var errorResp ChaosDBError
		if err := json.Unmarshal(resp.Body(), &errorResp); err == nil {
			return nil, fmt.Errorf("ChaosDB API error: %s", errorResp.Message)
		}
		return nil, fmt.Errorf("ChaosDB API returned status %d", resp.StatusCode())
	}

	var bulkResp ChaosDBBulkResponse
	if err := json.Unmarshal(resp.Body(), &bulkResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal bulk response: %w", err)
	}

	// Convert to discovery results
	var results []DiscoveryResult
	totalCount := 0
	errorCount := 0

	for _, chaosResp := range bulkResp.Results {
		result := DiscoveryResult{
			Domain:       chaosResp.Domain,
			Subdomains:   chaosResp.Subdomains,
			Count:        chaosResp.Count,
			DiscoveredAt: time.Now(),
		}

		if chaosResp.Error != "" {
			result.Error = chaosResp.Error
			errorCount++
		} else {
			totalCount += chaosResp.Count
		}

		results = append(results, result)
	}

	// Add errors from bulk response
	for _, errorMsg := range bulkResp.Errors {
		errorCount++
		logrus.Warnf("ChaosDB bulk error: %s", errorMsg)
	}

	bulkResult := &BulkDiscoveryResult{
		Results:      results,
		TotalCount:   totalCount,
		ErrorCount:   errorCount,
		DiscoveredAt: time.Now(),
	}

	logrus.Infof("Bulk discovery completed: %d domains, %d total subdomains, %d errors",
		len(results), totalCount, errorCount)

	return bulkResult, nil
}

// DiscoverDomainsConcurrent discovers subdomains for multiple domains concurrently
func (c *Client) DiscoverDomainsConcurrent(ctx context.Context, domains []string, maxConcurrent int) (*BulkDiscoveryResult, error) {
	if len(domains) == 0 {
		return &BulkDiscoveryResult{
			Results:      []DiscoveryResult{},
			TotalCount:   0,
			ErrorCount:   0,
			DiscoveredAt: time.Now(),
		}, nil
	}

	// Limit concurrency
	if maxConcurrent <= 0 {
		maxConcurrent = 10 // Default concurrency limit
	}

	// Create semaphore for concurrency control
	semaphore := make(chan struct{}, maxConcurrent)
	results := make(chan *DiscoveryResult, len(domains))
	errors := make(chan error, len(domains))

	// Start goroutines for each domain
	for _, domain := range domains {
		go func(d string) {
			semaphore <- struct{}{} // Acquire semaphore
			defer func() {
				<-semaphore // Release semaphore
			}()

			result, err := c.DiscoverDomain(ctx, d)
			if err != nil {
				errors <- err
				return
			}
			results <- result
		}(domain)
	}

	// Collect results
	var discoveryResults []DiscoveryResult
	totalCount := 0
	errorCount := 0

	for i := 0; i < len(domains); i++ {
		select {
		case result := <-results:
			discoveryResults = append(discoveryResults, *result)
			if result.Error == "" {
				totalCount += result.Count
			} else {
				errorCount++
			}
		case err := <-errors:
			errorCount++
			logrus.Warnf("Discovery error: %v", err)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	bulkResult := &BulkDiscoveryResult{
		Results:      discoveryResults,
		TotalCount:   totalCount,
		ErrorCount:   errorCount,
		DiscoveredAt: time.Now(),
	}

	logrus.Infof("Concurrent discovery completed: %d domains, %d total subdomains, %d errors",
		len(discoveryResults), totalCount, errorCount)

	return bulkResult, nil
}

// IsHealthy checks if the ChaosDB API is healthy
func (c *Client) IsHealthy(ctx context.Context) error {
	c.rateLimiter.Wait()

	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetQueryParam("domain", "example.com").
		Get(baseURL)

	if err != nil {
		return fmt.Errorf("failed to check ChaosDB API health: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("ChaosDB API returned status %d", resp.StatusCode())
	}

	return nil
}

// GetRateLimit returns the current rate limit
func (c *Client) GetRateLimit() int {
	return c.rateLimiter.GetRate()
}

// UpdateRateLimit updates the rate limit
func (c *Client) UpdateRateLimit(newRate int) {
	c.rateLimiter.UpdateRate(newRate)
}
