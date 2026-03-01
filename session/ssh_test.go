package session

import (
	"testing"

	"github.com/xssh/xssh/config"
)

func TestSSHSessionTitle(t *testing.T) {
	entry := &config.HostEntry{Alias: "web1", HostName: "192.168.1.1", Port: "22"}
	s := NewSSH(entry, nil)
	if s.Title() != "web1" {
		t.Errorf("Title = %q, want %q", s.Title(), "web1")
	}
}

func TestSSHSessionTitleFallback(t *testing.T) {
	entry := &config.HostEntry{Alias: "", HostName: "myhost.example.com", Port: "22"}
	s := NewSSH(entry, nil)
	if s.Title() != "myhost.example.com" {
		t.Errorf("Title fallback = %q, want %q", s.Title(), "myhost.example.com")
	}
}

func TestSSHSessionInitialStatus(t *testing.T) {
	entry := &config.HostEntry{Alias: "web1", HostName: "localhost", Port: "22"}
	s := NewSSH(entry, nil)
	if s.Status() != StatusConnecting {
		t.Errorf("initial Status = %v, want StatusConnecting", s.Status())
	}
}

func TestSSHSessionOutput(t *testing.T) {
	entry := &config.HostEntry{Alias: "web1", HostName: "localhost", Port: "22"}
	s := NewSSH(entry, nil)
	ch := s.Output()
	if ch == nil {
		t.Error("Output() returned nil channel")
	}
}

func TestSSHSessionConnectRefused(t *testing.T) {
	// Connect to a port guaranteed to refuse
	entry := &config.HostEntry{
		Alias:    "nohost",
		HostName: "127.0.0.1",
		Port:     "1", // port 1 should always refuse
		User:     "nobody",
	}
	s := NewSSH(entry, nil)
	err := s.Connect()
	if err == nil {
		t.Error("expected error connecting to refused port")
		s.Close()
	}
	if s.Status() == StatusConnected {
		t.Error("status should not be Connected after failed connect")
	}
}
