package app

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/xssh/xssh/layout"
	"github.com/xssh/xssh/pane"
)

// BorderMode controls how adjacent pane borders are rendered.
type BorderMode int

const (
	BorderShared BorderMode = iota // default: shared single-line dividers
	BorderFull                     // independent borders per pane
)

// FocusTarget describes which UI element currently receives keyboard input.
type FocusTarget int

const (
	FocusPane            FocusTarget = iota // key input forwarded to the focused pane
	FocusBroadcast                          // global broadcast input bar
	FocusSelector                           // interactive host selector overlay
	FocusBroadcastSelect                    // Ctrl+\+m: pane-selection checklist
	FocusAddPane                            // Ctrl+\+e: add-pane text input overlay
)

// PrefixState tracks whether we're waiting for the second key of a Ctrl+\ chord.
type PrefixState int

const (
	PrefixIdle    PrefixState = iota // normal input forwarding
	PrefixWaiting                    // Ctrl+\ pressed, awaiting second key
)

// Model is the top-level bubbletea model for xssh.
type Model struct {
	// Pane slice — always len = Rows*Cols from layout; closed slots have Pane.Closed = true.
	panes  []*pane.Pane
	layout layout.Layout

	// focusedPane is the index into panes of the currently active pane (-1 = none).
	focusedPane int
	focusTarget FocusTarget
	prefixState PrefixState

	// zoomedPane: -1 = no zoom; ≥0 = index of the full-screen pane.
	zoomedPane int

	// borderMode controls shared vs independent pane borders.
	borderMode BorderMode

	// Broadcast
	broadcastTo []bool // len = len(panes); true = include in broadcast

	// Password overlays: one textinput per pane that needs it.
	passwordInputs map[int]textinput.Model

	// Add-pane overlay input
	addPaneInput textinput.Model

	// Help overlay visibility
	showHelp bool

	// Terminal dimensions
	width, height int
	ready         bool

	// reconnectAttempts tracks per-pane retry count
	reconnectAttempts map[int]int

	// searchQuery is the active scroll-mode search string (shared via Pane.SearchQuery)
	searchResults map[int][]int // paneID → matching line indices
	searchCursor  map[int]int   // paneID → current result index
}

// NewModel constructs an initial Model with no panes.
func NewModel(borderMode BorderMode) Model {
	addInput := textinput.New()
	addInput.Placeholder = "user@host or ssh-alias"

	return Model{
		focusedPane:       -1,
		zoomedPane:        -1,
		focusTarget:       FocusBroadcast,
		borderMode:        borderMode,
		addPaneInput:      addInput,
		passwordInputs:    make(map[int]textinput.Model),
		reconnectAttempts: make(map[int]int),
		searchResults:     make(map[int][]int),
		searchCursor:      make(map[int]int),
	}
}

// inputBarHeight returns the total rows used by the input bar.
func (m Model) inputBarHeight() int {
	if m.borderMode == BorderShared {
		return inputBarHeightShared
	}
	return inputBarHeightFull
}

// reservedHeight returns the total rows reserved for chrome (status + input).
func (m Model) reservedHeight() int {
	return statusBarHeight + m.inputBarHeight()
}

// paneContentSize returns the content width and height for a pane rect,
// accounting for border mode. In shared mode, rect dimensions are already
// content-only. In full mode, subtract 2*borderWidth for each border.
func (m Model) paneContentSize(rect layout.PaneRect) (int, int) {
	if m.borderMode == BorderShared {
		return max(1, rect.Width), max(1, rect.Height)
	}
	return max(1, rect.Width-2*borderWidth), max(1, rect.Height-2*borderWidth)
}

// ActivePanes returns the slice of non-closed panes.
func (m *Model) ActivePanes() []*pane.Pane {
	var out []*pane.Pane
	for _, p := range m.panes {
		if !p.Closed {
			out = append(out, p)
		}
	}
	return out
}

// FirstEmptySlot returns the index of the first closed/empty pane slot, or -1.
func (m *Model) FirstEmptySlot() int {
	for i, p := range m.panes {
		if p.Closed {
			return i
		}
	}
	return -1
}

// BroadcastCount returns the number of panes included in broadcast.
func (m *Model) BroadcastCount() int {
	n := 0
	for _, b := range m.broadcastTo {
		if b {
			n++
		}
	}
	return n
}
