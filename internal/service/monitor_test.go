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
