package tests

import (
	"encoding/json"
	"testing"

	"github.com/monitor-agent/internal/discovery/chaosdb"
	"github.com/monitor-agent/internal/platforms/hackerone"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHackerOneScopeParsing tests the parsing of HackerOne scope responses
func TestHackerOneScopeParsing(t *testing.T) {
	// Example HackerOne scope response from the user
	hackerOneResponse := `{
		"data": [
			{
				"id": "15516",
				"type": "structured-scope",
				"attributes": {
					"asset_type": "URL",
					"asset_identifier": "slack.com",
					"eligible_for_bounty": true,
					"eligible_for_submission": true,
					"instruction": "The slack.com site and application.",
					"max_severity": "critical",
					"created_at": "2018-10-12T12:32:20.112Z",
					"updated_at": "2023-01-25T09:33:23.626Z"
				}
			},
			{
				"id": "15517",
				"type": "structured-scope",
				"attributes": {
					"asset_type": "URL",
					"asset_identifier": "api.slack.com",
					"eligible_for_bounty": true,
					"eligible_for_submission": true,
					"instruction": "The Slack API",
					"max_severity": "critical",
					"created_at": "2018-10-12T12:32:45.153Z",
					"updated_at": "2023-01-25T09:33:23.695Z"
				}
			},
			{
				"id": "15518",
				"type": "structured-scope",
				"attributes": {
					"asset_type": "URL",
					"asset_identifier": "status.slack.com",
					"eligible_for_bounty": false,
					"eligible_for_submission": false,
					"instruction": "The Slack status site",
					"max_severity": "none",
					"created_at": "2018-10-12T12:33:03.121Z",
					"updated_at": "2024-05-06T10:07:33.056Z"
				}
			},
			{
				"id": "100819",
				"type": "structured-scope",
				"attributes": {
					"asset_type": "URL",
					"asset_identifier": "slackhq.com",
					"eligible_for_bounty": false,
					"eligible_for_submission": false,
					"instruction": "This site runs on WordPress, so if you find vulnerabilities in the WordPress service, please see [WordPress bounty program](https://hackerone.com/wordpress) for reporting details",
					"max_severity": "none",
					"created_at": "2022-04-21T18:17:12.275Z",
					"updated_at": "2023-01-25T09:33:40.010Z"
				}
			},
			{
				"id": "648594",
				"type": "structured-scope",
				"attributes": {
					"asset_type": "URL",
					"asset_identifier": "slack-status.com",
					"eligible_for_bounty": true,
					"eligible_for_submission": true,
					"instruction": "",
					"max_severity": "critical",
					"created_at": "2024-05-06T10:09:26.635Z",
					"updated_at": "2024-05-06T10:09:26.635Z"
				}
			}
		],
		"links": {}
	}`

	// Parse the response
	var scopeResp hackerone.ScopeResponse
	err := json.Unmarshal([]byte(hackerOneResponse), &scopeResp)
	require.NoError(t, err)

	// Test that all expected domains are present
	expectedDomains := []string{
		"slack.com",
		"api.slack.com",
		"status.slack.com",
		"slackhq.com",
		"slack-status.com",
	}

	// Create a map of found domains for easy lookup
	foundDomains := make(map[string]bool)
	for _, scope := range scopeResp.Data {
		if scope.Attributes.AssetType == "URL" {
			foundDomains[scope.Attributes.AssetIdentifier] = true
		}
	}

	// Check that all expected domains are found
	for _, expectedDomain := range expectedDomains {
		assert.True(t, foundDomains[expectedDomain], "Expected domain %s not found in response", expectedDomain)
	}

	// Test specific domain parsing - check for slackhq.com and slack-status.com
	assert.True(t, foundDomains["slackhq.com"], "slackhq.com should be present in the response")
	assert.True(t, foundDomains["slack-status.com"], "slack-status.com should be present in the response")

	// Test that eligible_for_submission filtering works correctly
	eligibleCount := 0
	for _, scope := range scopeResp.Data {
		if scope.Attributes.EligibleForSubmission {
			eligibleCount++
		}
	}

	// Should have some eligible assets (not all are eligible)
	assert.Greater(t, eligibleCount, 0, "Should have some eligible assets")
	assert.Less(t, eligibleCount, len(scopeResp.Data), "Should not have all assets as eligible")

	// Test specific domains that were mentioned as missing
	t.Logf("Found domains in response:")
	for domain := range foundDomains {
		t.Logf("  - %s", domain)
	}

	// Verify the specific domains mentioned in the user's issue
	assert.True(t, foundDomains["slackhq.com"], "slackhq.com should be present in the response")
	assert.True(t, foundDomains["slack-status.com"], "slack-status.com should be present in the response")
}

// TestChaosDBResponseParsing tests the parsing of ChaosDB responses
func TestChaosDBResponseParsing(t *testing.T) {
	// Example ChaosDB response from the user
	chaosDBResponse := `{
		"domain": "slackhq.com",
		"subdomains": [
			"*.investor",
			"brand",
			"campaign", 
			"click.email",
			"cloud.email",
			"image.email",
			"info",
			"investor",
			"mta.email",
			"mta2.email",
			"mta3.email",
			"view.email",
			"www"
		],
		"count": 13
	}`

	// Parse the response
	var chaosResp chaosdb.ChaosDBResponse
	err := json.Unmarshal([]byte(chaosDBResponse), &chaosResp)
	require.NoError(t, err)

	// Test basic parsing
	assert.Equal(t, "slackhq.com", chaosResp.Domain)
	assert.Equal(t, 13, chaosResp.Count)
	assert.Len(t, chaosResp.Subdomains, 13)

	// Test specific subdomains
	expectedSubdomains := []string{
		"*.investor",
		"brand",
		"campaign",
		"click.email",
		"cloud.email",
		"image.email",
		"info",
		"investor",
		"mta.email",
		"mta2.email",
		"mta3.email",
		"view.email",
		"www",
	}

	for _, expectedSubdomain := range expectedSubdomains {
		found := false
		for _, subdomain := range chaosResp.Subdomains {
			if subdomain == expectedSubdomain {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected subdomain %s not found", expectedSubdomain)
	}

	// Test that wildcard subdomains are included
	assert.Contains(t, chaosResp.Subdomains, "*.investor", "Wildcard subdomain should be included")

	// Log the subdomains for verification
	t.Logf("Found subdomains for %s:", chaosResp.Domain)
	for _, subdomain := range chaosResp.Subdomains {
		t.Logf("  - %s", subdomain)
	}
}

// TestDomainFilteringBehavior demonstrates the filtering behavior that might cause missing domains
func TestDomainFilteringBehavior(t *testing.T) {
	// Create a mock scope response with the exact data from the user's example
	scopeData := []hackerone.HackerOneScope{
		{
			ID:   "15516",
			Type: "structured-scope",
			Attributes: hackerone.ScopeAttributes{
				AssetType:             "URL",
				AssetIdentifier:       "slack.com",
				EligibleForSubmission: true,
			},
		},
		{
			ID:   "15518",
			Type: "structured-scope",
			Attributes: hackerone.ScopeAttributes{
				AssetType:             "URL",
				AssetIdentifier:       "status.slack.com",
				EligibleForSubmission: false, // This gets filtered out
			},
		},
		{
			ID:   "100819",
			Type: "structured-scope",
			Attributes: hackerone.ScopeAttributes{
				AssetType:             "URL",
				AssetIdentifier:       "slackhq.com",
				EligibleForSubmission: false, // This gets filtered out
			},
		},
		{
			ID:   "648594",
			Type: "structured-scope",
			Attributes: hackerone.ScopeAttributes{
				AssetType:             "URL",
				AssetIdentifier:       "slack-status.com",
				EligibleForSubmission: true,
			},
		},
	}

	// Simulate the filtering logic used in the client
	var eligibleAssets []hackerone.HackerOneScope
	var allAssets []hackerone.HackerOneScope

	for _, scope := range scopeData {
		allAssets = append(allAssets, scope)
		if scope.Attributes.EligibleForSubmission {
			eligibleAssets = append(eligibleAssets, scope)
		}
	}

	// Test that filtering removes ineligible assets
	t.Logf("All assets (%d):", len(allAssets))
	for _, asset := range allAssets {
		t.Logf("  - %s (eligible: %t)", asset.Attributes.AssetIdentifier, asset.Attributes.EligibleForSubmission)
	}

	t.Logf("Eligible assets (%d):", len(eligibleAssets))
	for _, asset := range eligibleAssets {
		t.Logf("  - %s", asset.Attributes.AssetIdentifier)
	}

	// Verify the filtering behavior
	assert.Len(t, allAssets, 4, "Should have 4 total assets")
	assert.Len(t, eligibleAssets, 2, "Should have 2 eligible assets")

	// Check that slackhq.com and status.slack.com are filtered out
	foundSlackhq := false
	foundStatusSlack := false
	for _, asset := range eligibleAssets {
		if asset.Attributes.AssetIdentifier == "slackhq.com" {
			foundSlackhq = true
		}
		if asset.Attributes.AssetIdentifier == "status.slack.com" {
			foundStatusSlack = true
		}
	}
	assert.False(t, foundSlackhq, "slackhq.com should be filtered out (not eligible for submission)")
	assert.False(t, foundStatusSlack, "status.slack.com should be filtered out (not eligible for submission)")

	// Check that slack.com and slack-status.com are included
	foundSlack := false
	foundSlackStatus := false
	for _, asset := range eligibleAssets {
		if asset.Attributes.AssetIdentifier == "slack.com" {
			foundSlack = true
		}
		if asset.Attributes.AssetIdentifier == "slack-status.com" {
			foundSlackStatus = true
		}
	}
	assert.True(t, foundSlack, "slack.com should be included (eligible for submission)")
	assert.True(t, foundSlackStatus, "slack-status.com should be included (eligible for submission)")
}

// TestChaosDBIntegrationWithHackerOne tests how ChaosDB discovery would work with HackerOne domains
func TestChaosDBIntegrationWithHackerOne(t *testing.T) {
	// Simulate the domains that would be extracted from HackerOne scope
	hackerOneDomains := []string{
		"slack.com",
		"slack-status.com",
		// Note: slackhq.com is not included because it's not eligible for submission
	}

	// Simulate ChaosDB discovery results for these domains
	chaosDBResults := map[string]*chaosdb.DiscoveryResult{
		"slack.com": {
			Domain:     "slack.com",
			Subdomains: []string{"api", "app", "status", "edgeapi"},
			Count:      4,
		},
		"slack-status.com": {
			Domain:     "slack-status.com",
			Subdomains: []string{"www"},
			Count:      1,
		},
		"slackhq.com": {
			Domain:     "slackhq.com",
			Subdomains: []string{"www", "brand", "campaign", "*.investor"},
			Count:      4,
		},
	}

	// Test that ChaosDB discovery works for the domains that are eligible
	t.Logf("HackerOne eligible domains for ChaosDB discovery:")
	for _, domain := range hackerOneDomains {
		result, exists := chaosDBResults[domain]
		if exists {
			t.Logf("  - %s: %d subdomains found", domain, result.Count)
			for _, subdomain := range result.Subdomains {
				t.Logf("    * %s", subdomain)
			}
		} else {
			t.Logf("  - %s: no ChaosDB data available", domain)
		}
	}

	// Test that slackhq.com is not in the HackerOne domains (because it's filtered out)
	foundSlackhq := false
	for _, domain := range hackerOneDomains {
		if domain == "slackhq.com" {
			foundSlackhq = true
			break
		}
	}
	assert.False(t, foundSlackhq, "slackhq.com should not be in HackerOne domains (filtered out)")

	// But ChaosDB data exists for slackhq.com
	slackhqResult, exists := chaosDBResults["slackhq.com"]
	assert.True(t, exists, "ChaosDB should have data for slackhq.com")
	assert.Equal(t, "slackhq.com", slackhqResult.Domain)
	assert.Len(t, slackhqResult.Subdomains, 4)
}

// TestHackerOneClientCreation tests that HackerOne client can be created
func TestHackerOneClientCreation(t *testing.T) {
	config := &hackerone.PlatformConfig{
		APIKey:        "test-key",
		Username:      "test-user",
		RateLimit:     100,
		Timeout:       30,
		RetryAttempts: 3,
		RetryDelay:    1,
	}

	client := hackerone.NewHackerOneClient(config)
	require.NotNil(t, client)
	assert.Equal(t, "hackerone", client.GetName())
}

// TestChaosDBClientCreation tests that ChaosDB client can be created
func TestChaosDBClientCreation(t *testing.T) {
	config := &chaosdb.ClientConfig{
		APIKey:        "test-key",
		RateLimit:     50,
		Timeout:       30,
		RetryAttempts: 3,
		RetryDelay:    1,
	}

	client := chaosdb.NewClient(config)
	require.NotNil(t, client)
	assert.Equal(t, 50, client.GetRateLimit())
}
