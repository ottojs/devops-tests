package main

import (
	"fmt"
	"time"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

// Executes the load test attack with the given configuration
func runAttack(config LoadTestConfig, requests []RequestConfig, jsonOutput bool) vegeta.Metrics {
	duration := time.Duration(config.Duration) * time.Second

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

	if config.RampUp != nil {
		// Use ramp-up pacer
		metrics = runRampUpAttack(attacker, targeter, config, duration, jsonOutput)
	} else {
		// Use constant rate
		metrics = runConstantRateAttack(attacker, targeter, config, duration)
	}

	metrics.Close()
	return metrics
}

// Runs an attack with a constant request rate
func runConstantRateAttack(attacker *vegeta.Attacker, targeter vegeta.Targeter, config LoadTestConfig, duration time.Duration) vegeta.Metrics {
	rate := vegeta.Rate{
		Freq: config.Rate,
		Per:  time.Second,
	}

	var metrics vegeta.Metrics
	for res := range attacker.Attack(targeter, rate, duration, "Load Test") {
		metrics.Add(res)
	}

	return metrics
}

// Runs an attack with a ramping request rate
func runRampUpAttack(attacker *vegeta.Attacker, targeter vegeta.Targeter, config LoadTestConfig, duration time.Duration, jsonOutput bool) vegeta.Metrics {
	holdDuration := time.Duration(config.RampUp.HoldDuration) * time.Second
	pacer := createRampUpPacer(config.RampUp.StartRate, config.RampUp.EndRate, duration, holdDuration)

	// Set up periodic rate updates for non-JSON mode
	var updateTicker *time.Ticker
	var tickerDone chan bool
	if !jsonOutput {
		updateTicker = time.NewTicker(5 * time.Second)
		tickerDone = make(chan bool)
		startTime := time.Now()

		go func() {
			rampDuration := duration - holdDuration
			for {
				select {
				case <-updateTicker.C:
					elapsed := time.Since(startTime).Seconds()
					if elapsed < duration.Seconds() {
						var currentRate float64
						if holdDuration > 0 && elapsed >= rampDuration.Seconds() {
							// In hold phase
							currentRate = float64(config.RampUp.EndRate)
							fmt.Printf("  Current rate: %d req/s (holding at max)\n", int(currentRate))
						} else if rampDuration > 0 {
							// In ramp phase
							progress := elapsed / rampDuration.Seconds()
							if progress > 1.0 {
								progress = 1.0
							}
							currentRate = float64(config.RampUp.StartRate) +
								(float64(config.RampUp.EndRate-config.RampUp.StartRate) * progress)
							fmt.Printf("  Current rate: %d req/s\n", int(currentRate))
						}
					}
				case <-tickerDone:
					return
				}
			}
		}()
	}

	var metrics vegeta.Metrics
	for res := range attacker.Attack(targeter, pacer, duration, "Load Test") {
		metrics.Add(res)
	}

	if updateTicker != nil {
		updateTicker.Stop()
		tickerDone <- true
	}

	return metrics
}

// Prints test configuration and request information
func printStartupInfo(config LoadTestConfig, requests []RequestConfig) {
	duration := time.Duration(config.Duration) * time.Second

	fmt.Printf("Loaded %d requests\n", len(requests))
	fmt.Println("Request types:")
	for i, req := range requests {
		fmt.Printf("  %d. %s %s", i+1, req.Method, req.URL)
		fmt.Println()
	}
	fmt.Println("Test Configuration:")
	fmt.Printf("  Duration: %s\n", duration)
	if config.RampUp != nil {
		fmt.Printf("  Rate: Ramp from %d to %d requests/sec\n", config.RampUp.StartRate, config.RampUp.EndRate)
	} else {
		fmt.Printf("  Rate: %d requests/sec\n", config.Rate)
	}
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

// Prints information about the attack in progress
func printAttackInfo(config LoadTestConfig) {
	fmt.Println("Attacking in progress...")
	if config.RampUp != nil {
		if config.RampUp.HoldDuration > 0 {
			rampTime := config.Duration - config.RampUp.HoldDuration
			fmt.Printf("  Starting at %d req/s, ramping to %d req/s over %ds, then holding for %ds\n",
				config.RampUp.StartRate, config.RampUp.EndRate, rampTime, config.RampUp.HoldDuration)
		} else {
			fmt.Printf("  Starting at %d req/s, ramping to %d req/s over %ds\n",
				config.RampUp.StartRate, config.RampUp.EndRate, config.Duration)
		}
	} else {
		fmt.Printf("  Constant rate: %d req/s\n", config.Rate)
	}
}
