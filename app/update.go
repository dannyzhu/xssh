package app

import (
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/xssh/xssh/layout"
	"github.com/xssh/xssh/pane"
	"github.com/xssh/xssh/session"
)

// Init is called once at startup. It connects all sessions and enters alt-screen.
func (m Model) Init() tea.Cmd {
	return connectAll(&m)
}

// Update is the bubbletea update function. It handles all messages and returns
// a new model and optional command.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	// ── Terminal resize ──────────────────────────────────────────────────────
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.layout = layout.Compute(len(m.ActivePanes()), msg.Width, msg.Height, reservedHeight)
		m = m.resizePanes()

	// ── Pane output ──────────────────────────────────────────────────────────
	case PaneOutputMsg:
		if msg.PaneID >= 0 && msg.PaneID < len(m.panes) {
			p := m.panes[msg.PaneID]
			// Capture rows before write so we can add new lines to scroll buffer
			rowsBefore, _ := p.VTerm.Size()
			curRowBefore, _ := p.VTerm.Cursor()
			p.VTerm.Write(msg.Data)
			// Add newly rendered lines to scroll buffer
			curRowAfter, _ := p.VTerm.Cursor()
			feedScrollBuffer(p, curRowBefore, curRowAfter, rowsBefore)
			// Keep listening
			if p.Session != nil {
				cmds = append(cmds, listenPane(msg.PaneID, p.Session.Output()))
			}
		}

	// ── Session status changes ───────────────────────────────────────────────
	case PaneStatusMsg:
		if msg.PaneID >= 0 && msg.PaneID < len(m.panes) {
			p := m.panes[msg.PaneID]
			switch msg.Status {
			case session.StatusConnected:
				p.Mode = pane.ModeNormal
				// Start listening for output
				cmds = append(cmds, listenPane(msg.PaneID, p.Session.Output()))
				// Apply current terminal size to newly connected session
				if m.ready {
					rect := m.layout.Panes[msg.PaneID]
					contentW := max(1, rect.Width-2*borderWidth)
					contentH := max(1, rect.Height-2*borderWidth)
					p.Session.Resize(contentH, contentW) //nolint:errcheck
				}
			case session.StatusDisconnected:
				// Shell exited or SSH dropped — close the pane (grey blank slot).
				p.Closed = true
				p.Session = nil
				if m.focusedPane == msg.PaneID {
					m.focusToNextActive()
				}
				// All panes gone — exit the application.
				if len(m.ActivePanes()) == 0 {
					cmds = append(cmds, tea.Quit)
				}
			}
		}

	// ── Password submit ──────────────────────────────────────────────────────
	case PanePasswordSubmitMsg:
		if msg.PaneID >= 0 && msg.PaneID < len(m.panes) {
			m.panes[msg.PaneID].PasswordOverlay = false
			delete(m.passwordInputs, msg.PaneID)
		}

	// ── Keyboard input ───────────────────────────────────────────────────────
	case tea.KeyMsg:
		cmd := m.handleKey(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	// ── Mouse input ──────────────────────────────────────────────────────────
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionRelease && msg.Button == tea.MouseButtonLeft {
			m.handleMouseClick(msg.X, msg.Y)
		}
		if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
			m.handleMouseScroll(msg.X, msg.Y, msg.Button == tea.MouseButtonWheelUp)
		}
	}

	return m, tea.Batch(cmds...)
}

// handleKey dispatches keyboard events based on current focus and prefix state.
func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	// ── Password overlay ─────────────────────────────────────────────────────
	if m.focusedPane >= 0 && m.focusedPane < len(m.panes) {
		p := m.panes[m.focusedPane]
		if p.PasswordOverlay {
			return m.handlePasswordKey(m.focusedPane, key)
		}
	}

	// ── Broadcast select mode ────────────────────────────────────────────────
	if m.focusTarget == FocusBroadcastSelect {
		return m.handleBroadcastSelectKey(key)
	}

	// ── Broadcast input bar ──────────────────────────────────────────────────
	if m.focusTarget == FocusBroadcast {
		return m.handleBroadcastKey(key)
	}

	// ── Add-pane overlay ─────────────────────────────────────────────────────
	if m.focusTarget == FocusAddPane {
		return m.handleAddPaneKey(key)
	}

	// ── Help overlay ─────────────────────────────────────────────────────────
	if m.showHelp {
		m.showHelp = false
		return nil
	}

	// ── Scroll / search modes ────────────────────────────────────────────────
	if m.focusedPane >= 0 && m.focusedPane < len(m.panes) {
		p := m.panes[m.focusedPane]
		if p.Mode == pane.ModeScroll || p.Mode == pane.ModeSearch {
			return m.handleScrollKey(p, key)
		}
	}

	// Normal passthrough or prefix key
	return m.handleKeymapKey(key)
}

