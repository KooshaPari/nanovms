import { useState } from "react";
import { HiOutlinePlus, HiOutlineTrash } from "react-icons/hi";
import type { BacklogItem, Priority } from "../types";
import {
  getBacklog,
  createBacklogItem,
  updateBacklogItem,
  deleteBacklogItem,
  reorderBacklog,
} from "../store";

interface Props {
  onDataChange: () => void;
}

const PRIORITY_ORDER: Priority[] = ["P0", "P1", "P2", "P3"];
const STATUS_OPTIONS: BacklogItem["status"][] = ["todo", "in-progress", "review", "done"];

export default function BacklogView({ onDataChange }: Props) {
  const [items, setItems] = useState<BacklogItem[]>(() => getBacklog());
  const [showForm, setShowForm] = useState(false);
  const [newTitle, setNewTitle] = useState("");
  const [newPriority, setNewPriority] = useState<Priority>("P1");
  const [dragId, setDragId] = useState<string | null>(null);

  const refresh = () => {
    setItems(getBacklog());
    onDataChange();
  };

  const handleCreate = () => {
    if (!newTitle.trim()) return;
    createBacklogItem(newTitle.trim(), newPriority);
    setNewTitle("");
    setNewPriority("P1");
    setShowForm(false);
    refresh();
  };

  const handleStatusChange = (id: string) => {
    const item = items.find((b) => b.id === id);
    if (!item) return;
    const currentIdx = STATUS_OPTIONS.indexOf(item.status);
    const nextStatus = STATUS_OPTIONS[(currentIdx + 1) % STATUS_OPTIONS.length];
    updateBacklogItem(id, { status: nextStatus });
    refresh();
  };

  const handleDelete = (id: string) => {
    deleteBacklogItem(id);
    refresh();
  };

  // Drag handlers
  const handleDragStart = (e: React.DragEvent, id: string) => {
    setDragId(id);
    e.dataTransfer.effectAllowed = "move";
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = "move";
  };

  const handleDrop = (e: React.DragEvent, targetId: string) => {
    e.preventDefault();
    if (!dragId || dragId === targetId) return;

    const ids = items.map((b) => b.id);
    const fromIdx = ids.indexOf(dragId);
    const toIdx = ids.indexOf(targetId);
    if (fromIdx === -1 || toIdx === -1) return;

    ids.splice(fromIdx, 1);
    ids.splice(toIdx, 0, dragId);
    reorderBacklog(ids);
    setDragId(null);
    refresh();
  };

  const handleDragEnd = () => {
    setDragId(null);
  };

  const grouped = PRIORITY_ORDER.map((p) => ({
    priority: p,
    items: items.filter((b) => b.priority === p),
  }));

  return (
    <div className="backlog">
      <div className="backlog__header">
        <div>
          <h2 style={{ fontSize: 18, fontWeight: 700 }}>Backlog</h2>
          <p className="text-secondary" style={{ fontSize: 13, marginTop: 2 }}>
            {items.length} items —{" "}
            {items.filter((b) => b.status === "done").length} completed
          </p>
        </div>
        <button className="btn btn--sm btn--primary" onClick={() => setShowForm(!showForm)}>
          <HiOutlinePlus /> Add Item
        </button>
      </div>

      {showForm && (
        <div className="inline-form" style={{ marginBottom: 16 }}>
          <input
            value={newTitle}
            onChange={(e) => setNewTitle(e.target.value)}
            placeholder="Item title..."
            style={{ flex: 1 }}
            onKeyDown={(e) => e.key === "Enter" && handleCreate()}
            autoFocus
          />
          <select
            value={newPriority}
            onChange={(e) => setNewPriority(e.target.value as Priority)}
          >
            {PRIORITY_ORDER.map((p) => (
              <option key={p} value={p}>
                {p}
              </option>
            ))}
          </select>
          <button className="btn btn--sm btn--primary" onClick={handleCreate}>
            Add
          </button>
        </div>
      )}

      {items.length === 0 && !showForm ? (
        <div className="empty-state">
          <p>No backlog items yet. Add items to start planning.</p>
        </div>
      ) : (
        grouped.map((group) => (
          <div key={group.priority} className="backlog__priority-group">
            <div className="backlog__priority-label">
              <span className={`backlog__priority-badge backlog__priority-badge--${group.priority}`}>
                {group.priority}
              </span>
              <span style={{ fontWeight: 400, color: "var(--text-tertiary)" }}>
                {group.items.length} items
              </span>
            </div>
            <div className="backlog__items">
              {group.items.map((item) => (
                <div
                  key={item.id}
                  className={`backlog__item ${dragId === item.id ? "dragging" : ""}`}
                  draggable
                  onDragStart={(e) => handleDragStart(e, item.id)}
                  onDragOver={handleDragOver}
                  onDrop={(e) => handleDrop(e, item.id)}
                  onDragEnd={handleDragEnd}
                >
                  <span className="backlog__item-drag" title="Drag to reorder">⠿</span>
                  <span className="backlog__item-title">{item.title}</span>
                  <div className="backlog__item-actions">
                    <button
                      className={`backlog__item-status ${
                        item.status === "done" ? "backlog__item-status--done" : ""
                      }`}
                      onClick={() => handleStatusChange(item.id)}
                      title="Click to cycle status"
                    >
                      {item.status}
                    </button>
                    <button
                      className="backlog__item-delete"
                      onClick={() => handleDelete(item.id)}
                      title="Delete item"
                    >
                      <HiOutlineTrash />
                    </button>
                  </div>
                </div>
              ))}
              {group.items.length === 0 && (
                <div
                  style={{
                    padding: "8px 14px",
                    fontSize: 12,
                    color: "var(--text-tertiary)",
                  }}
                >
                  No {group.priority} items
                </div>
              )}
            </div>
          </div>
        ))
      )}
    </div>
  );
}
