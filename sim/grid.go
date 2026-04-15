package sim

// Duke Energy Schedule RSTC (NC Residential Solar TOU) rates.
const (
	OffPeakRate = 0.098 // $/kWh
	OnPeakRate  = 0.224 // $/kWh
	NEGCreditRate = 0.03 // $/kWh net excess generation credit
)

// IsOnPeak returns true if the given fractional hour falls within the
// on-peak window for the given season. On-peak applies weekdays only,
// but the simulation treats every day as a weekday for simplicity.
//
//   Summer (May–Sep):      3:00 PM – 6:00 PM  (15.0–18.0)
//   Non-summer (Oct–Apr):  6:00 AM – 9:00 AM  ( 6.0– 9.0)
func IsOnPeak(hour float64, season string) bool {
	if season == "summer" {
		return hour >= 15.0 && hour < 18.0
	}
	return hour >= 6.0 && hour < 9.0
}

// GridPrice returns the current electricity price for the given time and season.
func GridPrice(hour float64, season string) float64 {
	if IsOnPeak(hour, season) {
		return OnPeakRate
	}
	return OffPeakRate
}
