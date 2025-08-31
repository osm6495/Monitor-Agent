package database

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	return sqlxDB, mock, func() {
		sqlxDB.Close()
	}
}

func TestProgramRepository_CreateProgram(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewProgramRepository(db)
	ctx := context.Background()

	program := &Program{
		Name:       "Test Program",
		Platform:   "hackerone",
		URL:        "https://example.com",
		ProgramURL: "https://hackerone.com/program",
		IsActive:   true,
	}

	mock.ExpectExec("INSERT INTO programs").
		WithArgs(sqlmock.AnyArg(), program.Name, program.Platform, program.URL, program.ProgramURL, program.IsActive, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateProgram(ctx, program)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, program.ID)
	assert.False(t, program.CreatedAt.IsZero())
	assert.False(t, program.UpdatedAt.IsZero())
	assert.False(t, program.LastUpdated.IsZero())
}

func TestProgramRepository_GetProgramByID(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewProgramRepository(db)
	ctx := context.Background()

	expectedID := uuid.New()
	expectedProgram := &Program{
		ID:          expectedID,
		Name:        "Test Program",
		Platform:    "hackerone",
		URL:         "https://example.com",
		ProgramURL:  "https://hackerone.com/program",
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		LastUpdated: time.Now(),
	}

	rows := sqlmock.NewRows([]string{"id", "name", "platform", "url", "program_url", "is_active", "last_updated", "created_at", "updated_at"}).
		AddRow(expectedProgram.ID, expectedProgram.Name, expectedProgram.Platform, expectedProgram.URL, expectedProgram.ProgramURL, expectedProgram.IsActive, expectedProgram.LastUpdated, expectedProgram.CreatedAt, expectedProgram.UpdatedAt)

	mock.ExpectQuery("SELECT \\* FROM programs WHERE id = \\$1").
		WithArgs(expectedID).
		WillReturnRows(rows)

	program, err := repo.GetProgramByID(ctx, expectedID)
	assert.NoError(t, err)
	assert.Equal(t, expectedProgram, program)
}

func TestProgramRepository_GetProgramByID_NotFound(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewProgramRepository(db)
	ctx := context.Background()

	expectedID := uuid.New()

	mock.ExpectQuery("SELECT \\* FROM programs WHERE id = \\$1").
		WithArgs(expectedID).
		WillReturnError(sql.ErrNoRows)

	program, err := repo.GetProgramByID(ctx, expectedID)
	assert.NoError(t, err)
	assert.Nil(t, program)
}

func TestProgramRepository_GetAllActivePrograms(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewProgramRepository(db)
	ctx := context.Background()

	expectedPrograms := []*Program{
		{
			ID:          uuid.New(),
			Name:        "Program 1",
			Platform:    "hackerone",
			URL:         "https://example1.com",
			ProgramURL:  "https://hackerone.com/program1",
			IsActive:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			LastUpdated: time.Now(),
		},
		{
			ID:          uuid.New(),
			Name:        "Program 2",
			Platform:    "bugcrowd",
			URL:         "https://example2.com",
			ProgramURL:  "https://bugcrowd.com/program2",
			IsActive:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			LastUpdated: time.Now(),
		},
	}

	rows := sqlmock.NewRows([]string{"id", "name", "platform", "url", "program_url", "is_active", "last_updated", "created_at", "updated_at"})
	for _, p := range expectedPrograms {
		rows.AddRow(p.ID, p.Name, p.Platform, p.URL, p.ProgramURL, p.IsActive, p.LastUpdated, p.CreatedAt, p.UpdatedAt)
	}

	mock.ExpectQuery("SELECT \\* FROM programs WHERE is_active = true ORDER BY last_updated DESC").
		WillReturnRows(rows)

	programs, err := repo.GetAllActivePrograms(ctx)
	assert.NoError(t, err)
	assert.Len(t, programs, 2)
	assert.Equal(t, expectedPrograms[0].Name, programs[0].Name)
	assert.Equal(t, expectedPrograms[1].Name, programs[1].Name)
}

func TestAssetRepository_CreateAsset(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewAssetRepository(db)
	ctx := context.Background()

	programID := uuid.New()
	asset := &Asset{
		ProgramID:  programID,
		ProgramURL: "https://example.com/program1",
		URL:        "https://subdomain.example.com",
		Domain:     "example.com",
		Subdomain:  "subdomain",
		Status:     "active",
		Source:     "chaosdb",
	}

	mock.ExpectExec("INSERT INTO assets").
		WithArgs(sqlmock.AnyArg(), asset.ProgramID, asset.ProgramURL, asset.URL, asset.Domain, asset.Subdomain, asset.IP, asset.Status, asset.Source, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateAsset(ctx, asset)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, asset.ID)
	assert.False(t, asset.CreatedAt.IsZero())
	assert.False(t, asset.UpdatedAt.IsZero())
}

