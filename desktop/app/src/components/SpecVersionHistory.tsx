import { useState, useEffect, useCallback } from "react";
import {
  HiOutlineClock,
  HiOutlineRefresh,
  HiOutlineDocumentText,
} from "react-icons/hi";
import type { IpcResponse } from "../types";

interface Props {
  request: <T = unknown>(channel: string, data?: unknown) => Promise<IpcResponse<T>>;
}

interface SpecVersion {
  id: string;
  specId: string;
  timestamp: string;
  title: string;
  content: string;
  wordCount: number;
}

// ─── localStorage helpers for version history ───────────────────────────────
function getVersions(specId: string): SpecVersion[] {
  try {
    const raw = localStorage.getItem(`agileplus:versions:${specId}`);
    return raw ? JSON.parse(raw) : [];
  } catch {
    return [];
  }
}

function saveVersion(specId: string, version: SpecVersion): void {
  const versions = getVersions(specId);
  versions.unshift(version); // newest first
  // Keep at most 50 versions
  if (versions.length > 50) versions.length = 50;
  localStorage.setItem(`agileplus:versions:${specId}`, JSON.stringify(versions));
}

function deleteVersion(specId: string, versionId: string): void {
  const versions = getVersions(specId).filter((v) => v.id !== versionId);
  localStorage.setItem(`agileplus:versions:${specId}`, JSON.stringify(versions));
}

function uid(): string {
  return Date.now().toString(36) + Math.random().toString(36).slice(2, 8);
}

