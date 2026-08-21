import { useState, useCallback, useEffect, useRef } from "react";
import { marked } from "marked";
import { HiOutlinePlus, HiOutlineTrash } from "react-icons/hi";
import type { Spec, Pillar, IpcResponse } from "../types";

interface Props {
  onDataChange: () => void;
  request: <T = unknown>(channel: string, data?: unknown) => Promise<IpcResponse<T>>;
}

marked.setOptions({
  gfm: true,
  breaks: true,
});

export default function SpecEditor({ onDataChange, request }: Props) {
  const [specs, setSpecs] = useState<Spec[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [content, setContent] = useState("");
  const [title, setTitle] = useState("");
  const [saveStatus, setSaveStatus] = useState<"saved" | "unsaved" | "saving">("saved");
  const [pillars, setPillars] = useState<Pillar[]>([]);
  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Load specs on mount
  useEffect(() => {
    request<Spec[]>("specs:list").then((r) => {
      if (r.ok && Array.isArray(r.data)) {
        setSpecs(r.data);
        if (r.data.length > 0 && !selectedId) {
          setSelectedId(r.data[0].id);
        }
      }
    });
    request<Pillar[]>("pillars:list").then((r) => {
      if (r.ok && Array.isArray(r.data)) setPillars(r.data);
    });
  }, [request]); // eslint-disable-line react-hooks/exhaustive-deps

  // Load selected spec
  useEffect(() => {
    if (!selectedId) return;
    request<Spec>("specs:get", { id: selectedId }).then((r) => {
      if (r.ok && r.data) {
        setContent((r.data as Spec).content);
        setTitle((r.data as Spec).title);
        setSaveStatus("saved");
      }
    });
  }, [selectedId, request]);

  const loadSpecs = useCallback(() => {
    request<Spec[]>("specs:list").then((r) => {
      if (r.ok && Array.isArray(r.data)) setSpecs(r.data);
    });
  }, [request]);

  const autoSave = useCallback(
    (newContent: string, newTitle: string) => {
      if (!selectedId) return;
      setSaveStatus("unsaved");
      if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
      saveTimerRef.current = setTimeout(() => {
        request("specs:update", {
          id: selectedId,
          updates: { content: newContent, title: newTitle },
        }).then(() => {
          loadSpecs();
          setSaveStatus("saved");
          onDataChange();
        });
      }, 500);
    },
    [selectedId, request, loadSpecs, onDataChange],
  );

  const handleNewSpec = () => {
    request<Spec>("specs:create", { title: "Untitled Spec" }).then((r) => {
      if (r.ok && r.data) {
        loadSpecs();
        setSelectedId((r.data as Spec).id);
        setContent("");
        setTitle("Untitled Spec");
        onDataChange();
      }
    });
  };

  const handleDeleteSpec = (id: string) => {
    request("specs:delete", { id }).then(() => {
      loadSpecs();
      if (selectedId === id) {
        setSelectedId(null);
      }
      onDataChange();
    });
  };

  const handlePillarScoreChange = (pillarId: string, score: number) => {
    if (!selectedId) return;
    request<Spec>("specs:get", { id: selectedId }).then((r) => {
      if (r.ok && r.data) {
        const spec = r.data as Spec;
        const updatedScores = { ...spec.pillarScores, [pillarId]: score };
        request("specs:update", {
          id: selectedId,
          updates: { pillarScores: updatedScores },
        }).then(() => loadSpecs());
      }
    });
  };

  const handleManualSave = () => {
    if (!selectedId) return;
    if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
    request("specs:update", {
      id: selectedId,
      updates: { content, title },
    }).then(() => {
      loadSpecs();
      setSaveStatus("saved");
      onDataChange();
    });
  };

  const wordCount = content
    .split(/\s+/)
    .filter((w) => w.length > 0).length;

  const renderMarkdown = (md: string): string => {
    try {
      return marked.parse(md) as string;
    } catch {
      return "<p>Error rendering markdown</p>";
    }
  };

  const selectedSpec = specs.find((s) => s.id === selectedId);

  return (
    <div className="spec-editor-layout">
      <div className="spec-sidebar">
        <div className="spec-sidebar__header">
          <span className="text-tertiary" style={{ fontSize: 12 }}>
            {specs.length} specs
          </span>
          <button className="btn btn--sm btn--primary" onClick={handleNewSpec}>
            <HiOutlinePlus /> New
          </button>
        </div>
        <div className="spec-sidebar__list">
          {specs.map((spec) => (
            <div
              key={spec.id}
              className={`spec-sidebar__item ${selectedId === spec.id ? "spec-sidebar__item--active" : ""}`}
              onClick={() => setSelectedId(spec.id)}
            >
              <span style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                {spec.title || "Untitled"}
              </span>
              <button
                className="spec-sidebar__item-delete"
                onClick={(e) => {
                  e.stopPropagation();
                  handleDeleteSpec(spec.id);
                }}
                title="Delete spec"
              >
                <HiOutlineTrash />
              </button>
            </div>
          ))}
          {specs.length === 0 && (
            <div className="empty-state" style={{ padding: 20 }}>
              <p>No specs yet</p>
            </div>
          )}
        </div>
      </div>

      {selectedSpec ? (
        <div className="spec-workspace">
          <div className="spec-toolbar">
            <input
              className="spec-toolbar__title-input"
              value={title}
              onChange={(e) => {
                setTitle(e.target.value);
                autoSave(content, e.target.value);
              }}
              placeholder="Spec title..."
            />
            <span className="spec-toolbar__meta">
              {saveStatus === "saved"
                ? "Saved"
                : saveStatus === "saving"
                  ? "Saving..."
                  : "Unsaved changes"}
            </span>
            <span className="spec-toolbar__meta">{wordCount} words</span>
            <button className="btn btn--sm btn--primary" onClick={handleManualSave}>
              Save
            </button>
          </div>

          <div className="spec-split">
            <div className="spec-editor-pane">
              <div className="spec-editor-pane__label">
                <span>Markdown</span>
              </div>
              <textarea
                className="spec-textarea"
                value={content}
                onChange={(e) => {
                  setContent(e.target.value);
                  autoSave(e.target.value, title);
                }}
                placeholder="Write your spec in Markdown..."
                spellCheck={false}
              />
            </div>

            <div className="spec-preview-pane">
              <div className="spec-preview-pane__label">
                Preview
              </div>
              <div
                className="spec-preview"
                dangerouslySetInnerHTML={{ __html: renderMarkdown(content) }}
              />
            </div>
          </div>

          <PillarAssignment
            pillars={pillars}
            scores={selectedSpec.pillarScores}
            onChange={handlePillarScoreChange}
          />
        </div>
      ) : (
        <div className="spec-empty">
          <HiOutlineDocumentText style={{ fontSize: 48, opacity: 0.3 }} />
          <p style={{ color: "var(--text-tertiary)", fontSize: 14 }}>
            Select a spec or create a new one
          </p>
          <button className="btn btn--primary mt-12" onClick={handleNewSpec}>
            <HiOutlinePlus /> Create Spec
          </button>
        </div>
      )}
    </div>
  );
}

// ─── Pillar Assignment Sub-Component ──────────────────────────────────────────
interface PillarAssignmentProps {
  pillars: Pillar[];
  scores: Record<string, number>;
  onChange: (pillarId: string, score: number) => void;
}

function PillarAssignment({ pillars, scores, onChange }: PillarAssignmentProps) {
  if (pillars.length === 0) return null;

  return (
    <div className="pillar-assign">
      <h4>Pillar Score Assignment</h4>
      <div className="pillar-assign__grid">
        {pillars.map((p) => (
          <div key={p.id} className="pillar-assign__item">
            <label title={`${p.category} — target: ${p.target}`}>{p.name}</label>
            <input
              type="number"
              min={0}
              max={10}
              step={0.5}
              value={scores[p.id] ?? ""}
              placeholder="\u2014"
              onChange={(e) => {
                const val = parseFloat(e.target.value);
                if (!isNaN(val)) onChange(p.id, val);
              }}
            />
          </div>
        ))}
      </div>
    </div>
  );
}
