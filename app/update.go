package app

import (
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/xssh/xssh/layout"
	"github.com/xssh/xssh/pane"
	"github.com/xssh/xssh/session"
)

// Init is called once at startup. It connects all sessions and enters alt-screen.
func (m Model) Init() tea.Cmd {
	m.hostCursorShown = false
	return tea.Batch(connectAll(&m), hostCursorCmd(false))
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
		m.layout = layout.Compute(len(m.ActivePanes()), msg.Width, msg.Height, m.reservedHeight(), m.borderMode == BorderShared)
		m = m.resizePanes()

	// ── Pane output ──────────────────────────────────────────────────────────
	case PaneOutputMsg:
		if msg.PaneID >= 0 && msg.PaneID < len(m.panes) {
			p := m.panes[msg.PaneID]
			p.VTerm.Write(msg.Data)
			if replies := p.VTerm.DrainReplies(); len(replies) > 0 && p.Session != nil && p.IsActive() {
				p.Session.Write(replies) //nolint:errcheck
			}
			p.Scroll.AppendRaw(msg.Data)
			// New output arrives — exit scroll mode so cursor is visible
			if p.Mode == pane.ModeScroll || p.Mode == pane.ModeSearch {
				p.Mode = pane.ModeNormal
				p.Scroll.ToBottom()
			}
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
					contentW, contentH := m.paneContentSize(rect)
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
		switch {
		case msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress:
			m.handleMousePress(msg.X, msg.Y)
		case msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionMotion:
			m.handleMouseMotion(msg.X, msg.Y)
		case msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionRelease:
			cmd := m.handleMouseRelease(msg.X, msg.Y)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		case msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown:
			m.handleMouseScroll(msg.X, msg.Y, msg.Button == tea.MouseButtonWheelUp)
		}

	}

	if cmd := m.maybeHostCursorCmd(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func hostCursorCmd(show bool) tea.Cmd {
	return func() tea.Msg {
		if show {
			return tea.ShowCursor()
		}
		return tea.HideCursor()
	}
}

func (m *Model) hostCursorVisible() bool {
	if m.focusTarget == FocusAddPane {
		return true
	}
	if m.focusedPane >= 0 && m.focusedPane < len(m.panes) {
		return m.panes[m.focusedPane].PasswordOverlay
	}
	return false
}

func (m *Model) maybeHostCursorCmd() tea.Cmd {
	show := m.hostCursorVisible()
	if show == m.hostCursorShown {
		return nil
	}
	m.hostCursorShown = show
	return hostCursorCmd(show)
}

// handleKey dispatches keyboard events based on current focus and prefix state.
func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	// Clear any active text selection on keypress.
	m.selPaneIdx = -1

	key := msg.String()

	// ── Paste: forward raw content to the PTY (bypass keyBytes) ─────────────
	// Bubbletea wraps pasted text as "[content]" via Key.String() and sets
	// Paste=true. Detect both the flag and the string pattern as a fallback.
	if isPaste, content := detectPaste(msg, key); isPaste {
		data := []byte(content)
		if m.focusTarget == FocusBroadcast {
			m.sendBroadcast(data)
		} else if m.focusedPane >= 0 && m.focusedPane < len(m.panes) {
			p := m.panes[m.focusedPane]
			if p.IsActive() {
				p.Session.Write(data) //nolint:errcheck
			}
		}
		return nil
	}

	// ── Drop SGR mouse sequences that bubbletea delivered as KeyMsg ──────────
	if isSGRMouseSeq(key) {
		return nil
	}

	// ── Password overlay ─────────────────────────────────────────────────────
	if m.focusedPane >= 0 && m.focusedPane < len(m.panes) {
		p := m.panes[m.focusedPane]
		if p.PasswordOverlay {
			return m.handlePasswordKey(m.focusedPane, key)
		}
	}

	// ── Help overlay ─────────────────────────────────────────────────────────
	if m.showHelp {
		m.showHelp = false
		return nil
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
		idx := int(action - ActionFocusPane1)
		if idx < len(m.panes) && !m.panes[idx].Closed {
			m.focusedPane = idx
			m.zoomedPane = -1
			m.focusTarget = FocusPane
		}

	case ActionFocusUp:
		m.moveFocus(-m.layout.Cols, 0)
		m.focusTarget = FocusPane
	case ActionFocusDown:
		m.moveFocus(m.layout.Cols, 0)
		m.focusTarget = FocusPane
	case ActionFocusLeft:
		m.moveFocus(-1, 0)
		m.focusTarget = FocusPane
	case ActionFocusRight:
		m.moveFocus(1, 0)
		m.focusTarget = FocusPane

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
		if m.focusTarget == FocusBroadcast {
			m.focusTarget = FocusPane
		} else {
			m.focusTarget = FocusBroadcast
			for i, p := range m.panes {
				m.broadcastTo[i] = !p.Closed
			}
		}

	case ActionBroadcastSelect:
		m.focusTarget = FocusBroadcastSelect

	case ActionScrollMode:
		if m.focusedPane >= 0 && m.focusedPane < len(m.panes) {
			p := m.panes[m.focusedPane]
			p.Scroll.Replay(p.Width())
			p.Mode = pane.ModeScroll
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

	case ActionRepaint:
		return tea.ClearScreen

	case ActionDelete:
		// Same as entering scroll mode then searching (plan: Ctrl+\+/ = scroll mode /)
		if m.focusedPane >= 0 && m.focusedPane < len(m.panes) {
			m.panes[m.focusedPane].Mode = pane.ModeSearch
		}
	}
	return nil
}

// handleBroadcastKey forwards every key immediately to all broadcast targets.
func (m *Model) handleBroadcastKey(key string) tea.Cmd {
	// Ctrl+\ enters prefix mode (to exit broadcast via Ctrl+\+b, or use other shortcuts)
	if key == KeyCtrlBackslash {
		m.prefixState = PrefixWaiting
		return nil
	}
	if m.prefixState == PrefixWaiting {
		m.prefixState = PrefixIdle
		action := resolveSecondKey(key)
		return m.dispatchAction(action, key)
	}
	// Forward every key immediately to all broadcast targets
	m.sendBroadcast(keyBytes(key))
	return nil
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
		contentW, contentH := m.paneContentSize(rect)
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
	contentW, contentH := m.paneContentSize(rect)
	p := pane.New(slot, sess, contentH, contentW)
	m.panes[slot] = p
	m.broadcastTo[slot] = false
	m.focusedPane = slot
	return connectPane(slot, p)
}

// isSGRMouseSeq returns true when key looks like an SGR mouse escape
// that bubbletea failed to parse as a MouseMsg and delivered as a KeyMsg.
// These have the form "alt+[<NN;NN;NNM" or "alt+[<NN;NN;NNm".
func isSGRMouseSeq(key string) bool {
	const prefix = "alt+[<"
	if len(key) < len(prefix)+4 { // minimum: alt+[<0;0;0M
		return false
	}
	if key[:len(prefix)] != prefix {
		return false
	}
	last := key[len(key)-1]
	return last == 'M' || last == 'm'
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
		// Strip bubbletea paste brackets "[…]" if present.
		if len(key) > 2 && key[0] == '[' && key[len(key)-1] == ']' {
			return []byte(key[1 : len(key)-1])
		}
		// Multi-rune or unknown: best effort
		return []byte(key)
	}
}

// detectPaste checks whether msg is a paste event.  Bubbletea sets Paste=true
// and wraps the String() result in "[…]".  We check both as a belt-and-suspenders
// measure: the Paste flag, OR the "[…]" pattern on the string representation.
// Returns (true, rawContent) when a paste is detected.
func detectPaste(msg tea.KeyMsg, key string) (bool, string) {
	if msg.Paste {
		// Prefer Key.String() because Bubble Tea may wrap paste payload as
		// "[...]" there even when Runes can vary by terminal/backend.
		if unwrapped, ok := unwrapPasteBrackets(key); ok {
			return true, unwrapped
		}
		raw := string(msg.Runes)
		// Some backends may also place the wrapped form in Runes.
		if unwrapped, ok := unwrapPasteBrackets(raw); ok {
			return true, unwrapped
		}
		return true, raw
	}
	// Fallback: String() wraps paste in "[…]"; a single typed keystroke can
	// never produce that pattern (individual '[' or ']' are len==1).
	if len(key) > 2 && key[0] == '[' && key[len(key)-1] == ']' {
		return true, key[1 : len(key)-1]
	}
	return false, ""
}

func unwrapPasteBrackets(s string) (string, bool) {
	if len(s) > 2 && s[0] == '[' && s[len(s)-1] == ']' {
		return s[1 : len(s)-1], true
	}
	return "", false
}

// hitTestPane returns the pane index and pane-local (row, col) for terminal
// coordinate (x, y). Returns -1 if the click is outside any pane.
func (m *Model) hitTestPane(x, y int) (paneIdx, row, col int) {
	if m.zoomedPane >= 0 {
		// Zoomed: single pane fills the viewport.
		p := m.panes[m.zoomedPane]
		contentW := m.width - 2
		contentH := m.height - m.reservedHeight()
		// Account for status bar (1 row) + top border (1 row).
		localRow := y - statusBarHeight - 1
		localCol := x - 1 // left border
		if localRow >= 0 && localRow < contentH && localCol >= 0 && localCol < contentW {
			_ = p
			return m.zoomedPane, localRow, localCol
		}
		return -1, 0, 0
	}

	for i, rect := range m.layout.Panes {
		if i >= len(m.panes) || m.panes[i].Closed {
			continue
		}
		if x >= rect.X && x < rect.X+rect.Width &&
			y >= rect.Y && y < rect.Y+rect.Height {
			localCol := x - rect.X
			localRow := y - rect.Y
			if m.borderMode == BorderShared {
				// In shared mode, rect already describes content area
				// but Y includes the top border row.
				localRow = y - rect.Y
				localCol = x - rect.X
			} else {
				// Full border mode: subtract border
				localRow = y - rect.Y - borderWidth
				localCol = x - rect.X - borderWidth
			}
			return i, localRow, localCol
		}
	}
	return -1, 0, 0
}

// handleMousePress starts a text selection or focuses a pane.
func (m *Model) handleMousePress(x, y int) {
	// Clear any previous selection.
	m.selecting = false
	m.selPaneIdx = -1

	paneIdx, row, col := m.hitTestPane(x, y)
	if paneIdx < 0 {
		// Click in input bar → enter broadcast
		inputBarTop := m.height - m.inputBarHeight()
		if y >= inputBarTop && m.focusTarget == FocusPane {
			m.focusTarget = FocusBroadcast
			for i, p := range m.panes {
				m.broadcastTo[i] = !p.Closed
			}
		}
		return
	}

	// Focus the clicked pane.
	m.focusedPane = paneIdx
	m.focusTarget = FocusPane
	m.prefixState = PrefixIdle

	// Begin selection.
	m.selecting = true
	m.selPaneIdx = paneIdx
	m.selStartRow = row
	m.selStartCol = col
	m.selEndRow = row
	m.selEndCol = col
}

// handleMouseMotion extends the current text selection.
func (m *Model) handleMouseMotion(x, y int) {
	if !m.selecting || m.selPaneIdx < 0 {
		return
	}
	paneIdx, row, col := m.hitTestPane(x, y)
	if paneIdx != m.selPaneIdx {
		return // don't extend selection across panes
	}
	m.selEndRow = row
	m.selEndCol = col
}

// handleMouseRelease completes selection and copies to clipboard.
func (m *Model) handleMouseRelease(x, y int) tea.Cmd {
	if !m.selecting || m.selPaneIdx < 0 {
		return nil
	}

	// Update final position.
	paneIdx, row, col := m.hitTestPane(x, y)
	if paneIdx == m.selPaneIdx {
		m.selEndRow = row
		m.selEndCol = col
	}

	// If start == end, treat as a simple click (no selection).
	if m.selStartRow == m.selEndRow && m.selStartCol == m.selEndCol {
		m.selecting = false
		m.selPaneIdx = -1
		return nil
	}

	// Extract selected text.
	text := m.extractSelection()
	m.selecting = false

	if text == "" {
		m.selPaneIdx = -1
		return nil
	}

	// Copy to clipboard via OSC 52 (works in most modern terminals).
	return copyToClipboard(text)
}

// extractSelection reads plain text from the selected pane cells.
func (m *Model) extractSelection() string {
	if m.selPaneIdx < 0 || m.selPaneIdx >= len(m.panes) {
		return ""
	}
	p := m.panes[m.selPaneIdx]
	if p.Closed || p.VTerm == nil {
		return ""
	}

	// Normalise: ensure start <= end.
	r1, c1, r2, c2 := m.selStartRow, m.selStartCol, m.selEndRow, m.selEndCol
	if r1 > r2 || (r1 == r2 && c1 > c2) {
		r1, c1, r2, c2 = r2, c2, r1, c1
	}

	rows, cols := p.VTerm.Size()
	// Clamp to pane bounds.
	r1 = clamp(r1, 0, rows-1)
	r2 = clamp(r2, 0, rows-1)
	c1 = clamp(c1, 0, cols-1)
	c2 = clamp(c2, 0, cols-1)

	var sb strings.Builder
	for r := r1; r <= r2; r++ {
		startC := 0
		endC := cols - 1
		if r == r1 {
			startC = c1
		}
		if r == r2 {
			endC = c2
		}
		var lineBuf []byte
		for c := startC; c <= endC; c++ {
			cell := p.VTerm.Cell(r, c)
			ch := cell.Char
			if ch == 0 {
				ch = ' '
			}
			lineBuf = append(lineBuf, string(ch)...)
		}
		line := string(lineBuf)
		// Trim trailing spaces on each line.
		line = strings.TrimRight(line, " ")
		if r > r1 {
			sb.WriteByte('\n')
		}
		sb.WriteString(line)
	}
	return sb.String()
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
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
			w := p.Width()
			// Ensure replay cache is fresh before checking scroll state.
			p.Scroll.Replay(w)
			if up {
				if !p.Scroll.CanScrollUp(h) {
					return
				}
				if p.Mode == pane.ModeNormal {
					p.Mode = pane.ModeScroll
				}
				p.Scroll.ScrollUp(3)
			} else {
				if p.Mode == pane.ModeNormal {
					return
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

// clipboardMsg is emitted after clipboard copy completes.
type clipboardMsg struct{}

// copyToClipboard copies text to the system clipboard using the pbcopy
// command on macOS, falling back to OSC 52 for other systems.
func copyToClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(text)
		_ = cmd.Run()
		return clipboardMsg{}
	}
}
