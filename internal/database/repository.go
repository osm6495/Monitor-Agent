package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// Repository provides database operations
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new repository instance
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// GetDB returns the underlying database connection
func (r *Repository) GetDB() *sqlx.DB {
	return r.db
}

// ProgramRepository provides program-specific database operations
type ProgramRepository struct {
	*Repository
}

// AssetRepository provides asset-specific database operations
type AssetRepository struct {
	*Repository
}

// ScanRepository provides scan-specific database operations
type ScanRepository struct {
	*Repository
}

// NewProgramRepository creates a new program repository
func NewProgramRepository(db *sqlx.DB) *ProgramRepository {
	return &ProgramRepository{Repository: NewRepository(db)}
}

// NewAssetRepository creates a new asset repository
func NewAssetRepository(db *sqlx.DB) *AssetRepository {
	return &AssetRepository{Repository: NewRepository(db)}
}

// NewScanRepository creates a new scan repository
func NewScanRepository(db *sqlx.DB) *ScanRepository {
	return &ScanRepository{Repository: NewRepository(db)}
}

// Program Operations

// CreateProgram creates a new program
func (r *ProgramRepository) CreateProgram(ctx context.Context, program *Program) error {
	program.ID = uuid.New()
	program.CreatedAt = time.Now()
	program.UpdatedAt = time.Now()
	program.LastUpdated = time.Now()

	query := `
		INSERT INTO programs (id, name, platform, url, program_url, is_active, last_updated, created_at, updated_at)
		VALUES (:id, :name, :platform, :url, :program_url, :is_active, :last_updated, :created_at, :updated_at)
	`

	_, err := r.db.NamedExecContext(ctx, query, program)
	if err != nil {
		return fmt.Errorf("failed to create program: %w", err)
	}

	return nil
}

// GetProgramByID retrieves a program by ID
func (r *ProgramRepository) GetProgramByID(ctx context.Context, id uuid.UUID) (*Program, error) {
	var program Program
	query := `SELECT * FROM programs WHERE id = $1`

	err := r.db.GetContext(ctx, &program, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get program: %w", err)
	}

	return &program, nil
}

// GetProgramByPlatformAndURL retrieves a program by platform and URL
func (r *ProgramRepository) GetProgramByPlatformAndURL(ctx context.Context, platform, url string) (*Program, error) {
	var program Program
	query := `SELECT * FROM programs WHERE platform = $1 AND url = $2`

	err := r.db.GetContext(ctx, &program, query, platform, url)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get program: %w", err)
	}

	return &program, nil
}

// GetProgramByPlatformAndProgramURL retrieves a program by platform and program URL
func (r *ProgramRepository) GetProgramByPlatformAndProgramURL(ctx context.Context, platform, programURL string) (*Program, error) {
	var program Program
	query := `SELECT * FROM programs WHERE platform = $1 AND program_url = $2`

	err := r.db.GetContext(ctx, &program, query, platform, programURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get program: %w", err)
	}

	return &program, nil
}

// GetAllActivePrograms retrieves all active programs
func (r *ProgramRepository) GetAllActivePrograms(ctx context.Context) ([]*Program, error) {
	var programs []*Program
	query := `SELECT * FROM programs WHERE is_active = true ORDER BY last_updated DESC`

	err := r.db.SelectContext(ctx, &programs, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get active programs: %w", err)
	}

	return programs, nil
}

// GetProgramsByPlatform retrieves programs by platform
func (r *ProgramRepository) GetProgramsByPlatform(ctx context.Context, platform string) ([]*Program, error) {
	var programs []*Program
	query := `SELECT * FROM programs WHERE platform = $1 AND is_active = true ORDER BY last_updated DESC`

	err := r.db.SelectContext(ctx, &programs, query, platform)
	if err != nil {
		return nil, fmt.Errorf("failed to get programs by platform: %w", err)
	}

	return programs, nil
}

// UpdateProgram updates a program
func (r *ProgramRepository) UpdateProgram(ctx context.Context, program *Program) error {
	program.UpdatedAt = time.Now()
	program.LastUpdated = time.Now()

	query := `
		UPDATE programs 
		SET name = :name, platform = :platform, url = :url, program_url = :program_url, 
		    is_active = :is_active, last_updated = :last_updated, updated_at = :updated_at
		WHERE id = :id
	`

	result, err := r.db.NamedExecContext(ctx, query, program)
	if err != nil {
		return fmt.Errorf("failed to update program: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("program not found")
	}

	return nil
}

// MarkProgramInactive marks a program as inactive
func (r *ProgramRepository) MarkProgramInactive(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE programs SET is_active = false, updated_at = NOW() WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to mark program inactive: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("program not found")
	}

	return nil
}

