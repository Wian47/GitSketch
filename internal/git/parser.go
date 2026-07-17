package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Commit represents a single commit in the Git DAG.
type Commit struct {
	Hash      string
	ShortHash string
	Parents   []string
	Author    string
	Email     string
	Date      time.Time
	RelDate   string   // e.g. "3 days ago"
	Subject   string   // first line of commit message
	Body      string   // full commit message body
	Refs      []string // branch/tag names decorating this commit
}

// FileChange represents a single file changed in a commit.
type FileChange struct {
	Status string // M, A, D, R, C
	Path   string
}

const (
	recordSep = "\x1e"
	unitSep   = "\x1f"
)

// ParseLog runs `git log` with a custom format and parses the output into
// a slice of Commit structs representing the full DAG in topological order.
// Returns an empty slice (not an error) for repositories with no commits.
func ParseLog() ([]Commit, error) {
	format := "%H" + unitSep +
		"%P" + unitSep +
		"%an" + unitSep +
		"%ae" + unitSep +
		"%aI" + unitSep +
		"%ar" + unitSep +
		"%s" + unitSep +
		"%b" + unitSep +
		"%D" + recordSep

	cmd := exec.Command("git", "log", "--all", "--topo-order", fmt.Sprintf("--format=%s", format))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// An empty repository has no commits; `git log` exits non-zero in
		// that case. Detect it gracefully by checking for a known message.
		if strings.Contains(stderr.String(), "does not have any commits") ||
			strings.Contains(stderr.String(), "bad default revision") {
			return []Commit{}, nil
		}
		return nil, fmt.Errorf("git log: %w: %s", err, stderr.String())
	}

	raw := strings.TrimSpace(stdout.String())
	if raw == "" {
		return []Commit{}, nil
	}

	records := strings.Split(raw, recordSep)
	commits := make([]Commit, 0, len(records))

	for _, rec := range records {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}

		fields := strings.SplitN(rec, unitSep, 9)
		if len(fields) < 9 {
			// Pad missing fields with empty strings so indexing is safe.
			padded := make([]string, 9)
			copy(padded, fields)
			fields = padded
		}

		hash := strings.TrimSpace(fields[0])
		if hash == "" {
			continue
		}

		shortHash := hash
		if len(shortHash) > 7 {
			shortHash = shortHash[:7]
		}

		// Parents are space-separated full hashes.
		var parents []string
		parentStr := strings.TrimSpace(fields[1])
		if parentStr != "" {
			for _, p := range strings.Fields(parentStr) {
				parents = append(parents, p)
			}
		}

		date, _ := time.Parse(time.RFC3339, strings.TrimSpace(fields[4]))

		refs := parseRefs(strings.TrimSpace(fields[8]))

		c := Commit{
			Hash:      hash,
			ShortHash: shortHash,
			Parents:   parents,
			Author:    strings.TrimSpace(fields[2]),
			Email:     strings.TrimSpace(fields[3]),
			Date:      date,
			RelDate:   strings.TrimSpace(fields[5]),
			Subject:   strings.TrimSpace(fields[6]),
			Body:      strings.TrimSpace(fields[7]),
			Refs:      refs,
		}
		commits = append(commits, c)
	}

	return commits, nil
}

// parseRefs takes a git decoration string (e.g. "HEAD -> main, origin/main, tag: v1.0")
// and returns a cleaned slice of ref names.
func parseRefs(decoration string) []string {
	if decoration == "" {
		return nil
	}

	parts := strings.Split(decoration, ",")
	refs := make([]string, 0, len(parts))

	for _, part := range parts {
		ref := strings.TrimSpace(part)
		if ref == "" {
			continue
		}
		// Strip the "HEAD -> " prefix that git adds to the current branch.
		ref = strings.TrimPrefix(ref, "HEAD -> ")
		// Strip "tag: " prefix from tag decorations.
		ref = strings.TrimPrefix(ref, "tag: ")
		ref = strings.TrimSpace(ref)
		if ref != "" {
			refs = append(refs, ref)
		}
	}

	return refs
}

// GetChangedFiles returns the list of files changed in the given commit.
// For root commits (no parents) it uses --root so that initial additions
// are correctly reported.
func GetChangedFiles(hash string) ([]FileChange, error) {
	// First determine if this is a root commit (zero parents).
	parentCheck := exec.Command("git", "rev-parse", hash+"^")
	var parentErr bytes.Buffer
	parentCheck.Stderr = &parentErr
	isRoot := parentCheck.Run() != nil

	var cmd *exec.Cmd
	if isRoot {
		cmd = exec.Command("git", "diff-tree", "--root", "--no-commit-id", "-r", "--name-status", hash)
	} else {
		cmd = exec.Command("git", "diff-tree", "--no-commit-id", "-r", "--name-status", hash)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git diff-tree: %w: %s", err, stderr.String())
	}

	raw := strings.TrimSpace(stdout.String())
	if raw == "" {
		return nil, nil
	}

	lines := strings.Split(raw, "\n")
	changes := make([]FileChange, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Output is tab-separated: STATUS\tPATH
		// For renames/copies: STATUS\tOLD_PATH\tNEW_PATH — we take the new path.
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 2 {
			continue
		}

		status := strings.TrimSpace(parts[0])
		path := strings.TrimSpace(parts[len(parts)-1]) // last element handles renames

		changes = append(changes, FileChange{
			Status: status,
			Path:   path,
		})
	}

	return changes, nil
}

// IsGitRepo returns true if the current working directory is inside a
// git repository.
func IsGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}