export default function SpecVersionHistory({ request }: Props) {
  const [specId, setSpecId] = useState<string>("");
  const [versions, setVersions] = useState<SpecVersion[]>([]);
  const [selectedVersionId, setSelectedVersionId] = useState<string | null>(null);
  const [diffText, setDiffText] = useState<string | null>(null);
  const [specList, setSpecList] = useState<Array<{ id: string; title: string; content: string }>>([]);

  useEffect(() => {
    request<Array<{ id: string; title: string; content: string }>>("specs:list").then((r) => {
      if (r.ok && Array.isArray(r.data)) setSpecList(r.data);
    });
  }, [request]); // eslint-disable-line react-hooks/exhaustive-deps

  const loadVersions = useCallback((sid: string) => {
    setSpecId(sid);
    setVersions(getVersions(sid));
    setSelectedVersionId(null);
    setDiffText(null);
  }, []);

  useEffect(() => {
    if (specList.length > 0 && !specId) {
      loadVersions(specList[0].id);
    }
  }, [specList, specId, loadVersions]);

  // Create a snapshot of the current version
  const handleSnapshot = () => {
    if (!specId) return;
    const spec = specList.find((s) => s.id === specId);
    if (!spec) return;

    const version: SpecVersion = {
      id: uid(),
      specId,
      timestamp: new Date().toISOString(),
      title: spec.title,
      content: spec.content,
      wordCount: spec.content.split(/\s+/).filter((w) => w.length > 0).length,
    };

    saveVersion(specId, version);
    setVersions(getVersions(specId));
  };

  // Compute simple line diff
  const computeDiff = (oldContent: string, newContent: string): string => {
    const oldLines = oldContent.split("\n");
    const newLines = newContent.split("\n");
    const diff: string[] = [];

    const maxLen = Math.max(oldLines.length, newLines.length);
    for (let i = 0; i < maxLen; i++) {
      const oldLine = oldLines[i];
      const newLine = newLines[i];

      if (oldLine === undefined) {
        diff.push(`+ ${newLine}`);
      } else if (newLine === undefined) {
        diff.push(`- ${oldLine}`);
      } else if (oldLine !== newLine) {
        diff.push(`- ${oldLine}`);
        diff.push(`+ ${newLine}`);
      } else {
        diff.push(`  ${oldLine}`);
      }
    }

    return diff.join("\n");
  };

  const handleViewDiff = (version: SpecVersion) => {
    setSelectedVersionId(version.id);
    const spec = specList.find((s) => s.id === specId);
    if (spec) {
      setDiffText(computeDiff(version.content, spec.content));
    }
  };

  const handleRestore = (version: SpecVersion) => {
    if (!specId) return;
    request("specs:update", {
      id: specId,
      updates: { content: version.content, title: version.title },
    }).then(() => {
      // Refresh spec list
      request<Array<{ id: string; title: string; content: string }>>("specs:list").then((r) => {
        if (r.ok && Array.isArray(r.data)) setSpecList(r.data);
      });
    });
  };

  const handleDeleteVersion = (versionId: string) => {
    if (!specId) return;
    deleteVersion(specId, versionId);
    setVersions(getVersions(specId));
    if (selectedVersionId === versionId) {
      setSelectedVersionId(null);
      setDiffText(null);
    }
  };

  const selectedVersion = versions.find((v) => v.id === selectedVersionId);
  const currentSpec = specList.find((s) => s.id === specId);
  const currentWordCount = currentSpec
    ? currentSpec.content.split(/\s+/).filter((w) => w.length > 0).length
    : 0;

  return (
    <div className="version-history">
      <div className="version-history__layout">
        {/* Left: Spec selector + Version list */}
        <div className="version-history__sidebar">
          <div className="version-history__sidebar-header">
            <span style={{ fontSize: 12, fontWeight: 600 }}>Version History</span>
          </div>

          {/* Spec selector */}
          <div className="version-history__spec-selector">
            <select
              value={specId}
              onChange={(e) => loadVersions(e.target.value)}
              className="version-history__spec-select"
            >
              {specList.map((s) => (
                <option key={s.id} value={s.id}>{s.title || "Untitled"}</option>
              ))}
            </select>
            <button className="btn btn--sm btn--primary" onClick={handleSnapshot} title="Snapshot current version">
              <HiOutlineClock /> Snapshot
            </button>
          </div>

          {/* Version list */}
          <div className="version-history__list">
            {versions.length === 0 ? (
              <div className="empty-state" style={{ padding: 20 }}>
                <HiOutlineClock style={{ fontSize: 24, opacity: 0.3 }} />
                <p style={{ fontSize: 12 }}>No versions yet. Click Snapshot to save one.</p>
              </div>
            ) : (
              versions.map((version) => {
                const isSelected = selectedVersionId === version.id;
                const date = new Date(version.timestamp);
                return (
                  <div
                    key={version.id}
                    className={`version-history__item ${isSelected ? "version-history__item--active" : ""}`}
                    onClick={() => handleViewDiff(version)}
                  >
                    <div className="version-history__item-header">
                      <span className="version-history__item-date">
                        {date.toLocaleDateString()} {date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
                      </span>
                    </div>
                    <div className="version-history__item-meta">
                      {version.wordCount} words
                    </div>
                    <div className="version-history__item-actions">
                      <button
                        className="btn btn--sm"
                        onClick={(e) => { e.stopPropagation(); handleRestore(version); }}
                        title="Restore this version"
                      >
                        <HiOutlineRefresh /> Restore
                      </button>
                      <button
                        className="btn btn--sm btn--danger"
                        onClick={(e) => { e.stopPropagation(); handleDeleteVersion(version.id); }}
                        title="Delete version"
                      >
                        Delete
                      </button>
                    </div>
                  </div>
                );
              })
            )}
          </div>
        </div>

        {/* Right: Diff view */}
        <div className="version-history__diff">
          {diffText && selectedVersion ? (
            <>
              <div className="version-history__diff-header">
                <HiOutlineDocumentText />
                <span>
                  Diff: {new Date(selectedVersion.timestamp).toLocaleDateString()} snapshot vs. current
                </span>
              </div>
              <pre className="version-history__diff-content">
                {diffText.split("\n").map((line, i) => {
                  let className = "version-history__diff-line";
                  if (line.startsWith("+")) className += " version-history__diff-line--added";
                  else if (line.startsWith("-")) className += " version-history__diff-line--removed";
                  else if (line.startsWith("  ")) className += " version-history__diff-line--context";
                  return (
                    <div key={i} className={className}>
                      {line}
                    </div>
                  );
                })}
              </pre>
            </>
          ) : (
            <div className="empty-state">
              <HiOutlineDocumentText style={{ fontSize: 36, opacity: 0.3 }} />
              <p>Select a version to see the diff against current content</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
