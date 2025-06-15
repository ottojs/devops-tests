package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

// Exit codes
const (
	exitSuccess = 0
	exitError   = 1
)

// Linearly ramps from startRate to endRate over the duration
// If holdDuration > 0, it ramps up over (duration - holdDuration) and then holds at endRate
func createRampUpPacer(startRate, endRate int, duration time.Duration, holdDuration time.Duration) vegeta.Pacer {
	if holdDuration == 0 {
		// Simple linear ramp over entire duration
		return vegeta.LinearPacer{
			StartAt: vegeta.Rate{Freq: startRate, Per: time.Second},
			Slope:   float64(endRate-startRate) / duration.Seconds(),
		}
	}

	// Ramp up over (duration - holdDuration), then hold at endRate
	rampDuration := duration - holdDuration
	if rampDuration <= 0 {
		// If hold duration >= total duration, just use end rate
		return vegeta.ConstantPacer{Freq: endRate, Per: time.Second}
	}

	// Create a composite pacer that ramps up then holds
	return &rampHoldPacer{
		startRate:    startRate,
		endRate:      endRate,
		rampDuration: rampDuration,
		holdDuration: holdDuration,
	}
}

// Implements a pacer that ramps up linearly then holds at the end rate
type rampHoldPacer struct {
	startRate    int
	endRate      int
	rampDuration time.Duration
	holdDuration time.Duration
}

func (p *rampHoldPacer) Pace(elapsed time.Duration, _ uint64) (time.Duration, bool) {
	if elapsed >= p.rampDuration+p.holdDuration {
		return 0, true // Stop the attack
	}

	var currentRate float64
	if elapsed < p.rampDuration {
		// During ramp phase
		progress := elapsed.Seconds() / p.rampDuration.Seconds()
		currentRate = float64(p.startRate) + (float64(p.endRate-p.startRate) * progress)
	} else {
		// During hold phase
		currentRate = float64(p.endRate)
	}

	// Calculate wait time for current rate
	if currentRate > 0 {
		waitTime := time.Second / time.Duration(currentRate)
		return waitTime, false
	}

	return time.Second, false
}

func (p *rampHoldPacer) Rate(elapsed time.Duration) float64 {
	if elapsed < p.rampDuration {
		// During ramp phase
		progress := elapsed.Seconds() / p.rampDuration.Seconds()
		return float64(p.startRate) + (float64(p.endRate-p.startRate) * progress)
	}
	// During hold phase
	return float64(p.endRate)
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
		os.Exit(exitError)
	}

	loadedConfig, err := loadConfigFromFile(*configFile)
	if err != nil {
		fmt.Printf("Error loading config file: %v\n", err)
		os.Exit(exitError)
	}
	config = loadedConfig
	requests = config.Requests
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

	time.Sleep(time.Duration(config.WarmupDelay) * time.Second)

	if !*jsonOutput {
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

	if config.RampUp != nil {
		// Use ramp-up pacer
		holdDuration := time.Duration(config.RampUp.HoldDuration) * time.Second
		pacer := createRampUpPacer(config.RampUp.StartRate, config.RampUp.EndRate, duration, holdDuration)

		// Set up periodic rate updates for non-JSON mode
		var updateTicker *time.Ticker
		var tickerDone chan bool
		if !*jsonOutput {
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

		for res := range attacker.Attack(targeter, pacer, duration, "Load Test") {
			metrics.Add(res)
		}

		if updateTicker != nil {
			updateTicker.Stop()
			tickerDone <- true
		}
	} else {
		// Use constant rate
		for res := range attacker.Attack(targeter, rate, duration, "Load Test") {
			metrics.Add(res)
		}
	}

	metrics.Close()

	// Output results in the requested format
	outputResults(config, metrics, *jsonOutput)
}
