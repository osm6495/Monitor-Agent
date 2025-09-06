package utils

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// URLProcessor provides URL processing utilities
type URLProcessor struct{}

// NewURLProcessor creates a new URL processor
func NewURLProcessor() *URLProcessor {
	return &URLProcessor{}
}

// ExtractDomain extracts the domain from a URL
func (up *URLProcessor) ExtractDomain(urlStr string) (string, error) {
	// Handle empty or whitespace-only URLs
	if strings.TrimSpace(urlStr) == "" {
		return "", fmt.Errorf("empty URL provided")
	}

	// Handle URLs that are just protocol without hostname
	if strings.TrimSpace(urlStr) == "http://" || strings.TrimSpace(urlStr) == "https://" {
		return "", fmt.Errorf("no hostname found in URL: %s", urlStr)
	}

	// Handle wildcard domains
	if strings.HasPrefix(urlStr, "*.") {
		return strings.TrimPrefix(urlStr, "*."), nil
	}

	// Handle domain names without protocol (common in bug bounty scopes)
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") && !strings.HasPrefix(urlStr, "//") {
		// This looks like a domain name without protocol, extract it manually
		return up.extractDomainManually(urlStr), nil
	}

	// Parse URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		// If parsing fails, try to extract domain manually
		return up.extractDomainManually(urlStr), nil
	}

	// Handle IP addresses
	if up.IsIPAddress(parsedURL.Hostname()) {
		return parsedURL.Hostname(), nil
	}

	// Extract domain from hostname
	hostname := parsedURL.Hostname()
	if hostname == "" {
		return "", fmt.Errorf("no hostname found in URL: %s", urlStr)
	}

	return hostname, nil
}

// ExtractSubdomain extracts the subdomain from a URL
func (up *URLProcessor) ExtractSubdomain(urlStr string) (string, error) {
	_, err := up.ExtractDomain(urlStr)
	if err != nil {
		return "", err
	}

	// Handle wildcard domains
	if strings.HasPrefix(urlStr, "*.") {
		return "", nil // Wildcard doesn't have a specific subdomain
	}

	// Parse URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return up.extractSubdomainManually(urlStr), nil
	}

	hostname := parsedURL.Hostname()
	if hostname == "" {
		return "", nil
	}

	// Split hostname by dots
	parts := strings.Split(hostname, ".")
	if len(parts) <= 2 {
		return "", nil // No subdomain
	}

	// Return the subdomain part
	return parts[0], nil
}

// ConvertWildcardToDomain converts a wildcard domain to its base domain
func (up *URLProcessor) ConvertWildcardToDomain(wildcard string) string {
	// Remove ALL wildcard prefixes and return the base domain
	cleaned := wildcard

	// Keep removing *. prefixes until none remain
	for strings.HasPrefix(cleaned, "*.") {
		cleaned = strings.TrimPrefix(cleaned, "*.")
	}

	// Remove any trailing dots
	cleaned = strings.Trim(cleaned, ".")

	return cleaned
}

// NormalizeURL normalizes a URL to a standard format
func (up *URLProcessor) NormalizeURL(urlStr string) (string, error) {
	// Add protocol if missing
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		urlStr = "https://" + urlStr
	}

	// Parse URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}

	// Normalize scheme to HTTPS
	parsedURL.Scheme = "https"

	// Remove default ports
	if parsedURL.Port() == "443" || parsedURL.Port() == "80" {
		parsedURL.Host = parsedURL.Hostname()
	}

	// Remove trailing slash from path
	parsedURL.Path = strings.TrimSuffix(parsedURL.Path, "/")

	// Remove empty query parameters
	if parsedURL.RawQuery == "" {
		parsedURL.RawQuery = ""
	}

	return parsedURL.String(), nil
}

// IsValidURL checks if a URL is valid
func (up *URLProcessor) IsValidURL(urlStr string) bool {
	_, err := url.Parse(urlStr)
	return err == nil
}

// IsWildcardDomain checks if a URL is a wildcard domain
func (up *URLProcessor) IsWildcardDomain(urlStr string) bool {
	return strings.HasPrefix(urlStr, "*.")
}

// IsIPAddress checks if a string is an IP address
func (up *URLProcessor) IsIPAddress(hostname string) bool {
	ipRegex := regexp.MustCompile(`^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`)
	return ipRegex.MatchString(hostname)
}

