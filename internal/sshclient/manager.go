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
	mu       sync.Mutex
	emit     EmitFunc
	sessions map[string]*connection
}

type connection struct {
	id             string
	profile        domain.Profile
	secret         string
	passphrase     string
	connectSecret  string
	client         *ssh.Client
	session        *ssh.Session
	stdin          io.WriteCloser
	waitFn         func() error
	resizeFn       func(cols int, rows int) error
	closeFn        func() error
	reconnect      bool
	manualClose    bool
	reconnectDelay time.Duration
}

func NewManager(emit EmitFunc) *Manager {
	return &Manager{emit: emit, sessions: map[string]*connection{}}
}

func (m *Manager) Connect(ctx context.Context, profile domain.Profile, secret string, passphrase string, connectSecret string) (string, error) {
	sessionID := domain.NewID()
	conn := &connection{
		id:             sessionID,
		profile:        profile,
		secret:         secret,
		passphrase:     passphrase,
		connectSecret:  connectSecret,
		reconnect:      true,
		reconnectDelay: 3 * time.Second,
	}

	m.mu.Lock()
	m.sessions[sessionID] = conn
	m.mu.Unlock()

	m.emitStatus(sessionID, "connecting", profile, fmt.Sprintf("Connecting to %s...", profile.Host), 1)
	if err := m.establish(ctx, conn, 1); err != nil {
		m.Disconnect(sessionID)
		return "", err
	}

	return sessionID, nil
}

func (m *Manager) ConnectLocalShell(ctx context.Context) (string, error) {
	profile := domain.Profile{
		ID:       domain.NewID(),
		Name:     "Local Terminal",
		Username: currentUsername(),
		Host:     "localhost",
		Port:     0,
	}

	sessionID := domain.NewID()
	conn := &connection{
		id:        sessionID,
		profile:   profile,
		reconnect: false,
	}

	m.mu.Lock()
	m.sessions[sessionID] = conn
	m.mu.Unlock()

	m.emitStatus(sessionID, "connecting", profile, "Opening local shell...", 1)
	if err := m.establishLocal(ctx, conn); err != nil {
		m.Disconnect(sessionID)
		return "", err
	}

	return sessionID, nil
}

func (m *Manager) SendInput(sessionID string, input string) error {
	m.mu.Lock()
	conn := m.sessions[sessionID]
	defer m.mu.Unlock()

	if conn == nil || conn.stdin == nil {
		return fmt.Errorf("no active ssh session")
	}

	_, err := io.WriteString(conn.stdin, input)
	return err
}

func (m *Manager) Resize(sessionID string, cols int, rows int) error {
	m.mu.Lock()
	conn := m.sessions[sessionID]
	defer m.mu.Unlock()

	if conn == nil || conn.session == nil {
		if conn == nil || conn.resizeFn == nil {
			return fmt.Errorf("no active terminal session")
		}
	}
	if cols < 20 || rows < 5 {
		return nil
	}

	return conn.resizeFn(cols, rows)
}

func (m *Manager) Disconnect(sessionID string) {
	m.mu.Lock()
	conn := m.sessions[sessionID]
	if conn != nil {
		conn.manualClose = true
		delete(m.sessions, sessionID)
	}
	m.mu.Unlock()

	if conn == nil {
		return
	}

	if conn.closeFn != nil {
		_ = conn.closeFn()
	} else {
		if conn.session != nil {
			_ = conn.session.Close()
		}
		if conn.client != nil {
			_ = conn.client.Close()
		}
	}

	m.emitStatus(sessionID, "disconnected", conn.profile, "Disconnected.", 0)
}

