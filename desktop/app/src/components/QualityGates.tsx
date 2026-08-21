import { useState, useEffect } from "react";
import type { QualityGate, QualityGateStatus, IpcResponse } from "../types";

interface Props {
  onDataChange: () => void;
  request: <T = unknown>(channel: string, data?: unknown) => Promise<IpcResponse<T>>;
}

interface GateWithStatus {
  gate: QualityGate;
  status: QualityGateStatus | null;
}

export default function QualityGates({ onDataChange, request }: Props) {
  const [gates, setGates] = useState<GateWithStatus[]>([]);

  useEffect(() => {
    Promise.all([
      request<QualityGate[]>("gates:list"),
      request<QualityGateStatus[]>("gates:status"),
    ]).then(([gatesRes, statusRes]) => {
      const gateList = (gatesRes.ok && Array.isArray(gatesRes.data) ? gatesRes.data : []) as QualityGate[];
      const statusList = (statusRes.ok && Array.isArray(statusRes.data) ? statusRes.data : []) as QualityGateStatus[];

      // If no gates loaded, use defaults from quality-gates.yml
      const effectiveGates = gateList.length > 0 ? gateList : DEFAULT_GATES;

      const combined: GateWithStatus[] = effectiveGates.map((g) => ({
        gate: g,
        status: statusList.find((s) => s.gate_id === g.id) ?? null,
      }));

      setGates(combined);
    });
  }, [request]); // eslint-disable-line react-hooks/exhaustive-deps

  const passCount = gates.filter((g) => g.status?.status === "pass").length;
  const failCount = gates.filter((g) => g.status?.status === "fail").length;
  const pendingCount = gates.filter((g) => !g.status || g.status.status === "pending").length;
  const requiredGates = gates.filter((g) => g.gate.required);
  const requiredPass = requiredGates.filter((g) => g.status?.status === "pass").length;

  return (
    <div className="gates">
      <div className="gates__header">
        <div>
          <h2 style={{ fontSize: 18, fontWeight: 700 }}>Quality Gates</h2>
          <p className="text-secondary" style={{ fontSize: 13, marginTop: 2 }}>
            CI gates required for merge to main
          </p>
        </div>
      </div>

      <div className="gates__summary">
        <div className="gates__summary-card">
          <div className="gates__summary-value" style={{ color: "var(--green)" }}>{passCount}</div>
          <div className="gates__summary-label">Passing</div>
        </div>
        <div className="gates__summary-card">
          <div className="gates__summary-value" style={{ color: "var(--red)" }}>{failCount}</div>
          <div className="gates__summary-label">Failing</div>
        </div>
        <div className="gates__summary-card">
          <div className="gates__summary-value" style={{ color: "var(--yellow)" }}>{pendingCount}</div>
          <div className="gates__summary-label">Pending</div>
        </div>
        <div className="gates__summary-card">
          <div className="gates__summary-value" style={{ color: "var(--accent)" }}>
            {requiredPass}/{requiredGates.length}
          </div>
          <div className="gates__summary-label">Required Gates</div>
        </div>
      </div>

      <div className="gates__list">
        {gates.map(({ gate, status }) => {
          const st = status?.status ?? "pending";
          return (
            <div key={gate.id} className="gates__item">
              <div className={`gates__item-indicator gates__item-indicator--${st}`} />
              <div className="gates__item-body">
                <div className="gates__item-name">
                  {gate.name}
                  <span className={`gates__item-badge gates__item-badge--${gate.required ? "required" : "optional"}`}>
                    {gate.required ? "Required" : "Optional"}
                  </span>
                </div>
                <div className="gates__item-desc">{gate.description}</div>
                <div className="gates__item-meta">
                  <span>Pillar: {gate.pillar}</span>
                  {gate.floor != null && <span>Floor: {gate.floor}%</span>}
                  {status?.lastRun && <span>Last run: {new Date(status.lastRun).toLocaleDateString()}</span>}
                  {status?.duration_ms != null && <span>Duration: {(status.duration_ms / 1000).toFixed(1)}s</span>}
                </div>
              </div>
              <span className={`gates__item-status gates__item-status--${st}`}>
                {st}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}

// Default gates matching agileplus/quality-gates.yml
const DEFAULT_GATES: QualityGate[] = [
  {
    id: "lint",
    name: "Lint",
    description: "Static analysis for Go, Rust, YAML, Markdown, and shell.",
    pillar: "code-quality",
    required: true,
    command: "golangci-lint run ./... && cargo clippy --all-targets --all-features -- -D warnings",
  },
  {
    id: "test",
    name: "Test",
    description: "Unit + integration tests for Go and Rust workspaces.",
    pillar: "tests",
    required: true,
    command: "go test ./... -race -count=1 && cargo test --workspace --all-features",
  },
  {
    id: "security",
    name: "Security",
    description: "Supply-chain and secret scanning.",
    pillar: "security",
    required: true,
    command: "cargo deny check && trivy fs --severity HIGH,CRITICAL --no-progress . && gitleaks detect",
  },
  {
    id: "build",
    name: "Build",
    description: "Release-mode build for both Go and Rust binaries.",
    pillar: "build",
    required: true,
    command: "cargo build --release --workspace && go build -o bin/nanovms ./cmd/nanovms",
  },
  {
    id: "docs-build",
    name: "Docs Build",
    description: "VitePress documentation site builds cleanly.",
    pillar: "docs",
    required: false,
    command: "cd docs && npm ci && npm run build",
  },
  {
    id: "coverage",
    name: "Coverage",
    description: "Test coverage above floor; tracked, not blocking.",
    pillar: "tests",
    required: false,
    floor: 60,
    command: "cargo llvm-cov --workspace --lcov --output-path coverage.lcov",
  },
];
