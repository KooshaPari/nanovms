import { useState, useEffect, useMemo } from "react";
import {
  HiOutlineArrowUp,
  HiOutlineArrowDown,
  HiOutlineMinus,
} from "react-icons/hi";
import type { Pillar, ComparisonData, ComparisonEntry, DiffDirection, IpcResponse } from "../types";

interface Props {
  onDataChange: () => void;
  request: <T = unknown>(channel: string, data?: unknown) => Promise<IpcResponse<T>>;
  /** When opened from PillarScorecard, pre-fill the right side with current data */
  rightLabel?: string;
  rightScores?: Record<string, number>;
}

function computeComparison(
  leftScores: Record<string, number>,
  rightScores: Record<string, number>,
  leftLabel: string,
  rightLabel: string,
  pillars: Pillar[],
): ComparisonData {
  const entries: ComparisonEntry[] = pillars.map((pillar) => {
    const left = leftScores[pillar.id] ?? 0;
    const right = rightScores[pillar.id] ?? 0;
    const delta = right - left;
    const direction: DiffDirection =
      Math.abs(delta) < 0.01 ? "same" : delta > 0 ? "improved" : "regressed";
    return {
      pillarId: pillar.id,
      pillarName: pillar.name,
      leftScore: left,
      rightScore: right,
      delta,
      direction,
    };
  });

  const overallLeft =
    entries.length > 0
      ? entries.reduce((s, e) => s + e.leftScore, 0) / entries.length
      : 0;
  const overallRight =
    entries.length > 0
      ? entries.reduce((s, e) => s + e.rightScore, 0) / entries.length
      : 0;

  return {
    leftLabel,
    rightLabel,
    entries,
    overallLeft: Math.round(overallLeft * 10) / 10,
    overallRight: Math.round(overallRight * 10) / 10,
    overallDelta: Math.round((overallRight - overallLeft) * 10) / 10,
  };
}

