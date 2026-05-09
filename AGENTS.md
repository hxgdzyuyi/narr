# Repository Guidelines

## Scope

This repository is implementing `narrc`, a Go CLI for the Narr language.
The language grammar source of truth is `docs/syntax.md`; milestone planning
is tracked in `docs/plans/narrc-go-implementation-plan.md`.

## Build And Test

- Format Go changes with `gofmt`.
- Run `go test ./...` before handing off Go changes.
- Use `go run ./cmd/narrc --version` for the smallest CLI smoke test.
- Use `go run ./cmd/narrc lint --project examples/红楼梦` to verify the M1
  project-loading path.

## Project Layout

- CLI entry point: `cmd/narrc/main.go`.
- CLI parsing and command dispatch: `internal/cli`.
- Project discovery, config loading, and file collection: `internal/project`.
- Diagnostics and source positions: `internal/source`.
- User-facing output formatting: `internal/format`.

## Implementation Notes

- Keep parser/checker/evaluator work aligned with `docs/syntax.md`.
- Do not add syntax that is not documented in `docs/syntax.md`.
- Keep generated build output out of source control.
