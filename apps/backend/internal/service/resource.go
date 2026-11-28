package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/tutorgrid/tutorgrid/internal/lib/job"
	"github.com/tutorgrid/tutorgrid/internal/lib/storage"
	"github.com/tutorgrid/tutorgrid/internal/model"
	"github.com/tutorgrid/tutorgrid/internal/repository"
)

const (
	presignedURLTTL          = 5 * time.Minute
	defaultPendingStaleAge   = 20 * time.Minute
	defaultDeletingStaleAge  = 2 * time.Minute
	defaultReconcileBatchMax = 50
)

var (
	ErrUploadObjectMissing = errors.New("resource: uploaded file not found in object storage")
	ErrResourceNotReady    = errors.New("resource: resource is not ready for download")
	ErrDeleteForbidden     = errors.New("resource: only the uploader or an organization admin can delete this resource")
)

// MalwareScanEnqueuer defines the async scanning extension point.
// In Slice 3, malware scanning is not yet active; resources stay malware_scan_status='pending'
// and a real NATS JetStream scanning worker must be wired before learner-facing delivery (ADR 0003).
type MalwareScanEnqueuer interface {
	EnqueueScan(ctx context.Context, tenantID, resourceID uuid.UUID, objectKey string) error
}

// NoopScanEnqueuer logs that scanning is deferred without ever marking a file clean.
type NoopScanEnqueuer struct {
	logger *zerolog.Logger
}

func (n *NoopScanEnqueuer) EnqueueScan(_ context.Context, tenantID, resourceID uuid.UUID, _ string) error {
	if n.logger != nil {
		n.logger.Info().
			Str("tenant_id", tenantID.String()).
			Str("resource_id", resourceID.String()).
			Str("malware_scan_status", string(model.MalwareScanPending)).
			Msg("malware scanning deferred in Slice 3; resource remains unscanned (pending)")
	}
	return nil
}

// ResourceJobPublisher publishes durable NATS JetStream jobs for resource deletion/cleanup.
type ResourceJobPublisher interface {
	Publish(ctx context.Context, subject string, tenantID uuid.UUID, correlationID string, payload any) error
}

type ResourceService struct {
	repo           *repository.ResourceRepository
	store          storage.ObjectStore
	publisher      ResourceJobPublisher
	scanner        MalwareScanEnqueuer
	logger         *zerolog.Logger
	pendingMaxAge  time.Duration
	deletingMaxAge time.Duration
}

func NewResourceService(
	repo *repository.ResourceRepository,
	store storage.ObjectStore,
	publisher ResourceJobPublisher,
	logger *zerolog.Logger,
) *ResourceService {
	return &ResourceService{
		repo:           repo,
		store:          store,
		publisher:      publisher,
		scanner:        &NoopScanEnqueuer{logger: logger},
		logger:         logger,
		pendingMaxAge:  defaultPendingStaleAge,
		deletingMaxAge: defaultDeletingStaleAge,
	}
}

func (s *ResourceService) SetMalwareScanEnqueuer(scanner MalwareScanEnqueuer) {
	if scanner != nil {
		s.scanner = scanner
	}
}

func (s *ResourceService) RequestUploadURL(
	ctx context.Context,
	tc model.TenantContext,
	filename, contentType string,
	declaredSize int64,
) (model.PresignedUpload, error) {
	if s.store == nil {
		return model.PresignedUpload{}, errors.New("resource: object store is not configured")
	}
	safeName, err := SanitizeFilename(filename)
	if err != nil {
		return model.PresignedUpload{}, err
	}
	mime, err := NormalizeAndValidateMimeType(contentType)
	if err != nil {
		return model.PresignedUpload{}, err
	}
	if err := ValidateDeclaredSize(declaredSize); err != nil {
		return model.PresignedUpload{}, err
	}

	resourceID := uuid.New()
	objectKey := BuildObjectKey(tc.TenantID, resourceID, safeName)

	created, err := s.repo.CreatePending(ctx, tc, model.Resource{
		ID:          resourceID,
		Filename:    safeName,
		ContentType: mime,
		SizeBytes:   declaredSize,
		ObjectKey:   objectKey,
	})
	if err != nil {
		return model.PresignedUpload{}, err
	}

	expiresAt := time.Now().UTC().Add(presignedURLTTL)
	uploadURL, err := s.store.PresignPut(ctx, objectKey, mime, declaredSize, presignedURLTTL)
	if err != nil {
		// Mark failed so the pending row does not linger if signing failed.
		_, _ = s.repo.MarkStatus(ctx, tc.TenantID, created.ID, []model.ResourceStatus{model.ResourceStatusPending}, model.ResourceStatusFailed)
		return model.PresignedUpload{}, err
	}

	return model.PresignedUpload{
		Resource:  created,
		UploadURL: uploadURL,
		ExpiresAt: expiresAt,
	}, nil
}

