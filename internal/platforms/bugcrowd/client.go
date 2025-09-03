package bugcrowd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/monitor-agent/internal/utils"
	"github.com/sirupsen/logrus"
)

const (
	baseURL = "https://api.bugcrowd.com"
)

// Client represents a BugCrowd API client
type Client struct {
	httpClient   *resty.Client
	config       *PlatformConfig
	rateLimiter  *utils.RateLimiter
	urlProcessor *utils.URLProcessor // Added URLProcessor field
}

// NewBugCrowdClient creates a new BugCrowd client
func NewBugCrowdClient(config *PlatformConfig) *Client {
	client := resty.New()
	client.SetTimeout(config.Timeout)
	client.SetRetryCount(config.RetryAttempts)
	client.SetRetryWaitTime(config.RetryDelay)
	client.SetRetryMaxWaitTime(config.RetryDelay * 2)

	// Set default headers
	client.SetHeaders(map[string]string{
		"Accept":       "application/vnd.bugcrowd+json",
		"Content-Type": "application/json",
		"User-Agent":   "Monitor-Agent/1.0",
	})

	// Add authentication
	if config.APIKey != "" {
		client.SetHeader("Authorization", fmt.Sprintf("Token %s", config.APIKey))
	}

	return &Client{
		httpClient:   client,
		config:       config,
		rateLimiter:  utils.NewRateLimiter(config.RateLimit, time.Minute),
		urlProcessor: utils.NewURLProcessor(), // Initialize URLProcessor
	}
}

// GetName returns the platform name
func (c *Client) GetName() string {
	return "bugcrowd"
}

