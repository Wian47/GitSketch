package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// CheckoutResult holds the outcome of a git checkout operation.
type CheckoutResult struct {
	Hash    string
	Success bool
	Message string
}

// Checkout runs "git checkout <hash>" and returns a CheckoutResult
// describing whether the operation succeeded.
func Checkout(hash string) CheckoutResult {
	cmd := exec.Command("git", "checkout", hash)

	// CombinedOutput captures both stdout and stderr.
	output, err := cmd.CombinedOutput()
	msg := strings.TrimSpace(string(output))

	if err != nil {
		// If the command failed and produced no output, use the Go error message.
		if msg == "" {
			msg = err.Error()
		}
		return CheckoutResult{
			Hash:    hash,
			Success: false,
			Message: msg,
		}
	}

	return CheckoutResult{
		Hash:    hash,
		Success: true,
		Message: msg,
	}
}

// GetCurrentBranch returns the name of the current branch.
// If HEAD is detached it falls back to the short commit hash.
func GetCurrentBranch() (string, error) {
	// Try to resolve the symbolic ref first (works on a normal branch).
	cmd := exec.Command("git", "symbolic-ref", "--short", "HEAD")
	output, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output)), nil
	}

	// Detached HEAD — fall back to the short hash.
	cmd = exec.Command("git", "rev-parse", "--short", "HEAD")
	output, err = cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// GetCommitDiff retrieves the full unified diff of a commit.
func GetCommitDiff(hash string) (string, error) {
	cmd := exec.Command("git", "show", "--color=never", "--patch", hash)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git show: %w: %s", err, string(output))
	}
	return string(output), nil
}

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

// CreateBranch creates a new branch pointing to the specified commit.
func CreateBranch(name, hash string) error {
	cmd := exec.Command("git", "branch", name, hash)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(output)))
	}
	return nil
}

// DeleteBranch deletes the specified branch. If force is true, it uses -D.
func DeleteBranch(name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	cmd := exec.Command("git", "branch", flag, name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(output)))
	}
	return nil
}
