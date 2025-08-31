package chaosdb

import (
	"time"
)

// ChaosDBResponse represents a ChaosDB API response
type ChaosDBResponse struct {
	Domain     string   `json:"domain"`
	Subdomains []string `json:"subdomains"`
	Count      int      `json:"count"`
	Error      string   `json:"error,omitempty"`
}

// ChaosDBError represents a ChaosDB API error
type ChaosDBError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// DiscoveryResult represents a discovery result
type DiscoveryResult struct {
	Domain       string    `json:"domain"`
	Subdomains   []string  `json:"subdomains"`
	Count        int       `json:"count"`
	DiscoveredAt time.Time `json:"discovered_at"`
	Error        string    `json:"error,omitempty"`
}

// BulkDiscoveryResult represents a bulk discovery result
type BulkDiscoveryResult struct {
	Results      []DiscoveryResult `json:"results"`
	TotalCount   int               `json:"total_count"`
	ErrorCount   int               `json:"error_count"`
	DiscoveredAt time.Time         `json:"discovered_at"`
}
