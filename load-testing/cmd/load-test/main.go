package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	// Parse command line flags
	opts := parseFlags()

	// Load configuration
	config, err := loadConfigFromFile(opts.ConfigFile)
	if err != nil {
		fmt.Printf("Error loading config file: %v\n", err)
		os.Exit(exitError)
	}

	requests := config.Requests
	if len(requests) == 0 {
		fmt.Println("Error: No requests found in config file")
		os.Exit(exitError)
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
		os.Exit(exitError)
	}

	// Apply defaults for missing values
	config.applyDefaults()

	// Validate limits to prevent DoS
	if err := validateLimits(&config); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(exitError)
	}

	// Only show startup messages if not in JSON mode
	if !opts.JSONOutput {
		printStartupInfo(config, requests)
	}

	// Warmup delay
	time.Sleep(time.Duration(config.WarmupDelay) * time.Second)

	if !opts.JSONOutput {
		printAttackInfo(config)
	}

	// Run the attack
	metrics := runAttack(config, requests, opts.JSONOutput)

	// Output results in the requested format
	outputResults(config, metrics, opts.JSONOutput)
}
