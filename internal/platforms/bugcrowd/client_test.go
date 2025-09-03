package bugcrowd

import (
	"testing"

	"github.com/monitor-agent/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestClient_parseScopeAsset_URLNormalization(t *testing.T) {
	client := &Client{
		urlProcessor: utils.NewURLProcessor(),
	}

	tests := []struct {
		name        string
		targetType  string
		target      string
		expectedURL string
	}{
		{
			name:        "domain without protocol gets https://",
			targetType:  "website",
			target:      "example.com",
			expectedURL: "https://example.com",
		},
		{
			name:        "domain with https:// stays unchanged",
			targetType:  "website",
			target:      "https://example.com",
			expectedURL: "https://example.com",
		},
		{
			name:        "domain with http:// gets converted to https://",
			targetType:  "website",
			target:      "http://example.com",
			expectedURL: "https://example.com",
		},
		{
			name:        "wildcard domain gets normalized",
			targetType:  "wildcard",
			target:      "*.example.com",
			expectedURL: "https://example.com",
		},
		{
			name:        "subdomain without protocol gets https://",
			targetType:  "website",
			target:      "subdomain.example.com",
			expectedURL: "https://subdomain.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := BugCrowdScope{
				Type:   tt.targetType,
				Target: tt.target,
			}

			asset := client.parseScopeAsset(target)
			assert.NotNil(t, asset)
			assert.Equal(t, tt.expectedURL, asset.URL)
		})
	}
}
