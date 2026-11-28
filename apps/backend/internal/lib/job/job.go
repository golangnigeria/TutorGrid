package job

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"

	"github.com/tutorgrid/tutorgrid/internal/config"
	"github.com/tutorgrid/tutorgrid/internal/lib/email"
)

// JobService is TutorGrid's single durable background job & event backbone powered by NATS JetStream (ADR 0006).
type JobService struct {
	nc          *nats.Conn
	js          jetstream.JetStream
	logger      *zerolog.Logger
	emailClient *email.Client

	mu                 sync.RWMutex
	outboxStore        InvitationOutboxStore
	resourceJobHandler ResourceJobHandler

	consumeContexts []jetstream.ConsumeContext
	stopCh          chan struct{}
	stopOnce        sync.Once
	wg              sync.WaitGroup
}

// ResourceJobHandler executes idempotent, tenant-scoped resource deletion and pending-upload cleanup.
type ResourceJobHandler interface {
	ReconcileDeletingResource(ctx context.Context, tenantID, resourceID uuid.UUID) error
	CleanupAbandonedPending(ctx context.Context, tenantID, resourceID uuid.UUID) error
	RunReconciliationSweep(ctx context.Context) error
}

func NewJobService(logger *zerolog.Logger, cfg *config.Config) (*JobService, error) {
	natsURL := "nats://localhost:4222"
	if cfg != nil && cfg.NATS.URL != "" {
		natsURL = cfg.NATS.URL
	}

	scoped := logger.With().Str("component", "nats-jetstream").Logger()

	nc, err := nats.Connect(
		natsURL,
		nats.Name("tutorgrid-api"),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				scoped.Warn().Err(err).Msg("disconnected from NATS")
			}
		}),
		nats.ReconnectHandler(func(conn *nats.Conn) {
			scoped.Info().Str("url", conn.ConnectedUrl()).Msg("reconnected to NATS")
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("nats: connect %s: %w", natsURL, err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("nats: init jetstream: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        StreamJobs,
		Description: "TutorGrid durable background jobs and domain events",
		Subjects:    []string{SubjectJobsWildcard},
		Storage:     jetstream.FileStorage,
		Retention:   jetstream.LimitsPolicy,
		MaxAge:      7 * 24 * time.Hour,
		Duplicates:  5 * time.Minute,
	}); err != nil {
		nc.Close()
		return nil, fmt.Errorf("nats: ensure stream %s: %w", StreamJobs, err)
	}

	return &JobService{
		nc:     nc,
		js:     js,
		logger: logger,
		stopCh: make(chan struct{}),
	}, nil
}

func (j *JobService) InitHandlers(cfg *config.Config, logger *zerolog.Logger) {
	if cfg != nil && cfg.Integration.ResendAPIKey != "" {
		j.emailClient = email.NewClient(cfg, logger)
	}
}

func (j *JobService) SetInvitationOutboxStore(store InvitationOutboxStore) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.outboxStore = store
}

func (j *JobService) SetResourceJobHandler(handler ResourceJobHandler) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.resourceJobHandler = handler
}

// Publish persists a versioned event envelope onto the NATS JetStream TUTORGRID_JOBS stream.
func (j *JobService) Publish(ctx context.Context, subject string, tenantID uuid.UUID, correlationID string, payload any) error {
	if j == nil || j.js == nil {
		return errors.New("nats jetstream is not initialized")
	}
	env, data, err := NewEnvelope(tenantID, correlationID, payload)
	if err != nil {
		return fmt.Errorf("nats: marshal envelope: %w", err)
	}
	if _, err := j.js.Publish(ctx, subject, data, jetstream.WithMsgID(env.EventID)); err != nil {
		return fmt.Errorf("nats: publish %s: %w", subject, err)
	}
	return nil
}

// EnqueueInvitationEmail publishes a tenant invitation delivery job (IDs only, never raw tokens).
func (j *JobService) EnqueueInvitationEmail(ctx context.Context, tenantID, outboxID uuid.UUID) error {
	return j.Publish(ctx, SubjectEmailTenantInvitation, tenantID, outboxID.String(), TenantInvitationPayload{
		TenantID: tenantID,
		OutboxID: outboxID,
	})
}

