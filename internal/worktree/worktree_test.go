package worktree

import "testing"

func TestRelPathSameDirIsEmpty(t *testing.T) {
	// opencode's path.relative(x, x) == ""; a session at the worktree root must
	// carry path "" (not Go's filepath.Rel default of ".").
	if got := RelPath("/repo", "/repo"); got != "" {
		t.Errorf(`RelPath(root, root) = %q, want ""`, got)
	}
}

func TestRelPathBelowRoot(t *testing.T) {
	if got := RelPath("/repo", "/repo/pkg/sub"); got != "pkg/sub" {
		t.Errorf("RelPath = %q, want pkg/sub", got)
	}
}

func TestRelPathNonGitRootStripsLeadingSlash(t *testing.T) {
	// Non-git worktree is "/"; the recorded opencode "path" for /private/var/...
	// is the dir without its leading slash.
	if got := RelPath("/", "/private/var/folders/x/T/dir"); got != "private/var/folders/x/T/dir" {
		t.Errorf("RelPath = %q", got)
	}
}
