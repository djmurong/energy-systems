package sim

// TwoThresholdPolicy implements the smart energy management strategy:
//
//   - Never drain below X (MinReserve, e.g. 20% capacity) — backup reserve.
//   - Never grid-charge above Y (MaxCharge, e.g. 80% capacity) — leave
//     headroom for incoming solar.
//   - On-peak: discharge down to X before buying from the grid.
//   - Off-peak: charge from grid only up to Y, leaving room for midday solar.
//   - Solar is always prioritized: self-consume first, then battery (up to
//     100%, since solar charging has no Y ceiling), then spill to grid.
//
// The $0.03/kWh NEG rate makes solar spill nearly worthless, which is
// exactly why the headroom threshold Y matters.
func TwoThresholdPolicy(tick Tick, solarKW, loadKW float64, bat BatteryState, cfg SimConfig) PolicyDecision {
	var d PolicyDecision

	// --- Solar routing: self-consume → battery → grid spill ---
	if solarKW >= loadKW {
		d.SolarToLoad = loadKW
		surplus := solarKW - loadKW

		// Solar can charge battery all the way to 100% (no Y limit for solar)
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

		if tick.OnPeak {
			// Discharge battery down to MinReserve to avoid expensive grid power
			available := (bat.ChargeKWh - bat.MinReserve) / cfg.StepHours()
			if available < 0 {
				available = 0
			}
			if available > cfg.Battery.MaxDischargeKW {
				available = cfg.Battery.MaxDischargeKW
			}
			if available >= deficit {
				d.BatteryToLoad = deficit
			} else {
				d.BatteryToLoad = available
				d.GridToLoad = deficit - available
			}
		} else {
			d.GridToLoad = deficit
		}
	}

	// --- Off-peak grid charging: only up to Y (MaxCharge) ---
	if !tick.OnPeak {
		headroom := bat.MaxCharge - bat.ChargeKWh
		if headroom > 0 {
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
	}

	return d
}
