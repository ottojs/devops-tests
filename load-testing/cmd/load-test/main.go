package main

import (
	"fmt"
	"os"
	"time"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

// Settings
const TEST_URI string = "http://localhost/"
const TEST_SECONDS time.Duration = 10
const TEST_RATE int = 150
const TEST_TIMEOUT time.Duration = 5 // seconds

func main() {
	// ######################
	// ##### Safe Guard #####
	if TEST_URI == "http://localhost/" {
		fmt.Println("Not performing. Please edit the code to change the URI or remove this block")
		os.Exit(1)
	}
	// ######################
	duration := TEST_SECONDS * time.Second
	fmt.Println("Targeting", TEST_URI, "with", TEST_RATE, "connections for", duration, "seconds...")
	fmt.Println("Stop this process (CTRL+C) within 15 seconds to cancel")
	time.Sleep(15 * time.Second)
	fmt.Println("Attacking in progress...")

	rate := vegeta.Rate{
		Freq: TEST_RATE,
		Per:  time.Second,
	}
	// You can test POST requests with:
	// Method: "POST",
	// Body: []byte(`{"email":"user@example.com"}`),
	targeter := vegeta.NewStaticTargeter(vegeta.Target{
		Method: "GET",
		URL:    TEST_URI,
	})
	attacker := vegeta.NewAttacker()
	vegeta.KeepAlive(false)(attacker)
	vegeta.HTTP2(false)(attacker)
	vegeta.Redirects(0)(attacker)
	vegeta.Timeout(TEST_TIMEOUT * time.Second)(attacker)

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
