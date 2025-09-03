package httpx

import (
	"context"
	"fmt"
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

// ProbeConfig holds configuration for the HTTPX probe
type ProbeConfig struct {
	Timeout         time.Duration // Per-URL timeout (not total process timeout)
	TotalTimeout    time.Duration // Total operation timeout for the entire probe
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
			TotalTimeout:    30 * time.Minute, // Default to 30 minutes for total operation
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

	if len(urls) == 0 {
		return []ProbeResult{}, nil
	}

	// Create results channel with buffer
	results := make(chan ProbeResult, len(urls))

	// Create a done channel to signal when HTTPX is finished
	done := make(chan struct{})

	// Use a mutex to protect channel operations
	var mu sync.Mutex
	channelClosed := false

	// Create HTTPX runner options
	options := &runner.Options{
		InputTargetHost: urls,
		RateLimit:       c.config.RateLimit,
		Threads:         c.config.Concurrency,
		Timeout:         int(c.config.Timeout.Seconds()),
		FollowRedirects: c.config.FollowRedirects,
		MaxRedirects:    c.config.MaxRedirects,
		Silent:          true, // Suppress HTTPX output
		NoColor:         true,
		JSONOutput:      false,
		CSVOutput:       false,
		Verbose:         c.config.Debug, // Use debug config
		Debug:           c.config.Debug,
		OnResult: func(result runner.Result) {
			probeResult := ProbeResult{
				URL:    result.URL,
				Exists: result.StatusCode > 0, // Any status code means the domain exists
			}

			if result.StatusCode > 0 {
				probeResult.StatusCode = result.StatusCode
			} else {
				probeResult.Error = "Domain does not exist or is unreachable"
			}

			if c.config.Debug {
				logrus.Debugf("HTTPX result: %s -> status: %d, exists: %v", result.URL, result.StatusCode, probeResult.Exists)
			}

			// Safely send result to channel
			mu.Lock()
			if !channelClosed {
				select {
				case results <- probeResult:
					if c.config.Debug {
						logrus.Debugf("Result sent to channel: %s", result.URL)
					}
				case <-ctx.Done():
					if c.config.Debug {
						logrus.Debugf("Context cancelled while sending result: %s", result.URL)
					}
				}
			}
			mu.Unlock()
		},
	}

	// Start HTTPX runner
	httpxRunner, err := runner.New(options)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTPX runner: %w", err)
	}

	// Run HTTPX in a goroutine
	go func() {
		defer func() {
			httpxRunner.Close()
			mu.Lock()
			if !channelClosed {
				close(done)
			}
			mu.Unlock()
		}()
		httpxRunner.RunEnumeration()
	}()

	// Collect results with improved logic
	var probeResults []ProbeResult
	expectedResults := len(urls)
	resultsCollected := 0
	httpxFinished := false

	// Use configured total timeout if available, otherwise fall back to context deadline or calculated timeout
	var totalTimeout time.Duration
	if c.config.TotalTimeout > 0 {
		// Use the configured total timeout
		totalTimeout = c.config.TotalTimeout
		logrus.Debugf("Using configured total timeout: %v", totalTimeout)
	} else if deadline, ok := ctx.Deadline(); ok {
		// Use the context deadline with a small buffer
		totalTimeout = time.Until(deadline) - 5*time.Second
		if totalTimeout <= 0 {
			totalTimeout = 30 * time.Second // Fallback if deadline is too close
		}
		logrus.Debugf("Using context deadline: %v (timeout: %v)", deadline, totalTimeout)
	} else {
		// No deadline set, use a reasonable timeout based on domain count
		totalTimeout = time.Duration(len(urls)) * c.config.Timeout / 2
		if totalTimeout < 30*time.Second {
			totalTimeout = 30 * time.Second
		}
		// Cap at 30 minutes for very large lists to prevent excessive resource usage
		if totalTimeout > 30*time.Minute {
			totalTimeout = 30 * time.Minute
		}
		logrus.Debugf("No context deadline, using calculated timeout: %v", totalTimeout)
	}

	logrus.Debugf("HTTPX probe timeout set to %v for %d domains", totalTimeout, len(urls))

	// Create a timeout context for the entire operation
	timeoutCtx, cancel := context.WithTimeout(ctx, totalTimeout)
	defer cancel()

	// First phase: collect results while HTTPX is running
	for resultsCollected < expectedResults && !httpxFinished {
		select {
		case result := <-results:
			probeResults = append(probeResults, result)
			resultsCollected++
			if c.config.Debug {
				logrus.Debugf("Collected result %d/%d: %s", resultsCollected, expectedResults, result.URL)
			}
		case <-done:
			// HTTPX finished, but continue collecting any remaining results
			httpxFinished = true
			logrus.Debugf("HTTPX runner finished, collected %d/%d results, continuing to collect remaining results", resultsCollected, expectedResults)
			// Don't break here - continue collecting any results that might still be in the channel
		case <-timeoutCtx.Done():
			// Timeout reached, break out of the loop
			logrus.Warnf("HTTPX probe timeout after %v, collected %d/%d results", totalTimeout, resultsCollected, expectedResults)
			goto collectionComplete
		case <-ctx.Done():
			// Parent context cancelled
			return nil, ctx.Err()
		}
	}

	// Second phase: if HTTPX finished but we haven't collected all results, try to collect remaining results
	// This handles the case where HTTPX processes URLs faster than we can collect results
	if httpxFinished && resultsCollected < expectedResults {
		logrus.Debugf("HTTPX finished early, attempting to collect remaining %d results", expectedResults-resultsCollected)

		// Try to collect remaining results with a short timeout
		remainingTimeout := time.After(10 * time.Second)
		for resultsCollected < expectedResults {
			select {
			case result := <-results:
				probeResults = append(probeResults, result)
				resultsCollected++
				if c.config.Debug {
					logrus.Debugf("Collected remaining result %d/%d: %s", resultsCollected, expectedResults, result.URL)
				}
			case <-remainingTimeout:
				logrus.Warnf("Timeout collecting remaining results, collected %d/%d total", resultsCollected, expectedResults)
				goto collectionComplete
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

collectionComplete:
	// Safely close the results channel
	mu.Lock()
	if !channelClosed {
		channelClosed = true
		close(results)
	}
	mu.Unlock()

	// Wait for HTTPX to finish (with a short timeout)
	select {
	case <-done:
		logrus.Debug("HTTPX runner finished gracefully")
	case <-time.After(5 * time.Second):
		logrus.Warn("HTTPX runner did not finish within 5 seconds")
	}

	// Log summary
	existingCount := 0
	for _, result := range probeResults {
		if result.Exists {
			existingCount++
		}
	}

	logrus.Infof("HTTPX probe completed: %d/%d domains exist (collected %d results)", existingCount, len(domains), len(probeResults))

	return probeResults, nil
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
