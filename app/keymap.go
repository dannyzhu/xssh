package app

// Action is the semantic command produced by the keymap after interpreting a key.
type Action int

const (
	ActionNone Action = iota

	// Pane focus
	ActionFocusPane1
	ActionFocusPane2
	ActionFocusPane3
	ActionFocusPane4
	ActionFocusPane5
	ActionFocusPane6
	ActionFocusPane7
	ActionFocusPane8
	ActionFocusPane9
	ActionFocusUp
	ActionFocusDown
	ActionFocusLeft
	ActionFocusRight

	// Pane lifecycle
	ActionZoom
	ActionClosePane
	ActionReconnect
	ActionReconnectAll
	ActionAddPane

	// Broadcast
	ActionFocusBroadcast
	ActionBroadcastSelect

	// Scroll / search
	ActionScrollMode

	// Misc
	ActionHelp
	ActionSaveGroup
	ActionRepaint     // force full screen redraw
	ActionPassthrough // re-emit Ctrl+\ itself to the session
	ActionDelete      // Ctrl+\+/ → same as scroll-mode search (see plan)
)

// Key constants used by the keymap
const (
	KeyCtrlBackslash = "ctrl+\\"
)

// Keymap implements the Ctrl+\ prefix key state machine.
type Keymap struct {
	state PrefixState
}

// NewKeymap returns a Keymap starting in the Idle state.
func NewKeymap() *Keymap { return &Keymap{state: PrefixIdle} }

// State returns the current prefix state.
func (k *Keymap) State() PrefixState { return k.state }

// Timeout resets the state machine to Idle (e.g., after a 1-second timeout).
func (k *Keymap) Timeout() { k.state = PrefixIdle }

// Handle processes a key string and returns the resulting Action.
//
// In PrefixIdle:
//   - "ctrl+\\" → enter PrefixWaiting, return ActionNone
//   - anything else → ActionNone (caller handles normal forwarding)
//
// In PrefixWaiting:
//   - second key determines Action; always resets to PrefixIdle
func (k *Keymap) Handle(key string) Action {
	switch k.state {
	case PrefixIdle:
		if key == KeyCtrlBackslash {
			k.state = PrefixWaiting
			return ActionNone
		}
		return ActionNone

	case PrefixWaiting:
		k.state = PrefixIdle
		return k.resolveSecond(key)
	}
	return ActionNone
}

// resolveSecond maps the second key (after Ctrl+\) to an Action.
func (k *Keymap) resolveSecond(key string) Action {
	switch key {
	case "1":
		return ActionFocusPane1
	case "2":
		return ActionFocusPane2
	case "3":
		return ActionFocusPane3
	case "4":
		return ActionFocusPane4
	case "5":
		return ActionFocusPane5
	case "6":
		return ActionFocusPane6
	case "7":
		return ActionFocusPane7
	case "8":
		return ActionFocusPane8
	case "9":
		return ActionFocusPane9

	case "up", "k":
		return ActionFocusUp
	case "down", "j":
		return ActionFocusDown
	case "left", "h":
		return ActionFocusLeft
	case "right", "l":
		return ActionFocusRight

	case "z":
		return ActionZoom
	case "x":
		return ActionClosePane
	case "r":
		return ActionReconnect
	case "R":
		return ActionReconnectAll
	case "e":
		return ActionAddPane

	case "b":
		return ActionFocusBroadcast
	case "m":
		return ActionBroadcastSelect

	case "[":
		return ActionScrollMode

	case "?":
		return ActionHelp
	case "s":
		return ActionSaveGroup

	case "/":
		return ActionDelete // Ctrl+\+/ deletes/searches (plan: same as scroll /); handled in context

	case "ctrl+l":
		return ActionRepaint

	// Sending Ctrl+\ to the session itself
	case KeyCtrlBackslash:
		return ActionPassthrough
	}
	return ActionNone
}

// FocusPaneAction returns the ActionFocusPaneN for a 1-based pane number,
// or ActionNone if out of range.
func FocusPaneAction(n int) Action {
	if n < 1 || n > 9 {
		return ActionNone
	}
	return Action(int(ActionFocusPane1) + n - 1)
}
