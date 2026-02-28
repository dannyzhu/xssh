package session

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/creack/pty"
)

// LocalSession runs a local shell via PTY.
type LocalSession struct {
	shell  string
	ptmx   *os.File
	cmd    *exec.Cmd
	status Status
	outCh  chan []byte
	mu     sync.Mutex
}

// NewLocal creates a new LocalSession using the current $SHELL.
func NewLocal() *LocalSession {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	return &LocalSession{shell: shell, outCh: make(chan []byte, 256)}
}

func (s *LocalSession) Connect() error {
	cmd := exec.Command(s.shell)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		s.mu.Lock()
		s.status = StatusDisconnected
		s.mu.Unlock()
		return err
	}
	s.mu.Lock()
	s.ptmx = ptmx
	s.cmd = cmd
	s.status = StatusConnected
	s.mu.Unlock()
	go s.readLoop()
	return nil
}

func (s *LocalSession) readLoop() {
	buf := make([]byte, 4096)
	for {
		n, err := s.ptmx.Read(buf)
		if n > 0 {
			cp := make([]byte, n)
			copy(cp, buf[:n])
			s.outCh <- cp
		}
		if err != nil {
			s.mu.Lock()
			s.status = StatusDisconnected
			s.mu.Unlock()
			close(s.outCh)
			return
		}
	}
}

func (s *LocalSession) Write(data []byte) error {
	s.mu.Lock()
	ptmx := s.ptmx
	s.mu.Unlock()
	if ptmx == nil {
		return nil
	}
	_, err := ptmx.Write(data)
	return err
}

func (s *LocalSession) Output() <-chan []byte { return s.outCh }

func (s *LocalSession) Resize(rows, cols int) error {
	s.mu.Lock()
	ptmx := s.ptmx
	s.mu.Unlock()
	if ptmx == nil {
		return nil
	}
	return pty.Setsize(ptmx, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
}

func (s *LocalSession) Close() error {
	s.mu.Lock()
	s.status = StatusDisconnected
	ptmx := s.ptmx
	s.mu.Unlock()
	if ptmx != nil {
		return ptmx.Close()
	}
	return nil
}

func (s *LocalSession) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *LocalSession) Title() string {
	return filepath.Base(s.shell)
}
