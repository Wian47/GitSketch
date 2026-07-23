package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// StatusEntry represents one file's status letter (M, A, D, R, C, or "??"
// for untracked) and its path.
type StatusEntry struct {
	Status string
	Path   string
}

// Status is a snapshot of the working tree: current branch, how far ahead/
// behind its upstream it is (0/0 if there's no upstream), and the staged vs
// unstaged file lists.
type Status struct {
	Branch   string
	Ahead    int
	Behind   int
	Staged   []StatusEntry
	Unstaged []StatusEntry
}

// GetStatus runs `git status --porcelain=v2 --branch -z` and parses the
// working tree state. The -z form NUL-delimits every record (including the
// "# branch.*" header lines), which sidesteps all quoting/escaping rules
// that apply to filenames in the non-z porcelain format.
func GetStatus() (Status, error) {
	cmd := exec.Command("git", "status", "--porcelain=v2", "--branch", "-z")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return Status{}, fmt.Errorf("git status: %w: %s", err, stderr.String())
	}

	var st Status
	records := strings.Split(stdout.String(), "\x00")

	for i := 0; i < len(records); i++ {
		rec := records[i]
		switch {
		case rec == "":
			continue
		case strings.HasPrefix(rec, "# branch.head "):
			st.Branch = strings.TrimPrefix(rec, "# branch.head ")
		case strings.HasPrefix(rec, "# branch.ab "):
			ab := strings.TrimPrefix(rec, "# branch.ab ")
			parts := strings.Fields(ab) // e.g. ["+2", "-1"]
			if len(parts) == 2 {
				st.Ahead, _ = strconv.Atoi(strings.TrimPrefix(parts[0], "+"))
				st.Behind, _ = strconv.Atoi(strings.TrimPrefix(parts[1], "-"))
			}
		case strings.HasPrefix(rec, "1 "):
			// "1 XY sub mH mI mW hH hI path"
			fields := strings.SplitN(rec, " ", 9)
			if len(fields) == 9 {
				addStatusEntry(&st, fields[1], fields[8])
			}
		case strings.HasPrefix(rec, "2 "):
			// "2 XY sub mH mI mW hH hI Xscore path" — followed by a second
			// NUL-terminated record holding the original path, which we
			// don't need for status display, so just skip past it.
			fields := strings.SplitN(rec, " ", 10)
			if len(fields) == 10 {
				addStatusEntry(&st, fields[1], fields[9])
				i++
			}
		case strings.HasPrefix(rec, "? "):
			st.Unstaged = append(st.Unstaged, StatusEntry{
				Status: "??",
				Path:   strings.TrimPrefix(rec, "? "),
			})
		}
	}

	return st, nil
}

// addStatusEntry splits a porcelain v2 XY status pair into a staged entry
// (from X) and/or an unstaged entry (from Y). '.' means "no change in that
// slot" and is skipped.
func addStatusEntry(st *Status, xy, path string) {
	if len(xy) != 2 {
		return
	}
	x, y := xy[0], xy[1]
	if x != '.' {
		st.Staged = append(st.Staged, StatusEntry{Status: string(x), Path: path})
	}
	if y != '.' {
		st.Unstaged = append(st.Unstaged, StatusEntry{Status: string(y), Path: path})
	}
}
