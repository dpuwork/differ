# AGENTS.md

Instructions and context for AI agents working on `differ`.

## Critical Gotchas (Do Not Miss)

- **V2 Imports (Crucial)**: This project uses Bubble Tea v2. All Bubble Tea dependencies must be imported from `charm.land`, NOT `github.com/charmbracelet`.
  - `charm.land/bubbletea/v2` (aliased as `tea`)
  - `charm.land/bubbles/v2/...` (e.g., `charm.land/bubbles/v2/viewport`)
  - `charm.land/lipgloss/v2`
  Mixing v1 and v2 imports will cause compile/type errors.
- **Git via Shell**: All git operations must run through `os/exec.Command("git", ...)` with `cmd.Dir` set to the repo root. Do not use `go-git`.
- **Git Diff Flags**: Always include `--no-ext-diff --color=never` on git commands to ensure predictable plain-text outputs.
- **Untracked Files**: No diff is available from git. Instead, read the file content directly and format using `RenderNewFile()`.
- **Non-blocking Update**: All async work (network, git, disk, AI) must be returned as a `tea.Cmd`. Never block in `Update()`.
- **Terminal Width**: Respect `tea.WindowSizeMsg`. The file list panel is fixed at `35` chars (`fileListWidth`); the diff viewport takes the rest.
- **No Inline Styles**: All lipgloss styles must live in `internal/ui/styles.go`, derived from the `Theme` struct. Never define styles inline.
- **Chroma Backgrounds**: When highlighting, apply foreground colors token-by-token but preserve the line's diff background. Chroma must not override background.

## Commands

```bash
make test              # Run all unit tests
golangci-lint run      # Run linter
make dev               # Run locally (mise x go -- go run .)
go run . -s            # Staged only
go run . -r main       # Compare against a ref
go run . log           # Commit browser
```

## Architecture

- **Entrypoint**: `main.go` → `cmd/root.go` (Cobra CLI wrapper)
- **State/Config**: Config loader/saver in `internal/config/config.go` (`~/.config/differ/config.json`)
- **Theme**: Defined in `internal/theme/theme.go`. Bridged to lipgloss in `internal/ui/styles.go`.
- **UI Logic**: Bubble Tea views in `internal/ui/`:
  - `model.go`: Main viewer with file list, diff viewer, and branch picker.
  - `log.go`: Commit log browser.
  - `diff.go`: Parses and renders diffs.
  - `highlight.go`: Chroma syntax highlighter.
