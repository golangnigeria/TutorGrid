package job

import (
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"github.com/tutorgrid/tutorgrid/internal/config"
)

type JobService struct {
	Client      *asynq.Client
	server      *asynq.Server
	logger      *zerolog.Logger
	outboxStore InvitationOutboxStore
}

// asynqLogger routes Asynq's internal logs through TutorGrid's Zerolog instance.
type asynqLogger struct {
	log zerolog.Logger
}

func (l *asynqLogger) Debug(args ...any) { l.log.Debug().Msg(fmt.Sprint(args...)) }
func (l *asynqLogger) Info(args ...any)  { l.log.Info().Msg(fmt.Sprint(args...)) }
func (l *asynqLogger) Warn(args ...any)  { l.log.Warn().Msg(fmt.Sprint(args...)) }
func (l *asynqLogger) Error(args ...any) { l.log.Error().Msg(fmt.Sprint(args...)) }
func (l *asynqLogger) Fatal(args ...any) { l.log.Fatal().Msg(fmt.Sprint(args...)) }

func NewJobService(logger *zerolog.Logger, cfg *config.Config) *JobService {
	redisAddr := cfg.Redis.Address

	client := asynq.NewClient(asynq.RedisClientOpt{
		Addr: redisAddr,
	})

	scopedLogger := logger.With().Str("component", "asynq").Logger()

	server := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			Concurrency: 10,
			Logger:      &asynqLogger{log: scopedLogger},
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)

	return &JobService{
		Client: client,
		server: server,
		logger: logger,
	}
}

func (j *JobService) Start() error {
	mux := asynq.NewServeMux()
	mux.HandleFunc(TaskWelcome, j.handleWelcomeEmailTask)
	mux.HandleFunc(TaskTenantInvitation, j.handleTenantInvitationEmailTask)

	j.logger.Info().Str("component", "asynq").Msg("starting background job server")
	if err := j.server.Start(mux); err != nil {
		return err
	}

	return nil
}

func (j *JobService) Stop() {
	j.logger.Info().Str("component", "asynq").Msg("stopping background job server")
	j.server.Shutdown()
	j.Client.Close()
}
