package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestFitToViewport_PadsToRequestedSize(t *testing.T) {
	got := fitToViewport("a\nb", 4, 3)
	lines := strings.Split(got, "\n")
	if len(lines) != 3 {
		t.Fatalf("line count = %d, want 3", len(lines))
	}
	for i, line := range lines {
		if w := lipgloss.Width(line); w != 4 {
			t.Fatalf("line %d width = %d, want 4: %q", i, w, line)
		}
	}
}

func TestFitToViewport_TruncatesExtraLines(t *testing.T) {
	got := fitToViewport("1\n2\n3", 2, 2)
	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Fatalf("line count = %d, want 2", len(lines))
	}
	if !strings.HasPrefix(lines[0], "1") || !strings.HasPrefix(lines[1], "2") {
		t.Fatalf("unexpected lines: %#v", lines)
	}
}
