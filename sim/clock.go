package sim

// RunClock emits one Tick per simulated time step onto the returned channel,
// then closes it. Each tick carries the fractional hour, grid price, and
// on-peak flag derived from the season and Duke Energy TOU schedule.
func RunClock(cfg SimConfig) <-chan Tick {
	ch := make(chan Tick)
	go func() {
		defer close(ch)
		total := cfg.TotalSteps()
		for i := 0; i < total; i++ {
			hour := float64(i%cfg.StepsPerDay()) * cfg.StepHours()
			onPeak := IsOnPeak(hour, cfg.Season)
			ch <- Tick{
				Step:      i,
				Hour:      hour,
				Season:    cfg.Season,
				GridPrice: GridPrice(hour, cfg.Season),
				NEGRate:   NEGCreditRate,
				OnPeak:    onPeak,
			}
		}
	}()
	return ch
}
