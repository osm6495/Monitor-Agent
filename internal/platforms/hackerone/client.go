package hackerone

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
	baseURL = "https://api.hackerone.com/v1"
)

// Client represents a HackerOne API client
type Client struct {
	httpClient   *resty.Client
	config       *PlatformConfig
	rateLimiter  *utils.RateLimiter
	urlProcessor *utils.URLProcessor // Added URLProcessor field
}

// NewHackerOneClient creates a new HackerOne client
func NewHackerOneClient(config *PlatformConfig) *Client {
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

	// Add authentication
	if config.APIKey != "" && config.Username != "" {
		// Use the API key as-is (it appears to already be in the correct format)
		client.SetBasicAuth(config.Username, config.APIKey)
	} else if config.APIKey != "" {
		// Fallback for backward compatibility - try with empty username
		client.SetBasicAuth("", config.APIKey)
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
	return "hackerone"
}

// IsHealthy checks if the HackerOne API is healthy
func (c *Client) IsHealthy(ctx context.Context) error {
	c.rateLimiter.Wait()

	resp, err := c.httpClient.R().
		SetContext(ctx).
		Get(fmt.Sprintf("%s/hackers/programs", baseURL))

	if err != nil {
		return fmt.Errorf("failed to check HackerOne API health: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("HackerOne API returned status %d", resp.StatusCode())
	}

	return nil
}

// GetPublicPrograms retrieves all public bug bounty programs from HackerOne
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

	logrus.Infof("Retrieved %d programs from HackerOne", len(allPrograms))
	return allPrograms, nil
}

// getProgramsPage retrieves a single page of programs
func (c *Client) getProgramsPage(ctx context.Context, page, pageSize int) ([]*Program, bool, error) {
	params := url.Values{}
	params.Set("page[number]", fmt.Sprintf("%d", page))
	params.Set("page[size]", fmt.Sprintf("%d", pageSize))

	resp, err := c.httpClient.R().
		SetContext(ctx).
		Get(fmt.Sprintf("%s/hackers/programs?%s", baseURL, params.Encode()))

	if err != nil {
		return nil, false, fmt.Errorf("failed to make request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		var errorResp ErrorResponse
		if err := json.Unmarshal(resp.Body(), &errorResp); err == nil {
			return nil, false, fmt.Errorf("HackerOne API error: %s", errorResp.Errors[0].Detail)
		}
		return nil, false, fmt.Errorf("HackerOne API returned status %d", resp.StatusCode())
	}

	var apiResp HackerOneResponse
	if err := json.Unmarshal(resp.Body(), &apiResp); err != nil {
		return nil, false, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	var programs []*Program
	for _, program := range apiResp.Data {
		// Include programs that are in public mode and offer bounties
		if program.Attributes.State == "public_mode" && program.Attributes.OffersBounties {
			// Construct the program URL using the handle
			programURL := fmt.Sprintf("https://hackerone.com/%s", program.Attributes.Handle)

			platformProgram := &Program{
				Name:        program.Attributes.Name,
				Platform:    "hackerone",
				URL:         program.Attributes.Website,
				ProgramURL:  programURL,
				IsActive:    true,
				LastUpdated: program.Attributes.UpdatedAt,
			}
			programs = append(programs, platformProgram)
		}
	}

	// Check if there are more pages
	hasMore := apiResp.Links.Next != ""

	return programs, hasMore, nil
}

// GetProgramScope retrieves the in-scope assets for a specific program
func (c *Client) GetProgramScope(ctx context.Context, programURL string) ([]*ScopeAsset, error) {
	// Extract program handle from URL
	handle, err := c.extractHandleFromURL(programURL)
	if err != nil {
		return nil, fmt.Errorf("failed to extract handle from URL: %w", err)
	}

	logrus.Debugf("Extracted handle '%s' from URL '%s'", handle, programURL)

	c.rateLimiter.Wait()

	params := url.Values{}
	params.Set("page[size]", "100")

	scopeURL := fmt.Sprintf("%s/hackers/programs/%s/structured_scopes?%s", baseURL, handle, params.Encode())
	logrus.Debugf("Making scope request to: %s", scopeURL)

	resp, err := c.httpClient.R().
		SetContext(ctx).
		Get(scopeURL)

	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		var errorResp ErrorResponse
		if err := json.Unmarshal(resp.Body(), &errorResp); err == nil {
			return nil, fmt.Errorf("HackerOne API error: %s", errorResp.Errors[0].Detail)
		}
		return nil, fmt.Errorf("HackerOne API returned status %d", resp.StatusCode())
	}

	var scopeResp ScopeResponse
	if err := json.Unmarshal(resp.Body(), &scopeResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	var scopeAssets []*ScopeAsset
	for _, scope := range scopeResp.Data {
		// Include both in-scope and out-of-scope assets
		asset := c.parseScopeAsset(scope.Attributes)
		if asset != nil {
			scopeAssets = append(scopeAssets, asset)
		}
	}

	logrus.Infof("Retrieved %d scope assets for program %s", len(scopeAssets), handle)
	return scopeAssets, nil
}

// parseScopeAsset parses a scope attribute into a ScopeAsset
func (c *Client) parseScopeAsset(attr ScopeAttributes) *ScopeAsset {
	assetIdentifier := strings.TrimSpace(attr.AssetIdentifier)
	if assetIdentifier == "" {
		return nil
	}

	// Normalize URL to ensure it has https:// protocol
	normalizedURL, err := c.urlProcessor.NormalizeURL(assetIdentifier)
	if err != nil {
		// If normalization fails, fallback to simple https:// addition
		if !strings.HasPrefix(assetIdentifier, "http://") && !strings.HasPrefix(assetIdentifier, "https://") {
			normalizedURL = "https://" + assetIdentifier
		} else {
			normalizedURL = assetIdentifier
		}
	}

	// Handle different asset types
	switch attr.AssetType {
	case "URL":
		return &ScopeAsset{
			URL:                   normalizedURL,
			Domain:                c.extractDomain(normalizedURL),
			Type:                  "url",
			EligibleForSubmission: attr.EligibleForSubmission,
		}
	case "WILDCARD":
		// Convert wildcard to base domain for ChaosDB discovery
		domain := c.urlProcessor.ConvertWildcardToDomain(assetIdentifier)
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
			URL:                   normalizedDomain,
			Domain:                domain,
			Type:                  "wildcard",
			EligibleForSubmission: attr.EligibleForSubmission,
			OriginalPattern:       assetIdentifier, // Store original wildcard pattern
		}
	case "CIDR":
		// Handle CIDR ranges (for future implementation)
		return &ScopeAsset{
			URL:                   assetIdentifier,
			Domain:                assetIdentifier,
			Type:                  "cidr",
			EligibleForSubmission: attr.EligibleForSubmission,
		}
	default:
		return &ScopeAsset{
			URL:                   normalizedURL,
			Domain:                c.extractDomain(normalizedURL),
			Type:                  attr.AssetType,
			EligibleForSubmission: attr.EligibleForSubmission,
		}
	}
}

// extractHandleFromURL extracts the program handle from a HackerOne program URL
func (c *Client) extractHandleFromURL(programURL string) (string, error) {
	// Expected format: https://hackerone.com/program-name
	parts := strings.Split(programURL, "/")
	if len(parts) < 4 {
		return "", fmt.Errorf("invalid HackerOne program URL: %s", programURL)
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
