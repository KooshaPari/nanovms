import { useState, useEffect } from "react";
import { HiOutlinePlus, HiOutlineCheck } from "react-icons/hi";
import type { Sprint, BacklogItem, IpcResponse } from "../types";

interface Props {
  onDataChange: () => void;
  request: <T = unknown>(channel: string, data?: unknown) => Promise<IpcResponse<T>>;
}

export default function SprintTracker({ onDataChange, request }: Props) {
  const [sprints, setSprints] = useState<Sprint[]>([]);
  const [backlog, setBacklog] = useState<BacklogItem[]>([]);
  const [showForm, setShowForm] = useState(false);
  const [newName, setNewName] = useState("");
  const [newStart, setNewStart] = useState("");
  const [newEnd, setNewEnd] = useState("");
  const [newGoal, setNewGoal] = useState("");

  const loadAll = () => {
    request<Sprint[]>("sprints:list").then((r) => {
      if (r.ok && Array.isArray(r.data)) setSprints(r.data);
    });
    request<BacklogItem[]>("backlog:list").then((r) => {
      if (r.ok && Array.isArray(r.data)) setBacklog(r.data);
    });
  };

  useEffect(() => {
    loadAll();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const activeSprint = sprints.find((s) => s.status === "active") ?? null;
  const completedSprints = sprints.filter((s) => s.status === "completed");
  const doneCount = backlog.filter((b) => b.status === "done").length;
  const totalActive = backlog.filter(
    (b) => b.status === "in-progress" || b.status === "review" || b.status === "done",
  ).length;

  const sprintProgress =
    activeSprint && backlog.length > 0
      ? Math.round((doneCount / backlog.length) * 100)
      : 0;

  const handleCreate = () => {
    if (!newName.trim() || !newStart || !newEnd) return;
    request("sprints:create", {
      name: newName.trim(),
      startDate: newStart,
      endDate: newEnd,
      goal: newGoal.trim(),
    }).then(() => {
      setNewName("");
      setNewStart("");
      setNewEnd("");
      setNewGoal("");
      setShowForm(false);
      loadAll();
      onDataChange();
    });
  };

  const handleActivate = (id: string) => {
    if (activeSprint && activeSprint.id !== id) {
      request("sprints:update", { id: activeSprint.id, updates: { status: "completed" } });
    }
    request("sprints:update", { id, updates: { status: "active" } }).then(() => {
      loadAll();
      onDataChange();
    });
  };

  const handleComplete = (id: string) => {
    request("sprints:update", { id, updates: { status: "completed" } }).then(() => {
      loadAll();
      onDataChange();
    });
  };

  const handleVelocityChange = (id: string, velocity: number) => {
    request("sprints:update", { id, updates: { velocity } }).then(() => {
      loadAll();
    });
  };

  const velocityData = completedSprints.map((s) => ({
    name: s.name.length > 8 ? s.name.slice(0, 8) + "\u2026" : s.name,
    velocity: s.velocity,
  }));
  const maxVelocity = Math.max(...velocityData.map((v) => v.velocity), 1);

  return (
    <div className="sprint">
      {activeSprint ? (
        <div className="sprint__current">
          <div className="sprint__current-header">
            <div>
              <div className="sprint__current-name">{activeSprint.name}</div>
              <div className="sprint__current-dates">
                {activeSprint.startDate} \u2192 {activeSprint.endDate}
              </div>
            </div>
            <button className="btn btn--sm" onClick={() => handleComplete(activeSprint.id)}>
              <HiOutlineCheck /> Complete
            </button>
          </div>

          {activeSprint.goal && (
            <div className="sprint__goal">
              <strong>Goal:</strong> {activeSprint.goal}
            </div>
          )}

          <div className="sprint__progress">
            <div className="sprint__progress-bar">
              <div className="sprint__progress-fill" style={{ width: `${sprintProgress}%` }} />
            </div>
            <div className="sprint__progress-label">
              <span>{sprintProgress}% complete</span>
              <span>{doneCount}/{backlog.length} items done</span>
            </div>
          </div>

          <div className="sprint__velocity">
            <div className="sprint__velocity-stat">
              <div className="sprint__velocity-value">{activeSprint.velocity}</div>
              <div className="sprint__velocity-label">Velocity</div>
            </div>
            <div className="sprint__velocity-stat">
              <div className="sprint__velocity-value">{totalActive}</div>
              <div className="sprint__velocity-label">Items Active</div>
            </div>
            <div className="sprint__velocity-stat">
              <div className="sprint__velocity-value">{sprintProgress}%</div>
              <div className="sprint__velocity-label">Progress</div>
            </div>
          </div>
        </div>
      ) : (
        <div className="empty-state mb-12" style={{ padding: 30 }}>
          <p>No active sprint. Create one or activate a planned sprint.</p>
        </div>
      )}

      {velocityData.length > 0 && (
        <div className="sprint__velocity-chart">
          <h3 style={{ fontSize: 14, fontWeight: 600, color: "var(--text-secondary)", marginBottom: 8 }}>
            Velocity History
          </h3>
          <div className="sprint__velocity-bars">
            {velocityData.map((v, i) => (
              <div key={i} className="sprint__velocity-bar">
                <span style={{ fontSize: 11, color: "var(--text-primary)", fontWeight: 600 }}>
                  {v.velocity}
                </span>
                <div className="sprint__velocity-bar-fill" style={{ height: `${(v.velocity / maxVelocity) * 100}%` }} />
                <span className="sprint__velocity-bar-label">{v.name}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="sprint__list">
        <div className="sprint__list-header">
          <h3>All Sprints</h3>
          <button className="btn btn--sm btn--primary" onClick={() => setShowForm(!showForm)}>
            <HiOutlinePlus /> New Sprint
          </button>
        </div>

        {showForm && (
          <div className="inline-form" style={{ marginBottom: 12 }}>
            <input value={newName} onChange={(e) => setNewName(e.target.value)} placeholder="Sprint name" style={{ flex: 1, minWidth: 150 }} />
            <input type="date" value={newStart} onChange={(e) => setNewStart(e.target.value)} />
            <input type="date" value={newEnd} onChange={(e) => setNewEnd(e.target.value)} />
            <input value={newGoal} onChange={(e) => setNewGoal(e.target.value)} placeholder="Sprint goal (optional)" style={{ flex: 1, minWidth: 200 }} />
            <button className="btn btn--sm btn--primary" onClick={handleCreate}>Create</button>
          </div>
        )}

        {sprints.length === 0 ? (
          <div className="empty-state">
            <p>No sprints yet. Create your first sprint to get started.</p>
          </div>
        ) : (
          sprints.map((sprint) => (
            <div key={sprint.id} className="sprint__item">
              <div className="sprint__item-info">
                <div className="sprint__item-name">{sprint.name}</div>
                <div className="sprint__item-dates">
                  {sprint.startDate} \u2192 {sprint.endDate}
                  {sprint.goal && ` \u2014 ${sprint.goal}`}
                </div>
                <div style={{ display: "flex", alignItems: "center", gap: 8, marginTop: 4 }}>
                  <span style={{ fontSize: 11, color: "var(--text-tertiary)" }}>Velocity:</span>
                  <input
                    type="number"
                    min={0}
                    value={sprint.velocity}
                    onChange={(e) => handleVelocityChange(sprint.id, parseInt(e.target.value) || 0)}
                    style={{ width: 60, padding: "2px 6px", fontSize: 12, textAlign: "center" }}
                  />
                </div>
              </div>
              <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
                <span className={`sprint__item-status sprint__item-status--${sprint.status}`}>
                  {sprint.status}
                </span>
                {sprint.status !== "active" && sprint.status !== "completed" && (
                  <button className="btn btn--sm" onClick={() => handleActivate(sprint.id)}>
                    Activate
                  </button>
                )}
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
