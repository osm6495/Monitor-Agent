package service

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/monitor-agent/internal/config"
	"github.com/monitor-agent/internal/database"
	"github.com/monitor-agent/internal/discovery/chaosdb"
	"github.com/monitor-agent/internal/discovery/httpx"
	"github.com/monitor-agent/internal/platforms"
	"github.com/monitor-agent/internal/utils"
	"github.com/sirupsen/logrus"
)

// MonitorService orchestrates the monitoring of bug bounty programs
type MonitorService struct {
	config          *config.Config
	programRepo     *database.ProgramRepository
	assetRepo       *database.AssetRepository
	scanRepo        *database.ScanRepository
	platformFactory *platforms.PlatformFactory
	chaosDBClient   *chaosdb.Client
	httpxClient     *httpx.Client
	urlProcessor    *utils.URLProcessor
}

// NewMonitorService creates a new monitor service
func NewMonitorService(cfg *config.Config, db *sqlx.DB) *MonitorService {
	// Initialize repositories
	programRepo := database.NewProgramRepository(db)
	assetRepo := database.NewAssetRepository(db)
	scanRepo := database.NewScanRepository(db)

	// Initialize platform factory
	platformFactory := platforms.NewPlatformFactory()

	// Only register platforms that have API keys configured
	if cfg.HasHackerOneConfig() {
		platformFactory.RegisterPlatform("hackerone", &platforms.PlatformConfig{
			APIKey:        cfg.APIs.HackerOne.APIKey,
			Username:      cfg.APIs.HackerOne.Username,
			RateLimit:     cfg.APIs.HackerOne.RateLimit,
			Timeout:       cfg.HTTP.Timeout,
			RetryAttempts: cfg.HTTP.RetryAttempts,
			RetryDelay:    cfg.HTTP.RetryDelay,
		})
		logrus.Info("HackerOne platform configured")
	} else {
		logrus.Warn("HackerOne API key not provided, skipping HackerOne platform")
	}

	if cfg.HasBugCrowdConfig() {
		platformFactory.RegisterPlatform("bugcrowd", &platforms.PlatformConfig{
			APIKey:        cfg.APIs.BugCrowd.APIKey,
			RateLimit:     cfg.APIs.BugCrowd.RateLimit,
			Timeout:       cfg.HTTP.Timeout,
			RetryAttempts: cfg.HTTP.RetryAttempts,
			RetryDelay:    cfg.HTTP.RetryDelay,
		})
		logrus.Info("BugCrowd platform configured")
	} else {
		logrus.Warn("BugCrowd API key not provided, skipping BugCrowd platform")
	}

	// Initialize ChaosDB client (only if API key is provided)
	var chaosDBClient *chaosdb.Client
	if cfg.HasChaosDBConfig() {
		chaosDBClient = chaosdb.NewClient(&chaosdb.ClientConfig{
			APIKey:        cfg.APIs.ChaosDB.APIKey,
			RateLimit:     cfg.APIs.ChaosDB.RateLimit,
			Timeout:       cfg.HTTP.Timeout,
			RetryAttempts: cfg.HTTP.RetryAttempts,
			RetryDelay:    cfg.HTTP.RetryDelay,
		})
		logrus.Info("ChaosDB client configured")
	} else {
		logrus.Warn("ChaosDB API key not provided, ChaosDB discovery will be disabled")
	}

	// Initialize HTTPX client (only if enabled)
	var httpxClient *httpx.Client
	if cfg.Discovery.HTTPX.Enabled {
		httpxClient = httpx.NewClient(&httpx.ProbeConfig{
			Timeout:         cfg.Discovery.HTTPX.Timeout,
			Concurrency:     cfg.Discovery.HTTPX.Concurrency,
			RateLimit:       cfg.Discovery.HTTPX.RateLimit,
			FollowRedirects: cfg.Discovery.HTTPX.FollowRedirects,
			MaxRedirects:    cfg.Discovery.HTTPX.MaxRedirects,
			Debug:           cfg.Discovery.HTTPX.Debug,
		})
		logrus.Info("HTTPX probe client configured")
	} else {
		logrus.Info("HTTPX probe disabled")
	}

	return &MonitorService{
		config:          cfg,
		programRepo:     programRepo,
		assetRepo:       assetRepo,
		scanRepo:        scanRepo,
		platformFactory: platformFactory,
		chaosDBClient:   chaosDBClient,
		httpxClient:     httpxClient,
		urlProcessor:    utils.NewURLProcessor(),
	}
}

