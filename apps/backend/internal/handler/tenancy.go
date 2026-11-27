package handler

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/tutorgrid/tutorgrid/internal/errs"
	"github.com/tutorgrid/tutorgrid/internal/middleware"
	"github.com/tutorgrid/tutorgrid/internal/model"
	"github.com/tutorgrid/tutorgrid/internal/repository"
	"github.com/tutorgrid/tutorgrid/internal/server"
	"github.com/tutorgrid/tutorgrid/internal/service"
)

type TenancyHandler struct {
	Handler
	tenancySvc *service.TenancyService
	authSvc    *service.AuthService
}

func NewTenancyHandler(s *server.Server, tenancySvc *service.TenancyService, authSvc *service.AuthService) *TenancyHandler {
	return &TenancyHandler{
		Handler:    NewHandler(s),
		tenancySvc: tenancySvc,
		authSvc:    authSvc,
	}
}

func (h *TenancyHandler) Me(c echo.Context) error {
	user, ok := middleware.GetVerifiedUser(c)
	if !ok {
		return errs.NewUnauthorizedError("Unauthorized", false)
	}

	memberships, err := h.tenancySvc.ListUserMemberships(c.Request().Context(), user.ID)
	if err != nil {
		return errs.NewInternalServerError()
	}

	activeTenantID, err := h.tenancySvc.GetActiveTenantID(c.Request().Context(), user.ID)
	if err != nil {
		return errs.NewInternalServerError()
	}

	return c.JSON(http.StatusOK, map[string]any{
		"user":             user,
		"memberships":      memberships,
		"active_tenant_id": activeTenantID,
	})
}

func (h *TenancyHandler) Logout(c echo.Context) error {
	if token, ok := c.Get("session_token").(string); ok && token != "" && h.authSvc != nil {
		_ = h.authSvc.RevokeSession(c.Request().Context(), token)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "signed_out"})
}

func (h *TenancyHandler) CreateTenant(c echo.Context) error {
	user, ok := middleware.GetVerifiedUser(c)
	if !ok {
		return errs.NewUnauthorizedError("Unauthorized", false)
	}

	var req struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := c.Bind(&req); err != nil {
		return errs.NewBadRequestError("Invalid request body", false, nil, nil, nil)
	}

	membership, err := h.tenancySvc.CreateTenantWithAdmin(c.Request().Context(), user.ID, req.Name, req.Slug)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidName), errors.Is(err, service.ErrInvalidSlug):
			return errs.NewBadRequestError(err.Error(), false, nil, nil, nil)
		case errors.Is(err, repository.ErrSlugConflict):
			return c.JSON(http.StatusConflict, map[string]any{
				"code":    "CONFLICT",
				"message": "That workspace slug is already taken",
			})
		default:
			return errs.NewInternalServerError()
		}
	}

	return c.JSON(http.StatusCreated, map[string]any{"membership": membership})
}

func (h *TenancyHandler) SetActiveTenant(c echo.Context) error {
	user, ok := middleware.GetVerifiedUser(c)
	if !ok {
		return errs.NewUnauthorizedError("Unauthorized", false)
	}

	var req struct {
		TenantID string `json:"tenant_id"`
	}
	if err := c.Bind(&req); err != nil {
		return errs.NewBadRequestError("Invalid request body", false, nil, nil, nil)
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return errs.NewBadRequestError("Invalid tenant_id", false, nil, nil, nil)
	}

	role, err := h.tenancySvc.SetActiveTenant(c.Request().Context(), user.ID, tenantID)
	if err != nil {
		if errors.Is(err, repository.ErrNotMember) {
			return errs.NewForbiddenError("Not a member of the requested teaching space", false)
		}
		return errs.NewInternalServerError()
	}

	return c.JSON(http.StatusOK, map[string]any{
		"active_tenant_id": tenantID,
		"role":             role,
	})
}

func (h *TenancyHandler) ListCurrentMembers(c echo.Context) error {
	tc, ok := middleware.GetTenantContext(c)
	if !ok {
		return errs.NewForbiddenError("Tenant context required", false)
	}

	members, err := h.tenancySvc.ListCurrentTenantMembers(c.Request().Context(), tc)
	if err != nil {
		return errs.NewInternalServerError()
	}

	return c.JSON(http.StatusOK, map[string]any{"members": members})
}

func (h *TenancyHandler) ListMyPendingInvitations(c echo.Context) error {
	user, ok := middleware.GetVerifiedUser(c)
	if !ok {
		return errs.NewUnauthorizedError("Unauthorized", false)
	}

	invitations, err := h.tenancySvc.ListUserPendingInvitations(c.Request().Context(), user.ID, user.Email)
	if err != nil {
		return errs.NewInternalServerError()
	}

	return c.JSON(http.StatusOK, map[string]any{"invitations": invitations})
}

