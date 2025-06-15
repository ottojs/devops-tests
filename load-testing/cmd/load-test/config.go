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
const DEFAULT_TEST_SECONDS time.Duration = 10
const DEFAULT_TEST_RATE int = 150
const DEFAULT_TEST_TIMEOUT time.Duration = 5 // seconds
const DEFAULT_WARMUP_DELAY int = 15          // seconds

// Safety limits to prevent DoS
const MAX_TEST_DURATION = 1800                 // 30 minutes max
const MAX_TEST_RATE = 10000                    // 10k requests/sec max
const MAX_TEST_TIMEOUT = 30                    // 30 seconds max
const MAX_REQUEST_BODY_SIZE = 10 * 1024 * 1024 // 10MB max request body size

// Defines a single request
type RequestConfig struct {
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	Body        string            `json:"body,omitempty"`
	ContentType string            `json:"contentType,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// Defines the overall load test
type LoadTestConfig struct {
	Duration    int             `json:"duration,omitempty"`    // Test duration in seconds
	Rate        int             `json:"rate,omitempty"`        // Requests per second
	Timeout     int             `json:"timeout,omitempty"`     // Request timeout in seconds
	WarmupDelay int             `json:"warmupDelay,omitempty"` // Delay before starting test in seconds
	KeepAlive   *bool           `json:"keepAlive,omitempty"`   // Keep connections alive
	HTTP2       *bool           `json:"http2,omitempty"`       // Use HTTP/2
	Redirects   *int            `json:"redirects,omitempty"`   // Max redirects to follow
	Requests    []RequestConfig `json:"requests"`              // List of requests
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
		if len(req.Body) > MAX_REQUEST_BODY_SIZE {
			return LoadTestConfig{}, fmt.Errorf("request %d body size (%d bytes) exceeds maximum allowed size (%d bytes)",
				i+1, len(req.Body), MAX_REQUEST_BODY_SIZE)
		}
	}

	return config, nil
}

// Sets default values for missing configuration
func (config *LoadTestConfig) applyDefaults() {
	if config.Duration == 0 {
		config.Duration = int(DEFAULT_TEST_SECONDS)
	}
	if config.Rate == 0 {
		config.Rate = DEFAULT_TEST_RATE
	}
	if config.Timeout == 0 {
		config.Timeout = int(DEFAULT_TEST_TIMEOUT)
	}
	if config.WarmupDelay == 0 {
		config.WarmupDelay = DEFAULT_WARMUP_DELAY
	}
}
