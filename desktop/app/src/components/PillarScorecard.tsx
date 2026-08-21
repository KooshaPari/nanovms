import { useState, useEffect } from "react";
import type { PillarScore, ScorecardData, Pillar, IpcResponse } from "../types";
import PillarComparison from "./PillarComparison";
import TrendChart from "./TrendChart";

interface Props {
  onDataChange: () => void;
  request: <T = unknown>(channel: string, data?: unknown) => Promise<IpcResponse<T>>;
}

function scoreClass(score: number): string {
  if (score >= 9) return "score-great";
  if (score >= 7) return "score-good";
  if (score >= 5) return "score-warning";
  return "score-critical";
}

function fillClass(score: number): string {
  if (score >= 9) return "fill-great";
  if (score >= 7) return "fill-good";
  if (score >= 5) return "fill-warning";
  return "fill-critical";
}

function gradeColor(grade: string): string {
  if (grade === "A+" || grade === "A") return "score-great";
  if (grade === "B") return "score-good";
  if (grade === "C") return "score-warning";
  return "score-critical";
}

export default function PillarScorecard({ onDataChange, request }: Props) {
  const [scorecard, setScorecard] = useState<ScorecardData | null>(null);
  const [editPillars, setEditPillars] = useState<Pillar[]>([]);

  useEffect(() => {
    request<ScorecardData>("pillars:scorecard").then((r) => {
      if (r.ok && r.data) setScorecard(r.data as ScorecardData);
    });
    request<Pillar[]>("pillars:list").then((r) => {
      if (r.ok && Array.isArray(r.data)) setEditPillars(r.data);
    });
  }, [request]); // eslint-disable-line react-hooks/exhaustive-deps

  const handleScoreChange = (id: string, newScore: number) => {
    const clamped = Math.max(0, Math.min(10, newScore));
    request("pillars:update", { id, score: clamped }).then(() => {
      request<Pillar[]>("pillars:list").then((r) => {
        if (r.ok && Array.isArray(r.data)) setEditPillars(r.data);
      });
      onDataChange();
    });
  };

  const pillars: PillarScore[] = scorecard?.pillars ?? editPillars.map((p) => ({
    id: p.id,
    name: p.name,
    score: p.score,
    band: (p.score >= 9 ? "gold" : p.score >= 7 ? "green" : p.score >= 4 ? "yellow" : "red") as PillarScore["band"],
  }));

  const summary = scorecard?.summary;
  const average = summary?.average ?? (editPillars.length > 0
    ? editPillars.reduce((s, p) => s + p.score, 0) / editPillars.length
    : 0);
  const grade = summary?.grade ?? (average >= 9 ? "A+" : average >= 8 ? "A" : average >= 7 ? "B" : average >= 6 ? "C" : average >= 5 ? "D" : "F");
  const bandCounts = summary?.by_band ?? {
    red: pillars.filter((p) => p.band === "red").length,
    yellow: pillars.filter((p) => p.band === "yellow").length,
    green: pillars.filter((p) => p.band === "green").length,
    gold: pillars.filter((p) => p.band === "gold").length,
  };

  const [showComparison, setShowComparison] = useState(false);
  const [showTrends, setShowTrends] = useState(false);

  const CATEGORY_ORDER = ["performance", "reliability", "security", "developer-experience", "operations"];
  const CATEGORY_LABELS: Record<string, string> = {
    performance: "Performance",
    reliability: "Reliability",
    security: "Security",
    "developer-experience": "Developer Experience",
    operations: "Operations",
  };

  const grouped = editPillars.length > 0
    ? CATEGORY_ORDER.map((cat) => ({
        category: cat,
        label: CATEGORY_LABELS[cat] || cat,
        pillars: editPillars.filter((p) => p.category === cat),
      })).filter((g) => g.pillars.length > 0)
    : null;

  return (
    <div className="scorecard">
      <div className="scorecard__header">
        <div>
          <h2 style={{ fontSize: 20, fontWeight: 700 }}>31-Pillar Scorecard</h2>
          <p className="text-secondary" style={{ fontSize: 13, marginTop: 4 }}>
            Overall average across all pillars
            {scorecard && (
              <span style={{ marginLeft: 8, fontSize: 11, color: "var(--text-tertiary)" }}>
                Scored {scorecard.scored_at} by {scorecard.scored_by}
              </span>
            )}
          </p>
        </div>
        <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
          <div className="scorecard__overall">
            <span className="scorecard__overall-label">Overall</span>
            <span className={`scorecard__overall-value ${gradeColor(grade)}`}>
              {average > 0 ? average.toFixed(1) : "\u2014"}
            </span>
            <span className={`scorecard__overall-grade ${gradeColor(grade)}`}>
              {grade}
            </span>
          </div>
          <button className="btn btn--sm" onClick={() => setShowComparison(true)}>
            Compare
          </button>
          <button className="btn btn--sm" onClick={() => setShowTrends(!showTrends)}>
            {showTrends ? "Hide Trends" : "Trends"}
          </button>
        </div>
      </div>

      <div className="scorecard__band-summary">
        <div className="scorecard__band-item">
          <span className="scorecard__band-dot scorecard__band-dot--red" />
          <span>{bandCounts.red} Failing</span>
        </div>
        <div className="scorecard__band-item">
          <span className="scorecard__band-dot scorecard__band-dot--yellow" />
          <span>{bandCounts.yellow} Under Target</span>
        </div>
        <div className="scorecard__band-item">
          <span className="scorecard__band-dot scorecard__band-dot--green" />
          <span>{bandCounts.green} On Track</span>
        </div>
        <div className="scorecard__band-item">
          <span className="scorecard__band-dot scorecard__band-dot--gold" />
          <span>{bandCounts.gold} Gold</span>
        </div>
      </div>

      {grouped && grouped.map((group) => (
        <div key={group.category} className="scorecard__category">
          <div className="scorecard__category-title">
            {group.label} ({group.pillars.length})
          </div>
          {group.pillars.map((pillar) => (
            <PillarBar key={pillar.id} pillar={pillar} onChange={handleScoreChange} />
          ))}
        </div>
      ))}

      {!grouped && pillars.map((pillar) => (
        <div key={pillar.id} className="scorecard__category">
          <div className="scorecard__bar-row">
            <div className="scorecard__bar-name" title={pillar.name}>{pillar.name}</div>
            <div className="scorecard__bar-track">
              <div className={`scorecard__bar-fill ${fillClass(pillar.score)}`} style={{ width: `${(pillar.score / 10) * 100}%` }} />
            </div>
            <span className={`scorecard__bar-score ${scoreClass(pillar.score)}`}>{pillar.score.toFixed(1)}</span>
            <span className="scorecard__bar-target-val">{pillar.band}</span>
          </div>
        </div>
      ))}

      {pillars.length === 0 && (
        <div className="empty-state">
          <p>No pillar scores loaded. Run the pillar scorecard workflow or import data.</p>
        </div>
      )}

      {/* Trend Chart */}
      {showTrends && (
        <div className="scorecard__trend-section">
          <TrendChart request={request} />
        </div>
      )}

      {/* Comparison Modal */}
      {showComparison && (
        <div className="scorecard__comparison-overlay" onClick={() => setShowComparison(false)}>
          <div className="scorecard__comparison-modal" onClick={(e) => e.stopPropagation()}>
            <div className="scorecard__comparison-header">
              <h3>Pillar Comparison</h3>
              <button className="btn btn--sm" onClick={() => setShowComparison(false)}>Close</button>
            </div>
            <PillarComparison
              onDataChange={onDataChange}
              request={request}
              rightLabel="Current"
              rightScores={Object.fromEntries(pillars.map((p) => [p.id, p.score]))}
            />
          </div>
        </div>
      )}
    </div>
  );
}

