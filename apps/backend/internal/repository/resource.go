package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tutorgrid/tutorgrid/internal/database"
	"github.com/tutorgrid/tutorgrid/internal/model"
)

var (
	ErrResourceNotFound = errors.New("resource: not found")
)

type ResourceRepository struct {
	pool *pgxpool.Pool
}

func NewResourceRepository(pool *pgxpool.Pool) *ResourceRepository {
	return &ResourceRepository{pool: pool}
}

func (r *ResourceRepository) CreatePending(ctx context.Context, tc model.TenantContext, res model.Resource) (model.Resource, error) {
	var created model.Resource
	err := database.RunInTenantTx(ctx, r.pool, tc.TenantID, func(ctx context.Context, tx pgx.Tx) error {
		var statusStr, scanStr string
		var etagPtr *string
		err := tx.QueryRow(ctx, `
			INSERT INTO public.resources (
				id, tenant_id, uploaded_by_user_id, filename, content_type,
				size_bytes, object_key, status, malware_scan_status
			)
			VALUES (
				@id, @tenant_id, @uploaded_by_user_id, @filename, @content_type,
				@size_bytes, @object_key, 'pending', 'pending'
			)
			RETURNING id, tenant_id, uploaded_by_user_id, filename, content_type,
			          size_bytes, object_key, etag, status, malware_scan_status, created_at, updated_at
		`, pgx.StrictNamedArgs{
			"id":                  res.ID,
			"tenant_id":           tc.TenantID,
			"uploaded_by_user_id": tc.UserID,
			"filename":            res.Filename,
			"content_type":        res.ContentType,
			"size_bytes":          res.SizeBytes,
			"object_key":          res.ObjectKey,
		}).Scan(
			&created.ID,
			&created.TenantID,
			&created.UploadedByUserID,
			&created.Filename,
			&created.ContentType,
			&created.SizeBytes,
			&created.ObjectKey,
			&etagPtr,
			&statusStr,
			&scanStr,
			&created.CreatedAt,
			&created.UpdatedAt,
		)
		if err != nil {
			return err
		}
		if etagPtr != nil {
			created.ETag = *etagPtr
		}
		created.Status = model.ResourceStatus(statusStr)
		created.MalwareScanStatus = model.MalwareScanStatus(scanStr)
		return nil
	})
	if err != nil {
		return model.Resource{}, fmt.Errorf("repository: create pending resource: %w", err)
	}
	return created, nil
}

func (r *ResourceRepository) GetByIDForTenant(ctx context.Context, tenantID, resourceID uuid.UUID) (model.Resource, error) {
	var res model.Resource
	err := database.RunInTenantTx(ctx, r.pool, tenantID, func(ctx context.Context, tx pgx.Tx) error {
		var statusStr, scanStr string
		var etagPtr *string
		err := tx.QueryRow(ctx, `
			SELECT id, tenant_id, uploaded_by_user_id, filename, content_type,
			       size_bytes, object_key, etag, status, malware_scan_status, created_at, updated_at
			FROM public.resources
			WHERE id = @id
		`, pgx.StrictNamedArgs{
			"id": resourceID,
		}).Scan(
			&res.ID,
			&res.TenantID,
			&res.UploadedByUserID,
			&res.Filename,
			&res.ContentType,
			&res.SizeBytes,
			&res.ObjectKey,
			&etagPtr,
			&statusStr,
			&scanStr,
			&res.CreatedAt,
			&res.UpdatedAt,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrResourceNotFound
			}
			return err
		}
		if etagPtr != nil {
			res.ETag = *etagPtr
		}
		res.Status = model.ResourceStatus(statusStr)
		res.MalwareScanStatus = model.MalwareScanStatus(scanStr)
		return nil
	})
	if err != nil {
		return model.Resource{}, err
	}
	return res, nil
}

func (r *ResourceRepository) MarkReady(ctx context.Context, tc model.TenantContext, resourceID uuid.UUID, etag string, actualSize int64) (model.Resource, error) {
	var res model.Resource
	err := database.RunInTenantTx(ctx, r.pool, tc.TenantID, func(ctx context.Context, tx pgx.Tx) error {
		var statusStr, scanStr string
		var etagPtr *string
		err := tx.QueryRow(ctx, `
			UPDATE public.resources
			SET status = 'ready',
			    etag = @etag,
			    size_bytes = @size_bytes,
			    updated_at = NOW()
			WHERE id = @id AND status = 'pending'
			RETURNING id, tenant_id, uploaded_by_user_id, filename, content_type,
			          size_bytes, object_key, etag, status, malware_scan_status, created_at, updated_at
		`, pgx.StrictNamedArgs{
			"id":         resourceID,
			"etag":       etag,
			"size_bytes": actualSize,
		}).Scan(
			&res.ID,
			&res.TenantID,
			&res.UploadedByUserID,
			&res.Filename,
			&res.ContentType,
			&res.SizeBytes,
			&res.ObjectKey,
			&etagPtr,
			&statusStr,
			&scanStr,
			&res.CreatedAt,
			&res.UpdatedAt,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrResourceNotFound
			}
			return err
		}
		if etagPtr != nil {
			res.ETag = *etagPtr
		}
		res.Status = model.ResourceStatus(statusStr)
		res.MalwareScanStatus = model.MalwareScanStatus(scanStr)
		return nil
	})
	if err != nil {
		return model.Resource{}, err
	}
	return res, nil
}

