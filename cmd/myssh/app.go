package main

import (
	"context"
	"fmt"
	"strings"

	appsvc "myssh/internal/app"
	"myssh/internal/domain"
	"myssh/internal/secret"
)

type App struct {
	ctx         context.Context
	service     *appsvc.Service
	secretStore *secret.Store
	dataDir     string
}

type ProfileDTO struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Username        string `json:"username"`
	Host            string `json:"host"`
	Port            int    `json:"port"`
	AuthKind        string `json:"authKind"`
	KeySource       string `json:"keySource,omitempty"`
	KeyPath         string `json:"keyPath,omitempty"`
	HasStoredSecret bool   `json:"hasStoredSecret"`
	SecretRef       string `json:"secretRef,omitempty"`
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
	ID          string `json:"id"`
	Name        string `json:"name"`
	Username    string `json:"username"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	AuthKind    string `json:"authKind"`
	KeySource   string `json:"keySource"`
	KeyPath     string `json:"keyPath"`
	SecretValue string `json:"secretValue"`
}

func NewApp(service *appsvc.Service, secretStore *secret.Store, dataDir string) *App {
	return &App{
		service:     service,
		secretStore: secretStore,
		dataDir:     dataDir,
	}
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
		if profile.ID == id && profile.SecretRef != "" {
			if err := a.secretStore.Delete(profile.SecretRef); err != nil {
				return fmt.Errorf("delete secret from keyring: %w", err)
			}
			break
		}
	}

	return a.service.DeleteProfile(id)
}

func (a *App) Ping() string {
	return fmt.Sprintf("MySSH backend ready: %s", a.dataDir)
}

func toProfileDTO(profile domain.Profile) ProfileDTO {
	return ProfileDTO{
		ID:              profile.ID,
		Name:            profile.Name,
		Username:        profile.Username,
		Host:            profile.Host,
		Port:            profile.Port,
		AuthKind:        string(profile.AuthKind),
		KeySource:       string(profile.KeySource),
		KeyPath:         profile.KeyPath,
		HasStoredSecret: profile.HasStoredSecret,
		SecretRef:       profile.SecretRef,
	}
}
