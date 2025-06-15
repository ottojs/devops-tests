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

// Checks all requests have approved targets
func validateRequests(requests []RequestConfig) error {
	for i, req := range requests {
		if err := isApprovedTarget(req.URL); err != nil {
			return fmt.Errorf("request %d: %v", i+1, err)
		}
	}
	return nil
}

// Checks if config values are within safe limits
func validateLimits(config *LoadTestConfig) error {
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
