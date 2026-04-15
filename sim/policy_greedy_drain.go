package sim

// GreedyDrainPolicy aggressively discharges the battery during on-peak hours
// to avoid buying expensive grid power. It tries to maximize battery discharge
// at the on-peak rate, even when solar could cover the load.
//
// Known flaw: drains the battery to 0% with no reserve floor, leaving the
// house exposed during outages or unexpected demand spikes.
func GreedyDrainPolicy(tick Tick, solarKW, loadKW float64, bat BatteryState, cfg SimConfig) PolicyDecision {
	var d PolicyDecision

	if tick.OnPeak {
		// On-peak: discharge battery as aggressively as possible.
		// Use battery for load first, then solar for whatever remains.
		available := bat.ChargeKWh / cfg.StepHours() // no reserve floor
		if available > cfg.Battery.MaxDischargeKW {
			available = cfg.Battery.MaxDischargeKW
		}
		if available >= loadKW {
			d.BatteryToLoad = loadKW
			// Solar surplus goes to grid or battery (battery is draining, so skip)
			d.SolarToGrid = solarKW
		} else {
			d.BatteryToLoad = available
			remaining := loadKW - available
			if solarKW >= remaining {
				d.SolarToLoad = remaining
				d.SolarToGrid = solarKW - remaining
			} else {
				d.SolarToLoad = solarKW
				d.GridToLoad = remaining - solarKW
			}
		}
	} else {
		// Off-peak: solar covers load, excess charges battery
		if solarKW >= loadKW {
			d.SolarToLoad = loadKW
			surplus := solarKW - loadKW
			headroom := bat.CapacityKWh - bat.ChargeKWh
			canCharge := headroom / cfg.StepHours()
			if canCharge > cfg.Battery.MaxChargeKW {
				canCharge = cfg.Battery.MaxChargeKW
			}
			if surplus <= canCharge {
				d.SolarToBattery = surplus
			} else {
				d.SolarToBattery = canCharge
				d.SolarToGrid = surplus - canCharge
			}
		} else {
			d.SolarToLoad = solarKW
			d.GridToLoad = loadKW - solarKW
		}

		// Charge from grid to 100% during off-peak
		headroom := bat.CapacityKWh - bat.ChargeKWh
		canCharge := headroom / cfg.StepHours()
		maxRemaining := cfg.Battery.MaxChargeKW - d.SolarToBattery
		if canCharge > maxRemaining {
			canCharge = maxRemaining
		}
		if canCharge < 0 {
			canCharge = 0
		}
		d.GridToBattery = canCharge
	}

	return d
}
