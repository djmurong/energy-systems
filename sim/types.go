package sim

import "math/rand"

// Tick represents a single simulation time step broadcast by the clock.
type Tick struct {
	Step      int
	Hour      float64 // 0.0–24.0 fractional hour
	Season    string  // "summer" or "nonsummer"
	GridPrice float64 // $/kWh for current TOU period
	NEGRate   float64 // $/kWh credit for net excess generation
	OnPeak    bool
}

// BatteryConfig holds the physical parameters of the battery.
type BatteryConfig struct {
	CapacityKWh    float64 // total capacity (e.g. 13.5)
	MaxChargeKW    float64 // max charge rate (e.g. 5.0)
	MaxDischargeKW float64 // max discharge rate (e.g. 5.0)
	MinReserve     float64 // X threshold as fraction 0–1 (e.g. 0.20)
	MaxGridCharge  float64 // Y threshold as fraction 0–1 (e.g. 0.80)
	InitialSOC     float64 // starting state of charge as fraction 0–1
}

// BatteryState is the current snapshot of battery charge.
type BatteryState struct {
	ChargeKWh   float64
	CapacityKWh float64
	MinReserve  float64 // absolute kWh floor (X * Capacity)
	MaxCharge   float64 // absolute kWh ceiling for grid charging (Y * Capacity)
}

// SOC returns state of charge as a fraction 0–1.
func (b BatteryState) SOC() float64 {
	if b.CapacityKWh == 0 {
		return 0
	}
	return b.ChargeKWh / b.CapacityKWh
}

// PolicyDecision describes how to route energy for one tick.
// All values are in kW (power for the duration of one time step).
type PolicyDecision struct {
	SolarToLoad    float64
	SolarToBattery float64
	SolarToGrid    float64 // spill → NEG credit
	BatteryToLoad  float64
	GridToLoad     float64
	GridToBattery  float64
}

// TickResult captures the full state of one simulation step for logging.
type TickResult struct {
	Step           int
	Hour           float64
	GridPrice      float64
	OnPeak         bool
	SolarKW        float64
	LoadKW         float64
	BatterySOC     float64 // 0–1 fraction after this tick
	BatteryKWh     float64
	SolarToLoad    float64
	SolarToBattery float64
	SolarToGrid    float64
	BatteryToLoad  float64
	GridToLoad     float64
	GridToBattery  float64
	GridCost       float64 // positive = money spent
	NEGCredit      float64 // positive = money earned
	CumulativeCost float64
}

// SimConfig holds all tuneable parameters for one simulation run.
type SimConfig struct {
	Days          int
	Season        string
	StepMinutes   int // duration of each tick in minutes (default 5)
	SolarPeakKW   float64
	Battery       BatteryConfig
	Seed          int64
	OutputDir     string
	PolicyName    string
	Rng           *rand.Rand
}

// StepsPerDay returns the number of ticks in one simulated day.
func (c SimConfig) StepsPerDay() int {
	return (24 * 60) / c.StepMinutes
}

// StepHours returns the duration of one tick in hours.
func (c SimConfig) StepHours() float64 {
	return float64(c.StepMinutes) / 60.0
}

// TotalSteps returns the total number of ticks for the simulation.
func (c SimConfig) TotalSteps() int {
	return c.Days * c.StepsPerDay()
}

// PolicyFunc is the signature every policy must implement.
// Given the current tick, solar production, household load, and battery state,
// it returns a routing decision.
type PolicyFunc func(tick Tick, solarKW, loadKW float64, bat BatteryState, cfg SimConfig) PolicyDecision
