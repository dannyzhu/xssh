package pane

import (
	"strings"
	"testing"
)

func TestRenderVTermBasic(t *testing.T) {
	v := NewVTerm(4, 20)
	v.Write([]byte("Hello"))
	got := RenderVTerm(v, -1, -1)
	if !strings.Contains(got, "Hello") {
		t.Errorf("RenderVTerm output does not contain 'Hello'")
	}
}

func TestRenderVTermCursor(t *testing.T) {
	v := NewVTerm(4, 20)
	v.Write([]byte("AB"))
	// cursor is at (0, 2) after writing "AB"
	got := RenderVTerm(v, 0, 2)
	if !strings.Contains(got, "A") || !strings.Contains(got, "B") {
		t.Errorf("RenderVTerm cursor output missing text: %q", got)
	}
}

func TestRenderVTermMultiLine(t *testing.T) {
	v := NewVTerm(4, 20)
	v.Write([]byte("Line1\r\nLine2"))
	got := RenderVTerm(v, -1, -1)
	if !strings.Contains(got, "Line1") {
		t.Errorf("missing Line1 in output")
	}
	if !strings.Contains(got, "Line2") {
		t.Errorf("missing Line2 in output")
	}
}

func TestRenderVTerm_DoesNotSkipCellsAfterWideRune(t *testing.T) {
	v := NewVTerm(2, 10)
	v.Write([]byte("😀a"))

	got := RenderVTerm(v, -1, -1)
	if !strings.Contains(got, "a") {
		t.Fatalf("rendered output skipped cell after wide rune: %q", got)
	}
}
