// internal/tui/help.go
package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// HelpEntry is one row of the help overlay: the key(s) bound to an action
// and a short description of what it does.
type HelpEntry struct {
	Keys string
	Desc string
}

// helpEntries lists every global keybinding, built from the live Key* vars
// so it always reflects the user's configured keymap, not just the
// built-in defaults.
func helpEntries() []HelpEntry {
	return []HelpEntry{
		{fmt.Sprintf("%s/%s", KeyUp, KeyK), "Move up"},
		{fmt.Sprintf("%s/%s", KeyDown, KeyJ), "Move down"},
		{fmt.Sprintf("%s/%s", KeyG, KeyShiftG), "Jump to top/bottom"},
		{fmt.Sprintf("%s/%s", KeyPgUp, KeyPgDown), "Page up/down"},
		{KeyEnter, "View fullscreen diff / hunk-stage working-tree file"},
		{KeyFilter, "Filter commits (regex)"},
		{KeyBranch, "Branch manager"},
		{KeyStageFile, "Stage focused working-tree file"},
		{KeyUnstageFile, "Unstage focused working-tree file"},
		{KeyDiscard, "Discard changes to focused file (with confirm)"},
		{KeyC, "Checkout selected commit / commit staged changes"},
		{KeyHelp, "Toggle this help"},
		{KeyQ, "Quit"},
	}
}

// renderHelpOverlay renders the full-screen keybinding reference.
func (m Model) renderHelpOverlay() string {
	var lines []string
	lines = append(lines, SectionHeaderStyle.Render("Keybindings"))
	lines = append(lines, "")

	for _, e := range helpEntries() {
		row := DetailLabelStyle.Render(fmt.Sprintf("  %-8s", e.Keys)) + DimStyle.Render(e.Desc)
		lines = append(lines, row)
	}

	lines = append(lines, "")
	lines = append(lines, DimStyle.Render(fmt.Sprintf("Press %s or esc to close", KeyHelp)))

	content := strings.Join(lines, "\n")
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}
