# Specifications

- Remove only superseded root snapshot artifacts.
- Keep canonical replacements and all functional source files intact.
- Record validation results and any unrelated failures in the task worklog.

## ARUs

- Assumption: root-level raw worklogs are pre-consolidation snapshots because later canonical replacements exist in the same repository history.
- Risk: deleting a referenced status snapshot could break a comment-level reference only.
- Uncertainty: no explicit task note names the files, so cleanup is constrained to artifacts with direct replacement evidence.
