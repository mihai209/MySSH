package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	appsvc "myssh/internal/app"
	"myssh/internal/domain"
	"myssh/internal/secret"
	"myssh/internal/sftpclient"
	"myssh/internal/sshclient"
)

type App struct {
	ctx          context.Context
	service      *appsvc.Service
	secretStore  *secret.Store
	sshManager   *sshclient.Manager
	sftpManager  *sftpclient.Manager
	dataDir      string
	pendingHosts map[string]*sshclient.UnknownHostError
}

type ProfileDTO struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Username         string `json:"username"`
	Host             string `json:"host"`
	Port             int    `json:"port"`
	AuthKind         string `json:"authKind"`
	KeySource        string `json:"keySource,omitempty"`
	KeyPath          string `json:"keyPath,omitempty"`
	KeyPathExists    bool   `json:"keyPathExists"`
	HasStoredSecret  bool   `json:"hasStoredSecret"`
	SecretRef        string `json:"secretRef,omitempty"`
	HasPassphrase    bool   `json:"hasPassphrase"`
	HasConnectSecret bool   `json:"hasConnectSecret"`
}

type DashboardDTO struct {
	Profiles         []ProfileDTO `json:"profiles"`
	DataDir          string       `json:"dataDir"`
	AgentCount       int          `json:"agentCount"`
	KeyCount         int          `json:"keyCount"`
	PasswordCount    int          `json:"passwordCount"`
	RecommendedAuth  string       `json:"recommendedAuth"`
	SecurityHeadline string       `json:"securityHeadline"`
}

type SFTPFileDTO struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	IsDir    bool   `json:"isDir"`
	Size     int64  `json:"size"`
	Mode     string `json:"mode"`
	Modified string `json:"modified"`
}

type SFTPDirectoryDTO struct {
	SessionID string        `json:"sessionId"`
	Path      string        `json:"path"`
	Parent    string        `json:"parent"`
	Entries   []SFTPFileDTO `json:"entries"`
}

type SaveProfileInput struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Username           string `json:"username"`
	Host               string `json:"host"`
	Port               int    `json:"port"`
	AuthKind           string `json:"authKind"`
	KeySource          string `json:"keySource"`
	KeyPath            string `json:"keyPath"`
	SecretValue        string `json:"secretValue"`
	PassphraseValue    string `json:"passphraseValue"`
	ConnectSecretValue string `json:"connectSecretValue"`
}

func NewApp(service *appsvc.Service, secretStore *secret.Store, dataDir string) *App {
	app := &App{
		service:      service,
		secretStore:  secretStore,
		sftpManager:  sftpclient.NewManager(),
		dataDir:      dataDir,
		pendingHosts: map[string]*sshclient.UnknownHostError{},
	}
	app.sshManager = sshclient.NewManager(func(name string, payload interface{}) {
		if app.ctx != nil {
			runtime.EventsEmit(app.ctx, name, payload)
		}
	})
	return app
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) Dashboard() (DashboardDTO, error) {
	profiles, err := a.service.ListProfiles()
	if err != nil {
		return DashboardDTO{}, err
	}

	dashboard := DashboardDTO{
		Profiles:         make([]ProfileDTO, 0, len(profiles)),
		DataDir:          a.dataDir,
		RecommendedAuth:  "agent",
		SecurityHeadline: "Passwords and pasted private keys are persisted in your OS keyring. Agent mode remains the safest default.",
	}

	for _, profile := range profiles {
		switch profile.AuthKind {
		case domain.AuthPassword:
			dashboard.PasswordCount++
		case domain.AuthPrivateKey, domain.AuthAgentFallbackKey:
			dashboard.KeyCount++
		default:
			dashboard.AgentCount++
		}

		dashboard.Profiles = append(dashboard.Profiles, toProfileDTO(profile))
	}

	return dashboard, nil
}

