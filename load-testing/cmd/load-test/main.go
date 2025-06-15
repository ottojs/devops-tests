package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"time"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

// Default Settings
const DEFAULT_TEST_SECONDS time.Duration = 10
const DEFAULT_TEST_RATE int = 150
const DEFAULT_TEST_TIMEOUT time.Duration = 5 // seconds
const DEFAULT_WARMUP_DELAY int = 15          // seconds

// Safety limits to prevent DoS
const MAX_TEST_DURATION = 1800 // 30 minutes max
const MAX_TEST_RATE = 10000    // 10k requests/sec max
const MAX_TEST_TIMEOUT = 30    // 30 seconds max

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

// JSON output structure
type TestResults struct {
	Config    LoadTestConfig `json:"config"`
	Latencies LatencyResults `json:"latencies"`
	Metrics   MetricResults  `json:"metrics"`
	Errors    []string       `json:"errors,omitempty"`
}

type LatencyResults struct {
	Total string `json:"total"`
	Mean  string `json:"mean"`
	Min   string `json:"min"`
	Max   string `json:"max"`
	P50   string `json:"p50"`
	P90   string `json:"p90"`
	P95   string `json:"p95"`
	P99   string `json:"p99"`
}

type MetricResults struct {
	Success     bool           `json:"success"`
	Rate        float64        `json:"rate"`
	Duration    string         `json:"duration"`
	Wait        string         `json:"wait"`
	Requests    uint64         `json:"requests"`
	Throughput  float64        `json:"throughput"`
	BytesIn     uint64         `json:"bytesIn"`
	BytesOut    uint64         `json:"bytesOut"`
	StatusCodes map[string]int `json:"statusCodes"`
}

func main() {
	// Command line flags
	configFile := flag.String("config", "", "Path to JSON config file")
	jsonOutput := flag.Bool("json", false, "Output results in JSON format")
	flag.Parse()

	// Load configuration
	var config LoadTestConfig
	var requests []RequestConfig

	if *configFile == "" {
		fmt.Println("Error: No config file provided. Use -config flag to specify a configuration file.")
		flag.Usage()
		os.Exit(1)
	}

	loadedConfig, err := loadConfigFromFile(*configFile)
	if err != nil {
		fmt.Printf("Error loading config file: %v\n", err)
		os.Exit(1)
	}
	config = loadedConfig
	requests = config.Requests
	if len(requests) == 0 {
		fmt.Println("Error: No requests found in config file")
		os.Exit(1)
	}

	// Validate all request URLs are approved
	if err := validateRequests(requests); err != nil {
		fmt.Printf("Error: %v\n\n", err)
		fmt.Println("Only the following targets are allowed:")
		fmt.Println("  - localhost")
		fmt.Println("  - 127.0.0.1")
		fmt.Println("  - 10.0.0.0/8 (Class A private)")
		fmt.Println("  - 172.16.0.0/12 (Class B private)")
		fmt.Println("  - 192.168.0.0/16 (Class C private)")
		fmt.Println("To add more domains, modify APPROVED_DOMAINS in the source code and recompile.")
		os.Exit(1)
	}

	// Apply defaults for missing values
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

	// Validate limits to prevent DoS
	if config.Duration > MAX_TEST_DURATION {
		fmt.Printf("Error: Duration %ds exceeds maximum allowed (%ds)\n", config.Duration, MAX_TEST_DURATION)
		os.Exit(1)
	}
	if config.Rate > MAX_TEST_RATE {
		fmt.Printf("Error: Rate %d exceeds maximum allowed (%d requests/sec)\n", config.Rate, MAX_TEST_RATE)
		os.Exit(1)
	}
	if config.Timeout > MAX_TEST_TIMEOUT {
		fmt.Printf("Error: Timeout %ds exceeds maximum allowed (%ds)\n", config.Timeout, MAX_TEST_TIMEOUT)
		os.Exit(1)
	}

	duration := time.Duration(config.Duration) * time.Second

	// Only show startup messages if not in JSON mode
	if !*jsonOutput {
		fmt.Printf("Loaded %d requests\n", len(requests))
		fmt.Println("Request types:")
		for i, req := range requests {
			fmt.Printf("  %d. %s %s", i+1, req.Method, req.URL)
			fmt.Println()
		}
		fmt.Println("Test Configuration:")
		fmt.Printf("  Duration: %s\n", duration)
		fmt.Printf("  Rate: %d requests/sec\n", config.Rate)
		fmt.Printf("  Timeout: %ds\n", config.Timeout)
		fmt.Printf("  Warmup Delay: %ds\n", config.WarmupDelay)
		if config.KeepAlive != nil {
			fmt.Printf("  Keep-Alive: %v\n", *config.KeepAlive)
		}
		if config.HTTP2 != nil {
			fmt.Printf("  HTTP/2: %v\n", *config.HTTP2)
		}
		if config.Redirects != nil {
			fmt.Printf("  Max Redirects: %d\n", *config.Redirects)
		}
		fmt.Printf("Stop this process (CTRL+C) within %d seconds to cancel\n", config.WarmupDelay)
	}

	time.Sleep(time.Duration(config.WarmupDelay) * time.Second)

	if !*jsonOutput {
		fmt.Println("Attacking in progress...")
	}

	rate := vegeta.Rate{
		Freq: config.Rate,
		Per:  time.Second,
	}
	// Create request rotation targeter
	targeter := createRotatingTargeter(requests)
	attacker := vegeta.NewAttacker()

	// Apply attacker options
	if config.KeepAlive != nil {
		vegeta.KeepAlive(*config.KeepAlive)(attacker)
	}

	if config.HTTP2 != nil {
		vegeta.HTTP2(*config.HTTP2)(attacker)
	}

	if config.Redirects != nil {
		vegeta.Redirects(*config.Redirects)(attacker)
	}

	vegeta.Timeout(time.Duration(config.Timeout) * time.Second)(attacker)

	var metrics vegeta.Metrics
	for res := range attacker.Attack(targeter, rate, duration, "Load Test") {
		metrics.Add(res)
	}
	metrics.Close()

	if *jsonOutput {
		// Output JSON format
		results := TestResults{
			Config: config,
			Latencies: LatencyResults{
				Total: metrics.Latencies.Total.String(),
				Mean:  metrics.Latencies.Mean.String(),
				Min:   metrics.Latencies.Min.String(),
				Max:   metrics.Latencies.Max.String(),
				P50:   metrics.Latencies.P50.String(),
				P90:   metrics.Latencies.P90.String(),
				P95:   metrics.Latencies.P95.String(),
				P99:   metrics.Latencies.P99.String(),
			},
			Metrics: MetricResults{
				Success:     metrics.Success == 1,
				Rate:        metrics.Rate,
				Duration:    metrics.Duration.String(),
				Wait:        metrics.Wait.String(),
				Requests:    metrics.Requests,
				Throughput:  metrics.Throughput,
				BytesIn:     metrics.BytesIn.Total,
				BytesOut:    metrics.BytesOut.Total,
				StatusCodes: metrics.StatusCodes,
			},
		}

		// Add errors if any
		if len(metrics.Errors) > 0 {
			results.Errors = metrics.Errors
		}

		output, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(output))
	} else {
		// Output plain format (existing)
		fmt.Printf("===== Latencies =====\n")
		fmt.Printf("Total: %s\n", metrics.Latencies.Total)
		fmt.Printf("Average: %s\n", metrics.Latencies.Mean)
		fmt.Printf("Min: %s\n", metrics.Latencies.Min)
		fmt.Printf("Max: %s\n", metrics.Latencies.Max)
		fmt.Printf("50th: %s\n", metrics.Latencies.P50)
		fmt.Printf("90th: %s\n", metrics.Latencies.P90)
		fmt.Printf("95th: %s\n", metrics.Latencies.P95)
		fmt.Printf("99th: %s\n", metrics.Latencies.P99)
		fmt.Printf("Bytes In: %d\n", metrics.BytesIn.Total)
		fmt.Printf("Bytes Out: %d\n", metrics.BytesOut.Total)
		fmt.Printf("===== Info =====\n")
		fmt.Printf("Success: %t\n", metrics.Success == 1)
		fmt.Printf("Rate: %f\n", metrics.Rate)
		fmt.Printf("Duration: %s\n", metrics.Duration)
		fmt.Printf("Wait: %s\n", metrics.Wait)
		fmt.Printf("Total Requests: %d\n", metrics.Requests)
		fmt.Printf("Throughput: %f\n", metrics.Throughput)
		fmt.Printf("StatusCodes:\n")
		for k, v := range metrics.StatusCodes {
			fmt.Println(k, " => ", v)
		}
		fmt.Printf("Errors: %+v\n", metrics.Errors)
		fmt.Printf("\n\n\n")
	}

}

