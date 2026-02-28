package session

import "testing"

func TestStatusString(t *testing.T) {
	cases := []struct {
		s    Status
		want string
	}{
		{StatusConnecting, "connecting"},
		{StatusConnected, "connected"},
		{StatusDisconnected, "disconnected"},
		{StatusReconnecting, "reconnecting"},
		{StatusAuthFailed, "auth_failed"},
	}
	for _, c := range cases {
		if got := c.s.String(); got != c.want {
			t.Errorf("Status(%d).String() = %q, want %q", c.s, got, c.want)
		}
	}
}
