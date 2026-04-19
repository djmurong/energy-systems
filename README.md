# Electrotech: Energy Flow

A Go simulation of a home energy system where each component — solar panels, household loads, a battery, and the utility grid — runs as a goroutine communicating over channels.

The simulator compares three battery management policies against real Duke Energy time-of-use rates for North Carolina solar customers, showing how different strategies affect cost, solar utilization, and grid dependence.

## Architecture

```
Clock ──Tick──▸ Solar ──┐
                        ├──▸ Policy ──Decision──▸ Battery ──▸ Logger (CSV)
Clock ──Tick──▸ Load  ──┘
```

The clock, solar, and load components each run as goroutines, feeding a central coordination loop over channels. The clock drives fixed time steps (5-minute intervals, 288 ticks/day). Solar and load goroutines produce supply/demand values each tick. The coordinator applies the policy, updates the battery, and logs each result.

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

```bash
go run main.go                           # run all policies, summer, 1 day
go run main.go --policy greedy-fill      # single policy
go run main.go --days 7 --season nonsummer  # week-long non-summer run
go run main.go --seed 123               # reproducible randomness
go run main.go --output out/            # custom output directory
```

Results are written to `results/` as CSV files.

### Plotting

```bash
pip install -r scripts/requirements.txt
python scripts/plot.py results/
```

## Results

### Summer (1-Day, Durham NC)

| Policy | Net Cost | NEG Credits | Solar Utilization | Min SOC | Solar Spilled |
|--------|----------|-------------|-------------------|---------|---------------|
| Greedy Cheap-Fill | $3.12 | $0.457 | 65.3% | 53.1% | 15.23 kWh |
| Greedy Discharge | $3.28 | $0.745 | 43.5% | 0.0% | 24.83 kWh |
| Two-Threshold | $2.18 | $0.374 | 71.6% | 53.1% | 12.48 kWh |

### Non-Summer (1-Day, Durham NC)

| Policy | Net Cost | NEG Credits | Solar Utilization | Min SOC | Solar Spilled |
|--------|----------|-------------|-------------------|---------|---------------|
| Greedy Cheap-Fill | $3.70 | $0.519 | 43.0% | 53.1% | 17.31 kWh |
| Greedy Discharge | $2.71 | $0.499 | 45.0% | 27.0% | 16.64 kWh |
| Two-Threshold | $2.44 | $0.384 | 57.8% | 23.9% | 12.81 kWh |

## Conclusion

The simulation reveals that naive optimization of a single variable -- cost of stored energy or cost of consumed energy -- consistently underperforms a policy that respects the structure of the problem. The greedy cheap-fill policy minimizes the per-kWh cost of what goes *into* the battery, but by filling to 100% overnight it leaves no room for free solar energy at midday, forcing 15-25 kWh of solar to spill to the grid at the near-worthless $0.03/kWh NEG rate. The greedy discharge policy minimizes the per-kWh cost of what comes *out* of the battery during peak hours, but it drains to 0% SOC, sacrificing resilience entirely and wasting solar on grid export instead of self-consumption. Under Duke Energy's summer TOU schedule, the greedy discharge policy actually costs *more* ($3.28) than greedy cheap-fill ($3.12) because it exports solar during on-peak hours at $0.03/kWh rather than using it directly to displace $0.224/kWh grid power.

The two-threshold policy outperforms both by encoding two structural constraints: a reserve floor (20%) that guarantees backup capacity, and a charge ceiling (80%) that preserves headroom for incoming solar. These thresholds reduce net cost by 30% over greedy cheap-fill in summer ($2.18 vs. $3.12) and push solar utilization from 65% to 72%. The improvement comes not from better prediction or more computation, but from *leaving room* -- maintaining slack in the system so that cheap or free resources can be absorbed when they arrive.

### Application to Cloud Infrastructure

These findings map directly to capacity planning and auto-scaling in distributed systems:

**Over-provisioning is the greedy cheap-fill problem.** Pre-scaling to maximum capacity during cheap hours (spot instances overnight, reserved capacity during low-traffic windows) means the system has no room to absorb organic load growth. When the actual traffic spike arrives, the pre-provisioned resources are already committed and new capacity must be acquired at premium on-demand rates -- or worse, the surplus is wasted. This is the cloud equivalent of a full battery at solar noon.

**Aggressive scale-down is the greedy discharge problem.** Draining capacity to zero during off-peak to minimize idle cost is the auto-scaling equivalent of the 0% SOC failure. When an unexpected traffic spike hits, there is no warm capacity to absorb it. Cold-start latency, connection pool exhaustion, and cascading timeouts are the distributed systems analogs of a dead battery during a grid outage. The system optimized for cost at the expense of resilience.

**The two-threshold approach is how production auto-scalers actually work.** Kubernetes Horizontal Pod Autoscaler maintains a target utilization band (typically 50-80%) rather than scaling to exactly match demand. AWS Auto Scaling groups define minimum and maximum instance counts. These aren't arbitrary guardrails -- they are the same structural constraints as the battery's reserve floor and charge ceiling. The minimum guarantees the system can absorb a burst without latency degradation. The maximum prevents runaway scaling from consuming budget or hitting API rate limits. The band between them is *slack* -- the capacity to absorb variance without reactive intervention.

The core lesson is the same in both domains: the cost of maintaining idle capacity is almost always less than the cost of not having it when you need it. A watt-hour of solar energy spilled at $0.03 because the battery was full, an instance-hour of compute wasted because the cluster was pre-scaled too aggressively -- these are the same failure mode. The two-threshold policy works because it treats slack not as waste, but as the resource that makes adaptation possible.

## References

**Grid rates** -- [Duke Energy Schedule RSTC](https://www.duke-energy.com/-/media/pdfs/for-your-home/rates/electric-nc/ncschedulers-tc.pdf) (NC Residential Solar Time-of-Use), effective January 2024. Off-peak $0.098/kWh, on-peak $0.224/kWh, net excess generation credit $0.03/kWh. Peak windows: summer 3--6 PM, non-summer 6--9 AM weekdays. The simulation treats every day as a weekday.

**Solar insolation** -- Durham, NC (35.99°N, 78.90°W) parameters derived from [NREL Typical Meteorological Year (TMY3) data](https://docs.nrel.gov/docs/fy08osti/43156.pdf), validated against [NREL PVWatts](https://pvwatts.nrel.gov/). Peak sun hours: 5.5 kWh/m²/day (summer), 3.8 kWh/m²/day (non-summer). The daily irradiance curve is modeled as a Gaussian centered at solar noon (~13:15 summer with DST, ~12:30 non-summer) with sigma 3.0h (summer) / 2.5h (non-summer) to approximate the daylight window. ±15% per-tick noise simulates cloud cover.

**Household load** -- Synthetic profile based on typical US residential consumption patterns. Baseline 0.8 kW (daytime) / 1.2 kW (evening/night) represents always-on loads (refrigerator, lighting, electronics). HVAC modeled as a Gaussian curve: summer AC peaks at 4.0 kW around 4 PM, non-summer heating peaks at 3.0 kW around 7 AM. Random appliance spikes per tick: EV charger 7.2 kW (0.5% chance), oven 2.5 kW (1%), dryer 5.0 kW (1%).

**Battery** -- Modeled after the [Tesla Powerwall 2](https://www.tesla.com/sites/default/files/pdfs/powerwall/Powerwall%202_AC_Datasheet_en_northamerica.pdf): 13.5 kWh usable capacity, 5.0 kW max continuous charge/discharge rate. Initial SOC 50%. Policy thresholds: 20% reserve floor, 80% grid-charge ceiling.

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
```
