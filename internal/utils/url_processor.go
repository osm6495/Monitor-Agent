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
	// Handle wildcard domains
	if strings.HasPrefix(urlStr, "*.") {
		return strings.TrimPrefix(urlStr, "*."), nil
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

// extractDomainManually extracts domain from a string when URL parsing fails
func (up *URLProcessor) extractDomainManually(urlStr string) string {
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
