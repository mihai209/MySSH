package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	appsvc "myssh/internal/app"
	"myssh/internal/domain"
	"myssh/internal/secret"
	"myssh/internal/sshclient"
)

type App struct {
	ctx         context.Context
	service     *appsvc.Service
	secretStore *secret.Store
	sshManager  *sshclient.Manager
	dataDir     string
	pendingHost *sshclient.UnknownHostError
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
	HasStoredSecret  bool   `json:"hasStoredSecret"`
	SecretRef        string `json:"secretRef,omitempty"`
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
	ConnectSecretValue string `json:"connectSecretValue"`
}

func NewApp(service *appsvc.Service, secretStore *secret.Store, dataDir string) *App {
	app := &App{
		service:     service,
		secretStore: secretStore,
		dataDir:     dataDir,
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
		case domain.AuthPrivateKey:
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
				existingConnectRef = existing.ConnectSecretRef
				profile.ConnectSecretRef = existing.ConnectSecretRef
				profile.HasConnectSecret = existing.HasConnectSecret
				break
			}
		}
	}

	secretValue := strings.TrimSpace(input.SecretValue)
	if profile.AuthKind == domain.AuthPassword || (profile.AuthKind == domain.AuthPrivateKey && profile.KeySource == domain.KeySourceContent) {
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
		break
	}

	return a.service.DeleteProfile(id)
}

func (a *App) Ping() string {
	return fmt.Sprintf("MySSH backend ready: %s", a.dataDir)
}

func (a *App) ConnectProfile(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("profile id is required")
	}

	profiles, err := a.service.ListProfiles()
	if err != nil {
		return err
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
		return fmt.Errorf("profile not found")
	}

	secretValue := ""
	if profile.AuthKind == domain.AuthPassword {
		if profile.SecretRef == "" {
			return fmt.Errorf("password profile has no stored secret")
		}
		secretValue, err = a.secretStore.Get(profile.SecretRef)
		if err != nil {
			return fmt.Errorf("load password from keyring: %w", err)
		}
	}

	connectSecretValue := ""
	if profile.ConnectSecretRef != "" {
		connectSecretValue, err = a.secretStore.Get(profile.ConnectSecretRef)
		if err != nil {
			return fmt.Errorf("load connect secret from keyring: %w", err)
		}
	}

	err = a.sshManager.Connect(a.ctx, profile, secretValue, connectSecretValue)
	if err != nil {
		var unknownHostErr *sshclient.UnknownHostError
		if errors.As(err, &unknownHostErr) {
			a.pendingHost = unknownHostErr
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "ssh:hostkey", map[string]interface{}{
					"host":        unknownHostErr.HostWithPort,
					"fingerprint": unknownHostErr.Fingerprint,
					"message":     fmt.Sprintf("Unknown host key for %s", unknownHostErr.HostWithPort),
				})
			}
		}
		return err
	}

	a.pendingHost = nil
	return nil
}

func (a *App) SendTerminalInput(input string) error {
	return a.sshManager.SendInput(input)
}

func (a *App) ResizeTerminal(cols int, rows int) error {
	return a.sshManager.Resize(cols, rows)
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

func (a *App) DisconnectTerminal() {
	a.sshManager.Disconnect()
}

func (a *App) TrustPendingHost() error {
	if a.pendingHost == nil {
		return fmt.Errorf("no pending unknown host")
	}

	if err := sshclient.AddKnownHost(a.pendingHost.HostWithPort, a.pendingHost.Key); err != nil {
		return err
	}

	a.pendingHost = nil
	return nil
}

func toProfileDTO(profile domain.Profile) ProfileDTO {
	return ProfileDTO{
		ID:               profile.ID,
		Name:             profile.Name,
		Username:         profile.Username,
		Host:             profile.Host,
		Port:             profile.Port,
		AuthKind:         string(profile.AuthKind),
		KeySource:        string(profile.KeySource),
		KeyPath:          profile.KeyPath,
		HasStoredSecret:  profile.HasStoredSecret,
		SecretRef:        profile.SecretRef,
		HasConnectSecret: profile.HasConnectSecret,
	}
}
