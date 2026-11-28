package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/tutorgrid/tutorgrid/internal/errs"
	"github.com/tutorgrid/tutorgrid/internal/middleware"
	"github.com/tutorgrid/tutorgrid/internal/repository"
	"github.com/tutorgrid/tutorgrid/internal/server"
	"github.com/tutorgrid/tutorgrid/internal/service"
)

type ResourceHandler struct {
	Handler
	resourceSvc *service.ResourceService
}

func NewResourceHandler(s *server.Server, resourceSvc *service.ResourceService) *ResourceHandler {
	return &ResourceHandler{
		Handler:     NewHandler(s),
		resourceSvc: resourceSvc,
	}
}

func (h *ResourceHandler) List(c echo.Context) error {
	tc, ok := middleware.GetTenantContext(c)
	if !ok {
		return errs.NewForbiddenError("Tenant context required", false)
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	search := c.QueryParam("search")
	category := c.QueryParam("category")

	items, err := h.resourceSvc.ListResources(c.Request().Context(), tc, limit, offset, search, category)
	if err != nil {
		if h.server != nil && h.server.Logger != nil {
			h.server.Logger.Error().Err(err).Msg("failed to list resources")
		}
		return errs.NewInternalServerError()
	}

	return c.JSON(http.StatusOK, map[string]any{"resources": items})
}

func (h *ResourceHandler) RequestUploadURL(c echo.Context) error {
	tc, ok := middleware.GetTenantContext(c)
	if !ok {
		return errs.NewForbiddenError("Tenant context required", false)
	}

	var req struct {
		Filename    string `json:"filename"`
		ContentType string `json:"content_type"`
		SizeBytes   int64  `json:"size_bytes"`
	}
	if err := c.Bind(&req); err != nil {
		return errs.NewBadRequestError("Invalid request body", false, nil, nil, nil)
	}

	upload, err := h.resourceSvc.RequestUploadURL(
		c.Request().Context(),
		tc,
		req.Filename,
		req.ContentType,
		req.SizeBytes,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidFilename),
			errors.Is(err, service.ErrUnsupportedMimeType),
			errors.Is(err, service.ErrFileTooLarge):
			return errs.NewBadRequestError(err.Error(), false, nil, nil, nil)
		default:
			if h.server != nil && h.server.Logger != nil {
				h.server.Logger.Error().Err(err).Msg("failed to request upload URL")
			}
			return errs.NewInternalServerError()
		}
	}

	return c.JSON(http.StatusCreated, upload)
}

func (h *ResourceHandler) CompleteUpload(c echo.Context) error {
	tc, ok := middleware.GetTenantContext(c)
	if !ok {
		return errs.NewForbiddenError("Tenant context required", false)
	}

	resourceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errs.NewBadRequestError("Invalid resource ID", false, nil, nil, nil)
	}

	res, err := h.resourceSvc.CompleteUpload(c.Request().Context(), tc, resourceID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrResourceNotFound):
			return errs.NewNotFoundError("Resource not found", false, nil)
		case errors.Is(err, service.ErrUploadObjectMissing),
			errors.Is(err, service.ErrFileTooLarge),
			errors.Is(err, service.ErrInvalidFileContent),
			errors.Is(err, service.ErrUnsupportedMimeType):
			return errs.NewBadRequestError(err.Error(), false, nil, nil, nil)
		default:
			if h.server != nil && h.server.Logger != nil {
				h.server.Logger.Error().Err(err).Msg("failed to complete resource upload")
			}
			return errs.NewInternalServerError()
		}
	}

	return c.JSON(http.StatusOK, map[string]any{"resource": res})
}

func (h *ResourceHandler) GetDownloadURL(c echo.Context) error {
	tc, ok := middleware.GetTenantContext(c)
	if !ok {
		return errs.NewForbiddenError("Tenant context required", false)
	}

	resourceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errs.NewBadRequestError("Invalid resource ID", false, nil, nil, nil)
	}

	download, err := h.resourceSvc.GetDownloadURL(c.Request().Context(), tc, resourceID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrResourceNotFound):
			return errs.NewNotFoundError("Resource not found", false, nil)
		case errors.Is(err, service.ErrResourceNotReady):
			return errs.NewBadRequestError("Resource is not ready for download", false, nil, nil, nil)
		default:
			if h.server != nil && h.server.Logger != nil {
				h.server.Logger.Error().Err(err).Msg("failed to sign download URL")
			}
			return errs.NewInternalServerError()
		}
	}

	return c.JSON(http.StatusOK, download)
}

func (h *ResourceHandler) Delete(c echo.Context) error {
	tc, ok := middleware.GetTenantContext(c)
	if !ok {
		return errs.NewForbiddenError("Tenant context required", false)
	}

	resourceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errs.NewBadRequestError("Invalid resource ID", false, nil, nil, nil)
	}

	if err := h.resourceSvc.DeleteResource(c.Request().Context(), tc, resourceID); err != nil {
		switch {
		case errors.Is(err, repository.ErrResourceNotFound):
			return errs.NewNotFoundError("Resource not found", false, nil)
		case errors.Is(err, service.ErrDeleteForbidden):
			return errs.NewForbiddenError(err.Error(), false)
		default:
			if h.server != nil && h.server.Logger != nil {
				h.server.Logger.Error().Err(err).Msg("failed to delete resource")
			}
			return errs.NewInternalServerError()
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}
