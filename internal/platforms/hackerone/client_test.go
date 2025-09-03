package hackerone

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
		name            string
		assetType       string
		assetIdentifier string
		expectedURL     string
	}{
		{
			name:            "domain without protocol gets https://",
			assetType:       "URL",
			assetIdentifier: "example.com",
			expectedURL:     "https://example.com",
		},
		{
			name:            "domain with https:// stays unchanged",
			assetType:       "URL",
			assetIdentifier: "https://example.com",
			expectedURL:     "https://example.com",
		},
		{
			name:            "domain with http:// gets converted to https://",
			assetType:       "URL",
			assetIdentifier: "http://example.com",
			expectedURL:     "https://example.com",
		},
		{
			name:            "wildcard domain gets normalized",
			assetType:       "WILDCARD",
			assetIdentifier: "*.example.com",
			expectedURL:     "https://example.com",
		},
		{
			name:            "subdomain without protocol gets https://",
			assetType:       "URL",
			assetIdentifier: "subdomain.example.com",
			expectedURL:     "https://subdomain.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attr := ScopeAttributes{
				AssetType:       tt.assetType,
				AssetIdentifier: tt.assetIdentifier,
			}

			asset := client.parseScopeAsset(attr)
			assert.NotNil(t, asset)
			assert.Equal(t, tt.expectedURL, asset.URL)
		})
	}
}
