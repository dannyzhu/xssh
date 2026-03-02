package pane

// ScrollBuffer holds a capped history of rendered lines and tracks scroll offset.
type ScrollBuffer struct {
	lines    []string
	maxLines int
	offset   int // 0 = at bottom; positive = scrolled up by N lines
}

// NewScrollBuffer creates a ScrollBuffer capped at maxLines.
func NewScrollBuffer(maxLines int) *ScrollBuffer {
	return &ScrollBuffer{maxLines: maxLines}
}

// AddLine appends a line, evicting the oldest if over capacity.
func (b *ScrollBuffer) AddLine(line string) {
	b.lines = append(b.lines, line)
	if len(b.lines) > b.maxLines {
		b.lines = b.lines[len(b.lines)-b.maxLines:]
	}
}

// ScrollUp scrolls toward older history by n lines.
func (b *ScrollBuffer) ScrollUp(n int) {
	b.offset = min(b.offset+n, len(b.lines))
}

// CanScrollUp reports whether there is history above the current view.
func (b *ScrollBuffer) CanScrollUp(height int) bool {
	return len(b.lines) > height && b.offset < len(b.lines)-height
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

// Lines returns a view of up to height lines ending at the scroll position.
func (b *ScrollBuffer) Lines(height int) []string {
	total := len(b.lines)
	if total == 0 || height <= 0 {
		return nil
	}
	// end index (exclusive): distance from bottom determined by offset
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

// Search returns the indices of lines containing substr, newest first.
func (b *ScrollBuffer) Search(substr string) []int {
	var hits []int
	for i := len(b.lines) - 1; i >= 0; i-- {
		if containsSubstr(b.lines[i], substr) {
			hits = append(hits, i)
		}
	}
	return hits
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
