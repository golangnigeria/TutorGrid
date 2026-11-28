package handler

import (
	"github.com/tutorgrid/tutorgrid/internal/server"
	"github.com/tutorgrid/tutorgrid/internal/service"
)

type Handlers struct {
	Health   *HealthHandler
	OpenAPI  *OpenAPIHandler
	Tenancy  *TenancyHandler
	Resource *ResourceHandler
}

func NewHandlers(s *server.Server, services *service.Services) *Handlers {
	return &Handlers{
		Health:   NewHealthHandler(s),
		OpenAPI:  NewOpenAPIHandler(s),
		Tenancy:  NewTenancyHandler(s, services.Tenancy, services.Auth),
		Resource: NewResourceHandler(s, services.Resource),
	}
}
