package app

import "github.com/xssh/xssh/session"

// PaneOutputMsg carries raw PTY/SSH output for a specific pane.
type PaneOutputMsg struct {
	PaneID int
	Data   []byte
}

// PaneStatusMsg signals that a pane's session status has changed.
type PaneStatusMsg struct {
	PaneID int
	Status session.Status
}

// TermResizeMsg signals that the terminal window was resized.
type TermResizeMsg struct{}

// ReconnectTickMsg fires on each reconnect attempt timer tick.
type ReconnectTickMsg struct {
	PaneID  int
	Attempt int
}

// PanePasswordRequestMsg signals that a pane needs a password from the user.
type PanePasswordRequestMsg struct {
	PaneID int
}

// PanePasswordSubmitMsg carries the password typed by the user for a pane.
type PanePasswordSubmitMsg struct {
	PaneID   int
	Password string
}

// BroadcastSelectToggleMsg signals that the user toggled a pane in the
// selective-broadcast chooser.
type BroadcastSelectToggleMsg struct {
	PaneID int
}

// AddPaneMsg requests adding a new pane to the first empty slot.
type AddPaneMsg struct {
	Target string // host alias, user@host, or "-" for local
}

// PaneClosed signals that the user closed a specific pane.
type PaneClosedMsg struct {
	PaneID int
}

// SaveGroupMsg requests saving the current session targets as a named group.
type SaveGroupMsg struct {
	Name string
}