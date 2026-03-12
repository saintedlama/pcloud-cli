# AGENTS.md

Guidance for coding agents working in this repository.

## Project summary

- Project: `pcloud-cli`
- Language: Go (modules)
- Entry point: `cmd/pcloud-cli/main.go`
- CLI package: `internal/cli`
- Internal packages: `internal/cli`, `internal/helpers`, `internal/pcloud`, `internal/pcloud/models`, `internal/sync`, `internal/tui`
- There is **no** `internal/config` package — configuration is loaded via viper directly in `internal/cli/pcloud.go`

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
4. `go test ./...` (with `gotestfmt` for output formatting and a coverage report in the job summary)

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

---

## pCloud API reference

Base documentation: https://docs.pcloud.com/

### API regions and base URLs

pCloud stores user data in one of two regions. The correct base URL is returned
as `hostname` in the OAuth redirect and must be stored in config (`base_url`).

| Region | Base URL |
|--------|----------|
| US     | `https://api.pcloud.com` |
| EU     | `https://eapi.pcloud.com` |

All API calls must go to the region where the user's data resides. Mixing regions
returns `result: 1000` (Log in required) even with a valid token.

### HTTP protocol

- Parameters may be passed via GET query string or POST body.
- Every response is JSON. `result: 0` means success; non-zero is an error with an
  accompanying `error` string field.
- Docs: https://docs.pcloud.com/protocols/http_json_protocol/

### Endpoints used in this project

#### Folder operations

| Endpoint | CLI command | Docs |
|----------|-------------|------|
| `POST /listfolder` | `folder list` | https://docs.pcloud.com/methods/folder/listfolder.html |
| `POST /createfolder` | `folder create` | https://docs.pcloud.com/methods/folder/createfolder.html |
| `POST /deletefolder` | `folder delete` | https://docs.pcloud.com/methods/folder/deletefolder.html |
| `POST /renamefolder` | `folder rename` | https://docs.pcloud.com/methods/folder/renamefolder.html |

Key notes:
- `listfolder` accepts `path` or `folderid`; returns `metadata` with a `contents` array.
- `deletefolder` only deletes **empty** folders; use `deletefolderrecursive` for non-empty ones.
- All folder endpoints accept either `path` (string, discouraged by pCloud) or `folderid` (int, preferred).

#### File operations

| Endpoint | CLI command | Docs |
|----------|-------------|------|
| `POST /uploadfile` | `file upload` | https://docs.pcloud.com/methods/file/uploadfile.html |
| `POST /deletefile` | `file delete` | https://docs.pcloud.com/methods/file/deletefile.html |
| `POST /renamefile` | `file rename` | https://docs.pcloud.com/methods/file/renamefile.html |
| `POST /copyfile` | `file copy` | https://docs.pcloud.com/methods/file/copyfile.html |
| `POST /checksumfile` | `file checksum` | https://docs.pcloud.com/methods/file/checksumfile.html |

Key notes:
- `uploadfile` uses `multipart/form-data`; parameters must come before file data in the body.
- `checksumfile` returns `sha1` on both regions; `md5` on US only; `sha256` on EU only.
- `renamefile` can move files across folders by providing `tofolderid` or `topath`.

#### Streaming / download links

| Endpoint | CLI command | Docs |
|----------|-------------|------|
| `POST /getfilelink` | `file get` | https://docs.pcloud.com/methods/streaming/getfilelink.html |
| `POST /getziplink` | `folder download` | https://docs.pcloud.com/methods/archiving/getziplink.html |

Key notes:
- Both endpoints return a `hosts` array and a `path`. The download URL is constructed as
  `https://hosts[0] + path` — do **not** call the API base URL directly for the download.
- `getziplink` expects a [tree structure](https://docs.pcloud.com/structures/tree.html)
  (`folderid`, `folderids`, `fileids`, etc.) rather than a plain path.

---

## pCloud API authentication

Docs: https://docs.pcloud.com/methods/intro/authentication.html

### Auth mechanism used by this project

This project uses **session auth** exclusively. OAuth 2.0 is not used.

- The `onboard` command calls `POST /userinfo?getauth=1&username=...&password=...`.
- pCloud returns an `auth` session token in the response.
- The token is saved to `~/.pcloud.json` as `auth_token`. Credentials are never stored.
- Every subsequent API call passes the token as `?auth=<token>` query parameter.
- Session tokens work with **all** endpoints including archiving ones (`getziplink`, `getzip`).
- Tokens have a finite lifetime and expire after a period of inactivity. Re-run `onboard` to refresh.

### `onboard` flow

1. User selects region (US / EU).
2. User enters email and password (prompt, never stored).
3. CLI calls `POST /userinfo?getauth=1&username=...&password=...` — no Bearer header.
   The Bearer header must be absent; if present, pCloud validates the existing session
   instead of issuing a new token, returning an empty `auth` field.
4. Response `auth` field is saved to config.

### Config file format (`~/.pcloud.json`)

```json
{
  "auth_token": "<session auth token>",
  "base_url":   "https://eapi.pcloud.com",
  "userid":     12345
}
```

The `base_url` **must** match the region where the user's account is registered.

## Coding Guidelines for Agents

- Never use section comments like `// ---- normalizePath ----` or `// ---- end normalizePath ----` in code. This is an antipattern of code monkeys not knowing that files, functions and packages exist to structure code.

- Semantic Commits: Use prefixes 'chore:', 'fix:', 'feat:' for commit messages to indicate the type of change. For example, 'fix: correct file upload logic' or 'feat: add folder rename command'. Chore is for non-functional changes like refactors or documentation updates. Fix is for bug fixes. Feat is for new features or commands.

- One file per subcommand: every new CLI subcommand lives in its own file in `internal/cli/`, named after the command (e.g. `internal/cli/deletefile.go` for `file delete`). Use `init()` to call `parent.AddCommand(cmd)` and to bind flags with `cmd.Flags()`. Never register commands from anywhere other than the command's own `init()`.

- Error wrapping style: wrap errors with `fmt.Errorf("context phrase: %w", err)`. Use `errors.New()` for static validation messages. Error messages are lowercase with no trailing period (e.g. `"listing pCloud tree: %w"`). Do not define custom error types unless there is a clear need to inspect them with `errors.As`.

- CLI error exits: in command handler functions, handle errors by printing a message and calling `os.Exit(1)`. Never panic in a command handler. Sync operations are an exception — they log non-fatal warnings via `fmt.Fprintf(log, "warn: ...")` and continue rather than aborting.

- New API endpoints: every new pCloud endpoint needs (1) a response model struct in `internal/pcloud/models/` in its own file named after the endpoint (e.g. `models/renamefile.go`), and (2) a method on `*API` in `internal/pcloud/api.go` that constructs a `Request` and calls `p.Query(req)`. Exception: `UserinfoResponse` is defined inline in `api.go` because it is used by both `GetUserInfo` and `LoginWithPassword` — authentication types may stay in `api.go`.

- Test style: use table-driven tests with a local `tests := []struct{ name, in, want ... }` slice and `for _, tt := range tests { t.Run(tt.name, ...) }`. Use `testify/assert` for non-fatal assertions and `testify/require` to abort on unexpected errors. Mock HTTP responses with `httptest.NewServer()`.

- Context usage: pass `context.Context` in long-running operations (`Syncer.Run`, `Uploader.Run`, `Watch` loops) and check cancellation with `if ctx.Err() != nil`. Do not add context parameters to individual CLI command handler functions — cobra handlers have the signature `func(cmd *cobra.Command, args []string)` and do not use context.
