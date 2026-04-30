# Architecture Decisions & Refactoring

## Index

Add entries as ADRs, library extractions, or refactoring decisions are made.

## 2026-04-28 - Compatibility matrix as source of truth

- Treat `SPEC.md` and `README.md` as the canonical compatibility matrix for NanoVMS.
- Baseline OS targets are Windows, Linux, and macOS; adjacent compatibility layers stay reference-only unless product scope changes.
- The current adapter code is the authoritative source for implementation status, so docs should not promote planned surfaces to baseline support.