// GetConfig returns the service configuration
func (s *MonitorService) GetConfig() *config.Config {
	return s.config
}

// RunFullScan performs a complete scan of all platforms
func (s *MonitorService) RunFullScan(ctx context.Context) error {
	logrus.Info("Starting full scan of all bug bounty platforms")

	// Get all platforms
	platformList := s.platformFactory.GetAllPlatforms()
	if len(platformList) == 0 {
		logrus.Warn("No platforms configured with API keys. Please provide at least one API key (HACKERONE_USERNAME+HACKERONE_API_KEY, BUGCROWD_API_KEY, or CHAOSDB_API_KEY) to perform scans.")
		return fmt.Errorf("no platforms configured with API keys")
	}

	logrus.Infof("Starting scan of %d platforms", len(platformList))

	var wg sync.WaitGroup
	errors := make(chan error, len(platformList))

	// Scan each platform concurrently
	for i, platform := range platformList {
		wg.Add(1)
		go func(p platforms.Platform, platformIndex int) {
			defer wg.Done()
			logrus.Infof("Starting scan of platform %d/%d: %s", platformIndex+1, len(platformList), p.GetName())

			startTime := time.Now()
			if err := s.scanPlatform(ctx, p); err != nil {
				logrus.Errorf("Platform %s scan failed after %v: %v", p.GetName(), time.Since(startTime), err)
				errors <- fmt.Errorf("failed to scan platform %s: %w", p.GetName(), err)
			} else {
				logrus.Infof("Platform %s scan completed successfully in %v", p.GetName(), time.Since(startTime))
			}
		}(platform, i)
	}

	logrus.Info("Waiting for all platform scans to complete...")
	wg.Wait()
	close(errors)

	// Collect errors
	var errs []error
	for err := range errors {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("scan completed with %d errors: %v", len(errs), errs)
	}

	logrus.Info("Full scan completed successfully")
	return nil
}

// scanPlatform scans a single platform
func (s *MonitorService) scanPlatform(ctx context.Context, platform platforms.Platform) error {
	platformName := platform.GetName()
	logrus.Infof("Scanning platform: %s", platformName)

	// Check platform health
	if err := platform.IsHealthy(ctx); err != nil {
		return fmt.Errorf("platform %s is not healthy: %w", platformName, err)
	}

	// Get public programs from platform
	programs, err := platform.GetPublicPrograms(ctx)
	if err != nil {
		return fmt.Errorf("failed to get programs from %s: %w", platformName, err)
	}

	logrus.Infof("Found %d programs on platform %s", len(programs), platformName)

	// Process each program
	for _, program := range programs {
		if err := s.processProgram(ctx, platform, program); err != nil {
			logrus.Errorf("Failed to process program %s: %v", program.Name, err)
			continue
		}
	}

	// Mark inactive programs
	if err := s.markInactivePrograms(ctx, platformName, programs); err != nil {
		return fmt.Errorf("failed to mark inactive programs for %s: %w", platformName, err)
	}

	return nil
}

