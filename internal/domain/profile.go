package domain

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"
)

const (
	DefaultSSHPort = 22
)

type AuthKind string

const (
	AuthPassword   AuthKind = "password"
	AuthPrivateKey AuthKind = "private_key"
	AuthAgent      AuthKind = "agent"
)

type KeySource string

const (
	KeySourceNone    KeySource = ""
	KeySourcePath    KeySource = "path"
	KeySourceContent KeySource = "content"
)

type Profile struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Username         string    `json:"username"`
	Host             string    `json:"host"`
	Port             int       `json:"port"`
	AuthKind         AuthKind  `json:"auth_kind"`
	KeySource        KeySource `json:"key_source,omitempty"`
	KeyPath          string    `json:"key_path,omitempty"`
	SecretRef        string    `json:"secret_ref,omitempty"`
	HasStoredSecret  bool      `json:"has_stored_secret,omitempty"`
	ConnectSecretRef string    `json:"connect_secret_ref,omitempty"`
	HasConnectSecret bool      `json:"has_connect_secret,omitempty"`
}

func (p *Profile) Normalize() {
	p.Name = strings.TrimSpace(p.Name)
	p.Username = strings.TrimSpace(p.Username)
	p.Host = strings.TrimSpace(p.Host)
	p.KeyPath = strings.TrimSpace(p.KeyPath)

	if p.Port == 0 {
		p.Port = DefaultSSHPort
	}

	if p.ID == "" {
		p.ID = NewID()
	}
}

func (p Profile) Validate() error {
	if p.ID == "" {
		return errors.New("id is required")
	}
	if p.Name == "" {
		return errors.New("name is required")
	}
	if p.Username == "" {
		return errors.New("username is required")
	}
	if err := validateHost(p.Host); err != nil {
		return err
	}
	if p.Port < 1 || p.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	switch p.AuthKind {
	case AuthPassword:
		p.KeySource = KeySourceNone
		p.KeyPath = ""
	case AuthPrivateKey:
		switch p.KeySource {
		case KeySourcePath:
			if p.KeyPath == "" {
				return errors.New("key path is required for private_key path mode")
			}
		case KeySourceContent:
			p.KeyPath = ""
		default:
			return errors.New("key source must be path or content for private_key auth")
		}
	case AuthAgent:
		p.KeySource = KeySourceNone
		p.KeyPath = ""
	case "":
		return errors.New("auth kind is required")
	default:
		return fmt.Errorf("unsupported auth kind %q", p.AuthKind)
	}

	return nil
}

func NewID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		panic(fmt.Sprintf("read random bytes: %v", err))
	}
	return hex.EncodeToString(raw[:])
}

func validateHost(host string) error {
	if host == "" {
		return errors.New("host is required")
	}

	if net.ParseIP(host) != nil {
		return nil
	}

	labels := strings.Split(host, ".")
	if len(labels) == 0 {
		return errors.New("host is invalid")
	}

	for _, label := range labels {
		if label == "" {
			return errors.New("host is invalid")
		}
		for _, ch := range label {
			if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' {
				continue
			}
			return errors.New("host contains invalid characters")
		}
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return errors.New("host label cannot start or end with '-'")
		}
	}

	return nil
}
