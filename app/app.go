package app

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xssh/xssh/config"
	"github.com/xssh/xssh/layout"
	"github.com/xssh/xssh/pane"
	"github.com/xssh/xssh/session"
)

const (
	defaultRows        = 24
	defaultCols        = 80
	borderWidth        = 1 // border on each side
	statusBarHeight    = 1
	inputBarHeight     = 4 // 2 content lines + top/bottom border
	reservedHeight     = statusBarHeight + inputBarHeight
	maxReconnectTries  = 3
	reconnectInterval  = 5 * time.Second
)

// New builds a Model from a list of target strings and connects each session.
// Targets are: "-" for local shell, "user@host" or alias for SSH.
func New(targets []string, borderMode BorderMode) (Model, error) {
	if len(targets) == 0 {
		targets = []string{"-"}
	}
	if len(targets) > 9 {
		return Model{}, fmt.Errorf("maximum 9 panes, got %d", len(targets))
	}

	m := NewModel(borderMode)

	// Compute an initial layout at a placeholder size; real size comes from WindowSizeMsg.
	m.layout = layout.Compute(len(targets), defaultCols, defaultRows+reservedHeight, reservedHeight, borderMode == BorderShared)
	total := m.layout.Rows * m.layout.Cols

	m.panes = make([]*pane.Pane, total)
	m.broadcastTo = make([]bool, total)

	for i := 0; i < total; i++ {
		rect := m.layout.Panes[i]
		contentW, contentH := m.paneContentSize(rect)

		if i < len(targets) {
			sess, err := buildSession(targets[i])
			if err != nil {
				return Model{}, fmt.Errorf("target %q: %w", targets[i], err)
			}
			m.panes[i] = pane.New(i, sess, contentH, contentW)
			m.broadcastTo[i] = true
		} else {
			// Empty slot
			p := pane.New(i, nil, contentH, contentW)
			p.Closed = true
			m.panes[i] = p
		}
	}

	// Focus first pane
	m.focusedPane = 0

	return m, nil
}

// buildSession creates the right session type for a target string.
// SSH targets run "ssh <target>" in a local PTY so the system ssh binary
// handles all configuration, known_hosts, proxies and auth automatically.
func buildSession(target string) (session.Session, error) {
	if target == "-" {
		return session.NewLocal(), nil
	}
	return session.NewLocalCmd([]string{"ssh", target}, target), nil
}

// buildHostEntry resolves a target string to a HostEntry.
func buildHostEntry(target string) (*config.HostEntry, error) {
	// Try SSH config alias first
	if config.IsKnownAlias(target) {
		return config.Resolve(target)
	}
	// Parse user@host or host
	user := os.Getenv("USER")
	host := target
	port := "22"
	if strings.Contains(target, "@") {
		parts := strings.SplitN(target, "@", 2)
		user = parts[0]
		host = parts[1]
	}
	if strings.Contains(host, ":") {
		parts := strings.SplitN(host, ":", 2)
		host = parts[0]
		port = parts[1]
	}
	return &config.HostEntry{
		Alias:    target,
		HostName: host,
		User:     user,
		Port:     port,
	}, nil
}

// connectAll sends connect commands for all non-closed panes.
func connectAll(m *Model) tea.Cmd {
	var cmds []tea.Cmd
	for i, p := range m.panes {
		if !p.Closed && p.Session != nil {
			cmds = append(cmds, connectPane(i, p))
		}
	}
	return tea.Batch(cmds...)
}

// connectPane connects a single session and starts listening for output.
func connectPane(id int, p *pane.Pane) tea.Cmd {
	return func() tea.Msg {
		if err := p.Session.Connect(); err != nil {
			return PaneStatusMsg{PaneID: id, Status: session.StatusDisconnected}
		}
		return PaneStatusMsg{PaneID: id, Status: session.StatusConnected}
	}
}

// listenPane returns a Cmd that blocks on the pane output channel.
func listenPane(id int, ch <-chan []byte) tea.Cmd {
	return func() tea.Msg {
		data, ok := <-ch
		if !ok {
			return PaneStatusMsg{PaneID: id, Status: session.StatusDisconnected}
		}
		return PaneOutputMsg{PaneID: id, Data: data}
	}
}

// reconnectAfter schedules a reconnect attempt after a delay.
func reconnectAfter(id, attempt int, d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return ReconnectTickMsg{PaneID: id, Attempt: attempt}
	})
}

// Run starts the bubbletea program with the given targets.
func Run(targets []string, borderMode BorderMode) error {
	m, err := New(targets, borderMode)
	if err != nil {
		return err
	}
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err = p.Run()
	return err
}
