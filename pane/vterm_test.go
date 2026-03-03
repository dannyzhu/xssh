package pane

import (
	"strings"
	"testing"
)

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

func TestVTermWideRuneAdvancesCursorWidth(t *testing.T) {
	v := NewVTerm(24, 80)
	v.Write([]byte("😀a"))

	row, col := v.Cursor()
	if row != 0 {
		t.Errorf("cursor row = %d, want 0", row)
	}
	if col != 3 {
		t.Errorf("cursor col = %d, want 3", col)
	}

	cell := v.Cell(0, 2)
	if cell.Char != 'a' {
		t.Errorf("Cell(0,2).Char = %q, want 'a'", cell.Char)
	}
}

func TestVTermRISClearsWholeScreen(t *testing.T) {
	v := NewVTerm(24, 80)
	// Move to column 70 and print a marker.
	v.Write([]byte("\x1b[1;71H"))
	v.Write([]byte("X"))

	// Full terminal reset (RIS) must clear the entire screen.
	v.Write([]byte("\x1bc"))

	cell := v.Cell(0, 70)
	if cell.Char != ' ' {
		t.Fatalf("Cell(0,70).Char = %q, want space after RIS clear", cell.Char)
	}
}

func TestVTermRespondsToCPRQuery(t *testing.T) {
	v := NewVTerm(24, 80)
	v.Write([]byte("\x1b[6n"))

	reply := string(v.DrainReplies())
	if !strings.Contains(reply, "\x1b[1;1R") {
		t.Fatalf("CPR reply = %q, want to contain ESC[1;1R", reply)
	}
}

func TestVTermAltScreenStartsBlank(t *testing.T) {
	v := NewVTerm(24, 80)
	v.Write([]byte("normal"))
	v.Write([]byte("\x1b[?1049h"))

	cell := v.Cell(0, 0)
	if cell.Char != ' ' {
		t.Fatalf("Cell(0,0).Char = %q, want space after entering alt screen", cell.Char)
	}
}

func TestVTermAltScreenRestoreMainBuffer(t *testing.T) {
	v := NewVTerm(24, 80)
	v.Write([]byte("normal"))
	v.Write([]byte("\x1b[?1049h"))
	v.Write([]byte("alt"))
	v.Write([]byte("\x1b[?1049l"))

	c0 := v.Cell(0, 0)
	if c0.Char != 'n' {
		t.Fatalf("Cell(0,0).Char = %q, want 'n' from main buffer after leaving alt", c0.Char)
	}
}

func TestVTermAltScreenReenterStartsBlank(t *testing.T) {
	v := NewVTerm(24, 80)
	v.Write([]byte("normal"))
	v.Write([]byte("\x1b[?1049h"))
	v.Write([]byte("alt"))
	v.Write([]byte("\x1b[?1049l"))
	v.Write([]byte("\x1b[?1049h"))

	cell := v.Cell(0, 0)
	if cell.Char != ' ' {
		t.Fatalf("Cell(0,0).Char = %q, want blank on alt re-enter", cell.Char)
	}
}

func TestVTermZeroWidthRuneDoesNotAdvanceCursor(t *testing.T) {
	v := NewVTerm(24, 80)
	// ZWJ is zero-width. It must not consume a terminal cell.
	v.Write([]byte("a\u200db"))

	_, col := v.Cursor()
	if col != 2 {
		t.Fatalf("cursor col = %d, want 2", col)
	}
	cell := v.Cell(0, 1)
	if cell.Char != 'b' {
		t.Fatalf("Cell(0,1).Char = %q, want 'b'", cell.Char)
	}
}

func TestVTermPreservesSplitUTF8AcrossWrites(t *testing.T) {
	v := NewVTerm(24, 80)
	// "你" in UTF-8 split across two writes.
	v.Write([]byte{0xE4})
	v.Write([]byte{0xBD, 0xA0, 'a'})

	c0 := v.Cell(0, 0)
	if c0.Char != '你' {
		t.Fatalf("Cell(0,0).Char = %q, want '你'", c0.Char)
	}
	_, col := v.Cursor()
	if col != 3 { // '你' width 2 + 'a' width 1
		t.Fatalf("cursor col = %d, want 3", col)
	}
}

func TestVTermSynchronizedOutputBuffersUntilEnd(t *testing.T) {
	v := NewVTerm(24, 80)
	v.Write([]byte("\x1b[?2026hABC"))
	if c := v.Cell(0, 0).Char; c != 0 && c != ' ' {
		t.Fatalf("Cell(0,0).Char = %q, want blank while sync output active", c)
	}

	v.Write([]byte("DEF\x1b[?2026l"))
	if c := v.Cell(0, 0).Char; c != 'A' {
		t.Fatalf("Cell(0,0).Char = %q, want 'A' after sync output flush", c)
	}
	if c := v.Cell(0, 5).Char; c != 'F' {
		t.Fatalf("Cell(0,5).Char = %q, want 'F' after sync output flush", c)
	}
}

func TestVTermSynchronizedOutputEndMarkerSplitAcrossWrites(t *testing.T) {
	v := NewVTerm(24, 80)
	v.Write([]byte("\x1b[?2026hABC\x1b[?20"))
	// Still buffered: nothing should be visible yet.
	if c := v.Cell(0, 0).Char; c != 0 && c != ' ' {
		t.Fatalf("Cell(0,0).Char = %q, want blank while sync output active", c)
	}

	// Complete the marker in the next chunk.
	v.Write([]byte("26l"))
	if c := v.Cell(0, 0).Char; c != 'A' {
		t.Fatalf("Cell(0,0).Char = %q, want 'A' after split marker flush", c)
	}
	if c := v.Cell(0, 2).Char; c != 'C' {
		t.Fatalf("Cell(0,2).Char = %q, want 'C' after split marker flush", c)
	}
}

func TestVTermCSIUWithPrefixDoesNotRestoreCursor(t *testing.T) {
	v := NewVTerm(24, 80)
	v.Write([]byte("hello"))
	v.Write([]byte("\x1b[s"))    // save at col 5
	v.Write([]byte("\x1b[1;1H")) // move to col 0
	v.Write([]byte("\x1b[>1u"))  // kitty keyboard protocol (must NOT restore cursor)
	v.Write([]byte("\x1b[<u"))   // counterpart (must NOT restore cursor)
	v.Write([]byte("X"))         // should land at col 0

	if c := v.Cell(0, 0).Char; c != 'X' {
		t.Fatalf("Cell(0,0).Char = %q, want 'X'", c)
	}
	if c := v.Cell(0, 5).Char; c != ' ' {
		t.Fatalf("Cell(0,5).Char = %q, want space", c)
	}
}

func TestVTermCursorVisibilityFollowsDECTCEM(t *testing.T) {
	v := NewVTerm(24, 80)
	if !v.CursorVisible() {
		t.Fatalf("cursor should be visible by default")
	}

	v.Write([]byte("\x1b[?25l"))
	if v.CursorVisible() {
		t.Fatalf("cursor should be hidden after CSI ?25l")
	}

	v.Write([]byte("\x1b[?25h"))
	if !v.CursorVisible() {
		t.Fatalf("cursor should be visible after CSI ?25h")
	}
}
