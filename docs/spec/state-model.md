# State Model

## Session States

Allowed states:

- `active`
- `paused`
- `ended`
- `recovered`

Rules:

- Every new session starts as `active`.
- A session must not return from `ended` to `active`.
- Future work should create a new session instead of reusing an ended one.

Allowed transitions:

- `active -> paused`
- `active -> ended`
- `active -> recovered`
- `paused -> ended`
- `paused -> recovered`

## Memory Note States

Allowed states:

- `active`
- `resolved`
- `superseded`

Rules:

- New notes start as `active` unless explicitly imported otherwise.
- Notes should not be hard-deleted merely because they are resolved.
- Retrieval may rank `active` above `resolved` above `superseded`.

Allowed transitions:

- `active -> resolved`
- `active -> superseded`
- `resolved -> superseded`

## Handoff States

Allowed states:

- `open`
- `completed`
- `abandoned`

Rules:

- Handoffs intended for continuation should start as `open`.
- Bootstrap should prefer the latest `open` handoff over `completed` or `abandoned` handoffs.

Allowed transitions:

- `open -> completed`
- `open -> abandoned`

## Handoff Kinds

Kinds are not states, but they affect interpretation.

Allowed kinds:

- `final`
- `checkpoint`
- `recovery`

Rules:

- `final` is a normal end-of-session handoff.
- `checkpoint` is an intermediate continuation snapshot.
- `recovery` is inferred after interruption and must remain clearly labeled.
