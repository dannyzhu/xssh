package pane

import (
	"testing"
)

// --- ScrollBuffer tests ---

func TestScrollBufferMaxLines(t *testing.T) {
	b := NewScrollBuffer(3)
	b.AddLine("a")
	b.AddLine("b")
	b.AddLine("c")
	b.AddLine("d") // evicts "a"
	if b.Len() != 3 {
		t.Errorf("Len = %d, want 3", b.Len())
	}
	lines := b.Lines(3)
	if lines[0] != "b" {
		t.Errorf("oldest line = %q, want 'b'", lines[0])
	}
}

func TestScrollBufferScrollUp(t *testing.T) {
	b := NewScrollBuffer(100)
	for i := 0; i < 10; i++ {
		b.AddLine("line")
	}
	b.ScrollUp(3)
	if b.Offset() != 3 {
		t.Errorf("offset = %d, want 3", b.Offset())
	}
}

func TestScrollBufferScrollDown(t *testing.T) {
	b := NewScrollBuffer(100)
	for i := 0; i < 10; i++ {
		b.AddLine("line")
	}
	b.ScrollUp(5)
	b.ScrollDown(2)
	if b.Offset() != 3 {
		t.Errorf("offset = %d, want 3", b.Offset())
	}
}

func TestScrollBufferScrollDownFloor(t *testing.T) {
	b := NewScrollBuffer(100)
	b.AddLine("a")
	b.ScrollDown(99) // should not go below 0
	if b.Offset() != 0 {
		t.Errorf("offset = %d, want 0 (floor)", b.Offset())
	}
}

func TestScrollBufferScrollUpCap(t *testing.T) {
	b := NewScrollBuffer(100)
	b.AddLine("a")
	b.AddLine("b")
	b.ScrollUp(999) // should not exceed Len
	if b.Offset() != 2 {
		t.Errorf("offset = %d, want 2 (cap at len)", b.Offset())
	}
}

func TestScrollBufferToBottom(t *testing.T) {
	b := NewScrollBuffer(100)
	b.AddLine("a")
	b.ScrollUp(1)
	b.ToBottom()
	if !b.IsAtBottom() {
		t.Error("should be at bottom after ToBottom()")
	}
}

func TestScrollBufferLines(t *testing.T) {
	b := NewScrollBuffer(100)
	for i := 0; i < 5; i++ {
		b.AddLine("line")
	}
	got := b.Lines(3)
	if len(got) != 3 {
		t.Errorf("Lines(3) = %d items, want 3", len(got))
	}
}

func TestScrollBufferSearch(t *testing.T) {
	b := NewScrollBuffer(100)
	b.AddLine("hello world")
	b.AddLine("goodbye world")
	b.AddLine("nothing here")
	hits := b.Search("world")
	if len(hits) != 2 {
		t.Errorf("Search('world') = %d hits, want 2", len(hits))
	}
}

// --- Pane tests ---

func TestPaneNew(t *testing.T) {
	p := New(0, nil, 24, 80)
	if p.ID != 0 {
		t.Errorf("ID = %d, want 0", p.ID)
	}
	if p.Width() != 80 || p.Height() != 24 {
		t.Errorf("size = (%d,%d), want (80,24)", p.Width(), p.Height())
	}
}

func TestPaneResize(t *testing.T) {
	p := New(1, nil, 24, 80)
	p.Resize(30, 120)
	if p.Width() != 120 || p.Height() != 30 {
		t.Errorf("after resize: (%d,%d), want (120,30)", p.Width(), p.Height())
	}
	rows, cols := p.VTerm.Size()
	if rows != 30 || cols != 120 {
		t.Errorf("VTerm size = (%d,%d), want (30,120)", rows, cols)
	}
}

func TestPaneIsActive(t *testing.T) {
	p := New(0, nil, 24, 80)
	if p.IsActive() {
		t.Error("pane with nil session should not be active")
	}
	p.Closed = true
	if p.IsActive() {
		t.Error("closed pane should not be active")
	}
}

func TestPaneModeDefault(t *testing.T) {
	p := New(0, nil, 24, 80)
	if p.Mode != ModeNormal {
		t.Errorf("default mode = %v, want ModeNormal", p.Mode)
	}
}
