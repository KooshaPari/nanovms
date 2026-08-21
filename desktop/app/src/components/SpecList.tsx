import { useState, useEffect } from "react";
import { HiOutlineDocumentText, HiOutlinePlus, HiOutlineTrash } from "react-icons/hi";
import type { Spec, IpcResponse } from "../types";

interface Props {
  onDataChange: () => void;
  request: <T = unknown>(channel: string, data?: unknown) => Promise<IpcResponse<T>>;
  onSelectSpec?: (spec: Spec) => void;
  selectedId?: string | null;
}

export default function SpecList({ onDataChange, request, onSelectSpec, selectedId }: Props) {
  const [specs, setSpecs] = useState<Spec[]>([]);

  const loadSpecs = () => {
    request<Spec[]>("specs:list").then((r) => {
      if (r.ok && Array.isArray(r.data)) setSpecs(r.data);
    });
  };

  useEffect(() => {
    loadSpecs();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const handleNewSpec = () => {
    request<Spec>("specs:create", { title: "Untitled Spec" }).then((r) => {
      if (r.ok && r.data) {
        loadSpecs();
        onDataChange();
        onSelectSpec?.(r.data as Spec);
      }
    });
  };

  const handleDeleteSpec = (id: string) => {
    request("specs:delete", { id }).then(() => {
      loadSpecs();
      onDataChange();
    });
  };

  const sorted = [...specs].sort(
    (a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime(),
  );

  return (
    <div className="spec-list">
      <div className="spec-list__header">
        <h3 style={{ fontSize: 16, fontWeight: 600 }}>Specs</h3>
        <button className="btn btn--sm btn--primary" onClick={handleNewSpec}>
          <HiOutlinePlus /> New Spec
        </button>
      </div>

      {sorted.length === 0 ? (
        <div className="empty-state">
          <HiOutlineDocumentText />
          <p>No specs yet. Create your first spec to get started.</p>
          <button className="btn btn--primary mt-12" onClick={handleNewSpec}>
            <HiOutlinePlus /> Create Spec
          </button>
        </div>
      ) : (
        <div className="spec-list__items">
          {sorted.map((spec) => (
            <div
              key={spec.id}
              className={`spec-list__item ${selectedId === spec.id ? "spec-sidebar__item--active" : ""}`}
              onClick={() => onSelectSpec?.(spec)}
            >
              <div>
                <div className="spec-list__item-title">
                  {spec.title || "Untitled"}
                </div>
                <div className="spec-list__item-meta">
                  {new Date(spec.updatedAt).toLocaleDateString()} &middot;{" "}
                  {spec.content.split(/\s+/).filter((w) => w.length > 0).length} words
                </div>
              </div>
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
        </div>
      )}
    </div>
  );
}
