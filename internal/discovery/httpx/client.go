package httpx

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/monitor-agent/internal/database"
	"github.com/monitor-agent/internal/utils"
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
	config       *ProbeConfig
	urlProcessor *utils.URLProcessor
}

// NewClient creates a new HTTPX probe client
func NewClient(config *ProbeConfig) *Client {
	if config == nil {
		config = &ProbeConfig{
			Timeout:         30 * time.Second,
			Concurrency:     25,
			RateLimit:       50,
			UserAgent:       "Monitor-Agent/1.0",
			FollowRedirects: true,
			MaxRedirects:    3,
			Debug:           false,
		}
	}

	return &Client{
		config:       config,
		urlProcessor: utils.NewURLProcessor(),
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

	// Create HTTPX runner options with more conservative settings for reliability
	options := &runner.Options{
		Probe:           true,
		OmitBody:        true,
		InputTargetHost: urls,
		RateLimit:       c.config.RateLimit,
		Threads:         c.config.Concurrency,
		Timeout:         5,
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

	// Create a channel to signal when enumeration is complete
	enumerationComplete := make(chan struct{})

	// Start HTTPX enumeration in a goroutine
	go func() {
		defer close(enumerationComplete)
		httpxRunner.RunEnumeration()
	}()

	// Wait for enumeration to complete or context to be cancelled
	select {
	case <-enumerationComplete:
		logrus.Infof("HTTPX enumeration completed")
	case <-ctx.Done():
		logrus.Warnf("HTTPX enumeration cancelled due to context timeout: %v", ctx.Err())
		// Give a small grace period for any pending results
		time.Sleep(2 * time.Second)
	}

	// Log detailed results analysis
	logrus.Infof("HTTPX enumeration completed. Results collected: %d/%d", len(results), len(urls))

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

	// Create HTTPX runner options with more conservative settings for reliability
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
		// Add retry configuration for better reliability
		Retries: 2,
		OnResult: func(result runner.Result) {
			// Process result immediately as it arrives
			detailedResult := DetailedProbeResult{
				URL:        result.URL,
				Exists:     result.StatusCode > 0,
				StatusCode: result.StatusCode,
			}

			if result.StatusCode > 0 {
				// Extract headers from the result
				if result.ResponseHeaders != nil {
					detailedResult.Headers = make(map[string]string)
					for key, value := range result.ResponseHeaders {
						if strValue, ok := value.(string); ok {
							detailedResult.Headers[key] = strValue
						} else {
							// Convert non-string values to string
							detailedResult.Headers[key] = fmt.Sprintf("%v", value)
						}
					}
				}

				// Extract response body
				detailedResult.Body = result.ResponseBody

				// Extract response time (convert from string to milliseconds)
				if result.ResponseTime != "" {
					if responseTime, err := time.ParseDuration(result.ResponseTime); err == nil {
						detailedResult.ResponseTime = responseTime.Milliseconds()
					}
				}

				// Extract additional information
				detailedResult.ContentType = result.ContentType
				detailedResult.Server = result.WebServer
				detailedResult.Title = result.Title
				detailedResult.Technologies = result.Technologies

				// Log basic result information
				if c.config.Debug {
					logrus.Debugf("HTTPX result for %s: StatusCode=%d, Headers=%d, BodySize=%d, ResponseTime=%dms",
						result.URL, result.StatusCode, len(detailedResult.Headers), len(detailedResult.Body), detailedResult.ResponseTime)
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

	// Create a channel to signal when enumeration is complete
	enumerationComplete := make(chan struct{})

	// Start HTTPX enumeration in a goroutine
	go func() {
		defer close(enumerationComplete)
		httpxRunner.RunEnumeration()
	}()

	// Wait for enumeration to complete or context to be cancelled
	select {
	case <-enumerationComplete:
		logrus.Infof("HTTPX enumeration completed")
	case <-ctx.Done():
		logrus.Warnf("HTTPX enumeration cancelled due to context timeout: %v", ctx.Err())
		// Give a small grace period for any pending results
		time.Sleep(2 * time.Second)
	}

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
	var invalidDomains []string

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

		// Extract domain name for validation (remove protocol if present)
		domainName := cleanDomain
		if strings.HasPrefix(cleanDomain, "http://") {
			domainName = strings.TrimPrefix(cleanDomain, "http://")
		} else if strings.HasPrefix(cleanDomain, "https://") {
			domainName = strings.TrimPrefix(cleanDomain, "https://")
		}

		// Remove path and query parameters for domain validation
		if idx := strings.Index(domainName, "/"); idx != -1 {
			domainName = domainName[:idx]
		}
		if idx := strings.Index(domainName, "?"); idx != -1 {
			domainName = domainName[:idx]
		}
		if idx := strings.Index(domainName, ":"); idx != -1 {
			domainName = domainName[:idx]
		}

		// Skip wildcard domains
		if strings.Contains(domainName, "*") {
			invalidDomains = append(invalidDomains, domainName)
			continue
		}

		// Skip domains that are too short or contain invalid characters
		if len(domainName) < 3 || strings.ContainsAny(domainName, " \t\n\r") {
			invalidDomains = append(invalidDomains, domainName)
			continue
		}

		// Validate domain before adding to URLs
		if !c.urlProcessor.IsValidDomain(domainName) {
			invalidDomains = append(invalidDomains, domainName)
			continue
		}

		// Add protocol if not present
		if !strings.HasPrefix(cleanDomain, "http://") && !strings.HasPrefix(cleanDomain, "https://") {
			urls = append(urls, fmt.Sprintf("https://%s", cleanDomain))
		} else {
			urls = append(urls, cleanDomain)
		}
	}

	// Log invalid domains for debugging
	if len(invalidDomains) > 0 {
		examples := invalidDomains
		if len(examples) > 5 {
			examples = examples[:5]
		}
		logrus.Debugf("HTTPX filtered out %d invalid domains, examples: %v", len(invalidDomains), examples)
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

// FormatProbeResults formats probe results for output with SUCCESS/FAILED status
func (c *Client) FormatProbeResults(results []ProbeResult, showFailures bool) []string {
	var formatted []string

	for _, result := range results {
		if result.Exists {
			// Only show successful results without [SUCCESS] suffix
			formatted = append(formatted, result.URL)
		} else if showFailures {
			// Show failures with [FAILED] suffix if requested
			formatted = append(formatted, fmt.Sprintf("%s [FAILED]", result.URL))
		}
	}

	return formatted
}

// IsValidURLForDatabase checks if a URL is valid for saving to database
func (c *Client) IsValidURLForDatabase(url string) bool {
	// Extract domain from URL
	domain := c.ExtractDomainFromURL(url)
	if domain == "" {
		return false
	}

	// Check if domain contains wildcards
	if strings.Contains(domain, "*") {
		return false
	}

	// Check if domain is valid using URL processor
	return c.urlProcessor.IsValidDomain(domain)
}

// SaveSuccessfulProbeResultsAsAssets saves successful probe results as assets to the database
func (c *Client) SaveSuccessfulProbeResultsAsAssets(ctx context.Context, results []ProbeResult, programID uuid.UUID, programURL string, assetRepo *database.AssetRepository) error {
	var assets []*database.Asset

	for _, result := range results {
		// Only process successful results
		if !result.Exists {
			continue
		}

		// Validate URL before saving
		if !c.IsValidURLForDatabase(result.URL) {
			logrus.Debugf("Skipping invalid URL for database: %s", result.URL)
			continue
		}

		// Extract domain and subdomain
		domain := c.ExtractDomainFromURL(result.URL)
		if domain == "" {
			logrus.Debugf("Failed to extract domain from URL: %s", result.URL)
			continue
		}

		// Extract subdomain
		subdomain := ""
		if domain != result.URL {
			// Remove protocol and path to get just the hostname
			hostname := strings.TrimPrefix(result.URL, "https://")
			hostname = strings.TrimPrefix(hostname, "http://")
			if idx := strings.Index(hostname, "/"); idx != -1 {
				hostname = hostname[:idx]
			}
			if idx := strings.Index(hostname, ":"); idx != -1 {
				hostname = hostname[:idx]
			}

			// If hostname is different from domain, it's a subdomain
			if hostname != domain {
				subdomain = hostname
			}
		}

		asset := &database.Asset{
			ProgramID:  programID,
			ProgramURL: programURL,
			URL:        result.URL,
			Domain:     domain,
			Subdomain:  subdomain,
			Status:     "active",
			Source:     "httpx_probe", // Mark as httpx probe result
		}

		assets = append(assets, asset)
	}

	// Save assets to database if any were created
	if len(assets) > 0 {
		if err := assetRepo.CreateAssets(ctx, assets); err != nil {
			return fmt.Errorf("failed to save httpx probe assets: %w", err)
		}
		logrus.Infof("Successfully saved %d httpx probe assets to database", len(assets))
	} else {
		logrus.Infof("No valid httpx probe assets to save")
	}

	return nil
}
