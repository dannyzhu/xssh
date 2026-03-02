package layout

// PaneRect describes the position and size of one pane cell in the grid.
type PaneRect struct {
	X, Y          int
	Width, Height int
	Empty         bool // true if this slot has no active pane
}

// Layout holds the computed grid for n panes.
type Layout struct {
	Rows, Cols int
	Panes      []PaneRect // len = Rows*Cols, in row-major order
}

// layoutTable maps pane count → (rows, cols) grid dimensions.
var layoutTable = [10][2]int{
	0: {0, 0}, // unused
	1: {1, 1},
	2: {1, 2},
	3: {2, 2},
	4: {2, 2},
	5: {3, 2},
	6: {3, 2},
	7: {3, 3},
	8: {3, 3},
	9: {3, 3},
}

// Compute calculates pane rectangles for n active panes inside a terminal
// of (termWidth × termHeight) characters. reservedRows is the total number
// of terminal rows consumed by chrome (status bar + input bar).
// sharedBorders: if true, adjacent panes share a single border line instead
// of each having independent borders, saving space.
func Compute(n, termWidth, termHeight, reservedRows int, sharedBorders bool) Layout {
	if n < 1 {
		n = 1
	}
	if n > 9 {
		n = 9
	}
	rc := layoutTable[n]
	rows, cols := rc[0], rc[1]
	total := rows * cols

	contentHeight := termHeight - reservedRows

	if sharedBorders {
		// Shared borders: total border cols = cols+1, border rows = rows+1.
		// PaneRect stores content-only dimensions (no border included).
		availW := termWidth - (cols + 1)
		availH := contentHeight - (rows + 1)

		paneW := availW / cols
		paneH := availH / rows

		lastColW := availW - paneW*(cols-1)
		lastRowH := availH - paneH*(rows-1)

		// X/Y positions account for the border chars.
		panes := make([]PaneRect, total)
		idx := 0
		yOff := 1 + 1 // status bar row + top border row
		for r := 0; r < rows; r++ {
			h := paneH
			if r == rows-1 {
				h = lastRowH
			}
			xOff := 1 // left border column
			for c := 0; c < cols; c++ {
				w := paneW
				if c == cols-1 {
					w = lastColW
				}
				panes[idx] = PaneRect{
					X:      xOff,
					Y:      yOff,
					Width:  w,
					Height: h,
					Empty:  idx >= n,
				}
				xOff += w + 1 // content + divider
				idx++
			}
			yOff += h + 1 // content + divider
		}
		return Layout{Rows: rows, Cols: cols, Panes: panes}
	}

	// Full (independent) borders: each pane includes its own border chars.
	paneH := contentHeight / rows
	paneW := termWidth / cols

	// Last row/col absorbs any remainder so content fills the terminal exactly.
	lastRowH := contentHeight - paneH*(rows-1)
	lastColW := termWidth - paneW*(cols-1)

	panes := make([]PaneRect, total)
	idx := 0
	for r := 0; r < rows; r++ {
		h := paneH
		if r == rows-1 {
			h = lastRowH
		}
		for c := 0; c < cols; c++ {
			w := paneW
			if c == cols-1 {
				w = lastColW
			}
			panes[idx] = PaneRect{
				X:      c * paneW,
				Y:      1 + r*paneH, // +1 for status bar
				Width:  w,
				Height: h,
				Empty:  idx >= n,
			}
			idx++
		}
	}
	return Layout{Rows: rows, Cols: cols, Panes: panes}
}
