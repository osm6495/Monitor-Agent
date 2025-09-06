package service

import (
	"testing"

	"github.com/monitor-agent/internal/platforms"
	"github.com/monitor-agent/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestMonitorService_ExtractUniqueDomains(t *testing.T) {
	service := &MonitorService{
		urlProcessor: utils.NewURLProcessor(),
	}

	// Test with mixed asset types including wildcards
	scopeAssets := []*platforms.ScopeAsset{
		{
			URL:    "https://example.com",
			Domain: "example.com",
			Type:   "url",
		},
		{
			URL:    "https://subdomain.example.com",
			Domain: "subdomain.example.com",
			Type:   "url",
		},
		{
			URL:    "https://wildcard.com", // This was originally *.wildcard.com but got normalized
			Domain: "wildcard.com",
			Type:   "wildcard",
		},
		{
			URL:    "https://another.com", // This was originally *.another.com but got normalized
			Domain: "another.com",
			Type:   "wildcard",
		},
		{
			URL:    "192.168.1.1",
			Domain: "192.168.1.1",
			Type:   "ip",
		},
	}

	domains := service.extractUniqueDomains(scopeAssets)

	// Should extract 4 domains: example.com, subdomain.example.com, wildcard.com, another.com
	expectedDomains := []string{
		"example.com",
		"subdomain.example.com",
		"wildcard.com",
		"another.com",
	}

	assert.Len(t, domains, 4)
	for _, expectedDomain := range expectedDomains {
		assert.Contains(t, domains, expectedDomain, "Expected domain %s not found", expectedDomain)
	}

	// Verify that wildcard domains are properly stripped to base domains
	assert.Contains(t, domains, "wildcard.com")
	assert.Contains(t, domains, "another.com")
}

func TestMonitorService_WildcardSubdomainFiltering(t *testing.T) {
	service := &MonitorService{
		urlProcessor: utils.NewURLProcessor(),
	}

	// Test data: ChaosDB discovered subdomains including wildcards
	discoveredSubdomains := []string{
		"api.slackhq.com",
		"*.test.slackhq.com", // Wildcard subdomain
		"www.slackhq.com",
		"*.staging.slackhq.com", // Another wildcard subdomain
		"mail.slackhq.com",
	}

	// Filter out wildcard subdomains (this simulates what happens in processSingleDomain)
	var cleanSubdomains []string
	for _, subdomain := range discoveredSubdomains {
		cleanSubdomain := service.urlProcessor.ConvertWildcardToDomain(subdomain)
		if cleanSubdomain != "" {
			cleanSubdomains = append(cleanSubdomains, cleanSubdomain)
		}
	}

	// Should have 5 clean subdomains after filtering
	expectedCleanSubdomains := []string{
		"api.slackhq.com",
		"test.slackhq.com", // Wildcard stripped
		"www.slackhq.com",
		"staging.slackhq.com", // Wildcard stripped
		"mail.slackhq.com",
	}

	assert.Len(t, cleanSubdomains, 5)
	for _, expectedSubdomain := range expectedCleanSubdomains {
		assert.Contains(t, cleanSubdomains, expectedSubdomain, "Expected clean subdomain %s not found", expectedSubdomain)
	}

	// Verify that wildcard prefixes were properly removed
	assert.Contains(t, cleanSubdomains, "test.slackhq.com")
	assert.Contains(t, cleanSubdomains, "staging.slackhq.com")
	assert.NotContains(t, cleanSubdomains, "*.test.slackhq.com")
	assert.NotContains(t, cleanSubdomains, "*.staging.slackhq.com")
}

func TestMonitorService_filterOutOfScopeSubdomains(t *testing.T) {
	service := &MonitorService{
		urlProcessor: utils.NewURLProcessor(),
	}

	// Test subdomains that should be filtered
	subdomains := []string{
		"api.example.com",
		"test.test.sub.domain.com",
		"test.sub.domain.com",
		"allowed.example.com",
		"forbidden.example.com",
		"nested.forbidden.example.com",
	}

	// Out-of-scope assets
	outOfScopeAssets := []*platforms.ScopeAsset{
		{
			URL:                   "https://forbidden.example.com",
			Domain:                "forbidden.example.com",
			Type:                  "url",
			EligibleForSubmission: false,
		},
		{
			URL:                   "https://sub.domain.com",
			Domain:                "sub.domain.com",
			Type:                  "wildcard",
			EligibleForSubmission: false,
			OriginalPattern:       "*.sub.domain.com",
		},
	}

	filtered := service.filterOutOfScopeSubdomains(subdomains, outOfScopeAssets)

	// Should filter out:
	// - forbidden.example.com (exact match)
	// - nested.forbidden.example.com (subdomain of forbidden.example.com)
	// - test.test.sub.domain.com (matches *.sub.domain.com wildcard)
	// - test.sub.domain.com (matches *.sub.domain.com wildcard)
	// Should keep:
	// - api.example.com (no match)
	// - allowed.example.com (no match)

	expected := []string{
		"api.example.com",
		"allowed.example.com",
	}

	assert.ElementsMatch(t, expected, filtered)
}

