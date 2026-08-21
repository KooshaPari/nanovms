import { useState, useEffect, useCallback, useRef } from "react";
import {
  HiOutlineCheckCircle,
  HiOutlineExclamationCircle,
  HiOutlineXCircle,
  HiOutlineCog,
} from "react-icons/hi";
import type { Spec, SpecValidation, ValidationCheck, IpcResponse } from "../types";

interface Props {
  onDataChange: () => void;
  request: <T = unknown>(channel: string, data?: unknown) => Promise<IpcResponse<T>>;
}

// ─── 31 Pillar IDs (must match the model in types.ts) ─────────────────────
const ALL_PILLAR_IDS = [
  "perf-boot", "perf-throughput", "perf-latency-p99", "perf-memory",
  "perf-io", "perf-cpu", "perf-network",
  "rel-uptime", "rel-crash", "rel-recovery", "rel-data", "rel-timeout", "rel-health",
  "sec-auth", "sec-crypto", "sec-isolation", "sec-secrets", "sec-audit", "sec-vuln",
  "dx-api", "dx-docs", "dx-cli", "dx-debug", "dx-testing", "dx-contrib",
  "ops-deploy", "ops-monitor", "ops-scaling", "ops-config", "ops-migrate", "ops-cost",
];

// ─── Required sections for a well-formed spec ──────────────────────────────
const REQUIRED_SECTIONS: { id: string; label: string; patterns: RegExp[] }[] = [
  { id: "title", label: "Title (H1)", patterns: [/^#\s+\S/m] },
  { id: "overview", label: "Overview / Introduction", patterns: [/^##\s+(overview|intro|background|summary)/im] },
  { id: "pillars", label: "Pillar Coverage", patterns: [/pillar|31.pillar|scorecard|quality/i] },
  { id: "acceptance", label: "Acceptance Criteria", patterns: [/^##\s+(acceptance|criteria|definition of done|success criteria)/im] },
  { id: "risks", label: "Risks / Mitigations", patterns: [/^##\s+(risks|mitigation|risk analysis)/im] },
  { id: "architecture", label: "Architecture / Design", patterns: [/^##\s+(architecture|design|system design|technical)/im] },
  { id: "testing", label: "Testing Strategy", patterns: [/^##\s+(test|testing|quality assurance|qa)/im] },
];

// ─── Pillar coverage keywords mapping ──────────────────────────────────────
const PILLAR_KEYWORDS: Record<string, RegExp[]> = {
  "perf-boot": [/boot\s*time|cold\s*start/i],
  "perf-throughput": [/throughput|requests?\s*per\s*sec/i],
  "perf-latency-p99": [/p99|p95|latency|response\s*time/i],
  "perf-memory": [/memory|ram|heap|alloc/i],
  "perf-io": [/\bio\b|disk|file\s*system|storage/i],
  "perf-cpu": [/cpu|processor|thread/i],
  "perf-network": [/network|tcp|udp|socket/i],
  "rel-uptime": [/uptime|availability|slas?/i],
  "rel-crash": [/crash|panic|abort/i],
  "rel-recovery": [/recovery|retry|fallback/i],
  "rel-data": [/data\s*integrity|consistency|durability/i],
  "rel-timeout": [/timeout|deadline/i],
  "rel-health": [/health\s*check|heartbeat|alive/i],
  "sec-auth": [/auth|authenticat|identity/i],
  "sec-crypto": [/crypt|encrypt|sign|tls|ssl/i],
  "sec-isolation": [/isolat|sandbox|contain|namespace/i],
  "sec-secrets": [/secret|credential|vault|key\s*manag/i],
  "sec-audit": [/audit|trail|log\s*event/i],
  "sec-vuln": [/vuln|cve|dependab|advisory/i],
  "dx-api": [/\bapi\b|rest|graphql|endpoint/i],
  "dx-docs": [/doc|readme|guide/i],
  "dx-cli": [/\bcli\b|command.line|tooling/i],
  "dx-debug": [/debug|tracing|profil/i],
  "dx-testing": [/test|spec|coverage|lint/i],
  "dx-contrib": [/contribut|onboard|developer/i],
  "ops-deploy": [/deploy|release|ci.cd|pipeline/i],
  "ops-monitor": [/monitor|observ|metric|log|trace/i],
  "ops-scaling": [/scal|horizontal|replica/i],
  "ops-config": [/config|env\s*var|setting/i],
  "ops-migrate": [/migrat|upgrade|backward/i],
  "ops-cost": [/cost|pricing|bill|resource/i],
};

function validateSpec(spec: Spec): SpecValidation {
  const checks: ValidationCheck[] = [];

  // 1. Check required sections
  for (const section of REQUIRED_SECTIONS) {
    const found = section.patterns.some((p) => p.test(spec.content));
    checks.push({
      id: `section-${section.id}`,
      label: section.label,
      section: section.id,
      severity: found ? "pass" : "fail",
      message: found
        ? `Section "${section.label}" is present`
        : `Missing required section: "${section.label}"`,
    });
  }

  // 2. Check pillar coverage — each of the 31 pillars should be addressed
  let coveredPillars = 0;
  for (const pillarId of ALL_PILLAR_IDS) {
    const patterns = PILLAR_KEYWORDS[pillarId] ?? [];
    const found = patterns.some((p) => p.test(spec.content));
    if (found) coveredPillars++;
    checks.push({
      id: `pillar-${pillarId}`,
      label: pillarId,
      section: "pillars",
      severity: found ? "pass" : "warn",
      message: found
        ? `Pillar "${pillarId}" addressed in spec`
        : `Pillar "${pillarId}" not addressed — consider adding coverage`,
    });
  }

  // 3. Content quality checks
  const wordCount = spec.content.split(/\s+/).filter((w) => w.length > 0).length;
  const hasCodeBlocks = /```/.test(spec.content);
  const hasLists = /^[-*]\s+/m.test(spec.content) || /^\d+\.\s+/m.test(spec.content);

  checks.push({
    id: "quality-wordcount",
    label: "Word count",
    section: "quality",
    severity: wordCount >= 100 ? "pass" : wordCount >= 30 ? "warn" : "fail",
    message: `Spec has ${wordCount} words ${wordCount < 30 ? "(too short)" : wordCount < 100 ? "(could be more detailed)" : "(adequate)"}`,
  });

  checks.push({
    id: "quality-code",
    label: "Code examples",
    section: "quality",
    severity: hasCodeBlocks ? "pass" : "warn",
    message: hasCodeBlocks ? "Contains code examples" : "No code examples found",
  });

  checks.push({
    id: "quality-lists",
    label: "Structured lists",
    section: "quality",
    severity: hasLists ? "pass" : "warn",
    message: hasLists ? "Uses structured lists" : "No structured lists found",
  });

  // 4. Pillar score assignment
  const assignedPillars = Object.keys(spec.pillarScores).length;
  const pctAssigned = Math.round((assignedPillars / 31) * 100);
  checks.push({
    id: "quality-scores",
    label: "Pillar score assignment",
    section: "quality",
    severity: pctAssigned >= 80 ? "pass" : pctAssigned >= 50 ? "warn" : "fail",
    message: `${assignedPillars}/31 pillars scored (${pctAssigned}%)`,
  });

  // Calculate score
  const passCount = checks.filter((c) => c.severity === "pass").length;
  const warnCount = checks.filter((c) => c.severity === "warn").length;
  const totalWeighted = passCount * 1 + warnCount * 0.5;
  const score = Math.round((totalWeighted / checks.length) * 100);

  return {
    specId: spec.id,
    checks,
    score,
    validatedAt: new Date().toISOString(),
  };
}

export default function SpecValidator({ onDataChange, request }: Props) {
  const [specs, setSpecs] = useState<Spec[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [validation, setValidation] = useState<SpecValidation | null>(null);
  const [isValidating, setIsValidating] = useState(false);
  const [filter, setFilter] = useState<"all" | "pass" | "warn" | "fail">("all");
  const editorRef = useRef<HTMLTextAreaElement | null>(null);

  useEffect(() => {
    request<Spec[]>("specs:list").then((r) => {
      if (r.ok && Array.isArray(r.data)) {
        setSpecs(r.data);
        if (r.data.length > 0 && !selectedId) {
          setSelectedId(r.data[0].id);
        }
      }
    });
  }, [request]); // eslint-disable-line react-hooks/exhaustive-deps

  const runValidation = useCallback(
    (specId: string) => {
      setIsValidating(true);
      request<Spec>("specs:get", { id: specId }).then((r) => {
        if (r.ok && r.data) {
          const result = validateSpec(r.data as Spec);
          setValidation(result);
        }
        setIsValidating(false);
      });
    },
    [request],
  );

  useEffect(() => {
    if (selectedId) runValidation(selectedId);
  }, [selectedId, runValidation]);

  const selectedSpec = specs.find((s) => s.id === selectedId);
  const checks = validation?.checks ?? [];
  const filtered = filter === "all" ? checks : checks.filter((c) => c.severity === filter);
  const score = validation?.score ?? 0;

  const passCount = checks.filter((c) => c.severity === "pass").length;
  const warnCount = checks.filter((c) => c.severity === "warn").length;
  const failCount = checks.filter((c) => c.severity === "fail").length;

  const scoreColor =
    score >= 80 ? "var(--green)" : score >= 60 ? "var(--yellow)" : "var(--red)";

  const handleFixSection = (sectionId: string) => {
    // Scroll to the section in the editor if available
    const textarea = document.querySelector(".spec-textarea") as HTMLTextAreaElement | null;
    if (!textarea || !selectedSpec) return;
    const lines = selectedSpec.content.split("\n");
    let targetLine = 0;
    for (let i = 0; i < lines.length; i++) {
      if (lines[i].toLowerCase().includes(sectionId.toLowerCase())) {
        targetLine = i;
        break;
      }
    }
    // Approximate scroll position
    const lineHeight = 20; // approx px per line
    textarea.scrollTop = targetLine * lineHeight;
  };

  const severityIcon = (sev: "pass" | "warn" | "fail") => {
    switch (sev) {
      case "pass":
        return <HiOutlineCheckCircle className="validation-icon validation-icon--pass" />;
      case "warn":
        return <HiOutlineExclamationCircle className="validation-icon validation-icon--warn" />;
      case "fail":
        return <HiOutlineXCircle className="validation-icon validation-icon--fail" />;
    }
  };

  return (
    <div className="validation">
      <div className="validation__layout">
        {/* Left: Spec selector */}
        <div className="validation__sidebar">
          <div className="validation__sidebar-header">
            <span className="text-tertiary" style={{ fontSize: 12 }}>
              {specs.length} specs
            </span>
          </div>
          <div className="validation__sidebar-list">
            {specs.map((spec) => (
              <button
                key={spec.id}
                className={`validation__sidebar-item ${
                  selectedId === spec.id ? "validation__sidebar-item--active" : ""
                }`}
                onClick={() => setSelectedId(spec.id)}
              >
                <span style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                  {spec.title || "Untitled"}
                </span>
              </button>
            ))}
          </div>
        </div>

        {/* Right: Validation results */}
        <div className="validation__main">
          <div className="validation__header">
            <div>
              <h2 style={{ fontSize: 18, fontWeight: 700 }}>Spec Validation</h2>
              <p className="text-secondary" style={{ fontSize: 13, marginTop: 2 }}>
                Validates that all 31 pillars are addressed and required sections exist
              </p>
            </div>
            <button
              className="btn btn--primary"
              onClick={() => selectedId && runValidation(selectedId)}
              disabled={isValidating}
            >
              <HiOutlineCog className={isValidating ? "spin" : ""} />
              {isValidating ? "Validating..." : "Re-validate"}
            </button>
          </div>

          {/* Score banner */}
          <div className="validation__score-banner">
            <div className="validation__score-ring">
              <svg viewBox="0 0 100 100" className="validation__score-svg">
                <circle cx="50" cy="50" r="42" className="validation__score-track" />
                <circle
                  cx="50"
                  cy="50"
                  r="42"
                  className="validation__score-fill"
                  style={{
                    stroke: scoreColor,
                    strokeDasharray: `${(score / 100) * 264} 264`,
                  }}
                />
              </svg>
              <span className="validation__score-number" style={{ color: scoreColor }}>
                {score}
              </span>
            </div>
            <div className="validation__score-details">
              <div className="validation__score-detail">
                <span className="validation__score-dot validation__score-dot--pass" />
                <span>{passCount} Pass</span>
              </div>
              <div className="validation__score-detail">
                <span className="validation__score-dot validation__score-dot--warn" />
                <span>{warnCount} Warn</span>
              </div>
              <div className="validation__score-detail">
                <span className="validation__score-dot validation__score-dot--fail" />
                <span>{failCount} Fail</span>
              </div>
            </div>
            <div className="validation__score-label">
              {selectedSpec ? `Validating: ${selectedSpec.title}` : "Select a spec"}
            </div>
          </div>

          {/* Filter tabs */}
          <div className="validation__filters">
            {(["all", "pass", "warn", "fail"] as const).map((f) => {
              const count =
                f === "all" ? checks.length : checks.filter((c) => c.severity === f).length;
              return (
                <button
                  key={f}
                  className={`validation__filter-btn ${filter === f ? "validation__filter-btn--active" : ""}`}
                  onClick={() => setFilter(f)}
                >
                  {f === "all" ? "All" : f.charAt(0).toUpperCase() + f.slice(1)} ({count})
                </button>
              );
            })}
          </div>

          {/* Check list */}
          <div className="validation__checks">
            {filtered.length === 0 && (
              <div className="empty-state">
                <p>No checks to display</p>
              </div>
            )}
            {filtered.map((check) => (
              <div key={check.id} className={`validation__check validation__check--${check.severity}`}>
                <div className="validation__check-icon">{severityIcon(check.severity)}</div>
                <div className="validation__check-body">
                  <div className="validation__check-label">{check.label}</div>
                  <div className="validation__check-message">{check.message}</div>
                </div>
                {check.severity === "fail" && (
                  <button
                    className="btn btn--sm"
                    onClick={() => handleFixSection(check.section)}
                    title="Scroll to section in editor"
                  >
                    Fix
                  </button>
                )}
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
