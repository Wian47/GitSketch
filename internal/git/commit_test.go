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
