package sim

import (
	"math"
	"math/rand"

	"github.com/djmurong/energy-systems/data"
)

// RunSolar reads ticks and produces solar output (kW) for each step.
// Production is based on Durham NC insolation data with added cloud noise.
func RunSolar(ticks <-chan Tick, cfg SimConfig) <-chan float64 {
	ch := make(chan float64)
	go func() {
		defer close(ch)
		rng := rand.New(rand.NewSource(cfg.Seed))

		peakSunHrs := data.PeakSunHours(cfg.Season)
		// Scale factor: peakKW * (peakSunHrs / integralOfCurve) so that
		// the daily integral equals peakKW * peakSunHrs kWh.
		// The Gaussian integral over a day ≈ sigma * sqrt(2π).
		var sigma float64
		if cfg.Season == "summer" {
			sigma = 3.0
		} else {
			sigma = 2.5
		}
		gaussianIntegral := sigma * math.Sqrt(2*math.Pi)
		// scaleFactor converts normalized irradiance to kW such that
		// total daily production = peakKW * peakSunHrs (approximately).
		scaleFactor := cfg.SolarPeakKW * peakSunHrs / gaussianIntegral

		for tick := range ticks {
			irr := data.SolarIrradiance(tick.Hour, tick.Season)
			noise := 1.0 + 0.15*(rng.Float64()*2-1) // ±15% cloud noise
			production := scaleFactor * irr * noise
			if production < 0 {
				production = 0
			}
			ch <- production
		}
	}()
	return ch
}
