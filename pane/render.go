package pane

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/hinshun/vt10x"
	"github.com/mattn/go-runewidth"
)

// vt10x glyph mode bit values (unexported in vt10x, mirrored here).
// Must match the iota order in state.go: Reverse, Underline, Bold, Gfx, Italic, Blink, Wrap, Faint.
const (
	glyphAttrReverse   int16 = 1 << 0
	glyphAttrUnderline int16 = 1 << 1
	glyphAttrBold      int16 = 1 << 2
	// glyphAttrGfx    int16 = 1 << 3 (internal to vt10x, not needed for rendering)
	glyphAttrItalic int16 = 1 << 4
	glyphAttrBlink  int16 = 1 << 5
	// glyphAttrWrap   int16 = 1 << 6 (internal to vt10x, not needed for rendering)
	glyphAttrFaint int16 = 1 << 7
)

// Selection describes a text selection range in pane-local cell coordinates.
type Selection struct {
	StartRow, StartCol int
	EndRow, EndCol     int
}

// Normalise returns the selection with start <= end.
func (s Selection) Normalise() (r1, c1, r2, c2 int) {
	r1, c1, r2, c2 = s.StartRow, s.StartCol, s.EndRow, s.EndCol
	if r1 > r2 || (r1 == r2 && c1 > c2) {
		r1, c1, r2, c2 = r2, c2, r1, c1
	}
	return
}

// RenderVTerm renders the VTerm character matrix to a lipgloss string.
// cursorRow/cursorCol: current cursor position (-1 = no cursor).
// sel: optional selection highlight (nil = no selection).
func RenderVTerm(v *VTerm, cursorRow, cursorCol int, sel *Selection) string {
	rows, cols := v.Size()
	lines := make([]string, rows)
	for r := 0; r < rows; r++ {
		lines[r] = renderRow(v, r, cols, cursorRow, cursorCol, sel)
	}
	return strings.Join(lines, "\n")
}

func renderRow(v *VTerm, row, cols, cursorRow, cursorCol int, sel *Selection) string {
	// Pre-compute selection range for this row.
	selC1, selC2 := -1, -1
	if sel != nil {
		r1, c1, r2, c2 := sel.Normalise()
		if row >= r1 && row <= r2 {
			if r1 == r2 {
				selC1, selC2 = c1, c2
			} else if row == r1 {
				selC1, selC2 = c1, cols-1
			} else if row == r2 {
				selC1, selC2 = 0, c2
			} else {
				selC1, selC2 = 0, cols-1
			}
		}
	}

	var sb strings.Builder
	c := 0
	for c < cols {
		cell := v.Cell(row, c)
		ch := cell.Char
		if ch == 0 {
			ch = ' '
		}
		w := runewidth.RuneWidth(ch)
		if w == 0 {
			w = 1
		}

		s := string(ch)
		style := cellStyle(cell)
		// Some TUIs temporarily underline whole-space regions; rendering those
		// literally produces horizontal "rule" artifacts in pane backgrounds.
		if ch == ' ' {
			style = style.Underline(false).Blink(false)
		}
		if row == cursorRow && c == cursorCol {
			// Draw a deterministic block cursor that remains visible even when
			// the cell already uses reverse/video attributes.
			style = style.
				Foreground(lipgloss.Color("#000000")).
				Background(lipgloss.Color("#FFFFFF")).
				Reverse(false)
		}
		// Highlight selected cells.
		if c >= selC1 && c <= selC2 {
			style = style.
				Background(lipgloss.Color("#44475A")).
				Reverse(false)
		}
		sb.WriteString(style.Render(s))
		c += w
	}
	return sb.String()
}

func cellStyle(cell vt10x.Glyph) lipgloss.Style {
	style := lipgloss.NewStyle()

	fg, bg := cell.FG, cell.BG
	if cell.Mode&glyphAttrReverse != 0 {
		// vt10x pre-swaps FG/BG for reverse-video cells. Undo the swap
		// so we can emit SGR 7 (Reverse) and let the real terminal handle
		// the color reversal — this correctly handles mixed cases where
		// one color is explicit and the other is the terminal default.
		fg, bg = cell.BG, cell.FG
	}

	if fg != vt10x.DefaultFG && fg != vt10x.DefaultBG {
		style = style.Foreground(vtColor(fg))
	}
	if bg != vt10x.DefaultBG && bg != vt10x.DefaultFG {
		style = style.Background(vtColor(bg))
	}
	if cell.Mode&glyphAttrBold != 0 {
		style = style.Bold(true)
	}
	if cell.Mode&glyphAttrFaint != 0 {
		style = style.Faint(true)
	}
	if cell.Mode&glyphAttrItalic != 0 {
		style = style.Italic(true)
	}
	if cell.Mode&glyphAttrUnderline != 0 {
		style = style.Underline(true)
	}
	if cell.Mode&glyphAttrBlink != 0 {
		style = style.Blink(true)
	}
	if cell.Mode&glyphAttrReverse != 0 {
		style = style.Reverse(true)
	}
	return style
}

func vtColor(c vt10x.Color) lipgloss.Color {
	if c <= 255 {
		// ANSI 0-15 and 256-color 16-255: pass as index so the
		// terminal applies its own palette (matches user's theme).
		return lipgloss.Color(fmt.Sprintf("%d", c))
	}
	// vt10x uses bit 24 as a sentinel for default colors (DefaultFG, DefaultBG).
	// These should never reach here (filtered by cellStyle), but guard anyway.
	if c&(1<<24) != 0 {
		return lipgloss.Color("")
	}
	// True color (24-bit RGB): vt10x stores as r<<16 | g<<8 | b
	r := (c >> 16) & 0xFF
	g := (c >> 8) & 0xFF
	b := c & 0xFF
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
}

// renderRowPlain renders one VTerm row as plain text (no ANSI).
func renderRowPlain(v *VTerm, row, cols int) string {
	var buf []byte
	for c := 0; c < cols; c++ {
		cell := v.Cell(row, c)
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
