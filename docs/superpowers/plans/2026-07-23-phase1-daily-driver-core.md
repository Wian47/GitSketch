# Phase 1: Daily-Driver Core + Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn GitSketch from a read-only history viewer into a daily-driver Git tool: stage/unstage files and hunks, commit, see live repo status, and give users a way to customize keybindings/theme — without breaking any existing feature.

**Architecture:** Extend the existing `internal/git` command-wrapper layer with staging/status/commit operations; add a new `internal/config` package for user-overridable keymap/theme, loaded once at startup; extend the existing single-`Model` Bubbletea pattern (mode flags + `tea.Cmd`/`Msg` async refresh) with a working-tree pseudo-row, staging mode, and a help overlay — reusing the file pane / diff viewer / notification banner / confirm-dialog machinery that already exists rather than building new UI primitives.

**Tech Stack:** Go 1.26, `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`, `github.com/BurntSushi/toml` (new dependency, for the config file).

## Global Constraints

- Go module: `github.com/Wian47/GitSketch`, Go 1.26.5 (from `go.mod`) — match existing style.
- All git operations shell out via `os/exec`, following the exact pattern in `internal/git/commands.go` and `internal/git/parser.go`: `exec.Command`, `CombinedOutput()`/separate `Stdout`/`Stderr` buffers, trimmed error messages, typed `Result` structs where an operation can fail with a user-facing message (see `CheckoutResult`).
- All new async TUI work uses the existing `tea.Cmd` → typed `*Msg` → `Update()` case pattern (see `checkoutDoneMsg`, `filesLoadedMsg`). No new async pattern.
- All new git-layer tests needing a real repo use the existing helpers in `internal/git/git_test.go` (`initTestRepo`, `runGit`, `writeAndCommit`) — do not duplicate them, they're already in-package and reusable from any new `_test.go` file in `internal/git`.
- No placeholders, no TODOs, no stubbed error handling — every task ships a working, tested slice.
- Do not modify existing exported names/signatures in `internal/git`, `internal/graph`, or `internal/tui` unless a task explicitly says to (breaking existing tests is a hard failure).

---

### Task 1: Diff hunk parser and patch builder

**Files:**
- Create: `internal/git/hunk.go`
- Test: `internal/git/hunk_test.go`

**Interfaces:**
- Produces: `type Hunk struct { Header string; Lines []string }`, `func ParseHunks(diff string) (fileHeader string, hunks []Hunk)`, `func (h Hunk) Patch(fileHeader string) string`

This is pure text processing (no git calls), used by Task 3 (staging) and Task 13 (hunk-level staging UI). A diff for one file looks like:

```
diff --git a/foo.txt b/foo.txt
index 83db48f..bf269ba 100644
--- a/foo.txt
+++ b/foo.txt
@@ -1,3 +1,4 @@
 line1
+new line
 line2
 line3
```

Everything before the first `@@ ` line is the file header; each `@@ ` line starts a new hunk that runs until the next `@@ ` line or end of input.

- [ ] **Step 1: Write the failing tests**

```go
// internal/git/hunk_test.go
package git

import "testing"

const sampleDiff = `diff --git a/foo.txt b/foo.txt
index 83db48f..bf269ba 100644
--- a/foo.txt
+++ b/foo.txt
@@ -1,3 +1,4 @@
 line1
+new line
 line2
 line3
@@ -10,2 +11,3 @@
 old context
+another add
 more context
`

func TestParseHunksSplitsFileHeaderAndHunks(t *testing.T) {
	header, hunks := ParseHunks(sampleDiff)

	wantHeader := "diff --git a/foo.txt b/foo.txt\nindex 83db48f..bf269ba 100644\n--- a/foo.txt\n+++ b/foo.txt"
	if header != wantHeader {
		t.Fatalf("header = %q, want %q", header, wantHeader)
	}
	if len(hunks) != 2 {
		t.Fatalf("expected 2 hunks, got %d", len(hunks))
	}
	if hunks[0].Header != "@@ -1,3 +1,4 @@" {
		t.Fatalf("hunks[0].Header = %q", hunks[0].Header)
	}
	wantFirstLines := []string{"@@ -1,3 +1,4 @@", " line1", "+new line", " line2", " line3"}
	if len(hunks[0].Lines) != len(wantFirstLines) {
		t.Fatalf("hunks[0].Lines = %v, want %v", hunks[0].Lines, wantFirstLines)
	}
	for i, l := range wantFirstLines {
		if hunks[0].Lines[i] != l {
			t.Fatalf("hunks[0].Lines[%d] = %q, want %q", i, hunks[0].Lines[i], l)
		}
	}
	if hunks[1].Header != "@@ -10,2 +11,3 @@" {
		t.Fatalf("hunks[1].Header = %q", hunks[1].Header)
	}
}

func TestParseHunksEmptyDiff(t *testing.T) {
	header, hunks := ParseHunks("")
	if header != "" || hunks != nil {
		t.Fatalf("expected empty header and nil hunks for empty diff, got header=%q hunks=%v", header, hunks)
	}
}

func TestParseHunksNoHunksJustHeader(t *testing.T) {
	// e.g. a mode-only change or empty file add with no content hunks.
	diff := "diff --git a/empty.txt b/empty.txt\nnew file mode 100644\n"
	header, hunks := ParseHunks(diff)
	want := "diff --git a/empty.txt b/empty.txt\nnew file mode 100644"
	if header != want {
		t.Fatalf("header = %q, want %q", header, want)
	}
	if len(hunks) != 0 {
		t.Fatalf("expected 0 hunks, got %d", len(hunks))
	}
}

func TestHunkPatchReconstructsApplicablePatch(t *testing.T) {
	header, hunks := ParseHunks(sampleDiff)
	patch := hunks[0].Patch(header)

	want := "diff --git a/foo.txt b/foo.txt\n" +
		"index 83db48f..bf269ba 100644\n" +
		"--- a/foo.txt\n" +
		"+++ b/foo.txt\n" +
		"@@ -1,3 +1,4 @@\n" +
		" line1\n" +
		"+new line\n" +
		" line2\n" +
		" line3\n"
	if patch != want {
		t.Fatalf("Patch() = %q, want %q", patch, want)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/git/... -run TestParseHunks -v`
