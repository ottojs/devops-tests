package main

import (
	"fmt"
	"net/http"
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

	// Create a custom HTTP client that removes Vegeta headers
	client := createHeaderStrippingClient(config)
	attackerOpts = append(attackerOpts, vegeta.Client(client))

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
	for res := range attacker.Attack(targeter, rate, duration, "") {
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
	for res := range attacker.Attack(targeter, pacer, duration, "") {
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

// headerStrippingTransport wraps http.RoundTripper to remove Vegeta headers
type headerStrippingTransport struct {
	base http.RoundTripper
}

func (t *headerStrippingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Remove Vegeta headers before sending the request
	req.Header.Del("X-Vegeta-Attack")
	req.Header.Del("X-Vegeta-Seq")

	return t.base.RoundTrip(req)
}

// createHeaderStrippingClient creates an HTTP client that removes Vegeta headers
func createHeaderStrippingClient(config LoadTestConfig) *http.Client {
	// Create base transport with connection pooling settings
	transport := &http.Transport{
		MaxIdleConnsPerHost: 100,
		DisableCompression:  false,
		DisableKeepAlives:   false,
	}

	// Apply connection pool settings if specified
	if config.ConnectionPool != nil {
		if config.ConnectionPool.MaxIdleConns != nil {
			transport.MaxIdleConns = *config.ConnectionPool.MaxIdleConns
		}
		if config.ConnectionPool.MaxConnections != nil {
			transport.MaxConnsPerHost = *config.ConnectionPool.MaxConnections
		}
	}

	// Apply keep-alive settings
	if config.KeepAlive != nil {
		transport.DisableKeepAlives = !*config.KeepAlive
	}

	// Apply HTTP/2 settings
	if config.HTTP2 != nil && !*config.HTTP2 {
		transport.ForceAttemptHTTP2 = false
	}

	// Wrap with header stripping transport
	strippingTransport := &headerStrippingTransport{base: transport}

	client := &http.Client{
		Transport: strippingTransport,
		Timeout:   time.Duration(config.Timeout) * time.Second,
	}

	// Apply redirect settings
	if config.Redirects != nil {
		if *config.Redirects == 0 {
			client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			}
		} else {
			client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
				if len(via) >= *config.Redirects {
					return http.ErrUseLastResponse
				}
				return nil
			}
		}
	}

	return client
}