func TestMonitorService_matchesOutOfScopeAsset(t *testing.T) {
	service := &MonitorService{
		urlProcessor: utils.NewURLProcessor(),
	}

	tests := []struct {
		name            string
		subdomainURL    string
		outOfScopeAsset *platforms.ScopeAsset
		expected        bool
	}{
		{
			name:         "URL asset - exact match",
			subdomainURL: "https://forbidden.example.com",
			outOfScopeAsset: &platforms.ScopeAsset{
				URL:                   "https://forbidden.example.com",
				Domain:                "forbidden.example.com",
				Type:                  "url",
				EligibleForSubmission: false,
			},
			expected: true,
		},
		{
			name:         "URL asset - subdomain match",
			subdomainURL: "https://nested.forbidden.example.com",
			outOfScopeAsset: &platforms.ScopeAsset{
				URL:                   "https://forbidden.example.com",
				Domain:                "forbidden.example.com",
				Type:                  "url",
				EligibleForSubmission: false,
			},
			expected: true,
		},
		{
			name:         "URL asset - no match",
			subdomainURL: "https://allowed.example.com",
			outOfScopeAsset: &platforms.ScopeAsset{
				URL:                   "https://forbidden.example.com",
				Domain:                "forbidden.example.com",
				Type:                  "url",
				EligibleForSubmission: false,
			},
			expected: false,
		},
		{
			name:         "Wildcard asset - nested subdomain match",
			subdomainURL: "https://test.test.sub.domain.com",
			outOfScopeAsset: &platforms.ScopeAsset{
				URL:                   "https://sub.domain.com",
				Domain:                "sub.domain.com",
				Type:                  "wildcard",
				EligibleForSubmission: false,
				OriginalPattern:       "*.sub.domain.com",
			},
			expected: true,
		},
		{
			name:         "Wildcard asset - direct subdomain match",
			subdomainURL: "https://test.sub.domain.com",
			outOfScopeAsset: &platforms.ScopeAsset{
				URL:                   "https://sub.domain.com",
				Domain:                "sub.domain.com",
				Type:                  "wildcard",
				EligibleForSubmission: false,
				OriginalPattern:       "*.sub.domain.com",
			},
			expected: true,
		},
		{
			name:         "Wildcard asset - no match",
			subdomainURL: "https://sub.domain.com",
			outOfScopeAsset: &platforms.ScopeAsset{
				URL:                   "https://sub.domain.com",
				Domain:                "sub.domain.com",
				Type:                  "wildcard",
				EligibleForSubmission: false,
				OriginalPattern:       "*.sub.domain.com",
			},
			expected: false,
		},
		{
			name:         "CIDR asset - should not filter",
			subdomainURL: "https://example.com",
			outOfScopeAsset: &platforms.ScopeAsset{
				URL:                   "192.168.1.0/24",
				Domain:                "192.168.1.0/24",
				Type:                  "cidr",
				EligibleForSubmission: false,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.matchesOutOfScopeAsset(tt.subdomainURL, tt.outOfScopeAsset)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMonitorService_OutOfScopeAssetIsolation(t *testing.T) {
	service := &MonitorService{
		urlProcessor: utils.NewURLProcessor(),
	}

	// Test that out-of-scope assets from one program don't affect another program
	// This test simulates the scenario where Program A has *.forbidden.com out-of-scope
	// and Program B has *.allowed.com out-of-scope, and they shouldn't interfere with each other

	// Program A out-of-scope assets
	programAOutOfScope := []*platforms.ScopeAsset{
		{
			URL:                   "https://forbidden.com",
			Domain:                "forbidden.com",
			Type:                  "wildcard",
			EligibleForSubmission: false,
			OriginalPattern:       "*.forbidden.com",
		},
	}

	// Program B out-of-scope assets
	programBOutOfScope := []*platforms.ScopeAsset{
		{
			URL:                   "https://allowed.com",
			Domain:                "allowed.com",
			Type:                  "wildcard",
			EligibleForSubmission: false,
			OriginalPattern:       "*.allowed.com",
		},
	}

	// Test subdomains that should be filtered differently for each program
	subdomains := []string{
		"api.forbidden.com", // Should be filtered by Program A, not Program B
		"test.allowed.com",  // Should be filtered by Program B, not Program A
		"api.example.com",   // Should not be filtered by either program
	}

	// Test Program A filtering
	programAFiltered := service.filterOutOfScopeSubdomains(subdomains, programAOutOfScope)
	expectedProgramA := []string{
		"test.allowed.com", // Not filtered by Program A
		"api.example.com",  // Not filtered by Program A
	}
	assert.ElementsMatch(t, expectedProgramA, programAFiltered, "Program A should only filter its own out-of-scope assets")

	// Test Program B filtering
	programBFiltered := service.filterOutOfScopeSubdomains(subdomains, programBOutOfScope)
	expectedProgramB := []string{
		"api.forbidden.com", // Not filtered by Program B
		"api.example.com",   // Not filtered by Program B
	}
	assert.ElementsMatch(t, expectedProgramB, programBFiltered, "Program B should only filter its own out-of-scope assets")

	// Test that both programs together filter correctly
	combinedOutOfScope := append(programAOutOfScope, programBOutOfScope...)
	combinedFiltered := service.filterOutOfScopeSubdomains(subdomains, combinedOutOfScope)
	expectedCombined := []string{
		"api.example.com", // Only this should remain after both programs filter
	}
	assert.ElementsMatch(t, expectedCombined, combinedFiltered, "Combined filtering should work correctly")
}
