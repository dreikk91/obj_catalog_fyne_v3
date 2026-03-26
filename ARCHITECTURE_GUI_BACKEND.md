# GUI / Backend Architecture

## Layering

- `pkg/application`: app orchestration layer (startup wiring, UI composition, lifecycle).
- `pkg/ui` and `pkg/ui/dialogs`: GUI layer (Fyne widgets, dialogs, view state).
- `pkg/contracts`: boundary layer (interfaces + shared DTOs used by both sides).
- `pkg/backend`: backend composition adapter (creates backend providers and returns contract interfaces).
- `pkg/data`: backend implementation (DB queries and admin operations).

Dependency direction:

`main -> application -> (ui + backend via contracts)`

GUI must not import `pkg/data` directly.

## Current wiring

- `main.go` is a thin bootstrap entrypoint only.
- `pkg/application` stores provider as `contracts.DataProvider`.
- `pkg/application` creates provider via `backend.NewDBProvider(...)`.
- Admin features are enabled through `contracts.AdminProvider` type assertion.

## Compatibility

`pkg/data/provider.go` and `pkg/data/admin_provider.go` now contain type aliases to `pkg/contracts`.
This keeps older imports working while new code should use `pkg/contracts` directly.

## Rules for future changes

- Add/modify shared interfaces and DTOs only in `pkg/contracts`.
- Keep DB-specific structs/queries inside `pkg/data`.
- Keep UI logic and widgets inside `pkg/ui`.
- If backend implementation changes (new DB, API, mock), expose it through `pkg/backend` without touching GUI.
