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
