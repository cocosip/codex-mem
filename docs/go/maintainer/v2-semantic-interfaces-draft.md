# V2 Semantic Interfaces Draft

## Status

This document is a draft for potential `codex-mem` v2 work.

It is exploratory and non-normative.
The current implementation and v1 spec remain the source of truth unless a future v2 design is adopted.

## Purpose

The current v2 draft set now covers:

- hybrid retrieval direction
- runtime resurfacing behavior
- configuration gates
- embedding storage and backfill design
- conformance and operator workflows

This document narrows the next step:

- how those ideas should map onto the current Go package boundaries
- which interfaces should be introduced before implementation grows
- how to preserve the existing v1 retrieval entrypoints while adding semantic internals

Primary references:

- [Go Implementation Plan](./implementation-plan.md)
- [V2 Hybrid Retrieval Roadmap](./v2-hybrid-retrieval-roadmap.md)
- [V2 Runtime Resurfacing Draft](../../spec/v2-runtime-resurfacing.md)
- [V2 Embedding Storage And Backfill Draft](../../spec/v2-embedding-storage-draft.md)
- [V2 Conformance Scenarios Draft](../../spec/v2-conformance-scenarios-draft.md)

## Current Code-Facing Baseline

The current implementation already has a useful v1 shape:

- `internal/domain/retrieval.Service` owns bootstrap, recent-history, and search behavior
- `internal/domain/memory.Service` owns note validation and storage
- `internal/db.MemoryRepository` owns SQLite-backed note persistence and lexical search
- MCP handlers already call stable retrieval and memory entrypoints

That means v2 does not need a new top-level retrieval service.
It needs narrower collaborating interfaces introduced around the current shape.

## Interface Design Goals

The first implementation-facing interface layer should preserve these properties:

- existing v1 tool contracts remain stable
- lexical retrieval still works when semantic dependencies are nil or disabled
- semantic code is injectable behind interfaces, not hard-wired into `retrieval.Service`
- policy and ranking remain in Go domain code
- semantic state and embedding lifecycle do not pollute every v1 DTO unless necessary

## Recommended Package Ownership

### `internal/domain/retrieval`

Should own:

- hybrid candidate orchestration
- fusion and reranking
- runtime consult decisions for implicit resurfacing
- working-context shaping for resurfaced memory

### `internal/domain/memory`

Should own:

- note eligibility decisions for embeddings
- projection of a note into embedding source text
- embedding metadata transitions for note records

### `internal/db`

Should own:

- persistence of embedding metadata in SQLite
- lexical note search and current v1 record reads
- SQL-backed queries for pending, stale, ready, and failed embedding metadata

### `internal/app`

Should own:

- wiring for semantic dependencies
- operator-facing backfill, rebuild, and health workflows
- config-gated enablement and diagnostics flow

## Recommended First-Cut Types

The first v2 implementation should add semantic-specific types rather than overloading v1 search results immediately.

Suggested types:

```go
package retrieval

type CandidateSource string

const (
    CandidateSourceLexical  CandidateSource = "lexical"
    CandidateSourceSemantic CandidateSource = "semantic"
    CandidateSourceFused    CandidateSource = "fused"
)

type CandidateReason struct {
    Label   string
    Summary string
}

type Candidate struct {
    Kind           RecordKind
    ID             string
    Scope          scope.Ref
    Title          string
    Summary        string
    Importance     int
    CreatedAt      time.Time
    Source         string
    RelationType   string
    LexicalScore   float64
    SemanticScore  float64
    CandidateSource CandidateSource
    Reasons        []CandidateReason
}
```

These are internal retrieval-layer types.
They do not need to become public MCP payloads on day one.

## Suggested Retrieval Interfaces

The current `MemoryReader` and `HandoffReader` are still useful.
V2 should add semantic collaborators next to them instead of replacing them.

Recommended additions:

```go
package retrieval

type SemanticQuery struct {
    Scope                  scope.Ref
    QueryText              string
    Limit                  int
    IncludeRelatedProjects bool
}

type SemanticCandidate struct {
    RecordID       string
    RecordKind     RecordKind
    Scope          scope.Ref
    SemanticScore  float64
    RelationType   string
    ExplanationTag string
}

type SemanticSearcher interface {
    SearchNotes(ctx context.Context, query SemanticQuery) ([]SemanticCandidate, error)
    Available(ctx context.Context, scope scope.Ref) (bool, error)
}

type RuntimeConsultEvaluator interface {
    ShouldConsult(ctx context.Context, input RuntimeConsultInput) (RuntimeConsultDecision, error)
}
```

Why this shape fits the current code:

- `retrieval.Service` can keep using lexical note and handoff readers
- semantic search remains note-only at first
- availability checks allow lexical fallback without spreading backend logic everywhere

## Suggested Runtime Resurfacing Interfaces

The runtime resurfacing path should stay internal to retrieval, but its sub-pieces should still be separable.

Recommended internal interfaces:

```go
package retrieval

type RuntimeConsultInput struct {
    Scope             scope.Ref
    RequestText       string
    ActiveTask        string
    ExplicitSearch    bool
    RecentRecordIDs   []string
}

type RuntimeConsultDecision struct {
    Consult        bool
    Confidence     float64
    ReasonLabels   []string
    RelatedAllowed bool
}

type ResurfacingCache interface {
    RecentlyInjected(taskFingerprint string, recordID string) bool
    MarkInjected(taskFingerprint string, recordID string, confidence float64)
}

type WorkingContextSnippet struct {
    RecordID      string
    Kind          RecordKind
    Title         string
    Scope         scope.Ref
    Source        string
    RelationType  string
    ReasonLabel   string
    ReasonSummary string
    DurableSummary string
}
```

