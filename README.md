# Electrotech: Energy Flow

A Go simulation of a home energy system — solar panels, household loads, a battery, and the utility grid — where each component runs as a goroutine communicating over channels.

The simulator compares three battery-management policies against Duke Energy's time-of-use rates for North Carolina solar customers, showing how each strategy affects cost, solar self-consumption, and grid dependence.

## Architecture

```
Clock ──Tick──▸ Solar ──┐
                        ├──▸ Policy ──Decision──▸ Battery ──▸ Logger (CSV)
Clock ──Tick──▸ Load  ──┘
```

The clock, solar, and load components each run as goroutines feeding a central coordination loop over channels:

- **Clock** drives fixed time steps (5-minute intervals, 288 ticks/day).
- **Solar** and **Load** goroutines produce supply/demand values each tick.
- The **coordinator** applies the policy, updates the battery, and logs each tick's result.

## Policies

| Policy | Strategy | Known Flaw |
|--------|----------|------------|
| **Greedy Cheap-Fill** | Charge battery from grid during off-peak ($0.098/kWh) | Battery full when solar arrives; solar spills at $0.03/kWh |
| **Greedy Discharge** | Drain battery to zero during on-peak ($0.224/kWh) | Violates backup reserve; vulnerable to outages |
| **Two-Threshold** | Maintain reserve floor (20%) and charge ceiling (80%) | Slightly more complex logic |

## Grid Rates

Duke Energy Schedule RSTC (NC Residential Solar TOU):

- **Off-peak:** $0.098/kWh
- **On-peak:** $0.224/kWh (summer 3–6 PM, non-summer 6–9 AM)
- **Net Excess Generation:** $0.03/kWh credit

## Usage

### Running the simulation

```bash
go run main.go                              # all policies, summer, 1 day
go run main.go --policy greedy-fill         # single policy
go run main.go --days 7 --season nonsummer  # week-long non-summer run
go run main.go --seed 123                   # reproducible randomness
go run main.go --output out/                # custom output directory
```

CSV output is organized by season so summer and non-summer runs don't overwrite each other:

```
results/
  summer/    greedy-fill-summer.csv, greedy-drain-summer.csv, two-threshold-summer.csv
  nonsummer/ greedy-fill-nonsummer.csv, greedy-drain-nonsummer.csv, two-threshold-nonsummer.csv
```

### Plotting

```bash
pip install -r scripts/requirements.txt
python scripts/plot.py results/summer --save      # → results/plots/summer/
python scripts/plot.py results/nonsummer --save   # → results/plots/nonsummer/
```

Drop `--save` to open the plots in interactive matplotlib windows instead.

## Results

### Summer (1-Day, Durham NC)

| Policy | Net Cost | NEG Credits | Solar Utilization | Min SOC | Solar Spilled |
|--------|----------|-------------|-------------------|---------|---------------|
| Greedy Cheap-Fill | $2.26 | $0.558 | 57.6% | 53.1% | 18.61 kWh |
| Greedy Discharge  | $2.63 | $0.836 | 36.6% | 16.6% | 27.86 kWh |
| Two-Threshold     | $1.62 | $0.477 | 63.8% | 53.1% | 15.91 kWh |

### Non-Summer (1-Day, Durham NC)

| Policy | Net Cost | NEG Credits | Solar Utilization | Min SOC | Solar Spilled |
|--------|----------|-------------|-------------------|---------|---------------|
| Greedy Cheap-Fill | $3.17 | $0.590 | 35.3% | 53.1% | 19.65 kWh |
| Greedy Discharge  | $2.30 | $0.572 | 37.0% | 33.7% | 19.06 kWh |
| Two-Threshold     | $2.03 | $0.455 | 50.0% | 30.6% | 15.17 kWh |

*Numbers above are from the default 1-day run (seed 42) via `go run main.go` and `go run main.go --season nonsummer`.*

**At a glance:** The cheaper *greedy* in each season is different. In **summer**, cheap-fill ($2.26) edges discharge by **$0.37**; in **non-summer**, discharge ($2.30) leads cheap-fill by **$0.87**. **Two-threshold** still has the lowest net cost in both — about **28%** below the next-best greedy in summer and **12%** in non-summer. (Percent savings: (next-best greedy net − two-threshold net) / next-best greedy net, with next-best = cheap-fill in summer and discharge in non-summer.)

## Conclusion

Each greedy baseline optimizes a single variable — cheap-fill minimizes the cost of *stored* energy, greedy-discharge minimizes the cost of *consumed* energy during peak — and each pays for that one-dimensional objective in a different season. **The ranking of the two greedy policies flips between summer and non-summer** (dollar-level gaps are in the “At a glance” note above). **Two-threshold wins in both.**

