// internal/config/config.go
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// KeyMap holds the user-overridable primary key bindings. Vim-style
// alternates (k/j, Home/End, etc.) always remain available in the TUI
// regardless of these overrides — only the primary binding is configurable.
type KeyMap struct {
	Up       string `toml:"up"`
	Down     string `toml:"down"`
	Top      string `toml:"top"`
	Bottom   string `toml:"bottom"`
	PageUp   string `toml:"page_up"`
	PageDown string `toml:"page_down"`
	Enter    string `toml:"enter"`
	Checkout string `toml:"checkout"`
	Filter   string `toml:"filter"`
	Branch   string `toml:"branch"`
	Quit     string `toml:"quit"`
}

// Config is the full set of user-configurable GitSketch settings.
type Config struct {
	KeyMap KeyMap `toml:"keymap"`
	Theme  Theme  `toml:"theme"`
}

// DefaultKeyMap returns the built-in key bindings, matching GitSketch's
// behavior with no config file present.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: "up", Down: "down", Top: "g", Bottom: "G",
		PageUp: "pgup", PageDown: "pgdown", Enter: "enter",
		Checkout: "c", Filter: "/", Branch: "b", Quit: "q",
	}
}

// Theme holds every color GitSketch renders with, as hex strings. There is
// no theme switcher yet — this only makes the palette data instead of
// hardcoded, so a user's config file can override individual colors.
type Theme struct {
	BranchColors []string `toml:"branch_colors"`

	GraphLine     string `toml:"graph_line"`
	SelectedRowBg string `toml:"selected_row_bg"`

	Hash    string `toml:"hash"`
	Author  string `toml:"author"`
	Date    string `toml:"date"`
	Subject string `toml:"subject"`
	Body    string `toml:"body"`

	BranchRefFg string `toml:"branch_ref_fg"`
	BranchRefBg string `toml:"branch_ref_bg"`
	TagRefFg    string `toml:"tag_ref_fg"`
	TagRefBg    string `toml:"tag_ref_bg"`
	HeadRefFg   string `toml:"head_ref_fg"`
	HeadRefBg   string `toml:"head_ref_bg"`

	PaneBorder string `toml:"pane_border"`
	Title      string `toml:"title"`

	HelpBarFg string `toml:"help_bar_fg"`
	HelpBarBg string `toml:"help_bar_bg"`

	Success string `toml:"success"`
	Error   string `toml:"error"`

	Modified string `toml:"modified"`
	Added    string `toml:"added"`
	Deleted  string `toml:"deleted"`

	Dim  string `toml:"dim"`
	Logo string `toml:"logo"`

	DetailLabel   string `toml:"detail_label"`
	DetailValue   string `toml:"detail_value"`
	SectionHeader string `toml:"section_header"`
}

// DefaultTheme returns GitSketch's built-in color palette.
func DefaultTheme() Theme {
	return Theme{
		BranchColors: []string{
			"#B388FF", "#69F0AE", "#FFD54F", "#4FC3F7",
			"#FF8A65", "#80CBC4", "#F48FB1", "#AED581",
		},
		GraphLine:     "#546E7A",
		SelectedRowBg: "#1A237E",
		Hash:          "#FFD54F",
		Author:        "#80CBC4",
		Date:          "#90A4AE",
		Subject:       "#ECEFF1",
		Body:          "#B0BEC5",
		BranchRefFg:   "#1B5E20",
		BranchRefBg:   "#69F0AE",
		TagRefFg:      "#BF360C",
		TagRefBg:      "#FF8A65",
		HeadRefFg:     "#311B92",
		HeadRefBg:     "#B388FF",
		PaneBorder:    "#37474F",
		Title:         "#E0E0E0",
		HelpBarFg:     "#78909C",
		HelpBarBg:     "#263238",
		Success:       "#00E676",
		Error:         "#FF5252",
		Modified:      "#FFD54F",
		Added:         "#69F0AE",
		Deleted:       "#FF5252",
		Dim:           "#546E7A",
		Logo:          "#B388FF",
		DetailLabel:   "#78909C",
		DetailValue:   "#ECEFF1",
		SectionHeader: "#B388FF",
	}
}

// Default returns the built-in configuration.
func Default() Config {
	return Config{KeyMap: DefaultKeyMap(), Theme: DefaultTheme()}
}

// Path returns the resolved path to the user's config file:
// <os.UserConfigDir()>/gitsketch/config.toml.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gitsketch", "config.toml"), nil
}

// Load returns the effective configuration: built-in defaults merged with
// the user's config file, if one exists and parses. Load never fails: a
// missing file silently yields the defaults; a malformed file also yields
// the defaults, plus a non-empty warning describing the problem for the
// caller to surface however it likes (e.g. print to stderr).
func Load() (cfg Config, warning string) {
	cfg = Default()

	path, err := Path()
	if err != nil {
		return cfg, ""
	}

	if _, err := os.Stat(path); err != nil {
		return cfg, ""
	}

	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return Default(), fmt.Sprintf("config: failed to parse %s: %v (using defaults)", path, err)
	}

	return cfg, ""
}
