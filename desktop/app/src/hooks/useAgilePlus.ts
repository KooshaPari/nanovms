import { useCallback, useRef } from "react";
import type { IpcResponse } from "../types";

// ─── Electrobun IPC bridge ───────────────────────────────────────────────────
//
// When running inside Electrobun, window.electrobun exposes an ipc object
// with a request() method.  In the browser (e.g. `vite dev`) we fall back
// to localStorage so the app is dogfoodable without the native shell.

declare global {
  interface Window {
    electrobun?: {
      ipc: {
        request: (channel: string, data: unknown) => Promise<unknown>;
      };
    };
  }
}

function hasElectrobun(): boolean {
  return typeof window !== "undefined" && !!window.electrobun?.ipc;
}

// ─── useAgilePlus hook ───────────────────────────────────────────────────────

export function useAgilePlus() {
  const requestId = useRef(0);

  /** Send an IPC request to the Electrobun backend (or localStorage fallback). */
  const request = useCallback(
    async <T = unknown>(channel: string, data?: unknown): Promise<IpcResponse<T>> => {
      if (hasElectrobun()) {
        try {
          const id = ++requestId.current;
          const result = (await window.electrobun!.ipc.request(channel, {
            id,
            ...data,
          })) as IpcResponse<T>;
          return result;
        } catch (err) {
          return { ok: false, error: String(err) };
        }
      }

      // ─── Browser fallback (localStorage) ──────────────────────────────
      return localStorageFallback<T>(channel, data);
    },
    [],
  );

  return { request };
}

// ─── localStorage fallback for running outside Electrobun ─────────────────────
// Mirrors the IPC channels so the app is fully usable in a plain browser.

function lsGet<T>(key: string, fallback: T): T {
  try {
    const raw = localStorage.getItem(`agileplus:${key}`);
    return raw ? JSON.parse(raw) : fallback;
  } catch {
    return fallback;
  }
}

function lsSet(key: string, value: unknown): void {
  localStorage.setItem(`agileplus:${key}`, JSON.stringify(value));
}

function uid(): string {
  return Date.now().toString(36) + Math.random().toString(36).slice(2, 8);
}

function nowISO(): string {
  return new Date().toISOString();
}