Expected: FAIL with `undefined: ParseHunks` (compile error, since `hunk.go` doesn't exist yet)

- [ ] **Step 3: Implement**

```go
// internal/git/hunk.go
package git

import "strings"

// Hunk represents a single @@ ... @@ section of a unified diff for one file.
type Hunk struct {
	Header string   // the "@@ -a,b +c,d @@ ..." line
	Lines  []string // full hunk lines, including the header line itself
}

// ParseHunks splits a single-file unified diff (as produced by `git diff` or
// `git diff --cached` for one path) into its file header — everything before
// the first "@@ " line, e.g. the "diff --git"/"index"/"---"/"+++" lines —
// and its individual hunks. Returns ("", nil) for an empty diff.
func ParseHunks(diff string) (fileHeader string, hunks []Hunk) {
	diff = strings.TrimRight(diff, "\n")
	if diff == "" {
		return "", nil
	}

	lines := strings.Split(diff, "\n")

	i := 0
	var headerLines []string
	for i < len(lines) && !strings.HasPrefix(lines[i], "@@ ") {
		headerLines = append(headerLines, lines[i])
		i++
	}
	fileHeader = strings.Join(headerLines, "\n")

	for i < len(lines) {
		header := lines[i]
		hunkLines := []string{header}
		i++
		for i < len(lines) && !strings.HasPrefix(lines[i], "@@ ") {
			hunkLines = append(hunkLines, lines[i])
			i++
		}
		hunks = append(hunks, Hunk{Header: header, Lines: hunkLines})
	}
	return fileHeader, hunks
}

// Patch reconstructs a standalone, applicable patch containing only this
// hunk, prefixed with the file header. The hunk's line ranges are copied
// verbatim from the source diff, so this is safe for whole-hunk staging
// without any header recalculation. Suitable for piping to
// `git apply --cached` (see StageHunk/UnstageHunk in staging.go).
func (h Hunk) Patch(fileHeader string) string {
	var sb strings.Builder
	sb.WriteString(fileHeader)
	sb.WriteString("\n")
	for _, l := range h.Lines {
		sb.WriteString(l)
		sb.WriteString("\n")
	}
	return sb.String()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/git/... -run TestParseHunks -v && go test ./internal/git/... -run TestHunkPatch -v`
Expected: PASS for all four tests

- [ ] **Step 5: Commit**

```bash
git add internal/git/hunk.go internal/git/hunk_test.go
git commit -m "feat: add diff hunk parser and patch builder"
```

---

### Task 2: Working-tree status (staged/unstaged files, branch, ahead/behind)

**Files:**
- Create: `internal/git/status.go`
- Test: `internal/git/status_test.go`

**Interfaces:**
- Consumes: nothing new (uses only `os/exec`, same as `commands.go`)
- Produces: `type StatusEntry struct { Status string; Path string }`, `type Status struct { Branch string; Ahead int; Behind int; Staged []StatusEntry; Unstaged []StatusEntry }`, `func GetStatus() (Status, error)`

Uses `git status --porcelain=v2 --branch -z`: NUL-delimited output avoids all quoting/escaping ambiguity around filenames, and header lines (`# branch.head ...`, `# branch.ab ...`) are also NUL-terminated in `-z` mode, so the whole output can be split uniformly on `\x00`.

- [ ] **Step 1: Write the failing tests**

```go
// internal/git/status_test.go
package git

import (
	"os"
	"testing"
)

func TestGetStatusCleanRepo(t *testing.T) {
	initTestRepo(t)
	writeAndCommit(t, "a.txt", "one", "first commit")

	st, err := GetStatus()
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}
	if st.Branch != "main" {
		t.Fatalf("Branch = %q, want %q", st.Branch, "main")
	}
	if len(st.Staged) != 0 || len(st.Unstaged) != 0 {
		t.Fatalf("expected clean status, got staged=%v unstaged=%v", st.Staged, st.Unstaged)
	}
	if st.Ahead != 0 || st.Behind != 0 {
		t.Fatalf("expected 0/0 ahead/behind with no upstream, got %d/%d", st.Ahead, st.Behind)
	}
}

func TestGetStatusUnstagedModification(t *testing.T) {
	initTestRepo(t)
	writeAndCommit(t, "a.txt", "one", "first commit")

	if err := os.WriteFile("a.txt", []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}

	st, err := GetStatus()
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}
	if len(st.Staged) != 0 {
		t.Fatalf("expected no staged entries, got %v", st.Staged)
	}
	if len(st.Unstaged) != 1 || st.Unstaged[0].Path != "a.txt" || st.Unstaged[0].Status != "M" {
		t.Fatalf("expected unstaged M a.txt, got %v", st.Unstaged)
	}
}

func TestGetStatusStagedAddition(t *testing.T) {
	initTestRepo(t)
	writeAndCommit(t, "a.txt", "one", "first commit")

	if err := os.WriteFile("b.txt", []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, "add", "b.txt")

	st, err := GetStatus()
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}
	if len(st.Staged) != 1 || st.Staged[0].Path != "b.txt" || st.Staged[0].Status != "A" {
		t.Fatalf("expected staged A b.txt, got %v", st.Staged)
	}
	if len(st.Unstaged) != 0 {
		t.Fatalf("expected no unstaged entries, got %v", st.Unstaged)
	}
}

func TestGetStatusUntrackedFile(t *testing.T) {
	initTestRepo(t)
	writeAndCommit(t, "a.txt", "one", "first commit")

	if err := os.WriteFile("untracked.txt", []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}

	st, err := GetStatus()
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}
	if len(st.Unstaged) != 1 || st.Unstaged[0].Path != "untracked.txt" || st.Unstaged[0].Status != "??" {
		t.Fatalf("expected unstaged ?? untracked.txt, got %v", st.Unstaged)
	}
}

func TestGetStatusMixedStagedAndUnstagedSameFile(t *testing.T) {
	initTestRepo(t)
	writeAndCommit(t, "a.txt", "one\ntwo\n", "first commit")

	if err := os.WriteFile("a.txt", []byte("one-staged\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, "add", "a.txt")
	if err := os.WriteFile("a.txt", []byte("one-staged\ntwo-unstaged\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	st, err := GetStatus()
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}
	if len(st.Staged) != 1 || st.Staged[0].Status != "M" {
		t.Fatalf("expected 1 staged M entry, got %v", st.Staged)
	}
	if len(st.Unstaged) != 1 || st.Unstaged[0].Status != "M" {
		t.Fatalf("expected 1 unstaged M entry, got %v", st.Unstaged)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/git/... -run TestGetStatus -v`
Expected: FAIL with `undefined: GetStatus` (compile error)

- [ ] **Step 3: Implement**

```go
// internal/git/status.go
package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// StatusEntry represents one file's status letter (M, A, D, R, C, or "??"
// for untracked) and its path.
type StatusEntry struct {
	Status string
	Path   string
}

// Status is a snapshot of the working tree: current branch, how far ahead/
// behind its upstream it is (0/0 if there's no upstream), and the staged vs
// unstaged file lists.
type Status struct {
	Branch   string
	Ahead    int
	Behind   int
	Staged   []StatusEntry
	Unstaged []StatusEntry
}

// GetStatus runs `git status --porcelain=v2 --branch -z` and parses the
// working tree state. The -z form NUL-delimits every record (including the
// "# branch.*" header lines), which sidesteps all quoting/escaping rules
// that apply to filenames in the non-z porcelain format.
func GetStatus() (Status, error) {
	cmd := exec.Command("git", "status", "--porcelain=v2", "--branch", "-z")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return Status{}, fmt.Errorf("git status: %w: %s", err, stderr.String())
	}

	var st Status
	records := strings.Split(stdout.String(), "\x00")

	for i := 0; i < len(records); i++ {
		rec := records[i]
		switch {
		case rec == "":
			continue
		case strings.HasPrefix(rec, "# branch.head "):
			st.Branch = strings.TrimPrefix(rec, "# branch.head ")
		case strings.HasPrefix(rec, "# branch.ab "):
			ab := strings.TrimPrefix(rec, "# branch.ab ")
			parts := strings.Fields(ab) // e.g. ["+2", "-1"]
			if len(parts) == 2 {
				st.Ahead, _ = strconv.Atoi(strings.TrimPrefix(parts[0], "+"))
				st.Behind, _ = strconv.Atoi(strings.TrimPrefix(parts[1], "-"))
			}
		case strings.HasPrefix(rec, "1 "):
			// "1 XY sub mH mI mW hH hI path"
			fields := strings.SplitN(rec, " ", 9)
			if len(fields) == 9 {
				addStatusEntry(&st, fields[1], fields[8])
			}
		case strings.HasPrefix(rec, "2 "):
			// "2 XY sub mH mI mW hH hI Xscore path" — followed by a second
			// NUL-terminated record holding the original path, which we
			// don't need for status display, so just skip past it.
			fields := strings.SplitN(rec, " ", 10)
			if len(fields) == 10 {
				addStatusEntry(&st, fields[1], fields[9])
				i++
			}
		case strings.HasPrefix(rec, "? "):
			st.Unstaged = append(st.Unstaged, StatusEntry{
				Status: "??",
				Path:   strings.TrimPrefix(rec, "? "),
			})
		}
	}

	return st, nil
}

// addStatusEntry splits a porcelain v2 XY status pair into a staged entry
// (from X) and/or an unstaged entry (from Y). '.' means "no change in that
// slot" and is skipped.
func addStatusEntry(st *Status, xy, path string) {
	if len(xy) != 2 {
		return
	}
	x, y := xy[0], xy[1]
	if x != '.' {
		st.Staged = append(st.Staged, StatusEntry{Status: string(x), Path: path})
	}
	if y != '.' {
		st.Unstaged = append(st.Unstaged, StatusEntry{Status: string(y), Path: path})
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/git/... -run TestGetStatus -v`
Expected: PASS for all five tests

- [ ] **Step 5: Commit**

```bash
git add internal/git/status.go internal/git/status_test.go
git commit -m "feat: add working-tree status parsing"
```

---

### Task 3: Staging operations (whole-file and hunk-level)

**Files:**
- Create: `internal/git/staging.go`
- Test: `internal/git/staging_test.go`

**Interfaces:**
- Consumes: `Hunk`, `Hunk.Patch(fileHeader string) string` (Task 1)
- Produces: `func StageFile(path string) error`, `func UnstageFile(path string) error`, `func DiscardFile(path string) error`, `func StageHunk(fileHeader string, hunk Hunk) error`, `func UnstageHunk(fileHeader string, hunk Hunk) error`

`DiscardFile` uses `git checkout -- <path>` and only works for tracked, modified files — discarding an untracked file is a different (more destructive) operation and is out of scope for Phase 1.

- [ ] **Step 1: Write the failing tests**

```go
// internal/git/staging_test.go
package git

import (
	"os"
	"strings"
	"testing"
)

func TestStageAndUnstageFile(t *testing.T) {
	initTestRepo(t)
	writeAndCommit(t, "a.txt", "one", "first commit")
	if err := os.WriteFile("a.txt", []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := StageFile("a.txt"); err != nil {
		t.Fatalf("StageFile() error = %v", err)
	}
	st, err := GetStatus()
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Staged) != 1 || st.Staged[0].Path != "a.txt" {
		t.Fatalf("expected a.txt staged, got %v", st.Staged)
	}

	if err := UnstageFile("a.txt"); err != nil {
		t.Fatalf("UnstageFile() error = %v", err)
	}
	st, err = GetStatus()
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Staged) != 0 || len(st.Unstaged) != 1 {
		t.Fatalf("expected a.txt unstaged, got staged=%v unstaged=%v", st.Staged, st.Unstaged)
	}
}

func TestDiscardFile(t *testing.T) {
	initTestRepo(t)
	writeAndCommit(t, "a.txt", "original", "first commit")
	if err := os.WriteFile("a.txt", []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := DiscardFile("a.txt"); err != nil {
		t.Fatalf("DiscardFile() error = %v", err)
	}

	content, err := os.ReadFile("a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "original" {
		t.Fatalf("expected file reverted to %q, got %q", "original", string(content))
	}
}

func TestStageFileInvalidPath(t *testing.T) {
	initTestRepo(t)
	writeAndCommit(t, "a.txt", "one", "first commit")

	err := StageFile("does-not-exist.txt")
	if err == nil {
		t.Fatal("expected error staging a nonexistent path")
	}
}

func TestStageAndUnstageHunk(t *testing.T) {
	initTestRepo(t)
	writeAndCommit(t, "a.txt", "line1\nline2\nline3\n", "first commit")
	if err := os.WriteFile("a.txt", []byte("line1\nline2-changed\nline3\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	diff := runGit(t, "diff", "--", "a.txt")
	header, hunks := ParseHunks(diff)
	if len(hunks) != 1 {
		t.Fatalf("expected 1 hunk in fixture diff, got %d", len(hunks))
	}

	if err := StageHunk(header, hunks[0]); err != nil {
		t.Fatalf("StageHunk() error = %v: diff was:\n%s", err, diff)
	}

	st, err := GetStatus()
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Staged) != 1 || st.Staged[0].Path != "a.txt" {
		t.Fatalf("expected a.txt staged after StageHunk, got %v", st.Staged)
	}
	if len(st.Unstaged) != 0 {
		t.Fatalf("expected no unstaged changes after staging the only hunk, got %v", st.Unstaged)
	}

	if err := UnstageHunk(header, hunks[0]); err != nil {
		t.Fatalf("UnstageHunk() error = %v", err)
	}
	st, err = GetStatus()
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Staged) != 0 {
		t.Fatalf("expected no staged changes after UnstageHunk, got %v", st.Staged)
	}
	if len(st.Unstaged) != 1 || !strings.Contains(st.Unstaged[0].Status, "M") {
		t.Fatalf("expected a.txt back to unstaged M, got %v", st.Unstaged)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/git/... -run 'TestStage|TestDiscard' -v`
Expected: FAIL with `undefined: StageFile` etc. (compile error)

- [ ] **Step 3: Implement**

```go
// internal/git/staging.go
package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// StageFile stages the given path in full (`git add`).
func StageFile(path string) error {
	return runGitOp("add", "--", path)
}

// UnstageFile removes the given path from the index without touching the
// working tree (`git restore --staged`).
func UnstageFile(path string) error {
	return runGitOp("restore", "--staged", "--", path)
}

// DiscardFile reverts a tracked file's working-tree contents to match the
// index (`git checkout --`). It only applies to already-tracked files —
// discarding an untracked file is a separate, more destructive operation
// and is not supported here.
func DiscardFile(path string) error {
	return runGitOp("checkout", "--", path)
}

// StageHunk applies a single hunk's patch to the index only.
func StageHunk(fileHeader string, hunk Hunk) error {
	return applyPatch(hunk.Patch(fileHeader), false)
}

// UnstageHunk reverses a single hunk's patch in the index only, leaving the
// working tree untouched.
func UnstageHunk(fileHeader string, hunk Hunk) error {
	return applyPatch(hunk.Patch(fileHeader), true)
}

func runGitOp(args ...string) error {
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(output)))
	}
	return nil
}

func applyPatch(patch string, reverse bool) error {
	args := []string{"apply", "--cached"}
	if reverse {
		args = append(args, "-R")
	}
	cmd := exec.Command("git", args...)
	cmd.Stdin = strings.NewReader(patch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(output)))
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/git/... -run 'TestStage|TestDiscard' -v`
Expected: PASS for all five tests

- [ ] **Step 5: Commit**

```bash
git add internal/git/staging.go internal/git/staging_test.go
git commit -m "feat: add whole-file and hunk-level staging operations"
```

---

### Task 4: Commit operation

**Files:**
- Create: `internal/git/commit.go`
- Test: `internal/git/commit_test.go`

**Interfaces:**
- Produces: `type CommitResult struct { Success bool; Message string }`, `func CreateCommit(message string) CommitResult`

Mirrors `Checkout`'s existing shape in `commands.go` exactly (same `CombinedOutput`/trim/`Success` pattern) so the TUI layer (Task 12) can reuse the same `checkoutDoneMsg`-style handling. Named `CreateCommit` (not `Commit`) because `internal/git/parser.go` already defines `type Commit struct` (the DAG commit type used throughout `internal/tui` and `internal/graph`) — Go doesn't allow a function and a type to share a name in the same package, and matching the existing `CreateBranch`/`DeleteBranch` naming convention was the natural fix.

- [ ] **Step 1: Write the failing tests**

```go
// internal/git/commit_test.go
package git

import (
	"os"
	"strings"
	"testing"
)

func TestCommitSucceeds(t *testing.T) {
	initTestRepo(t)
	writeAndCommit(t, "a.txt", "one", "first commit")
	if err := os.WriteFile("a.txt", []byte("two"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := StageFile("a.txt"); err != nil {
		t.Fatal(err)
	}

	result := CreateCommit("second commit")
	if !result.Success {
		t.Fatalf("expected commit to succeed, got message: %q", result.Message)
	}

	log := runGit(t, "log", "-1", "--format=%s")
	if strings.TrimSpace(log) != "second commit" {
		t.Fatalf("expected HEAD subject %q, got %q", "second commit", strings.TrimSpace(log))
	}
}

func TestCommitNothingStagedFails(t *testing.T) {
	initTestRepo(t)
	writeAndCommit(t, "a.txt", "one", "first commit")

	result := CreateCommit("empty commit attempt")
	if result.Success {
		t.Fatal("expected commit with nothing staged to fail")
	}
	if result.Message == "" {
		t.Fatal("expected a non-empty error message")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/git/... -run TestCommit -v`
Expected: FAIL with `undefined: CreateCommit` (compile error)

- [ ] **Step 3: Implement**

```go
// internal/git/commit.go
package git

import (
	"os/exec"
	"strings"
)

// CommitResult holds the outcome of a git commit operation.
type CommitResult struct {
	Success bool
	Message string
}

// CreateCommit runs "git commit -m <message>" against whatever is currently
// staged and returns a CommitResult describing whether it succeeded.
func CreateCommit(message string) CommitResult {
	cmd := exec.Command("git", "commit", "-m", message)
	output, err := cmd.CombinedOutput()
	msg := strings.TrimSpace(string(output))

	if err != nil {
		if msg == "" {
			msg = err.Error()
		}
		return CommitResult{Success: false, Message: msg}
	}

	return CommitResult{Success: true, Message: msg}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/git/... -run TestCommit -v`
Expected: PASS for both tests

- [ ] **Step 5: Commit**

```bash
git add internal/git/commit.go internal/git/commit_test.go
git commit -m "feat: add commit operation"
```

---

### Task 5: Config package (keymap defaults + TOML loading)

**Files:**
- Create: `internal/config/config.go`
- Test: `internal/config/config_test.go`

**Interfaces:**
- Produces: `type KeyMap struct { Up, Down, Top, Bottom, PageUp, PageDown, Enter, Checkout, Filter, Branch, Quit string }`, `type Config struct { KeyMap KeyMap }`, `func DefaultKeyMap() KeyMap`, `func Default() Config`, `func Path() (string, error)`, `func Load() (cfg Config, warning string)`

Adds a new dependency, `github.com/BurntSushi/toml`, for parsing the user config file. `Path()` uses `os.UserConfigDir()` (stdlib) so resolution is correct on Linux/XDG, macOS, and Windows without any custom platform-detection code. `Load()` decodes the TOML file directly onto a struct pre-populated with `Default()` — `BurntSushi/toml` only overwrites fields actually present in the file, so this gives "merge over defaults" for free. `Load()` never returns an `error`: a missing file is normal (defaults), a malformed file falls back to defaults plus a warning string for the caller to print.

- [ ] **Step 1: Add the new dependency**

Run: `go get github.com/BurntSushi/toml@latest`
Expected: `go.mod`/`go.sum` updated with a `github.com/BurntSushi/toml` entry

- [ ] **Step 2: Write the failing tests**

```go
// internal/config/config_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultKeyMapMatchesCurrentBindings(t *testing.T) {
	km := DefaultKeyMap()
	if km.Up != "up" || km.Down != "down" || km.Top != "g" || km.Bottom != "G" {
		t.Fatalf("unexpected nav defaults: %+v", km)
	}
	if km.Checkout != "c" || km.Filter != "/" || km.Branch != "b" || km.Quit != "q" {
		t.Fatalf("unexpected action defaults: %+v", km)
	}
}

func TestLoadNoFileReturnsDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfg, warning := Load()
	if warning != "" {
		t.Fatalf("expected no warning when no config file exists, got %q", warning)
	}
	if cfg.KeyMap != DefaultKeyMap() {
		t.Fatalf("expected default keymap, got %+v", cfg.KeyMap)
	}
}

func TestLoadPartialFileMergesOverDefaults(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfgDir := filepath.Join(dir, "gitsketch")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	toml := "[keymap]\nquit = \"x\"\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, warning := Load()
	if warning != "" {
		t.Fatalf("expected no warning for a valid partial file, got %q", warning)
	}
	if cfg.KeyMap.Quit != "x" {
		t.Fatalf("expected overridden Quit=\"x\", got %q", cfg.KeyMap.Quit)
	}
	if cfg.KeyMap.Up != "up" {
		t.Fatalf("expected untouched field Up to keep default %q, got %q", "up", cfg.KeyMap.Up)
	}
}

func TestLoadMalformedFileFallsBackWithWarning(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfgDir := filepath.Join(dir, "gitsketch")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte("not valid toml [["), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, warning := Load()
	if warning == "" {
		t.Fatal("expected a non-empty warning for a malformed config file")
	}
	if cfg.KeyMap != DefaultKeyMap() {
		t.Fatalf("expected defaults on malformed file, got %+v", cfg.KeyMap)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/config/... -v`
Expected: FAIL with build errors (package `internal/config` doesn't exist yet)

- [ ] **Step 4: Implement**

```go
// internal/config/config.go
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// KeyMap holds the user-overridable primary key bindings. Vim-style
// alternates (k/j, Home/End, etc.) always remain available in the TUI
// regardless of these overrides — only the primary binding is configurable.
type KeyMap struct {
	Up       string `toml:"up"`
	Down     string `toml:"down"`
	Top      string `toml:"top"`
	Bottom   string `toml:"bottom"`
	PageUp   string `toml:"page_up"`
	PageDown string `toml:"page_down"`
	Enter    string `toml:"enter"`
	Checkout string `toml:"checkout"`
	Filter   string `toml:"filter"`
	Branch   string `toml:"branch"`
	Quit     string `toml:"quit"`
}

// Config is the full set of user-configurable GitSketch settings.
type Config struct {
	KeyMap KeyMap `toml:"keymap"`
}

// DefaultKeyMap returns the built-in key bindings, matching GitSketch's
// behavior with no config file present.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: "up", Down: "down", Top: "g", Bottom: "G",
		PageUp: "pgup", PageDown: "pgdown", Enter: "enter",
		Checkout: "c", Filter: "/", Branch: "b", Quit: "q",
	}
}

// Default returns the built-in configuration.
func Default() Config {
	return Config{KeyMap: DefaultKeyMap()}
}

// Path returns the resolved path to the user's config file:
// <os.UserConfigDir()>/gitsketch/config.toml.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gitsketch", "config.toml"), nil
}

// Load returns the effective configuration: built-in defaults merged with
// the user's config file, if one exists and parses. Load never fails: a
// missing file silently yields the defaults; a malformed file also yields
// the defaults, plus a non-empty warning describing the problem for the
// caller to surface however it likes (e.g. print to stderr).
func Load() (cfg Config, warning string) {
	cfg = Default()

	path, err := Path()
	if err != nil {
		return cfg, ""
	}

	if _, err := os.Stat(path); err != nil {
		return cfg, ""
	}

	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return Default(), fmt.Sprintf("config: failed to parse %s: %v (using defaults)", path, err)
	}

	return cfg, ""
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/config/... -v`
Expected: PASS for all four tests

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum internal/config/config.go internal/config/config_test.go
git commit -m "feat: add config package with keymap defaults and TOML loading"
```

---

### Task 6: Wire the keymap into the TUI and startup

**Files:**
- Modify: `internal/tui/keys.go`
- Modify: `internal/tui/model.go:379` (the `case "/":` block), `internal/tui/model.go:404` (the `case "b":` block) — search for the literal `"/"` and `"b"` case labels in `handleKey`'s Normal Mode switch
- Modify: `main.go`
- Test: `internal/tui/keys_test.go`

**Interfaces:**
- Consumes: `config.KeyMap` (Task 5), `config.Load()`
- Produces: `func ApplyKeyMap(km config.KeyMap)` in package `tui`; `KeyFilter`, `KeyBranch` new package vars alongside the existing `Key*` vars (now vars instead of consts)

Converts the hardcoded `const` block in `keys.go` to `var`, so `ApplyKeyMap` can override bindings at startup. Existing `switch` statements in `model.go` compare against these identifiers already — Go's `switch` doesn't require constant case values, so this is a safe, non-invasive change. `HelpText()` is untouched here (it's dead code today, unused by any call site — Task 8 replaces it with the dynamic help overlay).

- [ ] **Step 1: Write the failing test**

```go
// internal/tui/keys_test.go
package tui

import (
	"testing"

	"github.com/Wian47/GitSketch/internal/config"
)

func TestApplyKeyMapOverridesBindings(t *testing.T) {
	t.Cleanup(func() { ApplyKeyMap(config.DefaultKeyMap()) })

	ApplyKeyMap(config.KeyMap{Quit: "x", Up: "w"})

	if KeyQ != "x" {
		t.Fatalf("KeyQ = %q, want %q", KeyQ, "x")
	}
	if KeyUp != "w" {
		t.Fatalf("KeyUp = %q, want %q", KeyUp, "w")
	}
	// Fields left empty in the override must keep their previous value.
	if KeyDown != "down" {
		t.Fatalf("KeyDown = %q, want unchanged default %q", KeyDown, "down")
	}
}

func TestApplyKeyMapEmptyFieldsLeaveDefaultsUntouched(t *testing.T) {
	t.Cleanup(func() { ApplyKeyMap(config.DefaultKeyMap()) })

	ApplyKeyMap(config.KeyMap{})

	if KeyUp != "up" || KeyDown != "down" || KeyC != "c" || KeyFilter != "/" || KeyBranch != "b" {
		t.Fatal("expected an all-empty KeyMap to leave every binding at its default")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/... -run TestApplyKeyMap -v`
Expected: FAIL with `undefined: ApplyKeyMap` (compile error)

- [ ] **Step 3: Rewrite `keys.go`**

```go
// internal/tui/keys.go
package tui

import (
	"fmt"

	"github.com/Wian47/GitSketch/internal/config"
)

// Key bindings. These are package-level vars, not consts, so ApplyKeyMap
// can override them from the user's loaded config at startup. Vim-style
// alternates (KeyK, KeyJ, KeyHome, KeyEnd) are always active and are not
// user-configurable.
var (
	KeyUp     = "up"
	KeyDown   = "down"
	KeyK      = "k"
	KeyJ      = "j"
	KeyG      = "g" // jump to top
	KeyShiftG = "G" // jump to bottom
	KeyHome   = "home"
	KeyEnd    = "end"
	KeyPgUp   = "pgup"
	KeyPgDown = "pgdown"
	KeyEnter  = "enter"
	KeyC      = "c" // checkout
	KeyY      = "y" // confirm
	KeyN      = "n" // cancel
	KeyEsc    = "escape"
	KeyQ      = "q"      // quit
	KeyCtrlC  = "ctrl+c" // quit
	KeyFilter = "/"
	KeyBranch = "b"
)

// ApplyKeyMap overrides the package-level key bindings from a loaded
// config.KeyMap. Empty fields are left untouched, so a config file only
// needs to specify the bindings it wants to change.
func ApplyKeyMap(km config.KeyMap) {
	setIfNotEmpty(&KeyUp, km.Up)
	setIfNotEmpty(&KeyDown, km.Down)
	setIfNotEmpty(&KeyG, km.Top)
	setIfNotEmpty(&KeyShiftG, km.Bottom)
	setIfNotEmpty(&KeyPgUp, km.PageUp)
	setIfNotEmpty(&KeyPgDown, km.PageDown)
	setIfNotEmpty(&KeyEnter, km.Enter)
	setIfNotEmpty(&KeyC, km.Checkout)
	setIfNotEmpty(&KeyFilter, km.Filter)
	setIfNotEmpty(&KeyBranch, km.Branch)
	setIfNotEmpty(&KeyQ, km.Quit)
}

func setIfNotEmpty(dst *string, val string) {
	if val != "" {
		*dst = val
	}
}

// HelpText returns the formatted help bar text
func HelpText() string {
	return "  ↑/k Up  ↓/j Down  g/G Top/Bottom  enter Expand  c Checkout  q Quit"
}

// ConfirmText returns the checkout confirmation prompt
func ConfirmText(hash string) string {
	return fmt.Sprintf("  Checkout %s? (y/n)", hash)
}
```

- [ ] **Step 4: Update the two literal key cases in `model.go`**

In `handleKey`'s Normal Mode switch, change:

```go
	case "/":
		m.searchMode = true
		m.searchQuery = ""
		return m, nil

	case "b":
		m.branchMode = true
		m.branchSubMode = ""
		return m, nil
```

to:

```go
	case KeyFilter:
		m.searchMode = true
		m.searchQuery = ""
		return m, nil

	case KeyBranch:
		m.branchMode = true
		m.branchSubMode = ""
		return m, nil
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/tui/... -v`
Expected: PASS — the two new tests, and every pre-existing `internal/tui` test (confirms the const→var change didn't break anything)

- [ ] **Step 6: Wire config loading into `main.go`**

```go
// main.go
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/Wian47/GitSketch/internal/config"
	"github.com/Wian47/GitSketch/internal/git"
	"github.com/Wian47/GitSketch/internal/tui"
)

func main() {
	// Verify we're inside a git repository.
	if !git.IsGitRepo() {
		fmt.Fprintln(os.Stderr, "fatal: not a git repository (or any of the parent directories)")
		os.Exit(1)
	}

	cfg, warning := config.Load()
	if warning != "" {
		fmt.Fprintln(os.Stderr, warning)
	}
	tui.ApplyKeyMap(cfg.KeyMap)

	// Initialize and run the Bubbletea program.
	model := tui.NewModel()
	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running GitSketch: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 7: Build to verify it compiles**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 8: Commit**

```bash
git add internal/tui/keys.go internal/tui/keys_test.go internal/tui/model.go main.go
git commit -m "feat: wire user-configurable keymap into startup"
```

---

### Task 7: Theme data-plumbing

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/tui/styles.go`
- Test: `internal/config/config_test.go`, `internal/tui/styles_test.go`

**Interfaces:**
- Consumes: nothing new
- Produces: `type Theme struct { ... }` and `func DefaultTheme() Theme` in package `config`; `Config.Theme Theme` field; `func ApplyTheme(t config.Theme)` in package `tui`

Moves every hardcoded hex color out of `styles.go`'s `init()` into `config.Theme` data. `init()` keeps calling `ApplyTheme(config.DefaultTheme())` so zero-config behavior is byte-for-byte unchanged; a loaded user config can call `ApplyTheme` again afterward to override. No theme-switcher UI yet — this task is only the plumbing.

- [ ] **Step 1: Write the failing tests**

```go
// internal/config/config_test.go — add to the existing file
func TestDefaultThemeHasAllBranchColors(t *testing.T) {
	theme := DefaultTheme()
	if len(theme.BranchColors) != 8 {
		t.Fatalf("expected 8 branch colors, got %d", len(theme.BranchColors))
	}
	if theme.Hash != "#FFD54F" {
		t.Fatalf("Hash = %q, want %q", theme.Hash, "#FFD54F")
	}
}

func TestLoadMergesPartialTheme(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfgDir := filepath.Join(dir, "gitsketch")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	toml := "[theme]\nhash = \"#FF0000\"\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, warning := Load()
	if warning != "" {
		t.Fatalf("expected no warning, got %q", warning)
	}
	if cfg.Theme.Hash != "#FF0000" {
		t.Fatalf("Theme.Hash = %q, want %q", cfg.Theme.Hash, "#FF0000")
	}
	if cfg.Theme.Dim != DefaultTheme().Dim {
		t.Fatalf("expected untouched Theme.Dim to keep default, got %q", cfg.Theme.Dim)
	}
}
```

```go
// internal/tui/styles_test.go
package tui

import (
	"testing"

	"github.com/Wian47/GitSketch/internal/config"
)

func TestApplyThemeOverridesColors(t *testing.T) {
	t.Cleanup(func() { ApplyTheme(config.DefaultTheme()) })

	custom := config.DefaultTheme()
	custom.Hash = "#123456"
	ApplyTheme(custom)

	got := HashStyle.GetForeground()
	want := lipglossColor(t, "#123456")
	assertSameColor(t, got, want)
}

func TestApplyThemeDefaultMatchesOriginalPalette(t *testing.T) {
	ApplyTheme(config.DefaultTheme())

	got := DimStyle.GetForeground()
	want := lipglossColor(t, "#546E7A")
	assertSameColor(t, got, want)
}
```

Add this small helper to the same file (needed because comparing `color.Color` values requires comparing their `RGBA()` output, not `==`, since `lipgloss.Color` is a string type underneath):

```go
// internal/tui/styles_test.go — add below the tests

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

func lipglossColor(t *testing.T, hex string) color.Color {
	t.Helper()
	return lipgloss.Color(hex)
}

func assertSameColor(t *testing.T, got, want color.Color) {
	t.Helper()
	gr, gg, gb, ga := got.RGBA()
	wr, wg, wb, wa := want.RGBA()
	if gr != wr || gg != wg || gb != wb || ga != wa {
		t.Fatalf("color mismatch: got RGBA(%d,%d,%d,%d), want RGBA(%d,%d,%d,%d)", gr, gg, gb, ga, wr, wg, wb, wa)
	}
}
```

(Combine the two `internal/tui/styles_test.go` snippets above into one file with a single `import` block.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/... ./internal/tui/... -run 'TestDefaultTheme|TestLoadMergesPartialTheme|TestApplyTheme' -v`
Expected: FAIL with `undefined: DefaultTheme` / `undefined: ApplyTheme` (compile errors)

- [ ] **Step 3: Add `Theme` to `internal/config/config.go`**

```go
// Theme holds every color GitSketch renders with, as hex strings. There is
// no theme switcher yet — this only makes the palette data instead of
// hardcoded, so a user's config file can override individual colors.
type Theme struct {
	BranchColors []string `toml:"branch_colors"`

	GraphLine     string `toml:"graph_line"`
	SelectedRowBg string `toml:"selected_row_bg"`

	Hash    string `toml:"hash"`
	Author  string `toml:"author"`
	Date    string `toml:"date"`
	Subject string `toml:"subject"`
	Body    string `toml:"body"`

	BranchRefFg string `toml:"branch_ref_fg"`
	BranchRefBg string `toml:"branch_ref_bg"`
	TagRefFg    string `toml:"tag_ref_fg"`
	TagRefBg    string `toml:"tag_ref_bg"`
	HeadRefFg   string `toml:"head_ref_fg"`
	HeadRefBg   string `toml:"head_ref_bg"`

	PaneBorder string `toml:"pane_border"`
	Title      string `toml:"title"`

	HelpBarFg string `toml:"help_bar_fg"`
	HelpBarBg string `toml:"help_bar_bg"`

	Success string `toml:"success"`
	Error   string `toml:"error"`

	Modified string `toml:"modified"`
	Added    string `toml:"added"`
	Deleted  string `toml:"deleted"`

	Dim  string `toml:"dim"`
	Logo string `toml:"logo"`

	DetailLabel   string `toml:"detail_label"`
	DetailValue   string `toml:"detail_value"`
	SectionHeader string `toml:"section_header"`
}

// DefaultTheme returns GitSketch's built-in color palette.
func DefaultTheme() Theme {
	return Theme{
		BranchColors: []string{
			"#B388FF", "#69F0AE", "#FFD54F", "#4FC3F7",
			"#FF8A65", "#80CBC4", "#F48FB1", "#AED581",
		},
		GraphLine:     "#546E7A",
		SelectedRowBg: "#1A237E",
		Hash:          "#FFD54F",
		Author:        "#80CBC4",
		Date:          "#90A4AE",
		Subject:       "#ECEFF1",
		Body:          "#B0BEC5",
		BranchRefFg:   "#1B5E20",
		BranchRefBg:   "#69F0AE",
		TagRefFg:      "#BF360C",
		TagRefBg:      "#FF8A65",
		HeadRefFg:     "#311B92",
		HeadRefBg:     "#B388FF",
		PaneBorder:    "#37474F",
		Title:         "#E0E0E0",
		HelpBarFg:     "#78909C",
		HelpBarBg:     "#263238",
		Success:       "#00E676",
		Error:         "#FF5252",
		Modified:      "#FFD54F",
		Added:         "#69F0AE",
		Deleted:       "#FF5252",
		Dim:           "#546E7A",
		Logo:          "#B388FF",
		DetailLabel:   "#78909C",
		DetailValue:   "#ECEFF1",
		SectionHeader: "#B388FF",
	}
}
```

Also update `Config` and `Default()`:

```go
type Config struct {
	KeyMap KeyMap `toml:"keymap"`
	Theme  Theme  `toml:"theme"`
}

func Default() Config {
	return Config{KeyMap: DefaultKeyMap(), Theme: DefaultTheme()}
}
```

- [ ] **Step 4: Rewrite `internal/tui/styles.go`**

Replace the body of `init()` with a call to `ApplyTheme`, and turn the old literal-filled logic into `ApplyTheme` itself:

```go
// internal/tui/styles.go
package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"

	"github.com/Wian47/GitSketch/internal/config"
)

// ─── Color Palette ──────────────────────────────────────────────────────────

// BranchColors are cycled through for different branch lanes in the graph.
var BranchColors []color.Color

// ─── Styles ─────────────────────────────────────────────────────────────────

var (
	GraphLineStyle      lipgloss.Style
	SelectedRowStyle    lipgloss.Style
	NormalRowStyle      lipgloss.Style
	HashStyle           lipgloss.Style
	AuthorStyle         lipgloss.Style
	DateStyle           lipgloss.Style
	SubjectStyle        lipgloss.Style
	BodyStyle           lipgloss.Style
	BranchRefStyle      lipgloss.Style
	TagRefStyle         lipgloss.Style
	HeadRefStyle        lipgloss.Style
	PaneBorderColor     color.Color
	TitleStyle          lipgloss.Style
	HelpBarStyle        lipgloss.Style
	NotifySuccessStyle  lipgloss.Style
	NotifyErrorStyle    lipgloss.Style
	FileModifiedStyle   lipgloss.Style
	FileAddedStyle      lipgloss.Style
	FileDeletedStyle    lipgloss.Style
	DimStyle            lipgloss.Style
	LogoStyle           lipgloss.Style
	DetailLabelStyle    lipgloss.Style
	DetailValueStyle    lipgloss.Style
	SectionHeaderStyle  lipgloss.Style
)

func init() {
	ApplyTheme(config.DefaultTheme())
}

// ApplyTheme rebuilds every package-level style/color var from t. Called
// once at startup with the built-in defaults (via init), and again with the
// user's loaded config if they've customized any colors.
func ApplyTheme(t config.Theme) {
	BranchColors = BranchColors[:0]
	for _, hex := range t.BranchColors {
		BranchColors = append(BranchColors, lipgloss.Color(hex))
	}

	GraphLineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.GraphLine))

	SelectedRowStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(t.SelectedRowBg)).
		Bold(true)

	NormalRowStyle = lipgloss.NewStyle()

	HashStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Hash))
	AuthorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Author))
	DateStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Date)).Italic(true)
	SubjectStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Subject))
	BodyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Body))

	BranchRefStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.BranchRefFg)).
		Background(lipgloss.Color(t.BranchRefBg)).
		Bold(true).
		Padding(0, 1)

	TagRefStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.TagRefFg)).
		Background(lipgloss.Color(t.TagRefBg)).
		Bold(true).
		Padding(0, 1)

	HeadRefStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.HeadRefFg)).
		Background(lipgloss.Color(t.HeadRefBg)).
		Bold(true).
		Padding(0, 1)

	PaneBorderColor = lipgloss.Color(t.PaneBorder)

	TitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Title)).
		Bold(true).
		Padding(0, 1)

	HelpBarStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.HelpBarFg)).
		Background(lipgloss.Color(t.HelpBarBg))

	NotifySuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Success)).Bold(true)
	NotifyErrorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Error)).Bold(true)

	FileModifiedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Modified)).Bold(true)
	FileAddedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Added)).Bold(true)
	FileDeletedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Deleted)).Bold(true)

	DimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Dim))
	LogoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Logo)).Bold(true)

	DetailLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.DetailLabel)).Bold(true)
	DetailValueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.DetailValue))

	SectionHeaderStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.SectionHeader)).
		Bold(true).
		Underline(true)
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/config/... ./internal/tui/... -v`
Expected: PASS — all new tests, plus every pre-existing test in both packages (confirms the refactor preserved default rendering)

- [ ] **Step 6: Wire theme loading into `main.go`**

Add one line after the existing `tui.ApplyKeyMap(cfg.KeyMap)` in `main.go`:

```go
	tui.ApplyKeyMap(cfg.KeyMap)
	tui.ApplyTheme(cfg.Theme)
```

- [ ] **Step 7: Build to verify it compiles**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 8: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go internal/tui/styles.go internal/tui/styles_test.go main.go
git commit -m "feat: make the color palette data-driven via config.Theme"
```

---

### Task 8: Context-aware help overlay

**Files:**
- Create: `internal/tui/help.go`
- Modify: `internal/tui/keys.go` (add `KeyHelp` var + `config.KeyMap.Help` wiring)
- Modify: `internal/config/config.go` (add `KeyMap.Help` field)
- Modify: `internal/tui/model.go` (add `helpMode` state, key handling, View wiring)
- Test: `internal/tui/help_test.go`, additions to `internal/tui/model_test.go`

**Interfaces:**
- Consumes: `KeyUp`, `KeyDown`, `KeyG`, `KeyShiftG`, `KeyPgUp`, `KeyPgDown`, `KeyEnter`, `KeyFilter`, `KeyBranch`, `KeyC`, `KeyQ`, `KeyK`, `KeyJ` (Task 6)
- Produces: `KeyHelp` var; `type HelpEntry struct { Keys string; Desc string }`; `func helpEntries() []HelpEntry`; `func (m Model) renderHelpOverlay() string`

Replaces the dead `HelpText()` function with a real, scrollable-by-design (single screen is enough for Phase 1's binding count) overlay that reflects whatever keys are actually bound — including user overrides from Task 6.

- [ ] **Step 1: Add `Help` to the keymap (config + keys.go)**

In `internal/config/config.go`, add a field to `KeyMap`:

```go
	Help string `toml:"help"`
```

and to `DefaultKeyMap()`:

```go
		Help: "?",
```

In `internal/tui/keys.go`, add the var and wire it into `ApplyKeyMap`:

```go
	KeyHelp   = "?"
```

```go
	setIfNotEmpty(&KeyHelp, km.Help)
```

(Add both alongside the existing declarations from Task 6, in the same `var (...)` block and the same `ApplyKeyMap` function body.)

- [ ] **Step 2: Write the failing tests**

```go
// internal/tui/help_test.go
package tui

import "testing"

func TestHelpEntriesIncludeCoreBindings(t *testing.T) {
	entries := helpEntries()
	if len(entries) == 0 {
		t.Fatal("expected at least one help entry")
	}

	found := map[string]bool{}
	for _, e := range entries {
		found[e.Desc] = true
	}
	for _, want := range []string{"Move up", "Move down", "Quit", "Toggle this help"} {
		if !found[want] {
			t.Fatalf("expected a help entry for %q, got entries: %+v", want, entries)
		}
	}
}

func TestRenderHelpOverlayContainsQuitBinding(t *testing.T) {
	m := Model{width: 80, height: 24}
	out := m.renderHelpOverlay()
	if out == "" {
		t.Fatal("expected non-empty help overlay content")
	}
}
```

```go
// internal/tui/model_test.go — add to the existing file
func TestHandleKeyTogglesHelpMode(t *testing.T) {
	m := Model{width: 80, height: 24, allCommits: sampleCommits()}
	m.applyFilter()

	updated, _ := m.handleKey(tea.KeyPressMsg{Code: '?', Text: "?"})
	m = updated.(Model)
	if !m.helpMode {
		t.Fatal("expected '?' to open help mode")
	}

	updated, _ = m.handleKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = updated.(Model)
	if m.helpMode {
		t.Fatal("expected esc to close help mode")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/tui/... -run 'TestHelpEntries|TestRenderHelpOverlay|TestHandleKeyTogglesHelpMode' -v`
Expected: FAIL — `undefined: helpEntries` / `m.helpMode` compile errors

- [ ] **Step 4: Implement `internal/tui/help.go`**

```go
// internal/tui/help.go
package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// HelpEntry is one row of the help overlay: the key(s) bound to an action
// and a short description of what it does.
type HelpEntry struct {
	Keys string
	Desc string
}

// helpEntries lists every global keybinding, built from the live Key* vars
// so it always reflects the user's configured keymap, not just the
// built-in defaults.
func helpEntries() []HelpEntry {
	return []HelpEntry{
		{fmt.Sprintf("%s/%s", KeyUp, KeyK), "Move up"},
		{fmt.Sprintf("%s/%s", KeyDown, KeyJ), "Move down"},
		{fmt.Sprintf("%s/%s", KeyG, KeyShiftG), "Jump to top/bottom"},
		{fmt.Sprintf("%s/%s", KeyPgUp, KeyPgDown), "Page up/down"},
		{KeyEnter, "View fullscreen diff"},
		{KeyFilter, "Filter commits (regex)"},
		{KeyBranch, "Branch manager"},
		{KeyC, "Checkout selected commit"},
		{KeyHelp, "Toggle this help"},
		{KeyQ, "Quit"},
	}
}

// renderHelpOverlay renders the full-screen keybinding reference.
func (m Model) renderHelpOverlay() string {
	var lines []string
	lines = append(lines, SectionHeaderStyle.Render("Keybindings"))
	lines = append(lines, "")

	for _, e := range helpEntries() {
		row := DetailLabelStyle.Render(fmt.Sprintf("  %-8s", e.Keys)) + DimStyle.Render(e.Desc)
		lines = append(lines, row)
	}

	lines = append(lines, "")
	lines = append(lines, DimStyle.Render(fmt.Sprintf("Press %s or esc to close", KeyHelp)))

	content := strings.Join(lines, "\n")
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}
```

- [ ] **Step 5: Wire `helpMode` into `model.go`**

Add the field to `Model`'s UI State block:

```go
	helpMode bool // showing the full keybinding help overlay
```

Add a new mode block in `handleKey`, right after the `Mode: Fullscreen Diff Viewer` block (so it takes priority over Normal Mode, matching how `showDiff`/`searchMode` are checked):

```go
	// ── Mode: Help Overlay ──
	if m.helpMode {
		switch key {
		case KeyEsc, KeyHelp, KeyQ:
			m.helpMode = false
			return m, nil
		}
		return m, nil
	}
```

Add the toggle to Normal Mode's switch, alongside the other single-key actions:

```go
	case KeyHelp:
		m.helpMode = true
		return m, nil
```

Add a branch in `View()`, alongside the existing `m.showDiff` check:

```go
	} else if m.helpMode {
		content = m.renderHelpOverlay()
	} else if m.showDiff {
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/tui/... -v`
Expected: PASS — all new tests, plus every pre-existing `internal/tui` test

- [ ] **Step 7: Commit**

```bash
git add internal/tui/help.go internal/tui/help_test.go internal/tui/keys.go internal/tui/model.go internal/tui/model_test.go internal/config/config.go
git commit -m "feat: add context-aware help overlay"
```

---

### Task 9: Status bar (branch, ahead/behind, dirty state)

**Files:**
- Create: `internal/tui/statusbar.go`
- Modify: `internal/tui/model.go` (new state, `statusLoadedMsg`, `loadStatusCmd`, refresh wiring, layout)
- Modify: `internal/config/config.go` (add `Theme.StatusBarFg`/`Theme.StatusBarBg`)
- Modify: `internal/tui/styles.go` (add `StatusBarStyle`, wire into `ApplyTheme`)
- Test: `internal/tui/statusbar_test.go`, additions to `internal/tui/model_test.go`

**Interfaces:**
- Consumes: `git.GetStatus() (git.Status, error)` (Task 2)
- Produces: `func renderStatusBar(width int, branch string, ahead, behind, staged, unstaged int) string`; `Model` fields `repoBranch string`, `repoAhead int`, `repoBehind int`, `dirtyStaged int`, `dirtyUnstaged int`; `statusLoadedMsg{status git.Status; err error}`; `loadStatusCmd() tea.Cmd`

- [ ] **Step 1: Add `StatusBarFg`/`StatusBarBg` to the theme**

In `internal/config/config.go`, add to `Theme`:

```go
	StatusBarFg string `toml:"status_bar_fg"`
	StatusBarBg string `toml:"status_bar_bg"`
```

and to `DefaultTheme()`:

```go
		StatusBarFg: "#ECEFF1",
		StatusBarBg: "#37474F",
```

In `internal/tui/styles.go`, add `StatusBarStyle lipgloss.Style` to the var block, and in `ApplyTheme`:

```go
	StatusBarStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.StatusBarFg)).
		Background(lipgloss.Color(t.StatusBarBg))
```

- [ ] **Step 2: Write the failing tests**

```go
// internal/tui/statusbar_test.go
package tui

import "testing"

func TestRenderStatusBarShowsBranchAndAheadBehind(t *testing.T) {
	out := renderStatusBar(80, "main", 2, 1, 0, 0)
	for _, want := range []string{"main", "↑2", "↓1"} {
		if !contains(out, want) {
			t.Fatalf("expected status bar to contain %q, got %q", want, out)
		}
	}
}

func TestRenderStatusBarCleanState(t *testing.T) {
	out := renderStatusBar(80, "main", 0, 0, 0, 0)
	if !contains(out, "clean") {
		t.Fatalf("expected status bar to show \"clean\" with no dirty files, got %q", out)
	}
	if contains(out, "↑") || contains(out, "↓") {
		t.Fatalf("expected no ahead/behind markers at 0/0, got %q", out)
	}
}

func TestRenderStatusBarDirtyCounts(t *testing.T) {
	out := renderStatusBar(80, "main", 0, 0, 3, 2)
	if !contains(out, "3 staged, 2 unstaged") {
		t.Fatalf("expected dirty counts in status bar, got %q", out)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (needle == "" || indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
```

```go
// internal/tui/model_test.go — add to the existing file
func TestUpdateStatusLoadedMsg(t *testing.T) {
	m := Model{}
	updated, _ := m.Update(statusLoadedMsg{status: git.Status{
		Branch: "main", Ahead: 1, Behind: 2,
		Staged:   []git.StatusEntry{{Status: "M", Path: "a.txt"}},
		Unstaged: []git.StatusEntry{{Status: "M", Path: "b.txt"}, {Status: "??", Path: "c.txt"}},
	}})
	mm := updated.(Model)

	if mm.repoBranch != "main" || mm.repoAhead != 1 || mm.repoBehind != 2 {
		t.Fatalf("expected branch/ahead/behind to update, got %+v", mm)
	}
	if mm.dirtyStaged != 1 || mm.dirtyUnstaged != 2 {
		t.Fatalf("expected dirty counts 1/2, got %d/%d", mm.dirtyStaged, mm.dirtyUnstaged)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/tui/... -run 'TestRenderStatusBar|TestUpdateStatusLoadedMsg' -v`
Expected: FAIL — `undefined: renderStatusBar` / `statusLoadedMsg` compile errors

- [ ] **Step 4: Implement `internal/tui/statusbar.go`**

```go
// internal/tui/statusbar.go
package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// renderStatusBar renders the single-line always-visible status bar:
// branch name, ahead/behind counts (omitted when both are zero), and
// staged/unstaged dirty-file counts (or "clean" when there are none).
func renderStatusBar(width int, branch string, ahead, behind, staged, unstaged int) string {
	left := fmt.Sprintf(" %s", branch)
	if ahead > 0 {
		left += fmt.Sprintf(" ↑%d", ahead)
	}
	if behind > 0 {
		left += fmt.Sprintf(" ↓%d", behind)
	}

	var right string
	if staged > 0 || unstaged > 0 {
		right = fmt.Sprintf("%d staged, %d unstaged ", staged, unstaged)
	} else {
		right = "clean "
	}

	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}

	line := left + strings.Repeat(" ", gap) + right
	return StatusBarStyle.Width(width).Render(line)
}
```

- [ ] **Step 5: Wire status refresh and layout into `model.go`**

Add fields to `Model`'s Data block:

```go
	repoBranch    string
	repoAhead     int
	repoBehind    int
	dirtyStaged   int
	dirtyUnstaged int
```

Add the message type alongside the other `*Msg` types:

```go
type statusLoadedMsg struct {
	status git.Status
	err    error
}
```

Add the `Update()` case alongside `filesLoadedMsg`:

```go
	case statusLoadedMsg:
		if msg.err == nil {
			m.repoBranch = msg.status.Branch
			m.repoAhead = msg.status.Ahead
			m.repoBehind = msg.status.Behind
			m.dirtyStaged = len(msg.status.Staged)
			m.dirtyUnstaged = len(msg.status.Unstaged)
		}
		return m, nil
```

Add the command alongside `loadFilesCmd`:

```go
func loadStatusCmd() tea.Cmd {
	return func() tea.Msg {
		status, err := git.GetStatus()
		return statusLoadedMsg{status: status, err: err}
	}
}
```

Kick it off at startup — in `Init()`:

```go
func (m Model) Init() tea.Cmd {
	return tea.Batch(parseCommitsCmd(), loadStatusCmd())
}
```

Refresh it after any action that changes branch/HEAD/working-tree state — in the `checkoutDoneMsg` case, add `loadStatusCmd()` to the existing `tea.Batch`:

```go
		return m, tea.Batch(
			parseCommitsCmd(),
			loadStatusCmd(),
			clearNotifyAfter(3*time.Second),
		)
```

Finally, render it — in `renderLayout()`, prepend the status bar and reduce `paneHeight` by one line to make room:

```go
func (m Model) renderLayout() string {
	leftWidth := m.width * 60 / 100
	rightWidth := m.width - leftWidth

	paneHeight := m.height - 3 // was -2; -1 more for the new status bar line

	leftContent := m.renderGraphPane(leftWidth-4, paneHeight-2)
	rightContent := m.renderDetailPane(rightWidth-4, paneHeight-2)

	leftPane := lipgloss.NewStyle().
		Width(leftWidth-2).
		Height(paneHeight-2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PaneBorderColor).
		Padding(0, 1).
		Render(leftContent)

	rightPane := lipgloss.NewStyle().
		Width(rightWidth-2).
		Height(paneHeight-2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PaneBorderColor).
		Padding(0, 1).
		Render(rightContent)

	mainView := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	statusBar := renderStatusBar(m.width, m.repoBranch, m.repoAhead, m.repoBehind, m.dirtyStaged, m.dirtyUnstaged)
	helpBar := m.renderHelpBar()

	return lipgloss.JoinVertical(lipgloss.Left, statusBar, mainView, helpBar)
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./... -v`
Expected: PASS — every test in the repo, including all pre-existing ones (confirms the layout change didn't break anything)

- [ ] **Step 7: Commit**

```bash
git add internal/tui/statusbar.go internal/tui/statusbar_test.go internal/tui/model.go internal/tui/model_test.go internal/config/config.go internal/tui/styles.go
git commit -m "feat: add always-visible status bar with branch/ahead-behind/dirty state"
```

---

### Task 10: Working-tree row navigation and file list display

**Files:**
- Modify: `internal/tui/model.go`
- Test: additions to `internal/tui/model_test.go`

**Interfaces:**
- Consumes: `git.Status`, `git.StatusEntry` (Task 2), `statusLoadedMsg` (Task 9)
- Produces: `Model` fields `wtSelected bool`, `wtFileCursor int`, `wtStatus git.Status`; `type wtFileRef struct { entry git.StatusEntry; staged bool }`; `func (m Model) wtFileEntries() []wtFileRef`; `func (m Model) selectedWorkingTreeFile() (wtFileRef, bool)`

The working tree is rendered as one **pinned row above the scrollable graph**, not injected into `graph.BuildGraph` — this avoids touching the DAG layout algorithm (which expects real parent hashes) and keeps all existing scroll/cursor math for commits completely untouched. Pressing Up from the top commit (`cursor == 0`) enters the working-tree row; once there, Up/Down move a *file* cursor through the combined staged+unstaged list; moving down past the last file returns to commit 0. Only single-step Up/Down chain into/out of the working-tree row — PgUp/Home always land on commit 0, not the working-tree row, which is an intentional scope boundary, not a bug.

- [ ] **Step 1: Write the failing tests**

```go
// internal/tui/model_test.go — add to the existing file
func sampleWtStatus() git.Status {
	return git.Status{
		Branch: "main",
		Staged: []git.StatusEntry{
			{Status: "M", Path: "staged.txt"},
		},
		Unstaged: []git.StatusEntry{
			{Status: "M", Path: "unstaged1.txt"},
			{Status: "??", Path: "unstaged2.txt"},
		},
	}
}

func TestMoveCursorUpFromTopCommitEntersWorkingTree(t *testing.T) {
	m := Model{allCommits: sampleCommits(), wtStatus: sampleWtStatus()}
	m.applyFilter()
	m.cursor = 0

	m.moveCursor(-1)

	if !m.wtSelected {
		t.Fatal("expected Up from cursor 0 to select the working tree row")
	}
	if m.wtFileCursor != 0 {
		t.Fatalf("expected wtFileCursor 0 on entry, got %d", m.wtFileCursor)
	}
}

func TestMoveCursorWithinWorkingTreeFiles(t *testing.T) {
	m := Model{wtSelected: true, wtFileCursor: 0, wtStatus: sampleWtStatus()}

	m.moveCursor(1)
	if !m.wtSelected || m.wtFileCursor != 1 {
		t.Fatalf("expected to move to file 1 within working tree, got wtSelected=%v wtFileCursor=%d", m.wtSelected, m.wtFileCursor)
	}

	m.moveCursor(1)
	if !m.wtSelected || m.wtFileCursor != 2 {
		t.Fatalf("expected to move to file 2 within working tree, got wtSelected=%v wtFileCursor=%d", m.wtSelected, m.wtFileCursor)
	}
}

func TestMoveCursorDownPastLastFileExitsToCommits(t *testing.T) {
	m := Model{allCommits: sampleCommits(), wtSelected: true, wtFileCursor: 2, wtStatus: sampleWtStatus()}
	m.applyFilter()

	m.moveCursor(1)

	if m.wtSelected {
		t.Fatal("expected moving down past the last file to exit the working tree row")
	}
	if m.cursor != 0 {
		t.Fatalf("expected cursor to land on commit 0, got %d", m.cursor)
	}
}

func TestMoveCursorUpFromWorkingTreeStaysPut(t *testing.T) {
	m := Model{wtSelected: true, wtFileCursor: 0, wtStatus: sampleWtStatus()}
	m.moveCursor(-1)
	if !m.wtSelected || m.wtFileCursor != 0 {
		t.Fatal("expected Up at the top file of the working tree to be a no-op")
	}
}

func TestSelectedCommitNilWhenWorkingTreeSelected(t *testing.T) {
	m := Model{allCommits: sampleCommits(), wtSelected: true}
	m.applyFilter()
	if c := m.selectedCommit(); c != nil {
		t.Fatalf("expected nil selectedCommit while working tree is selected, got %+v", c)
	}
}

func TestSelectedWorkingTreeFile(t *testing.T) {
	m := Model{wtSelected: true, wtFileCursor: 1, wtStatus: sampleWtStatus()}
	ref, ok := m.selectedWorkingTreeFile()
	if !ok {
		t.Fatal("expected a selected working tree file")
	}
	if ref.staged || ref.entry.Path != "unstaged1.txt" {
		t.Fatalf("expected unstaged1.txt (unstaged), got %+v staged=%v", ref.entry, ref.staged)
	}
}

func TestSelectedWorkingTreeFileEmptyReturnsFalse(t *testing.T) {
	m := Model{wtSelected: true, wtFileCursor: 0}
	if _, ok := m.selectedWorkingTreeFile(); ok {
		t.Fatal("expected no selected file when the working tree is clean")
	}
}

func TestRenderGraphPaneShowsWorkingTreeRow(t *testing.T) {
	m := Model{allCommits: sampleCommits(), wtStatus: sampleWtStatus(), width: 80, height: 24}
	m.applyFilter()
	out := m.renderGraphPane(76, 20)
	if !strings.Contains(out, "Working Tree") {
		t.Fatalf("expected graph pane to show the working tree row, got: %s", out)
	}
}

func TestRenderDetailPaneShowsWorkingTreeFiles(t *testing.T) {
	m := Model{wtSelected: true, wtStatus: sampleWtStatus(), width: 80, height: 24}
	out := m.renderDetailPane(76, 20)
	if !strings.Contains(out, "staged.txt") || !strings.Contains(out, "unstaged1.txt") {
		t.Fatalf("expected detail pane to list working tree files, got: %s", out)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/... -run 'TestMoveCursor|TestSelectedCommitNil|TestSelectedWorkingTreeFile|TestRenderGraphPaneShowsWorkingTree|TestRenderDetailPaneShowsWorkingTree' -v`
Expected: FAIL with `m.wtSelected undefined` / `m.wtFileCursor undefined` (compile errors) — note this test file will need `"strings"` imported if not already (check the existing import block in `model_test.go` and add it if missing)

- [ ] **Step 3: Add new state fields to `Model`**

Add to the Data block in `Model`:

```go
	wtStatus git.Status // last-loaded working tree status (staged/unstaged files)
```

Add to a new UI State entry (near `cursor`/`scrollOff`):

```go
	wtSelected   bool // true when focus is on the working-tree row/file list, not a commit
	wtFileCursor int  // index into wtFileEntries() when wtSelected is true
```

- [ ] **Step 4: Implement the working-tree file helpers and cursor logic**

```go
// wtFileRef pairs a working-tree status entry with whether it came from the
// staged or unstaged list, so callers can tell them apart after they've
// been combined into one navigable sequence.
type wtFileRef struct {
	entry  git.StatusEntry
	staged bool
}

// wtFileEntries returns the working tree's staged files followed by its
// unstaged files, as one combined, indexable sequence — this ordering is
// what wtFileCursor indexes into.
func (m Model) wtFileEntries() []wtFileRef {
	entries := make([]wtFileRef, 0, len(m.wtStatus.Staged)+len(m.wtStatus.Unstaged))
	for _, e := range m.wtStatus.Staged {
		entries = append(entries, wtFileRef{entry: e, staged: true})
	}
	for _, e := range m.wtStatus.Unstaged {
		entries = append(entries, wtFileRef{entry: e, staged: false})
	}
	return entries
}

// selectedWorkingTreeFile returns the file at wtFileCursor, or false if the
// working tree is clean (or the cursor is out of range).
func (m Model) selectedWorkingTreeFile() (wtFileRef, bool) {
	entries := m.wtFileEntries()
	if m.wtFileCursor < 0 || m.wtFileCursor >= len(entries) {
		return wtFileRef{}, false
	}
	return entries[m.wtFileCursor], true
}
```

Replace `moveCursor` with a working-tree-aware version:

```go
func (m *Model) moveCursor(delta int) {
	if m.wtSelected {
		total := len(m.wtStatus.Staged) + len(m.wtStatus.Unstaged)
		if total == 0 {
			if delta > 0 {
				m.wtSelected = false
				m.cursor = 0
				m.adjustScroll()
			}
			return
		}
		newCursor := m.wtFileCursor + delta
		if newCursor < 0 {
			return // already at the top file — nothing above the working tree row
		}
		if newCursor >= total {
			m.wtSelected = false
			m.cursor = 0
			m.adjustScroll()
			return
		}
		m.wtFileCursor = newCursor
		return
	}

	total := m.commitRowCount()
	if total == 0 {
		return
	}
	if m.cursor == 0 && delta == -1 {
		m.wtSelected = true
		m.wtFileCursor = 0
		m.adjustScroll()
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= total {
		m.cursor = total - 1
	}
	m.adjustScroll()
}
```

Update `selectedCommit` to return `nil` while the working tree is selected:

```go
func (m Model) selectedCommit() *git.Commit {
	if m.wtSelected {
		return nil
	}
	commitIdx := 0
	for i := range m.graphRows {
		if m.graphRows[i].Commit != nil {
			if commitIdx == m.cursor {
				return m.graphRows[i].Commit
			}
			commitIdx++
		}
	}
	return nil
}
```

In the `KeyG, KeyHome` and `KeyShiftG, KeyEnd` cases of `handleKey`'s Normal Mode switch, clear `wtSelected` so jump-to-top/bottom always lands unambiguously on a commit:

```go
	case KeyG, KeyHome:
		m.wtSelected = false
		m.cursor = 0
		m.scrollOff = 0
		return m, m.loadFilesIfNeeded()

	case KeyShiftG, KeyEnd:
		m.wtSelected = false
		m.cursor = m.commitRowCount() - 1
		m.adjustScroll()
		return m, m.loadFilesIfNeeded()
```

Extend the `statusLoadedMsg` case (added in Task 9) to also cache the full status and clamp the file cursor:

```go
	case statusLoadedMsg:
		if msg.err == nil {
			m.repoBranch = msg.status.Branch
			m.repoAhead = msg.status.Ahead
			m.repoBehind = msg.status.Behind
			m.dirtyStaged = len(msg.status.Staged)
			m.dirtyUnstaged = len(msg.status.Unstaged)
			m.wtStatus = msg.status
			total := m.dirtyStaged + m.dirtyUnstaged
			if m.wtFileCursor >= total {
				m.wtFileCursor = total - 1
			}
			if m.wtFileCursor < 0 {
				m.wtFileCursor = 0
			}
		}
		return m, nil
```

- [ ] **Step 5: Render the working-tree row and file list**

Modify `renderGraphPane` to prepend a pinned row and reserve one line of height for it:

```go
func (m Model) renderGraphPane(width, height int) string {
	wtRow := m.renderWorkingTreeRow(width)
	height--
	if height < 0 {
		height = 0
	}

	if len(m.graphRows) == 0 {
		return wtRow + "\n" + DimStyle.Render("No matching commits.")
	}

	maxGraphCols := graph.MaxColumns(m.graphRows)
	graphWidth := maxGraphCols*2 + 1
	metaWidth := width - graphWidth - 2
	if metaWidth < 10 {
		metaWidth = 10
	}

	var lines []string

	endRow := m.scrollOff + height
	if endRow > len(m.graphRows) {
		endRow = len(m.graphRows)
	}

	for rowIdx := m.scrollOff; rowIdx < endRow; rowIdx++ {
		row := m.graphRows[rowIdx]
		graphStr := m.renderGraphCells(row, maxGraphCols)

		if row.Commit == nil {
			lines = append(lines, graphStr)
			continue
		}

		actualCommitIdx := 0
		for ci := 0; ci < rowIdx; ci++ {
			if m.graphRows[ci].Commit != nil {
				actualCommitIdx++
			}
		}
		isSelected := !m.wtSelected && actualCommitIdx == m.cursor

		meta := m.renderCommitMeta(row.Commit, metaWidth)
		line := graphStr + "  " + meta

		if isSelected {
			line = SelectedRowStyle.Width(width).Render(line)
		}

		lines = append(lines, line)
	}

	return wtRow + "\n" + strings.Join(lines, "\n")
}

// renderWorkingTreeRow renders the single pinned line representing the
// working tree, above the scrollable commit graph.
func (m Model) renderWorkingTreeRow(width int) string {
	staged, unstaged := len(m.wtStatus.Staged), len(m.wtStatus.Unstaged)
	var label string
	if staged == 0 && unstaged == 0 {
		label = "● Working Tree (clean)"
	} else {
		label = fmt.Sprintf("● Working Tree (%d staged, %d unstaged)", staged, unstaged)
	}
	if m.wtSelected {
		return SelectedRowStyle.Width(width).Render(label)
	}
	return DimStyle.Render(label)
}
```

(Note: `isSelected` on the commit-row loop now also requires `!m.wtSelected`, so no commit row is highlighted while the working tree row has focus.)

Modify `renderDetailPane` to branch to a working-tree view:

```go
func (m Model) renderDetailPane(width, height int) string {
	if m.wtSelected {
		return m.renderWorkingTreeDetail(width, height)
	}

	c := m.selectedCommit()
	if c == nil {
		return DimStyle.Render("No commit selected.")
	}
	// ... rest of the existing function body is unchanged ...
```

Add the new rendering function:

```go
// renderWorkingTreeDetail lists the working tree's staged and unstaged
// files, highlighting whichever one wtFileCursor currently points at.
func (m Model) renderWorkingTreeDetail(width, height int) string {
	var lines []string
	lines = append(lines, SectionHeaderStyle.Render("Working Tree"))
	lines = append(lines, "")

	entries := m.wtFileEntries()
	if len(entries) == 0 {
		lines = append(lines, DimStyle.Render("Nothing to commit, working tree clean."))
		return strings.Join(lines, "\n")
	}

	if len(m.wtStatus.Staged) > 0 {
		lines = append(lines, SectionHeaderStyle.Render(fmt.Sprintf("Staged (%d)", len(m.wtStatus.Staged))))
	}
	for i, ref := range entries {
		if i == len(m.wtStatus.Staged) && len(m.wtStatus.Unstaged) > 0 {
			lines = append(lines, "")
			lines = append(lines, SectionHeaderStyle.Render(fmt.Sprintf("Unstaged (%d)", len(m.wtStatus.Unstaged))))
		}
		row := renderWtFileRow(ref, width)
		if i == m.wtFileCursor {
			row = SelectedRowStyle.Width(width).Render(row)
		}
		lines = append(lines, row)
	}
	return strings.Join(lines, "\n")
}

func renderWtFileRow(ref wtFileRef, width int) string {
	var statusStyle lipgloss.Style
	switch ref.entry.Status {
	case "M":
		statusStyle = FileModifiedStyle
	case "A", "??":
		statusStyle = FileAddedStyle
	case "D":
		statusStyle = FileDeletedStyle
	default:
		statusStyle = DimStyle
	}
	path := ref.entry.Path
	if len(path) > width-6 && width > 7 {
		path = "…" + path[len(path)-(width-7):]
	}
	return "  " + statusStyle.Render(ref.entry.Status) + " " + DimStyle.Render(path)
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./... -v`
Expected: PASS — every test in the repo, including all pre-existing commit-navigation tests (confirms the working-tree row didn't change any existing cursor/scroll behavior)

- [ ] **Step 7: Commit**

```bash
git add internal/tui/model.go internal/tui/model_test.go
git commit -m "feat: add working-tree row navigation and file list display"
```

---

### Task 11: Whole-file stage/unstage/discard actions

**Files:**
- Modify: `internal/config/config.go` (add `KeyMap.StageFile`/`UnstageFile`/`Discard`)
- Modify: `internal/tui/keys.go` (add `KeyStageFile`/`KeyUnstageFile`/`KeyDiscard` vars + `ApplyKeyMap` wiring)
- Modify: `internal/tui/model.go` (new state, key handling, commands)
- Modify: `internal/tui/help.go` (add entries)
- Test: additions to `internal/tui/model_test.go`

**Interfaces:**
- Consumes: `git.StageFile`, `git.UnstageFile`, `git.DiscardFile` (Task 3), `wtFileRef`/`selectedWorkingTreeFile` (Task 10)
- Produces: `KeyStageFile = "a"`, `KeyUnstageFile = "u"`, `KeyDiscard = "x"`; `Model.confirmDiscard bool`; `stagingDoneMsg{action, path string; err error}`; `func stageFileCmd/unstageFileCmd/discardFileCmd(path string) tea.Cmd`

- [ ] **Step 1: Add the new keymap fields**

In `internal/config/config.go`, add to `KeyMap`:

```go
	StageFile   string `toml:"stage_file"`
	UnstageFile string `toml:"unstage_file"`
	Discard     string `toml:"discard"`
```

and to `DefaultKeyMap()`:

```go
		StageFile: "a", UnstageFile: "u", Discard: "x",
```

In `internal/tui/keys.go`, add the vars and wire them into `ApplyKeyMap`:

```go
	KeyStageFile   = "a"
	KeyUnstageFile = "u"
	KeyDiscard     = "x"
```

```go
	setIfNotEmpty(&KeyStageFile, km.StageFile)
	setIfNotEmpty(&KeyUnstageFile, km.UnstageFile)
	setIfNotEmpty(&KeyDiscard, km.Discard)
```

- [ ] **Step 2: Write the failing tests**

```go
// internal/tui/model_test.go — add to the existing file
func TestStageSelectedFileOnlyActsOnUnstaged(t *testing.T) {
	m := Model{wtSelected: true, wtFileCursor: 0, wtStatus: sampleWtStatus()} // index 0 is staged.txt (already staged)
	if cmd := m.stageSelectedFile(); cmd != nil {
		t.Fatal("expected no-op staging a file that's already staged")
	}

	m.wtFileCursor = 1 // unstaged1.txt
	if cmd := m.stageSelectedFile(); cmd == nil {
		t.Fatal("expected a command staging an unstaged file")
	}
}

func TestUnstageSelectedFileOnlyActsOnStaged(t *testing.T) {
	m := Model{wtSelected: true, wtFileCursor: 1, wtStatus: sampleWtStatus()} // unstaged1.txt
	if cmd := m.unstageSelectedFile(); cmd != nil {
		t.Fatal("expected no-op unstaging a file that's already unstaged")
	}

	m.wtFileCursor = 0 // staged.txt
	if cmd := m.unstageSelectedFile(); cmd == nil {
		t.Fatal("expected a command unstaging a staged file")
	}
}
```

`lipgloss.Style` (v2) contains a `[]color.Color` field and a `func(string) string` field, so it is not comparable with `==`/`!=` (this is a compile error, not a lint warning) — use `reflect.DeepEqual` instead, and add `"reflect"` to `model_test.go`'s import block if not already present. Both notify styles set no `Transform`, so the `func` field is always nil on both sides and `reflect.DeepEqual` compares correctly.

```go
func TestUpdateStagingDoneMsgSuccess(t *testing.T) {
	m := Model{}
	updated, cmd := m.Update(stagingDoneMsg{action: "staged", path: "a.txt"})
	mm := updated.(Model)
	if !reflect.DeepEqual(mm.notifyStyle, NotifySuccessStyle) {
		t.Fatal("expected success notify style")
	}
	if !strings.Contains(mm.notification, "staged") || !strings.Contains(mm.notification, "a.txt") {
		t.Fatalf("expected notification to mention the action and path, got %q", mm.notification)
	}
	if cmd == nil {
		t.Fatal("expected a refresh command to be returned")
	}
}

func TestUpdateStagingDoneMsgError(t *testing.T) {
	m := Model{}
	updated, _ := m.Update(stagingDoneMsg{action: "staged", path: "a.txt", err: errTest})
	mm := updated.(Model)
	if !reflect.DeepEqual(mm.notifyStyle, NotifyErrorStyle) {
		t.Fatal("expected error notify style")
	}
}

func TestHandleKeyDiscardOnlyPromptsForUnstagedFile(t *testing.T) {
	m := Model{wtSelected: true, wtFileCursor: 0, wtStatus: sampleWtStatus()} // staged.txt
	updated, _ := m.handleKey(tea.KeyPressMsg{Code: 'x', Text: "x"})
	mm := updated.(Model)
	if mm.confirmDiscard {
		t.Fatal("expected discard to not prompt for a staged file")
	}

	m.wtFileCursor = 1 // unstaged1.txt
	updated, _ = m.handleKey(tea.KeyPressMsg{Code: 'x', Text: "x"})
	mm = updated.(Model)
	if !mm.confirmDiscard {
		t.Fatal("expected discard to prompt for an unstaged file")
	}
}
```

Add a package-level test error near the top of `model_test.go` (or reuse an existing one if present):

```go
var errTest = fmt.Errorf("boom")
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/tui/... -run 'TestStageSelectedFile|TestUnstageSelectedFile|TestUpdateStagingDoneMsg|TestHandleKeyDiscard' -v`
Expected: FAIL — `m.stageSelectedFile undefined` etc. (compile errors)

- [ ] **Step 4: Add state, messages, and commands to `model.go`**

Add to `Model`'s Mode block:

```go
	confirmDiscard bool // showing discard-changes confirmation for the focused working-tree file
```

Add the message type and `Update()` case:

```go
type stagingDoneMsg struct {
	action string // "staged", "unstaged", or "discarded"
	path   string
	err    error
}
```

```go
	case stagingDoneMsg:
		if msg.err != nil {
			m.notification = fmt.Sprintf(" ✗ %s", msg.err.Error())
			m.notifyStyle = NotifyErrorStyle
		} else {
			m.notification = fmt.Sprintf(" ✓ %s %s", msg.action, msg.path)
			m.notifyStyle = NotifySuccessStyle
		}
		return m, tea.Batch(loadStatusCmd(), clearNotifyAfter(3*time.Second))
```

Add the commands and helper methods:

```go
func stageFileCmd(path string) tea.Cmd {
	return func() tea.Msg {
		return stagingDoneMsg{action: "staged", path: path, err: git.StageFile(path)}
	}
}

func unstageFileCmd(path string) tea.Cmd {
	return func() tea.Msg {
		return stagingDoneMsg{action: "unstaged", path: path, err: git.UnstageFile(path)}
	}
}

func discardFileCmd(path string) tea.Cmd {
	return func() tea.Msg {
		return stagingDoneMsg{action: "discarded", path: path, err: git.DiscardFile(path)}
	}
}

// stageSelectedFile stages the working-tree file under the cursor, or does
// nothing if it's already staged (or nothing is selected).
func (m Model) stageSelectedFile() tea.Cmd {
	ref, ok := m.selectedWorkingTreeFile()
	if !ok || ref.staged {
		return nil
	}
	return stageFileCmd(ref.entry.Path)
}

// unstageSelectedFile unstages the working-tree file under the cursor, or
// does nothing if it's already unstaged (or nothing is selected).
func (m Model) unstageSelectedFile() tea.Cmd {
	ref, ok := m.selectedWorkingTreeFile()
	if !ok || !ref.staged {
		return nil
	}
	return unstageFileCmd(ref.entry.Path)
}
```

- [ ] **Step 5: Wire key handling**

Add a new mode block in `handleKey`, right after the existing `Mode: Checkout Confirmation` block:

```go
	// ── Mode: Discard Confirmation ──
	if m.confirmDiscard {
		switch key {
		case KeyY:
			m.confirmDiscard = false
			if ref, ok := m.selectedWorkingTreeFile(); ok {
				return m, discardFileCmd(ref.entry.Path)
			}
			return m, nil
		case KeyN, KeyEsc:
			m.confirmDiscard = false
			return m, nil
		}
		return m, nil
	}
```

Add three cases to Normal Mode's switch:

```go
	case KeyStageFile:
		if m.wtSelected {
			return m, m.stageSelectedFile()
		}
		return m, nil

	case KeyUnstageFile:
		if m.wtSelected {
			return m, m.unstageSelectedFile()
		}
		return m, nil

	case KeyDiscard:
		if m.wtSelected {
			if ref, ok := m.selectedWorkingTreeFile(); ok && !ref.staged {
				m.confirmDiscard = true
			}
		}
		return m, nil
```

Add two branches to `renderHelpBar`, alongside the existing `m.confirmCheckout` branch:

```go
	} else if m.confirmDiscard {
		text = NotifyErrorStyle.Render("  Discard changes to this file? (y/n)")
	} else if m.wtSelected {
		text = fmt.Sprintf("  ↑/k ↓/j Files  %s Stage  %s Unstage  %s Discard  q Quit", KeyStageFile, KeyUnstageFile, KeyDiscard)
```

(Place `else if m.wtSelected` after `else if m.confirmCheckout` and before the final generic `else`.)

- [ ] **Step 6: Add help overlay entries**

In `internal/tui/help.go`'s `helpEntries()`, add three rows (after the `KeyBranch` entry):

```go
		{KeyStageFile, "Stage focused working-tree file"},
		{KeyUnstageFile, "Unstage focused working-tree file"},
		{KeyDiscard, "Discard changes to focused file (with confirm)"},
```

- [ ] **Step 7: Run tests to verify they pass**

Run: `go test ./... -v`
Expected: PASS — every test in the repo

- [ ] **Step 8: Commit**

```bash
git add internal/config/config.go internal/tui/keys.go internal/tui/model.go internal/tui/model_test.go internal/tui/help.go
git commit -m "feat: add whole-file stage/unstage/discard actions"
```

---

### Task 12: Commit input mode

**Files:**
- Modify: `internal/tui/model.go`
- Test: additions to `internal/tui/model_test.go`

**Interfaces:**
- Consumes: `git.CreateCommit(message string) git.CommitResult` (Task 4)
- Produces: `Model` fields `commitInputMode bool`, `commitMessage string`; `commitDoneMsg{result git.CommitResult}`; `func commitCmd(message string) tea.Cmd`

Reuses the existing `KeyC` binding: when the working-tree row is focused, `c` means "commit staged changes" instead of "checkout" (checkout only ever fires when `selectedCommit() != nil`, which is already `nil` while `m.wtSelected` — see Task 10 — so this is a safe, contextual reuse rather than a new binding).

- [ ] **Step 1: Write the failing tests**

```go
// internal/tui/model_test.go — add to the existing file
func TestHandleKeyCommitOpensInputOnlyWithStagedChanges(t *testing.T) {
	m := Model{wtSelected: true, dirtyStaged: 0}
	updated, _ := m.handleKey(tea.KeyPressMsg{Code: 'c', Text: "c"})
	mm := updated.(Model)
	if mm.commitInputMode {
		t.Fatal("expected commit input to not open with nothing staged")
	}

	m = Model{wtSelected: true, dirtyStaged: 1}
	updated, _ = m.handleKey(tea.KeyPressMsg{Code: 'c', Text: "c"})
	mm = updated.(Model)
	if !mm.commitInputMode {
		t.Fatal("expected commit input to open with something staged")
	}
}

func TestHandleKeyCommitInputCapturesTextAndSubmits(t *testing.T) {
	m := Model{commitInputMode: true, commitMessage: "fix bu"}
	updated, _ := m.handleKey(tea.KeyPressMsg{Code: 'g', Text: "g"})
	mm := updated.(Model)
	if mm.commitMessage != "fix bug" {
		t.Fatalf("expected commitMessage %q, got %q", "fix bug", mm.commitMessage)
	}

	updated, cmd := mm.handleKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	mm = updated.(Model)
	if mm.commitInputMode {
		t.Fatal("expected commit input mode to close on submit")
	}
	if cmd == nil {
		t.Fatal("expected a commit command to be returned on submit")
	}
}

func TestHandleKeyCommitInputEmptyMessageDoesNotSubmit(t *testing.T) {
	m := Model{commitInputMode: true, commitMessage: ""}
	updated, cmd := m.handleKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	mm := updated.(Model)
	if !mm.commitInputMode {
		t.Fatal("expected commit input mode to stay open when message is empty")
	}
	if cmd != nil {
		t.Fatal("expected no command when submitting an empty commit message")
	}
}
```

Note: `lipgloss.Style` (v2) is not comparable with `==`/`!=` (it contains a `[]color.Color` field and a `func(string) string` field — compile error, not a lint warning). Use `reflect.DeepEqual` instead, and add `"reflect"` to `model_test.go`'s import block if not already present (Task 11 already added it, if executed first).

```go
func TestUpdateCommitDoneMsgSuccess(t *testing.T) {
	m := Model{}
	updated, cmd := m.Update(commitDoneMsg{result: git.CommitResult{Success: true, Message: "1 file changed"}})
	mm := updated.(Model)
	if !reflect.DeepEqual(mm.notifyStyle, NotifySuccessStyle) {
		t.Fatal("expected success notify style")
	}
	if cmd == nil {
		t.Fatal("expected a refresh command to be returned")
	}
}

func TestUpdateCommitDoneMsgFailure(t *testing.T) {
	m := Model{}
	updated, _ := m.Update(commitDoneMsg{result: git.CommitResult{Success: false, Message: "nothing to commit"}})
	mm := updated.(Model)
	if !reflect.DeepEqual(mm.notifyStyle, NotifyErrorStyle) {
		t.Fatal("expected error notify style")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/... -run 'TestHandleKeyCommit|TestUpdateCommitDoneMsg' -v`
Expected: FAIL — `m.commitInputMode undefined` (compile errors)

- [ ] **Step 3: Add state, messages, and commands to `model.go`**

Add to `Model`'s Mode block:

```go
	commitInputMode bool
	commitMessage   string
```

Add the message type and command:

```go
type commitDoneMsg struct {
	result git.CommitResult
}

func commitCmd(message string) tea.Cmd {
	return func() tea.Msg {
		return commitDoneMsg{result: git.CreateCommit(message)}
	}
}
```

Add the `Update()` case, alongside `checkoutDoneMsg`:

```go
	case commitDoneMsg:
		if msg.result.Success {
			text := msg.result.Message
			if text == "" {
				text = "Committed"
			}
			m.notification = " ✓ " + text
			m.notifyStyle = NotifySuccessStyle
		} else {
			m.notification = fmt.Sprintf(" ✗ %s", msg.result.Message)
			m.notifyStyle = NotifyErrorStyle
		}
		return m, tea.Batch(
			parseCommitsCmd(),
			loadStatusCmd(),
			clearNotifyAfter(3*time.Second),
		)
```

- [ ] **Step 4: Wire key handling**

Add a new mode block in `handleKey`, right after the `Mode: Branch Manager` block:

```go
	// ── Mode: Commit Message Input ──
	if m.commitInputMode {
		switch key {
		case KeyEsc:
			m.commitInputMode = false
			m.commitMessage = ""
			return m, nil
		case KeyEnter:
			if m.commitMessage == "" {
				return m, nil
			}
			msg := m.commitMessage
			m.commitInputMode = false
			m.commitMessage = ""
			return m, commitCmd(msg)
		case "backspace":
			if len(m.commitMessage) > 0 {
				m.commitMessage = m.commitMessage[:len(m.commitMessage)-1]
			}
			return m, nil
		default:
			if len(key) == 1 {
				m.commitMessage += key
			} else if key == "space" {
				m.commitMessage += " "
			}
			return m, nil
		}
	}
```

Modify the existing `case KeyC:` in Normal Mode's switch:

```go
	case KeyC:
		if m.wtSelected {
			if m.dirtyStaged == 0 {
				return m, nil
			}
			m.commitInputMode = true
			m.commitMessage = ""
			return m, nil
		}
		c := m.selectedCommit()
		if c != nil {
			m.confirmCheckout = true
		}
		return m, nil
```

Add a branch to `renderHelpBar`, alongside the existing `m.branchMode` branches:

```go
	} else if m.commitInputMode {
		text = "  Commit message: " + m.commitMessage + "█ (enter Commit, esc Cancel)"
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./... -v`
Expected: PASS — every test in the repo

- [ ] **Step 6: Commit**

```bash
git add internal/tui/model.go internal/tui/model_test.go
git commit -m "feat: add commit input mode for staged changes"
```

---

### Task 13: Hunk-level interactive staging

**Files:**
- Modify: `internal/git/commands.go` (new `GetWorkingTreeDiff` function)
- Modify: `internal/tui/model.go` (new state, key handling, rendering, commands)
- Test: `internal/git/git_test.go` additions, `internal/tui/model_test.go` additions

**Interfaces:**
- Consumes: `git.ParseHunks`, `git.Hunk`, `git.StageHunk`, `git.UnstageHunk` (Tasks 1 & 3), `selectedWorkingTreeFile` (Task 10)
- Produces: `func GetWorkingTreeDiff(path string, staged bool) (string, error)` in package `git`; `Model` fields `stagingDiffMode bool`, `stagingFileHeader string`, `stagingHunks []git.Hunk`, `stagingHunkCursor int`, `stagingFilePath string`, `stagingFileStaged bool`

Pressing `Enter` while the working tree is focused (instead of a commit) opens the same fullscreen view used for commit diffs, but in a hunk-staging mode: `↑/↓` move between hunks, `space` stages (or unstages) the hunk under the cursor. This is whole-hunk staging only — no line-level/partial-hunk staging in Phase 1 (see the design spec's non-goals).

- [ ] **Step 1: Write the failing test for `GetWorkingTreeDiff`**

```go
// internal/git/git_test.go — add to the existing file
func TestGetWorkingTreeDiffUnstagedVsStaged(t *testing.T) {
	initTestRepo(t)
	writeAndCommit(t, "a.txt", "line1\nline2\n", "first commit")

	if err := os.WriteFile("a.txt", []byte("line1\nline2-changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	unstagedDiff, err := GetWorkingTreeDiff("a.txt", false)
	if err != nil {
		t.Fatalf("GetWorkingTreeDiff(unstaged) error = %v", err)
	}
	if !strings.Contains(unstagedDiff, "line2-changed") {
		t.Fatalf("expected unstaged diff to contain the change, got: %s", unstagedDiff)
	}

	stagedDiffBefore, err := GetWorkingTreeDiff("a.txt", true)
	if err != nil {
		t.Fatalf("GetWorkingTreeDiff(staged) error = %v", err)
	}
	if stagedDiffBefore != "" {
		t.Fatalf("expected empty staged diff before staging, got: %s", stagedDiffBefore)
	}

	runGit(t, "add", "a.txt")

	stagedDiffAfter, err := GetWorkingTreeDiff("a.txt", true)
	if err != nil {
		t.Fatalf("GetWorkingTreeDiff(staged) error = %v", err)
	}
	if !strings.Contains(stagedDiffAfter, "line2-changed") {
		t.Fatalf("expected staged diff to contain the change after staging, got: %s", stagedDiffAfter)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/git/... -run TestGetWorkingTreeDiff -v`
Expected: FAIL with `undefined: GetWorkingTreeDiff` (compile error)

- [ ] **Step 3: Implement `GetWorkingTreeDiff`**

Add to `internal/git/commands.go`, alongside `GetCommitDiff`:

```go
// GetWorkingTreeDiff retrieves the unified diff for a single path in the
// working tree: the unstaged diff (working tree vs index) if staged is
// false, or the staged diff (index vs HEAD) if staged is true.
func GetWorkingTreeDiff(path string, staged bool) (string, error) {
	args := []string{"diff", "--color=never"}
	if staged {
		args = append(args, "--cached")
	}
	args = append(args, "--", path)

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git diff: %w: %s", err, string(output))
	}
	return string(output), nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/git/... -run TestGetWorkingTreeDiff -v`
Expected: PASS

- [ ] **Step 5: Commit the git-layer addition**

```bash
git add internal/git/commands.go internal/git/git_test.go
git commit -m "feat: add per-file working-tree diff retrieval"
```

- [ ] **Step 6: Write the failing TUI tests**

```go
// internal/tui/model_test.go — add to the existing file
func TestOpenStagingDiffNoOpWhenTreeClean(t *testing.T) {
	m := Model{wtSelected: true}
	cmd := m.openStagingDiff()
	if cmd != nil {
		t.Fatal("expected no command when the working tree is clean")
	}
	if m.showDiff {
		t.Fatal("expected showDiff to stay false when there's nothing to open")
	}
}

func TestOpenStagingDiffSetsState(t *testing.T) {
	m := Model{wtSelected: true, wtFileCursor: 1, wtStatus: sampleWtStatus()} // unstaged1.txt
	cmd := m.openStagingDiff()
	if cmd == nil {
		t.Fatal("expected a load-diff command")
	}
	if !m.showDiff || !m.stagingDiffMode {
		t.Fatal("expected showDiff and stagingDiffMode to be set")
	}
	if m.stagingFilePath != "unstaged1.txt" || m.stagingFileStaged {
		t.Fatalf("expected unstaged1.txt/unstaged, got path=%q staged=%v", m.stagingFilePath, m.stagingFileStaged)
	}
}

func TestUpdateStagingDiffLoadedMsg(t *testing.T) {
	m := Model{}
	hunks := []git.Hunk{{Header: "@@ -1,1 +1,1 @@", Lines: []string{"@@ -1,1 +1,1 @@", "-a", "+b"}}}
	updated, _ := m.Update(stagingDiffLoadedMsg{path: "a.txt", header: "diff --git a/a.txt b/a.txt", hunks: hunks})
	mm := updated.(Model)
	if len(mm.stagingHunks) != 1 || mm.stagingFileHeader != "diff --git a/a.txt b/a.txt" {
		t.Fatalf("expected hunks/header to be set, got %+v", mm)
	}
}

func TestToggleSelectedHunkOutOfRangeIsNoOp(t *testing.T) {
	m := Model{stagingHunkCursor: 5, stagingHunks: nil}
	if cmd := m.toggleSelectedHunk(); cmd != nil {
		t.Fatal("expected no-op when there are no hunks")
	}
}

func TestToggleSelectedHunkReturnsCommand(t *testing.T) {
	m := Model{
		stagingHunkCursor:  0,
		stagingHunks:       []git.Hunk{{Header: "@@ -1,1 +1,1 @@", Lines: []string{"@@ -1,1 +1,1 @@", "-a", "+b"}}},
		stagingFileHeader:  "diff --git a/a.txt b/a.txt",
		stagingFilePath:    "a.txt",
		stagingFileStaged:  false,
	}
	if cmd := m.toggleSelectedHunk(); cmd == nil {
		t.Fatal("expected a command to stage the selected hunk")
	}
}

func TestHandleKeyEnterOnWorkingTreeOpensStagingDiff(t *testing.T) {
	m := Model{wtSelected: true, wtFileCursor: 1, wtStatus: sampleWtStatus()}
	updated, cmd := m.handleKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	mm := updated.(Model)
	if !mm.stagingDiffMode {
		t.Fatal("expected Enter on the working tree to open staging diff mode")
	}
	if cmd == nil {
		t.Fatal("expected a load command")
	}
}

func TestHandleKeyHunkNavigationInStagingMode(t *testing.T) {
	m := Model{
		showDiff:        true,
		stagingDiffMode: true,
		stagingHunks: []git.Hunk{
			{Header: "@@ -1,1 +1,1 @@"},
			{Header: "@@ -5,1 +5,1 @@"},
		},
		stagingHunkCursor: 0,
	}
	updated, _ := m.handleKey(tea.KeyPressMsg{Code: 'j', Text: "j"})
	mm := updated.(Model)
	if mm.stagingHunkCursor != 1 {
		t.Fatalf("expected cursor to move to hunk 1, got %d", mm.stagingHunkCursor)
	}

	updated, _ = mm.handleKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	mm = updated.(Model)
	if mm.showDiff || mm.stagingDiffMode {
		t.Fatal("expected esc to close staging diff mode")
	}
}
```

- [ ] **Step 7: Run tests to verify they fail**

Run: `go test ./internal/tui/... -run 'TestOpenStagingDiff|TestUpdateStagingDiffLoadedMsg|TestToggleSelectedHunk|TestHandleKeyEnterOnWorkingTree|TestHandleKeyHunkNavigation' -v`
Expected: FAIL — `m.openStagingDiff undefined` etc. (compile errors)

- [ ] **Step 8: Add state, messages, and commands to `model.go`**

Add to `Model`'s Fullscreen Diff View State block:

```go
	stagingDiffMode    bool // true when the fullscreen view (showDiff) is hunk-staging mode, not a plain commit diff
	stagingFileHeader  string
	stagingHunks       []git.Hunk
	stagingHunkCursor  int
	stagingFilePath    string
	stagingFileStaged  bool // whether the currently-open diff is the staged or unstaged view of stagingFilePath
```

Add the message types:

```go
type stagingDiffLoadedMsg struct {
	path   string
	staged bool
	header string
	hunks  []git.Hunk
	err    error
}

type hunkStagingDoneMsg struct {
	path   string
	staged bool // which view (staged/unstaged) we were operating against
	err    error
}
```

Add the `Update()` cases, alongside `diffLoadedMsg`:

```go
	case stagingDiffLoadedMsg:
		if msg.err != nil {
			m.diffContent = fmt.Sprintf("Error loading diff: %v", msg.err)
			return m, nil
		}
		m.stagingFileHeader = msg.header
		m.stagingHunks = msg.hunks
		if m.stagingHunkCursor >= len(m.stagingHunks) {
			m.stagingHunkCursor = 0
		}
		return m, nil

	case hunkStagingDoneMsg:
		if msg.err != nil {
			m.notification = fmt.Sprintf(" ✗ %s", msg.err.Error())
			m.notifyStyle = NotifyErrorStyle
			return m, clearNotifyAfter(3 * time.Second)
		}
		action := "staged"
		if msg.staged {
			action = "unstaged"
		}
		m.notification = fmt.Sprintf(" ✓ hunk %s in %s", action, msg.path)
		m.notifyStyle = NotifySuccessStyle
		return m, tea.Batch(
			loadStagingDiffCmd(msg.path, msg.staged),
			loadStatusCmd(),
			clearNotifyAfter(3*time.Second),
		)
```

Add the commands and helper methods:

```go
func loadStagingDiffCmd(path string, staged bool) tea.Cmd {
	return func() tea.Msg {
		diff, err := git.GetWorkingTreeDiff(path, staged)
		if err != nil {
			return stagingDiffLoadedMsg{path: path, staged: staged, err: err}
		}
		header, hunks := git.ParseHunks(diff)
		return stagingDiffLoadedMsg{path: path, staged: staged, header: header, hunks: hunks}
	}
}

// openStagingDiff opens the fullscreen hunk-staging view for whichever
// working-tree file is currently focused. Returns nil (no-op) if the
// working tree is clean.
func (m *Model) openStagingDiff() tea.Cmd {
	ref, ok := m.selectedWorkingTreeFile()
	if !ok {
		return nil
	}
	m.showDiff = true
	m.stagingDiffMode = true
	m.stagingFilePath = ref.entry.Path
	m.stagingFileStaged = ref.staged
	m.stagingHunkCursor = 0
	return loadStagingDiffCmd(ref.entry.Path, ref.staged)
}

// toggleSelectedHunk stages the hunk under the cursor if we're viewing the
// unstaged diff, or unstages it if we're viewing the staged diff.
func (m Model) toggleSelectedHunk() tea.Cmd {
	if m.stagingHunkCursor < 0 || m.stagingHunkCursor >= len(m.stagingHunks) {
		return nil
	}
	hunk := m.stagingHunks[m.stagingHunkCursor]
	header := m.stagingFileHeader
	path := m.stagingFilePath
	staged := m.stagingFileStaged
	return func() tea.Msg {
		var err error
		if staged {
			err = git.UnstageHunk(header, hunk)
		} else {
			err = git.StageHunk(header, hunk)
		}
		return hunkStagingDoneMsg{path: path, staged: staged, err: err}
	}
}
```

- [ ] **Step 9: Wire key handling**

Change the `case KeyEnter:` in Normal Mode's switch:

```go
	case KeyEnter:
		if m.wtSelected {
			return m, m.openStagingDiff()
		}
		c := m.selectedCommit()
		if c != nil {
			m.showDiff = true
			m.diffContent = ""
			m.diffScroll = 0
			return m, loadDiffCmd(c.Hash)
		}
		return m, nil
```

Split the `Mode: Fullscreen Diff Viewer` block in `handleKey` to branch on `stagingDiffMode`:

```go
	// ── Mode: Fullscreen Diff Viewer ──
	if m.showDiff {
		if m.stagingDiffMode {
			switch key {
			case KeyEsc, KeyQ:
				m.showDiff = false
				m.stagingDiffMode = false
				m.stagingHunks = nil
				return m, nil
			case KeyUp, KeyK:
				if m.stagingHunkCursor > 0 {
					m.stagingHunkCursor--
				}
				return m, nil
			case KeyDown, KeyJ:
				if m.stagingHunkCursor < len(m.stagingHunks)-1 {
					m.stagingHunkCursor++
				}
				return m, nil
			case "space":
				return m, m.toggleSelectedHunk()
			}
			return m, nil
		}
		switch key {
		case KeyEsc, KeyEnter, KeyQ:
			m.showDiff = false
			m.diffContent = ""
			return m, nil
		case KeyUp, KeyK:
			if m.diffScroll > 0 {
				m.diffScroll--
			}
			return m, nil
		case KeyDown, KeyJ:
			m.diffScroll++
			return m, nil
		case KeyPgUp:
			m.diffScroll -= m.height - 4
			if m.diffScroll < 0 {
				m.diffScroll = 0
			}
			return m, nil
		case KeyPgDown:
			m.diffScroll += m.height - 4
			return m, nil
		}
		return m, nil
	}
```

- [ ] **Step 10: Render the staging diff view**

Add the branch in `View()`:

```go
	} else if m.showDiff {
		if m.stagingDiffMode {
			content = m.renderStagingDiffView()
		} else {
			content = m.renderDiffView()
		}
	} else if len(m.graphRows) == 0 {
```

Add the rendering function, alongside `renderDiffView`:

```go
// renderStagingDiffView renders every hunk of the currently-open
// working-tree file diff, highlighting the hunk under stagingHunkCursor.
func (m Model) renderStagingDiffView() string {
	if len(m.stagingHunks) == 0 {
		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			DimStyle.Render("No changes left in this file."),
		)
	}

	var rendered []string
	title := fmt.Sprintf(" %s (esc to return, space to stage/unstage hunk)", m.stagingFilePath)
	rendered = append(rendered, SectionHeaderStyle.Render(title))
	rendered = append(rendered, "")

	for i, h := range m.stagingHunks {
		for _, l := range h.Lines {
			var styled string
			switch {
			case strings.HasPrefix(l, "+"):
				styled = FileAddedStyle.Render(l)
			case strings.HasPrefix(l, "-"):
				styled = FileDeletedStyle.Render(l)
			case strings.HasPrefix(l, "@@"):
				styled = lipgloss.NewStyle().Foreground(lipgloss.Color("#4FC3F7")).Render(l)
			default:
				styled = BodyStyle.Render(l)
			}
			if i == m.stagingHunkCursor {
				styled = SelectedRowStyle.Width(m.width).Render(styled)
			}
			rendered = append(rendered, styled)
		}
	}

	for len(rendered) < m.height-1 {
		rendered = append(rendered, "")
	}
	rendered = append(rendered, m.renderHelpBar())
	return strings.Join(rendered, "\n")
}
```

Add a branch to `renderHelpBar`, alongside the existing `m.showDiff` branch:

```go
	if m.showDiff && m.stagingDiffMode {
		text = "  ↑/k ↓/j Hunk  space Stage/Unstage  esc Close"
	} else if m.showDiff {
```

(This replaces the plain `if m.showDiff {` at the top of `renderHelpBar` with the two-branch form above; the existing body of that branch becomes the `else if` case.)

- [ ] **Step 11: Run tests to verify they pass**

Run: `go test ./... -v`
Expected: PASS — every test in the repo

- [ ] **Step 12: Commit**

```bash
git add internal/tui/model.go internal/tui/model_test.go
git commit -m "feat: add hunk-level interactive staging"
```

---

## Self-Review

**Spec coverage** — every Phase 1 goal from `docs/superpowers/specs/2026-07-23-phase1-daily-driver-core-design.md` maps to a task:
- Staging workflow (working-tree row, whole-file + hunk staging, commit) → Tasks 10, 11, 12, 13
- Status bar → Task 9
- Keymap/config system → Tasks 5, 6
- Theme data-plumbing → Task 7
- Help overlay (called out in the design's UI/UX section) → Task 8

**Placeholder scan** — no TODOs/TBDs; every step has complete, concrete code; every test has real assertions against real return values.

**Type consistency** — checked across tasks: `git.Hunk{Header, Lines}` (Task 1) is used identically in Tasks 3 and 13; `git.StatusEntry{Status, Path}` and `git.Status{Branch, Ahead, Behind, Staged, Unstaged}` (Task 2) are used identically in Tasks 9, 10, 11; `config.KeyMap` fields added incrementally in Tasks 5, 8, 11 are each immediately wired into the matching `tui.Key*` var and `ApplyKeyMap` in the same task; `wtFileRef{entry, staged}` (Task 10) is consumed identically in Tasks 11 and 13; `stagingDiffMode`/`stagingHunks`/`stagingHunkCursor`/`stagingFileHeader`/`stagingFilePath`/`stagingFileStaged` (all introduced in Task 13) are used consistently within that task only, no cross-task drift.

**Risk resolved from the design spec:** the "hunk-header recalculation" risk noted in the spec only applies to line-level (partial-hunk) staging, which is explicitly out of scope — Phase 1 only stages/unstages whole hunks copied verbatim from the source diff (Task 1's `Hunk.Patch`), so no recalculation is needed and the risk doesn't apply to what's actually being built.

