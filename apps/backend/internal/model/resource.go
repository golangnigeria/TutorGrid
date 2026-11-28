package model

import (
	"time"

	"github.com/google/uuid"
)

type ResourceStatus string

const (
	ResourceStatusPending  ResourceStatus = "pending"
	ResourceStatusReady    ResourceStatus = "ready"
	ResourceStatusFailed   ResourceStatus = "failed"
	ResourceStatusDeleting ResourceStatus = "deleting"
)

type MalwareScanStatus string

const (
	MalwareScanPending  MalwareScanStatus = "pending"
	MalwareScanSkipped  MalwareScanStatus = "skipped"
	MalwareScanClean    MalwareScanStatus = "clean"
	MalwareScanInfected MalwareScanStatus = "infected"
)

type Resource struct {
	ID                uuid.UUID         `json:"id"`
	TenantID          uuid.UUID         `json:"tenant_id"`
	UploadedByUserID  uuid.UUID         `json:"uploaded_by_user_id"`
	Filename          string            `json:"filename"`
	ContentType       string            `json:"content_type"`
	SizeBytes         int64             `json:"size_bytes"`
	ObjectKey         string            `json:"-"`
	ETag              string            `json:"etag,omitempty"`
	Status            ResourceStatus    `json:"status"`
	MalwareScanStatus MalwareScanStatus `json:"malware_scan_status"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

type PresignedUpload struct {
	Resource  Resource  `json:"resource"`
	UploadURL string    `json:"upload_url"`
	ExpiresAt time.Time `json:"expires_at"`
}

type PresignedDownload struct {
	ResourceID  uuid.UUID `json:"resource_id"`
	Filename    string    `json:"filename"`
	DownloadURL string    `json:"download_url"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// ReconcileCandidate is the minimal pointer returned by the constrained
// SECURITY DEFINER discovery function (never exposes filenames, keys, or user data).
type ReconcileCandidate struct {
	TenantID   uuid.UUID
	ResourceID uuid.UUID
	Status     ResourceStatus
}

// ReconcileStats captures bounded periodic sweep metrics.
type ReconcileStats struct {
	Discovered int `json:"discovered"`
	Repaired   int `json:"repaired"`
	Retried    int `json:"retried"`
	Failed     int `json:"failed"`
}
