package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initTestRepo creates an empty git repository in a temp dir, chdirs the
// test into it (restored automatically via t.Cleanup), and configures a
// commit identity so commits succeed regardless of the host's global config.
func initTestRepo(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)

	runGit(t, "init", "-q", "-b", "main")
	runGit(t, "config", "user.email", "test@example.com")
	runGit(t, "config", "user.name", "Test User")
}

func runGit(t *testing.T, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func writeAndCommit(t *testing.T, path, content, message string) string {
	t.Helper()
	full := filepath.Join(".", path)
	if dir := filepath.Dir(full); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, "add", path)
	runGit(t, "commit", "-q", "-m", message)
	return strings.TrimSpace(runGit(t, "rev-parse", "HEAD"))
}

func TestParseRefs(t *testing.T) {
	cases := []struct {
		name       string
		decoration string
		want       []string
	}{
		{"empty", "", nil},
		{"single branch", "main", []string{"main"}},
		{"head arrow", "HEAD -> main", []string{"main"}},
		{"head detached", "HEAD, main", []string{"HEAD", "main"}},
		{"tag", "tag: v1.0", []string{"v1.0"}},
		{"multiple", "HEAD -> main, origin/main, tag: v1.0", []string{"main", "origin/main", "v1.0"}},
		{"whitespace", "  main  ,  tag: v1.0  ", []string{"main", "v1.0"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseRefs(tc.decoration)
			if len(got) != len(tc.want) {
				t.Fatalf("parseRefs(%q) = %v, want %v", tc.decoration, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("parseRefs(%q) = %v, want %v", tc.decoration, got, tc.want)
				}
			}
		})
	}
}

func TestIsGitRepo(t *testing.T) {
	initTestRepo(t)
	if !IsGitRepo() {
		t.Fatal("expected IsGitRepo() to be true inside a git repository")
	}
}

func TestIsGitRepoFalseOutsideRepo(t *testing.T) {
	t.Chdir(t.TempDir())
	if IsGitRepo() {
		t.Fatal("expected IsGitRepo() to be false outside a git repository")
	}
}

func TestParseLogEmptyRepo(t *testing.T) {
	initTestRepo(t)

	commits, err := ParseLog()
	if err != nil {
		t.Fatalf("ParseLog() error = %v", err)
	}
	if len(commits) != 0 {
		t.Fatalf("expected 0 commits in empty repo, got %d", len(commits))
	}
}

func TestParseLogLinearHistory(t *testing.T) {
	initTestRepo(t)

	writeAndCommit(t, "a.txt", "one", "first commit")
	writeAndCommit(t, "a.txt", "two", "second commit")
	head := writeAndCommit(t, "a.txt", "three", "third commit")

	commits, err := ParseLog()
	if err != nil {
		t.Fatalf("ParseLog() error = %v", err)
	}
	if len(commits) != 3 {
		t.Fatalf("expected 3 commits, got %d", len(commits))
	}

	// Topo-order: newest first.
	if commits[0].Hash != head {
		t.Fatalf("expected newest commit first (%s), got %s", head, commits[0].Hash)
	}
	if commits[0].Subject != "third commit" {
		t.Fatalf("expected subject %q, got %q", "third commit", commits[0].Subject)
	}
	if len(commits[0].ShortHash) != 7 {
		t.Fatalf("expected 7-char short hash, got %q", commits[0].ShortHash)
	}
	if len(commits[0].Parents) != 1 || commits[0].Parents[0] != commits[1].Hash {
		t.Fatalf("expected commit 0's parent to be commit 1, got %v", commits[0].Parents)
	}
	if commits[2].Subject != "first commit" {
		t.Fatalf("expected oldest commit last, got subject %q", commits[2].Subject)
	}
	if len(commits[2].Parents) != 0 {
		t.Fatalf("expected root commit to have no parents, got %v", commits[2].Parents)
	}

	// HEAD/branch decoration should show up on the tip commit.
	foundMain := false
	for _, ref := range commits[0].Refs {
		if strings.Contains(ref, "main") {
			foundMain = true
		}
	}
	if !foundMain {
		t.Fatalf("expected branch ref containing \"main\" on tip commit, got %v", commits[0].Refs)
	}
}

