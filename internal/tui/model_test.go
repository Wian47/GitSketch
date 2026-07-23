package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Wian47/GitSketch/internal/git"
)

func sampleCommits() []git.Commit {
	return []git.Commit{
		{Hash: "h1", ShortHash: "h1", Subject: "add login page", Author: "Alice", Refs: []string{"main"}},
		{Hash: "h2", ShortHash: "h2", Subject: "fix bug in parser", Author: "Bob"},
		{Hash: "h3", ShortHash: "h3", Subject: "update README", Author: "Alice"},
		{Hash: "h4", ShortHash: "h4", Subject: "release v1.0", Author: "Carol", Refs: []string{"v1.0"}},
	}
}

func TestFilterMatcherRegex(t *testing.T) {
	matches := filterMatcher(`^fix.*parser$`)
	if !matches("fix bug in parser") {
		t.Fatal("expected regex to match")
	}
	if matches("add login page") {
		t.Fatal("expected regex not to match unrelated subject")
	}
}

func TestFilterMatcherCaseInsensitive(t *testing.T) {
	matches := filterMatcher("LOGIN")
	if !matches("add login page") {
		t.Fatal("expected case-insensitive match")
	}
}

func TestFilterMatcherInvalidRegexFallsBackToSubstring(t *testing.T) {
	// "(" alone is an invalid/incomplete regex, but should still work as a
	// literal substring so the filter doesn't go dead mid-type.
	matches := filterMatcher("(")
	if matches("no parens here") {
		t.Fatal("expected no match for literal \"(\" substring")
	}
	if !matches("has (a paren") {
		t.Fatal("expected literal \"(\" substring match as regex fallback")
	}
}

func TestApplyFilterEmptyQueryReturnsAll(t *testing.T) {
	m := Model{allCommits: sampleCommits()}
	m.applyFilter()
	if len(m.filteredCommits) != 4 {
		t.Fatalf("expected 4 commits with empty query, got %d", len(m.filteredCommits))
	}
}

func TestApplyFilterBySubject(t *testing.T) {
	m := Model{allCommits: sampleCommits(), searchQuery: "parser"}
	m.applyFilter()
	if len(m.filteredCommits) != 1 || m.filteredCommits[0].Hash != "h2" {
		t.Fatalf("expected only h2 to match \"parser\", got %+v", m.filteredCommits)
	}
}

func TestApplyFilterByRef(t *testing.T) {
	m := Model{allCommits: sampleCommits(), searchQuery: "v1.0"}
	m.applyFilter()
	if len(m.filteredCommits) != 1 || m.filteredCommits[0].Hash != "h4" {
		t.Fatalf("expected only h4 to match ref \"v1.0\", got %+v", m.filteredCommits)
	}
}

func TestApplyFilterByAuthorRegex(t *testing.T) {
	m := Model{allCommits: sampleCommits(), searchQuery: "^Alice$"}
	m.applyFilter()
	if len(m.filteredCommits) != 2 {
		t.Fatalf("expected 2 commits authored by Alice, got %d", len(m.filteredCommits))
	}
}

func TestApplyFilterClampsCursor(t *testing.T) {
	m := Model{allCommits: sampleCommits(), cursor: 3}
	m.searchQuery = "parser" // only 1 result remains
	m.applyFilter()
	if m.cursor != 0 {
		t.Fatalf("expected cursor to clamp to 0 for single-result filter, got %d", m.cursor)
	}
}

func TestMoveCursorClampsToBounds(t *testing.T) {
	m := Model{allCommits: sampleCommits(), height: 20}
	m.applyFilter()

	m.moveCursor(-5)
	if m.cursor != 0 {
		t.Fatalf("expected cursor clamped to 0, got %d", m.cursor)
	}

	m.moveCursor(100)
	if m.cursor != 3 {
		t.Fatalf("expected cursor clamped to last index 3, got %d", m.cursor)
	}
}

func TestSelectedCommitReturnsCorrectCommit(t *testing.T) {
	m := Model{allCommits: sampleCommits(), height: 20}
	m.applyFilter()
	m.cursor = 2

	c := m.selectedCommit()
	if c == nil || c.Hash != "h3" {
		t.Fatalf("expected selected commit h3, got %+v", c)
	}
}

func TestSelectedCommitNilWhenEmpty(t *testing.T) {
	m := Model{}
	m.applyFilter()
	if c := m.selectedCommit(); c != nil {
		t.Fatalf("expected nil selected commit for empty history, got %+v", c)
	}
}

// TestCommitsParsedMsgResetsCursorOnlyOnFirstLoad verifies that the initial
// load resets the cursor to the top, but subsequent refreshes (as happen
// after checkout/branch actions) preserve wherever the user had navigated to,
// since those operations only change ref decorations, not commit order.
func TestCommitsParsedMsgResetsCursorOnlyOnFirstLoad(t *testing.T) {
	m := Model{height: 20}

	updated, _ := m.Update(commitsParsedMsg{commits: sampleCommits()})
	m = updated.(Model)
	if m.cursor != 0 {
		t.Fatalf("expected cursor 0 after first load, got %d", m.cursor)
	}

	m.cursor = 2 // simulate user navigating down

	updated, _ = m.Update(commitsParsedMsg{commits: sampleCommits()})
	m = updated.(Model)
	if m.cursor != 2 {
		t.Fatalf("expected cursor to stay at 2 after refresh, got %d", m.cursor)
	}
}

func TestFilterDebounceMsgIgnoresStaleGeneration(t *testing.T) {
	m := Model{allCommits: sampleCommits(), height: 20}
	m.applyFilter()

	m.searchQuery = "parser"
	m.filterGen = 5 // a newer keystroke has already moved the generation on

	// A stale tick tagged with an older generation should be ignored.
	updated, _ := m.Update(filterDebounceMsg{gen: 3})
	m = updated.(Model)
	if len(m.filteredCommits) != 4 {
		t.Fatalf("expected stale debounce tick to be ignored, got %d filtered commits", len(m.filteredCommits))
	}

	// A tick matching the current generation should apply the filter.
	updated, _ = m.Update(filterDebounceMsg{gen: 5})
	m = updated.(Model)
	if len(m.filteredCommits) != 1 {
		t.Fatalf("expected current-generation debounce tick to apply filter, got %d filtered commits", len(m.filteredCommits))
	}
}

func TestVisibleRowsNeverBelowOne(t *testing.T) {
	m := Model{height: 2}
	if got := m.visibleRows(); got < 1 {
		t.Fatalf("expected visibleRows >= 1, got %d", got)
	}
}

// TestFilterDebounceCmdFiresWithGeneration ensures the scheduled command
// actually produces a filterDebounceMsg carrying the generation it was
// created with, once its delay elapses.
func TestFilterDebounceCmdFiresWithGeneration(t *testing.T) {
	cmd := filterDebounceCmd(7)
	if cmd == nil {
		t.Fatal("expected a non-nil command")
	}

	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()

	select {
	case msg := <-done:
		fdm, ok := msg.(filterDebounceMsg)
		if !ok {
			t.Fatalf("expected filterDebounceMsg, got %T", msg)
		}
		if fdm.gen != 7 {
			t.Fatalf("expected gen 7, got %d", fdm.gen)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for debounce command")
	}
}

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
