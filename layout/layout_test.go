package layout

import "testing"

func TestComputeLayouts(t *testing.T) {
	cases := []struct {
		n              int
		wantRows, wantCols int
	}{
		{1, 1, 1}, {2, 1, 2}, {3, 2, 2},
		{4, 2, 2}, {5, 3, 2}, {6, 3, 2},
		{7, 3, 3}, {8, 3, 3}, {9, 3, 3},
	}
	for _, c := range cases {
		l := Compute(c.n, 200, 50)
		if l.Rows != c.wantRows || l.Cols != c.wantCols {
			t.Errorf("Compute(%d): got %dx%d, want %dx%d",
				c.n, l.Rows, l.Cols, c.wantRows, c.wantCols)
		}
	}
}

func TestComputePaneCount(t *testing.T) {
	for n := 1; n <= 9; n++ {
		l := Compute(n, 200, 53)
		expected := l.Rows * l.Cols
		if len(l.Panes) != expected {
			t.Errorf("Compute(%d): len(Panes) = %d, want %d", n, len(l.Panes), expected)
		}
	}
}

func TestComputePaneSizes(t *testing.T) {
	// 4 panes → 2x2 grid; 200 wide, 53 high → content = 50; each pane ~100x25
	l := Compute(4, 200, 53)
	for i, p := range l.Panes {
		if p.Width != 100 || p.Height != 25 {
			t.Errorf("Pane[%d]: size=(%d,%d), want (100,25)", i, p.Width, p.Height)
		}
	}
}

func TestComputeEmptySlots(t *testing.T) {
	// 3 panes → 2x2 grid (4 slots), 1 empty
	l := Compute(3, 200, 53)
	emptyCount := 0
	for _, p := range l.Panes {
		if p.Empty {
			emptyCount++
		}
	}
	if emptyCount != 1 {
		t.Errorf("expected 1 empty slot, got %d", emptyCount)
	}
}

func TestComputeNoPanesEmpty_Full9(t *testing.T) {
	l := Compute(9, 200, 53)
	for i, p := range l.Panes {
		if p.Empty {
			t.Errorf("Pane[%d] should not be empty for n=9", i)
		}
	}
}

func TestComputePositions(t *testing.T) {
	// Single pane should start at Y=1 (after status bar) and X=0
	l := Compute(1, 80, 30)
	if l.Panes[0].X != 0 || l.Panes[0].Y != 1 {
		t.Errorf("single pane origin = (%d,%d), want (0,1)", l.Panes[0].X, l.Panes[0].Y)
	}
}

func TestComputeFillsWidth(t *testing.T) {
	// The sum of column widths should equal termWidth
	l := Compute(2, 201, 30) // odd width, 1x2 grid
	totalW := 0
	// Row 0 has 2 panes (1 row, 2 cols)
	for _, p := range l.Panes {
		totalW += p.Width
	}
	if totalW != 201 {
		t.Errorf("total width = %d, want 201", totalW)
	}
}

func TestComputeFillsHeight(t *testing.T) {
	// For 2 panes in 1x2 grid — both panes are in the same row, height = content
	l := Compute(2, 100, 30)
	contentH := 30 - 3
	if l.Panes[0].Height != contentH {
		t.Errorf("pane height = %d, want %d", l.Panes[0].Height, contentH)
	}
}