// processProgram processes a single program
func (s *MonitorService) processProgram(ctx context.Context, platform platforms.Platform, program *platforms.Program) error {
	logrus.Infof("Processing program: %s (%s)", program.Name, program.Platform)

	// Check if program already exists in database using ProgramURL as the unique identifier
	existingProgram, err := s.programRepo.GetProgramByPlatformAndProgramURL(ctx, program.Platform, program.ProgramURL)
	if err != nil {
		return fmt.Errorf("failed to check existing program: %w", err)
	}

	if existingProgram != nil {
		// Update existing program
		existingProgram.Name = program.Name
		existingProgram.ProgramURL = program.ProgramURL
		existingProgram.IsActive = program.IsActive
		existingProgram.LastUpdated = program.LastUpdated

		if err := s.programRepo.UpdateProgram(ctx, existingProgram); err != nil {
			return fmt.Errorf("failed to update program: %w", err)
		}

		logrus.Infof("Updated existing program: %s", program.Name)

		// Check if there are new primary assets before running discovery
		hasNewAssets, err := s.hasNewPrimaryAssets(ctx, existingProgram, platform)
		if err != nil {
			logrus.Warnf("Failed to check for new primary assets for program %s: %v", program.Name, err)
			// Continue with discovery as fallback
		} else if !hasNewAssets {
			logrus.Infof("No new primary assets found for program %s, skipping asset discovery", program.Name)
			return nil
		}

		// Refresh assets for existing programs only if new primary assets are found
		if err := s.discoverProgramAssets(ctx, existingProgram, platform); err != nil {
			return fmt.Errorf("failed to refresh assets for existing program %s: %w", program.Name, err)
		}

		return nil
	}

	// Create new program
	dbProgram := program.ConvertToDatabaseProgram()
	if err := s.programRepo.CreateProgram(ctx, dbProgram); err != nil {
		return fmt.Errorf("failed to create program: %w", err)
	}

	logrus.Infof("Created new program: %s", program.Name)

	// Get program scope and discover assets
	if err := s.discoverProgramAssets(ctx, dbProgram, platform); err != nil {
		return fmt.Errorf("failed to discover assets for program %s: %w", program.Name, err)
	}

	return nil
}

// hasNewPrimaryAssets checks if there are new primary assets compared to existing ones
func (s *MonitorService) hasNewPrimaryAssets(ctx context.Context, program *database.Program, platform platforms.Platform) (bool, error) {
	// Get current primary assets from database
	existingPrimaryAssets, err := s.assetRepo.GetAssetsByProgramIDAndSource(ctx, program.ID, "primary")
	if err != nil {
		return false, fmt.Errorf("failed to get existing primary assets: %w", err)
	}

	// Get new scope assets from platform
	scopeAssets, err := platform.GetProgramScope(ctx, program.ProgramURL)
	if err != nil {
		return false, fmt.Errorf("failed to get program scope: %w", err)
	}

	// Create a map of existing primary asset URLs for quick lookup
	existingURLs := make(map[string]bool)
	for _, asset := range existingPrimaryAssets {
		existingURLs[asset.URL] = true
	}

	// Check if any new scope assets are not in existing primary assets
	for _, scopeAsset := range scopeAssets {
		if !existingURLs[scopeAsset.URL] {
			return true, nil // Found a new primary asset
		}
	}

	return false, nil // No new primary assets found
}