func TestAssetRepository_CreateAssets(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewAssetRepository(db)
	ctx := context.Background()

	programID := uuid.New()
	assets := []*Asset{
		{
			ProgramID:  programID,
			ProgramURL: "https://example.com/program1",
			URL:        "https://sub1.example.com",
			Domain:     "example.com",
			Subdomain:  "sub1",
			Status:     "active",
			Source:     "chaosdb",
		},
		{
			ProgramID:  programID,
			ProgramURL: "https://example.com/program1",
			URL:        "https://sub2.example.com",
			Domain:     "example.com",
			Subdomain:  "sub2",
			Status:     "active",
			Source:     "chaosdb",
		},
	}

	mock.ExpectBegin()
	for i := 0; i < 2; i++ {
		mock.ExpectExec("INSERT INTO assets").
			WithArgs(sqlmock.AnyArg(), programID, assets[i].ProgramURL, assets[i].URL, assets[i].Domain, assets[i].Subdomain, assets[i].IP, assets[i].Status, assets[i].Source, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectCommit()

	err := repo.CreateAssets(ctx, assets)
	assert.NoError(t, err)
}

func TestAssetRepository_GetAssetsByProgramID(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewAssetRepository(db)
	ctx := context.Background()

	programID := uuid.New()
	expectedAssets := []*Asset{
		{
			ID:         uuid.New(),
			ProgramID:  programID,
			ProgramURL: "https://example.com/program1",
			URL:        "https://sub1.example.com",
			Domain:     "example.com",
			Subdomain:  "sub1",
			Status:     "active",
			Source:     "chaosdb",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
		{
			ID:         uuid.New(),
			ProgramID:  programID,
			ProgramURL: "https://example.com/program1",
			URL:        "https://sub2.example.com",
			Domain:     "example.com",
			Subdomain:  "sub2",
			Status:     "active",
			Source:     "chaosdb",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
	}

	rows := sqlmock.NewRows([]string{"id", "program_id", "program_url", "url", "domain", "subdomain", "ip", "status", "source", "created_at", "updated_at"})
	for _, a := range expectedAssets {
		rows.AddRow(a.ID, a.ProgramID, a.ProgramURL, a.URL, a.Domain, a.Subdomain, a.IP, a.Status, a.Source, a.CreatedAt, a.UpdatedAt)
	}

	mock.ExpectQuery("SELECT \\* FROM assets WHERE program_id = \\$1 ORDER BY created_at DESC").
		WithArgs(programID).
		WillReturnRows(rows)

	assets, err := repo.GetAssetsByProgramID(ctx, programID)
	assert.NoError(t, err)
	assert.Len(t, assets, 2)
	assert.Equal(t, expectedAssets[0].URL, assets[0].URL)
	assert.Equal(t, expectedAssets[1].URL, assets[1].URL)
}

func TestAssetRepository_DeleteAssetsByProgramID(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewAssetRepository(db)
	ctx := context.Background()

	programID := uuid.New()

	mock.ExpectExec("DELETE FROM assets WHERE program_id = \\$1").
		WithArgs(programID).
		WillReturnResult(sqlmock.NewResult(0, 5))

	err := repo.DeleteAssetsByProgramID(ctx, programID)
	assert.NoError(t, err)
}

func TestAssetRepository_GetAssetsByProgramIDAndSource(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewAssetRepository(db)
	ctx := context.Background()

	programID := uuid.New()
	source := "primary"
	expectedAssets := []*Asset{
		{
			ID:         uuid.New(),
			ProgramID:  programID,
			ProgramURL: "https://example.com/program1",
			URL:        "https://sub1.example.com",
			Domain:     "example.com",
			Subdomain:  "sub1",
			Status:     "active",
			Source:     source,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
		{
			ID:         uuid.New(),
			ProgramID:  programID,
			ProgramURL: "https://example.com/program1",
			URL:        "https://sub2.example.com",
			Domain:     "example.com",
			Subdomain:  "sub2",
			Status:     "active",
			Source:     source,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
	}

	rows := sqlmock.NewRows([]string{"id", "program_id", "program_url", "url", "domain", "subdomain", "ip", "status", "source", "created_at", "updated_at"})
	for _, a := range expectedAssets {
		rows.AddRow(a.ID, a.ProgramID, a.ProgramURL, a.URL, a.Domain, a.Subdomain, a.IP, a.Status, a.Source, a.CreatedAt, a.UpdatedAt)
	}

	mock.ExpectQuery("SELECT \\* FROM assets WHERE program_id = \\$1 AND source = \\$2 ORDER BY created_at DESC").
		WithArgs(programID, source).
		WillReturnRows(rows)

	assets, err := repo.GetAssetsByProgramIDAndSource(ctx, programID, source)
	assert.NoError(t, err)
	assert.Len(t, assets, 2)
	assert.Equal(t, expectedAssets[0].URL, assets[0].URL)
	assert.Equal(t, expectedAssets[1].URL, assets[1].URL)
	assert.Equal(t, source, assets[0].Source)
	assert.Equal(t, source, assets[1].Source)
}

func TestScanRepository_CreateScan(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewScanRepository(db)
	ctx := context.Background()

	programID := uuid.New()
	scan := &Scan{
		ProgramID:   programID,
		Status:      "running",
		AssetsFound: 0,
	}

	mock.ExpectExec("INSERT INTO scans").
		WithArgs(sqlmock.AnyArg(), scan.ProgramID, scan.Status, scan.AssetsFound, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateScan(ctx, scan)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, scan.ID)
	assert.False(t, scan.CreatedAt.IsZero())
	assert.False(t, scan.UpdatedAt.IsZero())
	assert.False(t, scan.StartedAt.IsZero())
}

func TestScanRepository_UpdateScan(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewScanRepository(db)
	ctx := context.Background()

	scanID := uuid.New()
	completedAt := time.Now()
	scan := &Scan{
		ID:          scanID,
		ProgramID:   uuid.New(),
		Status:      "completed",
		AssetsFound: 10,
		CompletedAt: &completedAt,
		Error:       "",
	}

	mock.ExpectExec("UPDATE scans").
		WithArgs(scan.Status, scan.AssetsFound, scan.CompletedAt, scan.Error, sqlmock.AnyArg(), scanID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateScan(ctx, scan)
	assert.NoError(t, err)
	assert.False(t, scan.UpdatedAt.IsZero())
}

func TestScanRepository_GetScanByID(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewScanRepository(db)
	ctx := context.Background()

	scanID := uuid.New()
	expectedScan := &Scan{
		ID:          scanID,
		ProgramID:   uuid.New(),
		Status:      "completed",
		AssetsFound: 10,
		StartedAt:   time.Now(),
		CompletedAt: &time.Time{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	rows := sqlmock.NewRows([]string{"id", "program_id", "status", "assets_found", "started_at", "completed_at", "error", "created_at", "updated_at"}).
		AddRow(expectedScan.ID, expectedScan.ProgramID, expectedScan.Status, expectedScan.AssetsFound, expectedScan.StartedAt, expectedScan.CompletedAt, expectedScan.Error, expectedScan.CreatedAt, expectedScan.UpdatedAt)

	mock.ExpectQuery("SELECT \\* FROM scans WHERE id = \\$1").
		WithArgs(scanID).
		WillReturnRows(rows)

	scan, err := repo.GetScanByID(ctx, scanID)
	assert.NoError(t, err)
	assert.Equal(t, expectedScan, scan)
}

func TestAssetRepository_GetAssetCountByProgramID(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewAssetRepository(db)
	ctx := context.Background()

	programID := uuid.New()
	expectedCount := 15

	rows := sqlmock.NewRows([]string{"count"}).AddRow(expectedCount)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM assets WHERE program_id = \\$1").
		WithArgs(programID).
		WillReturnRows(rows)

	count, err := repo.GetAssetCountByProgramID(ctx, programID)
	assert.NoError(t, err)
	assert.Equal(t, expectedCount, count)
}

func TestProgramRepository_MarkProgramInactive(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewProgramRepository(db)
	ctx := context.Background()

	programID := uuid.New()

	mock.ExpectExec("UPDATE programs SET is_active = false, updated_at = NOW\\(\\) WHERE id = \\$1").
		WithArgs(programID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.MarkProgramInactive(ctx, programID)
	assert.NoError(t, err)
}

func TestProgramRepository_MarkProgramInactive_NotFound(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewProgramRepository(db)
	ctx := context.Background()

	programID := uuid.New()

	mock.ExpectExec("UPDATE programs SET is_active = false, updated_at = NOW\\(\\) WHERE id = \\$1").
		WithArgs(programID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.MarkProgramInactive(ctx, programID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "program not found")
}
