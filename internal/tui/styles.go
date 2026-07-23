package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"

	"github.com/Wian47/GitSketch/internal/config"
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

	// StatusBarStyle styles the top status bar line.
	StatusBarStyle lipgloss.Style

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
	ApplyTheme(config.DefaultTheme())
}

// ApplyTheme rebuilds every package-level style/color var from t. Called
// once at startup with the built-in defaults (via init), and again with the
// user's loaded config if they've customized any colors.
func ApplyTheme(t config.Theme) {
	BranchColors = nil
	for _, hex := range t.BranchColors {
		BranchColors = append(BranchColors, lipgloss.Color(hex))
	}

	GraphLineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.GraphLine))

	SelectedRowStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(t.SelectedRowBg)).
		Bold(true)

	NormalRowStyle = lipgloss.NewStyle()

	HashStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Hash))
	AuthorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Author))
	DateStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Date)).Italic(true)
	SubjectStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Subject))
	BodyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Body))

	BranchRefStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.BranchRefFg)).
		Background(lipgloss.Color(t.BranchRefBg)).
		Bold(true).
		Padding(0, 1)

	TagRefStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.TagRefFg)).
		Background(lipgloss.Color(t.TagRefBg)).
		Bold(true).
		Padding(0, 1)

	HeadRefStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.HeadRefFg)).
		Background(lipgloss.Color(t.HeadRefBg)).
		Bold(true).
		Padding(0, 1)

	PaneBorderColor = lipgloss.Color(t.PaneBorder)

	TitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Title)).
		Bold(true).
		Padding(0, 1)

	HelpBarStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.HelpBarFg)).
		Background(lipgloss.Color(t.HelpBarBg))

	StatusBarStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.StatusBarFg)).
		Background(lipgloss.Color(t.StatusBarBg))

	NotifySuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Success)).Bold(true)
	NotifyErrorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Error)).Bold(true)

	FileModifiedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Modified)).Bold(true)
	FileAddedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Added)).Bold(true)
	FileDeletedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Deleted)).Bold(true)

	DimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Dim))
	LogoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Logo)).Bold(true)

	DetailLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.DetailLabel)).Bold(true)
	DetailValueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.DetailValue))

	SectionHeaderStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.SectionHeader)).
		Bold(true).
		Underline(true)
}