### Summer: peak overlaps with solar → hoarding loses

The on-peak window (3–6 PM) lands on the tail of the solar curve.

![Summer energy flows by policy](results/plots/summer/energy_flows.png)

- **Greedy cheap-fill** arrives at noon with a full battery from overnight grid-charging. No room for midday solar → **18.6 kWh spilled** at $0.03/kWh, and nothing stored to offset the 3–6 PM peak.
- **Greedy discharge** drains through the peak window and bottoms out at **16.6% SOC**. It also refuses to store solar it plans to dump, spilling the most of any policy — **27.9 kWh** and only 36.6% self-consumption.
- **Two-threshold** keeps 20% of overnight headroom, absorbs the midday solar, and empties into the peak without crashing the reserve floor.

![Summer cumulative cost](results/plots/summer/cumulative_cost.png)

All three policies track closely until the shaded on-peak band; that is where two-threshold pulls away. End-of-day net cost:

| Two-Threshold | Greedy Cheap-Fill | Greedy Discharge |
|:-:|:-:|:-:|
| **$1.62** | $2.26 (+40%) | $2.63 (+62%) |

The solar-utilization bars tell the same story from the supply side:

![Summer solar utilization](results/plots/summer/solar_utilization.png)

Two-threshold self-consumes **28.0 kWh** of solar; greedy discharge self-consumes only **16.1 kWh**, throwing away almost two-thirds of the array's output.

### Non-summer: peak precedes solar → draining is rational

The non-summer on-peak window (6–9 AM) sits *before* the sun comes up. This flips the greedy ranking.

![Non-summer energy flows by policy](results/plots/nonsummer/energy_flows.png)

- **Greedy discharge** is now reasonable. The battery covers the 7 AM heat-pump peak (blue wedge inside the pink band), and draining costs no solar-storage opportunity because solar hasn't arrived yet.
- **Greedy cheap-fill** is catastrophic. It never discharges aggressively, so the morning peak is served almost entirely from the grid at $0.224/kWh.

![Non-summer cumulative cost](results/plots/nonsummer/cumulative_cost.png)

Cheap-fill (red) jumps nearly a full dollar inside the shaded band and never recovers. The same policy that was mid-pack in summer is now the worst — purely because the peak window moved.

| Two-Threshold | Greedy Discharge | Greedy Cheap-Fill |
|:-:|:-:|:-:|
| **$2.03** | $2.30 (+13%) | $3.17 (+56%) |

Two-threshold still wins. It's the only policy that both covers the morning peak *and* stores the midday solar — self-consumption and spill reach an even 15.2 / 15.2 kWh split (50% utilization), compared to roughly 1:2 for both greedy variants.

### Why two-thresholds wins in every season

Two numbers doing two jobs, and neither requires forecasting:

- **Reserve floor (20%)** blocks the greedy-discharge failure — no draining to zero, no lost resilience, no forced grid imports later in the day.
- **Charge ceiling (80%)** blocks the greedy-cheap-fill failure — no filling overnight, no spilling tomorrow's solar.

The policy doesn't need to know when solar will arrive or how big the peak will be. It just refuses to commit the last 20% of capacity in either direction. That structural slack is enough to win in both seasons *for the same reason*: **leaving room is what lets the system absorb free resources and cover expensive ones.**

### Application to cloud infrastructure

The same failure modes appear in capacity planning and auto-scaling:

- **Over-provisioning ≡ greedy cheap-fill.** Pre-scaling to max capacity during cheap hours (spot overnight, reserved during low-traffic windows) leaves no room to absorb organic growth. The spike arrives, pre-committed capacity can't move, and new capacity is bought at on-demand rates — or goes unused. A full battery at solar noon.
- **Aggressive scale-down ≡ greedy discharge.** Draining to zero to minimize idle cost is the 0% SOC failure. A spike hits with no warm capacity; cold-start latency, pool exhaustion, and cascading timeouts are the distributed-systems analog of a dead battery during an outage.
- **Kubernetes HPA utilization bands, AWS ASG `min`/`max`, two-threshold.** These aren't arbitrary guardrails — they're the same structural constraints. The minimum absorbs bursts without degradation; the maximum bounds cost and rate-limit exposure; the band between them is *slack*.

The core lesson is the same in both domains: **the cost of maintaining idle capacity is almost always less than the cost of not having it when you need it.** A watt-hour spilled at $0.03 because the battery was full, an instance-hour wasted because the cluster was pre-scaled too aggressively — same failure mode. The two-threshold policy works because it treats slack as the resource that makes adaptation possible, not as waste.

## References

### Grid rates

