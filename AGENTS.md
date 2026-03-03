# Repository Guidelines

## Project Structure & Module Organization
`xssh` is a Go CLI/TUI application. Keep code grouped by responsibility:
- `main.go`: program entrypoint.
- `cmd/`: CLI parsing and command dispatch.
- `app/`: Bubble Tea app model, update loop, view rendering, keymap.
- `pane/`: pane rendering, virtual terminal behavior, scrollback.
- `session/`: SSH and local shell session lifecycle.
- `config/`: config and group persistence (`~/.xssh/config.yaml`) plus SSH config parsing.
- `layout/`: pane grid logic.
- `selector/`: interactive host selector.
- `internal/vt10x/`: vendored terminal internals (local module replacement).

Place tests next to implementation files as `*_test.go`.

## Build, Test, and Development Commands
- `go build -o xssh .`: build local binary from current source.
- `go run . [targets...]`: run without producing a binary.
- `go test ./...`: run all unit tests across packages.
- `go test ./... -cover`: run tests with coverage summary.
- `go install github.com/xssh/xssh@latest`: install released binary.

Example local run: `go run . - -` (two local panes).

## Coding Style & Naming Conventions
- Follow standard Go formatting: run `gofmt -w` on changed files.
- Keep package names short, lowercase, and domain-focused (`pane`, `layout`, `session`).
- Exported identifiers use `PascalCase`; unexported use `camelCase`.
- Prefer small functions with explicit error handling and actionable error messages (`xssh: <context>: <err>` style).
- Keep CLI/help text and README examples synchronized when flags or shortcuts change.

## Testing Guidelines
- Use Go’s `testing` package (existing pattern across `cmd`, `config`, `layout`, `pane`, `session`, `app`).
- Name tests clearly: `Test<FunctionOrBehavior>`.
- Add/adjust tests for behavior changes, especially around argument parsing, pane layout, reconnect logic, and scroll behavior.
- Run `go test ./...` before opening a PR.

## Commit & Pull Request Guidelines
- Follow Conventional Commits seen in history: `feat:`, `fix:`, `docs:`, `chore:`.
- Keep commits focused and scoped to one change.
- PRs should include a concise behavior summary, linked issue (if applicable), and test evidence (`go test ./...` output).
- Include updated docs/help text when CLI flags, shortcuts, or UX behavior changes.
- Attach TUI screenshots or short terminal recordings for visible UI changes.

## Configuration & Security Tips
- Never commit real hostnames, usernames, private keys, or secrets.
- Use SSH aliases in `~/.ssh/config` and saved groups in `~/.xssh/config.yaml` for local workflows.
