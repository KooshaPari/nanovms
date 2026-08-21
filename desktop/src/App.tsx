import { useState, useEffect, useCallback } from "react";
import {
  HiOutlineViewBoards,
  HiOutlineDocumentText,
  HiOutlineChartBar,
  HiOutlineClock,
  HiOutlineCollection,
  HiOutlineDownload,
  HiOutlineUpload,
} from "react-icons/hi";
import type { View } from "./types";
import { initPillarsIfEmpty, exportAll, importAll } from "./store";
import Dashboard from "./components/Dashboard";
import SpecEditor from "./components/SpecEditor";
import ScorecardView from "./components/ScorecardView";
import SprintTracker from "./components/SprintTracker";
import BacklogView from "./components/BacklogView";

const NAV_ITEMS: { key: View; label: string; icon: React.ReactNode }[] = [
  { key: "dashboard", label: "Dashboard", icon: <HiOutlineViewBoards /> },
  { key: "specs", label: "Specs", icon: <HiOutlineDocumentText /> },
  { key: "scorecards", label: "Scorecards", icon: <HiOutlineChartBar /> },
  { key: "sprints", label: "Sprints", icon: <HiOutlineClock /> },
  { key: "backlog", label: "Backlog", icon: <HiOutlineCollection /> },
];

export default function App() {
  const [view, setView] = useState<View>("dashboard");
  const [tick, setTick] = useState(0);

  // Initialize pillars on first load
  useEffect(() => {
    initPillarsIfEmpty();
  }, []);

  // Force re-render when child components change data
  const onDataChange = useCallback(() => {
    setTick((t) => t + 1);
  }, []);

  const handleExport = () => {
    const json = exportAll();
    const blob = new Blob([json], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `agileplus-export-${new Date().toISOString().slice(0, 10)}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const handleImport = () => {
    const input = document.createElement("input");
    input.type = "file";
    input.accept = ".json";
    input.onchange = async () => {
      const file = input.files?.[0];
      if (!file) return;
      const text = await file.text();
      if (importAll(text)) {
        onDataChange();
      }
    };
    input.click();
  };

  const renderView = () => {
    switch (view) {
      case "dashboard":
        return <Dashboard onDataChange={onDataChange} key={`dashboard-${tick}`} />;
      case "specs":
        return <SpecEditor onDataChange={onDataChange} key={`specs-${tick}`} />;
      case "scorecards":
        return <ScorecardView onDataChange={onDataChange} key={`scorecards-${tick}`} />;
      case "sprints":
        return <SprintTracker onDataChange={onDataChange} key={`sprints-${tick}`} />;
      case "backlog":
        return <BacklogView onDataChange={onDataChange} key={`backlog-${tick}`} />;
    }
  };

  return (
    <div className="app">
      <aside className="sidebar">
        <div className="sidebar__brand">
          <span>⚡</span> AgilePlus
        </div>
        <nav className="sidebar__nav">
          {NAV_ITEMS.map((item) => (
            <button
              key={item.key}
              className={`sidebar__item ${view === item.key ? "sidebar__item--active" : ""}`}
              onClick={() => setView(item.key)}
            >
              {item.icon}
              {item.label}
            </button>
          ))}
        </nav>
        <div className="sidebar__footer">
          <span>v0.1.0</span>
          <div className="flex gap-8">
            <button className="btn btn--sm" onClick={handleExport} title="Export data">
              <HiOutlineDownload />
            </button>
            <button className="btn btn--sm" onClick={handleImport} title="Import data">
              <HiOutlineUpload />
            </button>
          </div>
        </div>
      </aside>

      <main className="main">
        <header className="header">
          <div className="header__title">
            {NAV_ITEMS.find((n) => n.key === view)?.label}
          </div>
        </header>
        <div className="content" key={tick}>
          {renderView()}
        </div>
      </main>
    </div>
  );
}
