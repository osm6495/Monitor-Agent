package service

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
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
			defer func() {
				if r := recover(); r != nil {
					logrus.Errorf("Platform %s scan panicked: %v", p.GetName(), r)
					errors <- fmt.Errorf("platform %s scan panicked: %v", p.GetName(), r)
				}
				wg.Done()
			}()

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

	// Process each program with individual timeouts
	for i, program := range programs {
		logrus.Infof("Processing program %d/%d: %s", i+1, len(programs), program.Name)

		// Create a timeout context for each program
		programCtx, cancel := context.WithTimeout(ctx, s.config.Discovery.Timeouts.ProgramProcess)

		// Process the program with timeout and panic recovery
		func() {
			defer func() {
				if r := recover(); r != nil {
					logrus.Errorf("Program %s processing panicked: %v", program.Name, r)
				}
			}()

			if err := s.processProgram(programCtx, platform, program); err != nil {
				logrus.Errorf("Failed to process program %s: %v", program.Name, err)
				// Continue to next program instead of failing the entire scan
			}
		}()

		cancel()

		// Small delay between programs to avoid overwhelming the system
		time.Sleep(100 * time.Millisecond)
	}

	// Mark inactive programs
	if err := s.markInactivePrograms(ctx, platformName, programs); err != nil {
		return fmt.Errorf("failed to mark inactive programs for %s: %w", platformName, err)
	}

	return nil
}

