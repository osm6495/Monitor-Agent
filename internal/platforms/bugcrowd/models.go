package bugcrowd

import (
	"time"
)

// BugCrowdProgram represents a BugCrowd program
type BugCrowdProgram struct {
	UUID      string    `json:"uuid"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	URL       string    `json:"url"`
	MinReward int       `json:"min_reward"`
	MaxReward int       `json:"max_reward"`
	Currency  string    `json:"currency"`
	Status    string    `json:"status"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BugCrowdScope represents the scope of a BugCrowd program
type BugCrowdScope struct {
	UUID       string    `json:"uuid"`
	Target     string    `json:"target"`
	Type       string    `json:"type"`
	Eligible   bool      `json:"eligible"`
	Ineligible bool      `json:"ineligible"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// BugCrowdResponse represents a generic BugCrowd API response
type BugCrowdResponse struct {
	Programs []BugCrowdProgram `json:"programs"`
	Meta     ResponseMeta      `json:"meta"`
}

// ResponseMeta contains response metadata
type ResponseMeta struct {
	TotalCount int `json:"total_count"`
	PageCount  int `json:"page_count"`
	PageSize   int `json:"page_size"`
	Page       int `json:"page"`
}

// ScopeResponse represents a BugCrowd scope API response
type ScopeResponse struct {
	Targets []BugCrowdScope `json:"targets"`
	Meta    ResponseMeta    `json:"meta"`
}

// BugCrowdError represents a BugCrowd API error
type BugCrowdError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}
