# Operator Wails Shell

Stage-1 shell for migration from Fyne UI to React + Wails with live backend support.

## Dev run

From `cmd/operator-wails`:

```bash
wails dev
```

## Build frontend only

From repository root:

```bash
cd frontend
npm run build
```

## Current backend mode

The shell tries to initialize real backend sources from environment variables.
If no source can be initialized it falls back to `shell-only mode`.

### Environment variables

- `MOST_FIREBIRD_ENABLED` (`true|false`, default `true`)
- `MOST_DB_USER`, `MOST_DB_PASSWORD`, `MOST_DB_HOST`, `MOST_DB_PORT`, `MOST_DB_PATH`, `MOST_DB_PARAMS`
- `MOST_PHOENIX_ENABLED` (`true|false`, default `false`)
- `MOST_PHOENIX_USER`, `MOST_PHOENIX_PASSWORD`, `MOST_PHOENIX_HOST`, `MOST_PHOENIX_PORT`, `MOST_PHOENIX_INSTANCE`, `MOST_PHOENIX_DATABASE`, `MOST_PHOENIX_PARAMS`
- `MOST_CASL_ENABLED` (`true|false`, default `false`)
- `MOST_CASL_BASE_URL`, `MOST_CASL_TOKEN`, `MOST_CASL_EMAIL`, `MOST_CASL_PASSWORD`, `MOST_CASL_PULT_ID`
- `MOST_BACKEND_MODE` (`firebird|phoenix|casl_cloud`, default `firebird`)
