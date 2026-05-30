package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	forgeclient "github.com/rotemmiz/forge/sdk/go"
)

// modalKind is the active command overlay (none = the normal screen).
type modalKind int

const (
	modalNone modalKind = iota
	modalPalette
	modalSessions
)

// paletteItems are the command palette actions (dispatch by index in modalSelect).
var paletteItems = []string{"New session", "Switch session", "Refresh sessions"}

// Modal action results.
type (
	sessionOpenedMsg struct {
		session Session
		err     error
	}
	sessionDeletedMsg struct {
		id  string
		err error
	}
)

// newSessionCmd creates a session and opens it (no prompt).
func newSessionCmd(ctx context.Context, c *forgeclient.ForgeClient) tea.Cmd {
	return func() tea.Msg {
		var ss Session
		err := c.PostJSON(ctx, "/session", map[string]any{}, &ss)
		return sessionOpenedMsg{session: ss, err: err}
	}
}

// deleteSessionCmd deletes a session by id.
func deleteSessionCmd(ctx context.Context, c *forgeclient.ForgeClient, id string) tea.Cmd {
	return func() tea.Msg {
		return sessionDeletedMsg{id: id, err: c.Delete(ctx, "/session/"+id)}
	}
}

// orderedSessions returns the sessions newest-first (the store keeps them
// ascending by id; descending id == newest-first).
func (m Model) orderedSessions() []Session {
	out := make([]Session, len(m.store.sessions))
	for i, s := range m.store.sessions {
		out[len(out)-1-i] = s
	}
	return out
}

// modalItems returns the visible rows + an optional footer hint for the modal.
func (m Model) modalItems() (title string, rows []string, footer string) {
	switch m.modal {
	case modalPalette:
		return "Commands", paletteItems, "↑↓ move · enter select · esc close"
	case modalSessions:
		for _, s := range m.orderedSessions() {
			rows = append(rows, sessionRowLabel(s))
		}
		if len(rows) == 0 {
			rows = []string{"(no sessions — ctrl+n to create)"}
		}
		return "Sessions", rows, "enter open · ctrl+n new · ctrl+d delete · esc close"
	default:
		return "", nil, ""
	}
}

func sessionRowLabel(s Session) string {
	if s.Title != "" {
		return s.Title
	}
	return s.ID
}

// modalCount is the number of selectable rows in the active modal.
func (m Model) modalCount() int {
	_, rows, _ := m.modalItems()
	return len(rows)
}

// modalSelect dispatches the highlighted row.
func (m Model) modalSelect() (tea.Model, tea.Cmd) {
	switch m.modal {
	case modalPalette:
		m.modal = modalNone
		switch m.modalSel {
		case 0: // New session
			return m, newSessionCmd(m.ctx, m.client)
		case 1: // Switch session
			m.modal, m.modalSel = modalSessions, 0
			return m, loadSessionsCmd(m.ctx, m.client)
		case 2: // Refresh
			return m, loadSessionsCmd(m.ctx, m.client)
		}
	case modalSessions:
		ss := m.orderedSessions()
		m.modal = modalNone
		if m.modalSel < len(ss) {
			m.cfg.SessionID = ss[m.modalSel].ID
			m.screen = ScreenSession
			return m, loadMessagesCmd(m.ctx, m.client, m.cfg.SessionID)
		}
	}
	return m, nil
}

// modalView renders the active modal as a centered panel over the background.
func (m Model) modalView() string {
	s := m.styles
	title, rows, footer := m.modalItems()

	width := 56
	var lines []string
	lines = append(lines, s.Section.Render(title), "")
	for i, row := range rows {
		if i == m.modalSel {
			lines = append(lines, s.Selection.Width(width-2).Render(" "+row))
		} else {
			lines = append(lines, s.Base.Render("  "+row))
		}
	}
	lines = append(lines, "", s.Faint.Render(footer))

	panel := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(s.P.Border).
		Background(s.P.BgElev).
		Padding(1, 2).
		Width(width).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))

	if m.width == 0 || m.height == 0 {
		return panel
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, panel)
}
