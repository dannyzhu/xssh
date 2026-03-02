package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/xssh/xssh/pane"
	"github.com/xssh/xssh/session"
)

// ── Colours ──────────────────────────────────────────────────────────────────

var (
	colorFocused       = lipgloss.Color("#00BFFF")
	colorInactive      = lipgloss.Color("#555555")
	colorDisconnected  = lipgloss.Color("#FF4444")
	colorReconnecting  = lipgloss.Color("#FFB347")
	colorBroadcast     = lipgloss.Color("#FF6B6B")
	colorStatusBar     = lipgloss.Color("#222222")
	colorStatusText    = lipgloss.Color("#AAAAAA")
	colorEmpty         = lipgloss.Color("#333333")
)

// View renders the full TUI.
func (m Model) View() string {
	if !m.ready {
		return "initialising…"
	}

	// ── Zoom mode ────────────────────────────────────────────────────────────
	if m.zoomedPane >= 0 && m.zoomedPane < len(m.panes) {
		return m.renderZoomed()
	}

	// ── Help overlay ─────────────────────────────────────────────────────────
	if m.showHelp {
		return m.renderHelp()
	}

	// ── Normal multi-pane layout ─────────────────────────────────────────────
	statusBar := m.renderStatusBar()
	panesSection := m.renderPanesSection()
	inputSection := m.renderInputBar()

	return lipgloss.JoinVertical(lipgloss.Left,
		statusBar,
		panesSection,
		inputSection,
	)
}

// ── Status bar ───────────────────────────────────────────────────────────────

func (m Model) renderStatusBar() string {
	barStyle := lipgloss.NewStyle().
		Width(m.width).
		Background(colorStatusBar).
		Foreground(colorStatusText).
		Padding(0, 1)

	// Build title segments
	var parts []string
	for i, p := range m.panes {
		if p.Closed {
			continue
		}
		indicator := statusIndicator(p.Session)
		title := ""
		if p.Session != nil {
			title = p.Session.Title()
		} else {
			title = fmt.Sprintf("pane%d", i+1)
		}
		seg := fmt.Sprintf("[%d] %s %s", i+1, title, indicator)
		if i == m.focusedPane {
			seg = lipgloss.NewStyle().Foreground(colorFocused).Render(seg)
		}
		parts = append(parts, seg)
	}

	label := strings.Join(parts, "  ")

	// Broadcast indicator
	switch m.focusTarget {
	case FocusBroadcast:
		total := len(m.ActivePanes())
		bc := m.BroadcastCount()
		if bc == total {
			label += "  " + lipgloss.NewStyle().Foreground(colorBroadcast).Render("[BROADCAST]")
		} else {
			label += "  " + lipgloss.NewStyle().Foreground(colorBroadcast).
				Render(fmt.Sprintf("[BROADCAST:%d/%d]", bc, total))
		}
	case FocusBroadcastSelect:
		label += "  " + lipgloss.NewStyle().Foreground(colorBroadcast).Render("[SELECT PANES]")
	}

	// Prefix indicator
	if m.prefixState == PrefixWaiting {
		label += "  " + lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00")).Render("[Ctrl+\\]")
	}

	return barStyle.Render(label)
}

func statusIndicator(sess session.Session) string {
	if sess == nil {
		return "○"
	}
	switch sess.Status() {
	case session.StatusConnected:
		return "●"
	case session.StatusConnecting, session.StatusReconnecting:
		return "◌"
	case session.StatusAuthFailed:
		return "✗"
	default:
		return "○"
	}
}

// ── Pane grid ────────────────────────────────────────────────────────────────

func (m Model) renderPanesSection() string {
	rows := m.layout.Rows
	cols := m.layout.Cols
	if rows == 0 || cols == 0 {
		return ""
	}

	rowStrings := make([]string, rows)
	for r := 0; r < rows; r++ {
		rowPanes := make([]string, cols)
		for c := 0; c < cols; c++ {
			idx := r*cols + c
			if idx >= len(m.panes) {
				rowPanes[c] = ""
				continue
			}
			rect := m.layout.Panes[idx]
			rowPanes[c] = m.renderPane(idx, rect.Width, rect.Height)
		}
		rowStrings[r] = lipgloss.JoinHorizontal(lipgloss.Top, rowPanes...)
	}
	return lipgloss.JoinVertical(lipgloss.Left, rowStrings...)
}

