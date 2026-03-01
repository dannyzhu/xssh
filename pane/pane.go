package pane

import (
	"github.com/xssh/xssh/session"
)

// Mode represents what the pane is doing beyond normal terminal passthrough.
type Mode int

const (
	ModeNormal   Mode = iota // key input forwarded to session
	ModeScroll               // scrolling history; session still runs
	ModeSearch               // search within scroll history
	ModePassword             // password overlay covering this pane
)

// Pane is one cell in the multi-pane grid. It owns a Session and a VTerm.
type Pane struct {
	ID      int
	Session session.Session
	VTerm   *VTerm
	Scroll  *ScrollBuffer
	Mode    Mode

	// Closed is true after the user explicitly closes this pane (Ctrl+\+x).
	// A closed pane renders as a grey blank cell and can be reused.
	Closed bool

	// PasswordOverlay shows an in-pane password input when true.
	PasswordOverlay bool

	// SearchQuery is the active search string in ModeSearch.
	SearchQuery string

	width, height int
}

// New creates a Pane wired to sess with the given terminal dimensions.
func New(id int, sess session.Session, rows, cols int) *Pane {
	return &Pane{
		ID:      id,
		Session: sess,
		VTerm:   NewVTerm(rows, cols),
		Scroll:  NewScrollBuffer(5000),
		width:   cols,
		height:  rows,
	}
}

// Resize adjusts the pane's terminal size and propagates to the session.
func (p *Pane) Resize(rows, cols int) {
	p.width, p.height = cols, rows
	p.VTerm.Resize(rows, cols)
	if p.Session != nil && p.Session.Status() == session.StatusConnected {
		p.Session.Resize(rows, cols) //nolint:errcheck
	}
}

// Width returns the current column count.
func (p *Pane) Width() int { return p.width }

// Height returns the current row count.
func (p *Pane) Height() int { return p.height }

// IsActive returns true if the pane has a live session.
func (p *Pane) IsActive() bool {
	return !p.Closed && p.Session != nil && p.Session.Status() == session.StatusConnected
}