// CompleteUpload verifies the actual Backblaze B2 object via HeadObject (treating browser-declared
// size as untrusted), validates leading magic bytes and OOXML PPTX structure, and only transitions
// the resource to 'ready' once all checks pass. Safe and idempotent on retries.
func (s *ResourceService) CompleteUpload(
	ctx context.Context,
	tc model.TenantContext,
	resourceID uuid.UUID,
) (model.Resource, error) {
	if s.store == nil {
		return model.Resource{}, errors.New("resource: object store is not configured")
	}

	res, err := s.repo.GetByIDForTenant(ctx, tc.TenantID, resourceID)
	if err != nil {
		return model.Resource{}, err
	}

	switch res.Status {
	case model.ResourceStatusReady:
		return res, nil
	case model.ResourceStatusDeleting:
		return model.Resource{}, repository.ErrResourceNotFound
	case model.ResourceStatusFailed:
		return model.Resource{}, ErrInvalidFileContent
	}

	info, err := s.store.HeadObject(ctx, res.ObjectKey)
	if err != nil {
		return model.Resource{}, fmt.Errorf("resource: verify object in B2: %w", err)
	}
	if !info.Exists {
		return model.Resource{}, ErrUploadObjectMissing
	}

	// Untrusted client size check: enforce actual Backblaze B2 object size strictly <= 25 MiB.
	if info.SizeBytes <= 0 || info.SizeBytes > MaxResourceSizeBytes {
		s.rejectInvalidUpload(ctx, tc.TenantID, res)
		return model.Resource{}, ErrFileTooLarge
	}

	headerEnd := int64(511)
	if info.SizeBytes-1 < headerEnd {
		headerEnd = info.SizeBytes - 1
	}
	headerBytes, err := s.store.ReadRange(ctx, res.ObjectKey, 0, headerEnd)
	if err != nil {
		return model.Resource{}, fmt.Errorf("resource: read object header: %w", err)
	}

	if err := ValidateMagicBytes(res.ContentType, headerBytes); err != nil {
		s.rejectInvalidUpload(ctx, tc.TenantID, res)
		return model.Resource{}, ErrInvalidFileContent
	}

	if res.ContentType == MimePPTX {
		if err := ValidatePPTXPackage(ctx, s.store, res.ObjectKey, info.SizeBytes); err != nil {
			s.rejectInvalidUpload(ctx, tc.TenantID, res)
			return model.Resource{}, ErrInvalidFileContent
		}
	}

	readyRes, err := s.repo.MarkReady(ctx, tc, res.ID, info.ETag, info.SizeBytes)
	if err != nil {
		if errors.Is(err, repository.ErrResourceNotFound) {
			// Handle concurrent completion race idempotently.
			current, getErr := s.repo.GetByIDForTenant(ctx, tc.TenantID, resourceID)
			if getErr == nil && current.Status == model.ResourceStatusReady {
				return current, nil
			}
		}
		return model.Resource{}, err
	}

	if s.scanner != nil {
		_ = s.scanner.EnqueueScan(ctx, readyRes.TenantID, readyRes.ID, readyRes.ObjectKey)
	}

	return readyRes, nil
}

// rejectInvalidUpload deletes a spoofed or oversized object from Backblaze B2;
// if the B2 delete fails transiently, transitions status='deleting' so JetStream
// and the bounded periodic reconciler guarantee zero orphaned B2 objects.
func (s *ResourceService) rejectInvalidUpload(ctx context.Context, tenantID uuid.UUID, res model.Resource) {
	if err := s.store.DeleteObject(ctx, res.ObjectKey); err != nil {
		if s.logger != nil {
			s.logger.Warn().
				Str("tenant_id", tenantID.String()).
				Str("resource_id", res.ID.String()).
				Err(err).
				Msg("failed to immediately delete invalid upload from B2; scheduling durable deletion")
		}
		_, _ = s.repo.MarkStatus(ctx, tenantID, res.ID, []model.ResourceStatus{model.ResourceStatusPending}, model.ResourceStatusDeleting)
		s.publishDeleteJob(ctx, tenantID, res.ID)
		return
	}
	_, _ = s.repo.MarkStatus(ctx, tenantID, res.ID, []model.ResourceStatus{model.ResourceStatusPending}, model.ResourceStatusFailed)
}

