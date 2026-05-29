// Package worktree resolves a request's directory to its canonical form and
// locates the enclosing VCS worktree, mirroring opencode's instance context so
// session.directory / session.path come out byte-identical on the wire.
//
// opencode resolves the incoming directory through realpath and, for non-git
// projects, sets the worktree to "/" (project/instance-context.ts:20-22); the
// session "path" field is path.relative(worktree, directory)
// (session/session.ts:157-158,670).
package worktree

import (
	"os"
	"path/filepath"
)

// Resolve canonicalizes dir the way opencode's instance context does: it
// follows symlinks (so /var/folders/x → /private/var/folders/x on macOS) and
// cleans the result. If the path cannot be resolved (e.g. it does not exist) it
// falls back to filepath.Clean of the absolute input.
func Resolve(dir string) string {
	if dir == "" {
		return ""
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = filepath.Clean(dir)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	return abs
}

// Root returns the VCS worktree enclosing resolvedDir: the nearest ancestor
// containing a ".git" entry. When none is found it returns "/" to match
// opencode's non-git fallback (instance-context.ts:20-22).
func Root(resolvedDir string) string {
	dir := resolvedDir
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the filesystem root without finding a .git.
			return "/"
		}
		dir = parent
	}
}

// RelPath computes the session "path" field: the resolved directory relative to
// its worktree, with OS separators normalized to forward slashes
// (session/session.ts:157-158). When worktree is "/" this yields a path with no
// leading slash (e.g. "private/var/folders/...").
func RelPath(root, resolvedDir string) string {
	rel, err := filepath.Rel(root, resolvedDir)
	if err != nil {
		return resolvedDir
	}
	// Go's filepath.Rel(x, x) is "."; opencode's path.relative(x, x) is "".
	// Match opencode so a session at the worktree root carries path "".
	if rel == "." {
		return ""
	}
	return filepath.ToSlash(rel)
}
