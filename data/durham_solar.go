package data

import "math"

// Durham, NC (35.99°N, 78.90°W) solar insolation parameters derived from
// NREL TMY data. Peak sun hours ≈ 5.0 kWh/m²/day annual average.

// SolarIrradiance returns the normalized irradiance (0–1) for the given
// fractional hour of day, using a Gaussian bell curve centered at solar noon
// (≈13:00 local time in Durham during summer, shifted slightly for non-summer).
// The curve width models the ~14-hour daylight window in summer and ~10-hour
// window in winter.
func SolarIrradiance(hour float64, season string) float64 {
	var solarNoon, sigma float64
	if season == "summer" {
		solarNoon = 13.25 // slightly after noon due to DST
		sigma = 3.0       // wider curve: sunrise ~6 AM, sunset ~8:30 PM
	} else {
		solarNoon = 12.5
		sigma = 2.5 // narrower: sunrise ~7 AM, sunset ~5:30 PM
	}
	diff := hour - solarNoon
	irr := math.Exp(-(diff * diff) / (2 * sigma * sigma))
	// Zero out nighttime entirely
	if irr < 0.01 {
		return 0
	}
	return irr
}

// PeakSunHours returns the average daily insolation in kWh/m²/day for Durham.
func PeakSunHours(season string) float64 {
	if season == "summer" {
		return 5.5
	}
	return 3.8
}
