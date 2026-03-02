package pane

import "github.com/charmbracelet/lipgloss"

const replayHeight = 10000 // rows for the replay VTerm

// ScrollBuffer stores raw terminal output and replays it into a tall
// off-screen VTerm to produce a full, coloured scrollback history.
type ScrollBuffer struct {
	raw    []byte // raw terminal bytes (same data fed to the live VTerm)
	maxRaw int

	offset int // 0 = at bottom; positive = scrolled up by N lines

	// Replay cache — rebuilt lazily when raw data changes.
	cacheStyled []string // styled lines (with ANSI colours)
	cachePlain  []string // plain-text lines (for search/width)
	cacheDirty  bool
	cacheCols   int
}

// NewScrollBuffer creates a ScrollBuffer.
// maxLines is kept as the parameter name for API compat but internally
// controls the raw byte budget (maxLines * 200 bytes heuristic).
func NewScrollBuffer(maxLines int) *ScrollBuffer {
	maxRaw := maxLines * 200
	if maxRaw < 1<<20 {
		maxRaw = 1 << 20 // at least 1 MB
	}
	return &ScrollBuffer{maxRaw: maxRaw, cacheDirty: true}
}

// AppendRaw stores new terminal output bytes for later replay.
func (b *ScrollBuffer) AppendRaw(data []byte) {
	b.raw = append(b.raw, data...)
	if len(b.raw) > b.maxRaw {
		// Keep the most recent portion.  Try to land on a newline boundary
		// to minimise visual corruption after truncation.
		cut := len(b.raw) - b.maxRaw
		for cut < len(b.raw) && b.raw[cut] != '\n' {
			cut++
		}
		if cut < len(b.raw) {
			cut++ // skip the newline itself
		}
		b.raw = b.raw[cut:]
	}
	b.cacheDirty = true
}

// replay rebuilds the cache by feeding all stored raw bytes into a tall
// temporary VTerm and rendering every used row.
func (b *ScrollBuffer) replay(cols int) {
	if !b.cacheDirty && b.cacheCols == cols {
		return
	}
	if cols <= 0 {
		cols = 80
	}
	v := NewVTerm(replayHeight, cols)
	v.Write(b.raw)

	curRow, _ := v.Cursor()
	n := curRow + 1
	b.cacheStyled = make([]string, n)
	b.cachePlain = make([]string, n)
	for r := 0; r < n; r++ {
		b.cacheStyled[r] = renderRow(v, r, cols, -1, -1)
		b.cachePlain[r] = renderRowPlain(v, r, cols)
	}
	b.cacheDirty = false
	b.cacheCols = cols
}

// ---------- scroll navigation ----------

// ScrollUp scrolls toward older history by n lines.
func (b *ScrollBuffer) ScrollUp(n int) {
	b.offset = min(b.offset+n, b.totalLines())
}

// ScrollDown scrolls toward newer history by n lines.
func (b *ScrollBuffer) ScrollDown(n int) {
	b.offset = max(0, b.offset-n)
}

// ToBottom jumps to the most recent output.
func (b *ScrollBuffer) ToBottom() { b.offset = 0 }

// ToTop jumps to the oldest available line.
func (b *ScrollBuffer) ToTop() { b.offset = b.totalLines() }

// IsAtBottom returns true when no scroll offset is applied.
func (b *ScrollBuffer) IsAtBottom() bool { return b.offset == 0 }

// Offset returns the current scroll offset.
func (b *ScrollBuffer) Offset() int { return b.offset }

// Len returns the number of stored lines (requires a prior Replay call).
func (b *ScrollBuffer) Len() int { return b.totalLines() }

// CanScrollUp reports whether there is content above the visible window.
func (b *ScrollBuffer) CanScrollUp(height int) bool {
	return b.totalLines() > height && b.offset < b.totalLines()-height
}

func (b *ScrollBuffer) totalLines() int { return len(b.cacheStyled) }

// ---------- content access ----------

// Replay rebuilds the internal cache if new data has been appended.
// Must be called (with the current pane width) before StyledLines / Lines.
func (b *ScrollBuffer) Replay(cols int) { b.replay(cols) }

// StyledLines returns up to height ANSI-styled lines at the current offset.
func (b *ScrollBuffer) StyledLines(height int) []string {
	return b.sliceStyled(height)
}

// Lines returns up to height plain-text lines at the current offset.
func (b *ScrollBuffer) Lines(height int) []string {
	return b.slicePlain(height)
}

func (b *ScrollBuffer) sliceStyled(height int) []string {
	total := b.totalLines()
	if total == 0 || height <= 0 {
		return nil
	}
	start, end := b.window(height, total)
	return b.cacheStyled[start:end]
}

func (b *ScrollBuffer) slicePlain(height int) []string {
	total := b.totalLines()
	if total == 0 || height <= 0 {
		return nil
	}
	start, end := b.window(height, total)
	return b.cachePlain[start:end]
}

func (b *ScrollBuffer) window(height, total int) (start, end int) {
	end = total - b.offset
	if end <= 0 {
		end = 0
	}
	start = end - height
	if start < 0 {
		start = 0
	}
	if end > total {
		end = total
	}
	return
}

// ---------- search ----------

// Search returns the indices of lines containing substr (plain text), newest first.
func (b *ScrollBuffer) Search(substr string) []int {
	var hits []int
	for i := len(b.cachePlain) - 1; i >= 0; i-- {
		if containsSubstr(b.cachePlain[i], substr) {
			hits = append(hits, i)
		}
	}
	return hits
}

// ---------- backward compat ----------

// AddLine is retained for existing tests.
func (b *ScrollBuffer) AddLine(line string) {
	b.AppendRaw([]byte(line + "\n"))
}

// UpdateFromScreen is a no-op now; raw bytes are the source of truth.
func (b *ScrollBuffer) UpdateFromScreen(styled, plain []string) {}

// VisibleWidth returns the printable width of a styled line.
func VisibleWidth(s string) int { return lipgloss.Width(s) }

// ---------- helpers ----------

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