func (m Model) renderPane(idx, totalW, totalH int) string {
	p := m.panes[idx]

	borderColor := m.paneColor(idx)
	borderStyle := lipgloss.RoundedBorder()

	contentW := max(1, totalW-2)
	contentH := max(1, totalH-2)

	var content string
	if p.Closed {
		// Grey blank cell
		emptyStyle := lipgloss.NewStyle().
			Width(contentW).Height(contentH).
			Foreground(colorEmpty)
		content = emptyStyle.Render("")
	} else if p.Mode == pane.ModeScroll || p.Mode == pane.ModeSearch {
		content = m.renderScrollContent(p, contentW, contentH)
	} else {
		curRow, curCol := p.VTerm.Cursor()
		rendered := pane.RenderVTerm(p.VTerm, curRow, curCol)
		lines := strings.Split(rendered, "\n")
		if len(lines) > contentH {
			lines = lines[len(lines)-contentH:]
		}
		for len(lines) < contentH {
			lines = append(lines, strings.Repeat(" ", contentW))
		}
		content = strings.Join(lines, "\n")
	}

	// Password overlay
	if p.PasswordOverlay {
		content = m.renderPasswordOverlay(idx, contentW, contentH)
	}

	frame := lipgloss.NewStyle().
		Border(borderStyle).
		BorderForeground(borderColor).
		Width(contentW).
		Height(contentH).
		Render(content)

	return frame
}

func (m Model) paneColor(idx int) lipgloss.Color {
	p := m.panes[idx]
	if p.Closed {
		return colorEmpty
	}
	if m.focusTarget == FocusBroadcast || m.focusTarget == FocusBroadcastSelect {
		if m.broadcastTo[idx] {
			return colorBroadcast
		}
		return colorInactive
	}
	if idx == m.focusedPane {
		return colorFocused
	}
	if p.Session != nil {
		switch p.Session.Status() {
		case session.StatusDisconnected:
			return colorDisconnected
		case session.StatusReconnecting, session.StatusConnecting:
			return colorReconnecting
		case session.StatusAuthFailed:
			return colorDisconnected
		}
	}
	return colorInactive
}

// ── Scroll content ────────────────────────────────────────────────────────────

func (m Model) renderScrollContent(p *pane.Pane, w, h int) string {
	lines := p.Scroll.Lines(h)
	// Pad to h lines
	for len(lines) < h {
		lines = append([]string{strings.Repeat(" ", w)}, lines...)
	}
	// Highlight search matches
	if p.Mode == pane.ModeSearch && p.SearchQuery != "" {
		for i, l := range lines {
			if strings.Contains(l, p.SearchQuery) {
				lines[i] = lipgloss.NewStyle().
					Background(lipgloss.Color("#FFD700")).
					Render(l)
			}
		}
	}
	result := strings.Join(lines, "\n")
	// Search bar at bottom
	if p.Mode == pane.ModeSearch {
		result += "\n" + lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFF00")).
			Render("/" + p.SearchQuery)
	}
	return result
}

// ── Password overlay ──────────────────────────────────────────────────────────

func (m Model) renderPasswordOverlay(paneID, w, h int) string {
	ti, ok := m.passwordInputs[paneID]
	if !ok {
		return strings.Repeat(" ", w*h)
	}
	boxW := min(w-4, 40)
	boxH := 3
	paddingTop := (h - boxH) / 2
	paddingLeft := (w - boxW) / 2

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FFFF00")).
		Width(boxW).
		Padding(0, 1).
		Render("Password:\n" + ti.View())

	var sb strings.Builder
	for i := 0; i < paddingTop; i++ {
		sb.WriteString(strings.Repeat(" ", w) + "\n")
	}
	sb.WriteString(strings.Repeat(" ", paddingLeft) + box)
	return sb.String()
}

// ── Input bar ─────────────────────────────────────────────────────────────────

