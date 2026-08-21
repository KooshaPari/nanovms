import { useState, useCallback, useEffect } from "react";
import type { View } from "./types";
import { useAgilePlus } from "./hooks/useAgilePlus";
import Sidebar from "./components/Sidebar";
import Dashboard from "./components/Dashboard";
import SpecEditor from "./components/SpecEditor";
import PillarScorecard from "./components/PillarScorecard";
import SprintTracker from "./components/SprintTracker";
import QualityGates from "./components/QualityGates";

export default function App() {
  const [view, setView] = useState<View>("dashboard");
  const [tick, setTick] = useState(0);
  const { request } = useAgilePlus();

  // Force re-render when child components mutate data
  const onDataChange = useCallback(() => {
    setTick((t) => t + 1);
  }, []);

  // Initialize pillars on first load
  useEffect(() => {
    request("pillars:list"); // triggers seeding in handlers if empty
  }, [request]); // eslint-disable-line react-hooks/exhaustive-deps

  const renderView = () => {
    switch (view) {
      case "dashboard":
        return <Dashboard onDataChange={onDataChange} request={request} key={`dashboard-${tick}`} />;
      case "specs":
        return <SpecEditor onDataChange={onDataChange} request={request} key={`specs-${tick}`} />;
      case "scorecards":
        return <PillarScorecard onDataChange={onDataChange} request={request} key={`scorecards-${tick}`} />;
      case "sprints":
        return <SprintTracker onDataChange={onDataChange} request={request} key={`sprints-${tick}`} />;
      case "quality-gates":
        return <QualityGates onDataChange={onDataChange} request={request} key={`gates-${tick}`} />;
    }
  };

  return (
    <div className="app">
      <Sidebar view={view} onNavigate={setView} />
      <main className="main">
        <header className="header">
          <div className="header__title">{viewTitle(view)}</div>
        </header>
        <div className="content" key={tick}>
          {renderView()}
        </div>
      </main>
    </div>
  );
}

function viewTitle(view: View): string {
  const titles: Record<View, string> = {
    dashboard: "Dashboard",
    specs: "Specs",
    scorecards: "31-Pillar Scorecards",
    sprints: "Sprint Tracker",
    "quality-gates": "Quality Gates",
  };
  return titles[view];
}