// IsValidDomain checks if a string is a valid domain name
func (up *URLProcessor) IsValidDomain(domain string) bool {
	// Handle empty or whitespace-only domains
	if strings.TrimSpace(domain) == "" {
		return false
	}

	// Remove any wildcard prefixes for validation
	cleanDomain := strings.TrimPrefix(domain, "*.")
	cleanDomain = strings.TrimSpace(cleanDomain)

	// Check if it's just a wildcard without domain
	if cleanDomain == "" || cleanDomain == "*" {
		return false
	}

	// Check for obviously invalid patterns
	// Hash-like strings (32+ character hex strings)
	if len(cleanDomain) >= 32 && regexp.MustCompile(`^[a-f0-9]+$`).MatchString(cleanDomain) {
		return false
	}

	// Check for malformed strings with numbers and letters mixed without dots
	if !strings.Contains(cleanDomain, ".") && regexp.MustCompile(`^[a-zA-Z0-9]+$`).MatchString(cleanDomain) && len(cleanDomain) > 20 {
		return false
	}

	// Check for strings that are just numbers
	if regexp.MustCompile(`^[0-9]+$`).MatchString(cleanDomain) {
		return false
	}

	// Check for strings that are too short to be valid domains
	if len(cleanDomain) < 3 {
		return false
	}

	// Check for strings that are too long to be valid domains
	if len(cleanDomain) > 253 {
		return false
	}

	// Check for invalid characters (domains should only contain letters, numbers, dots, and hyphens)
	if !regexp.MustCompile(`^[a-zA-Z0-9.-]+$`).MatchString(cleanDomain) {
		return false
	}

	// Check that it doesn't start or end with a dot or hyphen
	if strings.HasPrefix(cleanDomain, ".") || strings.HasSuffix(cleanDomain, ".") ||
		strings.HasPrefix(cleanDomain, "-") || strings.HasSuffix(cleanDomain, "-") {
		return false
	}

	// Check that it has at least one dot (for TLD)
	if !strings.Contains(cleanDomain, ".") {
		return false
	}

	// Check that each label (between dots) is valid
	labels := strings.Split(cleanDomain, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		// Each label should only contain letters, numbers, and hyphens
		if !regexp.MustCompile(`^[a-zA-Z0-9-]+$`).MatchString(label) {
			return false
		}
		// Labels shouldn't start or end with hyphens
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
	}

	return true
}

// extractDomainManually extracts domain from a string when URL parsing fails
func (up *URLProcessor) extractDomainManually(urlStr string) string {
	// Remove protocol if present
	urlStr = strings.TrimPrefix(urlStr, "http://")
	urlStr = strings.TrimPrefix(urlStr, "https://")
	urlStr = strings.TrimPrefix(urlStr, "//")

	// Remove path and query parameters
	if idx := strings.Index(urlStr, "/"); idx != -1 {
		urlStr = urlStr[:idx]
	}

	// Remove port if present
	if idx := strings.Index(urlStr, ":"); idx != -1 {
		urlStr = urlStr[:idx]
	}

	// Remove any leading/trailing whitespace
	urlStr = strings.TrimSpace(urlStr)

	// Return empty string if nothing left after processing
	if urlStr == "" {
		return ""
	}

	return urlStr
}

// extractSubdomainManually extracts subdomain from a string when URL parsing fails
func (up *URLProcessor) extractSubdomainManually(urlStr string) string {
	// Remove protocol if present
	urlStr = strings.TrimPrefix(urlStr, "http://")
	urlStr = strings.TrimPrefix(urlStr, "https://")

	// Remove path and query parameters
	if idx := strings.Index(urlStr, "/"); idx != -1 {
		urlStr = urlStr[:idx]
	}

	// Remove port if present
	if idx := strings.Index(urlStr, ":"); idx != -1 {
		urlStr = urlStr[:idx]
	}

	// Split by dots
	parts := strings.Split(urlStr, ".")
	if len(parts) <= 2 {
		return "" // No subdomain
	}

	return parts[0]
}

// GetCommonSubdomains returns a list of common subdomains to test
func (up *URLProcessor) GetCommonSubdomains() []string {
	return []string{
		"www", "api", "admin", "app", "dev", "staging", "test", "mail", "ftp", "blog",
		"support", "help", "docs", "cdn", "static", "assets", "img", "images", "media",
		"shop", "store", "secure", "login", "auth", "dashboard", "portal", "console",
		"manage", "control", "panel", "webmail", "smtp", "pop", "imap", "ns1", "ns2",
		"mx1", "mx2", "dns1", "dns2", "gateway", "router", "firewall", "proxy",
		"vpn", "remote", "ssh", "telnet", "rdp", "vnc", "monitor", "nagios", "zabbix",
		"jenkins", "gitlab", "github", "bitbucket", "jira", "confluence", "redmine",
		"wordpress", "joomla", "drupal", "magento", "shopify", "woocommerce",
		"analytics", "stats", "metrics", "logging", "elk", "kibana", "grafana",
		"prometheus", "alertmanager", "consul", "etcd", "redis", "mysql", "postgres",
		"mongodb", "elasticsearch", "rabbitmq", "kafka", "zookeeper", "etcd",
	}
}
