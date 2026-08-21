import type { Spec, Pillar, Sprint, BacklogItem } from "./types";
import { DEFAULT_PILLARS } from "./types";

// ─── Helpers ──────────────────────────────────────────────────────────────────
function uid(): string {
  return Date.now().toString(36) + Math.random().toString(36).slice(2, 8);
}

function nowISO(): string {
  return new Date().toISOString();
}

function load<T>(key: string, fallback: T): T {
  try {
    const raw = localStorage.getItem(`agileplus:${key}`);
    return raw ? JSON.parse(raw) : fallback;
  } catch {
    return fallback;
  }
}

function save(key: string, value: unknown): void {
  localStorage.setItem(`agileplus:${key}`, JSON.stringify(value));
}

// ─── Specs ───────────────────────────────────────────────────────────────────
export function getSpecs(): Spec[] {
  return load<Spec[]>("specs", []);
}

export function getSpec(id: string): Spec | undefined {
  return getSpecs().find((s) => s.id === id);
}

export function createSpec(title: string, content: string = ""): Spec {
  const specs = getSpecs();
  const spec: Spec = {
    id: uid(),
    title,
    content,
    pillarScores: {},
    createdAt: nowISO(),
    updatedAt: nowISO(),
  };
  specs.push(spec);
  save("specs", specs);
  return spec;
}

export function updateSpec(id: string, updates: Partial<Omit<Spec, "id" | "createdAt">>): Spec | null {
  const specs = getSpecs();
  const idx = specs.findIndex((s) => s.id === id);
  if (idx === -1) return null;
  specs[idx] = { ...specs[idx], ...updates, updatedAt: nowISO() };
  save("specs", specs);
  return specs[idx];
}

export function deleteSpec(id: string): boolean {
  const specs = getSpecs();
  const filtered = specs.filter((s) => s.id !== id);
  if (filtered.length === specs.length) return false;
  save("specs", filtered);
  return true;
}

// ─── Pillars ─────────────────────────────────────────────────────────────────
export function getPillars(): Pillar[] {
  return load<Pillar[]>("pillars", []);
}

export function initPillarsIfEmpty(): void {
  if (getPillars().length > 0) return;
  const pillars: Pillar[] = DEFAULT_PILLARS.map((p) => ({
    ...p,
    score: 5,
  }));
  save("pillars", pillars);
}

export function updatePillar(id: string, score: number): void {
  const pillars = getPillars();
  const idx = pillars.findIndex((p) => p.id === id);
  if (idx === -1) return;
  pillars[idx].score = Math.max(0, Math.min(10, score));
  save("pillars", pillars);
}

export function getOverallScore(): number {
  const pillars = getPillars();
  if (pillars.length === 0) return 0;
  return pillars.reduce((sum, p) => sum + p.score, 0) / pillars.length;
}

// ─── Sprints ─────────────────────────────────────────────────────────────────
export function getSprints(): Sprint[] {
  return load<Sprint[]>("sprints", []);
}

export function getActiveSprint(): Sprint | null {
  return getSprints().find((s) => s.status === "active") ?? null;
}

export function createSprint(name: string, startDate: string, endDate: string, goal: string): Sprint {
  const sprints = getSprints();
  const sprint: Sprint = {
    id: uid(),
    name,
    startDate,
    endDate,
    status: "planned",
    velocity: 0,
    goal,
  };
  sprints.push(sprint);
  save("sprints", sprints);
  return sprint;
}

export function updateSprint(id: string, updates: Partial<Omit<Sprint, "id">>): Sprint | null {
  const sprints = getSprints();
  const idx = sprints.findIndex((s) => s.id === id);
  if (idx === -1) return null;
  sprints[idx] = { ...sprints[idx], ...updates };
  save("sprints", sprints);
  return sprints[idx];
}

// ─── Backlog ─────────────────────────────────────────────────────────────────
export function getBacklog(): BacklogItem[] {
  return load<BacklogItem[]>("backlog", []);
}

export function createBacklogItem(
  title: string,
  priority: BacklogItem["priority"],
  specId?: string,
): BacklogItem {
  const backlog = getBacklog();
  const item: BacklogItem = {
    id: uid(),
    title,
    priority,
    status: "todo",
    specId,
    createdAt: nowISO(),
  };
  backlog.push(item);
  save("backlog", backlog);
  return item;
}

export function updateBacklogItem(id: string, updates: Partial<Omit<BacklogItem, "id" | "createdAt">>): BacklogItem | null {
  const backlog = getBacklog();
  const idx = backlog.findIndex((b) => b.id === id);
  if (idx === -1) return null;
  backlog[idx] = { ...backlog[idx], ...updates };
  save("backlog", backlog);
  return backlog[idx];
}

export function deleteBacklogItem(id: string): boolean {
  const backlog = getBacklog();
  const filtered = backlog.filter((b) => b.id !== id);
  if (filtered.length === backlog.length) return false;
  save("backlog", filtered);
  return true;
}

export function reorderBacklog(orderedIds: string[]): void {
  const backlog = getBacklog();
  const map = new Map(backlog.map((b) => [b.id, b]));
  const reordered = orderedIds.map((id) => map.get(id)).filter(Boolean) as BacklogItem[];
  save("backlog", reordered);
}

// ─── Export all for convenience ──────────────────────────────────────────────
export function exportAll(): string {
  return JSON.stringify(
    {
      specs: getSpecs(),
      pillars: getPillars(),
      sprints: getSprints(),
      backlog: getBacklog(),
      exportedAt: nowISO(),
    },
    null,
    2,
  );
}

export function importAll(json: string): boolean {
  try {
    const data = JSON.parse(json);
    if (data.specs) save("specs", data.specs);
    if (data.pillars) save("pillars", data.pillars);
    if (data.sprints) save("sprints", data.sprints);
    if (data.backlog) save("backlog", data.backlog);
    return true;
  } catch {
    return false;
  }
}
