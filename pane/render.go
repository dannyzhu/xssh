package pane

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/hinshun/vt10x"
	"github.com/mattn/go-runewidth"
)

// vt10x glyph mode bit values (unexported in vt10x, mirrored here).
const (
	glyphAttrReverse   int16 = 1 << 0
	glyphAttrUnderline int16 = 1 << 1
	glyphAttrBold      int16 = 1 << 2
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
	if cell.Mode&glyphAttrUnderline != 0 {
		style = style.Underline(true)
	}
	if cell.Mode&glyphAttrReverse != 0 {
		style = style.Reverse(true)
	}
	return style
}

func vtColor(c vt10x.Color) lipgloss.Color {
	if c < 16 {
		return lipgloss.Color(ansi16[c])
	}
	// 256-color or truecolor: pass as ANSI index
	return lipgloss.Color(fmt.Sprintf("%d", c))
}

var ansi16 = [16]string{
	"#000000", "#800000", "#008000", "#808000",
	"#000080", "#800080", "#008080", "#c0c0c0",
	"#808080", "#ff0000", "#00ff00", "#ffff00",
	"#0000ff", "#ff00ff", "#00ffff", "#ffffff",
}

// SnapshotVTerm returns every VTerm row as two parallel slices:
//   - styled: ANSI-styled strings (for coloured display)
//   - plain:  plain text (for searching / width measurement)
//
// No cursor highlight is baked in.
func SnapshotVTerm(v *VTerm) (styled, plain []string) {
	rows, cols := v.Size()
	styled = make([]string, rows)
	plain = make([]string, rows)
	for r := 0; r < rows; r++ {
		styled[r] = renderRow(v, r, cols, -1, -1)
		plain[r] = renderRowPlain(v, r, cols)
	}
	return
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
