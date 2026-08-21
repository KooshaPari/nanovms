import { useState, useMemo, useCallback } from "react";
import type { TrendPoint, IpcResponse } from "../types";

interface Props {
  request: <T = unknown>(channel: string, data?: unknown) => Promise<IpcResponse<T>>;
}

// ─── 31 Pillar IDs and labels ──────────────────────────────────────────────
const PILLAR_LABELS: Record<string, string> = {
  "perf-boot": "Boot Time",
  "perf-throughput": "Throughput",
  "perf-latency-p99": "P99 Latency",
  "perf-memory": "Memory Efficiency",
  "perf-io": "I/O Performance",
  "perf-cpu": "CPU Utilization",
  "perf-network": "Network Stack",
  "rel-uptime": "Uptime / Availability",
  "rel-crash": "Crash Rate",
  "rel-recovery": "Error Recovery",
  "rel-data": "Data Integrity",
  "rel-timeout": "Timeout Handling",
  "rel-health": "Health Checks",
  "sec-auth": "Authentication",
  "sec-crypto": "Cryptographic Ops",
  "sec-isolation": "Isolation / Sandbox",
  "sec-secrets": "Secret Management",
  "sec-audit": "Audit Trail",
  "sec-vuln": "Vulnerability Posture",
  "dx-api": "API Design",
  "dx-docs": "Documentation",
  "dx-cli": "CLI / Tooling",
  "dx-debug": "Debugging",
  "dx-testing": "Test Coverage",
  "dx-contrib": "Contributor Experience",
  "ops-deploy": "Deployment",
  "ops-monitor": "Observability",
  "ops-scaling": "Auto-Scaling",
  "ops-config": "Configuration",
  "ops-migrate": "Migration Path",
  "ops-cost": "Cost Efficiency",
};

const CHART_WIDTH = 700;
const CHART_HEIGHT = 300;
const PADDING = { top: 20, right: 20, bottom: 40, left: 50 };

const COLORS = [
  "#1f6feb", "#3fb950", "#f85149", "#d29922", "#e3b341",
  "#a371f7", "#f0883e", "#79c0ff", "#56d4dd", "#d2a8ff",
];

