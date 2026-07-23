# Phase 1: Daily-Driver Core + Foundation — Design

## Context

GitSketch is a terminal Git branch/merge visualizer (Go + Bubbletea v2/Lipgloss v2). It currently
supports read/navigate operations on commit history: browsing the DAG, viewing fullscreen diffs,
regex-filtering commits, creating/deleting branches, and checking out commits. It has no support
for the working tree (staging, committing, stashing) — every such action currently requires
dropping to a separate `git` CLI session.

This is Phase 1 of a 3-phase roadmap to grow GitSketch into a full daily-driver Git tool (see
conversation history for Phases 2 and 3, which remain roadmap-level and are not designed here).

Existing architecture this design builds on:
- `internal/git/` — thin wrappers around `exec.Command("git", ...)`, returning typed
  results/errors (`commands.go`)
- `internal/graph/` — topological DAG layout/rendering (`renderer.go`)
- `internal/tui/model.go` — single Bubbletea `Model` with mode flags (`branchMode`, `searchMode`,
  `showDiff`, `confirmCheckout`) dispatched through one `Update()`, and async work via
  `tea.Cmd` → typed `*Msg` (e.g. `checkoutDoneMsg`, `filesLoadedMsg`)
- `internal/tui/keys.go` — key bindings as hardcoded string constants; `HelpText()` returns one
  static help-bar string
- `internal/tui/styles.go` — hardcoded Lipgloss color palette

## Goals (in scope for Phase 1)

1. **Staging workflow**: view working-tree status, stage/unstage whole files and individual
   hunks, write commits.
2. **Status bar**: always-visible branch name, ahead/behind counts, dirty-state summary.
3. **Keymap/config system**: user-overridable key bindings and a context-aware help overlay,
   replacing the hardcoded constants in `keys.go`.
