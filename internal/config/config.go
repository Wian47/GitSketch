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

// Default returns the built-in configuration.
func Default() Config {
	return Config{KeyMap: DefaultKeyMap()}
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
