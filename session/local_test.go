package session

import (
	"bytes"
	"testing"
	"time"
)

func TestLocalSessionEcho(t *testing.T) {
	s := NewLocal()
	if err := s.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer s.Close()

	if s.Status() != StatusConnected {
		t.Fatalf("expected StatusConnected, got %v", s.Status())
	}

	if err := s.Write([]byte("echo hello\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}

	timeout := time.After(2 * time.Second)
	var got []byte
	for {
		select {
		case data, ok := <-s.Output():
			if !ok {
				t.Fatalf("output channel closed, got: %q", got)
			}
			got = append(got, data...)
			if bytes.Contains(got, []byte("hello")) {
				return
			}
		case <-timeout:
			t.Fatalf("timeout waiting for echo output, got: %q", got)
		}
	}
}

func TestLocalSessionResize(t *testing.T) {
	s := NewLocal()
	if err := s.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer s.Close()

	if err := s.Resize(30, 120); err != nil {
		t.Errorf("Resize: %v", err)
	}
}

func TestLocalSessionTitle(t *testing.T) {
	s := NewLocal()
	title := s.Title()
	if title == "" {
		t.Error("Title should not be empty")
	}
}