export default function PillarComparison({ onDataChange, request, rightLabel, rightScores }: Props) {
  const [pillars, setPillars] = useState<Pillar[]>([]);
  const [leftScores, setLeftScores] = useState<Record<string, number>>({});
  const [rightScoresState, setRightScoresState] = useState<Record<string, number>>(rightScores ?? {});
  const [leftLabel, setLeftLabel] = useState<string>("Baseline");
  const [rightLabelState, setRightLabelState] = useState<string>(rightLabel ?? "Current");
  const [snapshotList, setSnapshotList] = useState<Array<{ label: string; date: string; scores: Record<string, number> }>>([]);

  useEffect(() => {
    request<Pillar[]>("pillars:list").then((r) => {
      if (r.ok && Array.isArray(r.data)) setPillars(r.data);
    });
    // Load saved snapshots for comparison
    request<Array<{ label: string; date: string; scores: Record<string, number> }>>("comparison:snapshots").then((r) => {
      if (r.ok && Array.isArray(r.data)) setSnapshotList(r.data);
    });
  }, [request]); // eslint-disable-line react-hooks/exhaustive-deps

  // Sync right scores from props
  useEffect(() => {
    if (rightScores) setRightScoresState(rightScores);
    if (rightLabel) setRightLabelState(rightLabel);
  }, [rightScores, rightLabel]);

  const comparison = useMemo(
    () => computeComparison(leftScores, rightScoresState, leftLabel, rightLabelState, pillars),
    [leftScores, rightScoresState, leftLabel, rightLabelState, pillars],
  );

  const improvedCount = comparison.entries.filter((e) => e.direction === "improved").length;
  const regressedCount = comparison.entries.filter((e) => e.direction === "regressed").length;
  const sameCount = comparison.entries.filter((e) => e.direction === "same").length;

  const maxBarScore = 10;

  return (
    <div className="comparison">
      <div className="comparison__header">
        <div>
          <h2 style={{ fontSize: 18, fontWeight: 700 }}>Pillar Comparison</h2>
          <p className="text-secondary" style={{ fontSize: 13, marginTop: 2 }}>
            Compare pillar scores across two snapshots or repos
          </p>
        </div>
      </div>

      {/* Controls */}
      <div className="comparison__controls">
        <div className="comparison__control-group">
          <label className="text-tertiary" style={{ fontSize: 12 }}>Left (Base)</label>
          <div className="comparison__control-row">
            <input
              value={leftLabel}
              onChange={(e) => setLeftLabel(e.target.value)}
              placeholder="Label..."
              className="comparison__label-input"
            />
            <select
              onChange={(e) => {
                const snap = snapshotList[Number(e.target.value)];
                if (snap) {
                  setLeftScores(snap.scores);
                  setLeftLabel(snap.label);
                }
              }}
              className="comparison__snapshot-select"
            >
              <option value="">Select snapshot...</option>
              {snapshotList.map((snap, i) => (
                <option key={i} value={i}>{snap.label} ({new Date(snap.date).toLocaleDateString()})</option>
              ))}
            </select>
          </div>
        </div>
        <div className="comparison__control-group">
          <label className="text-tertiary" style={{ fontSize: 12 }}>Right (Compare)</label>
          <div className="comparison__control-row">
            <input
              value={rightLabelState}
              onChange={(e) => setRightLabelState(e.target.value)}
              placeholder="Label..."
              className="comparison__label-input"
            />
            <select
              onChange={(e) => {
                const snap = snapshotList[Number(e.target.value)];
                if (snap) {
                  setRightScoresState(snap.scores);
                  setRightLabelState(snap.label);
                }
              }}
              className="comparison__snapshot-select"
            >
              <option value="">Select snapshot...</option>
              {snapshotList.map((snap, i) => (
                <option key={i} value={i}>{snap.label} ({new Date(snap.date).toLocaleDateString()})</option>
              ))}
            </select>
          </div>
        </div>
      </div>

      {/* Overall delta banner */}
      <div className="comparison__delta-banner">
        <div className="comparison__delta-sides">
          <div className="comparison__delta-side">
            <span className="comparison__delta-label">{leftLabel}</span>
            <span className="comparison__delta-value">{comparison.overallLeft.toFixed(1)}</span>
          </div>
          <div className="comparison__delta-arrow">
            {comparison.overallDelta > 0 ? (
              <HiOutlineArrowUp style={{ color: "var(--green)", fontSize: 20 }} />
            ) : comparison.overallDelta < 0 ? (
              <HiOutlineArrowDown style={{ color: "var(--red)", fontSize: 20 }} />
            ) : (
              <HiOutlineMinus style={{ color: "var(--text-tertiary)", fontSize: 20 }} />
            )}
            <span
              style={{
                fontSize: 16,
                fontWeight: 700,
                color: comparison.overallDelta > 0 ? "var(--green)" : comparison.overallDelta < 0 ? "var(--red)" : "var(--text-tertiary)",
              }}
            >
              {comparison.overallDelta > 0 ? "+" : ""}{comparison.overallDelta.toFixed(1)}
            </span>
          </div>
          <div className="comparison__delta-side">
            <span className="comparison__delta-label">{rightLabelState}</span>
            <span className="comparison__delta-value">{comparison.overallRight.toFixed(1)}</span>
          </div>
        </div>
        <div className="comparison__delta-summary">
          <span style={{ color: "var(--green)" }}>{improvedCount} improved</span>
          <span style={{ color: "var(--red)" }}>{regressedCount} regressed</span>
          <span style={{ color: "var(--text-tertiary)" }}>{sameCount} unchanged</span>
        </div>
      </div>

      {/* Side-by-side bar charts */}
      <div className="comparison__bars">
        <div className="comparison__bars-header">
          <span className="comparison__bars-col-name">Pillar</span>
          <span className="comparison__bars-col-score">{leftLabel}</span>
          <span className="comparison__bars-col-bar">Bar</span>
          <span className="comparison__bars-col-score">{rightLabelState}</span>
          <span className="comparison__bars-col-delta">Delta</span>
        </div>
        {comparison.entries.map((entry) => (
          <ComparisonBarRow key={entry.pillarId} entry={entry} maxScore={maxBarScore} />
        ))}
      </div>

      {pillars.length === 0 && (
        <div className="empty-state">
          <p>No pillar data loaded. Update pillar scores first.</p>
        </div>
      )}
    </div>
  );
}

function ComparisonBarRow({ entry, maxScore }: { entry: ComparisonEntry; maxScore: number }) {
  const leftPct = (entry.leftScore / maxScore) * 100;
  const rightPct = (entry.rightScore / maxScore) * 100;

  const directionIcon = () => {
    switch (entry.direction) {
      case "improved":
        return <HiOutlineArrowUp style={{ color: "var(--green)", fontSize: 14 }} />;
      case "regressed":
        return <HiOutlineArrowDown style={{ color: "var(--red)", fontSize: 14 }} />;
      default:
        return <HiOutlineMinus style={{ color: "var(--text-tertiary)", fontSize: 14 }} />;
    }
  };

  const deltaColor =
    entry.direction === "improved"
      ? "var(--green)"
      : entry.direction === "regressed"
        ? "var(--red)"
        : "var(--text-tertiary)";

  return (
    <div className="comparison__bar-row">
      <span className="comparison__bar-name" title={entry.pillarName}>{entry.pillarName}</span>
      <span className="comparison__bar-score">{entry.leftScore.toFixed(1)}</span>
      <div className="comparison__bar-dual-track">
        <div
          className="comparison__bar-dual-fill comparison__bar-dual-fill--left"
          style={{ width: `${leftPct}%` }}
        />
        <div
          className="comparison__bar-dual-fill comparison__bar-dual-fill--right"
          style={{ width: `${rightPct}%` }}
        />
      </div>
      <span className="comparison__bar-score">{entry.rightScore.toFixed(1)}</span>
      <span className="comparison__bar-delta" style={{ color: deltaColor }}>
        {directionIcon()}
        {entry.delta > 0 ? "+" : ""}{entry.delta.toFixed(1)}
      </span>
    </div>
  );
}
