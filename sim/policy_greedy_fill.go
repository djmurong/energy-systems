package sim

// GreedyFillPolicy charges the battery from the grid whenever power is cheap
// (off-peak), filling it to 100%. Solar is used for the load first, with
// excess sold to the grid.
//
// Known flaw: the battery is already full when solar production peaks midday,
// so solar energy spills to the grid at the low NEG rate ($0.03/kWh) instead
// of displacing $0.10–0.22/kWh consumption.
func GreedyFillPolicy(tick Tick, solarKW, loadKW float64, bat BatteryState, cfg SimConfig) PolicyDecision {
	var d PolicyDecision

	// Solar covers load first
	if solarKW >= loadKW {
		d.SolarToLoad = loadKW
		surplus := solarKW - loadKW
		// Try to charge battery with surplus
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
		deficit := loadKW - solarKW
		d.GridToLoad = deficit
	}

	// Off-peak: aggressively charge battery from grid to 100%
	if !tick.OnPeak {
		headroom := bat.CapacityKWh - bat.ChargeKWh
		canCharge := headroom / cfg.StepHours()
		if canCharge > cfg.Battery.MaxChargeKW-d.SolarToBattery {
			canCharge = cfg.Battery.MaxChargeKW - d.SolarToBattery
		}
		if canCharge < 0 {
			canCharge = 0
		}
		d.GridToBattery = canCharge
	}

	return d
}
