# Repository Guidelines

## Project Structure & Module Organization
- Entry point lives in `cmd/yoga/main.go`; keep it minimal and wire dependencies from here.
- Domain logic sits under `internal/` (for example `internal/app` for TUI flow, `internal/fsutil` for filesystem helpers, `internal/meta` for metadata caching). Add new packages under `internal/` and expose functionality via small, testable functions.
- Tests accompany the code they verify (for example `internal/fsutil/*.go` with matching `_test.go` files). Keep any binaries or assets in dedicated folders and avoid checking large media into git.

## Build, Test, and Development Commands
- `mage build` — compile the TUI (`go build ./cmd/yoga`).
- `mage install` — install the binary into your Go bin path.
- `mage test` — run `go test ./...` quickly during development.
- `mage coverage` — enforce ≥85% coverage; run before pushing.
- Format Go files with `gofumpt ./...` or rely on your editor’s integration prior to commits.

## Coding Style & Naming Conventions
- Follow Go idioms: exported names are PascalCase, unexported names camelCase; keep functions <50 lines and split shared logic once it exceeds 5 lines.
- Group types so that any type with >3 methods goes into its own file named after the type.
- Run `gofumpt` on every modified Go file; no files should exceed 1000 lines. Prefer short, single-purpose packages rather than sprawling utilities.

## Testing Guidelines
- Use Go’s standard `testing` package; name files `*_test.go` and test functions `TestThingDoesWhat`. Table-driven tests help keep coverage high without duplication.
- Mock external effects (filesystem, VLC launching) via interfaces in `internal` so behaviour remains deterministic.
- Always run `mage test` and `mage coverage` locally; investigate any coverage dips before merging.
