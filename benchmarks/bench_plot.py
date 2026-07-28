#!/usr/bin/env python3
"""Generate benchmark charts from bench_stats.csv.

Usage:
    python3 bench_plot.py <bench_stats.csv> [output_dir]

Produces one PNG per operation, with subplots for int and string element types.
Each line represents a set implementation, showing average operation time.
The shaded region shows the min-max range across samples.
"""

import csv
import sys
import os
from collections import defaultdict

import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
import matplotlib.ticker as ticker


# Colorblind-safe categorical palette (validated: adjacent-pair CVD deltaE >= 8,
# normal-vision deltaE >= 15, all slots within the lightness band on a light surface).
# Each series also gets a distinct marker so identity never rides on color alone.
COLORS = {
    'Map': '#2a78d6',            # blue
    'SyncMap': '#eb6834',        # orange
    'Locked': '#1baf7a',         # aqua
    'Ordered': '#eda100',        # yellow
    'SortedSet': '#e87ba4',      # magenta
    'LockedOrdered': '#008300',  # green
    'BitSet': '#4a3aa7',         # violet
}

MARKERS = {
    'Map': 'o',
    'SyncMap': 's',
    'Locked': '^',
    'Ordered': 'D',
    'SortedSet': 'v',
    'LockedOrdered': 'P',
    'BitSet': 'X',
}

# Matches the declaration order of benchImpls in bench_test.go.
IMPL_ORDER = ['Map', 'SyncMap', 'Locked', 'Ordered', 'SortedSet', 'LockedOrdered', 'BitSet']


def read_csv(path):
    data = []
    with open(path) as f:
        reader = csv.DictReader(f)
        for row in reader:
            row['size'] = int(row['size'])
            for k in ['min', 'max', 'avg', 'stddev', 'p50', 'p95', 'p99']:
                row[k] = float(row[k])
            data.append(row)
    return data


def format_ns(ns):
    """Format nanoseconds into a human-readable string."""
    if ns >= 1e9:
        return f'{ns/1e9:.1f}s'
    if ns >= 1e6:
        return f'{ns/1e6:.1f}ms'
    if ns >= 1e3:
        return f'{ns/1e3:.1f}µs'
    return f'{ns:.1f}ns'


def plot_operation(op, op_data, output_dir):
    types = sorted(set(r['type'] for r in op_data))
    unit = op_data[0]['unit']

    fig, axes = plt.subplots(1, len(types), figsize=(7 * len(types), 6), sharey=True, squeeze=False)
    fig.suptitle(op, fontsize=16, fontweight='bold')

    # Plot known impls in fixed order, then any impls present in the data but
    # missing from IMPL_ORDER (never silently drop a series).
    data_impls = set(r['impl'] for r in op_data)
    extra_impls = sorted(data_impls - set(IMPL_ORDER))
    if extra_impls:
        print(f'  WARNING: impls not in IMPL_ORDER (plotted in black): {", ".join(extra_impls)}')

    for ax, typ in zip(axes[0], types):
        type_data = [r for r in op_data if r['type'] == typ]

        for impl in IMPL_ORDER + extra_impls:
            impl_data = sorted(
                [r for r in type_data if r['impl'] == impl],
                key=lambda r: r['size'],
            )
            if not impl_data:
                continue

            sizes = [r['size'] for r in impl_data]
            avgs = [r['avg'] for r in impl_data]
            mins = [r['min'] for r in impl_data]
            maxs = [r['max'] for r in impl_data]

            color = COLORS.get(impl, '#000000')
            marker = MARKERS.get(impl, 'o')
            ax.plot(sizes, avgs, label=impl, color=color, linewidth=2,
                    marker=marker, markersize=5)
            ax.fill_between(sizes, mins, maxs, alpha=0.12, color=color)

        # log-log: series span many orders of magnitude (word-wise BitSet ops vs
        # element-wise iteration); a linear y-axis flattens all but the slowest.
        ax.set_xscale('log')
        ax.set_yscale('log')
        ax.set_xlabel('Set Size')
        ax.set_title(f'element type: {typ}')
        ax.grid(True, alpha=0.3, which='both')
        ax.legend(fontsize=9)

        # Format x-axis
        ax.xaxis.set_major_formatter(ticker.FuncFormatter(
            lambda x, p: f'{int(x):,}' if x >= 1 else str(x)
        ))
        ax.tick_params(axis='x', rotation=45)

        # Format y-axis with human-readable units
        ax.yaxis.set_major_formatter(ticker.FuncFormatter(
            lambda x, p: format_ns(x)
        ))

    axes[0][0].set_ylabel(unit)

    plt.tight_layout()
    path = os.path.join(output_dir, f'{op}.png')
    plt.savefig(path, dpi=150, bbox_inches='tight')
    plt.close()
    print(f'  {op}.png')


def main():
    if len(sys.argv) < 2:
        print('Usage: python3 bench_plot.py <bench_stats.csv> [output_dir]')
        sys.exit(1)

    csv_path = sys.argv[1]
    output_dir = sys.argv[2] if len(sys.argv) > 2 else 'bench_charts'
    os.makedirs(output_dir, exist_ok=True)

    data = read_csv(csv_path)
    ops = sorted(set(r['operation'] for r in data))

    print(f'Generating charts in {output_dir}/:')
    for op in ops:
        op_data = [r for r in data if r['operation'] == op]
        plot_operation(op, op_data, output_dir)

    print(f'\nDone — {len(ops)} charts saved to {output_dir}/')


if __name__ == '__main__':
    main()
