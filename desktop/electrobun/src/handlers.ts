// Electrobun IPC handlers for AgilePlus Desktop
// Reads/writes spec files, pillar scorecard JSON, and sprint data
// from the nanovms repo's agileplus/ directory.

import { readFileSync, writeFileSync, existsSync, readdirSync, mkdirSync } from "fs";
import { join, resolve } from "path";

// ─── Configuration ────────────────────────────────────────────────────────────
// Resolve the repo root relative to the desktop app location.
// In production this is the directory containing the app binary.
const REPO_ROOT = resolve(process.env.AGILEPLUS_REPO ?? process.cwd(), "..", "..");
const AGILEPLUS_DIR = join(REPO_ROOT, "agileplus");
const SPEC_DIR = join(AGILEPLUS_DIR, "specs");
const SCORECARD_PATH = join(AGILEPLUS_DIR, "pillars", "31-pillar-scorecard.json");
const SPRINT_DATA_PATH = join(AGILEPLUS_DIR, ".desktop", "sprint-data.json");
const BACKLOG_DATA_PATH = join(AGILEPLUS_DIR, ".desktop", "backlog-data.json");
const GATES_STATUS_PATH = join(AGILEPLUS_DIR, ".desktop", "gates-status.json");
const PILLARS_DATA_PATH = join(AGILEPLUS_DIR, ".desktop", "pillars-data.json");
const SPEC_SNAPSHOTS_DIR = join(AGILEPLUS_DIR, ".desktop", "spec-snapshots");
const GATES_HISTORY_PATH = join(AGILEPLUS_DIR, ".desktop", "gates-history.json");
const TREND_DATA_PATH = join(AGILEPLUS_DIR, ".desktop", "trend-data.json");
const COMPARISON_SNAPSHOTS_PATH = join(AGILEPLUS_DIR, ".desktop", "comparison-snapshots.json");
const BACKLOG_BOARD_PATH = join(AGILEPLUS_DIR, ".desktop", "backlog-board.json");

// ─── Helpers ──────────────────────────────────────────────────────────────────
function ensureDir(dir: string): void {
  if (!existsSync(dir)) {
    mkdirSync(dir, { recursive: true });
  }
}

function loadJson<T>(path: string, fallback: T): T {
  try {
    if (!existsSync(path)) return fallback;
    const raw = readFileSync(path, "utf-8");
    return JSON.parse(raw) as T;
  } catch {
    return fallback;
  }
}

function saveJson(path: string, data: unknown): void {
  ensureDir(join(path, ".."));
  writeFileSync(path, JSON.stringify(data, null, 2), "utf-8");
}

function uid(): string {
  return Date.now().toString(36) + Math.random().toString(36).slice(2, 8);
}

function nowISO(): string {
  return new Date().toISOString();
}

// ─── Default pillars (matching the 31-pillar model) ──────────────────────────
const DEFAULT_PILLARS = [
  { id: "code-quality", name: "Code Quality", target: 8, category: "performance" },
  { id: "tests", name: "Tests", target: 8, category: "reliability" },
  { id: "docs", name: "Docs", target: 7, category: "developer-experience" },
  { id: "ci-cd", name: "CI/CD", target: 8, category: "operations" },
  { id: "security", name: "Security", target: 9, category: "security" },
  { id: "architecture", name: "Architecture", target: 8, category: "developer-experience" },
  { id: "performance", name: "Performance", target: 8, category: "performance" },
  { id: "dx", name: "DX", target: 8, category: "developer-experience" },
  { id: "releases", name: "Releases", target: 7, category: "operations" },
  { id: "monitoring", name: "Monitoring", target: 8, category: "operations" },
  { id: "deps", name: "Dependencies", target: 7, category: "reliability" },
  { id: "reviews", name: "Reviews", target: 7, category: "developer-experience" },
  { id: "branch-mgmt", name: "Branch Mgmt", target: 7, category: "operations" },
  { id: "issue-tracking", name: "Issue Tracking", target: 7, category: "operations" },
  { id: "agile-pm", name: "Agile PM", target: 7, category: "operations" },
  { id: "accessibility", name: "Accessibility", target: 7, category: "developer-experience" },
  { id: "i18n", name: "i18n", target: 7, category: "developer-experience" },
  { id: "mobile", name: "Mobile", target: 7, category: "performance" },
  { id: "api", name: "API", target: 8, category: "performance" },
  { id: "db", name: "DB", target: 7, category: "reliability" },
  { id: "errors", name: "Errors", target: 7, category: "reliability" },
  { id: "logging", name: "Logging", target: 7, category: "operations" },
  { id: "config", name: "Config", target: 7, category: "operations" },
  { id: "env", name: "Env", target: 7, category: "operations" },
  { id: "build", name: "Build", target: 8, category: "performance" },
  { id: "pkg", name: "Packaging", target: 7, category: "operations" },
  { id: "license", name: "License", target: 8, category: "security" },
  { id: "community", name: "Community", target: 7, category: "developer-experience" },
  { id: "contributing", name: "Contributing", target: 7, category: "developer-experience" },
  { id: "coc", name: "Code of Conduct", target: 7, category: "security" },
  { id: "vuln-disc", name: "Vuln Disclosure", target: 8, category: "security" },
];