// handleKeymapKey handles the prefix state machine inline.
func (m *Model) handleKeymapKey(key string) tea.Cmd {
	if m.prefixState == PrefixWaiting {
		m.prefixState = PrefixIdle
		action := resolveSecondKey(key)
		return m.dispatchAction(action, key)
	}

	if key == KeyCtrlBackslash {
		m.prefixState = PrefixWaiting
		return nil
	}

	// Normal passthrough to focused pane
	if m.focusedPane >= 0 && m.focusedPane < len(m.panes) {
		p := m.panes[m.focusedPane]
		if p.IsActive() {
			p.Session.Write(keyBytes(key)) //nolint:errcheck
		}
	}
	return nil
}

// resolveSecondKey maps the second key of a Ctrl+\ chord to an Action.
func resolveSecondKey(key string) Action {
	km := &Keymap{}
	return km.resolveSecond(key)
}

// dispatchAction executes the action resolved from the prefix key.
func (m *Model) dispatchAction(action Action, _ string) tea.Cmd {
	switch action {
	case ActionFocusPane1, ActionFocusPane2, ActionFocusPane3,
		ActionFocusPane4, ActionFocusPane5, ActionFocusPane6,
		ActionFocusPane7, ActionFocusPane8, ActionFocusPane9:
		idx := int(action-ActionFocusPane1)
		if idx < len(m.panes) && !m.panes[idx].Closed {
			m.focusedPane = idx
			m.zoomedPane = -1
		}

	case ActionFocusUp:
		m.moveFocus(-m.layout.Cols, 0)
	case ActionFocusDown:
		m.moveFocus(m.layout.Cols, 0)
	case ActionFocusLeft:
		m.moveFocus(-1, 0)
	case ActionFocusRight:
		m.moveFocus(1, 0)

	case ActionZoom:
		if m.zoomedPane == m.focusedPane {
			m.zoomedPane = -1 // un-zoom
		} else {
			m.zoomedPane = m.focusedPane
		}

	case ActionClosePane:
		if m.focusedPane >= 0 && m.focusedPane < len(m.panes) {
			p := m.panes[m.focusedPane]
			if p.Session != nil {
				p.Session.Close() //nolint:errcheck
			}
			p.Closed = true
			m.broadcastTo[m.focusedPane] = false
			m.focusedPane = m.nextActivePane()
		}

	case ActionReconnect:
		if m.focusedPane >= 0 && m.focusedPane < len(m.panes) {
			p := m.panes[m.focusedPane]
			if p.Session != nil && !p.Closed {
				m.reconnectAttempts[m.focusedPane] = 0
				return connectPane(m.focusedPane, p)
			}
		}

	case ActionReconnectAll:
		var cmds []tea.Cmd
		for i, p := range m.panes {
			if !p.Closed && p.Session != nil {
				m.reconnectAttempts[i] = 0
				cmds = append(cmds, connectPane(i, p))
			}
		}
		return tea.Batch(cmds...)

	case ActionFocusBroadcast:
		m.focusTarget = FocusBroadcast
		m.inputBar.Focus()
		// Default: broadcast to all active panes
		for i, p := range m.panes {
			m.broadcastTo[i] = !p.Closed
		}

	case ActionBroadcastSelect:
		m.focusTarget = FocusBroadcastSelect

	case ActionScrollMode:
		if m.focusedPane >= 0 && m.focusedPane < len(m.panes) {
			m.panes[m.focusedPane].Mode = pane.ModeScroll
		}

	case ActionHelp:
		m.showHelp = true

	case ActionAddPane:
		slot := m.FirstEmptySlot()
		if slot < 0 {
			// No empty slot — signal via status (handled in view)
			break
		}
		m.focusTarget = FocusAddPane
		m.addPaneInput.SetValue("")
		m.addPaneInput.Focus()

	case ActionSaveGroup:
		// Collect current targets and save (config integration in Task 19)

	case ActionPassthrough:
		// Send Ctrl+\ itself to the session
		if m.focusedPane >= 0 && m.focusedPane < len(m.panes) {
			p := m.panes[m.focusedPane]
			if p.IsActive() {
				p.Session.Write([]byte{28}) //nolint:errcheck // ASCII FS = Ctrl+\
			}
		}

	case ActionDelete:
		// Same as entering scroll mode then searching (plan: Ctrl+\+/ = scroll mode /)
		if m.focusedPane >= 0 && m.focusedPane < len(m.panes) {
			m.panes[m.focusedPane].Mode = pane.ModeSearch
		}
	}
	return nil
}

