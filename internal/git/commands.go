package git

import (
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