// Start registers durable JetStream pull consumers and starts the bounded periodic resource reconciler.
func (j *JobService) Start() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	emailConsumer, err := j.js.CreateOrUpdateConsumer(ctx, StreamJobs, jetstream.ConsumerConfig{
		Durable:       "tutorgrid_email_worker",
		FilterSubject: SubjectEmailWildcard,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       30 * time.Second,
		MaxDeliver:    5,
		BackOff:       []time.Duration{2 * time.Second, 10 * time.Second, 30 * time.Second, 2 * time.Minute},
	})
	if err != nil {
		return fmt.Errorf("nats: create email consumer: %w", err)
	}

	emailCC, err := emailConsumer.Consume(func(msg jetstream.Msg) {
		j.dispatchEmailMessage(msg)
	})
	if err != nil {
		return fmt.Errorf("nats: start email consumer: %w", err)
	}
	j.consumeContexts = append(j.consumeContexts, emailCC)

	resourceConsumer, err := j.js.CreateOrUpdateConsumer(ctx, StreamJobs, jetstream.ConsumerConfig{
		Durable:       "tutorgrid_resource_worker",
		FilterSubject: SubjectResourcesWildcard,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       60 * time.Second,
		MaxDeliver:    5,
		BackOff:       []time.Duration{5 * time.Second, 20 * time.Second, 1 * time.Minute, 5 * time.Minute},
	})
	if err != nil {
		return fmt.Errorf("nats: create resource consumer: %w", err)
	}

	resourceCC, err := resourceConsumer.Consume(func(msg jetstream.Msg) {
		j.dispatchResourceMessage(msg)
	})
	if err != nil {
		return fmt.Errorf("nats: start resource consumer: %w", err)
	}
	j.consumeContexts = append(j.consumeContexts, resourceCC)

	// Bounded periodic reconciler (every 5 minutes) to recover any DB-commit/NATS-publish gaps.
	j.wg.Add(1)
	go j.runReconciliationLoop()

	j.logger.Info().
		Str("component", "nats-jetstream").
		Str("stream", StreamJobs).
		Msg("started NATS JetStream durable job consumers")
	return nil
}

func (j *JobService) runReconciliationLoop() {
	defer j.wg.Done()
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-j.stopCh:
			return
		case <-ticker.C:
			j.mu.RLock()
			handler := j.resourceJobHandler
			j.mu.RUnlock()
			if handler == nil {
				continue
			}
			sweepCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			if err := handler.RunReconciliationSweep(sweepCtx); err != nil {
				j.logger.Error().
					Str("component", "nats-jetstream").
					Err(err).
					Msg("periodic resource reconciliation sweep failed")
			}
			cancel()
		}
	}
}

func (j *JobService) dispatchEmailMessage(msg jetstream.Msg) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	var env Envelope
	if err := json.Unmarshal(msg.Data(), &env); err != nil {
		j.logger.Error().Err(err).Str("subject", msg.Subject()).Msg("malformed job envelope; terminating message")
		_ = msg.Term()
		return
	}

	var err error
	switch msg.Subject() {
	case SubjectEmailWelcome:
		err = j.handleWelcomeEmail(ctx, env)
	case SubjectEmailTenantInvitation:
		err = j.handleTenantInvitationEmail(ctx, env)
	default:
		j.logger.Warn().Str("subject", msg.Subject()).Msg("unknown email job subject; acknowledging")
		_ = msg.Ack()
		return
	}

	if err != nil {
		_ = msg.Nak()
		return
	}
	_ = msg.Ack()
}

func (j *JobService) dispatchResourceMessage(msg jetstream.Msg) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()

	var env Envelope
	if err := json.Unmarshal(msg.Data(), &env); err != nil {
		j.logger.Error().Err(err).Str("subject", msg.Subject()).Msg("malformed resource job envelope; terminating message")
		_ = msg.Term()
		return
	}

	err := j.handleResourceMessage(ctx, msg.Subject(), env)
	if err != nil {
		_ = msg.Nak()
		return
	}
	_ = msg.Ack()
}

func (j *JobService) Stop() {
	j.stopOnce.Do(func() {
		j.logger.Info().Str("component", "nats-jetstream").Msg("stopping NATS JetStream consumers")
		close(j.stopCh)
		for _, cc := range j.consumeContexts {
			cc.Stop()
		}
		j.wg.Wait()
		if j.nc != nil {
			_ = j.nc.Drain()
			j.nc.Close()
		}
	})
}