// handleBroadcastKey handles keys while in broadcast input mode.
func (m *Model) handleBroadcastKey(key string) tea.Cmd {
	switch key {
	case "esc", "ctrl+c":
		m.focusTarget = FocusPane
		m.inputBar.Blur()
		return nil
	case "enter":
		text := m.inputBar.Value()
		m.sendBroadcast([]byte(text + "\r"))
		m.inputBar.SetValue("")
		return nil
	}
	var cmd tea.Cmd
	m.inputBar, cmd = m.inputBar.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune(key)}))
	return cmd
}

// handleBroadcastSelectKey handles keys in the pane-selection checklist.
func (m *Model) handleBroadcastSelectKey(key string) tea.Cmd {
	switch key {
	case " ":
		// Toggle focused pane in broadcastTo
		if m.focusedPane >= 0 && m.focusedPane < len(m.broadcastTo) {
			m.broadcastTo[m.focusedPane] = !m.broadcastTo[m.focusedPane]
		}
	case "up", "k":
		m.moveFocus(-1, 0)
	case "down", "j":
		m.moveFocus(1, 0)
	case "ctrl+a":
		// Toggle all
		allOn := true
		for _, b := range m.broadcastTo {
			if !b {
				allOn = false
				break
			}
		}
		for i := range m.broadcastTo {
			m.broadcastTo[i] = !allOn
		}
	case "enter", "esc":
		m.focusTarget = FocusBroadcast
		m.inputBar.Focus()
	}
	return nil
}

// handlePasswordKey handles keys when a password overlay is active.
func (m *Model) handlePasswordKey(paneID int, key string) tea.Cmd {
	switch key {
	case "esc":
		m.panes[paneID].PasswordOverlay = false
		delete(m.passwordInputs, paneID)
	case "enter":
		ti, ok := m.passwordInputs[paneID]
		if ok {
			pw := ti.Value()
			m.panes[paneID].PasswordOverlay = false
			delete(m.passwordInputs, paneID)
			return func() tea.Msg {
				return PanePasswordSubmitMsg{PaneID: paneID, Password: pw}
			}
		}
	default:
		if ti, ok := m.passwordInputs[paneID]; ok {
			var cmd tea.Cmd
			ti, cmd = ti.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune(key)}))
			m.passwordInputs[paneID] = ti
			return cmd
		} else {
			// Initialise password input
			ti := textinput.New()
			ti.EchoMode = textinput.EchoPassword
			ti.EchoCharacter = '*'
			ti.Placeholder = "password"
			ti.Focus()
			m.passwordInputs[paneID] = ti
			m.panes[paneID].PasswordOverlay = true
		}
	}
	return nil
}

// handleAddPaneKey handles keys in the add-pane text overlay.
func (m *Model) handleAddPaneKey(key string) tea.Cmd {
	switch key {
	case "esc":
		m.focusTarget = FocusPane
		m.addPaneInput.Blur()
	case "enter":
		target := m.addPaneInput.Value()
		m.focusTarget = FocusPane
		m.addPaneInput.Blur()
		if target != "" {
			slot := m.FirstEmptySlot()
			if slot >= 0 {
				return m.addPaneToSlot(slot, target)
			}
		}
	default:
		var cmd tea.Cmd
		m.addPaneInput, cmd = m.addPaneInput.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune(key)}))
		return cmd
	}
	return nil
}

// handleScrollKey handles keys in scroll and search modes.
func (m *Model) handleScrollKey(p *pane.Pane, key string) tea.Cmd {
	if p.Mode == pane.ModeSearch {
		switch key {
		case "esc", "enter":
			p.Mode = pane.ModeScroll
			p.SearchQuery = ""
		case "n":
			m.scrollSearchNext(p, false)
		case "N":
			m.scrollSearchNext(p, true)
		default:
			// Append to search query
			if len(key) == 1 {
				p.SearchQuery += key
			}
		}
		return nil
	}
	// ModeScroll
	switch key {
	case "up", "k":
		p.Scroll.ScrollUp(1)
	case "down", "j":
		p.Scroll.ScrollDown(1)
	case "pgup":
		p.Scroll.ScrollUp(p.Height() / 2)
	case "pgdown":
		p.Scroll.ScrollDown(p.Height() / 2)
	case "g":
		p.Scroll.ToTop()
	case "G":
		p.Scroll.ToBottom()
		p.Mode = pane.ModeNormal
	case "/":
		p.Mode = pane.ModeSearch
		p.SearchQuery = ""
	case "q", "esc", "enter":
		p.Mode = pane.ModeNormal
		p.Scroll.ToBottom()
	}
	return nil
}

