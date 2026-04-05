package store

import (
	"path/filepath"
	"testing"

	"myssh/internal/domain"
)

func TestProfileRepositorySaveAndList(t *testing.T) {
	repo := NewProfileRepository(filepath.Join(t.TempDir(), "data"))

	profile := domain.Profile{
		ID:       "abc123",
		Name:     "Local",
		Username: "mihai",
		Host:     "127.0.0.1",
		Port:     22,
		AuthKind: domain.AuthAgent,
	}

	if err := repo.Save(profile); err != nil {
		t.Fatalf("save profile: %v", err)
	}

	profiles, err := repo.List()
	if err != nil {
		t.Fatalf("list profiles: %v", err)
	}

	if len(profiles) != 1 {
		t.Fatalf("expected one profile, got %d", len(profiles))
	}
	if profiles[0].Host != "127.0.0.1" {
		t.Fatalf("expected host to persist, got %s", profiles[0].Host)
	}
}

func TestProfileRepositoryDelete(t *testing.T) {
	repo := NewProfileRepository(filepath.Join(t.TempDir(), "data"))

	profile := domain.Profile{
		ID:       "delete-me",
		Name:     "Temp",
		Username: "mihai",
		Host:     "192.168.0.10",
		Port:     22,
		AuthKind: domain.AuthAgent,
	}

	if err := repo.Save(profile); err != nil {
		t.Fatalf("save profile: %v", err)
	}

	if err := repo.Delete(profile.ID); err != nil {
		t.Fatalf("delete profile: %v", err)
	}

	profiles, err := repo.List()
	if err != nil {
		t.Fatalf("list profiles: %v", err)
	}

	if len(profiles) != 0 {
		t.Fatalf("expected zero profiles after delete, got %d", len(profiles))
	}
}