func TestParseLogMergeCommit(t *testing.T) {
	initTestRepo(t)

	writeAndCommit(t, "base.txt", "base", "base commit")
	runGit(t, "checkout", "-q", "-b", "feature")
	writeAndCommit(t, "feature.txt", "feature", "feature commit")
	runGit(t, "checkout", "-q", "main")
	writeAndCommit(t, "main.txt", "main change", "main commit")
	runGit(t, "merge", "--no-ff", "-q", "-m", "merge feature", "feature")

	commits, err := ParseLog()
	if err != nil {
		t.Fatalf("ParseLog() error = %v", err)
	}
	if len(commits) != 4 {
		t.Fatalf("expected 4 commits, got %d", len(commits))
	}

	merge := commits[0]
	if merge.Subject != "merge feature" {
		t.Fatalf("expected merge commit first, got subject %q", merge.Subject)
	}
	if len(merge.Parents) != 2 {
		t.Fatalf("expected merge commit to have 2 parents, got %d (%v)", len(merge.Parents), merge.Parents)
	}
}

func TestGetChangedFiles(t *testing.T) {
	initTestRepo(t)

	root := writeAndCommit(t, "a.txt", "one", "add a")
	rootFiles, err := GetChangedFiles(root)
	if err != nil {
		t.Fatalf("GetChangedFiles(root) error = %v", err)
	}
	if len(rootFiles) != 1 || rootFiles[0].Status != "A" || rootFiles[0].Path != "a.txt" {
		t.Fatalf("expected root commit to show A a.txt, got %+v", rootFiles)
	}

	second := writeAndCommit(t, "a.txt", "two", "modify a")
	modFiles, err := GetChangedFiles(second)
	if err != nil {
		t.Fatalf("GetChangedFiles(second) error = %v", err)
	}
	if len(modFiles) != 1 || modFiles[0].Status != "M" || modFiles[0].Path != "a.txt" {
		t.Fatalf("expected modify commit to show M a.txt, got %+v", modFiles)
	}

	if err := os.Remove("a.txt"); err != nil {
		t.Fatal(err)
	}
	runGit(t, "add", "a.txt")
	runGit(t, "commit", "-q", "-m", "delete a")
	third := strings.TrimSpace(runGit(t, "rev-parse", "HEAD"))
	delFiles, err := GetChangedFiles(third)
	if err != nil {
		t.Fatalf("GetChangedFiles(third) error = %v", err)
	}
	if len(delFiles) != 1 || delFiles[0].Status != "D" || delFiles[0].Path != "a.txt" {
		t.Fatalf("expected delete commit to show D a.txt, got %+v", delFiles)
	}
}

func TestGetCurrentBranch(t *testing.T) {
	initTestRepo(t)
	writeAndCommit(t, "a.txt", "one", "first commit")

	branch, err := GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch() error = %v", err)
	}
	if branch != "main" {
		t.Fatalf("expected branch %q, got %q", "main", branch)
	}

	// Detached HEAD should fall back to a short hash rather than error.
	head := strings.TrimSpace(runGit(t, "rev-parse", "HEAD"))
	runGit(t, "checkout", "-q", head)

	detached, err := GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch() in detached HEAD error = %v", err)
	}
	if detached == "" || detached == "main" {
		t.Fatalf("expected a short hash for detached HEAD, got %q", detached)
	}
}

