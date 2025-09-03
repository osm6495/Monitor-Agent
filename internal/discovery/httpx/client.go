package httpx

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/projectdiscovery/httpx/runner"
	"github.com/sirupsen/logrus"
)

// ProbeResult represents the result of a domain probe
type ProbeResult struct {
	URL        string `json:"url"`
	StatusCode int    `json:"status_code"`
	Exists     bool   `json:"exists"`
	Error      string `json:"error,omitempty"`
}

// DetailedProbeResult represents a detailed probe result with full response information
type DetailedProbeResult struct {
	URL          string            `json:"url"`
	StatusCode   int               `json:"status_code"`
	Exists       bool              `json:"exists"`
	Error        string            `json:"error,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	Body         string            `json:"body,omitempty"`
	ResponseTime int64             `json:"response_time,omitempty"` // in milliseconds
	ContentType  string            `json:"content_type,omitempty"`
	Server       string            `json:"server,omitempty"`
	Title        string            `json:"title,omitempty"`
	Technologies []string          `json:"technologies,omitempty"`
}

// ProbeConfig holds configuration for the HTTPX probe
type ProbeConfig struct {
	Timeout         time.Duration // Per-URL timeout
	Concurrency     int
	RateLimit       int
	UserAgent       string
	FollowRedirects bool
	MaxRedirects    int
	Debug           bool
}

// Client represents an HTTPX probe client
type Client struct {
	config *ProbeConfig
}

// NewClient creates a new HTTPX probe client
func NewClient(config *ProbeConfig) *Client {
	if config == nil {
		config = &ProbeConfig{
			Timeout:         15 * time.Second,
			Concurrency:     25,
			RateLimit:       100,
			UserAgent:       "Monitor-Agent/1.0",
			FollowRedirects: true,
			MaxRedirects:    3,
			Debug:           false,
		}
	}

	return &Client{
		config: config,
	}
}

// ProbeDomains probes a list of domains to check if they exist
func (c *Client) ProbeDomains(ctx context.Context, domains []string) ([]ProbeResult, error) {
	if len(domains) == 0 {
		return []ProbeResult{}, nil
	}

	logrus.Infof("Starting HTTPX probe for %d domains", len(domains))

	// Convert domains to URLs
	urls := c.convertDomainsToURLs(domains)
	if len(urls) == 0 {
		return []ProbeResult{}, nil
	}

	// Collect results as they arrive
	var results []ProbeResult
	var mu sync.Mutex // Protect concurrent access to results slice

	logrus.Infof("Creating HTTPX runner with %d URLs", len(urls))

	// Create HTTPX runner options
	options := &runner.Options{
		InputTargetHost: urls,
		RateLimit:       c.config.RateLimit,
		Threads:         c.config.Concurrency,
		Timeout:         int(c.config.Timeout.Seconds()),
		FollowRedirects: c.config.FollowRedirects,
		MaxRedirects:    c.config.MaxRedirects,
		Silent:          true, // Suppress HTTPX output for cleaner operation
		NoColor:         true,
		JSONOutput:      false,
		CSVOutput:       false,
		Verbose:         c.config.Debug, // Use config debug setting
		Debug:           c.config.Debug, // Use config debug setting
		OnResult: func(result runner.Result) {
			// Process result immediately as it arrives
			probeResult := ProbeResult{
				URL:    result.URL,
				Exists: result.StatusCode > 0, // Any status code means the domain exists
			}
			if result.StatusCode > 0 {
				probeResult.StatusCode = result.StatusCode
			} else {
				// Log detailed error information
				if result.Error != "" {
					probeResult.Error = result.Error
					logrus.Debugf("HTTPX error for %s: %s", result.URL, result.Error)
				} else {
					probeResult.Error = "Domain does not exist or is unreachable"
				}
			}

			// Thread-safe append to results
			mu.Lock()
			results = append(results, probeResult)
			currentCount := len(results)
			mu.Unlock()

			// Log progress in real-time (use debug level for individual results)
			if c.config.Debug {
				logrus.Debugf("Result %d/%d: %s (status: %d, exists: %v)",
					currentCount, len(urls), result.URL, result.StatusCode, probeResult.Exists)
			}

			// Log progress summary every 10 results
			if currentCount%10 == 0 {
				logrus.Infof("HTTPX probe progress: %d/%d results collected", currentCount, len(urls))
			}
		},
	}

	// Run HTTPX synchronously - it handles concurrency internally
	httpxRunner, err := runner.New(options)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTPX runner: %w", err)
	}
	defer httpxRunner.Close()

	// Run the enumeration - HTTPX handles all concurrency internally
	logrus.Infof("Starting HTTPX enumeration for %d URLs", len(urls))

	// Add comprehensive debugging
	logrus.Infof("HTTPX configuration: Timeout=%ds, Threads=%d, RateLimit=%d",
		int(c.config.Timeout.Seconds()), c.config.Concurrency, c.config.RateLimit)

	// Run HTTPX enumeration
	httpxRunner.RunEnumeration()

	// Log detailed results analysis
	logrus.Infof("HTTPX enumeration completed. Results collected: %d/%d", len(results), len(urls))

	// Check if we're missing any URLs
	processedURLs := make(map[string]bool)
	for _, result := range results {
		processedURLs[result.URL] = true
	}

	missingURLs := []string{}
	for _, url := range urls {
		if !processedURLs[url] {
			missingURLs = append(missingURLs, url)
		}
	}

	if len(missingURLs) > 0 {
		logrus.Warnf("Missing results for %d URLs: %v", len(missingURLs), missingURLs)

		// Try to process missing URLs with a second HTTPX run
		logrus.Infof("Attempting to process %d missing URLs with second HTTPX run", len(missingURLs))

		// Create a second HTTPX runner for missing URLs
		missingOptions := &runner.Options{
			InputTargetHost: missingURLs,
			RateLimit:       c.config.RateLimit,
			Threads:         c.config.Concurrency,
			Timeout:         int(c.config.Timeout.Seconds()),
			FollowRedirects: c.config.FollowRedirects,
			MaxRedirects:    c.config.MaxRedirects,
			Silent:          true,
			NoColor:         true,
			JSONOutput:      false,
			CSVOutput:       false,
			Verbose:         c.config.Debug,
			Debug:           c.config.Debug,
			OnResult: func(result runner.Result) {
				// Process result immediately as it arrives
				probeResult := ProbeResult{
					URL:    result.URL,
					Exists: result.StatusCode > 0,
				}
				if result.StatusCode > 0 {
					probeResult.StatusCode = result.StatusCode
				} else {
					// Log detailed error information
					if result.Error != "" {
						probeResult.Error = result.Error
						logrus.Debugf("HTTPX error for %s: %s", result.URL, result.Error)
					} else {
						probeResult.Error = "Domain does not exist or is unreachable"
					}
				}

				// Thread-safe append to results
				mu.Lock()
				results = append(results, probeResult)
				currentCount := len(results)
				mu.Unlock()

				logrus.Debugf("Second run result %d/%d: %s (status: %d, exists: %v)",
					currentCount, len(urls), result.URL, result.StatusCode, probeResult.Exists)
			},
		}

		missingRunner, err := runner.New(missingOptions)
		if err != nil {
			logrus.Errorf("Failed to create second HTTPX runner: %v", err)
		} else {
			defer missingRunner.Close()
			logrus.Infof("Starting second HTTPX run for %d missing URLs", len(missingURLs))
			missingRunner.RunEnumeration()
			logrus.Infof("Second HTTPX run completed. Total results: %d/%d", len(results), len(urls))
		}
	} else {
		logrus.Infof("All URLs processed successfully")
	}

	// Log final summary
	existingCount := 0
	for _, result := range results {
		if result.Exists {
			existingCount++
		}
	}

	logrus.Infof("HTTPX probe completed: %d/%d domains exist (collected %d results)",
		existingCount, len(domains), len(results))

	return results, nil
}

// ProbeDomainsWithDetails probes a list of domains and returns detailed response information
func (c *Client) ProbeDomainsWithDetails(ctx context.Context, domains []string) ([]DetailedProbeResult, error) {
	if len(domains) == 0 {
		return []DetailedProbeResult{}, nil
	}

	logrus.Infof("Starting detailed HTTPX probe for %d domains", len(domains))

	// Convert domains to URLs
	urls := c.convertDomainsToURLs(domains)
	if len(urls) == 0 {
		return []DetailedProbeResult{}, nil
	}

	// Collect results as they arrive
	var results []DetailedProbeResult
	var mu sync.Mutex // Protect concurrent access to results slice

	logrus.Infof("Creating HTTPX runner with %d URLs for detailed probing", len(urls))

	// Create HTTPX runner options
	options := &runner.Options{
		InputTargetHost: urls,
		RateLimit:       c.config.RateLimit,
		Threads:         c.config.Concurrency,
		Timeout:         int(c.config.Timeout.Seconds()),
		FollowRedirects: c.config.FollowRedirects,
		MaxRedirects:    c.config.MaxRedirects,
		Silent:          true,
		NoColor:         true,
		JSONOutput:      false,
		CSVOutput:       false,
		Verbose:         c.config.Debug,
		Debug:           c.config.Debug,
		OnResult: func(result runner.Result) {
			// Process result immediately as it arrives
			detailedResult := DetailedProbeResult{
				URL:        result.URL,
				Exists:     result.StatusCode > 0,
				StatusCode: result.StatusCode,
			}

			if result.StatusCode > 0 {
				// Use reflection to inspect available fields in the result
				if c.config.Debug {
					resultValue := reflect.ValueOf(result)
					resultType := reflect.TypeOf(result)
					logrus.Debugf("HTTPX result fields for %s:", result.URL)
					for i := 0; i < resultValue.NumField(); i++ {
						field := resultType.Field(i)
						value := resultValue.Field(i)
						if value.IsValid() && !value.IsZero() {
							logrus.Debugf("  %s: %v", field.Name, value.Interface())
						}
					}
				}

				// Try to extract additional information from the result
				// Note: These fields may not exist in all versions of HTTPX
				detailedResult.ResponseTime = 0 // Will be populated if available

				// Basic information logging
				if c.config.Debug {
					logrus.Debugf("HTTPX result for %s: StatusCode=%d, Error=%s",
						result.URL, result.StatusCode, result.Error)
				}
			} else {
				// Log detailed error information
				if result.Error != "" {
					detailedResult.Error = result.Error
					logrus.Debugf("HTTPX error for %s: %s", result.URL, result.Error)
				} else {
					detailedResult.Error = "Domain does not exist or is unreachable"
				}
			}

			// Thread-safe append to results
			mu.Lock()
			results = append(results, detailedResult)
			currentCount := len(results)
			mu.Unlock()

			// Log progress in real-time (use debug level for individual results)
			if c.config.Debug {
				logrus.Debugf("Detailed result %d/%d: %s (status: %d, exists: %v)",
					currentCount, len(urls), result.URL, result.StatusCode, detailedResult.Exists)
			}

			// Log progress summary every 10 results
			if currentCount%10 == 0 {
				logrus.Infof("Detailed HTTPX probe progress: %d/%d results collected", currentCount, len(urls))
			}
		},
	}

	// Run HTTPX synchronously - it handles concurrency internally
	httpxRunner, err := runner.New(options)
	if err != nil {
		return nil, fmt.Errorf("failed to create detailed HTTPX runner: %w", err)
	}
	defer httpxRunner.Close()

	// Run the enumeration - HTTPX handles all concurrency internally
	logrus.Infof("Starting detailed HTTPX enumeration for %d URLs", len(urls))

	// Add comprehensive debugging
	logrus.Infof("Detailed HTTPX configuration: Timeout=%ds, Threads=%d, RateLimit=%d",
		int(c.config.Timeout.Seconds()), c.config.Concurrency, c.config.RateLimit)

	// Run HTTPX enumeration
	httpxRunner.RunEnumeration()

	// Log detailed results analysis
	logrus.Infof("Detailed HTTPX enumeration completed. Results collected: %d/%d", len(results), len(urls))

	// Log final summary
	existingCount := 0
	for _, result := range results {
		if result.Exists {
			existingCount++
		}
	}

	logrus.Infof("Detailed HTTPX probe completed: %d/%d domains exist (collected %d results)",
		existingCount, len(domains), len(results))

	return results, nil
}

// ToJSON converts a DetailedProbeResult to JSON string
func (r *DetailedProbeResult) ToJSON() (string, error) {
	jsonBytes, err := json.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("failed to marshal DetailedProbeResult to JSON: %w", err)
	}
	return string(jsonBytes), nil
}

// convertDomainsToURLs converts a list of domains to URLs with proper protocol handling
func (c *Client) convertDomainsToURLs(domains []string) []string {
	var urls []string
	for _, domain := range domains {
		// Clean domain and add protocol
		cleanDomain := strings.TrimSpace(domain)
		if cleanDomain == "" {
			continue
		}

		// Skip domains that are just protocol without hostname
		if cleanDomain == "http://" || cleanDomain == "https://" {
			continue
		}

		// Add protocol if not present
		if !strings.HasPrefix(cleanDomain, "http://") && !strings.HasPrefix(cleanDomain, "https://") {
			urls = append(urls, fmt.Sprintf("https://%s", cleanDomain))
		} else {
			urls = append(urls, cleanDomain)
		}
	}
	return urls
}

// FilterExistingDomains filters a list of domains to only include those that exist
func (c *Client) FilterExistingDomains(ctx context.Context, domains []string) ([]string, error) {
	probeResults, err := c.ProbeDomains(ctx, domains)
	if err != nil {
		return nil, err
	}

	var existingDomains []string
	for _, result := range probeResults {
		if result.Exists {
			// Extract domain from URL
			domain := c.ExtractDomainFromURL(result.URL)
			if domain != "" {
				existingDomains = append(existingDomains, domain)
			}
		}
	}

	return existingDomains, nil
}

// ExtractDomainFromURL extracts the domain from a URL
func (c *Client) ExtractDomainFromURL(urlStr string) string {
	// Handle empty or invalid URLs
	if strings.TrimSpace(urlStr) == "" {
		return ""
	}

	// Remove protocol
	urlStr = strings.TrimPrefix(urlStr, "https://")
	urlStr = strings.TrimPrefix(urlStr, "http://")

	// Handle URLs that are just protocol without hostname
	if strings.TrimSpace(urlStr) == "" {
		return ""
	}

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
