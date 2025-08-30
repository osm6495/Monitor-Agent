package utils

import (
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
