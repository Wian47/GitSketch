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