Based on [Duke Energy Schedule RSTC](https://www.duke-energy.com/-/media/pdfs/for-your-home/rates/electric-nc/ncschedulers-tc.pdf) (NC Residential Solar Time-of-Use), effective January 2024. Off-peak $0.098/kWh (rounded from 9.7997¢), on-peak $0.224/kWh (rounded from 22.3842¢), peak windows summer 3–6 PM / non-summer 6–9 AM.

The NEG credit of $0.03/kWh is an estimate in the range of Duke's avoided-cost filings under Rider RSC (which varies quarterly) rather than a value from the RSTC tariff itself. *Simplifications:* every day is treated as a weekday; the $14/month basic customer charge, Critical Peak Pricing tier, and Discount Energy tier are not modeled.

### Solar insolation

Location (Durham, NC, 35.99°N, 78.90°W) and annual-average insolation magnitude (~5.0 kWh/m²/day) come from [NREL Typical Meteorological Year (TMY3) data](https://docs.nrel.gov/docs/fy08osti/43156.pdf); equivalent values can be reproduced with [NREL PVWatts](https://pvwatts.nrel.gov/).

The rest of the solar model is a parametric approximation, not a direct TMY3 lookup:

- Seasonal peak sun hours (5.5 summer, 3.8 non-summer) hand-chosen to bracket the annual average.
- Daily curve is a Gaussian centered at solar noon (~13:15 summer, ~12:30 non-summer) with σ = 3.0 h (summer) / 2.5 h (non-summer).
- ±15% per-tick noise simulates cloud cover.

### Household load

Parametric profile with magnitudes taken from published residential energy sources; time-of-day shapes (Gaussian HVAC curves, baseline day/night split, random spike frequencies) are modeling choices.

| Component | Value | Source |
|-----------|-------|--------|
| Baseline load (day / night) | 0.5 / 1.0 kW | [EIA RECS](https://www.eia.gov/consumption/residential/), [NREL End-Use Load Profiles](https://www.nrel.gov/buildings/end-use-load-profiles.html) |
| Summer AC peak (16:00) | 3.5 kW | NREL [ResStock Technical Reference](https://research-hub.nrel.gov/en/publications/resstock-technical-reference-documentation-v330/) — 3-ton central AC at SEER 13–16 |
| Non-summer heat peak (07:00) | 3.0 kW | DOE/NREL heat-pump data — 2–3 ton ASHP, Climate Zone 3 |
| EV charger | 7.2 kW @ 0.5%/tick | [SAE J1772 Level 2](https://www.sae.org/standards/content/j1772_201710/) (30 A × 240 V) |
| Electric oven | 3.0 kW @ 1%/tick | [ENERGY STAR electric cooking criteria](https://www.energystar.gov/products/electric_cooking_products/key_product_criteria) |
| Electric dryer | 5.5 kW @ 1%/tick | [ENERGY STAR clothes-dryer criteria](https://www.energystar.gov/products/clothes_dryers/key_product_criteria) (~5,480 W nameplate) |

### Battery

Core specs match the [Tesla Powerwall 2 datasheet](https://www.tesla.com/sites/default/files/pdfs/powerwall/Powerwall%202_AC_Datasheet_en_northamerica.pdf): 13.5 kWh usable capacity, 5.0 kW max continuous charge and discharge.

*Simulation choices* (not datasheet values): 50% initial SOC, 20% reserve floor, 80% grid-charge ceiling. *Simplifications:* 90% round-trip efficiency, 7 kW peak discharge, and the 3.3 kW backup-mode grid-charge limit are not modeled.

## Project Structure

```
main.go              Entry point
sim/
  types.go           Core data structures
  clock.go           Clock goroutine (time step driver)
  solar.go           PV panel goroutine
  load.go            Household load goroutine
  battery.go         Battery state + physics
  grid.go            Grid pricing logic
  policy.go          Policy interface + registry
  policy_greedy_fill.go
  policy_greedy_drain.go
  policy_twothreshold.go
  logger.go          CSV writer + summary stats
data/
  durham_solar.go    Durham, NC insolation curves (code-generated, no external data files)
scripts/
  plot.py            Matplotlib visualization
  requirements.txt
results/             Generated output (gitignored)
  <season>/          CSV per policy for that season
    greedy-fill-<season>.csv
    greedy-drain-<season>.csv
    two-threshold-<season>.csv
  plots/<season>/    PNG charts saved by scripts/plot.py --save
```

## Acknowledgments

Developed with AI assistance (Anthropic Claude) for code scaffolding, plotting scripts, and editing of the README and writeup. All modeling decisions (policy design, calibration targets, simulation parameters) and analysis of results are my own.