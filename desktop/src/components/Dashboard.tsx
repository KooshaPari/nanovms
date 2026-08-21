import { HiOutlineDocumentText, HiOutlinePlus } from "react-icons/hi";
import {
  getSpecs,
  getOverallScore,
  getActiveSprint,
  getBacklog,
  getPillars,
  createSpec,
} from "../store";

interface Props {
  onDataChange: () => void;
}

export default function Dashboard({ onDataChange }: Props) {
  const specs = getSpecs();
  const pillars = getPillars();
  const overall = getOverallScore();
  const activeSprint = getActiveSprint();
  const backlog = getBacklog();
  const pendingItems = backlog.filter((b) => b.status !== "done").length;
  const p0Count = backlog.filter((b) => b.priority === "P0" && b.status !== "done").length;

  const scoreColor = (score: number) => {
    if (score >= 9) return "dashboard__card-value--gold";
    if (score >= 7) return "dashboard__card-value--green";
    if (score >= 5) return "dashboard__card-value--yellow";
    return "dashboard__card-value--red";
  };

  const handleNewSpec = () => {
    createSpec("Untitled Spec");
    onDataChange();
  };

  const recentSpecs = [...specs]
    .sort((a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime())
    .slice(0, 5);

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
            {overall > 0 ? overall.toFixed(1) : "—"}
          </div>
          <div className="dashboard__card-sub">
            {pillars.length} pillars tracked
          </div>
        </div>

        <div className="dashboard__card">
          <div className="dashboard__card-label">Active Sprint</div>
          <div className="dashboard__card-value dashboard__card-value--accent">
            {activeSprint ? activeSprint.name : "—"}
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
