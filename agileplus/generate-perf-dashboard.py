#!/usr/bin/env python3
"""
generate-perf-dashboard.py - Generates a self-contained HTML performance dashboard
from agileplus/perf-trend-history.json.

Reads the project's perf-trend-history.json, computes trend data (delta %, sparklines,
regressions), and writes docs/perf-dashboard.html with embedded Chart.js visualisation.

Usage:
    python agileplus/generate-perf-dashboard.py [--history PATH] [--config PATH] [--output PATH]

Defaults:
    --history  agileplus/perf-trend-history.json
    --config   agileplus/dashboard-config.json
    --output   docs/perf-dashboard.html
"""

from __future__ import annotations

import argparse
import json
import os
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------

DEFAULT_HISTORY = "agileplus/perf-trend-history.json"
DEFAULT_CONFIG = "agileplus/dashboard-config.json"
DEFAULT_OUTPUT = "docs/perf-dashboard.html"

WARN_THRESHOLD = 5.0
CRIT_THRESHOLD = 15.0

METRIC_LABELS: dict[str, str] = {
    "cold_startup_ms": "Cold Startup (ms)",
    "warm_command_ms": "Warm Command (ms)",
    "first_token_ms": "First Token (ms)",
    "memory_mb": "Memory (MB)",
    "binary_size_bytes": "Binary Size",
    "sandbox_creation_ms": "Sandbox Creation (ms)",
    "vm_boot_ms": "VM Boot (ms)",
    "api_response_ms": "API Response (ms)",
    "memory_usage_mb": "Memory Usage (MB)",
    "throughput_rps": "Throughput (req/s)",
    "cargo_check_secs": "cargo check (s)",
    "cargo_test_secs": "cargo test (s)",
    "clippy_warnings": "Clippy Warnings",
    "fuzz_corpus_size": "Fuzz Corpus Size",
    "dependency_count": "Dependency Count",
}

LOWER_IS_BETTER = {
    "cold_startup_ms", "warm_command_ms", "first_token_ms",
    "memory_mb", "binary_size_bytes", "sandbox_creation_ms",
    "vm_boot_ms", "api_response_ms", "memory_usage_mb",
    "cargo_check_secs", "cargo_test_secs", "clippy_warnings",
}

HIGHER_IS_BETTER = {"throughput_rps", "fuzz_corpus_size"}


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def load_json(path: Path) -> dict:
    with open(path, "r", encoding="utf-8") as fh:
        return json.load(fh)


def safe_project_name(history: dict) -> str:
    return history.get("project", Path.cwd().name)


def fmt_value(key: str, val: float) -> str:
    if "bytes" in key:
        if val >= 1_048_576:
            return "{:.2f} MB".format(val / 1_048_576)
        if val >= 1024:
            return "{:.1f} KB".format(val / 1024)
        return "{} B".format(int(val))
    if "ms" in key:
        return "{:,.0f} ms".format(val)
    if "secs" in key:
        return "{:,.1f} s".format(val)
    if "rps" in key:
        return "{:,.0f} req/s".format(val)
    if "mb" in key.lower():
        return "{:,.0f} MB".format(val)
    if isinstance(val, float) and val != int(val):
        return "{:,.2f}".format(val)
    return "{:,}".format(int(val))


def pct_delta(current: float, baseline: float) -> float | None:
    if baseline == 0:
        return None
    return ((current - baseline) / abs(baseline)) * 100.0


def delta_badge(delta: float | None, metric: str) -> str:
    if delta is None:
        return '<span class="badge badge-muted">N/A</span>'
    is_lower_better = metric in LOWER_IS_BETTER
    regression = delta > 0 if is_lower_better else delta < 0
    abs_d = abs(delta)
    cls = "badge-ok"
    if regression:
        if abs_d >= CRIT_THRESHOLD:
            cls = "badge-crit"
        elif abs_d >= WARN_THRESHOLD:
            cls = "badge-warn"
    else:
        if abs_d >= WARN_THRESHOLD:
            cls = "badge-improve"
    arrow = "\u25b2" if delta > 0 else ("\u25bc" if delta < 0 else "\u25cf")
    return '<span class="badge {}">{} {:.1f}%</span>'.format(cls, arrow, abs_d)


