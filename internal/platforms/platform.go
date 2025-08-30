package platforms

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/monitor-agent/internal/database"
	"github.com/monitor-agent/internal/platforms/bugcrowd"
	"github.com/monitor-agent/internal/platforms/hackerone"
)

var (
	// ErrPlatformNotSupported is returned when a platform is not supported
	ErrPlatformNotSupported = errors.New("platform not supported")
)

// HackerOneAdapter adapts hackerone.Client to the main Platform interface
type HackerOneAdapter struct {
	client *hackerone.Client
}

func (a *HackerOneAdapter) GetName() string {
	return a.client.GetName()
}

func (a *HackerOneAdapter) GetPublicPrograms(ctx context.Context) ([]*Program, error) {
	h1Programs, err := a.client.GetPublicPrograms(ctx)
	if err != nil {
		return nil, err
	}

	programs := make([]*Program, len(h1Programs))
	for i, h1Program := range h1Programs {
		programs[i] = &Program{
			Name:        h1Program.Name,
			Platform:    h1Program.Platform,
			URL:         h1Program.URL,
			ProgramURL:  h1Program.ProgramURL,
			IsActive:    h1Program.IsActive,
			LastUpdated: h1Program.LastUpdated,
		}
	}
	return programs, nil
}

func (a *HackerOneAdapter) GetProgramScope(ctx context.Context, programURL string) ([]*ScopeAsset, error) {
	h1Assets, err := a.client.GetProgramScope(ctx, programURL)
	if err != nil {
		return nil, err
	}

	assets := make([]*ScopeAsset, len(h1Assets))
	for i, h1Asset := range h1Assets {
		assets[i] = &ScopeAsset{
			URL:       h1Asset.URL,
			Domain:    h1Asset.Domain,
			Subdomain: h1Asset.Subdomain,
			Type:      h1Asset.Type,
		}
	}
	return assets, nil
}

func (a *HackerOneAdapter) IsHealthy(ctx context.Context) error {
	return a.client.IsHealthy(ctx)
}

// BugCrowdAdapter adapts bugcrowd.Client to the main Platform interface
type BugCrowdAdapter struct {
	client *bugcrowd.Client
}

func (a *BugCrowdAdapter) GetName() string {
	return a.client.GetName()
}

func (a *BugCrowdAdapter) GetPublicPrograms(ctx context.Context) ([]*Program, error) {
	bcPrograms, err := a.client.GetPublicPrograms(ctx)
	if err != nil {
		return nil, err
	}

	programs := make([]*Program, len(bcPrograms))
	for i, bcProgram := range bcPrograms {
		programs[i] = &Program{
			Name:        bcProgram.Name,
			Platform:    bcProgram.Platform,
			URL:         bcProgram.URL,
			ProgramURL:  bcProgram.ProgramURL,
			IsActive:    bcProgram.IsActive,
			LastUpdated: bcProgram.LastUpdated,
		}
	}
	return programs, nil
}

func (a *BugCrowdAdapter) GetProgramScope(ctx context.Context, programURL string) ([]*ScopeAsset, error) {
	bcAssets, err := a.client.GetProgramScope(ctx, programURL)
	if err != nil {
		return nil, err
	}

	assets := make([]*ScopeAsset, len(bcAssets))
	for i, bcAsset := range bcAssets {
		assets[i] = &ScopeAsset{
			URL:       bcAsset.URL,
			Domain:    bcAsset.Domain,
			Subdomain: bcAsset.Subdomain,
			Type:      bcAsset.Type,
		}
	}
	return assets, nil
}

func (a *BugCrowdAdapter) IsHealthy(ctx context.Context) error {
	return a.client.IsHealthy(ctx)
}

// PlatformFactory creates platform instances
type PlatformFactory struct {
	configs map[string]*PlatformConfig
}

// NewPlatformFactory creates a new platform factory
func NewPlatformFactory() *PlatformFactory {
	return &PlatformFactory{
		configs: make(map[string]*PlatformConfig),
	}
}

// RegisterPlatform registers a platform configuration
func (f *PlatformFactory) RegisterPlatform(name string, config *PlatformConfig) {
	f.configs[name] = config
}

// GetPlatform returns a platform instance by name
func (f *PlatformFactory) GetPlatform(name string) (Platform, error) {
	config, exists := f.configs[name]
	if !exists {
		return nil, ErrPlatformNotSupported
	}

	switch name {
	case "hackerone":
		// Convert config to hackerone.PlatformConfig
		h1Config := &hackerone.PlatformConfig{
			APIKey:        config.APIKey,
			RateLimit:     config.RateLimit,
			Timeout:       config.Timeout,
			RetryAttempts: config.RetryAttempts,
			RetryDelay:    config.RetryDelay,
		}
		return &HackerOneAdapter{client: hackerone.NewHackerOneClient(h1Config)}, nil
	case "bugcrowd":
		// Convert config to bugcrowd.PlatformConfig
		bcConfig := &bugcrowd.PlatformConfig{
			APIKey:        config.APIKey,
			RateLimit:     config.RateLimit,
			Timeout:       config.Timeout,
			RetryAttempts: config.RetryAttempts,
			RetryDelay:    config.RetryDelay,
		}
		return &BugCrowdAdapter{client: bugcrowd.NewBugCrowdClient(bcConfig)}, nil
	default:
		return nil, ErrPlatformNotSupported
	}
}

// GetAllPlatforms returns all registered platforms
func (f *PlatformFactory) GetAllPlatforms() []Platform {
	var platforms []Platform
	for name := range f.configs {
		if platform, err := f.GetPlatform(name); err == nil {
			platforms = append(platforms, platform)
		}
	}
	return platforms
}

// ConvertToDatabaseProgram converts a platform Program to a database Program
func (p *Program) ConvertToDatabaseProgram() *database.Program {
	return &database.Program{
		Name:        p.Name,
		Platform:    p.Platform,
		URL:         p.URL,
		ProgramURL:  p.ProgramURL,
		IsActive:    p.IsActive,
		LastUpdated: p.LastUpdated,
	}
}

// ConvertToDatabaseAsset converts a ScopeAsset to a database Asset
func (sa *ScopeAsset) ConvertToDatabaseAsset(programID string) *database.Asset {
	// Parse the programID string to UUID
	programUUID, err := uuid.Parse(programID)
	if err != nil {
		// If parsing fails, generate a new UUID
		programUUID = uuid.New()
	}

	return &database.Asset{
		ProgramID: programUUID,
		URL:       sa.URL,
		Domain:    sa.Domain,
		Subdomain: sa.Subdomain,
		Status:    "active",
		Source:    "direct",
	}
}
