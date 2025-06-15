package main

import (
	"encoding/json"
	"fmt"
	"os"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

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

// Displays the test results in the requested format
func outputResults(config LoadTestConfig, metrics vegeta.Metrics, jsonOutput bool) {
	if jsonOutput {
		outputJSON(config, metrics)
	} else {
		outputPlain(metrics)
	}
}

// Outputs results in JSON format
func outputJSON(config LoadTestConfig, metrics vegeta.Metrics) {
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
		os.Exit(exitError)
	}
	fmt.Println(string(output))
}

// Outputs results in plain text format
func outputPlain(metrics vegeta.Metrics) {
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
