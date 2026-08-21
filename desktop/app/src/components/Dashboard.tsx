import { useState, useEffect } from "react";
import { HiOutlineDocumentText, HiOutlinePlus } from "react-icons/hi";
import type { Spec, PillarScore, IpcResponse } from "../types";

interface Props {
  onDataChange: () => void;
  request: <T = unknown>(channel: string, data?: unknown) => Promise<IpcResponse<T>>;
}

export default function Dashboard({ onDataChange, request }: Props) {
  const [specs, setSpecs] = useState<Spec[]>([]);
  const [scorecard, setScorecard] = useState<{ average: number; grade: string; count: number; pillars: PillarScore[] } | null>(null);
  const [activeSprint, setActiveSprint] = useState<{ name: string; velocity: number; goal: string } | null>(null);
  const [backlog, setBacklog] = useState<Array<{ status: string; priority: string }>>([]);

  useEffect(() => {
    request<Spec[]>("specs:list").then((r) => {
      if (r.ok && Array.isArray(r.data)) setSpecs(r.data);
    });
    request("pillars:scorecard").then((r) => {
      if (r.ok && r.data) {
        const d = r.data as { summary: { average: number; grade: string; count: number }; pillars: PillarScore[] };
        setScorecard({ average: d.summary.average, grade: d.summary.grade, count: d.summary.count, pillars: d.pillars });
      }
    });
    request<{ status: string; name: string; velocity: number; goal: string }[]>("sprints:list").then((r) => {
      if (r.ok && Array.isArray(r.data)) {
        const active = r.data.find((s) => s.status === "active");
        if (active) setActiveSprint(active);
      }
    });
    request<Array<{ status: string; priority: string }>>("backlog:list").then((r) => {
      if (r.ok && Array.isArray(r.data)) setBacklog(r.data);
    });
  }, [request]); // eslint-disable-line react-hooks/exhaustive-deps

  const scoreColor = (score: number) => {
    if (score >= 9) return "dashboard__card-value--gold";
    if (score >= 7) return "dashboard__card-value--green";
    if (score >= 5) return "dashboard__card-value--yellow";
    return "dashboard__card-value--red";
  };

  const pendingItems = backlog.filter((b) => b.status !== "done").length;
  const p0Count = backlog.filter((b) => b.priority === "P0" && b.status !== "done").length;

  const recentSpecs = [...specs]
    .sort((a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime())
    .slice(0, 5);

  const overall = scorecard?.average ?? 0;

  const handleNewSpec = () => {
    request("specs:create", { title: "Untitled Spec" }).then(() => {
      onDataChange();
    });
  };

  return (
    <div>
      <div className="dashboard">
        <div className="dashboard__card">
          <div className="dashboard__card-label">Total Specs</div>
          <div className="dashboard__card-value dashboard__card-value--accent">
            {specs.length}
          </div>
          <div className="dashboard__card-sub">
            {specs.length === 0 ? "Create your first spec" : `${recentSpecs.length} recent`}
          </div>
        </div>

        <div className="dashboard__card">
          <div className="dashboard__card-label">Overall Pillar Score</div>
          <div className={`dashboard__card-value ${scoreColor(overall)}`}>
            {overall > 0 ? overall.toFixed(1) : "\u2014"}
          </div>
          <div className="dashboard__card-sub">
            {scorecard?.count ?? 31} pillars tracked
          </div>
        </div>

        <div className="dashboard__card">
          <div className="dashboard__card-label">Active Sprint</div>
          <div className="dashboard__card-value dashboard__card-value--accent">
            {activeSprint ? activeSprint.name : "\u2014"}
          </div>
          <div className="dashboard__card-sub">
            {activeSprint
              ? `Velocity: ${activeSprint.velocity}`
              : "No active sprint"}
          </div>
        </div>

        <div className="dashboard__card">
          <div className="dashboard__card-label">Pending Backlog</div>
          <div className="dashboard__card-value dashboard__card-value--yellow">
            {pendingItems}
          </div>
          <div className="dashboard__card-sub">
            {p0Count > 0 ? (
              <span style={{ color: "var(--red)" }}>{p0Count} P0 items</span>
            ) : (
              "No P0 items"
            )}
          </div>
        </div>
      </div>

      <div className="dashboard__recent">
        <h3>Recent Specs</h3>
        {recentSpecs.length === 0 ? (
          <div className="empty-state">
            <HiOutlineDocumentText />
            <p>No specs yet. Create one from the Specs view.</p>
            <button className="btn btn--primary mt-12" onClick={handleNewSpec}>
              <HiOutlinePlus /> New Spec
            </button>
          </div>
        ) : (
          <div className="dashboard__recent-list">
            {recentSpecs.map((spec) => (
              <div key={spec.id} className="dashboard__recent-item">
                <span>{spec.title}</span>
                <span>{new Date(spec.updatedAt).toLocaleDateString()}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
