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


COLORS = {
    'Map': '#2196F3',
    'SyncMap': '#F44336',
    'Locked': '#4CAF50',
    'Ordered': '#FF9800',
    'LockedOrdered': '#9C27B0',
}

IMPL_ORDER = ['Map', 'SyncMap', 'Locked', 'Ordered', 'LockedOrdered']


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

    for ax, typ in zip(axes[0], types):
        type_data = [r for r in op_data if r['type'] == typ]

        for impl in IMPL_ORDER:
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
            ax.plot(sizes, avgs, 'o-', label=impl, color=color, linewidth=2, markersize=4)
            ax.fill_between(sizes, mins, maxs, alpha=0.12, color=color)

        ax.set_xscale('log')
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
