package database

import (
	"time"

	"github.com/google/uuid"
)

// Program represents a bug bounty program
type Program struct {
	ID          uuid.UUID `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Platform    string    `db:"platform" json:"platform"`
	URL         string    `db:"url" json:"url"`
	ProgramURL  string    `db:"program_url" json:"program_url"`
	IsActive    bool      `db:"is_active" json:"is_active"`
	LastUpdated time.Time `db:"last_updated" json:"last_updated"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// Asset represents a discovered asset (subdomain/URL)
type Asset struct {
	ID         uuid.UUID `db:"id" json:"id"`
	ProgramID  uuid.UUID `db:"program_id" json:"program_id"`
	ProgramURL string    `db:"program_url" json:"program_url"`
	URL        string    `db:"url" json:"url"`
	Domain     string    `db:"domain" json:"domain"`
	Subdomain  string    `db:"subdomain" json:"subdomain"`
	IP         string    `db:"ip" json:"ip"`
	Status     string    `db:"status" json:"status"` // active, inactive, etc.
	Source     string    `db:"source" json:"source"` // chaosdb, direct, etc.
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}

// PlatformEntity represents a bug bounty platform entity in the database
type PlatformEntity struct {
	ID          uuid.UUID `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	APIEndpoint string    `db:"api_endpoint" json:"api_endpoint"`
	IsActive    bool      `db:"is_active" json:"is_active"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// Scan represents a discovery scan session
type Scan struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	ProgramID   uuid.UUID  `db:"program_id" json:"program_id"`
	Status      string     `db:"status" json:"status"` // running, completed, failed
	AssetsFound int        `db:"assets_found" json:"assets_found"`
	StartedAt   time.Time  `db:"started_at" json:"started_at"`
	CompletedAt *time.Time `db:"completed_at" json:"completed_at"`
	Error       string     `db:"error" json:"error"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
}

// Table names
const (
	TablePrograms  = "programs"
	TableAssets    = "assets"
	TablePlatforms = "platforms"
	TableScans     = "scans"
)
