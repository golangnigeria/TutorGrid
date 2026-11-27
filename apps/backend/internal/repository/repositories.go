package repository

import "github.com/tutorgrid/tutorgrid/internal/server"

type Repositories struct {
	Tenancy *TenancyRepository
}

func NewRepositories(s *server.Server) *Repositories {
	return &Repositories{
		Tenancy: NewTenancyRepository(s.DB.Pool),
	}
}
