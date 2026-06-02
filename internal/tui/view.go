package tui

// viewState holds the stream display toggles (plan 08a §D). Timestamps are
// omitted — the TUI store doesn't carry per-message time — leaving the two
// toggles backed by data the renderer already has.
type viewState struct {
	hideThinking bool // hide reasoning ("Thought …") lines
	hideTools    bool // hide tool rows (collapse generic tool output)
}

// toggleHint returns a one-line status string describing a toggle's new value.
func toggleHint(name string, on bool) string {
	if on {
		return name + ": on"
	}
	return name + ": off"
}
