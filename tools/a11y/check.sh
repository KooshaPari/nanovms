#!/usr/bin/env bash
# nanovms a11y audit (L76).
# Validates locales/en.json shape; real pa11y audit runs against the dashboard.

set -euo pipefail
echo "[a11y] nanovms L76 audit"
if [ -f "locales/en.json" ]; then
  python3 -c "import json; d=json.load(open('locales/en.json')); assert isinstance(d, dict); print(f'[a11y] locales/en.json: {len(d)} keys ok')"
fi
echo "[a11y] Real audit: pa11y against dashboard URL when available"
exit 0
