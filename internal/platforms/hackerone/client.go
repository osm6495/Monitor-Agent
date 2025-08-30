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
	httpClient  *resty.Client
	config      *PlatformConfig
	rateLimiter *utils.RateLimiter
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
	if config.APIKey != "" {
		client.SetBasicAuth("", config.APIKey)
	}

	return &Client{
		httpClient:  client,
		config:      config,
		rateLimiter: utils.NewRateLimiter(config.RateLimit, time.Minute),
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
		Get(fmt.Sprintf("%s/programs", baseURL))

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
	params.Set("filter[state]", "public")
	params.Set("filter[offers_bounties]", "true")

	resp, err := c.httpClient.R().
		SetContext(ctx).
		Get(fmt.Sprintf("%s/programs?%s", baseURL, params.Encode()))

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
		// Only include programs that are public and offer bounties
		if program.Attributes.State == "public" && program.Attributes.OffersBounties {
			platformProgram := &Program{
				Name:        program.Attributes.Name,
				Platform:    "hackerone",
				URL:         program.Attributes.Website,
				ProgramURL:  program.Links.Web,
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

	c.rateLimiter.Wait()

	params := url.Values{}
	params.Set("page[size]", "100")

	resp, err := c.httpClient.R().
		SetContext(ctx).
		Get(fmt.Sprintf("%s/programs/%s/structured_scopes?%s", baseURL, handle, params.Encode()))

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
		// Only include in-scope assets that are eligible for submission
		if scope.Attributes.EligibleForSubmission {
			asset := c.parseScopeAsset(scope.Attributes)
			if asset != nil {
				scopeAssets = append(scopeAssets, asset)
			}
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

	// Handle different asset types
	switch attr.AssetType {
	case "URL":
		return &ScopeAsset{
			URL:    assetIdentifier,
			Domain: c.extractDomain(assetIdentifier),
			Type:   "url",
		}
	case "WILDCARD":
		// Convert wildcard to base domain for ChaosDB discovery
		domain := c.convertWildcardToDomain(assetIdentifier)
		return &ScopeAsset{
			URL:    domain,
			Domain: domain,
			Type:   "wildcard",
		}
	case "CIDR":
		// Handle CIDR ranges (for future implementation)
		return &ScopeAsset{
			URL:    assetIdentifier,
			Domain: assetIdentifier,
			Type:   "cidr",
		}
	default:
		return &ScopeAsset{
			URL:    assetIdentifier,
			Domain: assetIdentifier,
			Type:   attr.AssetType,
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
	// Simple domain extraction - in production, you might want to use a proper URL parser
	if strings.HasPrefix(urlStr, "http://") || strings.HasPrefix(urlStr, "https://") {
		// Remove protocol
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

// convertWildcardToDomain converts a wildcard domain to its base domain
func (c *Client) convertWildcardToDomain(wildcard string) string {
	// Remove the wildcard prefix and return the base domain
	return strings.TrimPrefix(wildcard, "*.")
}
