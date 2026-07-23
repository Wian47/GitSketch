// internal/git/hunk.go
package git

import "strings"

// Hunk represents a single @@ ... @@ section of a unified diff for one file.
type Hunk struct {
	Header string   // the "@@ -a,b +c,d @@ ..." line
	Lines  []string // full hunk lines, including the header line itself
}

// ParseHunks splits a single-file unified diff (as produced by `git diff` or
// `git diff --cached` for one path) into its file header — everything before
// the first "@@ " line, e.g. the "diff --git"/"index"/"---"/"+++" lines —
// and its individual hunks. Returns ("", nil) for an empty diff.
func ParseHunks(diff string) (fileHeader string, hunks []Hunk) {
	diff = strings.TrimRight(diff, "\n")
	if diff == "" {
		return "", nil
	}

	lines := strings.Split(diff, "\n")

	i := 0
	var headerLines []string
	for i < len(lines) && !strings.HasPrefix(lines[i], "@@ ") {
		headerLines = append(headerLines, lines[i])
		i++
	}
	fileHeader = strings.Join(headerLines, "\n")

	for i < len(lines) {
		header := lines[i]
		hunkLines := []string{header}
		i++
		for i < len(lines) && !strings.HasPrefix(lines[i], "@@ ") {
			hunkLines = append(hunkLines, lines[i])
			i++
		}
		hunks = append(hunks, Hunk{Header: header, Lines: hunkLines})
	}
	return fileHeader, hunks
}

// Patch reconstructs a standalone, applicable patch containing only this
// hunk, prefixed with the file header. The hunk's line ranges are copied
// verbatim from the source diff, so this is safe for whole-hunk staging
// without any header recalculation. Suitable for piping to
// `git apply --cached` (see StageHunk/UnstageHunk in staging.go).
func (h Hunk) Patch(fileHeader string) string {
	var sb strings.Builder
	sb.WriteString(fileHeader)
	sb.WriteString("\n")
	for _, l := range h.Lines {
		sb.WriteString(l)
		sb.WriteString("\n")
	}
	return sb.String()
}
