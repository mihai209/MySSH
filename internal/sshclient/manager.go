package sshclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	sshagent "golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"

	"myssh/internal/domain"
)

type EmitFunc func(string, interface{})

type UnknownHostError struct {
	Host         string
	HostWithPort string
	Fingerprint  string
	Key          ssh.PublicKey
}

func (e *UnknownHostError) Error() string {
	return fmt.Sprintf("ssh: host %s is unknown (%s)", e.HostWithPort, e.Fingerprint)
}

type Manager struct {
	mu      sync.Mutex
	emit    EmitFunc
	current *connection
}

type connection struct {
	profile        domain.Profile
	secret         string
	connectSecret  string
	client         *ssh.Client
	session        *ssh.Session
	stdin          io.WriteCloser
	manualClose    bool
	reconnectDelay time.Duration
}

func NewManager(emit EmitFunc) *Manager {
	return &Manager{emit: emit}
}

func (m *Manager) Connect(ctx context.Context, profile domain.Profile, secret string, connectSecret string) error {
	m.Disconnect()

	conn := &connection{
		profile:        profile,
		secret:         secret,
		connectSecret:  connectSecret,
		reconnectDelay: 3 * time.Second,
	}

	m.mu.Lock()
	m.current = conn
	m.mu.Unlock()

	m.emitStatus("connecting", profile, fmt.Sprintf("Connecting to %s...", profile.Host), 1)
	if err := m.establish(ctx, conn, 1); err != nil {
		m.mu.Lock()
		if m.current == conn {
			m.current = nil
		}
		m.mu.Unlock()
		return err
	}

	return nil
}

func (m *Manager) SendInput(input string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.current == nil || m.current.stdin == nil {
		return fmt.Errorf("no active ssh session")
	}

	_, err := io.WriteString(m.current.stdin, input)
	return err
}

func (m *Manager) Resize(cols int, rows int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.current == nil || m.current.session == nil {
		return fmt.Errorf("no active ssh session")
	}
	if cols < 20 || rows < 5 {
		return nil
	}

	return m.current.session.WindowChange(rows, cols)
}

func (m *Manager) Disconnect() {
	m.mu.Lock()
	conn := m.current
	if conn != nil {
		conn.manualClose = true
		m.current = nil
	}
	m.mu.Unlock()

	if conn == nil {
		return
	}

	if conn.session != nil {
		_ = conn.session.Close()
	}
	if conn.client != nil {
		_ = conn.client.Close()
	}

	m.emitStatus("disconnected", conn.profile, "Disconnected.", 0)
}

func (m *Manager) establish(ctx context.Context, conn *connection, attempt int) error {
	client, session, stdin, stdout, stderr, err := dial(conn.profile, conn.secret, conn.connectSecret)
	if err != nil {
		return err
	}

	conn.client = client
	conn.session = session
	conn.stdin = stdin

	m.emitStatus("connected", conn.profile, fmt.Sprintf("Connected to %s", conn.profile.Host), attempt)

	go m.pipeOutput(stdout)
	go m.pipeOutput(stderr)
	go m.watchSession(ctx, conn)

	return nil
}

func (m *Manager) pipeOutput(reader io.Reader) {
	buffer := make([]byte, 4096)
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			m.emit("ssh:output", map[string]interface{}{
				"chunk": string(buffer[:n]),
			})
		}
		if err != nil {
			return
		}
	}
}

func (m *Manager) watchSession(ctx context.Context, conn *connection) {
	err := conn.session.Wait()

	m.mu.Lock()
	stillCurrent := m.current == conn
	manualClose := conn.manualClose
	m.mu.Unlock()

	if !stillCurrent {
		return
	}

	if manualClose {
		return
	}

	if err != nil {
		m.emitStatus("reconnecting", conn.profile, fmt.Sprintf("Connection dropped. Reconnecting to %s...", conn.profile.Host), 0)
	}

	_ = conn.client.Close()

	for attempt := 1; ; attempt++ {
		time.Sleep(conn.reconnectDelay)

		m.mu.Lock()
		if m.current != conn || conn.manualClose {
			m.mu.Unlock()
			return
		}
		m.mu.Unlock()

		m.emitStatus("connecting", conn.profile, fmt.Sprintf("Reconnecting to %s (attempt %d)...", conn.profile.Host, attempt), attempt)
		if err := m.establish(ctx, conn, attempt); err == nil {
			m.emit("ssh:output", map[string]interface{}{
				"chunk": "\n[MySSH] Reconnected successfully.\n",
			})
			return
		}

		m.emitStatus("reconnecting", conn.profile, fmt.Sprintf("Reconnect failed. Retrying %s...", conn.profile.Host), attempt)
	}
}

