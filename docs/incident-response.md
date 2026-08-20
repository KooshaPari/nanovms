# Incident Response Playbook - NanoVMs

This document outlines the procedures for responding to incidents affecting the NanoVMs platform, its microVM lifecycle, and underlying infrastructure.

## 1. Severity Levels

| Level | Name         | Description                                                                 | Example Impact                                      |
|-------|--------------|-----------------------------------------------------------------------------|-----------------------------------------------------|
| **P0** | Critical     | Total infrastructure failure, hypervisor compromise, or data breach.        | MicroVMs failing to launch globally, host node crash.|
| **P1** | High         | Significant degradation of microVM performance or network isolation failure.| VM-to-VM leakage, high latency in orchestration.    |
| **P2** | Medium       | Intermittent failures in specific regions or non-critical API issues.       | Snapshot corruption in one zone, CLI logs error.    |
| **P3** | Low          | Minor resource leaks or non-urgent platform enhancements.                  | Metric reporting delays, documentation gaps.        |

## 2. Response Times

| Severity | Acknowledgment | First Response | Resolution Target |
|----------|----------------|----------------|-------------------|
| **P0**   | 5 minutes      | 15 minutes     | 2 hours           |
| **P1**   | 15 minutes     | 1 hour         | 8 hours           |
| **P2**   | 2 hours        | 4 hours        | 2 business days   |
| **P3**   | 4 hours        | 1 business day | Next maintenance  |

## 3. Escalation Matrix

| Level | Role                                    | Contact Method       |
|-------|-----------------------------------------|----------------------|
| 1     | Infrastructure On-call                  | PagerDuty / Slack    |
| 2     | Platform Lead / Security Officer        | Phone / Slack DM     |
| 3     | VP of Infrastructure / CTO              | Emergency Bridge     |

## 4. Communication Templates

### Internal Notification (Slack/Teams)
```markdown
:rotating_light: **[P{X}] Infrastructure Incident: [Brief Title]**
*Status:* Investigating
*Impact:* [Description of VM/Host impact]
*Lead:* [Name]
*Channel:* #nano-incidents-[ticket-id]
```

### External Status Page Update
```markdown
**Degraded Performance** - We are investigating reports of latency in the [Region] microVM cluster. 
Operations may be slower than expected. We are working to restore full performance.
```

## 5. Post-Mortem Template

### Incident Summary
- **Date/Time of Incident:** YYYY-MM-DD HH:MM UTC
- **Duration:** X hours Y minutes
- **Severity:** P0 / P1 / P2 / P3
- **Authors:** [List of authors]

### Impact
- **Infrastructure Impact:** [Host nodes / VM count affected]
- **User Impact:** [Services running on NanoVMs affected]

### Timeline (UTC)
- **HH:MM:** [Event]
- **HH:MM:** [Event]
- ...

### Root Cause Analysis
[Detailed explanation of the failure at the kernel/hypervisor/orchestration level.]

### What went well?
- [Item 1]
- [Item 2]

### What didn't go well?
- [Item 1]
- [Item 2]

### Action Items
| Action Item | Owner | Priority | Status |
|-------------|-------|----------|--------|
| [Task 1]    | Name  | High     | Open   |
| [Task 2]    | Name  | Medium   | Open   |

---

## 6. Root Cause Analysis (RCA) Template

### 1. What happened?
[High-level summary of the infrastructure failure.]

### 2. Why did it happen? (The "Why" Chain)
- **Problem:** [Symptom]
- **Cause 1:** [Direct cause] → Why? [Deeper reason]
- **Cause 2:** [Systemic cause] → Why? [Process failure]

### 3. Contributing Factors
- **Technical:** [e.g., Kernel bug, resource exhaustion, network partition]
- **Process:** [e.g., Inadequate load testing, missing runbooks]
- **People:** [e.g., Misconfiguration, lack of infra training]

### 4. Corrective Actions
- **Immediate:** [Actions taken to stabilize the environment]
- **Short-term:** [Infrastructure hardening, security patches]
- **Long-term:** [Architectural changes, chaos engineering adoption]

### 5. Lessons Learned
[Key takeaways for the infrastructure team.]