def short_sha(sha: str) -> str:
    return sha[:8] if len(sha) >= 8 else sha


def build_chart_data(history: dict) -> dict[str, Any]:
    """Build Chart.js datasets keyed by metric name."""
    runs = history.get("runs", [])
    if not runs:
        return {}

    all_keys: list[str] = []
    seen: set[str] = set()
    for run in runs:
        for k in run.get("metrics", {}):
            if k not in seen:
                all_keys.append(k)
                seen.add(k)

    labels = []
    datasets: dict[str, list[float]] = {k: [] for k in all_keys}

    for run in runs:
        ts = run.get("timestamp", "unknown")
        commit = run.get("commit_sha", "")
        labels.append(short_sha(commit) if commit != "initial" else ts[:10])
        metrics = run.get("metrics", {})
        for k in all_keys:
            datasets[k].append(metrics.get(k))

    return {"labels": labels, "datasets": datasets, "keys": all_keys}


# ---------------------------------------------------------------------------
# CSS (string concatenation avoids braces issue)
# ---------------------------------------------------------------------------

CSS = (
    ":root {\n"
    "  --bg: #0d1117; --surface: #161b22; --border: #30363d;\n"
    "  --text: #e6edf3; --muted: #8b949e; --accent: #58a6ff;\n"
    "  --ok: #3fb950; --warn: #d29922; --crit: #f85149; --improve: #58a6ff;\n"
    "}\n"
    "* { margin: 0; padding: 0; box-sizing: border-box; }\n"
    "body {\n"
    "  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif;\n"
    "  background: var(--bg); color: var(--text); padding: 2rem; line-height: 1.5;\n"
    "}\n"
    "h1 { font-size: 1.8rem; margin-bottom: .3rem; }\n"
    ".subtitle { color: var(--muted); font-size: .9rem; margin-bottom: 2rem; }\n"
    ".grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr)); gap: 1.2rem; margin-bottom: 2rem; }\n"
    ".card {\n"
    "  background: var(--surface); border: 1px solid var(--border); border-radius: 8px;\n"
    "  padding: 1.2rem; position: relative;\n"
    "}\n"
    ".card h3 { font-size: 1rem; margin-bottom: .8rem; }\n"
    ".card .value { font-size: 2rem; font-weight: 600; margin-bottom: .2rem; }\n"
    ".card .baseline { color: var(--muted); font-size: .85rem; }\n"
    ".chart-container {\n"
    "  background: var(--surface); border: 1px solid var(--border);\n"
    "  border-radius: 8px; padding: 1.5rem; margin-bottom: 1.5rem;\n"
    "}\n"
    ".chart-container h2 { font-size: 1.1rem; margin-bottom: 1rem; }\n"
    "canvas { max-height: 300px; }\n"
    ".badge {\n"
    "  display: inline-block; padding: 2px 8px; border-radius: 12px;\n"
    "  font-size: .75rem; font-weight: 600; margin-left: 8px;\n"
    "}\n"
    ".badge-ok { background: rgba(63,185,80,0.15); color: var(--ok); }\n"
    ".badge-warn { background: rgba(210,153,34,0.15); color: var(--warn); }\n"
    ".badge-crit { background: rgba(248,81,73,0.15); color: var(--crit); }\n"
    ".badge-improve { background: rgba(88,166,255,0.15); color: var(--improve); }\n"
    ".badge-muted { background: rgba(139,148,158,0.15); color: var(--muted); }\n"
    "table.summary {\n"
    "  width: 100%; border-collapse: collapse; margin-bottom: 2rem;\n"
    "  background: var(--surface); border: 1px solid var(--border); border-radius: 8px;\n"
    "  overflow: hidden;\n"
    "}\n"
    "table.summary th, table.summary td {\n"
    "  padding: .6rem 1rem; text-align: left; border-bottom: 1px solid var(--border);\n"
    "}\n"
    "table.summary th {\n"
    "  background: rgba(88,166,255,0.08); font-size: .85rem;\n"
    "  text-transform: uppercase; letter-spacing: .05em;\n"
    "}\n"
    "table.summary tr:last-child td { border-bottom: none; }\n"
    "table.summary tr:hover td { background: rgba(88,166,255,0.04); }\n"
    ".footer {\n"
    "  color: var(--muted); font-size: .8rem; margin-top: 2rem;\n"
    "  padding-top: 1rem; border-top: 1px solid var(--border);\n"
    "}\n"
)

