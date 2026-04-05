package sftpclient

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/pkg/sftp"

	"myssh/internal/domain"
	"myssh/internal/sshclient"
)

type FileEntry struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	IsDir    bool   `json:"isDir"`
	Size     int64  `json:"size"`
	Mode     string `json:"mode"`
	Modified string `json:"modified"`
}

type Directory struct {
	SessionID string      `json:"sessionId"`
	Path      string      `json:"path"`
	Entries   []FileEntry `json:"entries"`
}

type session struct {
	id     string
	client *sftp.Client
	ssh    io.Closer
}

type Manager struct {
	mu       sync.Mutex
	sessions map[string]*session
}

func NewManager() *Manager {
	return &Manager{sessions: map[string]*session{}}
}

func (m *Manager) Open(profile domain.Profile, secret string, passphrase string) (Directory, error) {
	sshConn, err := sshclient.DialClient(profile, secret, passphrase)
	if err != nil {
		return Directory{}, err
	}

	client, err := sftp.NewClient(sshConn)
	if err != nil {
		_ = sshConn.Close()
		return Directory{}, fmt.Errorf("open sftp subsystem: %w", err)
	}

	sessionID := domain.NewID()
	s := &session{id: sessionID, client: client, ssh: sshConn}

	m.mu.Lock()
	m.sessions[sessionID] = s
	m.mu.Unlock()

	path, err := client.RealPath(".")
	if err != nil || path == "" {
		path = "."
	}

	dir, err := m.List(sessionID, path)
	if err != nil {
		_ = m.Close(sessionID)
		return Directory{}, err
	}
	return dir, nil
}

func (m *Manager) List(sessionID string, path string) (Directory, error) {
	s, err := m.get(sessionID)
	if err != nil {
		return Directory{}, err
	}

	resolved, err := s.client.RealPath(path)
	if err != nil || resolved == "" {
		resolved = path
	}

	entries, err := s.client.ReadDir(resolved)
	if err != nil {
		return Directory{}, fmt.Errorf("list sftp path %s: %w", resolved, err)
	}

	items := make([]FileEntry, 0, len(entries))
	for _, entry := range entries {
		items = append(items, FileEntry{
			Name:     entry.Name(),
			Path:     filepath.Join(resolved, entry.Name()),
			IsDir:    entry.IsDir(),
			Size:     entry.Size(),
			Mode:     entry.Mode().String(),
			Modified: entry.ModTime().Format(time.RFC3339),
		})
	}

	slices.SortFunc(items, func(a FileEntry, b FileEntry) int {
		if a.IsDir != b.IsDir {
			if a.IsDir {
				return -1
			}
			return 1
		}
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})

	return Directory{
		SessionID: sessionID,
		Path:      resolved,
		Entries:   items,
	}, nil
}

func (m *Manager) Download(sessionID string, remotePath string) (string, error) {
	s, err := m.get(sessionID)
	if err != nil {
		return "", err
	}

	src, err := s.client.Open(remotePath)
	if err != nil {
		return "", fmt.Errorf("open remote file: %w", err)
	}
	defer src.Close()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}

	targetDir := filepath.Join(homeDir, "Downloads", "MySSH")
	if err := os.MkdirAll(targetDir, 0o700); err != nil {
		return "", fmt.Errorf("create download dir: %w", err)
	}

	baseName := filepath.Base(remotePath)
	targetPath := filepath.Join(targetDir, baseName)
	targetPath = nextAvailablePath(targetPath)

	dst, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "", fmt.Errorf("create local file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("download remote file: %w", err)
	}

	return targetPath, nil
}

func (m *Manager) Close(sessionID string) error {
	m.mu.Lock()
	s := m.sessions[sessionID]
	if s != nil {
		delete(m.sessions, sessionID)
	}
	m.mu.Unlock()

	if s == nil {
		return nil
	}

	_ = s.client.Close()
	if s.ssh != nil {
		return s.ssh.Close()
	}
	return nil
}

func (m *Manager) get(sessionID string) (*session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.sessions[sessionID]
	if s == nil {
		return nil, fmt.Errorf("sftp session not found")
	}
	return s, nil
}

func nextAvailablePath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	ext := filepath.Ext(path)
	base := path[:len(path)-len(ext)]
	for index := 1; ; index++ {
		candidate := fmt.Sprintf("%s-%d%s", base, index, ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}
