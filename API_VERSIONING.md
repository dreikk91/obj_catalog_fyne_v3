# API Versioning

This repository uses explicit versioned in-process API packages:

- `pkg/frontendapi/v1`
- `pkg/adminapi/v1`

They are versioned even though the app currently runs as one process. The goal is contract stability, not network transport.

## Stability Rules

Everything exported from `v1` packages is treated as stable for UI callers in this repository.

Stable means:

- exported DTO field meanings must not change silently
- exported provider method semantics must not change silently
- existing success/error behavior should remain compatible

## Allowed `v1` Changes

The following changes are allowed in `v1`:

- add new optional fields
- add new provider interfaces
- add new helper/mapper functions
- add new enum values when older callers can safely ignore them

The following changes are not allowed in `v1`:

- rename or remove exported fields
- change field meaning while keeping the same name
- rename or remove exported provider methods
- change a method from supported to unsupported without a capability signal
- repurpose existing enum values

## When To Create `v2`

Create `v2` when a required change would break an existing `v1` caller, for example:

- a DTO needs incompatible field semantics
- an enum needs a different model, not just an extra value
- a provider method contract must change incompatibly
- an old shape was too source-specific and must be redesigned

`v2` should live alongside `v1`. Do not rewrite `v1` in place.

## Mapping Rules

`pkg/backend` is the only place where versioned API packages should be translated to:

- legacy `pkg/contracts`
- backend implementation details

UI code should not do this mapping directly.

## Practical Guidance

Before adding a new field or method:

1. Decide whether the concept belongs in `frontendapi/v1`, `adminapi/v1`, or an internal UI model.
2. Prefer additive changes first.
3. If the change is incompatible, create `v2` instead of mutating `v1`.
