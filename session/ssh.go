package session

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/xssh/xssh/config"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

// SSHSession manages an SSH connection to a remote host.
type SSHSession struct {
	entry  *config.HostEntry
	title  string

	client  *ssh.Client
	session *ssh.Session
	stdin   net.Conn // pipe to session stdin
	ptmx    net.Conn // pipe from session stdout/stderr

	status Status
	outCh  chan []byte
	mu     sync.Mutex

	// PasswordReply receives passwords typed in the TUI password overlay.
	PasswordReply <-chan string
}

// NewSSH creates an SSHSession for the given host entry.
// pwReply is the channel that delivers passwords from the TUI; may be nil if no interactive auth.
func NewSSH(entry *config.HostEntry, pwReply <-chan string) *SSHSession {
	title := entry.Alias
	if title == "" {
		title = entry.HostName
	}
	return &SSHSession{
		entry:         entry,
		title:         title,
		outCh:         make(chan []byte, 256),
		PasswordReply: pwReply,
		status:        StatusConnecting,
	}
}

func (s *SSHSession) Connect() error {
	cfg, err := s.buildClientConfig()
	if err != nil {
		s.setStatus(StatusDisconnected)
		return fmt.Errorf("build config: %w", err)
	}

	addr := net.JoinHostPort(s.entry.HostName, s.entry.Port)

	var conn net.Conn
	if s.entry.ProxyJump != "" {
		conn, err = s.dialViaProxy(s.entry.ProxyJump, addr, cfg)
	} else {
		conn, err = net.DialTimeout("tcp", addr, 15*time.Second)
	}
	if err != nil {
		s.setStatus(StatusDisconnected)
		return fmt.Errorf("dial %s: %w", addr, err)
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, cfg)
	if err != nil {
		conn.Close()
		if isAuthErr(err) {
			s.setStatus(StatusAuthFailed)
		} else {
			s.setStatus(StatusDisconnected)
		}
		return err
	}
	client := ssh.NewClient(sshConn, chans, reqs)

	sess, err := client.NewSession()
	if err != nil {
		client.Close()
		s.setStatus(StatusDisconnected)
		return fmt.Errorf("new session: %w", err)
	}

	// Request PTY
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := sess.RequestPty("xterm-256color", 24, 80, modes); err != nil {
		sess.Close()
		client.Close()
		s.setStatus(StatusDisconnected)
		return fmt.Errorf("request pty: %w", err)
	}

	// Wire up stdin pipe
	stdinPipe, err := sess.StdinPipe()
	if err != nil {
		sess.Close()
		client.Close()
		s.setStatus(StatusDisconnected)
		return fmt.Errorf("stdin pipe: %w", err)
	}

	// Wire up stdout
	sess.Stdout = &chanWriter{ch: s.outCh}
	sess.Stderr = &chanWriter{ch: s.outCh}

	if err := sess.Shell(); err != nil {
		sess.Close()
		client.Close()
		s.setStatus(StatusDisconnected)
		return fmt.Errorf("shell: %w", err)
	}

	s.mu.Lock()
	s.client = client
	s.session = sess
	s.status = StatusConnected
	s.mu.Unlock()

	// Store stdin pipe as a pipeConn wrapper
	go func() {
		sess.Wait()
		s.setStatus(StatusDisconnected)
		close(s.outCh)
		stdinPipe.Close()
	}()

	// Keep stdinPipe accessible via a wrapper
	s.mu.Lock()
	s.ptmx = &pipeConn{writer: stdinPipe}
	s.mu.Unlock()

	return nil
}

func (s *SSHSession) Write(data []byte) error {
	s.mu.Lock()
	ptmx := s.ptmx
	s.mu.Unlock()
	if ptmx == nil {
		return nil
	}
	_, err := ptmx.Write(data)
	return err
}

func (s *SSHSession) Output() <-chan []byte { return s.outCh }

func (s *SSHSession) Resize(rows, cols int) error {
	s.mu.Lock()
	sess := s.session
	s.mu.Unlock()
	if sess == nil {
		return nil
	}
	return sess.WindowChange(rows, cols)
}

func (s *SSHSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = StatusDisconnected
	if s.session != nil {
		s.session.Close()
		s.session = nil
	}
	if s.client != nil {
		s.client.Close()
		s.client = nil
	}
	return nil
}

func (s *SSHSession) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *SSHSession) Title() string { return s.title }

