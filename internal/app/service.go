package app

import "myssh/internal/domain"

type ProfileRepository interface {
	List() ([]domain.Profile, error)
	Save(domain.Profile) error
}

type Service struct {
	repo ProfileRepository
}

func NewService(repo ProfileRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListProfiles() ([]domain.Profile, error) {
	return s.repo.List()
}

func (s *Service) AddProfile(profile domain.Profile) (domain.Profile, error) {
	profile.Normalize()
	if err := profile.Validate(); err != nil {
		return domain.Profile{}, err
	}

	if err := s.repo.Save(profile); err != nil {
		return domain.Profile{}, err
	}

	return profile, nil
}
