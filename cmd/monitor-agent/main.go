package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/monitor-agent/internal/config"
	"github.com/monitor-agent/internal/service"
	"github.com/sirupsen/logrus"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	// Debug: Log the database configuration
	logrus.Infof("Database config - Host: '%s', Port: %d, Name: '%s', User: '%s'",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.Name, cfg.Database.User)
	logrus.Infof("Database DSN: %s", cfg.GetDSN())
	if err != nil {
		logrus.Errorf("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		logrus.Errorf("Invalid configuration: %v", err)
		os.Exit(1)
	}

	// Set log level
	level, err := logrus.ParseLevel(cfg.App.LogLevel)
	if err != nil {
		logrus.Errorf("Invalid log level: %v", err)
		os.Exit(1)
	}
	logrus.SetLevel(level)

	// Connect to database
	db, err := connectToDatabase(cfg)
	if err != nil {
		logrus.Errorf("Failed to connect to database: %v", err)
		os.Exit(1)
	}
	defer db.Close()

	// Run database migrations
	if err := runMigrations(db); err != nil {
		logrus.Errorf("Failed to run database migrations: %v", err)
		os.Exit(1)
	}

	// Initialize monitor service
	monitorService := service.NewMonitorService(cfg, db)

	// Log configured platforms
	configuredPlatforms := cfg.GetConfiguredPlatforms()
	if len(configuredPlatforms) == 0 {
		logrus.Warn("No API keys configured. The application will start but cannot perform scans.")
		logrus.Info("To enable scanning, set one or more of: HACKERONE_API_KEY, BUGCROWD_API_KEY, CHAOSDB_API_KEY")
	} else {
		logrus.Infof("Configured platforms: %v", configuredPlatforms)
	}

	// Check command line arguments
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "scan":
			if err := runScan(context.Background(), monitorService); err != nil {
				logrus.Errorf("Scan failed: %v", err)
				os.Exit(1)
			}
			return
		case "stats":
			if err := showStats(context.Background(), monitorService); err != nil {
				logrus.Errorf("Failed to get stats: %v", err)
				os.Exit(1)
			}
			return
		case "health":
			if err := checkHealth(context.Background(), monitorService); err != nil {
				logrus.Errorf("Health check failed: %v", err)
				os.Exit(1)
			}
			return
		case "help":
			showHelp()
			return
		default:
			logrus.Errorf("Unknown command: %s. Use 'help' for usage information.", os.Args[1])
			os.Exit(1)
		}
	}

	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Default behavior: run a scan
	logrus.Info("No command specified, running scan...")

	// Run scan in a goroutine so we can handle shutdown signals
	scanDone := make(chan error, 1)
	go func() {
		// Add a reasonable timeout for the scan
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		scanDone <- runScan(ctx, monitorService)
	}()

	// Wait for either scan completion or shutdown signal
	select {
	case err := <-scanDone:
		if err != nil {
			logrus.Errorf("Scan failed: %v", err)
			os.Exit(1)
		}
	case sig := <-sigChan:
		logrus.Infof("Received signal %v, shutting down gracefully...", sig)
		// Give the scan a chance to complete
		select {
		case err := <-scanDone:
			if err != nil {
				logrus.Errorf("Scan failed: %v", err)
				os.Exit(1)
			}
		case <-time.After(30 * time.Second):
			logrus.Warn("Scan did not complete within 30 seconds, forcing shutdown")
		}
	}
}

// connectToDatabase connects to the PostgreSQL database
func connectToDatabase(cfg *config.Config) (*sqlx.DB, error) {
	dsn := cfg.GetDSN()
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings from configuration
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	logrus.Info("Successfully connected to database")
	return db, nil
}

// runMigrations executes database migrations
func runMigrations(db *sqlx.DB) error {
	// Read migration file
	migrationSQL, err := os.ReadFile("migrations/001_initial_schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Execute migration
	_, err = db.Exec(string(migrationSQL))
	if err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	logrus.Info("Database migrations completed successfully")
	return nil
}

// runScan performs a single scan
func runScan(ctx context.Context, monitorService *service.MonitorService) error {
	logrus.Info("Starting scan of all bug bounty platforms...")

	startTime := time.Now()
	if err := monitorService.RunFullScan(ctx); err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	duration := time.Since(startTime)
	logrus.Infof("Scan completed successfully in %v", duration)
	return nil
}

// showStats displays program statistics
func showStats(ctx context.Context, monitorService *service.MonitorService) error {
	stats, err := monitorService.GetProgramStats(ctx)
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	fmt.Printf("\n=== Monitor Agent Statistics ===\n")
	fmt.Printf("Total Programs: %d\n", stats.TotalPrograms)
	fmt.Printf("Active Programs: %d\n", stats.ActivePrograms)
	fmt.Printf("Total Assets: %d\n", stats.TotalAssets)

	if len(stats.RecentScans) > 0 {
		fmt.Printf("\nRecent Scans:\n")
		for _, scan := range stats.RecentScans {
			status := scan.Status
			if scan.Error != "" {
				status = "failed"
			}
			fmt.Printf("  - %s: %s (%d assets found)\n",
				scan.StartedAt.Format("2006-01-02 15:04:05"),
				status,
				scan.AssetsFound)
		}
	}

	return nil
}

// checkHealth performs health checks
func checkHealth(ctx context.Context, monitorService *service.MonitorService) error {
	logrus.Info("Performing health checks...")

	// Check database connectivity
	if err := monitorService.CheckDatabaseHealth(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	// Check platform API health
	if err := monitorService.CheckPlatformHealth(ctx); err != nil {
		return fmt.Errorf("platform health check failed: %w", err)
	}

	// Check ChaosDB health
	if err := monitorService.CheckChaosDBHealth(ctx); err != nil {
		return fmt.Errorf("ChaosDB health check failed: %w", err)
	}

	// Check system resources
	if err := monitorService.CheckSystemHealth(ctx); err != nil {
		return fmt.Errorf("system health check failed: %w", err)
	}

	logrus.Info("All health checks passed")
	return nil
}

// showHelp displays usage information
func showHelp() {
	fmt.Printf(`
Monitor Agent - Bug Bounty Program Monitor

Usage:
  monitor-agent [command]

Commands:
  scan     Perform a scan of all platforms (default behavior)
  stats    Show program and asset statistics
  health   Perform health checks
  help     Show this help message

Environment Variables:
  DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD (required)
  HACKERONE_API_KEY, BUGCROWD_API_KEY, CHAOSDB_API_KEY (optional)
  LOG_LEVEL, ENVIRONMENT

Examples:
  monitor-agent          # Run a scan (default)
  monitor-agent scan     # Explicitly run a scan
  monitor-agent stats    # Show statistics
  monitor-agent health   # Health check

This application performs one-off scans of bug bounty platforms.
API keys are optional - the application will only scan platforms with configured keys.
For scheduled scanning, use external cron or Kubernetes CronJobs.
`)
}
