# SLO Alerting Runbook

This runbook explains how to configure, operate, and respond to SLO burn rate alerts for the NanoVMS repository.

## Overview

The SLO alerting system monitors CI/CD pipeline health against a **99.5% success rate** SLO target. It consists of two workflows:

1. **SLO Burn Rate Monitor** (`slo-monitor.yml`) -- Runs daily at 06:00 UTC, calculates burn rate from the last 7 days of CI data
2. **SLO Alert Dispatcher** (`slo-alert.yml`) -- Triggered automatically when the monitor completes, dispatches alerts to configured channels

## SLO Configuration

| Parameter | Value | Description |
|-----------|-------|-------------|
| SLO Target | 99.5% | Required CI success rate |
| Error Budget | 0.5% | Allowable failure rate |
| Monthly Budget | 216 minutes | Maximum allowed downtime per month |
| PAGE Threshold | 14.4x | Burn rate that triggers PagerDuty |
| WARN Threshold | 3.0x | Burn rate that triggers Slack alert |
| INFO Threshold | 1.0x | Burn rate that creates informational issue |

## Severity Levels

### PAGE (Burn Rate >= 14.4x)

**Immediate action required.** The error budget will be exhausted within 2 days at the current burn rate.

**Alert channels triggered:**
- Slack webhook notification (with red alert formatting)
- PagerDuty incident creation (critical severity)
- GitHub issue creation/update

**Response procedure:**
1. Stop all non-critical merges immediately
2. Investigate the root cause of CI failures
3. Bisect to find the offending commit(s)
4. Roll back recent changes if root cause is unclear
5. Notify on-call engineering lead
6. Post status update in the incident Slack channel
7. Resume merges only after burn rate drops below 3.0x

### WARN (Burn Rate >= 3.0x)

**Investigation required within 4 hours.** Budget consumption is unsustainable.

**Alert channels triggered:**
- Slack webhook notification (with warning formatting)
- GitHub issue creation/update

**Response procedure:**
1. Review recent CI failures for patterns
2. Check for newly introduced flaky tests
3. Create a remediation plan within 4 hours
4. Consider pausing feature merges if trend continues
5. Monitor burn rate at next scheduled check

### INFO (Burn Rate >= 1.0x)

**Monitoring only.** Burn rate is elevated but within acceptable bounds.

**Alert channels triggered:**
- GitHub issue creation/update (informational)

**Response procedure:**
1. Review the trend over the next 24 hours
2. Check for any newly introduced test flakiness
3. No immediate action required unless trend worsens

### OK (Burn Rate < 1.0x)

**No action needed.** CI is performing within SLO bounds.

**Alert channels triggered:**
- None

## Required Secrets

Configure these secrets in the repository settings (Settings > Secrets and variables > Actions):

### `SLACK_WEBHOOK_URL`

The incoming webhook URL for your Slack channel.

**Setup steps:**
1. Go to https://api.slack.com/apps
2. Create a new app (or use existing) > "From scratch"
3. Select your workspace
4. Go to "Incoming Webhooks" > Enable
5. Click "Add New Webhook to Workspace"
6. Select the channel for SLO alerts (e.g., `#slo-alerts` or `#ci-alerts`)
7. Copy the webhook URL
8. In GitHub repo settings, go to Secrets > Actions > New repository secret
9. Name: `SLACK_WEBHOOK_URL`, Value: the copied URL

**Testing:**
```bash
curl -X POST -H 'Content-type: application/json' \
  --data '{"text":"Test SLO alert from nanovms"}' \
  YOUR_WEBHOOK_URL
```

### `PAGERDUTY_TOKEN`

The PagerDuty Events API v2 routing key (integration key) for SLO alerts.

**Setup steps:**
1. Go to https://www.pagerduty.com/ and log in
2. Navigate to Services > Service Directory
3. Select or create a service for SLO alerts (e.g., "CI/CD SLO Alerts")
4. Go to Integrations > Add Integration
5. Select "Events API v2"
6. Name it "GitHub Actions SLO Monitor"
7. Copy the Integration Key
8. In GitHub repo settings, go to Secrets > Actions > New repository secret
9. Name: `PAGERDUTY_TOKEN`, Value: the copied integration key

**Note:** PagerDuty alerts are only triggered for PAGE-level severity (burn rate >= 14.4x).

**Testing:**
```bash
curl -X POST https://events.pagerduty.com/v2/enqueue \
  -H 'Content-Type: application/json' \
  -d '{
    "routing_key": "YOUR_TOKEN",
    "event_action": "trigger",
    "payload": {
      "summary": "Test SLO alert from nanovms",
      "source": "github-actions",
      "severity": "info",
      "component": "slo-monitor",
      "class": "test"
    }
  }'
```

## Email Notifications (GitHub Issues)

Email notifications are delivered via GitHub issue creation. Any user subscribed to the repository (or specific issues) will receive email notifications when:

- A new SLO alert issue is created (for any severity above OK)
- An existing alert issue is updated with new burn rate data

To manage email notification preferences:
1. Go to the repository on GitHub
2. Click "Watch" in the top right
3. Select "All Activity" or "Custom" > check "Issues"

## Workflow Architecture

```
slo-monitor.yml (daily schedule)
  |
  |-- Calculates burn rate from 7-day CI data
  |-- Determines severity (OK/INFO/WARN/PAGE)
  |-- Creates/updates GitHub issue for WARN/PAGE
  |-- Uploads slo-report.json artifact
  |
  v
slo-alert.yml (workflow_run trigger)
  |
  |-- Downloads artifact from monitor run
  |-- Sends Slack alert (WARN/PAGE only)
  |-- Triggers PagerDuty incident (PAGE only)
  |-- Creates/updates GitHub issue (all severities)
```

## Troubleshooting

### Alert not sent to Slack

1. Check that `SLACK_WEBHOOK_URL` secret is configured correctly
2. Verify the webhook URL is active in Slack app settings
3. Check the "Send Slack alert" step in the workflow run for errors
4. The step uses `continue-on-error: true`, so check the run logs

### PagerDuty incident not created

1. Verify `PAGERDUTY_TOKEN` secret is configured
2. Ensure burn rate is >= 14.4x (PAGE threshold)
3. Check the PagerDuty service is active and not in maintenance mode
4. Review the curl output in the workflow run logs

### Artifact not found

1. Ensure the `slo-monitor.yml` workflow completed successfully
2. Check that the "Upload burn rate JSON artifact" step ran
3. The artifact retention is set to 30 days

### Burn rate calculation seems wrong

1. Verify the repository has CI runs in the last 7 days
2. Check the "Calculate burn rate" step logs for the raw data
3. The burn rate uses a 7-day rolling window; recent failures have more impact

## Tuning Thresholds

To adjust alert thresholds, modify the `env` section in `.github/workflows/slo-monitor.yml`:

```yaml
env:
  PAGE_THRESHOLD: 14.4   # Adjust PagerDuty trigger point
  WARN_THRESHOLD: 3.0    # Adjust Slack alert trigger point
  INFO_THRESHOLD: 1.0    # Adjust informational alert trigger point
```

Threshold reference:
- 14.4x = budget exhausted in ~2 days
- 3.0x = budget exhausted in ~10 days
- 1.0x = budget exhausted in ~30 days (end of month)
