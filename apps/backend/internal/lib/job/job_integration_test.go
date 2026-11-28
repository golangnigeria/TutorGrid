package job_test

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/tutorgrid/tutorgrid/internal/config"
	"github.com/tutorgrid/tutorgrid/internal/lib/job"
)

type stubResourceHandler struct {
	deletedCount atomic.Int32
	lastTenantID atomic.Value
	lastResID    atomic.Value
}

func (s *stubResourceHandler) ReconcileDeletingResource(_ context.Context, tenantID, resourceID uuid.UUID) error {
	s.lastTenantID.Store(tenantID)
	s.lastResID.Store(resourceID)
	s.deletedCount.Add(1)
	return nil
}

func (s *stubResourceHandler) CleanupAbandonedPending(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

func (s *stubResourceHandler) RunReconciliationSweep(_ context.Context) error {
	return nil
}

func TestNATSJetStreamPublishAndConsume(t *testing.T) {
	natsURL := os.Getenv("TUTORGRID_TEST_NATS_URL")
	if natsURL == "" {
		natsURL = os.Getenv("TUTORGRID_NATS.URL")
	}
	if natsURL == "" {
		t.Skip("set TUTORGRID_TEST_NATS_URL to run NATS JetStream integration tests")
	}

	log := zerolog.New(zerolog.NewTestWriter(t)).Level(zerolog.WarnLevel)
	cfg := &config.Config{
		NATS: config.NATSConfig{URL: natsURL},
	}

	svc, err := job.NewJobService(&log, cfg)
	if err != nil {
		t.Fatalf("NewJobService: %v", err)
	}
	defer svc.Stop()

	handler := &stubResourceHandler{}
	svc.SetResourceJobHandler(handler)

	if err := svc.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	tenantID := uuid.New()
	resourceID := uuid.New()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := svc.Publish(ctx, job.SubjectResourceDelete, tenantID, resourceID.String(), job.ResourceJobPayload{
		TenantID:   tenantID,
		ResourceID: resourceID,
	}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if handler.deletedCount.Load() >= 1 {
			if got, _ := handler.lastTenantID.Load().(uuid.UUID); got != tenantID {
				t.Fatalf("expected tenantID %s, got %s", tenantID, got)
			}
			if got, _ := handler.lastResID.Load().(uuid.UUID); got != resourceID {
				t.Fatalf("expected resourceID %s, got %s", resourceID, got)
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for NATS JetStream consumer to process resource delete job")
}