// discoverProgramAssets discovers assets for a program
func (s *MonitorService) discoverProgramAssets(ctx context.Context, program *database.Program, platform platforms.Platform) error {
	logrus.Infof("Discovering assets for program: %s", program.Name)

	// Create scan record
	scan := &database.Scan{
		ProgramID:   program.ID,
		Status:      "running",
		AssetsFound: 0,
	}

	if err := s.scanRepo.CreateScan(ctx, scan); err != nil {
		return fmt.Errorf("failed to create scan record: %w", err)
	}

	defer func() {
		// Update scan status
		scan.Status = "completed"
		scan.CompletedAt = &time.Time{}
		if err := s.scanRepo.UpdateScan(ctx, scan); err != nil {
			logrus.Errorf("Failed to update scan status: %v", err)
		}
	}()

	// Get program scope from platform
	scopeAssets, err := platform.GetProgramScope(ctx, program.ProgramURL)
	if err != nil {
		scan.Status = "failed"
		scan.Error = err.Error()
		return fmt.Errorf("failed to get program scope: %w", err)
	}

	logrus.Infof("Found %d scope assets for program %s", len(scopeAssets), program.Name)

	// Log the first few scope assets for debugging
	if len(scopeAssets) > 0 {
		sampleSize := 3
		if len(scopeAssets) < sampleSize {
			sampleSize = len(scopeAssets)
		}
		logrus.Debugf("Sample scope assets for program %s: %v", program.Name, scopeAssets[:sampleSize])
	}

	// Save in-scope assets as primary assets (only domain and wildcard types)
	var primaryAssets []*database.Asset
	for _, scopeAsset := range scopeAssets {
		// Only save domain and wildcard type assets as primary assets
		if scopeAsset.Type == "url" || scopeAsset.Type == "wildcard" {
			dbAsset := scopeAsset.ConvertToDatabaseAsset(program.ID.String(), program.ProgramURL)
			dbAsset.Source = "primary" // Mark as primary asset
			primaryAssets = append(primaryAssets, dbAsset)
		} else {
			logrus.Debugf("Skipping non-domain asset type '%s' for program %s: %s", scopeAsset.Type, program.Name, scopeAsset.URL)
		}
	}

	// Save primary assets to database
	if len(primaryAssets) > 0 {
		if err := s.assetRepo.CreateAssets(ctx, primaryAssets); err != nil {
			scan.Status = "failed"
			scan.Error = err.Error()
			return fmt.Errorf("failed to save primary assets: %w", err)
		}
		logrus.Infof("Saved %d primary assets for program %s (filtered from %d total scope assets)", len(primaryAssets), program.Name, len(scopeAssets))
	} else {
		logrus.Infof("No primary assets to save for program %s (filtered from %d total scope assets)", program.Name, len(scopeAssets))
	}

	// Extract unique domains for ChaosDB discovery
	domains := s.extractUniqueDomains(scopeAssets)
	logrus.Infof("Extracted %d unique domains for ChaosDB discovery: %v", len(domains), domains)

	// Discover additional subdomains using ChaosDB (secondary assets)
	if len(domains) > 0 {
		secondaryAssets, err := s.discoverWithChaosDB(ctx, program.ID, program.ProgramURL, domains)
		if err != nil {
			logrus.Warnf("ChaosDB discovery failed for program %s: %v", program.Name, err)
		} else {
			logrus.Infof("ChaosDB discovered %d secondary assets for program %s", len(secondaryAssets), program.Name)
		}
	}

	// Update scan with final count
	assetCount, err := s.assetRepo.GetAssetCountByProgramID(ctx, program.ID)
	if err != nil {
		logrus.Warnf("Failed to get asset count for program %s: %v", program.Name, err)
	} else {
		scan.AssetsFound = assetCount
	}

	return nil
}

