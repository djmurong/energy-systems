#!/usr/bin/env python3
"""
Generate comparison charts from Electrotech Energy Flow simulation CSV output.

Usage:
    python scripts/plot.py results/          # default output directory
    python scripts/plot.py results/ --save   # save PNGs instead of showing
"""

import sys
import os
import glob
import pandas as pd
import matplotlib.pyplot as plt
import matplotlib.ticker as mticker
from matplotlib.patches import Patch

POLICY_COLORS = {
    "greedy-fill":   "#e74c3c",
    "greedy-drain":  "#f39c12",
    "two-threshold": "#2ecc71",
}

POLICY_LABELS = {
    "greedy-fill":   "Greedy Cheap-Fill",
    "greedy-drain":  "Greedy Discharge",
    "two-threshold": "Two-Threshold",
}


def load_data(results_dir):
    """Load all CSV files from the results directory."""
    frames = {}
    for path in sorted(glob.glob(os.path.join(results_dir, "*.csv"))):
        name = os.path.splitext(os.path.basename(path))[0]
        df = pd.read_csv(path)
        frames[name] = df
    return frames


def plot_energy_flows(frames, save_dir=None):
    """Stacked area chart of energy sources serving load for each policy."""
    n = len(frames)
    fig, axes = plt.subplots(n, 1, figsize=(14, 4 * n), sharex=True)
    if n == 1:
        axes = [axes]

    for ax, (name, df) in zip(axes, frames.items()):
        hours = df["hour"]
        ax.fill_between(hours, 0, df["solar_to_load"],
                        alpha=0.7, color="#f1c40f", label="Solar → Load")
        ax.fill_between(hours, df["solar_to_load"],
                        df["solar_to_load"] + df["battery_to_load"],
                        alpha=0.7, color="#3498db", label="Battery → Load")
        ax.fill_between(hours, df["solar_to_load"] + df["battery_to_load"],
                        df["solar_to_load"] + df["battery_to_load"] + df["grid_to_load"],
                        alpha=0.7, color="#e74c3c", label="Grid → Load")
        ax.plot(hours, df["load_kw"], color="black", linewidth=1.2,
                linestyle="--", label="Total Load")

        # Shade on-peak hours
        on_peak = df[df["on_peak"] == True]
        if not on_peak.empty:
            start = on_peak["hour"].iloc[0]
            end = on_peak["hour"].iloc[-1]
            ax.axvspan(start, end, alpha=0.08, color="red")
            ax.text(start + 0.1, ax.get_ylim()[1] * 0.9, "ON-PEAK",
                    fontsize=8, color="red", alpha=0.7)

        ax.set_ylabel("Power (kW)")
        ax.set_title(f"{POLICY_LABELS.get(name, name)}", fontsize=12, fontweight="bold")
        ax.legend(loc="upper left", fontsize=8)
        ax.set_xlim(0, 24)
        ax.xaxis.set_major_locator(mticker.MultipleLocator(3))

    axes[-1].set_xlabel("Hour of Day")
    fig.suptitle("Energy Flow by Source", fontsize=14, fontweight="bold", y=1.01)
    fig.tight_layout()
    _save_or_show(fig, save_dir, "energy_flows.png")


def plot_battery_soc(frames, save_dir=None):
    """Battery state of charge over time for all policies."""
    fig, ax = plt.subplots(figsize=(14, 5))

    for name, df in frames.items():
        color = POLICY_COLORS.get(name, "gray")
        label = POLICY_LABELS.get(name, name)
        ax.plot(df["hour"], df["battery_soc"] * 100, color=color,
                linewidth=2, label=label)

    # Threshold lines
    ax.axhline(y=20, color="red", linestyle=":", linewidth=1.5,
               alpha=0.7, label="Min Reserve (X=20%)")
    ax.axhline(y=80, color="blue", linestyle=":", linewidth=1.5,
               alpha=0.7, label="Max Grid Charge (Y=80%)")

    # On-peak shading
    first_df = list(frames.values())[0]
    on_peak = first_df[first_df["on_peak"] == True]
    if not on_peak.empty:
        ax.axvspan(on_peak["hour"].iloc[0], on_peak["hour"].iloc[-1],
                   alpha=0.08, color="red")

    ax.set_xlabel("Hour of Day")
    ax.set_ylabel("Battery SOC (%)")
    ax.set_title("Battery State of Charge", fontsize=14, fontweight="bold")
    ax.set_xlim(0, 24)
    ax.set_ylim(-5, 105)
    ax.xaxis.set_major_locator(mticker.MultipleLocator(3))
    ax.legend(loc="best", fontsize=9)
    ax.grid(True, alpha=0.3)
    fig.tight_layout()
    _save_or_show(fig, save_dir, "battery_soc.png")


def plot_cumulative_cost(frames, save_dir=None):
    """Cumulative cost over time for all policies."""
    fig, ax = plt.subplots(figsize=(14, 5))

    for name, df in frames.items():
        color = POLICY_COLORS.get(name, "gray")
        label = POLICY_LABELS.get(name, name)
        ax.plot(df["hour"], df["cumulative_cost"], color=color,
                linewidth=2, label=label)

    first_df = list(frames.values())[0]
    on_peak = first_df[first_df["on_peak"] == True]
    if not on_peak.empty:
        ax.axvspan(on_peak["hour"].iloc[0], on_peak["hour"].iloc[-1],
                   alpha=0.08, color="red")

    ax.set_xlabel("Hour of Day")
    ax.set_ylabel("Cumulative Cost ($)")
    ax.set_title("Cumulative Grid Cost Over Time", fontsize=14, fontweight="bold")
    ax.set_xlim(0, 24)
    ax.xaxis.set_major_locator(mticker.MultipleLocator(3))
    ax.legend(loc="best", fontsize=10)
    ax.grid(True, alpha=0.3)
    ax.yaxis.set_major_formatter(mticker.FormatStrFormatter("$%.2f"))
    fig.tight_layout()
    _save_or_show(fig, save_dir, "cumulative_cost.png")