func (m *Manager) establish(ctx context.Context, conn *connection, attempt int) error {
	client, session, stdin, stdout, stderr, err := dial(conn.profile, conn.secret, conn.passphrase, conn.connectSecret)
	if err != nil {
		return err
	}

	conn.client = client
	conn.session = session
	conn.stdin = stdin
	conn.waitFn = session.Wait
	conn.resizeFn = func(cols int, rows int) error {
		return session.WindowChange(rows, cols)
	}
	conn.closeFn = func() error {
		if conn.session != nil {
			_ = conn.session.Close()
		}
		if conn.client != nil {
			return conn.client.Close()
		}
		return nil
	}

	m.emitStatus(conn.id, "connected", conn.profile, fmt.Sprintf("Connected to %s", conn.profile.Host), attempt)

	go m.pipeOutput(conn.id, stdout)
	go m.pipeOutput(conn.id, stderr)
	go m.watchSession(ctx, conn)

	return nil
}

func (m *Manager) establishLocal(ctx context.Context, conn *connection) error {
	shell, master, waitFn, resizeFn, closeFn, err := startLocalShell()
	if err != nil {
		return err
	}

	conn.stdin = master
	conn.waitFn = waitFn
	conn.resizeFn = resizeFn
	conn.closeFn = closeFn

	m.emitStatus(conn.id, "connected", conn.profile, fmt.Sprintf("Local shell ready: %s", shell), 1)

	go m.pipeOutput(conn.id, master)
	go m.watchSession(ctx, conn)

	return nil
}

func (m *Manager) pipeOutput(sessionID string, reader io.Reader) {
	buffer := make([]byte, 4096)
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			m.emit("ssh:output", map[string]interface{}{
				"sessionId": sessionID,
				"chunk":     string(buffer[:n]),
			})
		}
		if err != nil {
			return
		}
	}
}

func (m *Manager) watchSession(ctx context.Context, conn *connection) {
	if conn.waitFn == nil {
		return
	}
	err := conn.waitFn()

	m.mu.Lock()
	currentConn := m.sessions[conn.id]
	stillCurrent := currentConn == conn
	manualClose := conn.manualClose
	m.mu.Unlock()

	if !stillCurrent {
		return
	}

	if manualClose {
		return
	}

	if !conn.reconnect {
		m.mu.Lock()
		if m.sessions[conn.id] == conn {
			delete(m.sessions, conn.id)
		}
		m.mu.Unlock()
		if err != nil {
			m.emitStatus(conn.id, "disconnected", conn.profile, "Terminal session ended.", 0)
		}
		return
	}

	if err != nil {
		m.emitStatus(conn.id, "reconnecting", conn.profile, fmt.Sprintf("Connection dropped. Reconnecting to %s...", conn.profile.Host), 0)
	}

	if conn.closeFn != nil {
		_ = conn.closeFn()
	} else {
		_ = conn.client.Close()
	}

	for attempt := 1; ; attempt++ {
		time.Sleep(conn.reconnectDelay)

		m.mu.Lock()
		if m.sessions[conn.id] != conn || conn.manualClose {
			m.mu.Unlock()
			return
		}
		m.mu.Unlock()

		m.emitStatus(conn.id, "connecting", conn.profile, fmt.Sprintf("Reconnecting to %s (attempt %d)...", conn.profile.Host, attempt), attempt)
		if err := m.establish(ctx, conn, attempt); err == nil {
			m.emit("ssh:output", map[string]interface{}{
				"sessionId": conn.id,
				"chunk":     "\n[MySSH] Reconnected successfully.\n",
			})
			return
		}

		m.emitStatus(conn.id, "reconnecting", conn.profile, fmt.Sprintf("Reconnect failed. Retrying %s...", conn.profile.Host), attempt)
	}
}

func currentUsername() string {
	if username := os.Getenv("USER"); username != "" {
		return username
	}
	return "local"
}