// IsHealthy checks if the BugCrowd API is healthy
func (c *Client) IsHealthy(ctx context.Context) error {
	c.rateLimiter.Wait()

	resp, err := c.httpClient.R().
		SetContext(ctx).
		Get(fmt.Sprintf("%s/programs", baseURL))

	if err != nil {
		return fmt.Errorf("failed to check BugCrowd API health: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("BugCrowd API returned status %d", resp.StatusCode())
	}

	return nil
}

// GetPublicPrograms retrieves all public bug bounty programs from BugCrowd
func (c *Client) GetPublicPrograms(ctx context.Context) ([]*Program, error) {
	var allPrograms []*Program
	page := 1
	pageSize := 100

	for {
		c.rateLimiter.Wait()

		programs, hasMore, err := c.getProgramsPage(ctx, page, pageSize)
		if err != nil {
			return nil, fmt.Errorf("failed to get programs page %d: %w", page, err)
		}

		allPrograms = append(allPrograms, programs...)

		if !hasMore {
			break
		}
		page++
	}

	logrus.Infof("Retrieved %d programs from BugCrowd", len(allPrograms))
	return allPrograms, nil
}

// getProgramsPage retrieves a single page of programs
func (c *Client) getProgramsPage(ctx context.Context, page, pageSize int) ([]*Program, bool, error) {
	params := url.Values{}
	params.Set("page", fmt.Sprintf("%d", page))
	params.Set("per_page", fmt.Sprintf("%d", pageSize))
	params.Set("status", "public")

	resp, err := c.httpClient.R().
		SetContext(ctx).
		Get(fmt.Sprintf("%s/programs?%s", baseURL, params.Encode()))

	if err != nil {
		return nil, false, fmt.Errorf("failed to make request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		var errorResp BugCrowdError
		if err := json.Unmarshal(resp.Body(), &errorResp); err == nil {
			return nil, false, fmt.Errorf("BugCrowd API error: %s", errorResp.Message)
		}
		return nil, false, fmt.Errorf("BugCrowd API returned status %d", resp.StatusCode())
	}

	var apiResp BugCrowdResponse
	if err := json.Unmarshal(resp.Body(), &apiResp); err != nil {
		return nil, false, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	var programs []*Program
	for _, program := range apiResp.Programs {
		// Only include public programs
		if program.Status == "public" {
			platformProgram := &Program{
				Name:        program.Name,
				Platform:    "bugcrowd",
				URL:         program.URL,
				ProgramURL:  fmt.Sprintf("https://bugcrowd.com/%s", program.Code),
				IsActive:    true,
				LastUpdated: program.UpdatedAt,
			}
			programs = append(programs, platformProgram)
		}
	}

	// Check if there are more pages
	hasMore := page < apiResp.Meta.PageCount

	return programs, hasMore, nil
}

// GetProgramScope retrieves the in-scope assets for a specific program
func (c *Client) GetProgramScope(ctx context.Context, programURL string) ([]*ScopeAsset, error) {
	// Extract program code from URL
	code, err := c.extractCodeFromURL(programURL)
	if err != nil {
		return nil, fmt.Errorf("failed to extract code from URL: %w", err)
	}

	c.rateLimiter.Wait()

	params := url.Values{}
	params.Set("per_page", "100")

	resp, err := c.httpClient.R().
		SetContext(ctx).
		Get(fmt.Sprintf("%s/programs/%s/targets?%s", baseURL, code, params.Encode()))

	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		var errorResp BugCrowdError
		if err := json.Unmarshal(resp.Body(), &errorResp); err == nil {
			return nil, fmt.Errorf("BugCrowd API error: %s", errorResp.Message)
		}
		return nil, fmt.Errorf("BugCrowd API returned status %d", resp.StatusCode())
	}

	var scopeResp ScopeResponse
	if err := json.Unmarshal(resp.Body(), &scopeResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	var scopeAssets []*ScopeAsset
	for _, target := range scopeResp.Targets {
		// Only include eligible targets
		if target.Eligible && !target.Ineligible {
			asset := c.parseScopeAsset(target)
			if asset != nil {
				scopeAssets = append(scopeAssets, asset)
			}
		}
	}

	logrus.Infof("Retrieved %d scope assets for program %s", len(scopeAssets), code)
	return scopeAssets, nil
}

// parseScopeAsset parses a scope target into a ScopeAsset
func (c *Client) parseScopeAsset(target BugCrowdScope) *ScopeAsset {
	targetStr := strings.TrimSpace(target.Target)
	if targetStr == "" {
		return nil
	}

	// Normalize URL to ensure it has https:// protocol
	normalizedURL, err := c.urlProcessor.NormalizeURL(targetStr)
	if err != nil {
		// If normalization fails, fallback to simple https:// addition
		if !strings.HasPrefix(targetStr, "http://") && !strings.HasPrefix(targetStr, "https://") {
			normalizedURL = "https://" + targetStr
		} else {
			normalizedURL = targetStr
		}
	}

	// Handle different target types
	switch target.Type {
	case "website":
		return &ScopeAsset{
			URL:    normalizedURL,
			Domain: c.extractDomain(normalizedURL),
			Type:   "url",
		}
	case "wildcard":
		// Convert wildcard to base domain for ChaosDB discovery
		domain := c.urlProcessor.ConvertWildcardToDomain(targetStr)
		normalizedDomain, err := c.urlProcessor.NormalizeURL(domain)
		if err != nil {
			// If normalization fails, fallback to simple https:// addition
			if !strings.HasPrefix(domain, "http://") && !strings.HasPrefix(domain, "https://") {
				normalizedDomain = "https://" + domain
			} else {
				normalizedDomain = domain
			}
		}
		return &ScopeAsset{
			URL:    normalizedDomain,
			Domain: domain,
			Type:   "wildcard",
		}
	case "ip":
		return &ScopeAsset{
			URL:    targetStr,
			Domain: targetStr,
			Type:   "ip",
		}
	default:
		return &ScopeAsset{
			URL:    normalizedURL,
			Domain: c.extractDomain(normalizedURL),
			Type:   target.Type,
		}
	}
}

// extractCodeFromURL extracts the program code from a BugCrowd program URL
func (c *Client) extractCodeFromURL(programURL string) (string, error) {
	// Expected format: https://bugcrowd.com/program-name
	parts := strings.Split(programURL, "/")
	if len(parts) < 4 {
		return "", fmt.Errorf("invalid BugCrowd program URL: %s", programURL)
	}
	return parts[len(parts)-1], nil
}

// extractDomain extracts the domain from a URL
func (c *Client) extractDomain(urlStr string) string {
	// Use URLProcessor for consistent domain extraction
	domain, err := c.urlProcessor.ExtractDomain(urlStr)
	if err != nil {
		// Fallback to simple extraction if URLProcessor fails
		if strings.HasPrefix(urlStr, "http://") || strings.HasPrefix(urlStr, "https://") {
			urlStr = strings.TrimPrefix(urlStr, "http://")
			urlStr = strings.TrimPrefix(urlStr, "https://")
		}

		// Remove path and query parameters
		if idx := strings.Index(urlStr, "/"); idx != -1 {
			urlStr = urlStr[:idx]
		}

		// Remove port if present
		if idx := strings.Index(urlStr, ":"); idx != -1 {
			urlStr = urlStr[:idx]
		}

		return urlStr
	}

	return domain
}
