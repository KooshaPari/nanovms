import type { View } from "../types";
import {
  HiOutlineViewBoards,
  HiOutlineDocumentText,
  HiOutlineChartBar,
  HiOutlineClock,
  HiOutlineShieldCheck,
  HiOutlineCheckCircle,
  HiOutlineAdjustments,
  HiOutlineViewGrid,
} from "react-icons/hi";

interface Props {
  view: View;
  onNavigate: (view: View) => void;
}

const NAV_ITEMS: { key: View; label: string; icon: React.ReactNode }[] = [
  { key: "dashboard", label: "Dashboard", icon: <HiOutlineViewBoards /> },
  { key: "specs", label: "Specs", icon: <HiOutlineDocumentText /> },
  { key: "scorecards", label: "Scorecards", icon: <HiOutlineChartBar /> },
  { key: "validation", label: "Validation", icon: <HiOutlineCheckCircle /> },
  { key: "compare", label: "Compare", icon: <HiOutlineAdjustments /> },
  { key: "backlog", label: "Backlog Board", icon: <HiOutlineViewGrid /> },
  { key: "sprints", label: "Sprints", icon: <HiOutlineClock /> },
  { key: "quality-gates", label: "Quality Gates", icon: <HiOutlineShieldCheck /> },
];

export default function Sidebar({ view, onNavigate }: Props) {
  return (
    <aside className="sidebar">
      <div className="sidebar__brand">
        <span className="sidebar__brand-icon">⚡</span> AgilePlus
      </div>
      <nav className="sidebar__nav">
        {NAV_ITEMS.map((item) => (
          <button
            key={item.key}
            className={`sidebar__item ${view === item.key ? "sidebar__item--active" : ""}`}
            onClick={() => onNavigate(item.key)}
          >
            {item.icon}
            {item.label}
          </button>
        ))}
      </nav>
      <div className="sidebar__footer">
        <span>v0.1.0</span>
        <span className="text-tertiary">nanovms</span>
      </div>
    </aside>
  );
}
