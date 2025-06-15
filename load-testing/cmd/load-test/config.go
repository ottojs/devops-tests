package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Default Settings
const defaultTestSeconds time.Duration = 10
const defaultTestRate int = 150
const defaultTestTimeout time.Duration = 5 // seconds
const defaultWarmupDelay int = 15          // seconds

// Default connection pool settings
const defaultConnectionPoolMaxConns = 1000
const defaultConnectionPoolMaxIdle = 100

// Safety limits to prevent DoS
const maxTestDuration = 1800                // 30 minutes max
const maxTestRate = 10000                   // 10k requests/sec max
const maxTestTimeout = 30                   // 30 seconds max
const maxRequestBodySize = 10 * 1024 * 1024 // 10MB max request body size
const maxConnectionPoolConns = 10000        // Max total connections allowed

// Defines a single request
type RequestConfig struct {
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	Body        string            `json:"body,omitempty"`
	ContentType string            `json:"contentType,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// HTTP client connection pool settings
type ConnectionPoolConfig struct {
	MaxConnections *int `json:"maxConnections,omitempty"` // Max total connections
	MaxIdleConns   *int `json:"maxIdleConns,omitempty"`   // Max idle connections
}

// Defines the overall load test
type LoadTestConfig struct {
	Duration       int                   `json:"duration,omitempty"`       // Test duration in seconds
	Rate           int                   `json:"rate,omitempty"`           // Requests per second
	Timeout        int                   `json:"timeout,omitempty"`        // Request timeout in seconds
	WarmupDelay    int                   `json:"warmupDelay,omitempty"`    // Delay before starting test in seconds
	KeepAlive      *bool                 `json:"keepAlive,omitempty"`      // Keep connections alive
	HTTP2          *bool                 `json:"http2,omitempty"`          // Use HTTP/2
	Redirects      *int                  `json:"redirects,omitempty"`      // Max redirects to follow
	ConnectionPool *ConnectionPoolConfig `json:"connectionPool,omitempty"` // Connection pool settings
	Requests       []RequestConfig       `json:"requests"`                 // List of requests
}

func loadConfigFromFile(filename string) (LoadTestConfig, error) {
	// Validate and sanitize the file path
	cleanPath := filepath.Clean(filename)

	// Ensure the file has a .json extension
	if !strings.HasSuffix(strings.ToLower(cleanPath), ".json") {
		return LoadTestConfig{}, fmt.Errorf("config file must have a .json extension")
	}

	// Prevent directory traversal - reject paths with ".."
	if strings.Contains(cleanPath, "..") {
		return LoadTestConfig{}, fmt.Errorf("invalid file path: directory traversal detected")
	}

	// If it's an absolute path, ensure it's not accessing system directories
	if filepath.IsAbs(cleanPath) {
		// Define a list of restricted directories
		restrictedPrefixes := []string{
			"/etc", "/sys", "/proc", "/dev", "/var/log", "/root",
			"/home", "/Users", "/tmp", "/private",
		}

		for _, prefix := range restrictedPrefixes {
			if strings.HasPrefix(cleanPath, prefix) {
				return LoadTestConfig{}, fmt.Errorf("access to system directories is not allowed")
			}
		}
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return LoadTestConfig{}, err
	}

	var config LoadTestConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return LoadTestConfig{}, err
	}

	// Validate request body sizes
	for i, req := range config.Requests {
		if len(req.Body) > maxRequestBodySize {
			return LoadTestConfig{}, fmt.Errorf("request %d body size (%d bytes) exceeds maximum allowed size (%d bytes)",
				i+1, len(req.Body), maxRequestBodySize)
		}
	}

	return config, nil
}

// Sets default values for missing configuration
func (config *LoadTestConfig) applyDefaults() {
	if config.Duration == 0 {
		config.Duration = int(defaultTestSeconds)
	}
	if config.Rate == 0 {
		config.Rate = defaultTestRate
	}
	if config.Timeout == 0 {
		config.Timeout = int(defaultTestTimeout)
	}
	if config.WarmupDelay == 0 {
		config.WarmupDelay = defaultWarmupDelay
	}
}
