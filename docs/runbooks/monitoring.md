# Monitoring Runbook

## sandbox-create high error rate

1. Check `kubectl logs -n nanovms -l app=daemon --tail=200`
2. Check adapter health: `curl -sf http://daemon:8080/healthz`
3. Check resource saturation: `kubectl top pods -n nanovms`
4. If adapter is failing, check the specific error type in metrics
5. If OOM, increase memory limits in deployment manifest
