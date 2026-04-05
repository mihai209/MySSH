package main

import (
	"context"
	"fmt"
	"strings"

	appsvc "myssh/internal/app"
	"myssh/internal/domain"
)

type App struct {
	ctx     context.Context
	service *appsvc.Service
	dataDir string
}

type ProfileDTO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Username  string `json:"username"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	AuthKind  string `json:"authKind"`
	SecretRef string `json:"secretRef,omitempty"`
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
	ID       string `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	AuthKind string `json:"authKind"`
}

func NewApp(service *appsvc.Service, dataDir string) *App {
	return &App{
		service: service,
		dataDir: dataDir,
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
		SecurityHeadline: "Secrets stay out of plain-text storage. Agent mode is the safest default until keyring support lands.",
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
		ID:       strings.TrimSpace(input.ID),
		Name:     input.Name,
		Username: input.Username,
		Host:     input.Host,
		Port:     input.Port,
		AuthKind: domain.AuthKind(input.AuthKind),
	}

	saved, err := a.service.AddProfile(profile)
	if err != nil {
		return ProfileDTO{}, err
	}

	return toProfileDTO(saved), nil
}

func (a *App) DeleteProfile(id string) error {
	return a.service.DeleteProfile(strings.TrimSpace(id))
}

func (a *App) Ping() string {
	return fmt.Sprintf("MySSH backend ready: %s", a.dataDir)
}

func toProfileDTO(profile domain.Profile) ProfileDTO {
	return ProfileDTO{
		ID:        profile.ID,
		Name:      profile.Name,
		Username:  profile.Username,
		Host:      profile.Host,
		Port:      profile.Port,
		AuthKind:  string(profile.AuthKind),
		SecretRef: profile.SecretRef,
	}
}
