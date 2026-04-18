# ADR: UI Boundary Rules

## Status

Accepted.

## Context

The project remains a single Go/Fyne application, but it now has explicit in-process boundaries:

- `pkg/frontendapi/v1` for the main frontend/backend contract
- `pkg/adminapi/v1` for admin flows
- `pkg/backend` adapters for mapping between versioned APIs and legacy providers

Without a written rule, new UI code can easily regress into direct usage of `pkg/contracts/admin_*` and bypass the boundary.

## Decision

The following rules are mandatory for new UI-facing code:

1. UI code in `pkg/ui`, `pkg/ui/dialogs`, `pkg/application` must not depend directly on `pkg/data`.
2. UI code must not use `pkg/contracts/admin_*` as its primary boundary for new admin flows.
3. Main UI flows must use `frontendapi/v1` or `contracts.FrontendBackend`.
4. Admin UI flows must use `adminapi/v1`.
5. `pkg/backend` adapters are the only supported place for mapping between:
   - versioned API DTOs
   - legacy `pkg/contracts` DTOs and interfaces
   - backend provider implementations in `pkg/data`
6. If temporary compatibility is required, it must live in an adapter/compat layer, not be spread across dialogs.

## Consequences

Benefits:

- UI and backend can evolve with less coupling.
- Legacy contracts stay isolated.
- Future transport extraction remains possible without rewriting dialogs again.

Costs:

- Additional mapper/adapter code.
- Some existing object viewmodels still carry legacy `pkg/contracts` DTOs internally and remain technical debt.

## Follow-up

The next cleanup step is to move remaining object-oriented UI viewmodels away from `contracts.AdminObject*` toward either:

- `adminapi/v1` DTOs, or
- a dedicated internal UI model owned by `pkg/ui/viewmodels`
