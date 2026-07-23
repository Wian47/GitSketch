package tui

import "testing"

func TestRenderStatusBarShowsBranchAndAheadBehind(t *testing.T) {
	out := renderStatusBar(80, "main", 2, 1, 0, 0)
	for _, want := range []string{"main", "↑2", "↓1"} {
		if !contains(out, want) {
			t.Fatalf("expected status bar to contain %q, got %q", want, out)
		}
	}
}

func TestRenderStatusBarCleanState(t *testing.T) {
	out := renderStatusBar(80, "main", 0, 0, 0, 0)
	if !contains(out, "clean") {
		t.Fatalf("expected status bar to show \"clean\" with no dirty files, got %q", out)
	}
	if contains(out, "↑") || contains(out, "↓") {
		t.Fatalf("expected no ahead/behind markers at 0/0, got %q", out)
	}
}

func TestRenderStatusBarDirtyCounts(t *testing.T) {
	out := renderStatusBar(80, "main", 0, 0, 3, 2)
	if !contains(out, "3 staged, 2 unstaged") {
		t.Fatalf("expected dirty counts in status bar, got %q", out)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (needle == "" || indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
