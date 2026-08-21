import { useState } from "react";
import { getPillars, updatePillar, getOverallScore } from "../store";
import type { Pillar } from "../types";

interface Props {
  onDataChange: () => void;
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

const CATEGORY_LABELS: Record<string, string> = {
  performance: "Performance",
  reliability: "Reliability",
  security: "Security",
  "developer-experience": "Developer Experience",
  operations: "Operations",
};

const CATEGORY_ORDER = [
  "performance",
  "reliability",
  "security",
  "developer-experience",
  "operations",
];

export default function ScorecardView({ onDataChange }: Props) {
  const [pillars, setPillars] = useState<Pillar[]>(() => getPillars());
  const overall = getOverallScore();

  const handleScoreChange = (id: string, newScore: number) => {
    updatePillar(id, newScore);
    setPillars(getPillars());
    onDataChange();
  };

  const grouped = CATEGORY_ORDER.map((cat) => ({
    category: cat,
    label: CATEGORY_LABELS[cat] || cat,
    pillars: pillars.filter((p) => p.category === cat),
  })).filter((g) => g.pillars.length > 0);

  return (
    <div className="scorecard">
      <div className="scorecard__header">
        <div>
          <h2 style={{ fontSize: 20, fontWeight: 700 }}>31-Pillar Scorecard</h2>
          <p className="text-secondary" style={{ fontSize: 13, marginTop: 4 }}>
            Overall average across all pillars
          </p>
        </div>
        <div className="scorecard__overall">
          <span className="scorecard__overall-label">Overall</span>
          <span className={`scorecard__overall-value ${scoreClass(overall)}`}>
            {overall > 0 ? overall.toFixed(1) : "—"}
          </span>
        </div>
      </div>

      {grouped.map((group) => (
        <div key={group.category} className="scorecard__category">
          <div className="scorecard__category-title">
            {group.label} ({group.pillars.length})
          </div>
          {group.pillars.map((pillar) => (
            <PillarBar
              key={pillar.id}
              pillar={pillar}
              onChange={handleScoreChange}
            />
          ))}
        </div>
      ))}
    </div>
  );
}

// ─── Individual Pillar Bar ───────────────────────────────────────────────────
interface PillarBarProps {
  pillar: Pillar;
  onChange: (id: string, score: number) => void;
}

function PillarBar({ pillar, onChange }: PillarBarProps) {
  const [editing, setEditing] = useState(false);
  const [editValue, setEditValue] = useState(pillar.score.toString());

  const handleBlur = () => {
    setEditing(false);
    const val = parseFloat(editValue);
    if (!isNaN(val)) {
      onChange(pillar.id, Math.max(0, Math.min(10, val)));
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") handleBlur();
    if (e.key === "Escape") {
      setEditValue(pillar.score.toString());
      setEditing(false);
    }
  };

  const pct = (pillar.score / 10) * 100;
  const targetPct = (pillar.target / 10) * 100;

  return (
    <div className="scorecard__bar-row">
      <div className="scorecard__bar-name" title={pillar.name}>
        {pillar.name}
      </div>
      <div className="scorecard__bar-track">
        <div
          className={`scorecard__bar-fill ${fillClass(pillar.score)}`}
          style={{ width: `${pct}%` }}
        />
        <div
          className="scorecard__bar-target"
          style={{ left: `${targetPct}%` }}
          title={`Target: ${pillar.target}`}
        />
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
          onClick={() => {
            setEditValue(pillar.score.toString());
            setEditing(true);
          }}
          style={{ cursor: "pointer" }}
          title="Click to edit"
        >
          {pillar.score.toFixed(1)}
        </span>
      )}
      <span className="scorecard__bar-target-val">
        / {pillar.target}
      </span>
    </div>
  );
}
