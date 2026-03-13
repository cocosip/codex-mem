# Migration Examples

## Purpose

This appendix provides concrete examples for identity-preserving and identity-changing migration scenarios.

## Example 1: Project Rename

Scenario:

- Repository display name changes from `order-ui` to `order-web`.
- Repository remote and logical codebase remain the same.

Recommended behavior:

- preserve `project_id`
- update display metadata only

Reason:

- logical identity has not changed

## Example 2: Workspace Path Move

Scenario:

- Local workspace moves from `D:/Code/go/order-web` to `E:/src/order-web`.
- Repository remote still matches the same project.

Recommended behavior:

- preserve `project_id`
- preserve `workspace_id` only if continuity evidence supports it
- otherwise create a new workspace under the same project and warn

## Example 3: Remote URL Form Change

Scenario:

- Remote changes from `git@github.com:example/order-api.git`
- to `https://github.com/example/order-api`

Recommended behavior:

- normalize remote identity
- preserve `project_id`

Reason:

- the remote changed format, not logical ownership

## Example 4: Repository Transfer

Scenario:

- Repository moves from `github.com/old-org/order-sdk` to `github.com/new-org/order-sdk`
- Ownership changes but logical project continuity remains explicit

Recommended behavior:

- preserve `project_id` if migration policy confirms continuity
- update canonical remote metadata
- record migration provenance

## Example 5: Project Split

Scenario:

- One repository is split into `billing-api` and `billing-worker`

Recommended behavior:

- historical records stay attached to the old project identity
- future records use the new project identities
- any migration of old records must be explicit and auditable

## Example 6: Project Merge

Scenario:

- Two repos are merged into one new monorepo

Recommended behavior:

- do not silently merge historical memory pools
- create new identity rules for future records
- migrate historical data only with explicit policy and provenance
