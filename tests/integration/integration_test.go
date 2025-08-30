//go:build integration

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/monitor-agent/internal/config"
	"github.com/monitor-agent/internal/database"
	"github.com/monitor-agent/internal/discovery/chaosdb"
	"github.com/monitor-agent/internal/platforms"
	"github.com/monitor-agent/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testDB *sqlx.DB
	cfg    *config.Config
)

func TestMain(m *testing.M) {
	// Setup test environment
	setupTestEnv()

	// Run tests
	code := m.Run()

	// Cleanup
	teardownTestEnv()

	os.Exit(code)
}

func setupTestEnv() {
	var err error

	// Load test configuration
	cfg, err = config.Load()
	if err != nil {
		panic(err)
	}

	// Connect to test database
	testDB, err = sqlx.Connect("postgres", cfg.GetDSN())
	if err != nil {
		panic(err)
	}

	// Run migrations
	runMigrations()
}

func teardownTestEnv() {
	if testDB != nil {
		testDB.Close()
	}
}

func runMigrations() {
	// Read and execute migration file
	migrationSQL, err := os.ReadFile("../../internal/database/migrations/001_initial_schema.sql")
	if err != nil {
		panic(err)
	}

	_, err = testDB.Exec(string(migrationSQL))
	if err != nil {
		panic(err)
	}
}

func TestMonitorServiceIntegration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Create monitor service
	monitorService := service.NewMonitorService(cfg, testDB)

	// Test getting program stats
	stats, err := monitorService.GetProgramStats(ctx)
	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats.TotalPrograms) // Should be empty initially
}

func TestDatabaseIntegration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Test program repository
	programRepo := database.NewProgramRepository(testDB)

	// Create a test program
	testProgram := &database.Program{
		Name:       "Test Program",
		Platform:   "test",
		URL:        "https://test.example.com",
		ProgramURL: "https://test.example.com/program",
		IsActive:   true,
	}

	err := programRepo.CreateProgram(ctx, testProgram)
	require.NoError(t, err)
	assert.NotEqual(t, "", testProgram.ID.String())

	// Retrieve the program
	retrievedProgram, err := programRepo.GetProgramByID(ctx, testProgram.ID)
	require.NoError(t, err)
	assert.Equal(t, testProgram.Name, retrievedProgram.Name)

	// Test asset repository
	assetRepo := database.NewAssetRepository(testDB)

	// Create a test asset
	testAsset := &database.Asset{
		ProgramID: testProgram.ID,
		URL:       "https://subdomain.test.example.com",
		Domain:    "test.example.com",
		Subdomain: "subdomain",
		Status:    "active",
		Source:    "test",
	}

	err = assetRepo.CreateAsset(ctx, testAsset)
	require.NoError(t, err)
	assert.NotEqual(t, "", testAsset.ID.String())

	// Retrieve assets for the program
	assets, err := assetRepo.GetAssetsByProgramID(ctx, testProgram.ID)
	require.NoError(t, err)
	assert.Len(t, assets, 1)
	assert.Equal(t, testAsset.URL, assets[0].URL)
}

func TestPlatformIntegration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test would require actual API keys and would test real API integration
	// For now, we'll just test the configuration loading

	// Test that platform factory can be created
	platformFactory := platforms.NewPlatformFactory()
	assert.NotNil(t, platformFactory)

	// Test that platforms can be registered
	platformFactory.RegisterPlatform("test", &platforms.PlatformConfig{
		APIKey:        "test-key",
		RateLimit:     100,
		Timeout:       30 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    1 * time.Second,
	})

	// Test that platform can be retrieved
	platform, err := platformFactory.GetPlatform("test")
	assert.NoError(t, err)
	assert.NotNil(t, platform)
}

func TestChaosDBIntegration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test would require actual ChaosDB API key
	// For now, we'll just test the client creation

	chaosDBClient := chaosdb.NewClient(&chaosdb.ClientConfig{
		APIKey:        "test-key",
		RateLimit:     50,
		Timeout:       30 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    1 * time.Second,
	})

	assert.NotNil(t, chaosDBClient)
	assert.Equal(t, 50, chaosDBClient.GetRateLimit())
}

func TestFullWorkflowIntegration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test would simulate a full workflow from program discovery to asset storage
	// It would require actual API keys and would be quite comprehensive

	ctx := context.Background()

	// Create monitor service
	monitorService := service.NewMonitorService(cfg, testDB)

	// Test that the service can be created without errors
	assert.NotNil(t, monitorService)

	// Test that we can get stats (even if empty)
	stats, err := monitorService.GetProgramStats(ctx)
	require.NoError(t, err)
	assert.NotNil(t, stats)
}
