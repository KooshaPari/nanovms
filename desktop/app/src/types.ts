// ─── AgilePlus Desktop — TypeScript Types ────────────────────────────────────

// ─── Spec ─────────────────────────────────────────────────────────────────────
export interface Spec {
  id: string;
  title: string;
  content: string;
  pillarScores: Record<string, number>; // pillarId → score 0-10
  createdAt: string;
  updatedAt: string;
}

// ─── Pillar (31-pillar model) ────────────────────────────────────────────────
export interface Pillar {
  id: string;
  name: string;
  score: number; // 0-10
  target: number;
  category: "performance" | "reliability" | "security" | "developer-experience" | "operations";
}

// ─── Pillar Score (from agileplus/pillars/31-pillar-scorecard.json) ──────────
export interface PillarScore {
  id: string;
  name: string;
  score: number; // 0-10
  band: "red" | "yellow" | "green" | "gold";
}

export interface ScorecardSummary {
  count: number;
  average: number;
  grade: string;
  grade_label: string;
  by_band: { red: number; yellow: number; green: number; gold: number };
  top_pillars: string[];
  weak_pillars: string[];
}

export interface ScorecardData {
  schema_version: string;
  repo: string;
  scored_at: string;
  scored_by: string;
  scale: {
    min: number;
    max: number;
    bands: Record<string, string>;
    grades: Record<string, string>;
  };
  pillars: PillarScore[];
  summary: ScorecardSummary;
}

// ─── Sprint ──────────────────────────────────────────────────────────────────
export interface Sprint {
  id: string;
  name: string;
  startDate: string;
  endDate: string;
  status: "planned" | "active" | "completed";
  velocity: number;
  goal: string;
}

// ─── Quality Gate ────────────────────────────────────────────────────────────
export interface QualityGate {
  id: string;
  name: string;
  description: string;
  pillar: string;
  required: boolean;
  command: string;
  floor?: number;
}

export interface QualityGateStatus {
  gate_id: string;
  status: "pass" | "fail" | "pending" | "skipped";
  lastRun: string;
  output?: string;
  duration_ms?: number;
}

// ─── Backlog Item ────────────────────────────────────────────────────────────
export type Priority = "P0" | "P1" | "P2" | "P3";

export interface BacklogItem {
  id: string;
  title: string;
  priority: Priority;
  status: "todo" | "in-progress" | "review" | "done";
  specId?: string;
  pillar?: string;
  createdAt: string;
}

// ─── View ────────────────────────────────────────────────────────────────────
export type View =
  | "dashboard"
  | "specs"
  | "scorecards"
  | "sprints"
  | "quality-gates"
  | "validation"
  | "compare"
  | "backlog";

// ─── Spec Validation ──────────────────────────────────────────────────────
export type ValidationSeverity = "pass" | "warn" | "fail";

export interface ValidationCheck {
  id: string;
  label: string;
  section: string; // required section name in the spec
  severity: ValidationSeverity;
  message: string;
}

export interface SpecValidation {
  specId: string;
  checks: ValidationCheck[];
  score: number; // 0-100
  validatedAt: string;
}

// ─── Trend Data ─────────────────────────────────────────────────────────────
export interface TrendPoint {
  date: string; // ISO date string
  pillarScores: Record<string, number>; // pillarId → score
  overall: number;
}

export interface TrendData {
  pillarId: string;
  points: TrendPoint[];
}

// ─── Pillar Comparison ──────────────────────────────────────────────────────
export type DiffDirection = "improved" | "regressed" | "same";

export interface ComparisonEntry {
  pillarId: string;
  pillarName: string;
  leftScore: number;
  rightScore: number;
  delta: number;
  direction: DiffDirection;
}

export interface ComparisonData {
  leftLabel: string;
  rightLabel: string;
  entries: ComparisonEntry[];
  overallLeft: number;
  overallRight: number;
  overallDelta: number;
}

// ─── Backlog Board ──────────────────────────────────────────────────────────
export interface BacklogBoardItem {
  id: string;
  title: string;
  priority: Priority;
  points: number;
  assignee: string;
  pillar?: string;
  specId?: string;
  createdAt: string;
}

// ─── Gate History ────────────────────────────────────────────────────────────
export interface GateRun {
  id: string;
  gateId: string;
  gateName: string;
  status: "pass" | "fail";
  triggeredBy: string;
  durationMs: number;
  runAt: string;
}

// ─── IPC Response ────────────────────────────────────────────────────────────
export interface IpcResponse<T = unknown> {
  ok: boolean;
  data?: T;
  error?: string;
}

// ─── Default Pillars (31-pillar nanovms model) ──────────────────────────────
export const DEFAULT_PILLARS: Omit<Pillar, "score">[] = [
  // Performance (7)
  { id: "perf-boot", name: "Boot Time", target: 8, category: "performance" },
  { id: "perf-throughput", name: "Throughput", target: 8, category: "performance" },
  { id: "perf-latency-p99", name: "P99 Latency", target: 9, category: "performance" },
  { id: "perf-memory", name: "Memory Efficiency", target: 8, category: "performance" },
  { id: "perf-io", name: "I/O Performance", target: 7, category: "performance" },
  { id: "perf-cpu", name: "CPU Utilization", target: 8, category: "performance" },
  { id: "perf-network", name: "Network Stack", target: 7, category: "performance" },

  // Reliability (6)
  { id: "rel-uptime", name: "Uptime / Availability", target: 9, category: "reliability" },
  { id: "rel-crash", name: "Crash Rate", target: 9, category: "reliability" },
  { id: "rel-recovery", name: "Error Recovery", target: 8, category: "reliability" },
  { id: "rel-data", name: "Data Integrity", target: 10, category: "reliability" },
  { id: "rel-timeout", name: "Timeout Handling", target: 7, category: "reliability" },
  { id: "rel-health", name: "Health Checks", target: 8, category: "reliability" },

  // Security (6)
  { id: "sec-auth", name: "Authentication", target: 9, category: "security" },
  { id: "sec-crypto", name: "Cryptographic Ops", target: 9, category: "security" },
  { id: "sec-isolation", name: "Isolation / Sandbox", target: 8, category: "security" },
  { id: "sec-secrets", name: "Secret Management", target: 9, category: "security" },
  { id: "sec-audit", name: "Audit Trail", target: 7, category: "security" },
  { id: "sec-vuln", name: "Vulnerability Posture", target: 8, category: "security" },

  // Developer Experience (6)
  { id: "dx-api", name: "API Design", target: 8, category: "developer-experience" },
  { id: "dx-docs", name: "Documentation", target: 7, category: "developer-experience" },
  { id: "dx-cli", name: "CLI / Tooling", target: 8, category: "developer-experience" },
  { id: "dx-debug", name: "Debugging", target: 7, category: "developer-experience" },
  { id: "dx-testing", name: "Test Coverage", target: 8, category: "developer-experience" },
  { id: "dx-contrib", name: "Contributor Experience", target: 7, category: "developer-experience" },

  // Operations (6)
  { id: "ops-deploy", name: "Deployment", target: 8, category: "operations" },
  { id: "ops-monitor", name: "Observability / Monitoring", target: 8, category: "operations" },
  { id: "ops-scaling", name: "Auto-Scaling", target: 7, category: "operations" },
  { id: "ops-config", name: "Configuration", target: 7, category: "operations" },
  { id: "ops-migrate", name: "Migration Path", target: 6, category: "operations" },
  { id: "ops-cost", name: "Cost Efficiency", target: 7, category: "operations" },
];
