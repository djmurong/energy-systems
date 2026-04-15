package sim

// NewBatteryState creates the initial battery state from config.
func NewBatteryState(cfg BatteryConfig) BatteryState {
	return BatteryState{
		ChargeKWh:   cfg.InitialSOC * cfg.CapacityKWh,
		CapacityKWh: cfg.CapacityKWh,
		MinReserve:  cfg.MinReserve * cfg.CapacityKWh,
		MaxCharge:   cfg.MaxGridCharge * cfg.CapacityKWh,
	}
}

// ApplyDecision takes a policy decision and the current battery state,
// clamps all flows to physical limits (capacity bounds, charge/discharge
// rate limits), and returns the adjusted decision plus the new battery state.
func ApplyDecision(d PolicyDecision, bat BatteryState, cfg SimConfig) (PolicyDecision, BatteryState) {
	stepHrs := cfg.StepHours()
	maxChargeKW := cfg.Battery.MaxChargeKW
	maxDischargeKW := cfg.Battery.MaxDischargeKW

	// --- Clamp charging ---
	totalChargeKW := d.SolarToBattery + d.GridToBattery
	if totalChargeKW > maxChargeKW {
		ratio := maxChargeKW / totalChargeKW
		d.SolarToBattery *= ratio
		d.GridToBattery *= ratio
		totalChargeKW = maxChargeKW
	}
	chargeKWh := totalChargeKW * stepHrs
	headroom := bat.CapacityKWh - bat.ChargeKWh
	if chargeKWh > headroom {
		ratio := headroom / chargeKWh
		d.SolarToBattery *= ratio
		d.GridToBattery *= ratio
		chargeKWh = headroom
	}

	// --- Clamp discharging ---
	totalDischargeKW := d.BatteryToLoad
	if totalDischargeKW > maxDischargeKW {
		d.BatteryToLoad = maxDischargeKW
		totalDischargeKW = maxDischargeKW
	}
	dischargeKWh := totalDischargeKW * stepHrs
	available := bat.ChargeKWh - 0 // physical floor is 0; policy enforces MinReserve
	if dischargeKWh > available {
		if available <= 0 {
			d.BatteryToLoad = 0
			dischargeKWh = 0
		} else {
			ratio := available / dischargeKWh
			d.BatteryToLoad *= ratio
			dischargeKWh = available
		}
	}

	newCharge := bat.ChargeKWh + chargeKWh - dischargeKWh
	if newCharge < 0 {
		newCharge = 0
	}
	if newCharge > bat.CapacityKWh {
		newCharge = bat.CapacityKWh
	}

	newBat := bat
	newBat.ChargeKWh = newCharge
	return d, newBat
}
