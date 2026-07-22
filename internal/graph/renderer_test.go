package graph

import (
	"testing"

	"github.com/Wian47/GitSketch/internal/git"
)

func commit(hash string, parents ...string) git.Commit {
	return git.Commit{Hash: hash, Parents: parents}
}

func TestBuildGraphEmpty(t *testing.T) {
	rows := BuildGraph(nil)
	if rows != nil {
		t.Fatalf("expected nil rows for empty input, got %v", rows)
	}
}

func TestBuildGraphLinearHistory(t *testing.T) {
	// Newest first, as git log --topo-order would produce.
	commits := []git.Commit{
		commit("C3", "C2"),
		commit("C2", "C1"),
		commit("C1"), // root
	}

	rows := BuildGraph(commits)
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows for linear history, got %d", len(rows))
	}

	for i, want := range []string{"C3", "C2", "C1"} {
		if rows[i].Commit == nil || rows[i].Commit.Hash != want {
			t.Fatalf("row %d: expected commit %s, got %+v", i, want, rows[i].Commit)
		}
		if rows[i].Column != 0 {
			t.Fatalf("row %d: expected column 0, got %d", i, rows[i].Column)
		}
		if got := RenderRow(rows[i]); got != "*" {
			t.Fatalf("row %d: expected single node glyph \"*\", got %q", i, got)
		}
	}
}

// TestBuildGraphMergeAndForkBase pins the exact lane layout for a merge
// commit fanning out to two parents that share a common base:
//
//	MERGE (parents: M1, B1)
//	M1 (parent: BASE)
//	B1 (parent: BASE)
//	BASE (root)
func TestBuildGraphMergeAndForkBase(t *testing.T) {
	commits := []git.Commit{
		commit("MERGE", "M1", "B1"),
		commit("M1", "BASE"),
		commit("B1", "BASE"),
		commit("BASE"),
	}

	rows := BuildGraph(commits)
	if len(rows) != 5 {
		t.Fatalf("expected 5 rows (4 commits + 1 fork connector), got %d", len(rows))
	}

	// Row 0: the merge commit itself, alone in column 0.
	if rows[0].Commit == nil || rows[0].Commit.Hash != "MERGE" {
		t.Fatalf("row 0: expected MERGE commit, got %+v", rows[0])
	}
	if got := RenderRow(rows[0]); got != "*" {
		t.Fatalf("row 0: expected \"*\", got %q", got)
	}

	// Row 1: connector row where the merge commit's second parent forks out
	// into a new lane to the right.
	if rows[1].Commit != nil || rows[1].Column != -1 {
		t.Fatalf("row 1: expected connector row, got %+v", rows[1])
	}
	if got := RenderRow(rows[1]); got != "├ ╮" {
		t.Fatalf("row 1: expected fork connector \"├ ╮\", got %q", got)
	}

	// Row 2: M1, in lane 0, with B1's lane still alive alongside it.
	if rows[2].Commit == nil || rows[2].Commit.Hash != "M1" {
		t.Fatalf("row 2: expected M1, got %+v", rows[2])
	}
	if got := RenderRow(rows[2]); got != "* │" {
		t.Fatalf("row 2: expected \"* │\", got %q", got)
	}

	// Row 3: B1, in lane 1.
	if rows[3].Commit == nil || rows[3].Commit.Hash != "B1" {
		t.Fatalf("row 3: expected B1, got %+v", rows[3])
	}
	if rows[3].Column != 1 {
		t.Fatalf("row 3: expected column 1, got %d", rows[3].Column)
	}
	if got := RenderRow(rows[3]); got != "│ *" {
		t.Fatalf("row 3: expected \"│ *\", got %q", got)
	}

	// Row 4: BASE, where both lanes converge back into one.
	if rows[4].Commit == nil || rows[4].Commit.Hash != "BASE" {
		t.Fatalf("row 4: expected BASE, got %+v", rows[4])
	}
	if got := RenderRow(rows[4]); got != "* ╯" {
		t.Fatalf("row 4: expected convergence \"* ╯\", got %q", got)
	}

	if max := MaxColumns(rows); max != 2 {
		t.Fatalf("expected MaxColumns 2, got %d", max)
	}
}

func TestBuildGraphOctopusMerge(t *testing.T) {
	// A single commit with three parents (an "octopus" merge).
	commits := []git.Commit{
		commit("OCTOPUS", "P1", "P2", "P3"),
		commit("P1"),
		commit("P2"),
		commit("P3"),
	}

	rows := BuildGraph(commits)

	var octopusRow GraphRow
	found := false
	for _, r := range rows {
		if r.Commit != nil && r.Commit.Hash == "OCTOPUS" {
			octopusRow = r
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected to find OCTOPUS commit row")
	}
	if octopusRow.Column != 0 {
		t.Fatalf("expected OCTOPUS in column 0, got %d", octopusRow.Column)
	}

	// All three parent commits must appear somewhere in the output, each in
	// its own distinct column, with no panics/out-of-range accesses.
	seenCols := map[int]bool{}
	for _, r := range rows {
		if r.Commit == nil {
			continue
		}
		if r.Commit.Hash == "P1" || r.Commit.Hash == "P2" || r.Commit.Hash == "P3" {
			seenCols[r.Column] = true
		}
	}
	if len(seenCols) != 3 {
		t.Fatalf("expected 3 distinct parent columns, got %d (%v)", len(seenCols), seenCols)
	}
}

func TestBuildGraphMultipleRoots(t *testing.T) {
	// Two completely independent linear histories (e.g. unrelated root
	// commits), which should each get their own column with no crossover.
	commits := []git.Commit{
		commit("A2", "A1"),
		commit("A1"),
		commit("B2", "B1"),
		commit("B1"),
	}

	rows := BuildGraph(commits)
	if len(rows) != 4 {
		t.Fatalf("expected 4 rows, got %d", len(rows))
	}
	for _, r := range rows {
		if r.Commit == nil {
			t.Fatalf("did not expect connector rows for disjoint histories, got %+v", r)
		}
	}
}

func TestRenderRowEmptyCells(t *testing.T) {
	row := GraphRow{Cells: nil}
	if got := RenderRow(row); got != "" {
		t.Fatalf("expected empty string for no cells, got %q", got)
	}
}