// discoverWithChaosDB discovers additional subdomains using ChaosDB and filters them with HTTPX probe
func (s *MonitorService) discoverWithChaosDB(ctx context.Context, programID uuid.UUID, programURL string, domains []string) ([]*database.Asset, error) {
	if s.chaosDBClient == nil {
		logrus.Warn("ChaosDB client not configured, skipping discovery")
		return nil, nil
	}

	// Create a timeout context for ChaosDB discovery and HTTPX probing
	// This prevents the discovery process from hanging indefinitely
	discoveryTimeout := s.config.Discovery.Timeouts.ChaosDiscovery
	discoveryCtx, cancel := context.WithTimeout(ctx, discoveryTimeout)
	defer cancel()

	logrus.Infof("Starting ChaosDB discovery for %d domains: %v", len(domains), domains)

	// Use bulk discovery for efficiency
	bulkResult, err := s.chaosDBClient.DiscoverDomainsBulk(discoveryCtx, domains)
	if err != nil {
		return nil, fmt.Errorf("ChaosDB bulk discovery failed: %w", err)
	}

	// Collect all subdomains from ChaosDB results
	var allSubdomains []string
	for _, result := range bulkResult.Results {
		if result.Error != "" {
			logrus.Warnf("ChaosDB error for domain %s: %s", result.Domain, result.Error)
			continue
		}

		allSubdomains = append(allSubdomains, result.Subdomains...)
	}

	logrus.Infof("ChaosDB discovered %d total subdomains", len(allSubdomains))

	// Filter subdomains using HTTPX probe if enabled
	var filteredSubdomains []string
	if s.httpxClient != nil && len(allSubdomains) > 0 {
		logrus.Infof("Starting HTTPX probe to filter %d subdomains", len(allSubdomains))
		logrus.Debugf("HTTPX probe timeout set to %v", discoveryTimeout)

		// Start HTTPX probe with progress logging
		probeStart := time.Now()
		filteredSubdomains, err = s.httpxClient.FilterExistingDomains(discoveryCtx, allSubdomains)
		probeDuration := time.Since(probeStart)

		if err != nil {
			logrus.Warnf("HTTPX probe failed after %v, using all subdomains: %v", probeDuration, err)
			filteredSubdomains = allSubdomains
		} else {
			logrus.Infof("HTTPX probe completed in %v: %d/%d subdomains exist", probeDuration, len(filteredSubdomains), len(allSubdomains))
		}
	} else {
		logrus.Info("HTTPX probe not configured or no subdomains to probe, using all subdomains")
		filteredSubdomains = allSubdomains
	}

	// Convert filtered subdomains to assets
	var assets []*database.Asset
	for _, subdomain := range filteredSubdomains {
		// Create full URL
		url := fmt.Sprintf("https://%s", subdomain)

		// Extract domain and subdomain
		domain, err := s.urlProcessor.ExtractDomain(url)
		if err != nil {
			logrus.Warnf("Failed to extract domain from %s: %v", url, err)
			continue
		}

		subdomainName, err := s.urlProcessor.ExtractSubdomain(url)
		if err != nil {
			logrus.Warnf("Failed to extract subdomain from %s: %v", url, err)
		}

		asset := &database.Asset{
			ProgramID:  programID,
			ProgramURL: programURL,
			URL:        url,
			Domain:     domain,
			Subdomain:  subdomainName,
			Status:     "active",
			Source:     "secondary", // Mark as secondary asset from ChaosDB
		}

		assets = append(assets, asset)
	}

	// Save filtered ChaosDB assets to database
	if len(assets) > 0 {
		if err := s.assetRepo.CreateAssets(ctx, assets); err != nil {
			return nil, fmt.Errorf("failed to save ChaosDB assets: %w", err)
		}
	}

	return assets, nil
}

// extractUniqueDomains extracts unique domains from scope assets (only domain and wildcard types)
func (s *MonitorService) extractUniqueDomains(scopeAssets []*platforms.ScopeAsset) []string {
	domainMap := make(map[string]bool)
	var domains []string

	for _, asset := range scopeAssets {
		// Only process domain and wildcard type assets for ChaosDB discovery
		if asset.Type != "url" && asset.Type != "wildcard" {
			continue
		}

		domain, err := s.urlProcessor.ExtractDomain(asset.URL)
		if err != nil {
			logrus.Warnf("Failed to extract domain from %s: %v", asset.URL, err)
			continue
		}

		if !domainMap[domain] {
			domainMap[domain] = true
			domains = append(domains, domain)
		}
	}

	return domains
}

// markInactivePrograms marks programs as inactive if they no longer exist
func (s *MonitorService) markInactivePrograms(ctx context.Context, platformName string, currentPrograms []*platforms.Program) error {
	// Get all active programs for this platform from database
	dbPrograms, err := s.programRepo.GetProgramsByPlatform(ctx, platformName)
	if err != nil {
		return fmt.Errorf("failed to get database programs: %w", err)
	}

	// Create a map of current program URLs
	currentProgramURLs := make(map[string]bool)
	for _, program := range currentPrograms {
		currentProgramURLs[program.ProgramURL] = true
	}

	// Mark programs as inactive if they're not in the current list
	for _, dbProgram := range dbPrograms {
		if !currentProgramURLs[dbProgram.ProgramURL] {
			if err := s.programRepo.MarkProgramInactive(ctx, dbProgram.ID); err != nil {
				logrus.Errorf("Failed to mark program %s as inactive: %v", dbProgram.Name, err)
				continue
			}
			logrus.Infof("Marked program %s as inactive", dbProgram.Name)
		}
	}

	return nil
}

