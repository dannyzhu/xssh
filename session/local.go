package session

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/creack/pty"
)

// LocalSession runs a command (local shell or ssh) via a PTY.
type LocalSession struct {
	args   []string // command + arguments to execute
	title  string   // display name shown in the status bar
	ptmx   *os.File
	cmd    *exec.Cmd
	status Status
	outCh  chan []byte
	mu     sync.Mutex
}

// NewLocal creates a LocalSession that opens $SHELL.
func NewLocal() *LocalSession {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	return &LocalSession{
		args:  []string{shell},
		title: filepath.Base(shell),
		outCh: make(chan []byte, 256),
	}
}

// NewLocalCmd creates a LocalSession that runs an arbitrary command.
// title is the label shown in the status bar / pane header.
func NewLocalCmd(args []string, title string) *LocalSession {
	return &LocalSession{
		args:  args,
		title: title,
		outCh: make(chan []byte, 256),
	}
}

func (s *LocalSession) Connect() error {
	// Always create a fresh channel: the previous one may have been closed by
	// readLoop after a disconnect, and sending to a closed channel panics.
	s.mu.Lock()
	s.outCh = make(chan []byte, 256)
	s.mu.Unlock()

	cmd := exec.Command(s.args[0], s.args[1:]...)
	cmd.Env = append(os.Environ(), "TERM="+EmulatedTerm, "COLORTERM=truecolor")
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
	return s.title
}