// buildClientConfig constructs an ssh.ClientConfig with an ordered auth chain.
func (s *SSHSession) buildClientConfig() (*ssh.ClientConfig, error) {
	user := s.entry.User
	if user == "" {
		u, _ := os.UserHomeDir()
		_ = u // use os.Getenv as fallback
		user = os.Getenv("USER")
	}

	var authMethods []ssh.AuthMethod

	// 1. ssh-agent
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		if conn, err := net.Dial("unix", sock); err == nil {
			ag := agent.NewClient(conn)
			authMethods = append(authMethods, ssh.PublicKeysCallback(ag.Signers))
		}
	}

	// 2. IdentityFile from ssh config
	if s.entry.IdentityFile != "" {
		if signer, err := loadKey(s.entry.IdentityFile); err == nil {
			authMethods = append(authMethods, ssh.PublicKeys(signer))
		}
	}

	// 3. Default key files
	home, _ := os.UserHomeDir()
	defaultKeys := []string{
		filepath.Join(home, ".ssh", "id_ed25519"),
		filepath.Join(home, ".ssh", "id_ecdsa"),
		filepath.Join(home, ".ssh", "id_rsa"),
	}
	for _, kf := range defaultKeys {
		if signer, err := loadKey(kf); err == nil {
			authMethods = append(authMethods, ssh.PublicKeys(signer))
		}
	}

	// 4. Password (interactive via TUI channel)
	if s.PasswordReply != nil {
		authMethods = append(authMethods, ssh.PasswordCallback(func() (string, error) {
			pw, ok := <-s.PasswordReply
			if !ok {
				return "", fmt.Errorf("password channel closed")
			}
			return pw, nil
		}))
		authMethods = append(authMethods, ssh.KeyboardInteractive(func(name, instruction string, questions []string, echos []bool) ([]string, error) {
			answers := make([]string, len(questions))
			for i := range questions {
				pw, ok := <-s.PasswordReply
				if !ok {
					return nil, fmt.Errorf("password channel closed")
				}
				answers[i] = pw
			}
			return answers, nil
		}))
	}

	// Known hosts
	knownHostsFile := filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts")
	hostKeyCallback := ssh.InsecureIgnoreHostKey()
	if _, err := os.Stat(knownHostsFile); err == nil {
		if cb, err := knownhosts.New(knownHostsFile); err == nil {
			hostKeyCallback = cb
		}
	}

	return &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         15 * time.Second,
	}, nil
}

// dialViaProxy dials target through a ProxyJump host.
func (s *SSHSession) dialViaProxy(proxyHost, targetAddr string, targetCfg *ssh.ClientConfig) (net.Conn, error) {
	// Build a minimal config for the proxy using the same auth
	proxyCfg := &ssh.ClientConfig{
		User:            targetCfg.User,
		Auth:            targetCfg.Auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}

	proxyEntry, err := config.Resolve(proxyHost)
	if err != nil {
		return nil, fmt.Errorf("resolve proxy %s: %w", proxyHost, err)
	}
	proxyAddr := net.JoinHostPort(proxyEntry.HostName, proxyEntry.Port)

	proxyClient, err := ssh.Dial("tcp", proxyAddr, proxyCfg)
	if err != nil {
		return nil, fmt.Errorf("proxy dial %s: %w", proxyAddr, err)
	}
	return proxyClient.Dial("tcp", targetAddr)
}

func (s *SSHSession) setStatus(st Status) {
	s.mu.Lock()
	s.status = st
	s.mu.Unlock()
}

func isAuthErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return contains(msg, "unable to authenticate") || contains(msg, "no supported methods remain")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func loadKey(path string) (ssh.Signer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(data)
}

// chanWriter adapts an io.Writer to write into a []byte channel.
type chanWriter struct {
	ch chan<- []byte
}

func (w *chanWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	cp := make([]byte, len(p))
	copy(cp, p)
	select {
	case w.ch <- cp:
	default:
		// drop if full to avoid blocking
	}
	return len(p), nil
}

// pipeConn wraps an io.WriteCloser as a net.Conn (write-only, stubs for reads).
type pipeConn struct {
	writer interface{ Write([]byte) (int, error) }
}

func (p *pipeConn) Write(b []byte) (int, error)        { return p.writer.Write(b) }
func (p *pipeConn) Read(b []byte) (int, error)         { return 0, nil }
func (p *pipeConn) Close() error                       { return nil }
func (p *pipeConn) LocalAddr() net.Addr                { return nil }
func (p *pipeConn) RemoteAddr() net.Addr               { return nil }
func (p *pipeConn) SetDeadline(t time.Time) error      { return nil }
func (p *pipeConn) SetReadDeadline(t time.Time) error  { return nil }
func (p *pipeConn) SetWriteDeadline(t time.Time) error { return nil }
