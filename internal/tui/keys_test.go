// internal/tui/keys_test.go
package tui

import (
	"testing"

	"github.com/Wian47/GitSketch/internal/config"
)

func TestApplyKeyMapOverridesBindings(t *testing.T) {
	t.Cleanup(func() { ApplyKeyMap(config.DefaultKeyMap()) })

	ApplyKeyMap(config.KeyMap{Quit: "x", Up: "w"})

	if KeyQ != "x" {
		t.Fatalf("KeyQ = %q, want %q", KeyQ, "x")
	}
	if KeyUp != "w" {
		t.Fatalf("KeyUp = %q, want %q", KeyUp, "w")
	}
	// Fields left empty in the override must keep their previous value.
	if KeyDown != "down" {
		t.Fatalf("KeyDown = %q, want unchanged default %q", KeyDown, "down")
	}
}

func TestApplyKeyMapEmptyFieldsLeaveDefaultsUntouched(t *testing.T) {
	t.Cleanup(func() { ApplyKeyMap(config.DefaultKeyMap()) })

	ApplyKeyMap(config.KeyMap{})

	if KeyUp != "up" || KeyDown != "down" || KeyC != "c" || KeyFilter != "/" || KeyBranch != "b" {
		t.Fatal("expected an all-empty KeyMap to leave every binding at its default")
	}
}
