package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/monitor-agent/internal/config"
	"github.com/monitor-agent/internal/service"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
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

	// Initialize monitor service
	monitorService := service.NewMonitorService(cfg, db)

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

	// Run as a service with cron scheduling
	runAsService(context.Background(), cfg, monitorService)
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

// runScan performs a single scan
func runScan(ctx context.Context, monitorService *service.MonitorService) error {
	logrus.Info("Starting manual scan...")

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
  scan     Perform a single scan of all platforms
  stats    Show program and asset statistics
  health   Perform health checks
  help     Show this help message

Environment Variables:
  DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD
  HACKERONE_API_KEY, BUGCROWD_API_KEY, CHAOSDB_API_KEY
  LOG_LEVEL, CRON_SCHEDULE

Examples:
  monitor-agent scan
  monitor-agent stats
  monitor-agent health

When run without arguments, the agent runs as a service with cron scheduling.
`)
}

// runAsService runs the agent as a service with cron scheduling
func runAsService(_ context.Context, cfg *config.Config, monitorService *service.MonitorService) {
	logrus.Info("Starting Monitor Agent as a service...")

	// Create cron scheduler
	c := cron.New(cron.WithSeconds())

	// Add the scan job
	entryID, err := c.AddFunc(cfg.App.CronSchedule, func() {
		logrus.Info("Starting scheduled scan...")
		scanCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		if err := monitorService.RunFullScan(scanCtx); err != nil {
			logrus.Errorf("Scheduled scan failed: %v", err)
		} else {
			logrus.Info("Scheduled scan completed successfully")
		}
	})

	if err != nil {
		logrus.Errorf("Failed to schedule scan job: %v", err)
		os.Exit(1)
	}

	logrus.Infof("Scheduled scan job with ID %d using schedule: %s", entryID, cfg.App.CronSchedule)

	// Start the cron scheduler
	c.Start()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logrus.Info("Monitor Agent is running. Press Ctrl+C to stop.")

	// Wait for signal
	<-sigChan

	logrus.Info("Shutting down Monitor Agent...")

	// Stop the cron scheduler
	c.Stop()

	logrus.Info("Monitor Agent stopped gracefully")
}
