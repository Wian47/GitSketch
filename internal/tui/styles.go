package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// ─── Color Palette ──────────────────────────────────────────────────────────

// BranchColors are cycled through for different branch lanes in the graph.
var BranchColors []color.Color

// ─── Styles ─────────────────────────────────────────────────────────────────

var (
	// GraphLineStyle for '│' continuation lines in the graph.
	GraphLineStyle lipgloss.Style

	// SelectedRowStyle highlights the currently selected commit row.
	SelectedRowStyle lipgloss.Style

	// NormalRowStyle is the default (non-selected) row style.
	NormalRowStyle lipgloss.Style

	// HashStyle styles commit hashes (short form).
	HashStyle lipgloss.Style

	// AuthorStyle styles author names.
	AuthorStyle lipgloss.Style

	// DateStyle styles relative dates.
	DateStyle lipgloss.Style

	// SubjectStyle styles commit subject lines.
	SubjectStyle lipgloss.Style

	// BodyStyle styles commit message body text.
	BodyStyle lipgloss.Style

	// BranchRefStyle styles branch name badges.
	BranchRefStyle lipgloss.Style

	// TagRefStyle styles tag name badges.
	TagRefStyle lipgloss.Style

	// HeadRefStyle styles the HEAD indicator badge.
	HeadRefStyle lipgloss.Style

	// PaneBorderColor is the color used for pane borders.
	PaneBorderColor color.Color

	// TitleStyle styles pane titles.
	TitleStyle lipgloss.Style

	// HelpBarStyle styles the bottom help legend.
	HelpBarStyle lipgloss.Style

	// NotifySuccessStyle styles success notification messages.
	NotifySuccessStyle lipgloss.Style

	// NotifyErrorStyle styles error notification messages.
	NotifyErrorStyle lipgloss.Style

	// FileModifiedStyle styles 'M' (modified) file status indicators.
	FileModifiedStyle lipgloss.Style

	// FileAddedStyle styles 'A' (added) file status indicators.
	FileAddedStyle lipgloss.Style

	// FileDeletedStyle styles 'D' (deleted) file status indicators.
	FileDeletedStyle lipgloss.Style

	// DimStyle for de-emphasized text.
	DimStyle lipgloss.Style

	// LogoStyle for the application title.
	LogoStyle lipgloss.Style

	// DetailLabelStyle styles labels in the detail pane (e.g., "Author:", "Hash:").
	DetailLabelStyle lipgloss.Style

	// DetailValueStyle styles values in the detail pane.
	DetailValueStyle lipgloss.Style

	// SectionHeaderStyle styles section headers in the detail pane.
	SectionHeaderStyle lipgloss.Style
)

func init() {
	// ── Branch lane colors (vibrant, high-contrast on dark backgrounds) ──
	BranchColors = []color.Color{
		lipgloss.Color("#B388FF"), // soft purple
		lipgloss.Color("#69F0AE"), // mint green
		lipgloss.Color("#FFD54F"), // amber
		lipgloss.Color("#4FC3F7"), // sky blue
		lipgloss.Color("#FF8A65"), // coral
		lipgloss.Color("#80CBC4"), // teal
		lipgloss.Color("#F48FB1"), // pink
		lipgloss.Color("#AED581"), // lime
	}

	// ── Graph ──
	GraphLineStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#546E7A"))

	// ── Row selection ──
	SelectedRowStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("#1A237E")).
		Bold(true)

	NormalRowStyle = lipgloss.NewStyle()

	// ── Commit metadata ──
	HashStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFD54F"))

	AuthorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#80CBC4"))

	DateStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#90A4AE")).
		Italic(true)

	SubjectStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ECEFF1"))

	BodyStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#B0BEC5"))

	// ── Ref badges ──
	BranchRefStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1B5E20")).
		Background(lipgloss.Color("#69F0AE")).
		Bold(true).
		Padding(0, 1)

	TagRefStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#BF360C")).
		Background(lipgloss.Color("#FF8A65")).
		Bold(true).
		Padding(0, 1)

	HeadRefStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#311B92")).
		Background(lipgloss.Color("#B388FF")).
		Bold(true).
		Padding(0, 1)

	// ── Pane borders ──
	PaneBorderColor = lipgloss.Color("#37474F")

	TitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E0E0E0")).
		Bold(true).
		Padding(0, 1)

	// ── Help bar ──
	HelpBarStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#78909C")).
		Background(lipgloss.Color("#263238"))

	// ── Notifications ──
	NotifySuccessStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00E676")).
		Bold(true)

	NotifyErrorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF5252")).
		Bold(true)

	// ── File status ──
	FileModifiedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFD54F")).
		Bold(true)

	FileAddedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#69F0AE")).
		Bold(true)

	FileDeletedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF5252")).
		Bold(true)

	// ── Misc ──
	DimStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#546E7A"))

	LogoStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#B388FF")).
		Bold(true)

	DetailLabelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#78909C")).
		Bold(true)

	DetailValueStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ECEFF1"))

	SectionHeaderStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#B388FF")).
		Bold(true).
		Underline(true)
}
