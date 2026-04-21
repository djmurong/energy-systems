package sim

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
)

// Logger collects TickResults and writes them to CSV.
type Logger struct {
	Results    []TickResult
	PolicyName string
	Season     string
}

// Record appends a result to the log.
func (l *Logger) Record(r TickResult) {
	l.Results = append(l.Results, r)
}

// WriteCSV writes all results to a CSV file. The file is placed in a
// per-season subdirectory (<dir>/<season>/) and the season is embedded in
// the filename so summer and non-summer runs don't overwrite each other.
func (l *Logger) WriteCSV(dir string) (string, error) {
	seasonDir := filepath.Join(dir, l.Season)
	if err := os.MkdirAll(seasonDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(seasonDir, l.PolicyName+"-"+l.Season+".csv")
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	header := []string{
		"step", "hour", "grid_price", "on_peak",
		"solar_kw", "load_kw",
		"battery_soc", "battery_kwh",
		"solar_to_load", "solar_to_battery", "solar_to_grid",
		"battery_to_load", "grid_to_load", "grid_to_battery",
		"grid_cost", "neg_credit", "cumulative_cost",
	}
	if err := w.Write(header); err != nil {
		return "", err
	}

	for _, r := range l.Results {
		row := []string{
			fmt.Sprintf("%d", r.Step),
			fmt.Sprintf("%.4f", r.Hour),
			fmt.Sprintf("%.4f", r.GridPrice),
			fmt.Sprintf("%t", r.OnPeak),
			fmt.Sprintf("%.4f", r.SolarKW),
			fmt.Sprintf("%.4f", r.LoadKW),
			fmt.Sprintf("%.4f", r.BatterySOC),
			fmt.Sprintf("%.4f", r.BatteryKWh),
			fmt.Sprintf("%.4f", r.SolarToLoad),
			fmt.Sprintf("%.4f", r.SolarToBattery),
			fmt.Sprintf("%.4f", r.SolarToGrid),
			fmt.Sprintf("%.4f", r.BatteryToLoad),
			fmt.Sprintf("%.4f", r.GridToLoad),
			fmt.Sprintf("%.4f", r.GridToBattery),
			fmt.Sprintf("%.4f", r.GridCost),
			fmt.Sprintf("%.4f", r.NEGCredit),
			fmt.Sprintf("%.4f", r.CumulativeCost),
		}
		if err := w.Write(row); err != nil {
			return "", err
		}
	}
	return path, nil
}

// Summary holds aggregate stats for one policy run.
type Summary struct {
	PolicyName     string
	TotalCost      float64
	TotalNEG       float64
	NetCost        float64
	SolarProduced  float64
	SolarConsumed  float64
	SolarSpilled   float64
	SolarUtilPct   float64
	MinBatterySOC  float64
}

// Summarize computes aggregate statistics from the logged results.
func (l *Logger) Summarize(stepHours float64) Summary {
	s := Summary{PolicyName: l.PolicyName, MinBatterySOC: 1.0}
	for _, r := range l.Results {
		s.TotalCost += r.GridCost
		s.TotalNEG += r.NEGCredit
		s.SolarProduced += (r.SolarToLoad + r.SolarToBattery + r.SolarToGrid) * stepHours
		s.SolarConsumed += (r.SolarToLoad + r.SolarToBattery) * stepHours
		s.SolarSpilled += r.SolarToGrid * stepHours
		if r.BatterySOC < s.MinBatterySOC {
			s.MinBatterySOC = r.BatterySOC
		}
	}
	s.NetCost = s.TotalCost - s.TotalNEG
	if s.SolarProduced > 0 {
		s.SolarUtilPct = s.SolarConsumed / s.SolarProduced * 100
	}
	return s
}

// PrintSummaryTable prints a side-by-side comparison of multiple policy runs.
func PrintSummaryTable(summaries []Summary) {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                        POLICY COMPARISON SUMMARY                            ║")
	fmt.Println("╠═══════════════════╦═══════════╦═══════════╦═══════════╦═══════════╦══════════╣")
	fmt.Println("║ Policy            ║ Net Cost  ║ NEG Cred  ║ Solar Util║ Min SOC   ║ Spilled  ║")
	fmt.Println("╠═══════════════════╬═══════════╬═══════════╬═══════════╬═══════════╬══════════╣")
	for _, s := range summaries {
		fmt.Printf("║ %-17s ║ $%7.2f  ║ $%7.3f  ║ %6.1f%%   ║ %6.1f%%   ║ %5.2f kWh║\n",
			s.PolicyName, s.NetCost, s.TotalNEG, s.SolarUtilPct, s.MinBatterySOC*100, s.SolarSpilled)
	}
	fmt.Println("╚═══════════════════╩═══════════╩═══════════╩═══════════╩═══════════╩══════════╝")
	fmt.Println()
}
