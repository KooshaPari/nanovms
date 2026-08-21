import { useState, useCallback, useEffect, useRef } from "react";
import { marked } from "marked";
import { HiOutlinePlus, HiOutlineTrash } from "react-icons/hi";
import { getPillars } from "../store";
import type { Spec, Pillar } from "../types";
import {
  getSpecs,
  getSpec,
  createSpec,
  updateSpec,
  deleteSpec,
} from "../store";

interface Props {
  onDataChange: () => void;
}

marked.setOptions({
  gfm: true,
  breaks: true,
});

export default function SpecEditor({ onDataChange }: Props) {
  const [specs, setSpecs] = useState<Spec[]>(() => getSpecs());
  const [selectedId, setSelectedId] = useState<string | null>(() => {
    return specs.length > 0 ? specs[0].id : null;
  });
  const [content, setContent] = useState("");
  const [title, setTitle] = useState("");
  const [saveStatus, setSaveStatus] = useState<"saved" | "unsaved" | "saving">("saved");
  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pillars = getPillars();

  const selectedSpec = selectedId ? getSpec(selectedId) : null;

  useEffect(() => {
    if (selectedSpec) {
      setContent(selectedSpec.content);
      setTitle(selectedSpec.title);
      setSaveStatus("saved");
    }
  }, [selectedSpec?.id]); // eslint-disable-line react-hooks/exhaustive-deps

  const autoSave = useCallback(
    (newContent: string, newTitle: string) => {
      if (!selectedId) return;
      setSaveStatus("unsaved");
      if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
      saveTimerRef.current = setTimeout(() => {
        updateSpec(selectedId, { content: newContent, title: newTitle });
        setSpecs(getSpecs());
        setSaveStatus("saved");
        onDataChange();
      }, 500);
    },
    [selectedId, onDataChange],
  );

  const handleNewSpec = () => {
    const spec = createSpec("Untitled Spec");
    setSpecs(getSpecs());
    setSelectedId(spec.id);
    setContent("");
    setTitle("Untitled Spec");
    onDataChange();
  };

  const handleDeleteSpec = (id: string) => {
    deleteSpec(id);
    const updated = getSpecs();
    setSpecs(updated);
    if (selectedId === id) {
      setSelectedId(updated.length > 0 ? updated[0].id : null);
    }
    onDataChange();
  };

  const handlePillarScoreChange = (pillarId: string, score: number) => {
    if (!selectedId) return;
    const current = getSpec(selectedId);
    if (!current) return;
    updateSpec(selectedId, {
      pillarScores: { ...current.pillarScores, [pillarId]: score },
    });
    setSpecs(getSpecs());
  };

  const handleManualSave = () => {
    if (!selectedId) return;
    if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
    updateSpec(selectedId, { content, title });
    setSpecs(getSpecs());
    setSaveStatus("saved");
    onDataChange();
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
              placeholder="—"
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
