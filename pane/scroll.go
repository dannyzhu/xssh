package pane

import "github.com/charmbracelet/lipgloss"

// scrollLine holds both an ANSI-styled representation (for coloured display)
// and a plain-text copy (for searching and width measurement).
type scrollLine struct {
	styled string
	plain  string
}

// ScrollBuffer holds a capped history of rendered lines and tracks scroll offset.
// Lines are stored with ANSI styling so scroll-mode display preserves terminal colours.
type ScrollBuffer struct {
	lines    []scrollLine
	maxLines int
	offset   int // 0 = at bottom; positive = scrolled up by N lines

	// prevPlain / prevStyledLines are the previous VTerm screen snapshot.
	// Used by UpdateFromScreen to detect which rows scrolled off the top.
	prevPlain       []string
	prevStyledLines []string
}

// NewScrollBuffer creates a ScrollBuffer capped at maxLines.
func NewScrollBuffer(maxLines int) *ScrollBuffer {
	return &ScrollBuffer{maxLines: maxLines}
}

// UpdateFromScreen takes a new VTerm screen snapshot (styled + plain slices
// of equal length) and detects lines that scrolled off the top compared to
// the previous snapshot.  Those lines are added to the history buffer.
func (b *ScrollBuffer) UpdateFromScreen(styled, plain []string) {
	if len(b.prevPlain) > 0 && len(plain) > 0 {
		scrolled := b.detectScroll(plain)
		for i := 0; i < scrolled; i++ {
			b.addLine(b.prevStyled(i), b.prevPlainLine(i))
		}
	}
	// Store current snapshot for next diff.
	b.prevPlain = make([]string, len(plain))
	copy(b.prevPlain, plain)
	b.prevStyledLines = make([]string, len(styled))
	copy(b.prevStyledLines, styled)
}

// detectScroll determines how many lines from the top of the previous screen
// have scrolled off, by finding where the new screen's top line appears in
// the old screen.
func (b *ScrollBuffer) detectScroll(newPlain []string) int {
	if len(newPlain) == 0 || len(b.prevPlain) == 0 {
		return 0
	}
	// If the top row didn't change, no scrolling happened — skip.
	// This avoids false positives from in-place edits or prompt redraws.
	if newPlain[0] == b.prevPlain[0] {
		return 0
	}
	// The top row changed — try to find a shift amount such that
	// newPlain[0..minMatch) == prevPlain[shift..shift+minMatch).
	// Combined with the top-row-must-change guard above, a 2-row
	// consecutive match is sufficient to avoid false positives while
	// still detecting scroll when new content appears below.
	minMatch := 2
	if minMatch > len(newPlain) {
		minMatch = len(newPlain)
	}
	for shift := 1; shift+minMatch <= len(b.prevPlain); shift++ {
		match := true
		for j := 0; j < minMatch; j++ {
			if newPlain[j] != b.prevPlain[shift+j] {
				match = false
				break
			}
		}
		if match {
			return shift
		}
	}
	// Top changed but no matching shift found (clear screen, full redraw).
	return 0
}

func (b *ScrollBuffer) prevStyled(i int) string {
	if i < len(b.prevStyledLines) {
		return b.prevStyledLines[i]
	}
	return ""
}

func (b *ScrollBuffer) prevPlainLine(i int) string {
	if i < len(b.prevPlain) {
		return b.prevPlain[i]
	}
	return ""
}

func (b *ScrollBuffer) addLine(styled, plain string) {
	b.lines = append(b.lines, scrollLine{styled: styled, plain: plain})
	if len(b.lines) > b.maxLines {
		b.lines = b.lines[len(b.lines)-b.maxLines:]
	}
}

// AddLine adds a single plain-text line (for backward compat / tests).
func (b *ScrollBuffer) AddLine(line string) {
	b.addLine(line, line)
}

// ScrollUp scrolls toward older history by n lines.
func (b *ScrollBuffer) ScrollUp(n int) {
	b.offset = min(b.offset+n, len(b.lines))
}

// ScrollDown scrolls toward newer history by n lines.
func (b *ScrollBuffer) ScrollDown(n int) {
	b.offset = max(0, b.offset-n)
}

// ToBottom jumps to the most recent output.
func (b *ScrollBuffer) ToBottom() { b.offset = 0 }

// ToTop jumps to the oldest available line.
func (b *ScrollBuffer) ToTop() { b.offset = len(b.lines) }

// IsAtBottom returns true when no scroll offset is applied.
func (b *ScrollBuffer) IsAtBottom() bool { return b.offset == 0 }

// Offset returns the current scroll offset.
func (b *ScrollBuffer) Offset() int { return b.offset }

// Len returns the number of stored lines.
func (b *ScrollBuffer) Len() int { return len(b.lines) }

// CanScrollUp reports whether there is history above the current view.
func (b *ScrollBuffer) CanScrollUp(height int) bool {
	return len(b.lines) > height && b.offset < len(b.lines)-height
}

// Lines returns a view of up to height plain-text lines ending at the scroll position.
func (b *ScrollBuffer) Lines(height int) []string {
	sl := b.sliceLines(height)
	out := make([]string, len(sl))
	for i, l := range sl {
		out[i] = l.plain
	}
	return out
}

// StyledLines returns a view of up to height ANSI-styled lines at the scroll position.
func (b *ScrollBuffer) StyledLines(height int) []string {
	sl := b.sliceLines(height)
	out := make([]string, len(sl))
	for i, l := range sl {
		out[i] = l.styled
	}
	return out
}

func (b *ScrollBuffer) sliceLines(height int) []scrollLine {
	total := len(b.lines)
	if total == 0 || height <= 0 {
		return nil
	}
	end := total - b.offset
	if end <= 0 {
		end = 0
	}
	start := end - height
	if start < 0 {
		start = 0
	}
	if end > total {
		end = total
	}
	return b.lines[start:end]
}

// Search returns the indices of lines containing substr (plain text), newest first.
func (b *ScrollBuffer) Search(substr string) []int {
	var hits []int
	for i := len(b.lines) - 1; i >= 0; i-- {
		if containsSubstr(b.lines[i].plain, substr) {
			hits = append(hits, i)
		}
	}
	return hits
}

// VisibleWidth returns the printable width of a styled line.
func VisibleWidth(s string) int {
	return lipgloss.Width(s)
}

func containsSubstr(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
