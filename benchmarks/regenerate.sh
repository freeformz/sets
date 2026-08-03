#!/usr/bin/env bash
# Regenerate the benchmark CSV and the charts embedded in the README.
#
# Usage:
#   ./benchmarks/regenerate.sh              # full run: benchmarks (slow!) + charts
#   ./benchmarks/regenerate.sh --plot-only  # re-plot charts from the existing CSV
#
# Environment:
#   BENCH_LARGE=1   also benchmark the 10M element size (much slower)
#
# The benchmark step runs TestBenchStats (bench_stats_test.go) across every set
# implementation registered in benchImpls (bench_test.go), so new implementations
# added there are picked up automatically. Expect the full run to take on the
# order of an hour on a fast machine.
set -euo pipefail

cd "$(dirname "$0")/.."

CSV=benchmarks/bench_stats.csv
CHARTS=benchmarks/bench_charts

# Prefer a python3 that already has matplotlib; otherwise fall back to uv,
# which provisions it on the fly.
if python3 -c 'import matplotlib' 2>/dev/null; then
    PLOT=(python3)
elif command -v uv >/dev/null; then
    PLOT=(uv run --with matplotlib python3)
else
    echo "error: need python3 with matplotlib (pip install matplotlib) or uv" >&2
    exit 1
fi

if [[ "${1:-}" != "--plot-only" ]]; then
    echo "==> Running statistical benchmarks (this takes a while)..."
    BENCH_STATS=1 BENCH_STATS_OUT="$CSV" \
        go test -v -run TestBenchStats -count=1 -timeout 180m .
fi

echo "==> Generating charts from $CSV..."
"${PLOT[@]}" benchmarks/bench_plot.py "$CSV" "$CHARTS"

echo "==> Done. Charts are in $CHARTS/ (referenced from README.MD)."