// Rotates through the requests
func createRotatingTargeter(requests []RequestConfig) vegeta.Targeter {
	var counter uint64

	// Pre-process requests to create header maps
	type processedRequest struct {
		method  string
		url     string
		body    []byte
		headers map[string][]string
	}

	processed := make([]processedRequest, len(requests))
	for i, req := range requests {
		pr := processedRequest{
			method:  req.Method,
			url:     req.URL,
			headers: make(map[string][]string),
		}

		// Pre-convert body
		if req.Body != "" {
			pr.body = []byte(req.Body)
		}

		// Pre-build headers
		pr.headers["User-Agent"] = []string{"otto-load-test"}
		if req.ContentType != "" {
			pr.headers["Content-Type"] = []string{req.ContentType}
		}
		for k, v := range req.Headers {
			pr.headers[k] = []string{v}
		}

		processed[i] = pr
	}

	return func(tgt *vegeta.Target) error {
		// Rotate through requests using atomic counter
		idx := int(atomic.AddUint64(&counter, 1)-1) % len(processed)
		req := processed[idx]

		// Set fields from pre-processed data
		tgt.Method = req.method
		tgt.URL = req.url
		tgt.Body = req.body

		// Clone the pre-built headers map
		tgt.Header = make(map[string][]string, len(req.headers))
		for k, v := range req.headers {
			tgt.Header[k] = v
		}

		return nil
	}
}

func loadConfigFromFile(filename string) (LoadTestConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return LoadTestConfig{}, err
	}

	var config LoadTestConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return LoadTestConfig{}, err
	}

	return config, nil
}

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

// Check all requests have approved targets
func validateRequests(requests []RequestConfig) error {
	for i, req := range requests {
		if err := isApprovedTarget(req.URL); err != nil {
			return fmt.Errorf("request %d: %v", i+1, err)
		}
	}
	return nil
}
