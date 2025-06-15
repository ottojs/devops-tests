package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

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
	config.applyDefaults()

	// Validate limits to prevent DoS
	if err := validateLimits(&config); err != nil {
		fmt.Printf("Error: %v\n", err)
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
		if config.ConnectionPool != nil {
			fmt.Println("  Connection Pool:")
			if config.ConnectionPool.MaxConnections != nil {
				fmt.Printf("    Max Connections: %d\n", *config.ConnectionPool.MaxConnections)
			}
			if config.ConnectionPool.MaxIdleConns != nil {
				fmt.Printf("    Max Idle Connections: %d\n", *config.ConnectionPool.MaxIdleConns)
			}
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

	// Build attacker options
	var attackerOpts []func(*vegeta.Attacker)

	if config.KeepAlive != nil {
		attackerOpts = append(attackerOpts, vegeta.KeepAlive(*config.KeepAlive))
	}

	if config.HTTP2 != nil {
		attackerOpts = append(attackerOpts, vegeta.HTTP2(*config.HTTP2))
	}

	if config.Redirects != nil {
		attackerOpts = append(attackerOpts, vegeta.Redirects(*config.Redirects))
	}

	// Apply connection pool settings
	if config.ConnectionPool != nil {
		if config.ConnectionPool.MaxConnections != nil {
			attackerOpts = append(attackerOpts, vegeta.Connections(*config.ConnectionPool.MaxConnections))
		}
	}

	attackerOpts = append(attackerOpts, vegeta.Timeout(time.Duration(config.Timeout)*time.Second))

	attacker := vegeta.NewAttacker(attackerOpts...)

	var metrics vegeta.Metrics
	for res := range attacker.Attack(targeter, rate, duration, "Load Test") {
		metrics.Add(res)
	}
	metrics.Close()

	// Output results in the requested format
	outputResults(config, metrics, *jsonOutput)
}
