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
