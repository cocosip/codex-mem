# Glossary

## Core Terms

### System

A larger product, platform, business domain, or technical system containing one or more related projects.

### Project

One logical codebase or repository within a system. This is the default memory isolation boundary.

### Workspace

One concrete local checkout or worktree of a project.

### Session

One Codex conversation lifecycle in a workspace.

### Memory Note

A high-value structured memory item intended to help future sessions act faster or more correctly.

### Handoff

A structured continuation record written when a session pauses, checkpoints, or ends.

### Startup Brief

A compact continuation packet derived from prior memory during session bootstrap.

### Related Project

Another project in the same system that may be retrieved only under controlled policy.

### Import Record

A record that tracks imported data from secondary sources such as watchers or relay logs.

## Scope Terms

### Current Workspace

The local working copy from which the active Codex session is operating.

### Current Project

The logical repository or codebase associated with the current workspace.

### Current System

The broader product grouping associated with the current project.

## Provenance Terms

### codex_explicit

A record explicitly written by Codex through the memory tool flow.

### watcher_import

A record or artifact imported from a local watcher or cache reader.

### relay_import

A record or artifact imported from a relay-side capture path.

### recovery_generated

A record synthesized during recovery after interruption.

## Retrieval Terms

### Scope-Local

Memory from the current workspace or current project.

### Cross-Project Retrieval

Retrieval that includes a different project from the same system.

### Cross-System Retrieval

Retrieval that reaches outside the current system. This is never default behavior.

### High-Value Memory

Durable, reusable, continuity-relevant information such as decisions, bug root causes, confirmed fixes, discoveries, constraints, preferences, and unresolved todos.