func (a *App) SaveProfile(input SaveProfileInput) (ProfileDTO, error) {
	if strings.TrimSpace(input.AuthKind) == "" {
		input.AuthKind = string(domain.AuthAgent)
	}

	profile := domain.Profile{
		ID:        strings.TrimSpace(input.ID),
		Name:      input.Name,
		Username:  input.Username,
		Host:      input.Host,
		Port:      input.Port,
		AuthKind:  domain.AuthKind(input.AuthKind),
		KeySource: domain.KeySource(strings.TrimSpace(input.KeySource)),
		KeyPath:   input.KeyPath,
	}
	profile.Normalize()

	existingRef := ""
	existingPassphraseRef := ""
	existingConnectRef := ""
	if strings.TrimSpace(input.ID) != "" {
		profiles, err := a.service.ListProfiles()
		if err != nil {
			return ProfileDTO{}, err
		}
		for _, existing := range profiles {
			if existing.ID == profile.ID {
				existingRef = existing.SecretRef
				profile.SecretRef = existing.SecretRef
				profile.HasStoredSecret = existing.HasStoredSecret
				existingPassphraseRef = existing.PassphraseRef
				profile.PassphraseRef = existing.PassphraseRef
				profile.HasPassphrase = existing.HasPassphrase
				existingConnectRef = existing.ConnectSecretRef
				profile.ConnectSecretRef = existing.ConnectSecretRef
				profile.HasConnectSecret = existing.HasConnectSecret
				break
			}
		}
	}

	secretValue := strings.TrimSpace(input.SecretValue)
	if profile.AuthKind == domain.AuthPassword || ((profile.AuthKind == domain.AuthPrivateKey || profile.AuthKind == domain.AuthAgentFallbackKey) && profile.KeySource == domain.KeySourceContent) {
		if secretValue == "" && existingRef == "" {
			return ProfileDTO{}, fmt.Errorf("secret value is required for %s", profile.AuthKind)
		}
		if secretValue != "" {
			if profile.SecretRef == "" {
				profile.SecretRef = "profile:" + profile.ID
			}
			if err := a.secretStore.Set(profile.SecretRef, input.SecretValue); err != nil {
				return ProfileDTO{}, fmt.Errorf("store secret in keyring: %w", err)
			}
			profile.HasStoredSecret = true
		}
	} else {
		if existingRef != "" {
			if err := a.secretStore.Delete(existingRef); err != nil {
				return ProfileDTO{}, fmt.Errorf("delete previous secret from keyring: %w", err)
			}
		}
		profile.SecretRef = ""
		profile.HasStoredSecret = false
	}

	passphraseValue := strings.TrimSpace(input.PassphraseValue)
	switch {
	case profile.AuthKind == domain.AuthPrivateKey || profile.AuthKind == domain.AuthAgentFallbackKey:
		if passphraseValue != "" {
			if profile.PassphraseRef == "" {
				profile.PassphraseRef = "profile:" + profile.ID + ":passphrase"
			}
			if err := a.secretStore.Set(profile.PassphraseRef, input.PassphraseValue); err != nil {
				return ProfileDTO{}, fmt.Errorf("store key passphrase in keyring: %w", err)
			}
			profile.HasPassphrase = true
		} else if existingPassphraseRef == "" {
			profile.PassphraseRef = ""
			profile.HasPassphrase = false
		}
	default:
		if existingPassphraseRef != "" {
			if err := a.secretStore.Delete(existingPassphraseRef); err != nil {
				return ProfileDTO{}, fmt.Errorf("delete key passphrase from keyring: %w", err)
			}
		}
		profile.PassphraseRef = ""
		profile.HasPassphrase = false
	}

	connectSecretValue := strings.TrimSpace(input.ConnectSecretValue)
	switch {
	case connectSecretValue != "":
		if profile.ConnectSecretRef == "" {
			profile.ConnectSecretRef = "profile:" + profile.ID + ":connect-secret"
		}
		if err := a.secretStore.Set(profile.ConnectSecretRef, input.ConnectSecretValue); err != nil {
			return ProfileDTO{}, fmt.Errorf("store connect secret in keyring: %w", err)
		}
		profile.HasConnectSecret = true
	case existingConnectRef == "":
		profile.ConnectSecretRef = ""
		profile.HasConnectSecret = false
	}

	saved, err := a.service.AddProfile(profile)
	if err != nil {
		return ProfileDTO{}, err
	}

	return toProfileDTO(saved), nil
}