func (h *TenancyHandler) AcceptMyInvitation(c echo.Context) error {
	user, ok := middleware.GetVerifiedUser(c)
	if !ok {
		return errs.NewUnauthorizedError("Unauthorized", false)
	}

	invID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errs.NewBadRequestError("Invalid invitation ID", false, nil, nil, nil)
	}

	membership, err := h.tenancySvc.AcceptInvitationByID(c.Request().Context(), user.ID, user.Email, invID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrInvalidInvitation):
			return errs.NewNotFoundError("Invitation not found", false, nil)
		case errors.Is(err, repository.ErrInvitationExpired):
			return c.JSON(http.StatusGone, map[string]any{"code": "INVITATION_EXPIRED", "message": "Invitation has expired"})
		case errors.Is(err, repository.ErrInvitationRevoked):
			return c.JSON(http.StatusGone, map[string]any{"code": "INVITATION_REVOKED", "message": "Invitation has been revoked"})
		case errors.Is(err, repository.ErrInvitationAlreadyAccepted):
			return c.JSON(http.StatusConflict, map[string]any{"code": "INVITATION_ALREADY_ACCEPTED", "message": "Invitation has already been accepted"})
		case errors.Is(err, repository.ErrInvitationEmailMismatch):
			return errs.NewForbiddenError("This invitation is restricted to a different email address", false)
		default:
			if h.server != nil && h.server.Logger != nil {
				h.server.Logger.Error().Err(err).Msg("unexpected error accepting invitation by id")
			}
			return errs.NewInternalServerError()
		}
	}

	return c.JSON(http.StatusOK, map[string]any{"membership": membership})
}

func (h *TenancyHandler) ListCurrentInvitations(c echo.Context) error {
	tc, ok := middleware.GetTenantContext(c)
	if !ok {
		return errs.NewForbiddenError("Tenant context required", false)
	}

	invitations, err := h.tenancySvc.ListCurrentTenantInvitations(c.Request().Context(), tc)
	if err != nil {
		return errs.NewInternalServerError()
	}

	return c.JSON(http.StatusOK, map[string]any{"invitations": invitations})
}

func (h *TenancyHandler) CreateInvitation(c echo.Context) error {
	tc, ok := middleware.GetTenantContext(c)
	if !ok {
		return errs.NewForbiddenError("Tenant context required", false)
	}

	var req struct {
		Email string     `json:"email"`
		Role  model.Role `json:"role"`
	}
	if err := c.Bind(&req); err != nil {
		return errs.NewBadRequestError("Invalid request body", false, nil, nil, nil)
	}

	inv, err := h.tenancySvc.CreateInvitation(c.Request().Context(), tc, req.Email, req.Role)
	if err != nil {
		if errors.Is(err, service.ErrInvalidEmail) || errors.Is(err, service.ErrInvalidRole) {
			return errs.NewBadRequestError(err.Error(), false, nil, nil, nil)
		}
		return errs.NewInternalServerError()
	}

	return c.JSON(http.StatusCreated, map[string]any{"invitation": inv})
}

func (h *TenancyHandler) RevokeInvitation(c echo.Context) error {
	tc, ok := middleware.GetTenantContext(c)
	if !ok {
		return errs.NewForbiddenError("Tenant context required", false)
	}

	invID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errs.NewBadRequestError("Invalid invitation ID", false, nil, nil, nil)
	}

	if err := h.tenancySvc.RevokeInvitation(c.Request().Context(), tc, invID); err != nil {
		if errors.Is(err, repository.ErrInvalidInvitation) {
			return errs.NewNotFoundError("Invitation not found or already resolved", false, nil)
		}
		return errs.NewInternalServerError()
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "revoked"})
}

func (h *TenancyHandler) AcceptInvitation(c echo.Context) error {
	user, ok := middleware.GetVerifiedUser(c)
	if !ok {
		return errs.NewUnauthorizedError("Unauthorized", false)
	}

	var req struct {
		Token string `json:"token"`
	}
	if err := c.Bind(&req); err != nil {
		return errs.NewBadRequestError("Invalid request body", false, nil, nil, nil)
	}

	membership, err := h.tenancySvc.AcceptInvitation(c.Request().Context(), user.ID, user.Email, req.Token)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrInvalidInvitation):
			return errs.NewNotFoundError("Invitation not found", false, nil)
		case errors.Is(err, repository.ErrInvitationExpired):
			return c.JSON(http.StatusGone, map[string]any{"code": "INVITATION_EXPIRED", "message": "Invitation has expired"})
		case errors.Is(err, repository.ErrInvitationRevoked):
			return c.JSON(http.StatusGone, map[string]any{"code": "INVITATION_REVOKED", "message": "Invitation has been revoked"})
		case errors.Is(err, repository.ErrInvitationAlreadyAccepted):
			return c.JSON(http.StatusConflict, map[string]any{"code": "INVITATION_ALREADY_ACCEPTED", "message": "Invitation has already been accepted"})
		case errors.Is(err, repository.ErrInvitationEmailMismatch):
			return errs.NewForbiddenError("This invitation is restricted to a different email address", false)
		default:
			if h.server != nil && h.server.Logger != nil {
				h.server.Logger.Error().Err(err).Msg("unexpected error accepting invitation")
			}
			return errs.NewInternalServerError()
		}
	}

	return c.JSON(http.StatusOK, map[string]any{"membership": membership})
}