// ─── IPC Handler Registry ─────────────────────────────────────────────────────
// Each handler receives (channel, payload) and returns a response.

type Handler = (payload: Record<string, unknown>) => Promise<unknown>;

const handlers: Record<string, Handler> = {
  // ─── Specs ──────────────────────────────────────────────────────────
  "specs:list": async () => {
    ensureDir(SPEC_DIR);
    const files = readdirSync(SPEC_DIR).filter((f) => f.endsWith(".md"));
    const specs = files.map((file) => {
      const id = file.replace(/\.md$/, "");
      const content = readFileSync(join(SPEC_DIR, file), "utf-8");
      const titleMatch = content.match(/^#\s+(.+)/m);
      return {
        id,
        title: titleMatch?.[1] ?? id,
        content,
        pillarScores: {},
        createdAt: nowISO(),
        updatedAt: nowISO(),
      };
    });
    return { ok: true, data: specs };
  },

  "specs:get": async (payload) => {
    const id = payload.id as string;
    const filePath = join(SPEC_DIR, `${id}.md`);
    if (!existsSync(filePath)) return { ok: false, error: "Spec not found" };
    const content = readFileSync(filePath, "utf-8");
    const titleMatch = content.match(/^#\s+(.+)/m);
    return {
      ok: true,
      data: {
        id,
        title: titleMatch?.[1] ?? id,
        content,
        pillarScores: {},
        createdAt: nowISO(),
        updatedAt: nowISO(),
      },
    };
  },

  "specs:create": async (payload) => {
    ensureDir(SPEC_DIR);
    const title = (payload.title as string) || "Untitled Spec";
    const content = (payload.content as string) || `# ${title}\n\n`;
    const id = uid();
    const filePath = join(SPEC_DIR, `${id}.md`);
    writeFileSync(filePath, content, "utf-8");
    return {
      ok: true,
      data: { id, title, content, pillarScores: {}, createdAt: nowISO(), updatedAt: nowISO() },
    };
  },

  "specs:update": async (payload) => {
    const id = payload.id as string;
    const updates = payload.updates as Record<string, unknown>;
    const filePath = join(SPEC_DIR, `${id}.md`);
    if (!existsSync(filePath)) return { ok: false, error: "Spec not found" };
    if (updates.content != null) {
      writeFileSync(filePath, updates.content as string, "utf-8");
    }
    return { ok: true, data: { id, ...updates, updatedAt: nowISO() } };
  },

  "specs:delete": async (payload) => {
    const id = payload.id as string;
    const filePath = join(SPEC_DIR, `${id}.md`);
    if (existsSync(filePath)) {
      const { unlinkSync } = await import("fs");
      unlinkSync(filePath);
    }
    return { ok: true };
  },

  // ─── Pillars ────────────────────────────────────────────────────────
  "pillars:list": async () => {
    const data = loadJson<Record<string, unknown>[]>(PILLARS_DATA_PATH, []);
    if (data.length === 0) {
      // Seed from the 31-pillar scorecard JSON
      const scorecard = loadJson<{ pillars: Record<string, unknown>[] } | null>(SCORECARD_PATH, null);
      if (scorecard?.pillars) {
        const pillars = scorecard.pillars.map((p) => ({
          id: p.id,
          name: p.name,
          score: p.score,
          target: DEFAULT_PILLARS.find((d) => d.id === p.id)?.target ?? 7,
          category: DEFAULT_PILLARS.find((d) => d.id === p.id)?.category ?? "operations",
        }));
        saveJson(PILLARS_DATA_PATH, pillars);
        return { ok: true, data: pillars };
      }
      // Fallback: create defaults with score 5
      const defaults = DEFAULT_PILLARS.map((p) => ({ ...p, score: 5 }));
      saveJson(PILLARS_DATA_PATH, defaults);
      return { ok: true, data: defaults };
    }
    return { ok: true, data };
  },

  "pillars:update": async (payload) => {
    const id = payload.id as string;
    const score = Number(payload.score) || 0;
    const pillars = loadJson<Record<string, unknown>[]>(PILLARS_DATA_PATH, []);
    const idx = pillars.findIndex((p) => p.id === id);
    if (idx === -1) return { ok: false, error: "Pillar not found" };
    pillars[idx].score = Math.max(0, Math.min(10, score));
    saveJson(PILLARS_DATA_PATH, pillars);
    return { ok: true, data: pillars[idx] };
  },

  "pillars:scorecard": async () => {
    // Read directly from agileplus/pillars/31-pillar-scorecard.json
    if (existsSync(SCORECARD_PATH)) {
      const data = loadJson<unknown>(SCORECARD_PATH, null);
      return { ok: true, data };
    }
    return { ok: true, data: null };
  },

  "pillars:scorecard:save": async (payload) => {
    ensureDir(join(SCORECARD_PATH, ".."));
    saveJson(SCORECARD_PATH, payload.scorecard);
    return { ok: true };
  },

  // ─── Sprints ────────────────────────────────────────────────────────
  "sprints:list": async () => {
    const data = loadJson<Record<string, unknown>[]>(SPRINT_DATA_PATH, []);
    return { ok: true, data };
  },

  "sprints:active": async () => {
    const sprints = loadJson<Record<string, unknown>[]>(SPRINT_DATA_PATH, []);
    const active = sprints.find((s) => s.status === "active");
    return { ok: true, data: active ?? null };
  },

  "sprints:create": async (payload) => {
    const sprints = loadJson<Record<string, unknown>[]>(SPRINT_DATA_PATH, []);
    const newSprint = {
      id: uid(),
      name: payload.name ?? "Sprint",
      startDate: payload.startDate ?? nowISO().slice(0, 10),
      endDate: payload.endDate ?? nowISO().slice(0, 10),
      status: "planned",
      velocity: 0,
      goal: payload.goal ?? "",
    };
    sprints.push(newSprint);
    saveJson(SPRINT_DATA_PATH, sprints);
    return { ok: true, data: newSprint };
  },

  "sprints:update": async (payload) => {
    const id = payload.id as string;
    const updates = payload.updates as Record<string, unknown>;
    const sprints = loadJson<Record<string, unknown>[]>(SPRINT_DATA_PATH, []);
    const idx = sprints.findIndex((s) => s.id === id);
    if (idx === -1) return { ok: false, error: "Sprint not found" };
    sprints[idx] = { ...sprints[idx], ...updates };
    saveJson(SPRINT_DATA_PATH, sprints);
    return { ok: true, data: sprints[idx] };
  },

  // ─── Backlog ────────────────────────────────────────────────────────
  "backlog:list": async () => {
    const data = loadJson<Record<string, unknown>[]>(BACKLOG_DATA_PATH, []);
    return { ok: true, data };
  },

  "backlog:create": async (payload) => {
    const backlog = loadJson<Record<string, unknown>[]>(BACKLOG_DATA_PATH, []);
    const newItem = {
      id: uid(),
      title: payload.title ?? "New item",
      priority: payload.priority ?? "P2",
      status: "todo",
      specId: payload.specId ?? undefined,
      pillar: payload.pillar ?? undefined,
      createdAt: nowISO(),
    };
    backlog.push(newItem);
    saveJson(BACKLOG_DATA_PATH, backlog);
    return { ok: true, data: newItem };
  },

  "backlog:update": async (payload) => {
    const id = payload.id as string;
    const updates = payload.updates as Record<string, unknown>;
    const backlog = loadJson<Record<string, unknown>[]>(BACKLOG_DATA_PATH, []);
    const idx = backlog.findIndex((b) => b.id === id);
    if (idx === -1) return { ok: false, error: "Backlog item not found" };
    backlog[idx] = { ...backlog[idx], ...updates };
    saveJson(BACKLOG_DATA_PATH, backlog);
    return { ok: true, data: backlog[idx] };
  },

  "backlog:delete": async (payload) => {
    const id = payload.id as string;
    const backlog = loadJson<Record<string, unknown>[]>(BACKLOG_DATA_PATH, []);
    const filtered = backlog.filter((b) => b.id !== id);
    saveJson(BACKLOG_DATA_PATH, filtered);
    return { ok: true };
  },

  "backlog:reorder": async (payload) => {
    const ids = payload.orderedIds as string[];
    if (!Array.isArray(ids)) return { ok: false, error: "orderedIds must be an array" };
    const backlog = loadJson<Record<string, unknown>[]>(BACKLOG_DATA_PATH, []);
    const map = new Map(backlog.map((b) => [b.id, b]));
    const reordered = ids.map((id) => map.get(id)).filter(Boolean);
    saveJson(BACKLOG_DATA_PATH, reordered);
    return { ok: true };
  },

  // ─── Quality Gates ──────────────────────────────────────────────────
  "gates:list": async () => {
    // Read from agileplus/quality-gates.yml
    // For M1, return the default gates (YAML parsing requires a dependency)
    const gates = [
      { id: "lint", name: "Lint", description: "Static analysis for Go, Rust, YAML, Markdown, and shell.", pillar: "code-quality", required: true, command: "" },
      { id: "test", name: "Test", description: "Unit + integration tests for Go and Rust workspaces.", pillar: "tests", required: true, command: "" },
      { id: "security", name: "Security", description: "Supply-chain and secret scanning.", pillar: "security", required: true, command: "" },
      { id: "build", name: "Build", description: "Release-mode build for both Go and Rust binaries.", pillar: "build", required: true, command: "" },
      { id: "docs-build", name: "Docs Build", description: "VitePress documentation site builds cleanly.", pillar: "docs", required: false, command: "" },
      { id: "coverage", name: "Coverage", description: "Test coverage above floor; tracked, not blocking.", pillar: "tests", required: false, floor: 60, command: "" },
    ];
    return { ok: true, data: gates };
  },

  "gates:status": async () => {
    const data = loadJson<Record<string, unknown>[]>(GATES_STATUS_PATH, []);
    return { ok: true, data };
  },

  "gates:history": async () => {
    const data = loadJson<Record<string, unknown>[]>(GATES_HISTORY_PATH, []);
    return { ok: true, data };
  },

  // ─── Spec Snapshots (version history) ────────────────────────────
  "versions:list": async (payload) => {
    const specId = payload.specId as string;
    if (!specId) return { ok: true, data: [] };
    const filePath = join(SPEC_SNAPSHOTS_DIR, `${specId}.json`);
    const data = loadJson<Record<string, unknown>[]>(filePath, []);
    return { ok: true, data };
  },

  "versions:save": async (payload) => {
    const specId = payload.specId as string;
    if (!specId) return { ok: false, error: "No specId" };
    ensureDir(SPEC_SNAPSHOTS_DIR);
    const filePath = join(SPEC_SNAPSHOTS_DIR, `${specId}.json`);
    const versions = loadJson<Record<string, unknown>[]>(filePath, []);
    const newVersion = {
      id: uid(),
      specId,
      timestamp: nowISO(),
      title: payload.title ?? "",
      content: payload.content ?? "",
      wordCount: payload.wordCount ?? 0,
    };
    versions.unshift(newVersion);
    if (versions.length > 50) versions.length = 50;
    saveJson(filePath, versions);
    return { ok: true, data: newVersion };
  },

  // ─── Comparison Snapshots ─────────────────────────────────────────
  "comparison:snapshots": async () => {
    const data = loadJson<Record<string, unknown>[]>(COMPARISON_SNAPSHOTS_PATH, []);
    return { ok: true, data };
  },

  "comparison:snapshot:save": async (payload) => {
    const snapshots = loadJson<Record<string, unknown>[]>(COMPARISON_SNAPSHOTS_PATH, []);
    snapshots.push({
      label: payload.label ?? "Snapshot",
      date: nowISO(),
      scores: payload.scores ?? {},
    });
    saveJson(COMPARISON_SNAPSHOTS_PATH, snapshots);
    return { ok: true };
  },

  // ─── Trend Data ──────────────────────────────────────────────────
  "trend:data": async () => {
    const data = loadJson<Record<string, unknown>[]>(TREND_DATA_PATH, []);
    return { ok: true, data };
  },

  "trend:data:save": async (payload) => {
    saveJson(TREND_DATA_PATH, payload.points ?? []);
    return { ok: true };
  },

  // ─── Backlog Board ───────────────────────────────────────────────
  "backlog:board:list": async () => {
    const data = loadJson<Record<string, unknown>[]>(BACKLOG_BOARD_PATH, []);
    return { ok: true, data };
  },

  "backlog:board:create": async (payload) => {
    const board = loadJson<Record<string, unknown>[]>(BACKLOG_BOARD_PATH, []);
    const newItem = {
      id: uid(),
      title: payload.title ?? "New item",
      priority: payload.priority ?? "P2",
      points: payload.points ?? 3,
      assignee: payload.assignee ?? "Unassigned",
      pillar: payload.pillar ?? undefined,
      specId: payload.specId ?? undefined,
      createdAt: nowISO(),
    };
    board.push(newItem);
    saveJson(BACKLOG_BOARD_PATH, board);
    return { ok: true, data: newItem };
  },

  "backlog:board:update": async (payload) => {
    const id = payload.id as string;
    const updates = payload.updates as Record<string, unknown>;
    const board = loadJson<Record<string, unknown>[]>(BACKLOG_BOARD_PATH, []);
    const idx = board.findIndex((b) => b.id === id);
    if (idx === -1) return { ok: false, error: "Board item not found" };
    board[idx] = { ...board[idx], ...updates };
    saveJson(BACKLOG_BOARD_PATH, board);
    return { ok: true, data: board[idx] };
  },

  "backlog:board:delete": async (payload) => {
    const id = payload.id as string;
    const board = loadJson<Record<string, unknown>[]>(BACKLOG_BOARD_PATH, []);
    const filtered = board.filter((b) => b.id !== id);
    saveJson(BACKLOG_BOARD_PATH, filtered);
    return { ok: true };
  },

  // ─── Filesystem ─────────────────────────────────────────────────────
  "fs:read": async (payload) => {
    const filePath = payload.path as string;
    if (!filePath) return { ok: false, error: "No path specified" };
    const resolved = resolve(REPO_ROOT, filePath);
    if (!existsSync(resolved)) return { ok: false, error: "File not found" };
    const content = readFileSync(resolved, "utf-8");
    return { ok: true, data: content };
  },

  "fs:write": async (payload) => {
    const filePath = payload.path as string;
    const content = payload.content as string;
    if (!filePath) return { ok: false, error: "No path specified" };
    const resolved = resolve(REPO_ROOT, filePath);
    ensureDir(join(resolved, ".."));
    writeFileSync(resolved, content, "utf-8");
    return { ok: true };
  },

  "fs:list": async (payload) => {
    const dirPath = (payload.path as string) ?? ".";
    const resolved = resolve(REPO_ROOT, dirPath);
    if (!existsSync(resolved)) return { ok: true, data: [] };
    const entries = readdirSync(resolved, { withFileTypes: true });
    return {
      ok: true,
      data: entries.map((e) => ({
        name: e.name,
        isDirectory: e.isDirectory(),
      })),
    };
  },
};

// ─── Register handlers with Electrobun IPC ───────────────────────────────────

export function registerAgilePlusHandlers(ipc: { on: (channel: string, handler: (payload: unknown) => Promise<unknown>) => void }): void {
  for (const [channel, handler] of Object.entries(handlers)) {
    ipc.on(channel, async (payload: unknown) => {
      try {
        return await handler(payload as Record<string, unknown>);
      } catch (err) {
        console.error(`[agileplus] handler error on ${channel}:`, err);
        return { ok: false, error: String(err) };
      }
    });
  }
  console.log("[agileplus] IPC handlers registered for", Object.keys(handlers).length, "channels");
}

export { handlers, REPO_ROOT, AGILEPLUS_DIR };
