package pane

import "testing"

func TestVTermWrite(t *testing.T) {
	v := NewVTerm(24, 80)
	v.Write([]byte("Hello"))

	cell := v.Cell(0, 0)
	if cell.Char != 'H' {
		t.Errorf("Cell(0,0).Char = %q, want 'H'", cell.Char)
	}
}

func TestVTermResize(t *testing.T) {
	v := NewVTerm(24, 80)
	v.Resize(30, 120)
	rows, cols := v.Size()
	if rows != 30 || cols != 120 {
		t.Errorf("Size = (%d,%d), want (30,120)", rows, cols)
	}
}

func TestVTermCursor(t *testing.T) {
	v := NewVTerm(24, 80)
	v.Write([]byte("AB"))
	row, col := v.Cursor()
	if row != 0 {
		t.Errorf("cursor row = %d, want 0", row)
	}
	if col != 2 {
		t.Errorf("cursor col = %d, want 2", col)
	}
}

func TestVTermMultiLine(t *testing.T) {
	v := NewVTerm(24, 80)
	v.Write([]byte("Line1\r\nLine2"))
	cell := v.Cell(1, 0)
	if cell.Char != 'L' {
		t.Errorf("Cell(1,0).Char = %q, want 'L'", cell.Char)
	}
}
