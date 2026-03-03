package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDetectPaste_UnwrapsBracketWrappedContent(t *testing.T) {
	msg := tea.KeyMsg(tea.Key{
		Type:  tea.KeyRunes,
		Runes: []rune("[W]"),
		Paste: true,
	})

	isPaste, got := detectPaste(msg, "[W]")
	if !isPaste {
		t.Fatal("expected paste to be detected")
	}
	if got != "W" {
		t.Fatalf("content = %q, want %q", got, "W")
	}
}

func TestDetectPaste_PasteFlagUsesRawRunesWhenNotWrapped(t *testing.T) {
	msg := tea.KeyMsg(tea.Key{
		Type:  tea.KeyRunes,
		Runes: []rune("hello"),
		Paste: true,
	})

	isPaste, got := detectPaste(msg, "hello")
	if !isPaste {
		t.Fatal("expected paste to be detected")
	}
	if got != "hello" {
		t.Fatalf("content = %q, want %q", got, "hello")
	}
}

func TestDetectPaste_FallbackBracketPattern(t *testing.T) {
	msg := tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune("ignored")})

	isPaste, got := detectPaste(msg, "[abc]")
	if !isPaste {
		t.Fatal("expected fallback paste detection")
	}
	if got != "abc" {
		t.Fatalf("content = %q, want %q", got, "abc")
	}
}
