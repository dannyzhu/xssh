package session

// Status represents the connection status of a session.
type Status int

const (
	StatusConnecting   Status = iota
	StatusConnected
	StatusDisconnected
	StatusReconnecting
	StatusAuthFailed
)

func (s Status) String() string {
	switch s {
	case StatusConnecting:
		return "connecting"
	case StatusConnected:
		return "connected"
	case StatusDisconnected:
		return "disconnected"
	case StatusReconnecting:
		return "reconnecting"
	case StatusAuthFailed:
		return "auth_failed"
	default:
		return "unknown"
	}
}

// Session is the interface for both local PTY and SSH sessions.
type Session interface {
	Connect() error
	Write(data []byte) error
	Output() <-chan []byte
	Resize(rows, cols int) error
	Close() error
	Status() Status
	Title() string
}
