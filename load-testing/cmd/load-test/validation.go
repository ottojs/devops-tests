package main

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// Approved domains/IPs - hardcoded to prevent abuse
// To add more domains, modify this list and recompile
var APPROVED_DOMAINS = []string{
	"localhost",
	"127.0.0.1",
}

// Allowed HTTP methods
var ALLOWED_HTTP_METHODS = []string{
	"GET",
	"POST",
	"PUT",
	"DELETE",
	"PATCH",
	"HEAD",
	"OPTIONS",
}

// Private IP ranges that are allowed
var PRIVATE_IP_RANGES = []string{
	"10.0.0.0/8",     // Class A private
	"172.16.0.0/12",  // Class B private
	"192.168.0.0/16", // Class C private
}

// Checks if the URL is allowed to be targeted
func isApprovedTarget(targetURL string) error {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %v", err)
	}

	// Validate URL scheme
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("only HTTP and HTTPS schemes are allowed, got: %s", parsed.Scheme)
	}

	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("no hostname in URL")
	}

	// Check against approved domains
	for _, approved := range APPROVED_DOMAINS {
		if strings.EqualFold(host, approved) {
			return nil
		}
	}

	// Check if it's an IP address
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("target '%s' is not in approved domains list", host)
	}

	// Check against private IP ranges
	for _, cidr := range PRIVATE_IP_RANGES {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if ipnet.Contains(ip) {
			return nil
		}
	}

	return fmt.Errorf("target IP '%s' is not in approved private ranges", host)
}

// Checks if HTTP method is allowed
func isAllowedMethod(method string) bool {
	upperMethod := strings.ToUpper(method)
	for _, allowed := range ALLOWED_HTTP_METHODS {
		if upperMethod == allowed {
			return true
		}
	}
	return false
}

// Checks all requests have approved targets
func validateRequests(requests []RequestConfig) error {
	for i, req := range requests {
		// Validate HTTP method
		if !isAllowedMethod(req.Method) {
			return fmt.Errorf("request %d: invalid HTTP method '%s'. Allowed methods: %v",
				i+1, req.Method, ALLOWED_HTTP_METHODS)
		}

		// Validate target URL
		if err := isApprovedTarget(req.URL); err != nil {
			return fmt.Errorf("request %d: %v", i+1, err)
		}
	}
	return nil
}

// Checks if config values are within safe limits
func validateLimits(config *LoadTestConfig) error {
	// Check for negative values
	if config.Duration < 0 {
		return fmt.Errorf("duration cannot be negative (got %d)", config.Duration)
	}
	if config.Rate < 0 {
		return fmt.Errorf("rate cannot be negative (got %d)", config.Rate)
	}
	if config.Timeout < 0 {
		return fmt.Errorf("timeout cannot be negative (got %d)", config.Timeout)
	}
	if config.WarmupDelay < 0 {
		return fmt.Errorf("warmup delay cannot be negative (got %d)", config.WarmupDelay)
	}
	if config.Redirects != nil && *config.Redirects < 0 {
		return fmt.Errorf("redirects cannot be negative (got %d)", *config.Redirects)
	}

	// Check maximum limits
	if config.Duration > MAX_TEST_DURATION {
		return fmt.Errorf("duration %ds exceeds maximum allowed (%ds)", config.Duration, MAX_TEST_DURATION)
	}
	if config.Rate > MAX_TEST_RATE {
		return fmt.Errorf("rate %d exceeds maximum allowed (%d requests/sec)", config.Rate, MAX_TEST_RATE)
	}
	if config.Timeout > MAX_TEST_TIMEOUT {
		return fmt.Errorf("timeout %ds exceeds maximum allowed (%ds)", config.Timeout, MAX_TEST_TIMEOUT)
	}
	return nil
}