function PillarBar({ pillar, onChange }: { pillar: Pillar; onChange: (id: string, score: number) => void }) {
  const [editing, setEditing] = useState(false);
  const [editValue, setEditValue] = useState(pillar.score.toString());

  const handleBlur = () => {
    setEditing(false);
    const val = parseFloat(editValue);
    if (!isNaN(val)) onChange(pillar.id, Math.max(0, Math.min(10, val)));
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") handleBlur();
    if (e.key === "Escape") { setEditValue(pillar.score.toString()); setEditing(false); }
  };

  const pct = (pillar.score / 10) * 100;
  const targetPct = (pillar.target / 10) * 100;

  // Trend indicator: compare score to target for a simple up/down/neutral arrow
  const trendDirection = pillar.score > pillar.target + 0.5
    ? "up"
    : pillar.score < pillar.target - 0.5
      ? "down"
      : "neutral";

  return (
    <div className="scorecard__bar-row">
      <div className="scorecard__bar-name" title={pillar.name}>{pillar.name}</div>
      <div className="scorecard__bar-track">
        <div className={`scorecard__bar-fill ${fillClass(pillar.score)}`} style={{ width: `${pct}%` }} />
        <div className="scorecard__bar-target" style={{ left: `${targetPct}%` }} title={`Target: ${pillar.target}`} />
      </div>
      {editing ? (
        <input
          className="scorecard__bar-input"
          type="number"
          min={0}
          max={10}
          step={0.5}
          value={editValue}
          onChange={(e) => setEditValue(e.target.value)}
          onBlur={handleBlur}
          onKeyDown={handleKeyDown}
          autoFocus
        />
      ) : (
        <span
          className={`scorecard__bar-score ${scoreClass(pillar.score)}`}
          onClick={() => { setEditValue(pillar.score.toString()); setEditing(true); }}
          style={{ cursor: "pointer" }}
          title="Click to edit"
        >
          {pillar.score.toFixed(1)}
        </span>
      )}
      <span className="scorecard__bar-target-val">/ {pillar.target}</span>
      <span
        className={`scorecard__bar-trend scorecard__bar-trend--${trendDirection}`}
        title={trendDirection === "up" ? "Above target" : trendDirection === "down" ? "Below target" : "Near target"}
      >
        {trendDirection === "up" ? "\u2191" : trendDirection === "down" ? "\u2193" : "\u2192"}
      </span>
    </div>
  );
}
