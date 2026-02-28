package pane

import (
	"sync"

	"github.com/hinshun/vt10x"
)

// VTerm wraps vt10x.Terminal providing thread-safe access.
type VTerm struct {
	mu   sync.Mutex
	term vt10x.Terminal
	rows int
	cols int
}

// NewVTerm creates a new VTerm with the given dimensions.
func NewVTerm(rows, cols int) *VTerm {
	v := &VTerm{rows: rows, cols: cols}
	v.term = vt10x.New(vt10x.WithSize(cols, rows))
	return v
}

// Write feeds raw terminal data into the vt10x parser.
func (v *VTerm) Write(data []byte) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.term.Write(data) //nolint:errcheck
}

// Resize changes the terminal dimensions.
func (v *VTerm) Resize(rows, cols int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.rows, v.cols = rows, cols
	v.term.Resize(cols, rows)
}

// Cell returns the glyph at (row, col). Thread-safe.
func (v *VTerm) Cell(row, col int) vt10x.Glyph {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.term.Cell(col, row)
}

// Cursor returns the current cursor position as (row, col). Thread-safe.
func (v *VTerm) Cursor() (row, col int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	cur := v.term.Cursor()
	return cur.Y, cur.X
}

// Size returns (rows, cols).
func (v *VTerm) Size() (rows, cols int) { return v.rows, v.cols }
