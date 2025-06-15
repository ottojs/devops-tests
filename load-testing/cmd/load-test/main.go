package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

// Default Settings
const DEFAULT_TEST_SECONDS time.Duration = 10
const DEFAULT_TEST_RATE int = 150
const DEFAULT_TEST_TIMEOUT time.Duration = 5 // seconds
const DEFAULT_WARMUP_DELAY int = 15          // seconds

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

func main() {
	// Command line flags
	configFile := flag.String("config", "", "Path to JSON config file")
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

	duration := time.Duration(config.Duration) * time.Second
	fmt.Printf("Loaded %d requests\n", len(requests))
	fmt.Println("Request types:")
	for i, req := range requests {
		fmt.Printf("  %d. %s %s", i+1, req.Method, req.URL)
		if req.ContentType != "" {
			fmt.Printf(" (Content-Type: %s)", req.ContentType)
		}
		fmt.Println()
	}
	fmt.Printf("\nTest Configuration:\n")
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
	fmt.Printf("\nStop this process (CTRL+C) within %d seconds to cancel\n", config.WarmupDelay)
	time.Sleep(time.Duration(config.WarmupDelay) * time.Second)
	fmt.Println("Attacking in progress...")

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
	} else {
		vegeta.KeepAlive(false)(attacker)
	}

	if config.HTTP2 != nil {
		vegeta.HTTP2(*config.HTTP2)(attacker)
	} else {
		vegeta.HTTP2(false)(attacker)
	}

	if config.Redirects != nil {
		vegeta.Redirects(*config.Redirects)(attacker)
	} else {
		vegeta.Redirects(0)(attacker)
	}

	vegeta.Timeout(time.Duration(config.Timeout) * time.Second)(attacker)

	var metrics vegeta.Metrics
	for res := range attacker.Attack(targeter, rate, duration, "Load Test") {
		metrics.Add(res)
	}
	metrics.Close()

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
	//fmt.Printf("\n %+v", metrics)

}

// Rotates through the requests
func createRotatingTargeter(requests []RequestConfig) vegeta.Targeter {
	var counter uint64

	return func(tgt *vegeta.Target) error {
		// Rotate through requests using atomic counter
		idx := int(atomic.AddUint64(&counter, 1)-1) % len(requests)
		req := requests[idx]

		// Set basic fields
		tgt.Method = req.Method
		tgt.URL = req.URL

		// Set body if present
		if req.Body != "" {
			tgt.Body = []byte(req.Body)
		}

		// Set headers
		tgt.Header = make(map[string][]string)
		if req.ContentType != "" {
			tgt.Header["Content-Type"] = []string{req.ContentType}
		}

		// Add custom headers if any
		for k, v := range req.Headers {
			tgt.Header[k] = []string{v}
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