func (a *App) DeleteProfile(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}

	profiles, err := a.service.ListProfiles()
	if err != nil {
		return err
	}

	for _, profile := range profiles {
		if profile.ID != id {
			continue
		}
		if profile.SecretRef != "" {
			if err := a.secretStore.Delete(profile.SecretRef); err != nil {
				return fmt.Errorf("delete secret from keyring: %w", err)
			}
		}
		if profile.ConnectSecretRef != "" {
			if err := a.secretStore.Delete(profile.ConnectSecretRef); err != nil {
				return fmt.Errorf("delete connect secret from keyring: %w", err)
			}
		}
		if profile.PassphraseRef != "" {
			if err := a.secretStore.Delete(profile.PassphraseRef); err != nil {
				return fmt.Errorf("delete key passphrase from keyring: %w", err)
			}
		}
		break
	}

	return a.service.DeleteProfile(id)
}

func (a *App) Ping() string {
	return fmt.Sprintf("MySSH backend ready: %s", a.dataDir)
}

func (a *App) ConnectProfile(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("profile id is required")
	}

	profiles, err := a.service.ListProfiles()
	if err != nil {
		return "", err
	}

	var profile domain.Profile
	found := false
	for _, item := range profiles {
		if item.ID == id {
			profile = item
			found = true
			break
		}
	}

	if !found {
		return "", fmt.Errorf("profile not found")
	}

	secretValue, passphraseValue, err := a.loadProfileSecrets(profile)
	if err != nil {
		return "", err
	}

	connectSecretValue := ""
	if profile.ConnectSecretRef != "" {
		connectSecretValue, err = a.secretStore.Get(profile.ConnectSecretRef)
		if err != nil {
			return "", fmt.Errorf("load connect secret from keyring: %w", err)
		}
	}

	sessionID, err := a.sshManager.Connect(a.ctx, profile, secretValue, passphraseValue, connectSecretValue)
	if err != nil {
		var unknownHostErr *sshclient.UnknownHostError
		if errors.As(err, &unknownHostErr) {
			a.pendingHosts[profile.ID] = unknownHostErr
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "ssh:hostkey", map[string]interface{}{
					"sessionId":    profile.ID,
					"host":        unknownHostErr.HostWithPort,
					"fingerprint": unknownHostErr.Fingerprint,
					"message":     fmt.Sprintf("Unknown host key for %s", unknownHostErr.HostWithPort),
				})
			}
		}
		return "", err
	}

	delete(a.pendingHosts, profile.ID)
	return sessionID, nil
}

func (a *App) OpenSFTP(id string) (SFTPDirectoryDTO, error) {
	profile, secretValue, passphraseValue, err := a.loadProfileByID(id)
	if err != nil {
		return SFTPDirectoryDTO{}, err
	}

	dir, err := a.sftpManager.Open(profile, secretValue, passphraseValue)
	if err != nil {
		var unknownHostErr *sshclient.UnknownHostError
		if errors.As(err, &unknownHostErr) {
			a.pendingHosts[profile.ID] = unknownHostErr
		}
		return SFTPDirectoryDTO{}, err
	}

	return toSFTPDirectoryDTO(dir), nil
}

func (a *App) ListSFTP(sessionID string, path string) (SFTPDirectoryDTO, error) {
	dir, err := a.sftpManager.List(strings.TrimSpace(sessionID), path)
	if err != nil {
		return SFTPDirectoryDTO{}, err
	}
	return toSFTPDirectoryDTO(dir), nil
}

func (a *App) DownloadSFTPFile(sessionID string, remotePath string) (string, error) {
	return a.sftpManager.Download(strings.TrimSpace(sessionID), strings.TrimSpace(remotePath))
}

