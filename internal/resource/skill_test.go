package resource

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSkills(t *testing.T) {
	dir := project(t)
	writeFile(t, dir, ".opencode/skills/effect/SKILL.md",
		"---\nname: effect\ndescription: Work with Effect\n---\nUse Effect for typed errors.")
	writeFile(t, dir, ".opencode/skill/single/SKILL.md",
		"---\nname: single\n---\nbody here")
	// No-name skill is skipped.
	writeFile(t, dir, ".opencode/skills/noname/SKILL.md", "---\ndescription: x\n---\nignored")

	skills := LoadSkills(dir, map[string]any{})
	byName := map[string]Skill{}
	for _, s := range skills {
		byName[s.Name] = s
	}
	eff, ok := byName["effect"]
	if !ok {
		t.Fatalf("effect skill not loaded: %v", skills)
	}
	if eff.Description != "Work with Effect" || eff.Content != "Use Effect for typed errors." {
		t.Fatalf("effect fields wrong: %+v", eff)
	}
	if !strings.HasSuffix(eff.Location, "SKILL.md") {
		t.Fatalf("location should be the SKILL.md path: %q", eff.Location)
	}
	if _, ok := byName["single"]; !ok {
		t.Errorf("single skill (from skill/ dir) not loaded")
	}
	if _, ok := byName[""]; ok {
		t.Error("a no-name skill should be skipped")
	}
	if len(skills) != 2 {
		t.Fatalf("want 2 skills, got %d: %v", len(skills), skills)
	}
}

func TestLoadSkills_ProjectOverridesGlobal(t *testing.T) {
	dir := project(t) // sets XDG_CONFIG_HOME → configHome() is sandboxed
	globalSkills := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "opencode")
	writeFile(t, globalSkills, "skills/shared/SKILL.md", "---\nname: shared\n---\nGLOBAL version")
	writeFile(t, dir, ".opencode/skills/shared/SKILL.md", "---\nname: shared\n---\nPROJECT version")

	skills := LoadSkills(dir, map[string]any{})
	var got string
	for _, s := range skills {
		if s.Name == "shared" {
			got = s.Content
		}
	}
	if got != "PROJECT version" {
		t.Fatalf("project skill should override global; got %q", got)
	}
}

func TestLoadSkills_ExternalAndConfigPaths(t *testing.T) {
	dir := project(t) // sandboxes HOME
	home := os.Getenv("HOME")
	writeFile(t, home, ".claude/skills/cl/SKILL.md", "---\nname: cl-skill\n---\nclaude skill")
	writeFile(t, home, ".agents/skills/ag/SKILL.md", "---\nname: ag-skill\n---\nagents skill")
	writeFile(t, dir, "extra/custom/SKILL.md", "---\nname: custom\n---\ncustom skill")
	cfg := map[string]any{"skills": map[string]any{"paths": []any{"extra"}}}

	got := map[string]bool{}
	for _, s := range LoadSkills(dir, cfg) {
		got[s.Name] = true
	}
	for _, want := range []string{"cl-skill", "ag-skill", "custom"} {
		if !got[want] {
			t.Errorf("skill %q not loaded; got %v", want, got)
		}
	}
}

func TestLoadSkills_ConfigPathOverridesExternal(t *testing.T) {
	dir := project(t)
	home := os.Getenv("HOME")
	writeFile(t, home, ".claude/skills/dup/SKILL.md", "---\nname: dup\n---\nexternal version")
	writeFile(t, dir, "local/dup/SKILL.md", "---\nname: dup\n---\nconfig-path version")
	cfg := map[string]any{"skills": map[string]any{"paths": []any{"local"}}}
	var content string
	for _, s := range LoadSkills(dir, cfg) {
		if s.Name == "dup" {
			content = s.Content
		}
	}
	// scan order is external → .opencode → skills.paths, last-wins.
	if content != "config-path version" {
		t.Fatalf("skills.paths should override external; got %q", content)
	}
}

func TestSkillContent(t *testing.T) {
	dir := project(t)
	writeFile(t, dir, ".opencode/skills/foo/SKILL.md", "---\nname: foo\n---\nthe foo instructions")
	if c, ok := SkillContent(dir, map[string]any{}, "foo"); !ok || c != "the foo instructions" {
		t.Fatalf("SkillContent = %q,%v", c, ok)
	}
	if _, ok := SkillContent(dir, map[string]any{}, "missing"); ok {
		t.Fatal("missing skill should not be found")
	}
}