func (m Model) renderInputBar() string {
	innerW := m.width - 2 // width inside the border (border adds 1 char each side)

	innerStyle := lipgloss.NewStyle().
		Width(innerW).
		Background(lipgloss.Color("#111111")).
		Foreground(lipgloss.Color("#CCCCCC")).
		Padding(0, 1)

	var line0, line1 string

	switch m.focusTarget {
	case FocusBroadcast:
		line0 = lipgloss.NewStyle().Foreground(colorBroadcast).Render("BROADCAST") +
			"  " + m.inputBar.View()
		line1 = lipgloss.NewStyle().Foreground(colorInactive).
			Render("Enter: send  Esc: cancel  Ctrl+\\+m: select panes")
	case FocusBroadcastSelect:
		line0 = m.renderBroadcastSelectList()
		line1 = lipgloss.NewStyle().Foreground(colorInactive).
			Render("Space: toggle  Ctrl+A: all  Enter/Esc: confirm")
	case FocusAddPane:
		line0 = lipgloss.NewStyle().Foreground(colorFocused).Render("Add pane: ") +
			m.addPaneInput.View()
		line1 = lipgloss.NewStyle().Foreground(colorInactive).
			Render("Enter: connect  Esc: cancel")
	default:
		if m.focusedPane >= 0 && m.focusedPane < len(m.panes) {
			p := m.panes[m.focusedPane]
			if p.Mode == pane.ModeScroll {
				line0 = lipgloss.NewStyle().Foreground(colorFocused).Render("SCROLL MODE")
				line1 = lipgloss.NewStyle().Foreground(colorInactive).
					Render("↑/k ↓/j PgUp PgDn g/G: top/bottom  /: search  q/Esc: exit")
			} else if p.Mode == pane.ModeSearch {
				line0 = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00")).
					Render("SEARCH: " + p.SearchQuery)
				line1 = lipgloss.NewStyle().Foreground(colorInactive).
					Render("n/N: next/prev  Esc: back to scroll")
			} else {
				line0 = lipgloss.NewStyle().Foreground(colorInactive).
					Render("Ctrl+\\: prefix  ?help  b:broadcast  [: scroll  z: zoom")
			}
		}
	}

	// Border colour reflects active state
	borderColor := colorInactive
	switch m.focusTarget {
	case FocusBroadcast, FocusBroadcastSelect:
		borderColor = colorBroadcast
	case FocusAddPane:
		borderColor = colorFocused
	}

	content := innerStyle.Render(line0) + "\n" + innerStyle.Render(line1)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Background(lipgloss.Color("#111111")).
		Width(innerW).
		Render(content)
}

func (m Model) renderBroadcastSelectList() string {
	var parts []string
	for i, p := range m.panes {
		if p.Closed {
			continue
		}
		check := "[ ]"
		if m.broadcastTo[i] {
			check = "[x]"
		}
		title := fmt.Sprintf("pane%d", i+1)
		if p.Session != nil {
			title = p.Session.Title()
		}
		seg := fmt.Sprintf("%s %s", check, title)
		if i == m.focusedPane {
			seg = lipgloss.NewStyle().Foreground(colorFocused).Render(seg)
		}
		parts = append(parts, seg)
	}
	return strings.Join(parts, "  ")
}

// ── Zoom view ─────────────────────────────────────────────────────────────────

func (m Model) renderZoomed() string {
	p := m.panes[m.zoomedPane]
	contentH := m.height - reservedHeight
	contentW := m.width - 2

	var content string
	if p.Closed {
		content = strings.Repeat(" ", contentW*contentH)
	} else {
		curRow, curCol := p.VTerm.Cursor()
		rendered := pane.RenderVTerm(p.VTerm, curRow, curCol)
		content = rendered
	}

	title := fmt.Sprintf("pane%d", m.zoomedPane+1)
	if p.Session != nil {
		title = p.Session.Title()
	}

	frame := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorFocused).
		Width(contentW).
		Height(contentH).
		Render(content)

	statusBar := lipgloss.NewStyle().
		Width(m.width).
		Background(colorStatusBar).
		Foreground(colorStatusText).
		Padding(0, 1).
		Render(fmt.Sprintf("[ZOOM] %s  Ctrl+\\+z: restore", title))

	return lipgloss.JoinVertical(lipgloss.Left, statusBar, frame)
}

// ── Help overlay ──────────────────────────────────────────────────────────────

func (m Model) renderHelp() string {
	help := `
  xssh — keyboard shortcuts

  Ctrl+\ 1-9   Focus pane 1-9
  Ctrl+\ h/j/k/l  Focus left/down/up/right
  Ctrl+\ z     Zoom current pane (toggle)
  Ctrl+\ x     Close current pane
  Ctrl+\ r     Reconnect current pane
  Ctrl+\ R     Reconnect all panes
  Ctrl+\ b     Broadcast input to all panes
  Ctrl+\ m     Select panes for broadcast
  Ctrl+\ [     Enter scroll mode
  Ctrl+\ e     Add a new pane
  Ctrl+\ s     Save current session as group
  Ctrl+\ ?     Show this help
  Ctrl+\ \     Send Ctrl+\ to session

  Scroll mode:
    ↑/k ↓/j    Scroll up/down
    PgUp PgDn  Half-page scroll
    g / G      Top / bottom (G exits scroll)
    /          Search  (n/N: next/prev)
    q Esc      Exit scroll mode

  Press any key to close.
`
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorFocused).
		Padding(1, 2).
		Width(m.width - 4).
		Foreground(lipgloss.Color("#EEEEEE"))

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		style.Render(help))
}

// min/max helpers (Go 1.21+)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
