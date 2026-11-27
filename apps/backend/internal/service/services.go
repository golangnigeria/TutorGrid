package service

import (
	"github.com/tutorgrid/tutorgrid/internal/lib/job"
	"github.com/tutorgrid/tutorgrid/internal/repository"
	"github.com/tutorgrid/tutorgrid/internal/server"
)

type Services struct {
	Auth    *AuthService
	Tenancy *TenancyService
	Job     *job.JobService
}

func NewServices(s *server.Server, repos *repository.Repositories) (*Services, error) {
	authService, err := NewAuthService(s)
	if err != nil {
		return nil, err
	}

	if s.Job != nil && repos != nil && repos.Tenancy != nil {
		s.Job.SetInvitationOutboxStore(repos.Tenancy)
	}

	tenancyService := NewTenancyService(repos.Tenancy, s.Job, s.Config.Auth.FrontendBaseURL, s.Logger)

	return &Services{
		Job:     s.Job,
		Auth:    authService,
		Tenancy: tenancyService,
	}, nil
}
