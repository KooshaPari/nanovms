import { useState, useEffect, useCallback } from "react";
import { HiOutlinePlus, HiOutlineTrash } from "react-icons/hi";
import type { BacklogBoardItem, Priority, IpcResponse } from "../types";

interface Props {
  onDataChange: () => void;
  request: <T = unknown>(channel: string, data?: unknown) => Promise<IpcResponse<T>>;
}

const PRIORITY_COLS: { key: Priority; label: string; color: string }[] = [
  { key: "P0", label: "P0 Critical", color: "var(--red)" },
  { key: "P1", label: "P1 High", color: "var(--yellow)" },
  { key: "P2", label: "P2 Medium", color: "var(--accent)" },
  { key: "P3", label: "P3 Low", color: "var(--text-tertiary)" },
];

export default function BacklogBoard({ onDataChange, request }: Props) {
  const [items, setItems] = useState<BacklogBoardItem[]>([]);
  const [showForm, setShowForm] = useState(false);
  const [newTitle, setNewTitle] = useState("");
  const [newPriority, setNewPriority] = useState<Priority>("P2");
  const [newPoints, setNewPoints] = useState<number>(3);
  const [newAssignee, setNewAssignee] = useState("");
  const [draggedId, setDraggedId] = useState<string | null>(null);
  const [dragOverCol, setDragOverCol] = useState<Priority | null>(null);

  useEffect(() => {
    request<BacklogBoardItem[]>("backlog:board:list").then((r) => {
      if (r.ok && Array.isArray(r.data)) setItems(r.data);
    });
  }, [request]); // eslint-disable-line react-hooks/exhaustive-deps

  const loadItems = useCallback(() => {
    request<BacklogBoardItem[]>("backlog:board:list").then((r) => {
      if (r.ok && Array.isArray(r.data)) setItems(r.data);
    });
  }, [request]);

  const handleAddItem = () => {
    if (!newTitle.trim()) return;
    request("backlog:board:create", {
      title: newTitle.trim(),
      priority: newPriority,
      points: newPoints,
      assignee: newAssignee.trim() || "Unassigned",
    }).then(() => {
      setNewTitle("");
      setNewPriority("P2");
      setNewPoints(3);
      setNewAssignee("");
      setShowForm(false);
      loadItems();
      onDataChange();
    });
  };

  const handleDeleteItem = (id: string) => {
    request("backlog:board:delete", { id }).then(() => {
      loadItems();
      onDataChange();
    });
  };

  const handlePriorityChange = (id: string, newPriority: Priority) => {
    request("backlog:board:update", { id, updates: { priority: newPriority } }).then(() => {
      loadItems();
      onDataChange();
    });
  };

  // Drag and drop handlers
  const handleDragStart = (e: React.DragEvent, id: string) => {
    setDraggedId(id);
    e.dataTransfer.effectAllowed = "move";
    e.dataTransfer.setData("text/plain", id);
  };

  const handleDragOver = (e: React.DragEvent, col: Priority) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = "move";
    setDragOverCol(col);
  };

  const handleDragLeave = () => {
    setDragOverCol(null);
  };

  const handleDrop = (e: React.DragEvent, targetPriority: Priority) => {
    e.preventDefault();
    const id = e.dataTransfer.getData("text/plain");
    if (!id) return;

    const item = items.find((i) => i.id === id);
    if (item && item.priority !== targetPriority) {
      handlePriorityChange(id, targetPriority);
    }
    setDraggedId(null);
    setDragOverCol(null);
  };

  const handleDragEnd = () => {
    setDraggedId(null);
    setDragOverCol(null);
  };

  const getColumnItems = (priority: Priority): BacklogBoardItem[] => {
    return items.filter((i) => i.priority === priority);
  };

  const totalPoints = (priority: Priority) =>
    getColumnItems(priority).reduce((sum, item) => sum + (item.points || 0), 0);

  return (
    <div className="backlog-board">
      <div className="backlog-board__header">
        <div>
          <h2 style={{ fontSize: 18, fontWeight: 700 }}>Backlog Board</h2>
          <p className="text-secondary" style={{ fontSize: 13, marginTop: 2 }}>
            Drag items between priority columns
          </p>
        </div>
        <button className="btn btn--primary" onClick={() => setShowForm(!showForm)}>
          <HiOutlinePlus /> Add Item
        </button>
      </div>

      {/* Add item form */}
      {showForm && (
        <div className="backlog-board__form">
          <input
            value={newTitle}
            onChange={(e) => setNewTitle(e.target.value)}
            placeholder="Item title..."
            className="backlog-board__form-input"
            onKeyDown={(e) => e.key === "Enter" && handleAddItem()}
          />
          <select
            value={newPriority}
            onChange={(e) => setNewPriority(e.target.value as Priority)}
            className="backlog-board__form-select"
          >
            {PRIORITY_COLS.map((col) => (
              <option key={col.key} value={col.key}>{col.label}</option>
            ))}
          </select>
          <div className="backlog-board__form-points">
            <label className="text-tertiary" style={{ fontSize: 11 }}>Points:</label>
            <input
              type="number"
              min={0}
              max={21}
              value={newPoints}
              onChange={(e) => setNewPoints(parseInt(e.target.value) || 0)}
              className="backlog-board__form-points-input"
            />
          </div>
          <input
            value={newAssignee}
            onChange={(e) => setNewAssignee(e.target.value)}
            placeholder="Assignee..."
            className="backlog-board__form-input"
            style={{ width: 140 }}
          />
          <button className="btn btn--sm btn--primary" onClick={handleAddItem}>
            Add
          </button>
          <button className="btn btn--sm" onClick={() => setShowForm(false)}>
            Cancel
          </button>
        </div>
      )}

      {/* Board columns */}
      <div className="backlog-board__columns">
        {PRIORITY_COLS.map((col) => {
          const colItems = getColumnItems(col.key);
          const isDragOver = dragOverCol === col.key;
          return (
            <div
              key={col.key}
              className={`backlog-board__column ${isDragOver ? "backlog-board__column--dragover" : ""}`}
              onDragOver={(e) => handleDragOver(e, col.key)}
              onDragLeave={handleDragLeave}
              onDrop={(e) => handleDrop(e, col.key)}
            >
              <div className="backlog-board__column-header" style={{ borderColor: col.color }}>
                <span className="backlog-board__column-title">{col.label}</span>
                <span className="backlog-board__column-count">
                  {colItems.length} ({totalPoints(col.key)} pts)
                </span>
              </div>
              <div className="backlog-board__column-items">
                {colItems.length === 0 && (
                  <div className="backlog-board__column-empty">
                    Drop items here
                  </div>
                )}
                {colItems.map((item) => (
                  <div
                    key={item.id}
                    className={`backlog-board__card ${draggedId === item.id ? "backlog-board__card--dragging" : ""}`}
                    draggable
                    onDragStart={(e) => handleDragStart(e, item.id)}
                    onDragEnd={handleDragEnd}
                  >
                    <div className="backlog-board__card-header">
                      <span className="backlog-board__card-title">{item.title}</span>
                      <button
                        className="backlog-board__card-delete"
                        onClick={() => handleDeleteItem(item.id)}
                        title="Delete"
                      >
                        <HiOutlineTrash />
                      </button>
                    </div>
                    <div className="backlog-board__card-meta">
                      <span className="backlog-board__card-points">{item.points} pts</span>
                      <span className="backlog-board__card-assignee">{item.assignee}</span>
                    </div>
                    {item.pillar && (
                      <div className="backlog-board__card-pillar">{item.pillar}</div>
                    )}
                  </div>
                ))}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