// processProgram processes a single program
func (s *MonitorService) processProgram(ctx context.Context, platform platforms.Platform, program *platforms.Program) error {
	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("processProgram panicked for program %s: %v", program.Name, r)
		}
	}()

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
	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("hasNewPrimaryAssets panicked for program %s: %v", program.Name, r)
		}
	}()

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

	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Program %s asset discovery panicked: %v", program.Name, r)
			scan.Status = "failed"
			scan.Error = fmt.Sprintf("Panic: %v", r)
			// Try to update scan status even if we panicked
			if err := s.scanRepo.UpdateScan(ctx, scan); err != nil {
				logrus.Errorf("Failed to update scan status after panic: %v", err)
			}
		}
	}()

	// Get program scope from platform with timeout protection
	scopeAssets, err := platform.GetProgramScope(ctx, program.ProgramURL)
	if err != nil {
		scan.Status = "failed"
		scan.Error = err.Error()
		logrus.Errorf("Failed to get program scope for %s: %v", program.Name, err)
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

	// Separate in-scope and out-of-scope assets
	var inScopeAssets []*platforms.ScopeAsset
	var outOfScopeAssets []*platforms.ScopeAsset
	var primaryAssets []*database.Asset

	for _, scopeAsset := range scopeAssets {
		if scopeAsset.EligibleForSubmission {
			inScopeAssets = append(inScopeAssets, scopeAsset)
			// Only save domain and wildcard type assets as primary assets
			if scopeAsset.Type == "url" || scopeAsset.Type == "wildcard" {
				dbAsset := scopeAsset.ConvertToDatabaseAsset(program.ID.String(), program.ProgramURL)
				dbAsset.Source = "primary" // Mark as primary asset
				primaryAssets = append(primaryAssets, dbAsset)
			} else {
				logrus.Debugf("Skipping non-domain asset type '%s' for program %s: %s", scopeAsset.Type, program.Name, scopeAsset.URL)
			}
		} else {
			// Only include URL and wildcard type assets for out-of-scope filtering
			if scopeAsset.Type == "url" || scopeAsset.Type == "wildcard" {
				outOfScopeAssets = append(outOfScopeAssets, scopeAsset)
			}
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

	logrus.Infof("Found %d in-scope assets and %d out-of-scope assets for program %s", len(inScopeAssets), len(outOfScopeAssets), program.Name)

	// Extract unique domains for ChaosDB discovery
	domains := s.extractUniqueDomains(scopeAssets)
	logrus.Infof("Extracted %d unique domains for ChaosDB discovery: %v", len(domains), domains)

	// Discover additional subdomains using ChaosDB (secondary assets)
	if len(domains) > 0 {
		secondaryAssets, err := s.discoverWithChaosDB(ctx, program.ID, program.ProgramURL, domains, outOfScopeAssets)
		if err != nil {
			logrus.Warnf("ChaosDB discovery failed for program %s: %v", program.Name, err)
			// Continue processing even if ChaosDB fails
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
func (s *MonitorService) discoverWithChaosDB(ctx context.Context, programID uuid.UUID, programURL string, domains []string, outOfScopeAssets []*platforms.ScopeAsset) ([]*database.Asset, error) {
	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("discoverWithChaosDB panicked: %v", r)
		}
	}()

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

	// Process domains sequentially to respect ChaosDB rate limits
	return s.processDomainsSequentially(discoveryCtx, programID, programURL, domains, outOfScopeAssets)
}

// processDomainsSequentially processes domains one by one to respect rate limits
func (s *MonitorService) processDomainsSequentially(ctx context.Context, programID uuid.UUID, programURL string, domains []string, outOfScopeAssets []*platforms.ScopeAsset) ([]*database.Asset, error) {
	var allAssets []*database.Asset
	totalSubdomains := 0
	successfulDomains := 0
	errorCount := 0

	for i, domain := range domains {
		logrus.Infof("Processing domain %d/%d: %s", i+1, len(domains), domain)

		// Process single domain with HTTPX probe
		domainAssets, err := s.processSingleDomain(ctx, programID, programURL, domain, i+1, len(domains), outOfScopeAssets)
		if err != nil {
			logrus.Warnf("Failed to process domain %s: %v", domain, err)
			errorCount++
			continue
		}

		allAssets = append(allAssets, domainAssets...)
		totalSubdomains += len(domainAssets)
		successfulDomains++

		// Small delay between domains to respect rate limits
		if i < len(domains)-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	logrus.Infof("ChaosDB discovery completed: %d domains, %d total subdomains, %d successful domains, %d errors",
		len(domains), totalSubdomains, successfulDomains, errorCount)

	return allAssets, nil
}

// processSingleDomain processes a single domain using ChaosDB and HTTPX probe
func (s *MonitorService) processSingleDomain(ctx context.Context, programID uuid.UUID, programURL string, domain string, domainIndex int, totalDomains int, outOfScopeAssets []*platforms.ScopeAsset) ([]*database.Asset, error) {
	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("processSingleDomain panicked for domain %s: %v", domain, r)
		}
	}()

	if s.chaosDBClient == nil {
		logrus.Warnf("ChaosDB client not configured, skipping domain %s", domain)
		return nil, nil
	}

	// Create a timeout context for ChaosDB discovery and HTTPX probing
	// This prevents the discovery process from hanging indefinitely
	discoveryTimeout := s.config.Discovery.Timeouts.ChaosDiscovery
	domainCtx, cancel := context.WithTimeout(ctx, discoveryTimeout)
	defer cancel()

	logrus.Infof("Starting ChaosDB discovery for domain %d/%d: %s", domainIndex, totalDomains, domain)

	// Use bulk discovery for efficiency
	bulkResult, err := s.chaosDBClient.DiscoverDomainsBulk(domainCtx, []string{domain})
	if err != nil {
		logrus.Warnf("ChaosDB bulk discovery failed for domain %s: %v", domain, err)
		return nil, nil // Return empty result instead of error to continue processing
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

	logrus.Infof("ChaosDB discovered %d total subdomains for domain %s", len(allSubdomains), domain)

	// Filter out wildcard subdomains and validate domains before HTTPX probing
	var cleanSubdomains []string
	var invalidSubdomains []string
	for _, subdomain := range allSubdomains {
		// Remove wildcard prefixes (e.g., *.test.slackhq.com -> test.slackhq.com)
		cleanSubdomain := s.urlProcessor.ConvertWildcardToDomain(subdomain)
		if cleanSubdomain != "" {
			// Validate the domain before adding it to the list
			if s.urlProcessor.IsValidDomain(cleanSubdomain) {
				cleanSubdomains = append(cleanSubdomains, cleanSubdomain)
			} else {
				invalidSubdomains = append(invalidSubdomains, cleanSubdomain)
			}
		}
	}

	logrus.Infof("Filtered %d wildcard subdomains, %d clean subdomains, %d invalid subdomains for domain %s",
		len(allSubdomains)-len(cleanSubdomains)-len(invalidSubdomains), len(cleanSubdomains), len(invalidSubdomains), domain)

	// Log some examples of invalid subdomains for debugging
	if len(invalidSubdomains) > 0 {
		examples := invalidSubdomains
		if len(examples) > 5 {
			examples = examples[:5]
		}
		logrus.Debugf("Examples of invalid subdomains filtered out: %v", examples)
	}

	// Filter subdomains using HTTPX probe if enabled and capture detailed responses
	var filteredSubdomains []string
	var detailedResults []httpx.DetailedProbeResult
	if s.httpxClient != nil && len(cleanSubdomains) > 0 {
		logrus.Infof("Starting detailed HTTPX probe to filter %d subdomains for domain %s", len(cleanSubdomains), domain)
		logrus.Debugf("HTTPX probe timeout set to %v", discoveryTimeout)

		// Start HTTPX probe with progress logging
		probeStart := time.Now()

		// Use a separate context for HTTPX probe with its own timeout
		httpxCtx, httpxCancel := context.WithTimeout(domainCtx, discoveryTimeout)

		// Log the timeout being used
		logrus.Infof("HTTPX probe timeout set to %v for domain %s", discoveryTimeout, domain)

		detailedResults, err = s.httpxClient.ProbeDomainsWithDetails(httpxCtx, cleanSubdomains)
		httpxCancel()

		probeDuration := time.Since(probeStart)

		if err != nil {
			logrus.Warnf("Detailed HTTPX probe failed after %v for domain %s, using all subdomains: %v", probeDuration, domain, err)
			filteredSubdomains = allSubdomains
		} else {
			// Log detailed results analysis
			logrus.Infof("HTTPX probe returned %d results for %d subdomains", len(detailedResults), len(cleanSubdomains))

			// Extract existing subdomains from detailed results
			existingCount := 0
			for _, result := range detailedResults {
				if result.Exists {
					existingCount++
					// Extract domain from URL
					resultDomain := s.httpxClient.ExtractDomainFromURL(result.URL)
					if resultDomain != "" {
						filteredSubdomains = append(filteredSubdomains, resultDomain)
					}
				}
			}

			logrus.Infof("Detailed HTTPX probe completed in %v for domain %s: %d/%d subdomains exist (captured %d detailed responses, %d existing)",
				probeDuration, domain, len(filteredSubdomains), len(allSubdomains), len(detailedResults), existingCount)

			// Warn if we got significantly fewer results than expected
			if len(detailedResults) < len(cleanSubdomains) {
				missingCount := len(cleanSubdomains) - len(detailedResults)
				logrus.Warnf("HTTPX probe incomplete for domain %s: %d/%d subdomains processed, %d missing",
					domain, len(detailedResults), len(cleanSubdomains), missingCount)
			}
		}
	} else {
		logrus.Infof("HTTPX probe not configured or no subdomains to probe for domain %s, using all subdomains", domain)
		filteredSubdomains = allSubdomains
	}

	// Filter out subdomains that match out-of-scope assets
	if len(outOfScopeAssets) > 0 {
		filteredSubdomains = s.filterOutOfScopeSubdomains(filteredSubdomains, outOfScopeAssets)
		logrus.Infof("After out-of-scope filtering: %d subdomains remain for domain %s", len(filteredSubdomains), domain)
	}

	// Convert filtered subdomains to assets
	var assets []*database.Asset
	for _, subdomain := range filteredSubdomains {
		// Skip empty subdomains
		if strings.TrimSpace(subdomain) == "" {
			continue
		}

		// Create full URL
		url := fmt.Sprintf("https://%s", subdomain)

		// Extract domain and subdomain
		extractedDomain, err := s.urlProcessor.ExtractDomain(url)
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
			Domain:     extractedDomain,
			Subdomain:  subdomainName,
			Status:     "active",
			Source:     "secondary", // Mark as secondary asset from ChaosDB
		}

		assets = append(assets, asset)
	}

	// Save filtered ChaosDB assets to database
	if len(assets) > 0 {
		if err := s.assetRepo.CreateAssets(ctx, assets); err != nil {
			logrus.Warnf("Failed to save ChaosDB assets for domain %s: %v", domain, err)
			// Don't return error, just log warning to continue processing
			// Skip saving detailed responses since assets weren't saved
		} else {
			// Save detailed HTTPX responses if we have them and assets were successfully saved
			if len(detailedResults) > 0 {
				s.saveDetailedResponses(ctx, assets, detailedResults)
			}
		}
	}

	return assets, nil
}

// extractUniqueDomains extracts unique domains from scope assets (only domain and wildcard types)
func (s *MonitorService) extractUniqueDomains(scopeAssets []*platforms.ScopeAsset) []string {
	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("extractUniqueDomains panicked: %v", r)
		}
	}()

	domainMap := make(map[string]bool)
	var domains []string

	for _, asset := range scopeAssets {
		// Only process domain and wildcard type assets for ChaosDB discovery
		if asset.Type != "url" && asset.Type != "wildcard" {
			continue
		}

		// Skip empty URLs
		if strings.TrimSpace(asset.URL) == "" {
			continue
		}

		var domain string
		var err error

		if asset.Type == "wildcard" {
			// For wildcard assets, extract the base domain (wildcards already stripped)
			// This ensures ChaosDB discovers subdomains under the base domain
			domain, err = s.urlProcessor.ExtractDomain(asset.URL)
			if err != nil {
				logrus.Warnf("Failed to extract base domain from %s: %v", asset.URL, err)
				continue
			}
		} else {
			// For regular URL assets, extract domain normally
			domain, err = s.urlProcessor.ExtractDomain(asset.URL)
			if err != nil {
				logrus.Warnf("Failed to extract domain from %s: %v", asset.URL, err)
				continue
			}
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
	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("markInactivePrograms panicked for platform %s: %v", platformName, r)
		}
	}()

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

// saveDetailedResponses saves detailed HTTPX responses to the database
func (s *MonitorService) saveDetailedResponses(ctx context.Context, assets []*database.Asset, detailedResults []httpx.DetailedProbeResult) {
	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("saveDetailedResponses panicked: %v", r)
		}
	}()

	// Create a map of URL to Asset for quick lookup
	urlToAsset := make(map[string]*database.Asset)
	for _, asset := range assets {
		urlToAsset[asset.URL] = asset
	}

	// Save each detailed response
	savedCount := 0
	for _, result := range detailedResults {
		if !result.Exists {
			continue // Skip non-existing domains
		}

		// Find the corresponding asset
		asset, exists := urlToAsset[result.URL]
		if !exists {
			logrus.Debugf("No corresponding asset found for URL: %s", result.URL)
			continue
		}

		// Skip if asset doesn't have a valid ID (wasn't saved to database)
		if asset.ID == uuid.Nil {
			logrus.Debugf("Asset for URL %s has no valid ID, skipping response save", result.URL)
			continue
		}

		// Convert headers to JSON string
		var headersJSON string
		if len(result.Headers) > 0 {
			if headersBytes, err := json.Marshal(result.Headers); err == nil {
				headersJSON = string(headersBytes)
			} else {
				logrus.Warnf("Failed to marshal headers for %s: %v", result.URL, err)
				headersJSON = "{}"
			}
		} else {
			headersJSON = "{}"
		}

		// Create AssetResponse record
		assetResponse := &database.AssetResponse{
			AssetID:      asset.ID,
			StatusCode:   result.StatusCode,
			Headers:      headersJSON,
			Body:         result.Body,
			ResponseTime: result.ResponseTime,
		}

		// Save to database
		if err := s.assetRepo.CreateAssetResponse(ctx, assetResponse); err != nil {
			logrus.Warnf("Failed to save asset response for %s: %v", result.URL, err)
		} else {
			savedCount++
			logrus.Debugf("Saved detailed response for %s (status: %d, body size: %d bytes)",
				result.URL, result.StatusCode, len(result.Body))
		}
	}

	logrus.Infof("Saved %d detailed HTTPX responses to database", savedCount)
}

