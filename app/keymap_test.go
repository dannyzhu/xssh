package app

import "testing"

func TestPrefixKeyStateMachine(t *testing.T) {
	km := NewKeymap()

	// Initial state: Idle
	if km.State() != PrefixIdle {
		t.Fatal("should start Idle")
	}

	// Press Ctrl+\ → enter Waiting, no action yet
	action := km.Handle(KeyCtrlBackslash)
	if km.State() != PrefixWaiting {
		t.Fatalf("after Ctrl+\\: state = %v, want PrefixWaiting", km.State())
	}
	if action != ActionNone {
		t.Fatalf("after Ctrl+\\: action = %v, want ActionNone", action)
	}

	// Press "1" → focus pane 1, return to Idle
	action = km.Handle("1")
	if action != ActionFocusPane1 {
		t.Errorf("after '1': action = %v, want ActionFocusPane1", action)
	}
	if km.State() != PrefixIdle {
		t.Fatal("should return to Idle after second key")
	}
}

func TestPrefixKeyTimeout(t *testing.T) {
	km := NewKeymap()
	km.Handle(KeyCtrlBackslash)
	km.Timeout()
	if km.State() != PrefixIdle {
		t.Fatal("Timeout should reset to Idle")
	}
}

func TestPrefixKeyAllPanes(t *testing.T) {
	expected := []Action{
		ActionFocusPane1, ActionFocusPane2, ActionFocusPane3,
		ActionFocusPane4, ActionFocusPane5, ActionFocusPane6,
		ActionFocusPane7, ActionFocusPane8, ActionFocusPane9,
	}
	for i, want := range expected {
		km := NewKeymap()
		km.Handle(KeyCtrlBackslash)
		got := km.Handle(string(rune('1' + i)))
		if got != want {
			t.Errorf("pane %d: action = %v, want %v", i+1, got, want)
		}
	}
}

func TestPrefixKeyDirections(t *testing.T) {
	cases := []struct {
		key  string
		want Action
	}{
		{"up", ActionFocusUp}, {"k", ActionFocusUp},
		{"down", ActionFocusDown}, {"j", ActionFocusDown},
		{"left", ActionFocusLeft}, {"h", ActionFocusLeft},
		{"right", ActionFocusRight}, {"l", ActionFocusRight},
	}
	for _, c := range cases {
		km := NewKeymap()
		km.Handle(KeyCtrlBackslash)
		got := km.Handle(c.key)
		if got != c.want {
			t.Errorf("key %q: action = %v, want %v", c.key, got, c.want)
		}
	}
}

func TestPrefixKeyMisc(t *testing.T) {
	cases := []struct {
		key  string
		want Action
	}{
		{"z", ActionZoom},
		{"x", ActionClosePane},
		{"r", ActionReconnect},
		{"R", ActionReconnectAll},
		{"e", ActionAddPane},
		{"b", ActionFocusBroadcast},
		{"m", ActionBroadcastSelect},
		{"[", ActionScrollMode},
		{"?", ActionHelp},
		{"s", ActionSaveGroup},
		{KeyCtrlBackslash, ActionPassthrough},
	}
	for _, c := range cases {
		km := NewKeymap()
		km.Handle(KeyCtrlBackslash)
		got := km.Handle(c.key)
		if got != c.want {
			t.Errorf("key %q: action = %v, want %v", c.key, got, c.want)
		}
	}
}

func TestPrefixKeyUnknownSecond(t *testing.T) {
	km := NewKeymap()
	km.Handle(KeyCtrlBackslash)
	got := km.Handle("q") // not a defined second key
	if got != ActionNone {
		t.Errorf("unknown key 'q': action = %v, want ActionNone", got)
	}
	if km.State() != PrefixIdle {
		t.Error("should return to Idle on unknown second key")
	}
}

func TestPrefixKeyIdleNonPrefix(t *testing.T) {
	km := NewKeymap()
	got := km.Handle("a")
	if got != ActionNone {
		t.Errorf("idle 'a': action = %v, want ActionNone", got)
	}
	if km.State() != PrefixIdle {
		t.Error("state should stay Idle for non-prefix key")
	}
}

func TestFocusPaneAction(t *testing.T) {
	if FocusPaneAction(1) != ActionFocusPane1 {
		t.Error("FocusPaneAction(1) wrong")
	}
	if FocusPaneAction(9) != ActionFocusPane9 {
		t.Error("FocusPaneAction(9) wrong")
	}
	if FocusPaneAction(0) != ActionNone {
		t.Error("FocusPaneAction(0) should be ActionNone")
	}
	if FocusPaneAction(10) != ActionNone {
		t.Error("FocusPaneAction(10) should be ActionNone")
	}
}
