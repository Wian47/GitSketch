// internal/tui/statusbar.go
package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// renderStatusBar renders the single-line always-visible status bar:
// branch name, ahead/behind counts (omitted when both are zero), and
// staged/unstaged dirty-file counts (or "clean" when there are none).
func renderStatusBar(width int, branch string, ahead, behind, staged, unstaged int) string {
	left := fmt.Sprintf(" %s", branch)
	if ahead > 0 {
		left += fmt.Sprintf(" ↑%d", ahead)
	}
	if behind > 0 {
		left += fmt.Sprintf(" ↓%d", behind)
	}

	var right string
	if staged > 0 || unstaged > 0 {
		right = fmt.Sprintf("%d staged, %d unstaged ", staged, unstaged)
	} else {
		right = "clean "
	}

	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}

	line := left + strings.Repeat(" ", gap) + right
	return StatusBarStyle.Width(width).Render(line)
}