// filterOutOfScopeSubdomains filters out subdomains that match out-of-scope assets
func (s *MonitorService) filterOutOfScopeSubdomains(subdomains []string, outOfScopeAssets []*platforms.ScopeAsset) []string {
	var filteredSubdomains []string

	for _, subdomain := range subdomains {
		shouldExclude := false

		// Create full URL for comparison
		subdomainURL := fmt.Sprintf("https://%s", subdomain)

		for _, outOfScopeAsset := range outOfScopeAssets {
			if s.matchesOutOfScopeAsset(subdomainURL, outOfScopeAsset) {
				logrus.Debugf("Excluding subdomain %s - matches out-of-scope asset: %s", subdomain, outOfScopeAsset.URL)
				shouldExclude = true
				break
			}
		}

		if !shouldExclude {
			filteredSubdomains = append(filteredSubdomains, subdomain)
		}
	}

	return filteredSubdomains
}

// matchesOutOfScopeAsset checks if a subdomain URL matches an out-of-scope asset
func (s *MonitorService) matchesOutOfScopeAsset(subdomainURL string, outOfScopeAsset *platforms.ScopeAsset) bool {
	switch outOfScopeAsset.Type {
	case "url":
		// For URL assets, check if the subdomain URL exactly matches or is a subdomain of the out-of-scope URL
		return s.urlProcessor.IsSubdomainOf(subdomainURL, outOfScopeAsset.URL)
	case "wildcard":
		// For wildcard assets, check if the subdomain matches the wildcard pattern
		// Use OriginalPattern if available, otherwise fall back to URL
		pattern := outOfScopeAsset.OriginalPattern
		if pattern == "" {
			pattern = outOfScopeAsset.URL
		}
		return s.urlProcessor.MatchesWildcard(subdomainURL, pattern)
	default:
		// For other types, don't filter
		return false
	}
}

// ProgramStats represents program statistics
type ProgramStats struct {
	TotalPrograms  int              `json:"total_programs"`
	ActivePrograms int              `json:"active_programs"`
	TotalAssets    int              `json:"total_assets"`
	RecentScans    []*database.Scan `json:"recent_scans"`
}
