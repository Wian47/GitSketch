# Roadmap

This tracks the path from `v0.3.0` (Daily-Driver Core — staging, commits, config,
status bar, help overlay) to `v0.4.0`, GitSketch's "History Power-Features"
milestone. Each `v0.3.x` release is a small, independently shippable increment;
`v0.4.0` marks the point where that whole feature set is stable and documented.

Nothing here is a promise of dates — it's an ordering of what's next and why,
so priorities are visible and easy to reshuffle.

## Status

- **Shipped:** `v0.3.0` — working-tree row, file/hunk stage/unstage/discard,
  commit input, fullscreen hunk-staging diff viewer, status bar, `?` help
  overlay, configurable keybindings/theme via TOML.
- **In progress:** none yet — this file defines what's next.

---

## v0.3.1 — Stabilization & Hardening

No new features. Closes out debt flagged in the `v0.3.0` final review before
building further on top of it.

- [ ] Rewrite `README.md` to cover the `v0.3.0` feature set (staging, config,
      keymaps, themes, status bar, help overlay) — it currently only
      describes the pre-staging graph viewer.
- [ ] Add `CHANGELOG.md`, seeded from the `v0.1.0`–`v0.3.0` GitHub release notes.
- [ ] Fix: status bar overflow/wrap on narrow terminals (`internal/tui/statusbar.go`).
- [ ] Fix: clamp pane height instead of going negative on sub-5-row terminals
      (`internal/tui/model.go`).
- [ ] Harden: add a `--` separator before user-supplied branch names in
      `CreateBranch`/`DeleteBranch` (`internal/git/commands.go`) so a name
      like `-d` can't be arg-interpreted by `git branch`.
- [ ] Add boundary-case tests for `TestToggleSelectedHunkReturnsCommand`
      (empty hunk list, cursor out of range).

## v0.3.2 — Staging & Commit UX Completion

Rounds out the staging workflow shipped in `v0.3.0` rather than opening new
surface area — lowest-risk slice of what's left.

- [ ] Line-level (sub-hunk) staging: stage/unstage individual `+`/`-` lines
      within a hunk, not just the whole hunk. Extends `internal/git/hunk.go`'s
      patch builder to emit a single-line patch.
- [ ] `git commit --amend` support, reachable from the working-tree row's
      commit flow.
- [ ] Empty-commit guard + explicit message when `git commit` would be a
      no-op (staged changes race with an external `git reset`, etc.).

## v0.3.3 — Mouse Support

First genuinely new Phase 2 capability. Self-contained (input handling only,
no new git operations), good warm-up before the history-rewriting releases.

- [ ] Click a commit row to select it; click a working-tree file to select it.
- [ ] Scroll wheel support in the graph pane and both diff viewers.
- [ ] Click a branch-lane glyph to filter the graph to that lane.

## v0.3.4 — Blame View

Read-only, so it can't corrupt repo state — a safe first foray into
per-commit history features before mutating ones (revert, cherry-pick,
rebase).

- [ ] `git blame` integration: per-line author/commit/date gutter, toggled
      from the file list or diff view.
- [ ] Jump from a blame line straight to that commit in the graph.

## v0.3.5 — Revert & Cherry-Pick

First history-*mutating* operations. Ships with the minimum conflict
handling needed to be safe, not the full resolution UI — that's `v0.3.6`,
reused across rebase too instead of building it twice.

- [ ] Cherry-pick the selected commit onto the current branch.
- [ ] Revert the selected commit.
- [ ] On conflict: stop, list conflicted files, and offer **abort** or
      **exit to shell/$EDITOR to resolve manually, then continue**. No
      in-app conflict editor yet.

## v0.3.6 — Interactive Rebase + Conflict Resolution UI

The highest-complexity, highest-risk release in this line — history
rewriting plus the first in-app conflict resolution UI, built once and
reused by rebase, cherry-pick, and revert.

- [ ] Interactive rebase onto a selected commit: reorder, squash, drop,
      reword via the TUI (mirrors `git rebase -i`'s todo list).
- [ ] In-app conflict resolution: per-hunk choose ours/theirs/manual,
      replacing the v0.3.5 "exit to shell" fallback for cherry-pick/revert
      too.

---

## v0.4.0 — History Power-Features

Not new feature work — a stabilization pass across everything in
`v0.3.2`–`v0.3.6`, plus the release housekeeping a minor version deserves.

- [ ] Whole-branch review across the full `v0.3.1..v0.3.6` diff for
      cross-cutting issues no single patch release would catch alone.
- [ ] Update `README.md`'s feature list and keybinding table for the full
      Phase 2 surface.
- [ ] Confirm mouse support, blame, cherry-pick/revert, and rebase all
      respect the configurable keymap/theme system from `v0.3.0` — nothing
      hardcoded that should be user-overridable.

---

## Beyond v0.4.0 (Phase 3 — not yet scoped into releases)

Noted for context, not committed to a release cadence yet:

- Multi-repo / workspace support
- Performance validation and optimization for 100k+ commit histories
- First-run onboarding
- Theme presets (a picker UI on top of the `v0.3.0` theme system)
- Additional package-manager distribution (Homebrew, Scoop, etc.)
