# AGENTS.md

Guidance for coding agents working in this repository.

## Project summary

- Project: `pcloud-cli`
- Language: Go (modules)
- Entry point: `cmd/pcloud-cli/main.go`
- CLI package: `internal/cli`
- Internal packages: `internal/config`, `internal/helpers`, `internal/pcloud/models`

## Core commands

Use these commands from repository root.

- Build all packages: `go build ./...`
- Build binary (Makefile): `make build`
- Run formatter check: `make fmt`
- Run lints: `go vet ./...` and `make lint` (if `golint` is installed)
- Run tests: `go test ./...`

## CI expectations

The workflow in `.github/workflows/ci.yml` enforces:

1. `gofmt` formatting
2. `go vet ./...`
3. `go build ./...`

Any change should pass these checks locally before handoff.

## Coding conventions

- Keep changes minimal and focused on the requested task.
- Preserve existing CLI behavior and flags unless explicitly asked to change UX.
- Use idiomatic Go naming and package boundaries.
- Prefer extracting small helpers over copying logic across command handlers.
- Do not add new dependencies unless they are clearly justified.

## Repository-specific notes

- This project uses Go modules (`go.mod`, `go.sum`) as source of truth.
- Do not reintroduce legacy dep tooling (`govendor`, `Godeps`, etc.).
- Never add credentials to git (source, docs, tests, examples, or CI files).
- Never hardcode access tokens, OAuth client IDs, OAuth client secrets, API keys, or passwords.
- Keep secrets and tokens out of logs, docs, and commit history.
- Avoid touching unrelated command files when changing one subcommand.

## Safe refactor checklist

When doing structural or cross-file refactors:

1. Update imports/package names first.
2. Build immediately: `go build ./...`.
3. Run vet/format checks.
4. Update Makefile/README only if behavior changed.
5. Summarize changed paths and validation steps in handoff.