func TestCreateAndDeleteBranch(t *testing.T) {
	initTestRepo(t)
	head := writeAndCommit(t, "a.txt", "one", "first commit")

	if err := CreateBranch("feature", head); err != nil {
		t.Fatalf("CreateBranch() error = %v", err)
	}
	branches := runGit(t, "branch", "--list", "feature")
	if !strings.Contains(branches, "feature") {
		t.Fatalf("expected branch \"feature\" to exist, git branch --list output: %q", branches)
	}

	if err := DeleteBranch("feature", false); err != nil {
		t.Fatalf("DeleteBranch() error = %v", err)
	}
	branches = runGit(t, "branch", "--list", "feature")
	if strings.Contains(branches, "feature") {
		t.Fatalf("expected branch \"feature\" to be deleted, git branch --list output: %q", branches)
	}
}

func TestCreateBranchDuplicateFails(t *testing.T) {
	initTestRepo(t)
	head := writeAndCommit(t, "a.txt", "one", "first commit")

	if err := CreateBranch("dup", head); err != nil {
		t.Fatalf("CreateBranch() error = %v", err)
	}
	if err := CreateBranch("dup", head); err == nil {
		t.Fatal("expected error creating a branch that already exists")
	}
}

func TestCheckout(t *testing.T) {
	initTestRepo(t)
	first := writeAndCommit(t, "a.txt", "one", "first commit")
	writeAndCommit(t, "a.txt", "two", "second commit")

	result := Checkout(first)
	if !result.Success {
		t.Fatalf("expected checkout to succeed, got message: %q", result.Message)
	}

	head := strings.TrimSpace(runGit(t, "rev-parse", "HEAD"))
	if head != first {
		t.Fatalf("expected HEAD to be %s after checkout, got %s", first, head)
	}
}

func TestCheckoutInvalidRef(t *testing.T) {
	initTestRepo(t)
	writeAndCommit(t, "a.txt", "one", "first commit")

	result := Checkout("not-a-real-ref")
	if result.Success {
		t.Fatal("expected checkout of an invalid ref to fail")
	}
	if result.Message == "" {
		t.Fatal("expected a non-empty error message on failed checkout")
	}
}

func TestGetCommitDiff(t *testing.T) {
	initTestRepo(t)
	head := writeAndCommit(t, "a.txt", "hello\n", "add a")

	diff, err := GetCommitDiff(head)
	if err != nil {
		t.Fatalf("GetCommitDiff() error = %v", err)
	}
	if !strings.Contains(diff, "a.txt") {
		t.Fatalf("expected diff to mention a.txt, got: %s", diff)
	}
	if !strings.Contains(diff, "+hello") {
		t.Fatalf("expected diff to contain added line, got: %s", diff)
	}
}

func TestGetWorkingTreeDiffUnstagedVsStaged(t *testing.T) {
	initTestRepo(t)
	writeAndCommit(t, "a.txt", "line1\nline2\n", "first commit")

	if err := os.WriteFile("a.txt", []byte("line1\nline2-changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	unstagedDiff, err := GetWorkingTreeDiff("a.txt", false)
	if err != nil {
		t.Fatalf("GetWorkingTreeDiff(unstaged) error = %v", err)
	}
	if !strings.Contains(unstagedDiff, "line2-changed") {
		t.Fatalf("expected unstaged diff to contain the change, got: %s", unstagedDiff)
	}

	stagedDiffBefore, err := GetWorkingTreeDiff("a.txt", true)
	if err != nil {
		t.Fatalf("GetWorkingTreeDiff(staged) error = %v", err)
	}
	if stagedDiffBefore != "" {
		t.Fatalf("expected empty staged diff before staging, got: %s", stagedDiffBefore)
	}

	runGit(t, "add", "a.txt")

	stagedDiffAfter, err := GetWorkingTreeDiff("a.txt", true)
	if err != nil {
		t.Fatalf("GetWorkingTreeDiff(staged) error = %v", err)
	}
	if !strings.Contains(stagedDiffAfter, "line2-changed") {
		t.Fatalf("expected staged diff to contain the change after staging, got: %s", stagedDiffAfter)
	}
}