// GetProgramStats returns statistics about programs and assets
func (s *MonitorService) GetProgramStats(ctx context.Context) (*ProgramStats, error) {
	// Get programs with asset counts
	programsWithCounts, err := s.programRepo.GetProgramsWithAssetCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get programs with asset counts: %w", err)
	}

	// Get recent scans
	recentScans, err := s.scanRepo.GetRecentScans(ctx, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent scans: %w", err)
	}

	stats := &ProgramStats{
		TotalPrograms:  len(programsWithCounts),
		ActivePrograms: 0,
		TotalAssets:    0,
		RecentScans:    recentScans,
	}

	for _, programWithCount := range programsWithCounts {
		if programWithCount.Program.IsActive {
			stats.ActivePrograms++
		}
		stats.TotalAssets += programWithCount.AssetCount
	}

	return stats, nil
}

// CheckDatabaseHealth checks database connectivity and health
func (s *MonitorService) CheckDatabaseHealth(ctx context.Context) error {
	// Test basic connectivity
	if err := s.programRepo.GetDB().PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Test a simple query
	if _, err := s.programRepo.GetProgramsWithAssetCount(ctx); err != nil {
		return fmt.Errorf("database query test failed: %w", err)
	}

	logrus.Debug("Database health check passed")
	return nil
}

// CheckPlatformHealth checks health of all configured platforms
func (s *MonitorService) CheckPlatformHealth(ctx context.Context) error {
	platforms := s.platformFactory.GetAllPlatforms()
	if len(platforms) == 0 {
		logrus.Warn("No platforms configured with API keys, skipping platform health checks")
		return nil
	}

	for _, platform := range platforms {
		platformName := platform.GetName()
		if err := platform.IsHealthy(ctx); err != nil {
			return fmt.Errorf("platform %s health check failed: %w", platformName, err)
		}
		logrus.Debugf("Platform %s health check passed", platformName)
	}

	return nil
}

// CheckChaosDBHealth checks ChaosDB service health
func (s *MonitorService) CheckChaosDBHealth(ctx context.Context) error {
	if s.chaosDBClient == nil {
		logrus.Warn("ChaosDB client not configured, skipping ChaosDB health check")
		return nil
	}

	if err := s.chaosDBClient.IsHealthy(ctx); err != nil {
		return fmt.Errorf("ChaosDB health check failed: %w", err)
	}

	logrus.Debug("ChaosDB health check passed")
	return nil
}

// CheckSystemHealth checks system resources and limits
func (s *MonitorService) CheckSystemHealth(ctx context.Context) error {
	// Check memory usage (basic implementation)
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Log memory usage for monitoring
	logrus.Debugf("System memory - Alloc: %d MB, Sys: %d MB, NumGC: %d",
		m.Alloc/1024/1024, m.Sys/1024/1024, m.NumGC)

	// Check if memory usage is reasonable (less than 1GB allocated)
	if m.Alloc > 1024*1024*1024 {
		return fmt.Errorf("high memory usage: %d MB allocated", m.Alloc/1024/1024)
	}

	// Check goroutine count
	numGoroutines := runtime.NumGoroutine()
	if numGoroutines > 1000 {
		return fmt.Errorf("high goroutine count: %d", numGoroutines)
	}

	logrus.Debugf("System health check passed - Goroutines: %d", numGoroutines)
	return nil
}

// ProgramStats represents program statistics
type ProgramStats struct {
	TotalPrograms  int              `json:"total_programs"`
	ActivePrograms int              `json:"active_programs"`
	TotalAssets    int              `json:"total_assets"`
	RecentScans    []*database.Scan `json:"recent_scans"`
}
