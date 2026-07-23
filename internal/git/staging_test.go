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