This keeps the implicit path bounded and testable without turning it into a separate public subsystem.

## Suggested Memory-Domain Interfaces

The memory domain should decide note eligibility and source projection instead of leaving that logic in `internal/app`.

Recommended additions:

```go
package memory

type EmbeddingStatus string

const (
    EmbeddingStatusNotApplicable EmbeddingStatus = "not_applicable"
    EmbeddingStatusPending       EmbeddingStatus = "pending"
    EmbeddingStatusReady         EmbeddingStatus = "ready"
    EmbeddingStatusStale         EmbeddingStatus = "stale"
    EmbeddingStatusFailed        EmbeddingStatus = "failed"
)

type EmbeddingMetadata struct {
    NoteID              string
    Eligible            bool
    Status              EmbeddingStatus
    ModelID             string
    ContentHash         string
    ContentVersion      int
    EmbeddedAt          time.Time
    ErrorCode           string
    ErrorAt             time.Time
}

type EmbeddingProjection struct {
    NoteID         string
    SourceText     string
    ContentHash    string
    ContentVersion int
}

type EmbeddingProjector interface {
    ProjectNote(note Note) (EmbeddingProjection, error)
}
```

This lets note-specific rules stay with note semantics.

## Suggested Persistence Interfaces

SQLite should continue to own embedding metadata.
That suggests adding a narrower repository interface instead of pushing semantic state into the current `memory.Repository` too broadly.

Recommended shape:

```go
package memory

type EmbeddingMetadataRepository interface {
    GetEmbeddingMetadata(noteID string) (*EmbeddingMetadata, error)
    UpsertEmbeddingMetadata(metadata EmbeddingMetadata) error
    ListEmbeddingCandidates(scope scope.Ref, limit int, statuses []EmbeddingStatus) ([]Note, error)
    ListReadyEmbeddingNotes(scope scope.Ref, limit int) ([]Note, error)
}
```

Why separate it:

- `SaveNote` remains a small v1-oriented path
- backfill workflows can depend on embedding metadata without complicating ordinary note writes
- migrations remain easier to stage because semantic writes become additive

## Suggested Semantic Backend Interfaces

The storage draft recommends splitting embedding generation from the semantic index itself.

Recommended shape:

```go
package retrieval

type EmbeddingVector struct {
    ModelID string
    Values  []float32
}

type EmbeddingProvider interface {
    ModelID(ctx context.Context) (string, error)
    Embed(ctx context.Context, texts []string) ([]EmbeddingVector, error)
}

type SemanticIndexRecord struct {
    RecordID        string
    Scope           scope.Ref
    ContentVersion  int
    ModelID         string
    Vector          []float32
}

type SemanticIndex interface {
    UpsertNotes(ctx context.Context, records []SemanticIndexRecord) error
    SearchNotes(ctx context.Context, queryVector []float32, ref scope.Ref, limit int, includeRelatedProjects bool) ([]SemanticCandidate, error)
    Health(ctx context.Context, ref scope.Ref) (SemanticIndexHealth, error)
    Rebuild(ctx context.Context, ref scope.Ref) error
}

type SemanticIndexHealth struct {
    Available    bool
    Compatible   bool
    Version      string
    WarningCodes []string
}
```

This is intentionally biased toward note-only rollout.

## Retrieval Service Evolution

The safest change to `retrieval.Service` is constructor expansion by options, not by replacing existing collaborators abruptly.

Suggested direction:

```go
type Options struct {
    SemanticSearcher       SemanticSearcher
    RuntimeConsultEvaluator RuntimeConsultEvaluator
    ResurfacingCache       ResurfacingCache
}

func NewService(
    scopeResolver ScopeResolver,
    sessionStarter SessionStarter,
    memoryReader MemoryReader,
    handoffReader HandoffReader,
    options Options,
) *Service
```

Why options are safer here:

- existing callers can keep nil semantic dependencies
- lexical-only behavior remains the default when options are empty
- implementation spikes can land in smaller patches

## Suggested Backfill Application Interfaces

Operator workflows should likely live in `internal/app`, but they need explicit service boundaries.

Recommended shape:

```go
package app

type SemanticBackfillService interface {
    BackfillNotes(ctx context.Context, input SemanticBackfillInput) (SemanticBackfillOutput, error)
    RebuildIndex(ctx context.Context, input SemanticRebuildInput) (SemanticRebuildOutput, error)
    Health(ctx context.Context, input SemanticHealthInput) (SemanticHealthOutput, error)
}
```

This keeps CLI or MCP-facing orchestration separate from storage and retrieval internals.

## First Implementation Slice

If implementation begins, the smallest code-facing slice should be:

1. add additive note embedding metadata support in SQLite
2. add memory-domain projection and metadata types
3. add retrieval-layer semantic interfaces behind nil-safe options
4. keep semantic dependencies disabled in normal app wiring
5. add backfill and health workflows behind config gates

This sequence matches the existing codebase shape and avoids reopening stable v1 entrypoints too early.

## Current Recommendation

Use this document as the maintainer-facing bridge from the v2 design set into Go implementation planning.

The best next implementation step is not a full semantic feature.
It is introducing the narrow interfaces and additive types that let lexical fallback remain boring.