4. **Theme data-plumbing**: colors become data loaded at startup rather than hardcoded in
   `styles.go` (no theme *switcher* UI yet — that's Phase 3).

## Non-goals (explicitly deferred)

- Interactive rebase, cherry-pick, revert, blame, conflict-resolution UI, mouse support → Phase 2
- Multi-repo/workspace support, 100k+ commit performance validation, first-run onboarding, theme
  presets, additional package-manager distribution → Phase 3
- Line-level (sub-hunk) staging — hunk-level is the Phase 1 granularity; line-level is a
  candidate backlog item, not required for the daily-driver bar

## Architecture

### New files

```
internal/git/status.go     → parse `git status --porcelain=v2`, ahead/behind counts
internal/git/staging.go    → stage/unstage whole files (`git add`/`git reset`) and hunks
                              (`git apply --cached [-R]`)
internal/git/commit.go     → `git commit -m`, `git commit --amend`
internal/git/hunk.go       → pure functions: parse unified diff text into `[]Hunk`, and
                              serialize a single Hunk back into an applicable patch
internal/config/config.go  → load/parse config file (keymap + theme), merge over built-in
                              defaults, resolve platform config path
internal/tui/statusbar.go  → render the status bar component
```

`keys.go` is refactored from constants into a `KeyMap` struct populated by `internal/config` at
startup (falling back to today's bindings as defaults, so behavior is unchanged for anyone
without a config file).

### Model changes (`internal/tui/model.go`)

- A synthetic working-tree row (`isWorkingTree bool` on the row type) is pinned above `HEAD` in
  `filteredCommits`/`graphRows`, carrying a live dirty-state badge (e.g. `3⊙ 2○`).
- New state: `stagedFiles`/`unstagedFiles []git.FileChange`, `stagingMode bool`, `hunks
  []git.Hunk`, `hunkCursor int`, `commitInputMode bool`, `commitMessage string`.
- New state: `statusBar` data (branch, ahead, behind, dirty counts) refreshed alongside commit
  history.
- `keyMap KeyMap` field threaded through `Update()` dispatch (replacing direct constant
  comparisons) and into help-overlay rendering.

### Data flow

Follows the existing `tea.Cmd` → typed `Msg` → `Update()` pattern used by
`checkoutDoneMsg`/`filesLoadedMsg` — no new async pattern is introduced:

1. Startup: load config (keymap + theme) before `Init()`; parse commit history as today; kick off
   a status refresh (`git status` + ahead/behind) as a parallel `tea.Cmd`.
2. Any mutating action (stage, unstage, commit, discard) fires a command, returns a result `Msg`,
   and triggers the same status-refresh cascade — reusing the cursor/scroll-stability logic
   already added for post-checkout refreshes (bb7582d), not reinventing it.
3. Selecting the working-tree row loads staged/unstaged file lists (from status). Selecting a
   file loads its diff (`git diff -- file` or `git diff --cached -- file`) and parses it into
   hunks via `internal/git/hunk.go`.
4. Space on a hunk builds a hunk-only patch and runs `git apply --cached` (stage) or `git apply
   --cached -R` (unstage), then refreshes.
5. `a` on a file stages/unstages the whole file (`git add` / `git reset`), then refreshes.
6. `c` while the working-tree row is focused opens `commitInputMode`; Enter submits via `git
   commit -m`, then refreshes and exits input mode.

### Error handling

- All new git operations return `(Result, error)` — non-zero exit surfaces through the existing
  `m.notification`/`notifyStyle` banner. No new error-UI pattern.
- Patch-apply failures (stale hunk, whitespace mismatch) show a clear inline message; state is
  left unchanged — no partial/silent apply.
- Malformed or unreadable config file: fall back to built-in defaults and show a one-time
  startup warning banner. Never fatal.

### Testing

- `internal/git`: temp-repo integration tests for status parsing, whole-file staging, hunk
  staging, and commit — following the existing `git_test.go` pattern of real temp repos.
- `internal/git/hunk.go`: table-driven pure unit tests for diff-parsing and patch
  reconstruction (no git dependency required).
- `internal/config`: unit tests for parsing, default-merge behavior, and malformed-file
  fallback.
- `internal/tui`: `model_test.go` additions covering the new `Update()` branches (staging-mode
  transitions, commit-input flow), following existing model test patterns.

## UI/UX details

- **Status bar**: single line, e.g. `main ↑2 ↓0  •  3 staged, 2 unstaged`, always visible.
- **Help overlay**: `?` opens a scrollable, context-aware keybinding list generated from the
  live `KeyMap` (only bindings relevant to the current mode are shown), replacing the single
  static `HelpText()` string.
- **Working-tree row**: rendered distinctly in the graph pane (distinguishing glyph/style) with
  a live dirty-count badge, always pinned above `HEAD`.

## Suggested workstream split (for parallel implementation)

The new surface area splits into mostly file-disjoint workstreams, suited to parallel agent
work followed by one integration pass:

- **A — Config & Keymap**: `internal/config/`, `keys.go` refactor, help overlay. Touches
  `tui/keys.go`, adds `internal/config/`.
- **B — Git staging backend**: `status.go`, `staging.go`, `commit.go`, `hunk.go` in
  `internal/git/`. Independently unit/integration-testable without any TUI wiring.
- **C — Status bar component**: `internal/tui/statusbar.go` + ahead/behind computation. Small,
  loosely depends on A's rendering conventions (styles/theme data).
- **D — TUI integration**: wires B into `Model`/`Update()`, working-tree row rendering, commit
  input mode, staging-mode diff viewer. Depends on A, B, and C being far enough along; done last
  as an integration pass.

## Open questions / risks

- `git apply --cached` hunk staging needs correct hunk-header recalculation when a file already
  has a mix of staged and unstaged hunks — worth validating behavior against how lazygit/gitui
  handle the same case before finalizing `hunk.go`'s patch builder.
- Config file path resolution must be cross-platform (Linux/XDG, macOS, Windows/AppData) — needs
  a small dedicated path-resolution helper in `internal/config`.