func (a *App) UploadSFTPFile(sessionID string) (SFTPDirectoryDTO, error) {
	if a.ctx == nil {
		return SFTPDirectoryDTO{}, fmt.Errorf("ui context not ready")
	}
	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Upload File to SFTP",
	})
	if err != nil {
		return SFTPDirectoryDTO{}, err
	}
	if strings.TrimSpace(filePath) == "" {
		return SFTPDirectoryDTO{}, fmt.Errorf("no local file selected")
	}

	dir, err := a.sftpManager.Upload(strings.TrimSpace(sessionID), filePath, ".")
	if err != nil {
		return SFTPDirectoryDTO{}, err
	}
	return toSFTPDirectoryDTO(dir), nil
}

func (a *App) UploadSFTPFileToPath(sessionID string, remoteDir string) (SFTPDirectoryDTO, error) {
	if a.ctx == nil {
		return SFTPDirectoryDTO{}, fmt.Errorf("ui context not ready")
	}
	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Upload File to SFTP",
	})
	if err != nil {
		return SFTPDirectoryDTO{}, err
	}
	if strings.TrimSpace(filePath) == "" {
		return SFTPDirectoryDTO{}, fmt.Errorf("no local file selected")
	}

	dir, err := a.sftpManager.Upload(strings.TrimSpace(sessionID), filePath, strings.TrimSpace(remoteDir))
	if err != nil {
		return SFTPDirectoryDTO{}, err
	}
	return toSFTPDirectoryDTO(dir), nil
}

func (a *App) RenameSFTPPath(sessionID string, oldPath string, newName string) (SFTPDirectoryDTO, error) {
	oldPath = strings.TrimSpace(oldPath)
	newName = strings.TrimSpace(newName)
	if oldPath == "" || newName == "" {
		return SFTPDirectoryDTO{}, fmt.Errorf("old path and new name are required")
	}

	newPath := filepath.Join(filepath.Dir(oldPath), newName)
	dir, err := a.sftpManager.Rename(strings.TrimSpace(sessionID), oldPath, newPath)
	if err != nil {
		return SFTPDirectoryDTO{}, err
	}
	return toSFTPDirectoryDTO(dir), nil
}

func (a *App) DeleteSFTPPath(sessionID string, remotePath string, isDir bool) (SFTPDirectoryDTO, error) {
	dir, err := a.sftpManager.Delete(strings.TrimSpace(sessionID), strings.TrimSpace(remotePath), isDir)
	if err != nil {
		return SFTPDirectoryDTO{}, err
	}
	return toSFTPDirectoryDTO(dir), nil
}

func (a *App) MkdirSFTP(sessionID string, parentDir string, name string) (SFTPDirectoryDTO, error) {
	dir, err := a.sftpManager.Mkdir(strings.TrimSpace(sessionID), strings.TrimSpace(parentDir), strings.TrimSpace(name))
	if err != nil {
		return SFTPDirectoryDTO{}, err
	}
	return toSFTPDirectoryDTO(dir), nil
}

func (a *App) CloseSFTP(sessionID string) error {
	return a.sftpManager.Close(strings.TrimSpace(sessionID))
}

func (a *App) ConnectLocalShell() (string, error) {
	return a.sshManager.ConnectLocalShell(a.ctx)
}

func (a *App) SendTerminalInput(sessionID string, input string) error {
	return a.sshManager.SendInput(sessionID, input)
}

func (a *App) ResizeTerminal(sessionID string, cols int, rows int) error {
	return a.sshManager.Resize(sessionID, cols, rows)
}

func (a *App) CopyToClipboard(text string) {
	if a.ctx == nil {
		return
	}
	runtime.ClipboardSetText(a.ctx, text)
}

func (a *App) PasteFromClipboard() string {
	if a.ctx == nil {
		return ""
	}
	text, err := runtime.ClipboardGetText(a.ctx)
	if err != nil {
		return ""
	}
	return text
}

func (a *App) DisconnectTerminal(sessionID string) {
	a.sshManager.Disconnect(sessionID)
}

