// internal/git/hunk_test.go
package git

import "testing"

const sampleDiff = `diff --git a/foo.txt b/foo.txt
index 83db48f..bf269ba 100644
--- a/foo.txt
+++ b/foo.txt
@@ -1,3 +1,4 @@
 line1
+new line
 line2
 line3
@@ -10,2 +11,3 @@
 old context
+another add
 more context
`

func TestParseHunksSplitsFileHeaderAndHunks(t *testing.T) {
	header, hunks := ParseHunks(sampleDiff)

	wantHeader := "diff --git a/foo.txt b/foo.txt\nindex 83db48f..bf269ba 100644\n--- a/foo.txt\n+++ b/foo.txt"
	if header != wantHeader {
		t.Fatalf("header = %q, want %q", header, wantHeader)
	}
	if len(hunks) != 2 {
		t.Fatalf("expected 2 hunks, got %d", len(hunks))
	}
	if hunks[0].Header != "@@ -1,3 +1,4 @@" {
		t.Fatalf("hunks[0].Header = %q", hunks[0].Header)
	}
	wantFirstLines := []string{"@@ -1,3 +1,4 @@", " line1", "+new line", " line2", " line3"}
	if len(hunks[0].Lines) != len(wantFirstLines) {
		t.Fatalf("hunks[0].Lines = %v, want %v", hunks[0].Lines, wantFirstLines)
	}
	for i, l := range wantFirstLines {
		if hunks[0].Lines[i] != l {
			t.Fatalf("hunks[0].Lines[%d] = %q, want %q", i, hunks[0].Lines[i], l)
		}
	}
	if hunks[1].Header != "@@ -10,2 +11,3 @@" {
		t.Fatalf("hunks[1].Header = %q", hunks[1].Header)
	}
}

func TestParseHunksEmptyDiff(t *testing.T) {
	header, hunks := ParseHunks("")
	if header != "" || hunks != nil {
		t.Fatalf("expected empty header and nil hunks for empty diff, got header=%q hunks=%v", header, hunks)
	}
}

func TestParseHunksNoHunksJustHeader(t *testing.T) {
	// e.g. a mode-only change or empty file add with no content hunks.
	diff := "diff --git a/empty.txt b/empty.txt\nnew file mode 100644\n"
	header, hunks := ParseHunks(diff)
	want := "diff --git a/empty.txt b/empty.txt\nnew file mode 100644"
	if header != want {
		t.Fatalf("header = %q, want %q", header, want)
	}
	if len(hunks) != 0 {
		t.Fatalf("expected 0 hunks, got %d", len(hunks))
	}
}

func TestHunkPatchReconstructsApplicablePatch(t *testing.T) {
	header, hunks := ParseHunks(sampleDiff)
	patch := hunks[0].Patch(header)

	want := "diff --git a/foo.txt b/foo.txt\n" +
		"index 83db48f..bf269ba 100644\n" +
		"--- a/foo.txt\n" +
		"+++ b/foo.txt\n" +
		"@@ -1,3 +1,4 @@\n" +
		" line1\n" +
		"+new line\n" +
		" line2\n" +
		" line3\n"
	if patch != want {
		t.Fatalf("Patch() = %q, want %q", patch, want)
	}
}