def plot_solar_utilization(frames, step_hours, save_dir=None):
    """Bar chart of solar self-consumed vs spilled per policy."""
    policies = []
    consumed = []
    spilled = []

    for name, df in frames.items():
        total_solar = (df["solar_to_load"] + df["solar_to_battery"] + df["solar_to_grid"]).sum() * step_hours
        self_consumed = (df["solar_to_load"] + df["solar_to_battery"]).sum() * step_hours
        grid_spill = df["solar_to_grid"].sum() * step_hours
        policies.append(POLICY_LABELS.get(name, name))
        consumed.append(self_consumed)
        spilled.append(grid_spill)

    x = range(len(policies))
    width = 0.35

    fig, ax = plt.subplots(figsize=(10, 6))
    bars1 = ax.bar([i - width / 2 for i in x], consumed, width,
                   label="Self-Consumed", color="#2ecc71", alpha=0.85)
    bars2 = ax.bar([i + width / 2 for i in x], spilled, width,
                   label="Spilled to Grid (NEG)", color="#e74c3c", alpha=0.85)

    ax.set_ylabel("Energy (kWh)")
    ax.set_title("Solar Utilization by Policy", fontsize=14, fontweight="bold")
    ax.set_xticks(list(x))
    ax.set_xticklabels(policies, fontsize=11)
    ax.legend(fontsize=10)
    ax.grid(True, axis="y", alpha=0.3)

    for bar in bars1:
        h = bar.get_height()
        ax.text(bar.get_x() + bar.get_width() / 2, h + 0.3,
                f"{h:.1f}", ha="center", va="bottom", fontsize=9)
    for bar in bars2:
        h = bar.get_height()
        ax.text(bar.get_x() + bar.get_width() / 2, h + 0.3,
                f"{h:.1f}", ha="center", va="bottom", fontsize=9)

    fig.tight_layout()
    _save_or_show(fig, save_dir, "solar_utilization.png")


def plot_grid_price_overlay(frames, save_dir=None):
    """Grid price schedule with energy buying/selling activity."""
    first_name, first_df = list(frames.items())[0]
    fig, ax1 = plt.subplots(figsize=(14, 5))

    ax1.fill_between(first_df["hour"], 0, first_df["grid_price"] * 100,
                     alpha=0.15, color="gray")
    ax1.plot(first_df["hour"], first_df["grid_price"] * 100,
             color="gray", linewidth=1.5, linestyle="--", label="Grid Price")
    ax1.set_ylabel("Grid Price (¢/kWh)", color="gray")
    ax1.set_ylim(0, 30)

    ax2 = ax1.twinx()
    for name, df in frames.items():
        color = POLICY_COLORS.get(name, "gray")
        label = POLICY_LABELS.get(name, name)
        net_grid = df["grid_to_load"] + df["grid_to_battery"] - df["solar_to_grid"]
        ax2.plot(df["hour"], net_grid, color=color, linewidth=1.5, label=label)

    ax2.set_ylabel("Net Grid Draw (kW)")
    ax2.axhline(y=0, color="black", linewidth=0.5, alpha=0.5)

    lines1, labels1 = ax1.get_legend_handles_labels()
    lines2, labels2 = ax2.get_legend_handles_labels()
    ax2.legend(lines1 + lines2, labels1 + labels2, loc="upper left", fontsize=9)

    ax1.set_xlabel("Hour of Day")
    ax1.set_xlim(0, 24)
    ax1.xaxis.set_major_locator(mticker.MultipleLocator(3))
    ax1.set_title("Grid Interaction vs. Price Schedule",
                  fontsize=14, fontweight="bold")
    fig.tight_layout()
    _save_or_show(fig, save_dir, "grid_price_overlay.png")


def _save_or_show(fig, save_dir, filename):
    if save_dir:
        os.makedirs(save_dir, exist_ok=True)
        path = os.path.join(save_dir, filename)
        fig.savefig(path, dpi=150, bbox_inches="tight")
        print(f"  saved {path}")
        plt.close(fig)
    else:
        plt.show()


def main():
    if len(sys.argv) < 2:
        print("Usage: python plot.py <results_dir> [--save]")
        sys.exit(1)

    results_dir = sys.argv[1]
    save = "--save" in sys.argv
    save_dir = os.path.join(results_dir, "plots") if save else None

    frames = load_data(results_dir)
    if not frames:
        print(f"No CSV files found in {results_dir}")
        sys.exit(1)

    print(f"Loaded {len(frames)} policy results: {', '.join(frames.keys())}")

    # Infer step duration from first CSV
    first_df = list(frames.values())[0]
    if len(first_df) > 1:
        step_hours = first_df["hour"].iloc[1] - first_df["hour"].iloc[0]
    else:
        step_hours = 5.0 / 60.0

    plot_energy_flows(frames, save_dir)
    plot_battery_soc(frames, save_dir)
    plot_cumulative_cost(frames, save_dir)
    plot_solar_utilization(frames, step_hours, save_dir)
    plot_grid_price_overlay(frames, save_dir)

    if save:
        print(f"\nAll plots saved to {save_dir}/")
    else:
        print("Close plot windows to exit.")


if __name__ == "__main__":
    main()