func (m *Manager) emitStatus(sessionID string, state string, profile domain.Profile, message string, attempt int) {
	m.emit("ssh:status", map[string]interface{}{
		"sessionId": sessionID,
		"state":     state,
		"message":   message,
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

func dial(profile domain.Profile, secret string, passphrase string, connectSecret string) (*ssh.Client, *ssh.Session, io.WriteCloser, io.Reader, io.Reader, error) {
	client, err := DialClient(profile, secret, passphrase)
	if err != nil {
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

func DialClient(profile domain.Profile, secret string, passphrase string) (*ssh.Client, error) {
	hostKeyCallback, err := knownHostsCallback()
	if err != nil {
		return nil, err
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
	case domain.AuthPrivateKey:
		auth, err := privateKeyAuthMethod(profile, secret, passphrase)
		if err != nil {
			return nil, err
		}
		config.Auth = []ssh.AuthMethod{auth}
	case domain.AuthAgentFallbackKey:
		authMethods, signerCount, usedFallback, err := agentFallbackAuthMethods(profile, secret, passphrase)
		if err != nil {
			return nil, err
		}
		agentSignerCount = signerCount
		config.Auth = authMethods
		_ = usedFallback
	case domain.AuthAgent:
		auth, signerCount, err := agentAuthMethod()
		if err != nil {
			return nil, err
		}
		agentSignerCount = signerCount
		config.Auth = []ssh.AuthMethod{auth}
	default:
		return nil, fmt.Errorf("connect currently supports password, agent, and private_key auth")
	}

	address := fmt.Sprintf("%s:%d", profile.Host, profile.Port)
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		if profile.AuthKind == domain.AuthAgent {
			return nil, fmt.Errorf("ssh-agent auth failed (%d keys loaded). Make sure the correct key is loaded with ssh-add and that the server accepts it: %w", agentSignerCount, err)
		}
		if profile.AuthKind == domain.AuthAgentFallbackKey {
			return nil, fmt.Errorf("agent + fallback key auth failed (%d agent keys tried): %w", agentSignerCount, err)
		}
		if profile.AuthKind == domain.AuthPrivateKey {
			return nil, fmt.Errorf("private key auth failed: %w", err)
		}
		return nil, err
	}

	return client, nil
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

func privateKeyAuthMethod(profile domain.Profile, secret string, passphrase string) (ssh.AuthMethod, error) {
	var keyData []byte

	switch profile.KeySource {
	case domain.KeySourcePath:
		if profile.KeyPath == "" {
			return nil, fmt.Errorf("private key path is empty")
		}
		data, err := os.ReadFile(profile.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("read private key file: %w", err)
		}
		keyData = data
	case domain.KeySourceContent:
		if secret == "" {
			return nil, fmt.Errorf("private key content is missing from keyring")
		}
		keyData = []byte(secret)
	default:
		return nil, fmt.Errorf("private key auth requires key source path or content")
	}

	var (
		signer ssh.Signer
		err    error
	)
	if passphrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(keyData, []byte(passphrase))
	} else {
		signer, err = ssh.ParsePrivateKey(keyData)
	}
	if err != nil {
		if _, ok := err.(*ssh.PassphraseMissingError); ok {
			return nil, fmt.Errorf("private key requires a passphrase")
		}
		if passphrase != "" {
			return nil, fmt.Errorf("parse private key with passphrase: %w", err)
		}
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	return ssh.PublicKeys(signer), nil
}

func agentFallbackAuthMethods(profile domain.Profile, secret string, passphrase string) ([]ssh.AuthMethod, int, bool, error) {
	authMethods := make([]ssh.AuthMethod, 0, 2)
	usedFallback := false
	signerCount := 0

	if agentAuth, count, err := agentAuthMethod(); err == nil {
		authMethods = append(authMethods, agentAuth)
		signerCount = count
	}

	keyAuth, err := privateKeyAuthMethod(profile, secret, passphrase)
	if err != nil {
		if len(authMethods) > 0 {
			return authMethods, signerCount, usedFallback, nil
		}
		return nil, signerCount, usedFallback, err
	}
	authMethods = append(authMethods, keyAuth)
	usedFallback = true

	return authMethods, signerCount, usedFallback, nil
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
