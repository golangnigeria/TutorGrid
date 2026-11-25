package tenancy_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/tutorgrid/tutorgrid/internal/tenancy"
	"github.com/tutorgrid/tutorgrid/internal/tenancy/tenancytest"
)

func TestNewVerifiedContextValidation(t *testing.T) {
	validTenant := uuid.New()
	validUser := uuid.New()

	if _, err := tenancy.NewVerifiedContext(uuid.Nil, validUser, tenancy.RoleTeacher); !errors.Is(err, tenancy.ErrInvalidTenantID) {
		t.Fatalf("expected ErrInvalidTenantID, got %v", err)
	}
	if _, err := tenancy.NewVerifiedContext(validTenant, uuid.Nil, tenancy.RoleTeacher); !errors.Is(err, tenancy.ErrInvalidUserID) {
		t.Fatalf("expected ErrInvalidUserID, got %v", err)
	}
	if _, err := tenancy.NewVerifiedContext(validTenant, validUser, tenancy.Role("invalid")); !errors.Is(err, tenancy.ErrInvalidRole) {
		t.Fatalf("expected ErrInvalidRole, got %v", err)
	}

	tc, err := tenancy.NewVerifiedContext(validTenant, validUser, tenancy.RoleTeacher)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !tc.Valid() || tc.TenantID() != validTenant || tc.UserID() != validUser || tc.Role() != tenancy.RoleTeacher {
		t.Fatalf("unexpected context values: %+v", tc)
	}
}

func TestContextRoundTrip(t *testing.T) {
	if _, ok := tenancy.FromContext(context.Background()); ok {
		t.Fatal("expected missing tenant context to return ok=false")
	}

	tc := tenancytest.MustContext(t, uuid.New(), uuid.New(), tenancy.RoleOrgAdmin)
	ctx := tenancy.WithContext(context.Background(), tc)

	got, ok := tenancy.FromContext(ctx)
	if !ok {
		t.Fatal("expected tenant context to be present")
	}
	if got.TenantID() != tc.TenantID() || got.UserID() != tc.UserID() || got.Role() != tc.Role() {
		t.Fatalf("round-trip mismatch: got %+v, want %+v", got, tc)
	}
}