async function localStorageFallback<T>(
  channel: string,
  data?: unknown,
): Promise<IpcResponse<T>> {
  const d = data as Record<string, unknown> | undefined;

  switch (channel) {
    // ─── Specs ────────────────────────────────────────────────────────
    case "specs:list": {
      return { ok: true, data: lsGet("specs", []) as T };
    }
    case "specs:get": {
      const specs = lsGet<Array<Record<string, unknown>>>("specs", []);
      const spec = specs.find((s) => s.id === d?.id);
      return spec
        ? { ok: true, data: spec as T }
        : { ok: false, error: "Spec not found" };
    }
    case "specs:create": {
      const specs = lsGet<Array<Record<string, unknown>>>("specs", []);
      const newSpec = {
        id: uid(),
        title: d?.title ?? "Untitled Spec",
        content: d?.content ?? "",
        pillarScores: {},
        createdAt: nowISO(),
        updatedAt: nowISO(),
      };
      specs.push(newSpec);
      lsSet("specs", specs);
      return { ok: true, data: newSpec as T };
    }
    case "specs:update": {
      const specs = lsGet<Array<Record<string, unknown>>>("specs", []);
      const idx = specs.findIndex((s) => s.id === d?.id);
      if (idx === -1) return { ok: false, error: "Spec not found" };
      specs[idx] = { ...specs[idx], ...d?.updates, updatedAt: nowISO() };
      lsSet("specs", specs);
      return { ok: true, data: specs[idx] as T };
    }
    case "specs:delete": {
      const specs = lsGet<Array<Record<string, unknown>>>("specs", []);
      lsSet(
        "specs",
        specs.filter((s) => s.id !== d?.id),
      );
      return { ok: true, data: undefined as T };
    }

    // ─── Pillars ──────────────────────────────────────────────────────
    case "pillars:list": {
      return { ok: true, data: lsGet("pillars", []) as T };
    }
    case "pillars:update": {
      const pillars = lsGet<Array<Record<string, unknown>>>("pillars", []);
      const pidx = pillars.findIndex((p) => p.id === d?.id);
      if (pidx === -1) return { ok: false, error: "Pillar not found" };
      pillars[pidx].score = Math.max(0, Math.min(10, Number(d?.score) || 0));
      lsSet("pillars", pillars);
      return { ok: true, data: pillars[pidx] as T };
    }
    case "pillars:scorecard": {
      // Return the full 31-pillar scorecard data
      const scorecard = lsGet("scorecard", null);
      return { ok: true, data: scorecard as T };
    }
    case "pillars:scorecard:save": {
      lsSet("scorecard", d?.scorecard);
      return { ok: true, data: undefined as T };
    }

    // ─── Sprints ──────────────────────────────────────────────────────
    case "sprints:list": {
      return { ok: true, data: lsGet("sprints", []) as T };
    }
    case "sprints:active": {
      const sprints = lsGet<Array<Record<string, unknown>>>("sprints", []);
      const active = sprints.find((s) => s.status === "active");
      return { ok: true, data: (active ?? null) as T };
    }
    case "sprints:create": {
      const sprints = lsGet<Array<Record<string, unknown>>>("sprints", []);
      const newSprint = {
        id: uid(),
        name: d?.name ?? "Sprint",
        startDate: d?.startDate ?? nowISO().slice(0, 10),
        endDate: d?.endDate ?? nowISO().slice(0, 10),
        status: "planned",
        velocity: 0,
        goal: d?.goal ?? "",
      };
      sprints.push(newSprint);
      lsSet("sprints", sprints);
      return { ok: true, data: newSprint as T };
    }
    case "sprints:update": {
      const sprints = lsGet<Array<Record<string, unknown>>>("sprints", []);
      const sidx = sprints.findIndex((s) => s.id === d?.id);
      if (sidx === -1) return { ok: false, error: "Sprint not found" };
      sprints[sidx] = { ...sprints[sidx], ...(d?.updates as object) };
      lsSet("sprints", sprints);
      return { ok: true, data: sprints[sidx] as T };
    }

    // ─── Backlog ──────────────────────────────────────────────────────
    case "backlog:list": {
      return { ok: true, data: lsGet("backlog", []) as T };
    }
    case "backlog:create": {
      const backlog = lsGet<Array<Record<string, unknown>>>("backlog", []);
      const newItem = {
        id: uid(),
        title: d?.title ?? "New item",
        priority: d?.priority ?? "P2",
        status: "todo",
        specId: d?.specId ?? undefined,
        pillar: d?.pillar ?? undefined,
        createdAt: nowISO(),
      };
      backlog.push(newItem);
      lsSet("backlog", backlog);
      return { ok: true, data: newItem as T };
    }
    case "backlog:update": {
      const backlog = lsGet<Array<Record<string, unknown>>>("backlog", []);
      const bidx = backlog.findIndex((b) => b.id === d?.id);
      if (bidx === -1) return { ok: false, error: "Backlog item not found" };
      backlog[bidx] = { ...backlog[bidx], ...(d?.updates as object) };
      lsSet("backlog", backlog);
      return { ok: true, data: backlog[bidx] as T };
    }
    case "backlog:delete": {
      const backlog = lsGet<Array<Record<string, unknown>>>("backlog", []);
      lsSet(
        "backlog",
        backlog.filter((b) => b.id !== d?.id),
      );
      return { ok: true, data: undefined as T };
    }
    case "backlog:reorder": {
      const backlog = lsGet<Array<Record<string, unknown>>>("backlog", []);
      const ids = d?.orderedIds as string[];
      if (Array.isArray(ids)) {
        const map = new Map(backlog.map((b) => [b.id, b]));
        const reordered = ids.map((id) => map.get(id)).filter(Boolean);
        lsSet("backlog", reordered);
      }
      return { ok: true, data: undefined as T };
    }

    // ─── Quality Gates ────────────────────────────────────────────────
    case "gates:list": {
      return { ok: true, data: lsGet("gates", []) as T };
    }
    case "gates:status": {
      return { ok: true, data: lsGet("gates_status", []) as T };
    }

    // ─── Filesystem ───────────────────────────────────────────────────
    case "fs:read": {
      // In localStorage fallback, return empty — real file I/O happens
      // through the Electrobun handler.
      return { ok: true, data: "" as T };
    }
    case "fs:write": {
      return { ok: true, data: undefined as T };
    }
    case "fs:list": {
      return { ok: true, data: [] as T };
    }

    default:
      return { ok: false, error: `Unknown channel: ${channel}` };
  }
}
