package domain

import "testing"

func TestProfileNormalizeAndValidate(t *testing.T) {
	profile := Profile{
		Name:     "  Prod  ",
		Username: " root ",
		Host:     " 192.168.1.10 ",
		AuthKind: AuthPassword,
	}

	profile.Normalize()

	if profile.Port != DefaultSSHPort {
		t.Fatalf("expected default port %d, got %d", DefaultSSHPort, profile.Port)
	}
	if profile.ID == "" {
		t.Fatal("expected generated id")
	}
	if profile.Name != "Prod" || profile.Username != "root" || profile.Host != "192.168.1.10" {
		t.Fatal("expected normalized string fields")
	}
	if err := profile.Validate(); err != nil {
		t.Fatalf("expected valid profile, got %v", err)
	}
}

func TestProfileValidateRejectsInvalidHost(t *testing.T) {
	profile := Profile{
		ID:       "id",
		Name:     "Prod",
		Username: "root",
		Host:     "bad host",
		Port:     22,
		AuthKind: AuthAgent,
	}

	if err := profile.Validate(); err == nil {
		t.Fatal("expected invalid host error")
	}
}
