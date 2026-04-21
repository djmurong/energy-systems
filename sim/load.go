package sim

import "math/rand"

// RunLoad reads ticks and produces household demand (kW) for each step.
// Demand is composed of:
//   - A constant baseline (fridge, lights, electronics)
//   - A temperature-driven HVAC curve that spikes during afternoon heat
//   - Random appliance events (oven, EV charger, dryer)
func RunLoad(ticks <-chan Tick, cfg SimConfig) <-chan float64 {
	ch := make(chan float64)
	go func() {
		defer close(ch)
		rng := rand.New(rand.NewSource(cfg.Seed + 1))

		for tick := range ticks {
			demand := baselineLoad(tick.Hour) + hvacLoad(tick.Hour, tick.Season) + applianceSpike(rng)
			ch <- demand
		}
	}()
	return ch
}

// baselineLoad returns the always-on household consumption in kW.
// Slightly higher in evening hours when lights and entertainment are on.
func baselineLoad(hour float64) float64 {
	if hour >= 18.0 || hour < 6.0 {
		return 1.0 // evening/night: more lighting
	}
	return 0.5
}

// hvacLoad models air conditioning (summer) or heating (non-summer) demand.
// Summer AC peaks in the late afternoon; heating peaks in early morning.
func hvacLoad(hour float64, season string) float64 {
	if season == "summer" {
		// AC load ramps from noon, peaks 3–6 PM, tapers by 9 PM.
		if hour >= 12.0 && hour < 21.0 {
			// Peak at 16:00
			peak := 3.5
			dist := (hour - 16.0)
			// Gaussian-ish shape
			load := peak * gaussFast(dist, 2.5)
			if load < 0.3 {
				return 0.3
			}
			return load
		}
		return 0.3 // overnight minimum AC
	}
	// Non-summer: heating peaks around 7 AM when the house wakes up.
	if hour >= 5.0 && hour < 10.0 {
		peak := 3.0
		dist := hour - 7.0
		load := peak * gaussFast(dist, 1.5)
		if load < 0.2 {
			return 0.2
		}
		return load
	}
	return 0.2
}

// applianceSpike randomly adds large discrete loads.
func applianceSpike(rng *rand.Rand) float64 {
	r := rng.Float64()
	switch {
	case r < 0.005: // ~0.5% chance per tick → EV charger (7.2 kW)
		return 7.2
	case r < 0.015: // ~1% → oven (3.0 kW)
		return 3.0
	case r < 0.025: // ~1% → dryer (5.5 kW)
		return 5.5
	default:
		return 0
	}
}

func gaussFast(x, sigma float64) float64 {
	return exp(-x * x / (2 * sigma * sigma))
}

// exp is a quick alias to avoid importing math in this file.
func exp(x float64) float64 {
	// Taylor-free: just use the standard library via a small wrapper.
	// We import math at the top if needed; for now use a fast approximation
	// that's fine for load shaping (not scientific precision).
	if x < -10 {
		return 0
	}
	// Use the identity: e^x ≈ (1 + x/256)^256 for |x| < 10
	r := 1.0 + x/256.0
	for i := 0; i < 8; i++ {
		r *= r
	}
	if r < 0 {
		return 0
	}
	return r
}
