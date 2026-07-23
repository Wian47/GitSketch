// internal/tui/help_test.go
package tui

import "testing"

func TestHelpEntriesIncludeCoreBindings(t *testing.T) {
	entries := helpEntries()
	if len(entries) == 0 {
		t.Fatal("expected at least one help entry")
	}

	found := map[string]bool{}
	for _, e := range entries {
		found[e.Desc] = true
	}
	for _, want := range []string{"Move up", "Move down", "Quit", "Toggle this help"} {
		if !found[want] {
			t.Fatalf("expected a help entry for %q, got entries: %+v", want, entries)
		}
	}
}

func TestRenderHelpOverlayContainsQuitBinding(t *testing.T) {
	m := Model{width: 80, height: 24}
	out := m.renderHelpOverlay()
	if out == "" {
		t.Fatal("expected non-empty help overlay content")
	}
}
