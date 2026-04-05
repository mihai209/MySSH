package secret

import (
	"fmt"

	"github.com/99designs/keyring"
)

type Store struct {
	ring keyring.Keyring
}

func NewStore(appName string) (*Store, error) {
	ring, err := keyring.Open(keyring.Config{
		ServiceName:  appName,
		KeychainName: appName,
		AllowedBackends: []keyring.BackendType{
			keyring.SecretServiceBackend,
			keyring.KeychainBackend,
			keyring.WinCredBackend,
			keyring.KWalletBackend,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("open secure key store: %w", err)
	}

	return &Store{ring: ring}, nil
}

func (s *Store) Set(key string, value string) error {
	return s.ring.Set(keyring.Item{
		Key:  key,
		Data: []byte(value),
	})
}

func (s *Store) Delete(key string) error {
	err := s.ring.Remove(key)
	if err == keyring.ErrKeyNotFound {
		return nil
	}
	return err
}