func (r *ResourceRepository) MarkStatus(ctx context.Context, tenantID, resourceID uuid.UUID, fromStatuses []model.ResourceStatus, nextStatus model.ResourceStatus) (bool, error) {
	from := make([]string, len(fromStatuses))
	for i, s := range fromStatuses {
		from[i] = string(s)
	}
	var affected int64
	err := database.RunInTenantTx(ctx, r.pool, tenantID, func(ctx context.Context, tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE public.resources
			SET status = @next_status,
			    updated_at = NOW()
			WHERE id = @id AND status = ANY(@from_statuses)
		`, pgx.StrictNamedArgs{
			"id":            resourceID,
			"next_status":   string(nextStatus),
			"from_statuses": from,
		})
		if err != nil {
			return err
		}
		affected = tag.RowsAffected()
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("repository: mark resource status %s: %w", nextStatus, err)
	}
	return affected > 0, nil
}

func (r *ResourceRepository) DeleteByIDWithStatus(ctx context.Context, tenantID, resourceID uuid.UUID, expectedStatus model.ResourceStatus) (bool, error) {
	var affected int64
	err := database.RunInTenantTx(ctx, r.pool, tenantID, func(ctx context.Context, tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			DELETE FROM public.resources
			WHERE id = @id AND status = @expected_status
		`, pgx.StrictNamedArgs{
			"id":              resourceID,
			"expected_status": string(expectedStatus),
		})
		if err != nil {
			return err
		}
		affected = tag.RowsAffected()
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("repository: delete resource: %w", err)
	}
	return affected > 0, nil
}

func (r *ResourceRepository) ListReady(
	ctx context.Context,
	tc model.TenantContext,
	limit, offset int,
	search, category string,
) ([]model.Resource, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	searchPattern := "%"
	if trimmed := strings.TrimSpace(search); trimmed != "" {
		searchPattern = "%" + trimmed + "%"
	}

	var mimeFilter []string
	switch strings.ToLower(strings.TrimSpace(category)) {
	case "images":
		mimeFilter = []string{"image/png", "image/jpeg", "image/webp"}
	case "documents":
		mimeFilter = []string{"application/pdf"}
	case "slides":
		mimeFilter = []string{
			"application/vnd.ms-powerpoint",
			"application/vnd.openxmlformats-officedocument.presentationml.presentation",
		}
	}

	items := make([]model.Resource, 0)
	err := database.RunInTenantTx(ctx, r.pool, tc.TenantID, func(ctx context.Context, tx pgx.Tx) error {
		var (
			rows pgx.Rows
			err  error
		)
		if len(mimeFilter) > 0 {
			rows, err = tx.Query(ctx, `
				SELECT id, tenant_id, uploaded_by_user_id, filename, content_type,
				       size_bytes, object_key, etag, status, malware_scan_status, created_at, updated_at
				FROM public.resources
				WHERE status = 'ready'
				  AND filename ILIKE @search
				  AND content_type = ANY(@mime_types)
				ORDER BY created_at DESC, id DESC
				LIMIT @limit OFFSET @offset
			`, pgx.StrictNamedArgs{
				"search":     searchPattern,
				"mime_types": mimeFilter,
				"limit":      limit,
				"offset":     offset,
			})
		} else {
			rows, err = tx.Query(ctx, `
				SELECT id, tenant_id, uploaded_by_user_id, filename, content_type,
				       size_bytes, object_key, etag, status, malware_scan_status, created_at, updated_at
				FROM public.resources
				WHERE status = 'ready'
				  AND filename ILIKE @search
				ORDER BY created_at DESC, id DESC
				LIMIT @limit OFFSET @offset
			`, pgx.StrictNamedArgs{
				"search": searchPattern,
				"limit":  limit,
				"offset": offset,
			})
		}
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var res model.Resource
			var statusStr, scanStr string
			var etagPtr *string
			if err := rows.Scan(
				&res.ID,
				&res.TenantID,
				&res.UploadedByUserID,
				&res.Filename,
				&res.ContentType,
				&res.SizeBytes,
				&res.ObjectKey,
				&etagPtr,
				&statusStr,
				&scanStr,
				&res.CreatedAt,
				&res.UpdatedAt,
			); err != nil {
				return err
			}
			if etagPtr != nil {
				res.ETag = *etagPtr
			}
			res.Status = model.ResourceStatus(statusStr)
			res.MalwareScanStatus = model.MalwareScanStatus(scanStr)
			items = append(items, res)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("repository: list ready resources: %w", err)
	}
	return items, nil
}

// ListReconcileCandidates calls the tightly constrained SECURITY DEFINER pointer function
// owned by tutorgrid_reconcile_definer (NOLOGIN NOSUPERUSER NOBYPASSRLS), returning ONLY
// (tenant_id, resource_id, status) pointers for bounded reconciliation.
func (r *ResourceRepository) ListReconcileCandidates(
	ctx context.Context,
	pendingCutoff, deletingCutoff time.Time,
	limit int,
) ([]model.ReconcileCandidate, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `
		SELECT tenant_id, resource_id, status
		FROM public.resources_list_reconcile_candidates(@pending_cutoff, @deleting_cutoff, @limit)
	`, pgx.StrictNamedArgs{
		"pending_cutoff":  pendingCutoff,
		"deleting_cutoff": deletingCutoff,
		"limit":           limit,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: list reconcile candidates: %w", err)
	}
	defer rows.Close()

	candidates := make([]model.ReconcileCandidate, 0)
	for rows.Next() {
		var c model.ReconcileCandidate
		var statusStr string
		if err := rows.Scan(&c.TenantID, &c.ResourceID, &statusStr); err != nil {
			return nil, err
		}
		c.Status = model.ResourceStatus(statusStr)
		candidates = append(candidates, c)
	}
	return candidates, rows.Err()
}