JS_CHART_COLORS = [
    "#58a6ff", "#3fb950", "#d29922", "#f85149", "#bc8cff",
    "#79c0ff", "#56d364", "#e3b341", "#ff7b72", "#d2a8ff",
]


# ---------------------------------------------------------------------------
# HTML builder
# ---------------------------------------------------------------------------

def generate_html(project: str, history: dict, chart_data: dict) -> str:
    runs = history.get("runs", [])
    baselines = history.get("baselines", {})
    last_updated = history.get("last_updated", "unknown")
    schema_ver = history.get("schema_version", "N/A")
    config = history.get("config", {})
    retention = config.get("retention_days", 90)

    latest_run: dict = {}
    if runs:
        latest_run = runs[-1]

    latest_metrics = latest_run.get("metrics", {})
    keys = chart_data.get("keys", [])
    labels = chart_data.get("labels", [])
    datasets_raw = chart_data.get("datasets", {})

    # Summary table rows
    summary_rows = ""
    for k in keys:
        label = METRIC_LABELS.get(k, k)
        current = latest_metrics.get(k)
        baseline = baselines.get(k)
        current_str = fmt_value(k, current) if current is not None else "\u2014"
        baseline_str = fmt_value(k, baseline) if baseline is not None else "\u2014"
        delta = pct_delta(current, baseline) if current is not None and baseline is not None else None
        badge = delta_badge(delta, k)
        summary_rows += (
            '      <tr><td>{label}</td><td>{cur}</td><td>{base}</td><td>{badge}</td></tr>\n'
        ).format(label=label, cur=current_str, base=baseline_str, badge=badge)

    # KPI cards
    kpi_cards = ""
    for k in keys:
        label = METRIC_LABELS.get(k, k)
        current = latest_metrics.get(k)
        baseline = baselines.get(k)
        current_str = fmt_value(k, current) if current is not None else "\u2014"
        delta = pct_delta(current, baseline) if current is not None and baseline is not None else None
        badge = delta_badge(delta, k)
        baseline_str = "Baseline: {}".format(fmt_value(k, baseline)) if baseline is not None else ""
        kpi_cards += (
            '      <div class="card">\n'
            '        <h3>{label}{badge}</h3>\n'
            '        <div class="value">{current}</div>\n'
            '        <div class="baseline">{baseline}</div>\n'
            '      </div>\n'
        ).format(label=label, badge=badge, current=current_str, baseline=baseline_str)

    # Per-metric charts
    mini_charts = ""
    mini_chart_scripts = ""
    for idx, k in enumerate(keys):
        label = METRIC_LABELS.get(k, k)
        color = JS_CHART_COLORS[idx % len(JS_CHART_COLORS)]
        data = datasets_raw.get(k, [])
        canvas_id = "chart-{}".format(k.replace("_", "-"))

        mini_charts += (
            '    <div class="chart-container">\n'
            '      <h2>{label}</h2>\n'
            '      <canvas id="{cid}"></canvas>\n'
            '    </div>\n'
        ).format(label=label, cid=canvas_id)

        mini_chart_scripts += (
            "\n    new Chart(document.getElementById('{cid}'), {{\n"
            "      type: 'line',\n"
            "      data: {{\n"
            "        labels: {labels},\n"
            "        datasets: [{{\n"
            "          label: '{label}',\n"
            "          data: {data},\n"
            "          borderColor: '{color}',\n"
            "          backgroundColor: '{color}22',\n"
            "          tension: 0.3,\n"
            "          fill: true,\n"
            "          pointRadius: 5,\n"
            "          pointHoverRadius: 7,\n"
            "        }}]\n"
            "      }},\n"
            "      options: {{\n"
            "        responsive: true,\n"
            "        plugins: {{ legend: {{ display: false }} }},\n"
            "        scales: {{\n"
            "          x: {{ ticks: {{ color: '#8b949e' }}, grid: {{ color: '#30363d' }} }},\n"
            "          y: {{ ticks: {{ color: '#8b949e' }}, grid: {{ color: '#30363d' }} }}\n"
            "        }}\n"
            "      }}\n"
            "    }});\n"
        ).format(
            cid=canvas_id,
            labels=json.dumps(labels),
            label=label,
            data=json.dumps(data),
            color=color,
        )

    now_str = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")

    html = (
        '<!DOCTYPE html>\n'
        '<html lang="en">\n'
        '<head>\n'
        '<meta charset="UTF-8">\n'
        '<meta name="viewport" content="width=device-width, initial-scale=1.0">\n'
        '<title>{project} \u2014 Performance Dashboard</title>\n'
        '<script src="https://cdn.jsdelivr.net/npm/chart.js@4"></script>\n'
        '<style>\n{css}</style>\n'
        '</head>\n'
        '<body>\n'
        '<h1>{project} Performance Dashboard</h1>\n'
        '<div class="subtitle">\n'
        '  Schema v{schema} &middot; Last updated {last_updated} &middot; Retention {retention} days\n'
        '  &middot; Generated {now}\n'
        '</div>\n'
        '\n'
        '<!-- KPI Cards -->\n'
        '<div class="grid">\n{kpi_cards}</div>\n'
        '\n'
        '<!-- Summary Table -->\n'
        '<table class="summary">\n'
        '  <thead><tr><th>Metric</th><th>Latest</th><th>Baseline</th><th>Delta</th></tr></thead>\n'
        '  <tbody>\n{summary_rows}  </tbody>\n'
        '</table>\n'
        '\n'
        '<!-- Per-Metric Trend Charts -->\n{mini_charts}\n'
        '<div class="footer">\n'
        '  Auto-generated by <code>agileplus/generate-perf-dashboard.py</code> &middot;\n'
        '  Data source: <code>agileplus/perf-trend-history.json</code>\n'
        '</div>\n'
        '\n'
        '<script>\n'
        '(function() {{{mini_chart_scripts}\n}})();\n'
        '</script>\n'
        '</body>\n'
        '</html>\n'
    ).format(
        project=project,
        css=CSS,
        schema=schema_ver,
        last_updated=last_updated,
        retention=retention,
        now=now_str,
        kpi_cards=kpi_cards,
        summary_rows=summary_rows,
        mini_charts=mini_charts,
        mini_chart_scripts=mini_chart_scripts,
    )
    return html


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main() -> int:
    parser = argparse.ArgumentParser(
        description="Generate an HTML performance dashboard from perf-trend-history.json"
    )
    parser.add_argument(
        "--history", default=DEFAULT_HISTORY,
        help="Path to perf-trend-history.json (default: %(default)s)"
    )
    parser.add_argument(
        "--config", default=DEFAULT_CONFIG,
        help="Path to dashboard-config.json (default: %(default)s)"
    )
    parser.add_argument(
        "--output", default=DEFAULT_OUTPUT,
        help="Output HTML path (default: %(default)s)"
    )
    args = parser.parse_args()

    history_path = Path(args.history)
    config_path = Path(args.config)
    output_path = Path(args.output)

    if not history_path.exists():
        print("ERROR: {} not found".format(history_path), file=sys.stderr)
        return 1

    history = load_json(history_path)
    project = safe_project_name(history)

    # Optionally load dashboard-config.json for future enrichment
    dashboard_config = {}
    if config_path.exists():
        dashboard_config = load_json(config_path)

    chart_data = build_chart_data(history)
    html = generate_html(project, history, chart_data)

    output_path.parent.mkdir(parents=True, exist_ok=True)
    with open(output_path, "w", encoding="utf-8") as fh:
        fh.write(html)

    print("Dashboard written to {}".format(output_path.resolve()))
    runs_count = len(history.get("runs", []))
    metrics_count = len(chart_data.get("keys", []))
    print("  Project: {}".format(project))
    print("  Runs: {}  Metrics: {}".format(runs_count, metrics_count))
    return 0


if __name__ == "__main__":
    sys.exit(main())