// scrollSearchNext moves through search results.
func (m *Model) scrollSearchNext(p *pane.Pane, reverse bool) {
	if p.SearchQuery == "" {
		return
	}
	hits := p.Scroll.Search(p.SearchQuery)
	if len(hits) == 0 {
		return
	}
	id := p.ID
	cur := m.searchCursor[id]
	if reverse {
		cur = (cur + 1) % len(hits)
	} else {
		cur = (cur - 1 + len(hits)) % len(hits)
	}
	m.searchCursor[id] = cur
	// Scroll to the hit line
	lineIdx := hits[cur]
	total := p.Scroll.Len()
	p.Scroll.ScrollUp(total - lineIdx - p.Height())
}

// sendBroadcast writes data to all panes in broadcastTo.
func (m *Model) sendBroadcast(data []byte) {
	for i, shouldSend := range m.broadcastTo {
		if shouldSend && i < len(m.panes) && m.panes[i].IsActive() {
			m.panes[i].Session.Write(data) //nolint:errcheck
		}
	}
}

// moveFocus moves the focusedPane index by delta (wraps on grid boundaries).
// focusToNextActive moves focus to the first non-closed pane, searching
// forward then backward from the current position.
func (m *Model) focusToNextActive() {
	for i := m.focusedPane + 1; i < len(m.panes); i++ {
		if !m.panes[i].Closed {
			m.focusedPane = i
			return
		}
	}
	for i := m.focusedPane - 1; i >= 0; i-- {
		if !m.panes[i].Closed {
			m.focusedPane = i
			return
		}
	}
	m.focusedPane = -1 // no active panes left
}

func (m *Model) moveFocus(delta, _ int) {
	if len(m.panes) == 0 {
		return
	}
	next := m.focusedPane + delta
	if next < 0 {
		next = 0
	}
	if next >= len(m.panes) {
		next = len(m.panes) - 1
	}
	if !m.panes[next].Closed {
		m.focusedPane = next
	}
}

// nextActivePane finds the next non-closed pane index.
func (m *Model) nextActivePane() int {
	for i, p := range m.panes {
		if !p.Closed {
			return i
		}
	}
	return -1
}

// resizePanes recomputes pane sizes based on the current layout.
func (m Model) resizePanes() Model {
	for i, p := range m.panes {
		if i >= len(m.layout.Panes) {
			break
		}
		rect := m.layout.Panes[i]
		contentW := max(1, rect.Width-2*borderWidth)
		contentH := max(1, rect.Height-2*borderWidth)
		p.Resize(contentH, contentW)
	}
	return m
}

// addPaneToSlot connects a new session to an empty slot.
func (m *Model) addPaneToSlot(slot int, target string) tea.Cmd {
	sess, err := buildSession(target)
	if err != nil {
		return nil
	}
	rect := m.layout.Panes[slot]
	contentW := max(1, rect.Width-2*borderWidth)
	contentH := max(1, rect.Height-2*borderWidth)
	p := pane.New(slot, sess, contentH, contentW)
	m.panes[slot] = p
	m.broadcastTo[slot] = false
	m.focusedPane = slot
	return connectPane(slot, p)
}

