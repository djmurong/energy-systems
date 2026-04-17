package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"

	"github.com/djmurong/energy-systems/sim"
)

func main() {
	days := flag.Int("days", 1, "number of simulated days")
	season := flag.String("season", "summer", "season: summer or nonsummer")
	policy := flag.String("policy", "all", "policy to run: greedy-fill, greedy-drain, two-threshold, or all")
	output := flag.String("output", "results", "output directory for CSV files")
	seed := flag.Int64("seed", 42, "random seed for reproducible runs")
	flag.Parse()

	cfg := sim.SimConfig{
		Days:        *days,
		Season:      *season,
		StepMinutes: 5,
		SolarPeakKW: 8.0,
		Battery: sim.BatteryConfig{
			CapacityKWh:    13.5,
			MaxChargeKW:    5.0,
			MaxDischargeKW: 5.0,
			MinReserve:     0.20,
			MaxGridCharge:  0.80,
			InitialSOC:     0.50,
		},
		Seed:      *seed,
		OutputDir: *output,
	}

	policies := sim.AllPolicyNames()
	if *policy != "all" {
		if _, ok := sim.PolicyRegistry[*policy]; !ok {
			fmt.Fprintf(os.Stderr, "unknown policy %q; choose from: greedy-fill, greedy-drain, two-threshold, all\n", *policy)
			os.Exit(1)
		}
		policies = []string{*policy}
	}

	fmt.Printf("Electrotech: Energy Flow Simulator\n")
	fmt.Printf("  Season: %s | Days: %d | Steps/day: %d | Seed: %d\n",
		cfg.Season, cfg.Days, cfg.StepsPerDay(), cfg.Seed)
	fmt.Printf("  Battery: %.1f kWh | Solar: %.1f kW peak\n",
		cfg.Battery.CapacityKWh, cfg.SolarPeakKW)
	fmt.Printf("  Grid rates: off-peak $%.3f | on-peak $%.3f | NEG $%.3f\n",
		sim.OffPeakRate, sim.OnPeakRate, sim.NEGCreditRate)
	fmt.Println()

	var summaries []sim.Summary

	for _, policyName := range policies {
		fmt.Printf("Running policy: %s ...\n", policyName)
		cfg.PolicyName = policyName
		cfg.Rng = rand.New(rand.NewSource(cfg.Seed))

		summary, err := runSimulation(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error running %s: %v\n", policyName, err)
			os.Exit(1)
		}
		summaries = append(summaries, summary)
	}

	sim.PrintSummaryTable(summaries)
}

// runSimulation executes a full simulation for a single policy and returns
// aggregate stats. Each component runs as a goroutine connected by channels.
func runSimulation(cfg sim.SimConfig) (sim.Summary, error) {
	policyFn := sim.PolicyRegistry[cfg.PolicyName]

	// Fan the clock out to solar and load goroutines.
	// Each needs its own tick stream so they can consume independently.
	clockCh := sim.RunClock(cfg)
	solarTicks := make(chan sim.Tick, 1)
	loadTicks := make(chan sim.Tick, 1)

	go func() {
		for tick := range clockCh {
			solarTicks <- tick
			loadTicks <- tick
		}
		close(solarTicks)
		close(loadTicks)
	}()

	solarCh := sim.RunSolar(solarTicks, cfg)
	loadCh := sim.RunLoad(loadTicks, cfg)

	logger := &sim.Logger{PolicyName: cfg.PolicyName}
	bat := sim.NewBatteryState(cfg.Battery)
	var cumCost float64

	totalSteps := cfg.TotalSteps()
	for i := 0; i < totalSteps; i++ {
		solarKW := <-solarCh
		loadKW := <-loadCh

		hour := float64(i%cfg.StepsPerDay()) * cfg.StepHours()
		tick := sim.Tick{
			Step:      i,
			Hour:      hour,
			Season:    cfg.Season,
			GridPrice: sim.GridPrice(hour, cfg.Season),
			NEGRate:   sim.NEGCreditRate,
			OnPeak:    sim.IsOnPeak(hour, cfg.Season),
		}

		decision := policyFn(tick, solarKW, loadKW, bat, cfg)
		decision, bat = sim.ApplyDecision(decision, bat, cfg)

		gridCost := (decision.GridToLoad + decision.GridToBattery) * cfg.StepHours() * tick.GridPrice
		negCredit := decision.SolarToGrid * cfg.StepHours() * tick.NEGRate
		cumCost += gridCost - negCredit

		logger.Record(sim.TickResult{
			Step:           i,
			Hour:           hour,
			GridPrice:      tick.GridPrice,
			OnPeak:         tick.OnPeak,
			SolarKW:        solarKW,
			LoadKW:         loadKW,
			BatterySOC:     bat.SOC(),
			BatteryKWh:     bat.ChargeKWh,
			SolarToLoad:    decision.SolarToLoad,
			SolarToBattery: decision.SolarToBattery,
			SolarToGrid:    decision.SolarToGrid,
			BatteryToLoad:  decision.BatteryToLoad,
			GridToLoad:     decision.GridToLoad,
			GridToBattery:  decision.GridToBattery,
			GridCost:       gridCost,
			NEGCredit:      negCredit,
			CumulativeCost: cumCost,
		})
	}

	path, err := logger.WriteCSV(cfg.OutputDir)
	if err != nil {
		return sim.Summary{}, err
	}
	fmt.Printf("  -> wrote %s (%d rows)\n", path, len(logger.Results))

	return logger.Summarize(cfg.StepHours()), nil
}
