// Package theme is the Forge TUI's color system, lifted verbatim from the
// design handoff's tokens (design/tui/styles.css :root). It exposes a Palette of
// truecolor values and the canonical Lipgloss styles the design defines
// (selection bar, mode chip, semantic text). Lipgloss degrades truecolor to the
// terminal's best available palette automatically.
package theme

import "github.com/charmbracelet/lipgloss"

// Palette holds the design tokens. Names mirror the CSS custom properties.
type Palette struct {
	// Surfaces (neutral charcoal).
	Bg         lipgloss.Color // terminal background
	BgPanel    lipgloss.Color // collapsible panels, composer, autocomplete
	BgElev     lipgloss.Color // modals / popovers
	BgSel      lipgloss.Color // row hover
	Border     lipgloss.Color // table/panel/modal borders
	BorderSoft lipgloss.Color // hairline dividers, sidebar edge

	// Text.
	Fg      lipgloss.Color // primary
	FgDim   lipgloss.Color // secondary, tool-call lines
	FgFaint lipgloss.Color // hints, line numbers, metadata
	FgGhost lipgloss.Color // placeholders, disabled, diff gutters

	// Semantic colors (meanings fixed by the design — do not repurpose).
	Blue   lipgloss.Color // agent mode, prompt accent, function names
	Green  lipgloss.Color // added diff, success, paths, strings
	Red    lipgloss.Color // removed diff, errors, blocked
	Amber  lipgloss.Color // selection highlight, in-progress, thinking
	Purple lipgloss.Color // section headers, keywords, table headers
	Cyan   lipgloss.Color // types, @mentions, links, hunk markers
	Yellow lipgloss.Color // reserve / rarely used

	// Selection bar (modal & table highlight): solid amber, near-black text.
	SelBg lipgloss.Color
	SelFg lipgloss.Color
}

// Accent is the UI accent color (the design aliases --accent to --blue).
func (p Palette) Accent() lipgloss.Color { return p.Blue }

// Default returns the design's charcoal palette.
func Default() Palette {
	return Palette{
		Bg: "#15171a", BgPanel: "#1c1f23", BgElev: "#20242a", BgSel: "#262b31",
		Border: "#2c3137", BorderSoft: "#23272c",
		Fg: "#d6dade", FgDim: "#8b929a", FgFaint: "#585f67", FgGhost: "#3a4047",
		Blue: "#6fa8dc", Green: "#8cc265", Red: "#e0606e", Amber: "#d99a4e",
		Purple: "#b08cd4", Cyan: "#5fb3c4", Yellow: "#d6c370",
		SelBg: "#d99a4e", SelFg: "#1a1207",
	}
}

// Styles are the canonical reusable Lipgloss styles the design defines. Screen
// renderers compose these rather than re-deriving colors.
type Styles struct {
	P Palette

	// Base is primary text on the terminal background.
	Base lipgloss.Style
	// Dim / Faint / Ghost are the secondary text tiers.
	Dim   lipgloss.Style
	Faint lipgloss.Style
	Ghost lipgloss.Style
	// Section is a purple bold header (markdown h3, modal labels, table headers).
	Section lipgloss.Style
	// Selection is the canonical cursor row: full-width amber bar, dark bold text.
	Selection lipgloss.Style
	// ModeChip is the status-bar mode chip: accent bg, dark bold text.
	ModeChip lipgloss.Style
	// Accent is accent-colored text (prompt accent, agent mode).
	Accent lipgloss.Style
}

// New builds the Styles for a palette.
func New(p Palette) Styles {
	return Styles{
		P:         p,
		Base:      lipgloss.NewStyle().Foreground(p.Fg),
		Dim:       lipgloss.NewStyle().Foreground(p.FgDim),
		Faint:     lipgloss.NewStyle().Foreground(p.FgFaint),
		Ghost:     lipgloss.NewStyle().Foreground(p.FgGhost),
		Section:   lipgloss.NewStyle().Foreground(p.Purple).Bold(true),
		Selection: lipgloss.NewStyle().Background(p.SelBg).Foreground(p.SelFg).Bold(true),
		ModeChip:  lipgloss.NewStyle().Background(p.Accent()).Foreground(p.SelFg).Bold(true).Padding(0, 1),
		Accent:    lipgloss.NewStyle().Foreground(p.Accent()),
	}
}

// DefaultStyles returns the Styles for the default palette.
func DefaultStyles() Styles { return New(Default()) }
