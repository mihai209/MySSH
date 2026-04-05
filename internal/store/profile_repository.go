package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"myssh/internal/domain"
)

const profilesFilename = "profiles.json"

type ProfileRepository struct {
	dataDir string
}

func NewProfileRepository(dataDir string) *ProfileRepository {
	return &ProfileRepository{dataDir: dataDir}
}

func (r *ProfileRepository) List() ([]domain.Profile, error) {
	state, err := r.load()
	if err != nil {
		return nil, err
	}

	return state.Profiles, nil
}

func (r *ProfileRepository) Save(profile domain.Profile) error {
	if err := profile.Validate(); err != nil {
		return err
	}

	state, err := r.load()
	if err != nil {
		return err
	}

	index := slices.IndexFunc(state.Profiles, func(existing domain.Profile) bool {
		return existing.ID == profile.ID
	})

	if index >= 0 {
		state.Profiles[index] = profile
	} else {
		state.Profiles = append(state.Profiles, profile)
	}

	return r.save(state)
}

func (r *ProfileRepository) Delete(id string) error {
	state, err := r.load()
	if err != nil {
		return err
	}

	state.Profiles = slices.DeleteFunc(state.Profiles, func(profile domain.Profile) bool {
		return profile.ID == id
	})

	return r.save(state)
}

type profileState struct {
	Profiles []domain.Profile `json:"profiles"`
}

func (r *ProfileRepository) load() (profileState, error) {
	if err := os.MkdirAll(r.dataDir, 0o700); err != nil {
		return profileState{}, fmt.Errorf("create data dir: %w", err)
	}

	path := filepath.Join(r.dataDir, profilesFilename)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return profileState{Profiles: []domain.Profile{}}, nil
		}
		return profileState{}, fmt.Errorf("read profiles file: %w", err)
	}

	var state profileState
	if err := json.Unmarshal(data, &state); err != nil {
		return profileState{}, fmt.Errorf("decode profiles file: %w", err)
	}

	if state.Profiles == nil {
		state.Profiles = []domain.Profile{}
	}

	return state, nil
}

func (r *ProfileRepository) save(state profileState) error {
	path := filepath.Join(r.dataDir, profilesFilename)

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode profiles file: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write profiles file: %w", err)
	}

	return nil
}