func (s *ResourceService) ListResources(
	ctx context.Context,
	tc model.TenantContext,
	limit, offset int,
	search, category string,
) ([]model.Resource, error) {
	return s.repo.ListReady(ctx, tc, limit, offset, search, category)
}

func (s *ResourceService) GetDownloadURL(
	ctx context.Context,
	tc model.TenantContext,
	resourceID uuid.UUID,
) (model.PresignedDownload, error) {
	if s.store == nil {
		return model.PresignedDownload{}, errors.New("resource: object store is not configured")
	}

	res, err := s.repo.GetByIDForTenant(ctx, tc.TenantID, resourceID)
	if err != nil {
		return model.PresignedDownload{}, err
	}
	if res.Status != model.ResourceStatusReady {
		return model.PresignedDownload{}, ErrResourceNotReady
	}

	expiresAt := time.Now().UTC().Add(presignedURLTTL)
	downloadURL, err := s.store.PresignGet(ctx, res.ObjectKey, res.Filename, presignedURLTTL)
	if err != nil {
		return model.PresignedDownload{}, err
	}

	return model.PresignedDownload{
		ResourceID:  res.ID,
		Filename:    res.Filename,
		DownloadURL: downloadURL,
		ExpiresAt:   expiresAt,
	}, nil
}

// DeleteResource enforces uploader-or-admin RBAC, durably marks status='deleting' under RLS
// (immediately hiding the resource), and completes B2 + DB cleanup idempotently with NATS
// JetStream and periodic reconciler recovery on any partial failure.
func (s *ResourceService) DeleteResource(
	ctx context.Context,
	tc model.TenantContext,
	resourceID uuid.UUID,
) error {
	if s.store == nil {
		return errors.New("resource: object store is not configured")
	}

	res, err := s.repo.GetByIDForTenant(ctx, tc.TenantID, resourceID)
	if err != nil {
		return err
	}
	if res.Status == model.ResourceStatusDeleting {
		return repository.ErrResourceNotFound
	}

	isAdmin := tc.Role == model.RoleOrgAdmin || tc.Role == model.RolePlatformAdmin
	if res.UploadedByUserID != tc.UserID && !isAdmin {
		return ErrDeleteForbidden
	}

	updated, err := s.repo.MarkStatus(
		ctx,
		tc.TenantID,
		res.ID,
		[]model.ResourceStatus{model.ResourceStatusPending, model.ResourceStatusReady, model.ResourceStatusFailed},
		model.ResourceStatusDeleting,
	)
	if err != nil {
		return err
	}
	if !updated {
		return repository.ErrResourceNotFound
	}

	// Attempt immediate cleanup on the happy path; on partial failure, publish to NATS JetStream
	// (and even if NATS publish fails, status='deleting' stays durable for the periodic reconciler).
	if err := s.ReconcileDeletingResource(ctx, tc.TenantID, res.ID); err != nil {
		if s.logger != nil {
			s.logger.Warn().
				Str("tenant_id", tc.TenantID.String()).
				Str("resource_id", res.ID.String()).
				Err(err).
				Msg("resource marked deleting; deferring storage/DB cleanup to NATS JetStream & reconciler")
		}
		s.publishDeleteJob(ctx, tc.TenantID, res.ID)
	}

	return nil
}

func (s *ResourceService) publishDeleteJob(ctx context.Context, tenantID, resourceID uuid.UUID) {
	if s.publisher == nil {
		return
	}
	if err := s.publisher.Publish(ctx, job.SubjectResourceDelete, tenantID, resourceID.String(), job.ResourceJobPayload{
		TenantID:   tenantID,
		ResourceID: resourceID,
	}); err != nil && s.logger != nil {
		s.logger.Warn().
			Str("tenant_id", tenantID.String()).
			Str("resource_id", resourceID.String()).
			Err(err).
			Msg("failed to publish resource delete job to NATS; periodic reconciler will recover")
	}
}

// ReconcileDeletingResource is fully idempotent:
// 1. Re-reads the resource under RunInTenantTx and verifies status='deleting'.
// 2. Deletes from Backblaze B2 (treating NotFound as success).
// 3. Deletes the PostgreSQL row WHERE status='deleting' under RunInTenantTx.
func (s *ResourceService) ReconcileDeletingResource(ctx context.Context, tenantID, resourceID uuid.UUID) error {
	if s.store == nil {
		return errors.New("resource: object store is not configured")
	}
	res, err := s.repo.GetByIDForTenant(ctx, tenantID, resourceID)
	if err != nil {
		if errors.Is(err, repository.ErrResourceNotFound) {
			return nil
		}
		return err
	}
	if res.Status != model.ResourceStatusDeleting {
		return nil
	}

	if err := s.store.DeleteObject(ctx, res.ObjectKey); err != nil {
		return fmt.Errorf("reconcile deleting resource: b2 delete: %w", err)
	}

	if _, err := s.repo.DeleteByIDWithStatus(ctx, tenantID, resourceID, model.ResourceStatusDeleting); err != nil {
		return fmt.Errorf("reconcile deleting resource: db delete: %w", err)
	}
	return nil
}

