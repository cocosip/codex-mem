# Conformance Matrix

## Purpose

This appendix defines a minimal scenario matrix for validating `codex-mem` v1 implementations.

## Matrix

| ID | Scenario | Expected Result |
|---|---|---|
| C01 | Empty store bootstrap | New session created successfully; no error; startup brief is minimal |
| C02 | Same-workspace recovery | Latest open workspace handoff is preferred during bootstrap |
| C03 | Project fallback recovery | Project handoff is used when workspace handoff is absent |
| C04 | No search hits | Search succeeds with empty result list, not error |
| C05 | Cross-project isolation | Default retrieval excludes unrelated project memory |
| C06 | Related-project expansion | Related-project results appear only when explicitly enabled and are labeled |
| C07 | Save note scope validation | Invalid session/scope mismatch is rejected |
| C08 | Save handoff validity | Handoff without actionable `next_steps` is rejected |
| C09 | Privacy exclusion | Private/do-not-store content is excluded from durable searchable memory |
| C10 | Import suppression | Imported duplicate or privacy-blocked artifact is suppressed with warning/audit visibility |
| C11 | AGENTS safe install | Existing AGENTS file is not silently overwritten in safe/default mode |
| C12 | Identity conflict handling | Conflicting scope identity produces warning or error, not silent merge |

## Recommended Test Grouping

### Core continuity

- C01
- C02
- C03
- C08

### Retrieval safety

- C04
- C05
- C06

### Integrity and privacy

- C07
- C09
- C10
- C12

### Workflow integration

- C11

## Conformance Recommendation

An implementation should not claim v1 readiness unless all scenarios above are handled consistently with the v1 baseline.
