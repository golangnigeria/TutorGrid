package job

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	StreamJobs = "TUTORGRID_JOBS"

	SubjectJobsWildcard          = "tutorgrid.jobs.>"
	SubjectEmailWildcard         = "tutorgrid.jobs.email.>"
	SubjectResourcesWildcard     = "tutorgrid.jobs.resources.>"
	SubjectEmailWelcome          = "tutorgrid.jobs.email.welcome"
	SubjectEmailTenantInvitation = "tutorgrid.jobs.email.tenant_invitation"
	SubjectResourceDelete        = "tutorgrid.jobs.resources.delete"
	SubjectResourceCleanup       = "tutorgrid.jobs.resources.cleanup_pending"

	CurrentSchemaVersion = 1
)

// Envelope is the canonical TutorGrid NATS JetStream job/event envelope.
// It carries stable metadata and references only—never raw secrets or binary files.
type Envelope struct {
	EventID       string          `json:"event_id"`
	SchemaVersion int             `json:"schema_version"`
	TenantID      string          `json:"tenant_id,omitempty"`
	CorrelationID string          `json:"correlation_id,omitempty"`
	Timestamp     time.Time       `json:"timestamp"`
	Payload       json.RawMessage `json:"payload"`
}

func NewEnvelope(tenantID uuid.UUID, correlationID string, payload any) (Envelope, []byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return Envelope{}, nil, err
	}
	env := Envelope{
		EventID:       uuid.NewString(),
		SchemaVersion: CurrentSchemaVersion,
		CorrelationID: correlationID,
		Timestamp:     time.Now().UTC(),
		Payload:       raw,
	}
	if tenantID != uuid.Nil {
		env.TenantID = tenantID.String()
	}
	encoded, err := json.Marshal(env)
	if err != nil {
		return Envelope{}, nil, err
	}
	return env, encoded, nil
}
