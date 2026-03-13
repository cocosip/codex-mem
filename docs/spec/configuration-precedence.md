# Configuration and Precedence

## Purpose

This document defines how policy layers interact in `codex-mem` v1.

## Policy Layers

Recommended layers:

1. product defaults
2. global user configuration
3. global `AGENTS.md`
4. project configuration
5. project `AGENTS.md`
6. runtime tool parameters
7. explicit private or do-not-store intent

## Precedence

Strongest to weakest:

1. explicit private or do-not-store intent
2. runtime tool parameters
3. project configuration
4. project `AGENTS.md`
5. global user configuration
6. global `AGENTS.md`
7. product defaults

## Rules

- Privacy exclusions outrank retrieval convenience.
- Runtime overrides apply only to the current invocation unless explicitly persisted.
- `AGENTS.md` influences workflow behavior but does not replace durable memory state.

## Configurable Areas

Recommended configurable areas:

- bootstrap limits
- default minimum importance
- related-project retrieval allowance
- checkpoint encouragement
- exclusion patterns
- raw import retention
- AGENTS installation defaults

## Non-Configurable Core Safety Rules

These should remain fixed for v1:

- unrelated project memory must not appear in default retrieval
- private content must remain excluded
- bootstrap must create a new session
- zero search results is not an error

## Transparency

Implementations should make it understandable:

- which policy layer affected a decision
- why related-project retrieval was used or not used
- why AGENTS installation wrote or skipped a file
