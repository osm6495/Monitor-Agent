package hackerone

import (
	"time"
)

// HackerOneProgram represents a HackerOne program
type HackerOneProgram struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Attributes ProgramAttributes `json:"attributes"`
	Links      ProgramLinks      `json:"links"`
}

// ProgramAttributes contains program details
type ProgramAttributes struct {
	Name                         string    `json:"name"`
	Handle                       string    `json:"handle"`
	URL                          string    `json:"url"`
	Website                      string    `json:"website"`
	SubmissionState              string    `json:"submission_state"`
	State                        string    `json:"state"`
	OffersBounties               bool      `json:"offers_bounties"`
	OffersSwag                   bool      `json:"offers_swag"`
	AllowsDisclosure             bool      `json:"allows_disclosure"`
	AllowsPrivateDisclosure      bool      `json:"allows_private_disclosure"`
	ResponseEfficiencyPercentage int       `json:"response_efficiency_percentage"`
	CreatedAt                    time.Time `json:"created_at"`
	UpdatedAt                    time.Time `json:"updated_at"`
}

// ProgramLinks contains program links
type ProgramLinks struct {
	Self string `json:"self"`
	Web  string `json:"web"`
}

// HackerOneScope represents the scope of a HackerOne program
type HackerOneScope struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Attributes ScopeAttributes `json:"attributes"`
}

// ScopeAttributes contains scope details
type ScopeAttributes struct {
	AssetIdentifier       string    `json:"asset_identifier"`
	AssetType             string    `json:"asset_type"`
	EligibleForBounty     bool      `json:"eligible_for_bounty"`
	EligibleForSubmission bool      `json:"eligible_for_submission"`
	Instruction           string    `json:"instruction"`
	Reference             string    `json:"reference"`
	Confidentiality       string    `json:"confidentiality"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// HackerOneResponse represents a generic HackerOne API response
type HackerOneResponse struct {
	Data  []HackerOneProgram `json:"data"`
	Links ResponseLinks      `json:"links"`
	Meta  ResponseMeta       `json:"meta"`
}

// ResponseLinks contains pagination links
type ResponseLinks struct {
	First string `json:"first"`
	Last  string `json:"last"`
	Next  string `json:"next"`
	Prev  string `json:"prev"`
	Self  string `json:"self"`
}

// ResponseMeta contains response metadata
type ResponseMeta struct {
	TotalCount int `json:"total_count"`
	PageCount  int `json:"page_count"`
	PageSize   int `json:"page_size"`
	Page       int `json:"page"`
}

// ScopeResponse represents a HackerOne scope API response
type ScopeResponse struct {
	Data  []HackerOneScope `json:"data"`
	Links ResponseLinks    `json:"links"`
	Meta  ResponseMeta     `json:"meta"`
}

// ErrorResponse represents a HackerOne API error
type ErrorResponse struct {
	Errors []ErrorDetail `json:"errors"`
}

// ErrorDetail contains error details
type ErrorDetail struct {
	Status string      `json:"status"`
	Title  string      `json:"title"`
	Detail string      `json:"detail"`
	Source ErrorSource `json:"source"`
}

// ErrorSource contains error source information
type ErrorSource struct {
	Pointer string `json:"pointer"`
}
