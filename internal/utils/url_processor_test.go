package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestURLProcessor_ExtractDomain(t *testing.T) {
	processor := NewURLProcessor()

	tests := []struct {
		name     string
		url      string
		expected string
		wantErr  bool
	}{
		{
			name:     "simple domain",
			url:      "https://example.com",
			expected: "example.com",
			wantErr:  false,
		},
		{
			name:     "subdomain",
			url:      "https://subdomain.example.com",
			expected: "subdomain.example.com",
			wantErr:  false,
		},
		{
			name:     "with path",
			url:      "https://example.com/path",
			expected: "example.com",
			wantErr:  false,
		},
		{
			name:     "with port",
			url:      "https://example.com:8080",
			expected: "example.com",
			wantErr:  false,
		},
		{
			name:     "wildcard domain",
			url:      "*.example.com",
			expected: "example.com",
			wantErr:  false,
		},
		{
			name:     "IP address",
			url:      "192.168.1.1",
			expected: "192.168.1.1",
			wantErr:  false,
		},
		{
			name:     "domain without protocol",
			url:      "example.com",
			expected: "example.com",
			wantErr:  false,
		},
		{
			name:     "subdomain without protocol",
			url:      "subdomain.example.com",
			expected: "subdomain.example.com",
			wantErr:  false,
		},
		{
			name:     "domain with path without protocol",
			url:      "example.com/path",
			expected: "example.com",
			wantErr:  false,
		},
		{
			name:     "domain with port without protocol",
			url:      "example.com:8080",
			expected: "example.com",
			wantErr:  false,
		},
		{
			name:     "empty URL",
			url:      "",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "whitespace only URL",
			url:      "   ",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "protocol only URL",
			url:      "https://",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "http protocol only URL",
			url:      "http://",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.ExtractDomain(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestURLProcessor_ExtractSubdomain(t *testing.T) {
	processor := NewURLProcessor()

	tests := []struct {
		name     string
		url      string
		expected string
		wantErr  bool
	}{
		{
			name:     "no subdomain",
			url:      "https://example.com",
			expected: "",
			wantErr:  false,
		},
		{
			name:     "with subdomain",
			url:      "https://subdomain.example.com",
			expected: "subdomain",
			wantErr:  false,
		},
		{
			name:     "multiple subdomains",
			url:      "https://api.staging.example.com",
			expected: "api",
			wantErr:  false,
		},
		{
			name:     "wildcard domain",
			url:      "*.example.com",
			expected: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.ExtractSubdomain(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestURLProcessor_ConvertWildcardToDomain(t *testing.T) {
	processor := NewURLProcessor()

	tests := []struct {
		name     string
		wildcard string
		expected string
	}{
		{
			name:     "simple wildcard",
			wildcard: "*.example.com",
			expected: "example.com",
		},
		{
			name:     "complex wildcard",
			wildcard: "*.subdomain.example.com",
			expected: "subdomain.example.com",
		},
		{
			name:     "multiple wildcards",
			wildcard: "*.*.example.com",
			expected: "example.com",
		},
		{
			name:     "multiple wildcards with subdomain",
			wildcard: "*.*.subdomain.example.com",
			expected: "subdomain.example.com",
		},
		{
			name:     "no wildcard",
			wildcard: "example.com",
			expected: "example.com",
		},
		{
			name:     "empty string",
			wildcard: "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.ConvertWildcardToDomain(tt.wildcard)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestURLProcessor_NormalizeURL(t *testing.T) {
	processor := NewURLProcessor()

	tests := []struct {
		name     string
		url      string
		expected string
		wantErr  bool
	}{
		{
			name:     "add https",
			url:      "example.com",
			expected: "https://example.com",
			wantErr:  false,
		},
		{
			name:     "change http to https",
			url:      "http://example.com",
			expected: "https://example.com",
			wantErr:  false,
		},
		{
			name:     "remove default port",
			url:      "https://example.com:443",
			expected: "https://example.com",
			wantErr:  false,
		},
		{
			name:     "remove trailing slash",
			url:      "https://example.com/",
			expected: "https://example.com",
			wantErr:  false,
		},
		{
			name:     "complex url",
			url:      "http://example.com:80/path/",
			expected: "https://example.com/path",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.NormalizeURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestURLProcessor_IsValidURL(t *testing.T) {
	processor := NewURLProcessor()

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "valid url",
			url:      "https://example.com",
			expected: true,
		},
		{
			name:     "valid url with path",
			url:      "https://example.com/path",
			expected: true,
		},
		{
			name:     "invalid url",
			url:      "not-a-url",
			expected: true,
		},
		{
			name:     "empty string",
			url:      "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.IsValidURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestURLProcessor_IsWildcardDomain(t *testing.T) {
	processor := NewURLProcessor()

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "wildcard domain",
			url:      "*.example.com",
			expected: true,
		},
		{
			name:     "not wildcard domain",
			url:      "example.com",
			expected: false,
		},
		{
			name:     "empty string",
			url:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.IsWildcardDomain(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestURLProcessor_IsIPAddress(t *testing.T) {
	processor := NewURLProcessor()

	tests := []struct {
		name     string
		hostname string
		expected bool
	}{
		{
			name:     "valid ipv4",
			hostname: "192.168.1.1",
			expected: true,
		},
		{
			name:     "invalid ipv4",
			hostname: "192.168.1.256",
			expected: false,
		},
		{
			name:     "domain name",
			hostname: "example.com",
			expected: false,
		},
		{
			name:     "empty string",
			hostname: "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.IsIPAddress(tt.hostname)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestURLProcessor_IsValidDomain(t *testing.T) {
	processor := NewURLProcessor()

	tests := []struct {
		name     string
		domain   string
		expected bool
	}{
		{
			name:     "valid domain",
			domain:   "example.com",
			expected: true,
		},
		{
			name:     "valid subdomain",
			domain:   "subdomain.example.com",
			expected: true,
		},
		{
			name:     "valid wildcard domain",
			domain:   "*.example.com",
			expected: true,
		},
		{
			name:     "valid domain with hyphen",
			domain:   "test-domain.example.com",
			expected: true,
		},
		{
			name:     "valid IP address",
			domain:   "192.168.1.1",
			expected: true,
		},
		{
			name:     "empty string",
			domain:   "",
			expected: false,
		},
		{
			name:     "just wildcard",
			domain:   "*",
			expected: false,
		},
		{
			name:     "hash-like string",
			domain:   "0027ccb97c839fec02edebe904d50ff8",
			expected: false,
		},
		{
			name:     "malformed string",
			domain:   "57777hh21124156674c00b9",
			expected: false,
		},
		{
			name:     "just numbers",
			domain:   "123456789",
			expected: false,
		},
		{
			name:     "too short",
			domain:   "ab",
			expected: false,
		},
		{
			name:     "no dots",
			domain:   "example",
			expected: false,
		},
		{
			name:     "starts with dot",
			domain:   ".example.com",
			expected: false,
		},
		{
			name:     "ends with dot",
			domain:   "example.com.",
			expected: false,
		},
		{
			name:     "starts with hyphen",
			domain:   "-example.com",
			expected: false,
		},
		{
			name:     "ends with hyphen",
			domain:   "example.com-",
			expected: false,
		},
		{
			name:     "invalid characters",
			domain:   "example@.com",
			expected: false,
		},
		{
			name:     "label too long",
			domain:   "a" + strings.Repeat("b", 63) + ".com",
			expected: false,
		},
		{
			name:     "label starts with hyphen",
			domain:   "example.-test.com",
			expected: false,
		},
		{
			name:     "label ends with hyphen",
			domain:   "example.test-.com",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.IsValidDomain(tt.domain)
			assert.Equal(t, tt.expected, result)
		})
	}
}
