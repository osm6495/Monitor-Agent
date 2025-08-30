package platforms

import (
	"context"
	"time"
)

// Platform represents a bug bounty platform
type Platform interface {
	// GetName returns the platform name
	GetName() string

	// GetPublicPrograms retrieves all public bug bounty programs from the platform
	GetPublicPrograms(ctx context.Context) ([]*Program, error)

	// GetProgramScope retrieves the in-scope assets for a specific program
	GetProgramScope(ctx context.Context, programURL string) ([]*ScopeAsset, error)

	// IsHealthy checks if the platform API is healthy
	IsHealthy(ctx context.Context) error
}

// Program represents a bug bounty program
type Program struct {
	Name        string    `json:"name"`
	Platform    string    `json:"platform"`
	URL         string    `json:"url"`
	ProgramURL  string    `json:"program_url"`
	IsActive    bool      `json:"is_active"`
	LastUpdated time.Time `json:"last_updated"`
}

// ScopeAsset represents an in-scope asset for a bug bounty program
type ScopeAsset struct {
	URL       string `json:"url"`
	Domain    string `json:"domain"`
	Subdomain string `json:"subdomain,omitempty"`
	Type      string `json:"type"` // url, wildcard, etc.
}

// PlatformConfig holds configuration for a platform
type PlatformConfig struct {
	APIKey        string
	RateLimit     int
	Timeout       time.Duration
	RetryAttempts int
	RetryDelay    time.Duration
}
