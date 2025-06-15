package main

import (
	"time"

	vegeta "github.com/tsenart/vegeta/v12/lib"
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