// keyBytes converts a bubbletea key string back to raw bytes for session write.
func keyBytes(key string) []byte {
	switch key {
	case "enter":
		return []byte{'\r'}
	case "tab":
		return []byte{'\t'}
	case "backspace":
		return []byte{127}
	case "ctrl+c":
		return []byte{3}
	case "ctrl+d":
		return []byte{4}
	case "ctrl+z":
		return []byte{26}
	case "ctrl+l":
		return []byte{12}
	case "ctrl+a":
		return []byte{1}
	case "ctrl+e":
		return []byte{5}
	case "ctrl+k":
		return []byte{11}
	case "ctrl+u":
		return []byte{21}
	case "ctrl+w":
		return []byte{23}
	case "up":
		return []byte{27, '[', 'A'}
	case "down":
		return []byte{27, '[', 'B'}
	case "right":
		return []byte{27, '[', 'C'}
	case "left":
		return []byte{27, '[', 'D'}
	case "pgup":
		return []byte{27, '[', '5', '~'}
	case "pgdown":
		return []byte{27, '[', '6', '~'}
	case "home":
		return []byte{27, '[', 'H'}
	case "end":
		return []byte{27, '[', 'F'}
	case "delete":
		return []byte{27, '[', '3', '~'}
	case "esc":
		return []byte{27}
	case "f1":
		return []byte{27, 'O', 'P'}
	case "f2":
		return []byte{27, 'O', 'Q'}
	case "f3":
		return []byte{27, 'O', 'R'}
	case "f4":
		return []byte{27, 'O', 'S'}
	case "f5":
		return []byte{27, '[', '1', '5', '~'}
	default:
		if len(key) == 1 {
			return []byte(key)
		}
		// Multi-rune or unknown: best effort
		return []byte(key)
	}
}

// feedScrollBuffer captures newly rendered VTerm rows and adds them to the
// pane's ScrollBuffer. Only rows the cursor moved through are captured;
// when the cursor moves up (clear / vim / readline) nothing is added to
// avoid dumping duplicate or irrelevant screen content.
func feedScrollBuffer(p *pane.Pane, rowBefore, rowAfter, totalRows int) {
	if rowAfter < rowBefore {
		// Cursor moved up (clear, full-screen app, readline edit).
		// Do not dump the visible screen — that would add duplicate lines.
		return
	}
	// Capture rows from rowBefore to rowAfter (inclusive).
	_, cols := p.VTerm.Size()
	for r := rowBefore; r <= rowAfter && r < totalRows; r++ {
		line := renderRowPlain(p, r, cols)
		p.Scroll.AddLine(line)
	}
}

// renderRowPlain renders one VTerm row as plain text (no ANSI) for scroll storage.
func renderRowPlain(p *pane.Pane, row, cols int) string {
	var buf []byte
	for c := 0; c < cols; c++ {
		cell := p.VTerm.Cell(row, c)
		ch := cell.Char
		if ch == 0 {
			ch = ' '
		}
		buf = append(buf, string(ch)...)
	}
	// Trim trailing spaces
	line := string(buf)
	i := len(line)
	for i > 0 && line[i-1] == ' ' {
		i--
	}
	return line[:i]
}

// handleMouseClick focuses the pane at terminal coordinate (x, y), or
// activates the input bar when the user clicks the bottom chrome area.
func (m *Model) handleMouseClick(x, y int) {
	// Click in the input bar area → focus broadcast input bar
	inputBarTop := m.height - inputBarHeight
	if y >= inputBarTop {
		if m.focusTarget == FocusPane {
			m.focusTarget = FocusBroadcast
			m.inputBar.Focus()
		}
		return
	}

	// Click in a pane — find which one
	for i, rect := range m.layout.Panes {
		if i >= len(m.panes) || m.panes[i].Closed {
			continue
		}
		if x >= rect.X && x < rect.X+rect.Width &&
			y >= rect.Y && y < rect.Y+rect.Height {
			m.focusedPane = i
			m.focusTarget = FocusPane
			m.prefixState = PrefixIdle
			return
		}
	}
}

// handleMouseScroll scrolls the pane under (x, y) up or down by 3 lines.
// Scrolling up is only allowed when the buffer has more history than the
// pane height (i.e. there is actually something to reveal).
func (m *Model) handleMouseScroll(x, y int, up bool) {
	for i, rect := range m.layout.Panes {
		if i >= len(m.panes) || m.panes[i].Closed {
			continue
		}
		if x >= rect.X && x < rect.X+rect.Width &&
			y >= rect.Y && y < rect.Y+rect.Height {
			p := m.panes[i]
			h := p.Height()
			if up {
				// Only enter scroll mode if there is history above the live view.
				if !p.Scroll.CanScrollUp(h) {
					return
				}
				if p.Mode == pane.ModeNormal {
					p.Mode = pane.ModeScroll
				}
				p.Scroll.ScrollUp(3)
			} else {
				if p.Mode == pane.ModeNormal {
					return // already at bottom, nothing to do
				}
				p.Scroll.ScrollDown(3)
				if p.Scroll.IsAtBottom() {
					p.Mode = pane.ModeNormal
				}
			}
			return
		}
	}
}

// Ensure time import is used
var _ = time.Second
