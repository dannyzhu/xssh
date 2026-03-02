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
		l := Compute(c.n, 200, 50, 3, false)
		if l.Rows != c.wantRows || l.Cols != c.wantCols {
			t.Errorf("Compute(%d): got %dx%d, want %dx%d",
				c.n, l.Rows, l.Cols, c.wantRows, c.wantCols)
		}
	}
}

func TestComputePaneCount(t *testing.T) {
	for n := 1; n <= 9; n++ {
		l := Compute(n, 200, 53, 3, false)
		expected := l.Rows * l.Cols
		if len(l.Panes) != expected {
			t.Errorf("Compute(%d): len(Panes) = %d, want %d", n, len(l.Panes), expected)
		}
	}
}

func TestComputePaneSizes(t *testing.T) {
	// 4 panes → 2x2 grid; 200 wide, 53 high → content = 50; each pane ~100x25
	l := Compute(4, 200, 53, 3, false)
	for i, p := range l.Panes {
		if p.Width != 100 || p.Height != 25 {
			t.Errorf("Pane[%d]: size=(%d,%d), want (100,25)", i, p.Width, p.Height)
		}
	}
}

func TestComputeEmptySlots(t *testing.T) {
	// 3 panes → 2x2 grid (4 slots), 1 empty
	l := Compute(3, 200, 53, 3, false)
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
	l := Compute(9, 200, 53, 3, false)
	for i, p := range l.Panes {
		if p.Empty {
			t.Errorf("Pane[%d] should not be empty for n=9", i)
		}
	}
}

func TestComputePositions(t *testing.T) {
	// Single pane should start at Y=1 (after status bar) and X=0
	l := Compute(1, 80, 30, 3, false)
	if l.Panes[0].X != 0 || l.Panes[0].Y != 1 {
		t.Errorf("single pane origin = (%d,%d), want (0,1)", l.Panes[0].X, l.Panes[0].Y)
	}
}

func TestComputeFillsWidth(t *testing.T) {
	// The sum of column widths should equal termWidth
	l := Compute(2, 201, 30, 3, false) // odd width, 1x2 grid
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
	l := Compute(2, 100, 30, 3, false)
	contentH := 30 - 3
	if l.Panes[0].Height != contentH {
		t.Errorf("pane height = %d, want %d", l.Panes[0].Height, contentH)
	}
}

// ── Shared border tests ──────────────────────────────────────────────────────

func TestSharedBorderPaneSizes(t *testing.T) {
	// 4 panes → 2x2 grid, 200 wide, 53 high, reserved=3
	// content height = 53-3 = 50; avail = 50 - (2+1) = 47; each row = 23, last = 24
	// avail width = 200 - (2+1) = 197; each col = 98, last = 99
	l := Compute(4, 200, 53, 3, true)
	if l.Rows != 2 || l.Cols != 2 {
		t.Fatalf("expected 2x2, got %dx%d", l.Rows, l.Cols)
	}
	// Width: 197/2 = 98, last = 197-98 = 99
	if l.Panes[0].Width != 98 || l.Panes[1].Width != 99 {
		t.Errorf("widths: pane0=%d pane1=%d, want 98,99", l.Panes[0].Width, l.Panes[1].Width)
	}
	// Height: 47/2 = 23, last = 47-23 = 24
	if l.Panes[0].Height != 23 || l.Panes[2].Height != 24 {
		t.Errorf("heights: pane0=%d pane2=%d, want 23,24", l.Panes[0].Height, l.Panes[2].Height)
	}
}

func TestSharedBorderFillsTerminal(t *testing.T) {
	// Total rendered width should equal termWidth:
	// cols+1 border chars + sum of content widths
	l := Compute(4, 200, 53, 3, true)
	totalContentW := 0
	for c := 0; c < l.Cols; c++ {
		totalContentW += l.Panes[c].Width
	}
	totalW := totalContentW + l.Cols + 1
	if totalW != 200 {
		t.Errorf("total rendered width = %d, want 200", totalW)
	}

	totalContentH := 0
	for r := 0; r < l.Rows; r++ {
		totalContentH += l.Panes[r*l.Cols].Height
	}
	totalH := totalContentH + l.Rows + 1
	wantH := 53 - 3
	if totalH != wantH {
		t.Errorf("total rendered height = %d, want %d", totalH, wantH)
	}
}

func TestSharedBorderPositions(t *testing.T) {
	// 2 panes → 1x2 grid, 100 wide
	// avail = 100 - 3 = 97; paneW = 48, last = 49
	// pane0: X=1 (left border), pane1: X=1+48+1=50
	l := Compute(2, 100, 30, 3, true)
	if l.Panes[0].X != 1 {
		t.Errorf("pane0.X = %d, want 1", l.Panes[0].X)
	}
	expectedX1 := 1 + l.Panes[0].Width + 1
	if l.Panes[1].X != expectedX1 {
		t.Errorf("pane1.X = %d, want %d", l.Panes[1].X, expectedX1)
	}
}