// DeleteProgram deletes a program and all associated assets
func (r *ProgramRepository) DeleteProgram(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM programs WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete program: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("program not found")
	}

	return nil
}

// Asset Operations

// CreateAsset creates a new asset
func (r *AssetRepository) CreateAsset(ctx context.Context, asset *Asset) error {
	asset.ID = uuid.New()
	asset.CreatedAt = time.Now()
	asset.UpdatedAt = time.Now()

	query := `
		INSERT INTO assets (id, program_id, program_url, url, domain, subdomain, ip, status, source, created_at, updated_at)
		VALUES (:id, :program_id, :program_url, :url, :domain, :subdomain, :ip, :status, :source, :created_at, :updated_at)
		ON CONFLICT (program_id, url) DO UPDATE SET
			program_url = EXCLUDED.program_url,
			domain = EXCLUDED.domain,
			subdomain = EXCLUDED.subdomain,
			ip = EXCLUDED.ip,
			status = EXCLUDED.status,
			source = EXCLUDED.source,
			updated_at = NOW()
	`

	_, err := r.db.NamedExecContext(ctx, query, asset)
	if err != nil {
		return fmt.Errorf("failed to create asset: %w", err)
	}

	return nil
}

// CreateAssets creates multiple assets in a transaction
func (r *AssetRepository) CreateAssets(ctx context.Context, assets []*Asset) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Track if we've committed the transaction
	committed := false
	defer func() {
		if !committed {
			if err := tx.Rollback(); err != nil {
				logrus.Errorf("Failed to rollback transaction: %v", err)
			}
		}
	}()

	query := `
		INSERT INTO assets (id, program_id, program_url, url, domain, subdomain, ip, status, source, created_at, updated_at)
		VALUES (:id, :program_id, :program_url, :url, :domain, :subdomain, :ip, :status, :source, :created_at, :updated_at)
		ON CONFLICT (program_id, url) DO UPDATE SET
			program_url = EXCLUDED.program_url,
			domain = EXCLUDED.domain,
			subdomain = EXCLUDED.subdomain,
			ip = EXCLUDED.ip,
			status = EXCLUDED.status,
			source = EXCLUDED.source,
			updated_at = NOW()
	`

	for _, asset := range assets {
		asset.ID = uuid.New()
		asset.CreatedAt = time.Now()
		asset.UpdatedAt = time.Now()

		_, err := tx.NamedExecContext(ctx, query, asset)
		if err != nil {
			return fmt.Errorf("failed to create asset %s: %w", asset.URL, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	committed = true
	return nil
}

// GetAssetsByProgramID retrieves assets by program ID
func (r *AssetRepository) GetAssetsByProgramID(ctx context.Context, programID uuid.UUID) ([]*Asset, error) {
	var assets []*Asset
	query := `SELECT * FROM assets WHERE program_id = $1 ORDER BY created_at DESC`

	err := r.db.SelectContext(ctx, &assets, query, programID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assets by program ID: %w", err)
	}

	return assets, nil
}

// GetAssetsByProgramIDAndSource retrieves assets by program ID and source
func (r *AssetRepository) GetAssetsByProgramIDAndSource(ctx context.Context, programID uuid.UUID, source string) ([]*Asset, error) {
	var assets []*Asset
	query := `SELECT * FROM assets WHERE program_id = $1 AND source = $2 ORDER BY created_at DESC`

	err := r.db.SelectContext(ctx, &assets, query, programID, source)
	if err != nil {
		return nil, fmt.Errorf("failed to get assets by program ID and source: %w", err)
	}

	return assets, nil
}

// GetAssetsByDomain retrieves assets by domain
func (r *AssetRepository) GetAssetsByDomain(ctx context.Context, domain string) ([]*Asset, error) {
	var assets []*Asset
	query := `SELECT * FROM assets WHERE domain = $1 ORDER BY created_at DESC`

	err := r.db.SelectContext(ctx, &assets, query, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to get assets by domain: %w", err)
	}

	return assets, nil
}

// GetAssetsByStatus retrieves assets by status
func (r *AssetRepository) GetAssetsByStatus(ctx context.Context, status string) ([]*Asset, error) {
	var assets []*Asset
	query := `SELECT * FROM assets WHERE status = $1 ORDER BY created_at DESC`

	err := r.db.SelectContext(ctx, &assets, query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to get assets by status: %w", err)
	}

	return assets, nil
}

// DeleteAssetsByProgramID deletes all assets for a program
func (r *AssetRepository) DeleteAssetsByProgramID(ctx context.Context, programID uuid.UUID) error {
	query := `DELETE FROM assets WHERE program_id = $1`

	result, err := r.db.ExecContext(ctx, query, programID)
	if err != nil {
		return fmt.Errorf("failed to delete assets by program ID: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	logrus.Infof("Deleted %d assets for program %s", rowsAffected, programID)
	return nil
}

// Scan Operations

// CreateScan creates a new scan
func (r *ScanRepository) CreateScan(ctx context.Context, scan *Scan) error {
	scan.ID = uuid.New()
	scan.CreatedAt = time.Now()
	scan.UpdatedAt = time.Now()
	scan.StartedAt = time.Now()

	query := `
		INSERT INTO scans (id, program_id, status, assets_found, started_at, created_at, updated_at)
		VALUES (:id, :program_id, :status, :assets_found, :started_at, :created_at, :updated_at)
	`

	_, err := r.db.NamedExecContext(ctx, query, scan)
	if err != nil {
		return fmt.Errorf("failed to create scan: %w", err)
	}

	return nil
}

// UpdateScan updates a scan
func (r *ScanRepository) UpdateScan(ctx context.Context, scan *Scan) error {
	scan.UpdatedAt = time.Now()

	query := `
		UPDATE scans 
		SET status = :status, assets_found = :assets_found, completed_at = :completed_at, 
		    error = :error, updated_at = :updated_at
		WHERE id = :id
	`

	result, err := r.db.NamedExecContext(ctx, query, scan)
	if err != nil {
		return fmt.Errorf("failed to update scan: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("scan not found")
	}

	return nil
}

// GetScanByID retrieves a scan by ID
func (r *ScanRepository) GetScanByID(ctx context.Context, id uuid.UUID) (*Scan, error) {
	var scan Scan
	query := `SELECT * FROM scans WHERE id = $1`

	err := r.db.GetContext(ctx, &scan, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get scan: %w", err)
	}

	return &scan, nil
}

// GetScansByProgramID retrieves scans by program ID
func (r *ScanRepository) GetScansByProgramID(ctx context.Context, programID uuid.UUID) ([]*Scan, error) {
	var scans []*Scan
	query := `SELECT * FROM scans WHERE program_id = $1 ORDER BY started_at DESC`

	err := r.db.SelectContext(ctx, &scans, query, programID)
	if err != nil {
		return nil, fmt.Errorf("failed to get scans by program ID: %w", err)
	}

	return scans, nil
}

// GetRecentScans retrieves recent scans
func (r *ScanRepository) GetRecentScans(ctx context.Context, limit int) ([]*Scan, error) {
	var scans []*Scan
	query := `SELECT * FROM scans ORDER BY started_at DESC LIMIT $1`

	err := r.db.SelectContext(ctx, &scans, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent scans: %w", err)
	}

	return scans, nil
}

// AssetResponse Operations

// CreateAssetResponse creates a new asset response record
func (r *AssetRepository) CreateAssetResponse(ctx context.Context, assetResponse *AssetResponse) error {
	assetResponse.ID = uuid.New()
	assetResponse.CreatedAt = time.Now()

	query := `
		INSERT INTO asset_responses (id, asset_id, status_code, headers, body, response_time, created_at)
		VALUES (:id, :asset_id, :status_code, :headers, :body, :response_time, :created_at)
	`

	_, err := r.db.NamedExecContext(ctx, query, assetResponse)
	if err != nil {
		return fmt.Errorf("failed to create asset response: %w", err)
	}

	return nil
}

// GetAssetResponsesByAssetID retrieves asset responses by asset ID
func (r *AssetRepository) GetAssetResponsesByAssetID(ctx context.Context, assetID uuid.UUID) ([]*AssetResponse, error) {
	var responses []*AssetResponse
	query := `SELECT * FROM asset_responses WHERE asset_id = $1 ORDER BY created_at DESC`

	err := r.db.SelectContext(ctx, &responses, query, assetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get asset responses: %w", err)
	}

	return responses, nil
}

// GetLatestAssetResponseByAssetID retrieves the latest asset response for an asset
func (r *AssetRepository) GetLatestAssetResponseByAssetID(ctx context.Context, assetID uuid.UUID) (*AssetResponse, error) {
	var response AssetResponse
	query := `SELECT * FROM asset_responses WHERE asset_id = $1 ORDER BY created_at DESC LIMIT 1`

	err := r.db.GetContext(ctx, &response, query, assetID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest asset response: %w", err)
	}

	return &response, nil
}

// SearchAssetResponsesByHeaders searches asset responses by header content
func (r *AssetRepository) SearchAssetResponsesByHeaders(ctx context.Context, headerPattern string) ([]*AssetResponse, error) {
	var responses []*AssetResponse
	query := `SELECT * FROM asset_responses WHERE headers ILIKE $1 ORDER BY created_at DESC`

	err := r.db.SelectContext(ctx, &responses, query, "%"+headerPattern+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to search asset responses by headers: %w", err)
	}

	return responses, nil
}

// SearchAssetResponsesByBody searches asset responses by body content
func (r *AssetRepository) SearchAssetResponsesByBody(ctx context.Context, bodyPattern string) ([]*AssetResponse, error) {
	var responses []*AssetResponse
	query := `SELECT * FROM asset_responses WHERE body ILIKE $1 ORDER BY created_at DESC`

	err := r.db.SelectContext(ctx, &responses, query, "%"+bodyPattern+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to search asset responses by body: %w", err)
	}

	return responses, nil
}

// SearchAssetResponsesByStatusCode searches asset responses by status code
func (r *AssetRepository) SearchAssetResponsesByStatusCode(ctx context.Context, statusCode int) ([]*AssetResponse, error) {
	var responses []*AssetResponse
	query := `SELECT * FROM asset_responses WHERE status_code = $1 ORDER BY created_at DESC`

	err := r.db.SelectContext(ctx, &responses, query, statusCode)
	if err != nil {
		return nil, fmt.Errorf("failed to search asset responses by status code: %w", err)
	}

	return responses, nil
}

// GetAssetResponsesWithAssetInfo retrieves asset responses with associated asset information
func (r *AssetRepository) GetAssetResponsesWithAssetInfo(ctx context.Context, limit int) ([]struct {
	AssetResponse *AssetResponse `db:"asset_response"`
	Asset         *Asset         `db:"asset"`
}, error) {
	query := `
		SELECT 
			ar.id as "asset_response.id",
			ar.asset_id as "asset_response.asset_id",
			ar.status_code as "asset_response.status_code",
			ar.headers as "asset_response.headers",
			ar.body as "asset_response.body",
			ar.response_time as "asset_response.response_time",
			ar.created_at as "asset_response.created_at",
			a.id as "asset.id",
			a.program_id as "asset.program_id",
			a.program_url as "asset.program_url",
			a.url as "asset.url",
			a.domain as "asset.domain",
			a.subdomain as "asset.subdomain",
			a.ip as "asset.ip",
			a.status as "asset.status",
			a.source as "asset.source",
			a.created_at as "asset.created_at",
			a.updated_at as "asset.updated_at"
		FROM asset_responses ar
		JOIN assets a ON ar.asset_id = a.id
		ORDER BY ar.created_at DESC
		LIMIT $1
	`

	rows, err := r.db.QueryxContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get asset responses with asset info: %w", err)
	}
	defer rows.Close()

	var results []struct {
		AssetResponse *AssetResponse `db:"asset_response"`
		Asset         *Asset         `db:"asset"`
	}

	for rows.Next() {
		var result struct {
			AssetResponse AssetResponse `db:"asset_response"`
			Asset         Asset         `db:"asset"`
		}
		if err := rows.StructScan(&result); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		results = append(results, struct {
			AssetResponse *AssetResponse `db:"asset_response"`
			Asset         *Asset         `db:"asset"`
		}{
			AssetResponse: &result.AssetResponse,
			Asset:         &result.Asset,
		})
	}

	return results, nil
}

// Utility Methods

// GetAssetCountByProgramID gets the count of assets for a program
func (r *AssetRepository) GetAssetCountByProgramID(ctx context.Context, programID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM assets WHERE program_id = $1`

	err := r.db.GetContext(ctx, &count, query, programID)
	if err != nil {
		return 0, fmt.Errorf("failed to get asset count: %w", err)
	}

	return count, nil
}

// GetProgramsWithAssetCount gets programs with their asset counts
func (r *ProgramRepository) GetProgramsWithAssetCount(ctx context.Context) ([]struct {
	Program    *Program `db:"program"`
	AssetCount int      `db:"asset_count"`
}, error) {
	query := `
		SELECT p.*, COUNT(a.id) as asset_count
		FROM programs p
		LEFT JOIN assets a ON p.id = a.program_id
		WHERE p.is_active = true
		GROUP BY p.id
		ORDER BY asset_count DESC
	`

	rows, err := r.db.QueryxContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get programs with asset count: %w", err)
	}
	defer rows.Close()

	var results []struct {
		Program    *Program `db:"program"`
		AssetCount int      `db:"asset_count"`
	}

	for rows.Next() {
		var result struct {
			Program    Program `db:"program"`
			AssetCount int     `db:"asset_count"`
		}
		if err := rows.StructScan(&result); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		results = append(results, struct {
			Program    *Program `db:"program"`
			AssetCount int      `db:"asset_count"`
		}{
			Program:    &result.Program,
			AssetCount: result.AssetCount,
		})
	}

	return results, nil
}
