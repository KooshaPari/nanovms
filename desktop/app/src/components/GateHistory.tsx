import { useState, useEffect, useMemo } from "react";
import type { GateRun, IpcResponse } from "../types";

interface Props {
  request: <T = unknown>(channel: string, data?: unknown) => Promise<IpcResponse<T>>;
}

const CHART_WIDTH = 700;
const CHART_HEIGHT = 200;
const PADDING = { top: 20, right: 20, bottom: 40, left: 50 };

export default function GateHistory({ request }: Props) {
  const [runs, setRuns] = useState<GateRun[]>([]);

  useEffect(() => {
    request<GateRun[]>("gates:history").then((r) => {
      if (r.ok && Array.isArray(r.data)) setRuns(r.data);
    });
  }, [request]); // eslint-disable-line react-hooks/exhaustive-deps

  // Sort by date descending
  const sorted = useMemo(
    () => [...runs].sort((a, b) => new Date(b.runAt).getTime() - new Date(a.runAt).getTime()),
    [runs],
  );

  // Compute pass rate over time for chart
  const chartData = useMemo(() => {
    if (runs.length === 0) return [];

    // Group by date and compute daily pass rates
    const byDate = new Map<string, { pass: number; total: number }>();
    const sortedByDate = [...runs].sort(
      (a, b) => new Date(a.runAt).getTime() - new Date(b.runAt).getTime(),
    );

    for (const run of sortedByDate) {
      const dateKey = new Date(run.runAt).toISOString().slice(0, 10);
      const entry = byDate.get(dateKey) ?? { pass: 0, total: 0 };
      entry.total++;
      if (run.status === "pass") entry.pass++;
      byDate.set(dateKey, entry);
    }

    return Array.from(byDate.entries()).map(([date, { pass, total }]) => ({
      date,
      passRate: total > 0 ? Math.round((pass / total) * 100) : 0,
      pass,
      total,
    }));
  }, [runs]);

  const totalRuns = runs.length;
  const passRuns = runs.filter((r) => r.status === "pass").length;
  const failRuns = runs.filter((r) => r.status === "fail").length;
  const overallPassRate = totalRuns > 0 ? Math.round((passRuns / totalRuns) * 100) : 0;
  const avgDuration = totalRuns > 0
    ? Math.round(runs.reduce((s, r) => s + r.durationMs, 0) / totalRuns)
    : 0;

  // Chart calculations
  const chartInner = {
    width: CHART_WIDTH - PADDING.left - PADDING.right,
    height: CHART_HEIGHT - PADDING.top - PADDING.bottom,
  };

  const xScale = (i: number): number => {
    if (chartData.length <= 1) return PADDING.left + chartInner.width / 2;
    return PADDING.left + (i / (chartData.length - 1)) * chartInner.width;
  };

  const yScale = (val: number): number => {
    return PADDING.top + chartInner.height - (val / 100) * chartInner.height;
  };

  const pathD = chartData
    .map((d, i) => `${i === 0 ? "M" : "L"} ${xScale(i).toFixed(1)} ${yScale(d.passRate).toFixed(1)}`)
    .join(" ");

  const areaD = pathD +
    ` L ${xScale(chartData.length - 1).toFixed(1)} ${(PADDING.top + chartInner.height).toFixed(1)}` +
    ` L ${xScale(0).toFixed(1)} ${(PADDING.top + chartInner.height).toFixed(1)} Z`;

  const yTicks = [0, 25, 50, 75, 100];

  const showLabels = useMemo(() => {
    if (chartData.length <= 6) return chartData.map((d, i) => ({ index: i, label: d.date.slice(5) }));
    const step = Math.ceil(chartData.length / 6);
    return chartData
      .filter((_, i) => i % step === 0 || i === chartData.length - 1)
      .map((d, _i) => ({
        index: chartData.indexOf(d),
        label: d.date.slice(5),
      }));
  }, [chartData]);

  const formatDuration = (ms: number) => {
    if (ms < 1000) return `${ms}ms`;
    return `${(ms / 1000).toFixed(1)}s`;
  };

  return (
    <div className="gate-history">
      <div className="gate-history__header">
        <div>
          <h2 style={{ fontSize: 18, fontWeight: 700 }}>Gate Run History</h2>
          <p className="text-secondary" style={{ fontSize: 13, marginTop: 2 }}>
            Historical quality gate execution results
          </p>
        </div>
      </div>

      {/* Summary cards */}
      <div className="gate-history__summary">
        <div className="gate-history__summary-card">
          <div className="gate-history__summary-value" style={{ color: "var(--accent)" }}>{totalRuns}</div>
          <div className="gate-history__summary-label">Total Runs</div>
        </div>
        <div className="gate-history__summary-card">
          <div className="gate-history__summary-value" style={{ color: "var(--green)" }}>{passRuns}</div>
          <div className="gate-history__summary-label">Passing</div>
        </div>
        <div className="gate-history__summary-card">
          <div className="gate-history__summary-value" style={{ color: "var(--red)" }}>{failRuns}</div>
          <div className="gate-history__summary-label">Failing</div>
        </div>
        <div className="gate-history__summary-card">
          <div className="gate-history__summary-value" style={{ color: overallPassRate >= 80 ? "var(--green)" : overallPassRate >= 50 ? "var(--yellow)" : "var(--red)" }}>
            {overallPassRate}%
          </div>
          <div className="gate-history__summary-label">Pass Rate</div>
        </div>
        <div className="gate-history__summary-card">
          <div className="gate-history__summary-value">{formatDuration(avgDuration)}</div>
          <div className="gate-history__summary-label">Avg Duration</div>
        </div>
      </div>

      {/* Pass rate chart */}
      {chartData.length > 1 && (
        <div className="gate-history__chart">
          <h3 style={{ fontSize: 14, fontWeight: 600, color: "var(--text-secondary)", marginBottom: 8 }}>
            Pass Rate Over Time
          </h3>
          <div className="gate-history__chart-wrapper">
            <svg viewBox={`0 0 ${CHART_WIDTH} ${CHART_HEIGHT}`} className="gate-history__svg">
              {/* Grid lines */}
              {yTicks.map((tick) => (
                <g key={tick}>
                  <line
                    x1={PADDING.left}
                    y1={yScale(tick)}
                    x2={PADDING.left + chartInner.width}
                    y2={yScale(tick)}
                    className="gate-history__grid-line"
                  />
                  <text
                    x={PADDING.left - 8}
                    y={yScale(tick)}
                    className="gate-history__y-label"
                    textAnchor="end"
                    dominantBaseline="middle"
                  >
                    {tick}%
                  </text>
                </g>
              ))}

              {/* X-axis labels */}
              {showLabels.map(({ index, label }) => (
                <text
                  key={index}
                  x={xScale(index)}
                  y={PADDING.top + chartInner.height + 20}
                  className="gate-history__x-label"
                  textAnchor="middle"
                >
                  {label}
                </text>
              ))}

              {/* Area */}
              <path d={areaD} className="gate-history__area" />

              {/* Line */}
              <path d={pathD} className="gate-history__line" />

              {/* Data points */}
              {chartData.map((d, i) => (
                <circle
                  key={i}
                  cx={xScale(i)}
                  cy={yScale(d.passRate)}
                  r={3}
                  className="gate-history__dot"
                >
                  <title>{d.date}: {d.passRate}% ({d.pass}/{d.total})</title>
                </circle>
              ))}
            </svg>
          </div>
        </div>
      )}

      {/* Runs table */}
      <div className="gate-history__table-wrapper">
        <table className="gate-history__table">
          <thead>
            <tr>
              <th>Date</th>
              <th>Gate</th>
              <th>Status</th>
              <th>Duration</th>
              <th>Triggered By</th>
            </tr>
          </thead>
          <tbody>
            {sorted.length === 0 ? (
              <tr>
                <td colSpan={5} style={{ textAlign: "center", padding: 20, color: "var(--text-tertiary)" }}>
                  No gate runs recorded yet
                </td>
              </tr>
            ) : (
              sorted.map((run) => (
                <tr key={run.id}>
                  <td>{new Date(run.runAt).toLocaleString()}</td>
                  <td>{run.gateName}</td>
                  <td>
                    <span className={`gate-history__status gate-history__status--${run.status}`}>
                      {run.status}
                    </span>
                  </td>
                  <td>{formatDuration(run.durationMs)}</td>
                  <td className="text-secondary">{run.triggeredBy}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