export default function TrendChart({ request }: Props) {
  const [trendData, setTrendData] = useState<TrendPoint[]>([]);
  const [selectedPillar, setSelectedPillar] = useState<string>("overall");
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null);

  // Generate mock trend data on mount (in real app, loaded from IPC)
  const generateMockData = useCallback(() => {
    const points: TrendPoint[] = [];
    const now = Date.now();
    const weekMs = 7 * 24 * 60 * 60 * 1000;
    for (let i = 11; i >= 0; i--) {
      const date = new Date(now - i * weekMs);
      const pillarScores: Record<string, number> = {};
      let total = 0;
      Object.keys(PILLAR_LABELS).forEach((id) => {
        const base = 5 + Math.sin(i * 0.5 + id.length * 0.3) * 2;
        const noise = Math.random() * 1.5 - 0.75;
        const score = Math.max(0, Math.min(10, base + noise));
        pillarScores[id] = Math.round(score * 10) / 10;
        total += score;
      });
      points.push({
        date: date.toISOString(),
        pillarScores,
        overall: Math.round((total / Object.keys(PILLAR_LABELS).length) * 10) / 10,
      });
    }
    setTrendData(points);
  }, []);

  // Auto-generate mock data
  useMemo(() => {
    if (trendData.length === 0) generateMockData();
  }, [trendData.length, generateMockData]);

  const pillarIds = useMemo(() => ["overall", ...Object.keys(PILLAR_LABELS)], []);

  // Calculate chart coordinates
  const chartInner = useMemo(() => ({
    width: CHART_WIDTH - PADDING.left - PADDING.right,
    height: CHART_HEIGHT - PADDING.top - PADDING.bottom,
  }), []);

  const getScore = (point: TrendPoint): number => {
    if (selectedPillar === "overall") return point.overall;
    return point.pillarScores[selectedPillar] ?? 0;
  };

  const scores = trendData.map(getScore);
  const minScore = Math.max(0, Math.min(...scores) - 0.5);
  const maxScore = Math.min(10, Math.max(...scores) + 0.5);

  const xScale = (i: number): number => {
    if (trendData.length <= 1) return PADDING.left + chartInner.width / 2;
    return PADDING.left + (i / (trendData.length - 1)) * chartInner.width;
  };

  const yScale = (val: number): number => {
    const range = maxScore - minScore || 1;
    return PADDING.top + chartInner.height - ((val - minScore) / range) * chartInner.height;
  };

  // Build SVG path
  const pathD = trendData
    .map((point, i) => `${i === 0 ? "M" : "L"} ${xScale(i).toFixed(1)} ${yScale(getScore(point)).toFixed(1)}`)
    .join(" ");

  // Build area path
  const areaD = pathD +
    ` L ${xScale(trendData.length - 1).toFixed(1)} ${(PADDING.top + chartInner.height).toFixed(1)}` +
    ` L ${xScale(0).toFixed(1)} ${(PADDING.top + chartInner.height).toFixed(1)} Z`;

  // Y-axis ticks
  const yTicks = useMemo(() => {
    const ticks: number[] = [];
    const step = Math.ceil((maxScore - minScore) / 5 * 2) / 2 || 1;
    for (let v = Math.ceil(minScore); v <= maxScore; v += step) {
      ticks.push(Math.round(v * 10) / 10);
    }
    return ticks;
  }, [minScore, maxScore]);

  // X-axis labels
  const xLabels = useMemo(() => {
    return trendData.map((point, i) => {
      const d = new Date(point.date);
      const label = `${d.getMonth() + 1}/${d.getDate()}`;
      return { index: i, label };
    });
  }, [trendData]);

  // Determine which labels to show (max ~6)
  const showLabels = useMemo(() => {
    if (xLabels.length <= 6) return xLabels;
    const step = Math.ceil(xLabels.length / 6);
    return xLabels.filter((_, i) => i % step === 0 || i === xLabels.length - 1);
  }, [xLabels]);

  const hoveredPoint = hoveredIndex !== null ? trendData[hoveredIndex] : null;
  const hoveredScore = hoveredPoint ? getScore(hoveredPoint) : null;
  const hoveredDate = hoveredPoint ? new Date(hoveredPoint.date).toLocaleDateString() : null;

  return (
    <div className="trend-chart">
      <div className="trend-chart__header">
        <div>
          <h2 style={{ fontSize: 18, fontWeight: 700 }}>Pillar Score Trends</h2>
          <p className="text-secondary" style={{ fontSize: 13, marginTop: 2 }}>
            Weekly score snapshots over the past 12 weeks
          </p>
        </div>
        <div className="trend-chart__pill-filter">
          <label className="text-tertiary" style={{ fontSize: 12 }}>Pillar:</label>
          <select
            value={selectedPillar}
            onChange={(e) => setSelectedPillar(e.target.value)}
            className="trend-chart__select"
          >
            {pillarIds.map((id) => (
              <option key={id} value={id}>
                {id === "overall" ? "Overall Average" : PILLAR_LABELS[id] ?? id}
              </option>
            ))}
          </select>
        </div>
      </div>

      {/* Tooltip */}
      {hoveredPoint && hoveredDate && hoveredScore !== null && (
        <div className="trend-chart__tooltip">
          <div className="trend-chart__tooltip-date">{hoveredDate}</div>
          <div className="trend-chart__tooltip-score">
            {selectedPillar === "overall" ? "Overall" : PILLAR_LABELS[selectedPillar] ?? selectedPillar}:{" "}
            <strong>{hoveredScore.toFixed(1)}</strong>
          </div>
        </div>
      )}

      <div className="trend-chart__svg-wrapper">
        <svg
          viewBox={`0 0 ${CHART_WIDTH} ${CHART_HEIGHT}`}
          className="trend-chart__svg"
          onMouseLeave={() => setHoveredIndex(null)}
        >
          {/* Grid lines */}
          {yTicks.map((tick) => (
            <g key={tick}>
              <line
                x1={PADDING.left}
                y1={yScale(tick)}
                x2={PADDING.left + chartInner.width}
                y2={yScale(tick)}
                className="trend-chart__grid-line"
              />
              <text
                x={PADDING.left - 8}
                y={yScale(tick)}
                className="trend-chart__y-label"
                textAnchor="end"
                dominantBaseline="middle"
              >
                {tick.toFixed(1)}
              </text>
            </g>
          ))}

          {/* X-axis labels */}
          {showLabels.map(({ index, label }) => (
            <text
              key={index}
              x={xScale(index)}
              y={PADDING.top + chartInner.height + 20}
              className="trend-chart__x-label"
              textAnchor="middle"
            >
              {label}
            </text>
          ))}

          {/* Area fill */}
          <path d={areaD} className="trend-chart__area" />

          {/* Line */}
          <path d={pathD} className="trend-chart__line" />

          {/* Data points */}
          {trendData.map((point, i) => (
            <circle
              key={i}
              cx={xScale(i)}
              cy={yScale(getScore(point))}
              r={hoveredIndex === i ? 5 : 3}
              className={`trend-chart__dot ${hoveredIndex === i ? "trend-chart__dot--hovered" : ""}`}
              onMouseEnter={() => setHoveredIndex(i)}
            />
          ))}

          {/* Hover indicator line */}
          {hoveredIndex !== null && (
            <line
              x1={xScale(hoveredIndex)}
              y1={PADDING.top}
              x2={xScale(hoveredIndex)}
              y2={PADDING.top + chartInner.height}
              className="trend-chart__hover-line"
            />
          )}
        </svg>
      </div>

      {/* Legend */}
      <div className="trend-chart__legend">
        <span className="trend-chart__legend-label">
          {selectedPillar === "overall"
            ? "Overall Average Score"
            : `${PILLAR_LABELS[selectedPillar] ?? selectedPillar} Score`}
        </span>
        {trendData.length >= 2 && (
          <span className="trend-chart__legend-delta" style={{
            color: getScore(trendData[trendData.length - 1]) >= getScore(trendData[0])
              ? "var(--green)" : "var(--red)"
          }}>
            {getScore(trendData[trendData.length - 1]) >= getScore(trendData[0]) ? "+" : ""}
            {(getScore(trendData[trendData.length - 1]) - getScore(trendData[0])).toFixed(1)} from start
          </span>
        )}
      </div>
    </div>
  );
}
