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

// RenderVTerm renders the VTerm character matrix to a lipgloss string.
// cursorRow/cursorCol: current cursor position (-1 = no cursor).
func RenderVTerm(v *VTerm, cursorRow, cursorCol int) string {
	rows, cols := v.Size()
	lines := make([]string, rows)
	for r := 0; r < rows; r++ {
		lines[r] = renderRow(v, r, cols, cursorRow, cursorCol)
	}
	return strings.Join(lines, "\n")
}

func renderRow(v *VTerm, row, cols, cursorRow, cursorCol int) string {
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
		if row == cursorRow && c == cursorCol {
			style = style.Reverse(true)
		}
		sb.WriteString(style.Render(s))
		c += w
	}
	return sb.String()
}

func cellStyle(cell vt10x.Glyph) lipgloss.Style {
	style := lipgloss.NewStyle()
	if cell.FG != vt10x.DefaultFG {
		style = style.Foreground(vtColor(cell.FG))
	}
	if cell.BG != vt10x.DefaultBG {
		style = style.Background(vtColor(cell.BG))
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