func (a *App) TrustPendingHost(sessionID string) error {
	pendingHost := a.pendingHosts[sessionID]
	if pendingHost == nil {
		return fmt.Errorf("no pending unknown host")
	}

	if err := sshclient.AddKnownHost(pendingHost.HostWithPort, pendingHost.Key); err != nil {
		return err
	}

	delete(a.pendingHosts, sessionID)
	return nil
}

func (a *App) loadProfileByID(id string) (domain.Profile, string, string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.Profile{}, "", "", fmt.Errorf("profile id is required")
	}

	profiles, err := a.service.ListProfiles()
	if err != nil {
		return domain.Profile{}, "", "", err
	}

	for _, item := range profiles {
		if item.ID == id {
			secretValue, passphraseValue, err := a.loadProfileSecrets(item)
			return item, secretValue, passphraseValue, err
		}
	}

	return domain.Profile{}, "", "", fmt.Errorf("profile not found")
}

func (a *App) loadProfileSecrets(profile domain.Profile) (string, string, error) {
	secretValue := ""
	if profile.AuthKind == domain.AuthPassword || ((profile.AuthKind == domain.AuthPrivateKey || profile.AuthKind == domain.AuthAgentFallbackKey) && profile.KeySource == domain.KeySourceContent) {
		if profile.SecretRef == "" {
			if profile.AuthKind == domain.AuthPassword {
				return "", "", fmt.Errorf("password profile has no stored secret")
			}
			return "", "", fmt.Errorf("private key content profile has no stored secret")
		}
		value, err := a.secretStore.Get(profile.SecretRef)
		if err != nil {
			if profile.AuthKind == domain.AuthPassword {
				return "", "", fmt.Errorf("load password from keyring: %w", err)
			}
			return "", "", fmt.Errorf("load private key content from keyring: %w", err)
		}
		secretValue = value
	}

	passphraseValue := ""
	if profile.PassphraseRef != "" {
		value, err := a.secretStore.Get(profile.PassphraseRef)
		if err != nil {
			return "", "", fmt.Errorf("load key passphrase from keyring: %w", err)
		}
		passphraseValue = value
	}

	return secretValue, passphraseValue, nil
}

func toProfileDTO(profile domain.Profile) ProfileDTO {
	keyPathExists := false
	if (profile.AuthKind == domain.AuthPrivateKey || profile.AuthKind == domain.AuthAgentFallbackKey) && profile.KeySource == domain.KeySourcePath && profile.KeyPath != "" {
		if _, err := os.Stat(profile.KeyPath); err == nil {
			keyPathExists = true
		}
	}

	return ProfileDTO{
		ID:               profile.ID,
		Name:             profile.Name,
		Username:         profile.Username,
		Host:             profile.Host,
		Port:             profile.Port,
		AuthKind:         string(profile.AuthKind),
		KeySource:        string(profile.KeySource),
		KeyPath:          profile.KeyPath,
		KeyPathExists:    keyPathExists,
		HasStoredSecret:  profile.HasStoredSecret,
		SecretRef:        profile.SecretRef,
		HasPassphrase:    profile.HasPassphrase,
		HasConnectSecret: profile.HasConnectSecret,
	}
}

func toSFTPDirectoryDTO(dir sftpclient.Directory) SFTPDirectoryDTO {
	parent := filepath.Dir(dir.Path)
	if parent == "." || parent == dir.Path {
		parent = ""
	}

	items := make([]SFTPFileDTO, 0, len(dir.Entries))
	for _, entry := range dir.Entries {
		items = append(items, SFTPFileDTO{
			Name:     entry.Name,
			Path:     entry.Path,
			IsDir:    entry.IsDir,
			Size:     entry.Size,
			Mode:     entry.Mode,
			Modified: entry.Modified,
		})
	}

	return SFTPDirectoryDTO{
		SessionID: dir.SessionID,
		Path:      dir.Path,
		Parent:    parent,
		Entries:   items,
	}
}
