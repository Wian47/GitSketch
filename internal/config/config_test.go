// internal/config/config_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultKeyMapMatchesCurrentBindings(t *testing.T) {
	km := DefaultKeyMap()
	if km.Up != "up" || km.Down != "down" || km.Top != "g" || km.Bottom != "G" {
		t.Fatalf("unexpected nav defaults: %+v", km)
	}
	if km.Checkout != "c" || km.Filter != "/" || km.Branch != "b" || km.Quit != "q" {
		t.Fatalf("unexpected action defaults: %+v", km)
	}
}

func TestLoadNoFileReturnsDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfg, warning := Load()
	if warning != "" {
		t.Fatalf("expected no warning when no config file exists, got %q", warning)
	}
	if cfg.KeyMap != DefaultKeyMap() {
		t.Fatalf("expected default keymap, got %+v", cfg.KeyMap)
	}
}

func TestLoadPartialFileMergesOverDefaults(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfgDir := filepath.Join(dir, "gitsketch")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	toml := "[keymap]\nquit = \"x\"\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, warning := Load()
	if warning != "" {
		t.Fatalf("expected no warning for a valid partial file, got %q", warning)
	}
	if cfg.KeyMap.Quit != "x" {
		t.Fatalf("expected overridden Quit=\"x\", got %q", cfg.KeyMap.Quit)
	}
	if cfg.KeyMap.Up != "up" {
		t.Fatalf("expected untouched field Up to keep default %q, got %q", "up", cfg.KeyMap.Up)
	}
}

func TestLoadMalformedFileFallsBackWithWarning(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfgDir := filepath.Join(dir, "gitsketch")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte("not valid toml [["), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, warning := Load()
	if warning == "" {
		t.Fatal("expected a non-empty warning for a malformed config file")
	}
	if cfg.KeyMap != DefaultKeyMap() {
		t.Fatalf("expected defaults on malformed file, got %+v", cfg.KeyMap)
	}
}

func TestDefaultThemeHasAllBranchColors(t *testing.T) {
	theme := DefaultTheme()
	if len(theme.BranchColors) != 8 {
		t.Fatalf("expected 8 branch colors, got %d", len(theme.BranchColors))
	}
	if theme.Hash != "#FFD54F" {
		t.Fatalf("Hash = %q, want %q", theme.Hash, "#FFD54F")
	}
}

func TestLoadMergesPartialTheme(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfgDir := filepath.Join(dir, "gitsketch")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	toml := "[theme]\nhash = \"#FF0000\"\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, warning := Load()
	if warning != "" {
		t.Fatalf("expected no warning, got %q", warning)
	}
	if cfg.Theme.Hash != "#FF0000" {
		t.Fatalf("Theme.Hash = %q, want %q", cfg.Theme.Hash, "#FF0000")
	}
	if cfg.Theme.Dim != DefaultTheme().Dim {
		t.Fatalf("expected untouched Theme.Dim to keep default, got %q", cfg.Theme.Dim)
	}
}
