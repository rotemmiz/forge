package resource

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSystemInstructions_ProjectFirstFilenameWins(t *testing.T) {
	dir := project(t) // isolates XDG_CONFIG_HOME (no global AGENTS.md)
	// Both AGENTS.md and CLAUDE.md exist; AGENTS.md wins (first in the priority
	// list) and CLAUDE.md is NOT stacked.
	writeFile(t, dir, "AGENTS.md", "follow the agents rules")
	writeFile(t, dir, "CLAUDE.md", "claude rules")

	out := SystemInstructions(dir, map[string]any{})
	if len(out) != 1 {
		t.Fatalf("want 1 instruction block, got %d: %v", len(out), out)
	}
	if !strings.Contains(out[0], "follow the agents rules") || !strings.HasPrefix(out[0], "Instructions from: ") {
		t.Fatalf("block wrong: %q", out[0])
	}
	if strings.Contains(out[0], "claude rules") {
		t.Fatal("CLAUDE.md should not be stacked when AGENTS.md matched")
	}
}

func TestSystemInstructions_FallThroughToClaude(t *testing.T) {
	dir := project(t)
	writeFile(t, dir, "CLAUDE.md", "claude only")
	out := SystemInstructions(dir, map[string]any{})
	if len(out) != 1 || !strings.Contains(out[0], "claude only") {
		t.Fatalf("CLAUDE.md should be used when AGENTS.md absent: %v", out)
	}
}

func TestSystemInstructions_ConfigInstructionsAndURLSkip(t *testing.T) {
	dir := project(t)
	writeFile(t, dir, "docs/rules.md", "extra rules")
	cfg := map[string]any{"instructions": []any{
		"docs/rules.md",
		"https://example.com/remote.md", // skipped (remote not fetched)
	}}
	out := SystemInstructions(dir, cfg)
	joined := strings.Join(out, "\n---\n")
	if !strings.Contains(joined, "extra rules") {
		t.Fatalf("config instruction not loaded: %v", out)
	}
	if strings.Contains(joined, "example.com") {
		t.Fatal("remote instruction URL should be skipped")
	}
}

func TestSystemInstructions_RelativeConfigGlobsUpToParent(t *testing.T) {
	// A relative config instruction living in a parent dir must be found by
	// globbing up the ancestors (opencode globUp), not just the cwd.
	root := project(t)
	writeFile(t, root, ".git/HEAD", "ref: refs/heads/main") // make root the worktree root
	writeFile(t, root, "rules/team.md", "team rules at root")
	sub := filepath.Join(root, "pkg", "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	out := SystemInstructions(sub, map[string]any{"instructions": []any{"rules/team.md"}})
	joined := strings.Join(out, "\n")
	if !strings.Contains(joined, "team rules at root") {
		t.Fatalf("relative config instruction in a parent dir not found: %v", out)
	}
}

func TestSystemInstructions_AbsoluteConfigPath(t *testing.T) {
	root := project(t)
	writeFile(t, root, "shared/abs.md", "absolute rules")
	abs := filepath.Join(root, "shared", "abs.md")
	out := SystemInstructions(t.TempDir(), map[string]any{"instructions": []any{abs}})
	if len(out) != 1 || !strings.Contains(out[0], "absolute rules") {
		t.Fatalf("absolute config instruction not loaded: %v", out)
	}
}

func TestSystemInstructions_EmptyWhenNone(t *testing.T) {
	dir := project(t)
	if out := SystemInstructions(dir, map[string]any{}); len(out) != 0 {
		t.Fatalf("expected no instructions, got %v", out)
	}
}

func TestSystemInstructions_DedupesSamePath(t *testing.T) {
	dir := project(t)
	writeFile(t, dir, "AGENTS.md", "rules")
	// Reference the same file again via config; it must not be added twice.
	cfg := map[string]any{"instructions": []any{"AGENTS.md"}}
	out := SystemInstructions(dir, cfg)
	if len(out) != 1 {
		t.Fatalf("same path should be de-duplicated, got %d: %v", len(out), out)
	}
}