func (m *Manager) emitStatus(state string, profile domain.Profile, message string, attempt int) {
	m.emit("ssh:status", map[string]interface{}{
		"state":   state,
		"message": message,
		"profile": map[string]interface{}{
			"id":       profile.ID,
			"name":     profile.Name,
			"host":     profile.Host,
			"username": profile.Username,
			"port":     profile.Port,
		},
		"attempt": attempt,
	})
}

func dial(profile domain.Profile, secret string, connectSecret string) (*ssh.Client, *ssh.Session, io.WriteCloser, io.Reader, io.Reader, error) {
	hostKeyCallback, err := knownHostsCallback()
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	config := &ssh.ClientConfig{
		User:            profile.Username,
		HostKeyCallback: hostKeyCallback,
		Timeout:         15 * time.Second,
	}

	agentSignerCount := 0
	switch profile.AuthKind {
	case domain.AuthPassword:
		config.Auth = []ssh.AuthMethod{ssh.Password(secret)}
	case domain.AuthAgent:
		auth, signerCount, err := agentAuthMethod()
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
		agentSignerCount = signerCount
		config.Auth = []ssh.AuthMethod{auth}
	default:
		return nil, nil, nil, nil, nil, fmt.Errorf("connect currently supports only password and agent auth")
	}

	address := fmt.Sprintf("%s:%d", profile.Host, profile.Port)
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		if profile.AuthKind == domain.AuthAgent {
			return nil, nil, nil, nil, nil, fmt.Errorf("ssh-agent auth failed (%d keys loaded). Make sure the correct key is loaded with ssh-add and that the server accepts it: %w", agentSignerCount, err)
		}
		return nil, nil, nil, nil, nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		_ = client.Close()
		return nil, nil, nil, nil, nil, err
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		_ = session.Close()
		_ = client.Close()
		return nil, nil, nil, nil, nil, err
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		_ = session.Close()
		_ = client.Close()
		return nil, nil, nil, nil, nil, err
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		_ = session.Close()
		_ = client.Close()
		return nil, nil, nil, nil, nil, err
	}

	if err := session.RequestPty("xterm-256color", 40, 120, ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}); err != nil {
		_ = session.Close()
		_ = client.Close()
		return nil, nil, nil, nil, nil, err
	}

	if connectSecret != "" {
		if err := session.Setenv("SECRET", connectSecret); err != nil {
			_ = session.Close()
			_ = client.Close()
			return nil, nil, nil, nil, nil, fmt.Errorf("set remote SECRET env: %w", err)
		}
	}

	if err := session.Shell(); err != nil {
		_ = session.Close()
		_ = client.Close()
		return nil, nil, nil, nil, nil, err
	}

	return client, session, stdin, stdout, stderr, nil
}

func agentAuthMethod() (ssh.AuthMethod, int, error) {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil, 0, fmt.Errorf("SSH_AUTH_SOCK is not set; no ssh-agent available")
	}

	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil, 0, fmt.Errorf("connect to ssh-agent: %w", err)
	}

	agentClient := sshagent.NewClient(conn)
	signers, err := agentClient.Signers()
	if err != nil {
		return nil, 0, fmt.Errorf("read keys from ssh-agent: %w", err)
	}
	if len(signers) == 0 {
		return nil, 0, fmt.Errorf("ssh-agent has no loaded keys; run ssh-add first")
	}

	return ssh.PublicKeys(signers...), len(signers), nil
}

func knownHostsCallback() (ssh.HostKeyCallback, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home dir for known_hosts: %w", err)
	}

	candidates := []string{
		filepath.Join(homeDir, ".ssh", "known_hosts"),
		"/etc/ssh/ssh_known_hosts",
	}

	files := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			files = append(files, candidate)
		}
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no known_hosts file found; add the host to ~/.ssh/known_hosts first")
	}

	callback, err := knownhosts.New(files...)
	if err != nil {
		return nil, fmt.Errorf("load known_hosts: %w", err)
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := callback(hostname, remote, key)
		if err == nil {
			return nil
		}

		var keyErr *knownhosts.KeyError
		if errors.As(err, &keyErr) && len(keyErr.Want) == 0 {
			return &UnknownHostError{
				Host:         remote.String(),
				HostWithPort: hostname,
				Fingerprint:  ssh.FingerprintSHA256(key),
				Key:          key,
			}
		}

		return err
	}, nil
}

func KnownHostsPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, ".ssh", "known_hosts"), nil
}

func AddKnownHost(host string, key ssh.PublicKey) error {
	path, err := KnownHostsPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create .ssh dir: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open known_hosts: %w", err)
	}
	defer file.Close()

	line := knownhosts.Line([]string{host}, key)
	if _, err := io.WriteString(file, line+"\n"); err != nil {
		return fmt.Errorf("append known_hosts: %w", err)
	}
	return nil
}
