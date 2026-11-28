package service

import (
	"context"
	"time"

	"github.com/tutorgrid/tutorgrid/internal/lib/job"
	"github.com/tutorgrid/tutorgrid/internal/lib/storage"
	"github.com/tutorgrid/tutorgrid/internal/repository"
	"github.com/tutorgrid/tutorgrid/internal/server"
)

type Services struct {
	Auth     *AuthService
	Tenancy  *TenancyService
	Resource *ResourceService
	Job      *job.JobService
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

	initCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	b2Store, err := storage.NewB2Store(initCtx, s.Config.Storage)
	if err != nil {
		return nil, err
	}

	resourceService := NewResourceService(repos.Resource, b2Store, s.Job, s.Logger)
	if s.Job != nil {
		s.Job.SetResourceJobHandler(resourceService)
	}

	return &Services{
		Job:      s.Job,
		Auth:     authService,
		Tenancy:  tenancyService,
		Resource: resourceService,
	}, nil
}
