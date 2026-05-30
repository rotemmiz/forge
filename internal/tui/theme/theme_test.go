package theme

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestDefaultPaletteMatchesTokens(t *testing.T) {
	p := Default()
	cases := map[string]lipgloss.Color{
		"#15171a": p.Bg, "#1c1f23": p.BgPanel, "#20242a": p.BgElev,
		"#d6dade": p.Fg, "#8b929a": p.FgDim,
		"#6fa8dc": p.Blue, "#8cc265": p.Green, "#e0606e": p.Red,
		"#d99a4e": p.Amber, "#b08cd4": p.Purple, "#5fb3c4": p.Cyan,
		"#1a1207": p.SelFg,
	}
	for hex, got := range cases {
		if string(got) != hex {
			t.Errorf("token mismatch: got %q want %q", string(got), hex)
		}
	}
	if p.Accent() != p.Blue {
		t.Fatalf("accent should alias blue")
	}
}

func TestSelectionStyleColors(t *testing.T) {
	s := DefaultStyles()
	// The canonical cursor row is amber bg + near-black fg, bold.
	out := s.Selection.Render("row")
	if !strings.Contains(out, "row") {
		t.Fatalf("selection render lost content: %q", out)
	}
	if s.Selection.GetBackground() != Default().SelBg || s.Selection.GetForeground() != Default().SelFg {
		t.Fatalf("selection colors wrong")
	}
	if !s.Selection.GetBold() {
		t.Fatalf("selection should be bold")
	}
}

func TestSectionIsPurpleBold(t *testing.T) {
	s := DefaultStyles()
	if s.Section.GetForeground() != Default().Purple || !s.Section.GetBold() {
		t.Fatalf("section header should be purple bold")
	}
}
