// Package selector implements an interactive host selection TUI that runs
// before the main xssh session when no targets are given on the command line.
package selector

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xssh/xssh/config"
)

const maxPanes = 9

// Run launches the interactive host selector and returns the selected targets.
// Returns nil, nil when the user cancels (Esc/Ctrl+C).
func Run() ([]string, error) {
	hosts, err := config.ListHosts()
	if err != nil {
		hosts = nil // run with empty list; user can type hosts manually
	}

	entries := make([]entry, len(hosts))
	for i, h := range hosts {
		label := h.Alias
		if h.HostName != h.Alias && h.HostName != "" {
			label += " (" + h.HostName + ")"
		}
		if h.User != "" {
			label += " [" + h.User + "]"
		}
		entries[i] = entry{host: h.Alias, label: label}
	}

	m := newModel(entries)
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return nil, err
	}
	result, ok := final.(model)
	if !ok || result.cancelled {
		return nil, nil
	}
	return result.selected(), nil
}

// ── Model ─────────────────────────────────────────────────────────────────────

type entry struct {
	host    string
	label   string
	checked bool
}

type model struct {
	all       []entry  // full unfiltered list
	filtered  []entry  // list after fuzzy filter
	cursor    int      // index in filtered
	search    textinput.Model
	cancelled bool
	done      bool
	width     int
	height    int
}

func newModel(entries []entry) model {
	ti := textinput.New()
	ti.Placeholder = "filter hosts…"
	ti.Focus()
	return model{
		all:      entries,
		filtered: entries,
		search:   ti,
	}
}

func (m model) Init() tea.Cmd { return textinput.Blink }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "enter":
			if len(m.selected()) > 0 {
				m.done = true
				return m, tea.Quit
			}
		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "ctrl+n":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case " ":
			m.toggleCurrent()
		case "ctrl+a":
			m.toggleAll()
		default:
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(msg)
			m.applyFilter()
			return m, cmd
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		return "loading…"
	}

	var sb strings.Builder

	// ── Header ────────────────────────────────────────────────────────────────
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00BFFF")).
		Bold(true).
		Render("xssh — select hosts")
	sb.WriteString(title + "\n\n")

	// ── Search box ────────────────────────────────────────────────────────────
	sb.WriteString(m.search.View() + "\n\n")

	// ── Host list ─────────────────────────────────────────────────────────────
	listH := m.height - 8 // reserve header + search + footer
	if listH < 1 {
		listH = 1
	}
	start := 0
	if m.cursor >= listH {
		start = m.cursor - listH + 1
	}
	end := start + listH
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	if len(m.filtered) == 0 {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).
			Render("  (no hosts match)") + "\n")
	}

	for i := start; i < end; i++ {
		e := m.filtered[i]
		check := "[ ]"
		if e.checked {
			check = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render("[x]")
		}
		row := check + " " + e.label
		if i == m.cursor {
			row = lipgloss.NewStyle().Foreground(lipgloss.Color("#00BFFF")).Render("▶ " + row)
		} else {
			row = "  " + row
		}
		sb.WriteString(row + "\n")
	}

	// ── Footer ────────────────────────────────────────────────────────────────
	selected := m.selected()
	n := len(selected)
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	warn := ""
	if n > maxPanes {
		warn = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4444")).
			Render(fmt.Sprintf("  ⚠ max %d panes — deselect %d", maxPanes, n-maxPanes))
	}

	status := fmt.Sprintf("Selected: %d", n) + warn
	sb.WriteString("\n" + footerStyle.Render(status) + "\n")
	sb.WriteString(footerStyle.Render(
		"Space: toggle  Ctrl+A: all  ↑/↓: navigate  Enter: launch  Esc: cancel",
	))

	return sb.String()
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func (m *model) toggleCurrent() {
	if len(m.filtered) == 0 {
		return
	}
	host := m.filtered[m.cursor].host
	for i := range m.all {
		if m.all[i].host == host {
			m.all[i].checked = !m.all[i].checked
		}
	}
	m.applyFilter() // refresh filtered with updated checked state
}

func (m *model) toggleAll() {
	// If all currently filtered are checked, uncheck all; else check all.
	allChecked := true
	for _, e := range m.filtered {
		if !e.checked {
			allChecked = false
			break
		}
	}
	target := !allChecked
	for i := range m.all {
		for _, f := range m.filtered {
			if m.all[i].host == f.host {
				m.all[i].checked = target
			}
		}
	}
	m.applyFilter()
}

func (m *model) applyFilter() {
	query := strings.ToLower(m.search.Value())
	var out []entry
	for _, e := range m.all {
		if query == "" || fuzzyMatch(e.label, query) {
			out = append(out, e)
		}
	}
	m.filtered = out
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

// fuzzyMatch returns true if all chars of query appear in s in order.
func fuzzyMatch(s, query string) bool {
	s = strings.ToLower(s)
	qi := 0
	for _, c := range s {
		if qi < len(query) && c == rune(query[qi]) {
			qi++
		}
	}
	return qi == len(query)
}

func (m model) selected() []string {
	var out []string
	for _, e := range m.all {
		if e.checked {
			out = append(out, e.host)
		}
	}
	return out
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