// CleanupAbandonedPending verifies the resource is still 'pending' and older than cutoff
// inside RunInTenantTx before idempotently removing any partial B2 object and DB row.
func (s *ResourceService) CleanupAbandonedPending(ctx context.Context, tenantID, resourceID uuid.UUID) error {
	return s.cleanupAbandonedPendingWithCutoff(ctx, tenantID, resourceID, time.Now().UTC().Add(-s.pendingMaxAge))
}

func (s *ResourceService) cleanupAbandonedPendingWithCutoff(
	ctx context.Context,
	tenantID, resourceID uuid.UUID,
	cutoff time.Time,
) error {
	if s.store == nil {
		return errors.New("resource: object store is not configured")
	}
	res, err := s.repo.GetByIDForTenant(ctx, tenantID, resourceID)
	if err != nil {
		if errors.Is(err, repository.ErrResourceNotFound) {
			return nil
		}
		return err
	}
	// Never touch ready, failed, or non-stale pending rows.
	if res.Status != model.ResourceStatusPending || res.CreatedAt.After(cutoff) {
		return nil
	}

	if err := s.store.DeleteObject(ctx, res.ObjectKey); err != nil {
		return fmt.Errorf("cleanup pending resource: b2 delete: %w", err)
	}
	if _, err := s.repo.DeleteByIDWithStatus(ctx, tenantID, resourceID, model.ResourceStatusPending); err != nil {
		return fmt.Errorf("cleanup pending resource: db delete: %w", err)
	}
	return nil
}

// RunReconciliationSweep implements job.ResourceJobHandler for bounded periodic recovery.
func (s *ResourceService) RunReconciliationSweep(ctx context.Context) error {
	_, err := s.RunReconciliationSweepWithStats(ctx, s.pendingMaxAge, s.deletingMaxAge, defaultReconcileBatchMax)
	return err
}

// RunReconciliationSweepWithStats discovers bounded stale pending/deleting pointers via the
// constrained SECURITY DEFINER function and repairs each one strictly inside RunInTenantTx.
func (s *ResourceService) RunReconciliationSweepWithStats(
	ctx context.Context,
	pendingMaxAge, deletingMaxAge time.Duration,
	batchLimit int,
) (model.ReconcileStats, error) {
	now := time.Now().UTC()
	pendingCutoff := now.Add(-pendingMaxAge)
	deletingCutoff := now.Add(-deletingMaxAge)

	candidates, err := s.repo.ListReconcileCandidates(ctx, pendingCutoff, deletingCutoff, batchLimit)
	if err != nil {
		return model.ReconcileStats{}, err
	}

	stats := model.ReconcileStats{
		Discovered: len(candidates),
	}

	for _, c := range candidates {
		var recErr error
		switch c.Status {
		case model.ResourceStatusPending:
			recErr = s.cleanupAbandonedPendingWithCutoff(ctx, c.TenantID, c.ResourceID, pendingCutoff)
		case model.ResourceStatusDeleting:
			recErr = s.ReconcileDeletingResource(ctx, c.TenantID, c.ResourceID)
		default:
			continue
		}

		if recErr != nil {
			stats.Failed++
			if s.publisher != nil && c.Status == model.ResourceStatusDeleting {
				s.publishDeleteJob(ctx, c.TenantID, c.ResourceID)
				stats.Retried++
			}
			if s.logger != nil {
				s.logger.Warn().
					Str("component", "resource-reconciler").
					Str("tenant_id", c.TenantID.String()).
					Str("resource_id", c.ResourceID.String()).
					Str("status", string(c.Status)).
					Err(recErr).
					Msg("resource reconciliation item failed; will retry on next sweep")
			}
			continue
		}
		stats.Repaired++
	}

	if s.logger != nil && stats.Discovered > 0 {
		s.logger.Info().
			Str("component", "resource-reconciler").
			Int("discovered", stats.Discovered).
			Int("repaired", stats.Repaired).
			Int("retried", stats.Retried).
			Int("failed", stats.Failed).
			Msg("completed bounded resource reconciliation sweep")
	}
	return stats, nil
}
