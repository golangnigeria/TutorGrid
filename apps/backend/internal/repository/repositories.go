package repository

import "github.com/tutorgrid/tutorgrid/internal/server"

type Repositories struct {
	Tenancy  *TenancyRepository
	Resource *ResourceRepository
}

func NewRepositories(s *server.Server) *Repositories {
	return &Repositories{
		Tenancy:  NewTenancyRepository(s.DB.Pool),
		Resource: NewResourceRepository(s.DB.Pool),
	}
}
