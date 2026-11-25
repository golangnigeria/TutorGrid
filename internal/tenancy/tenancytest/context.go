package tenancytest

import (
	"testing"

	"github.com/google/uuid"

	"github.com/tutorgrid/tutorgrid/internal/tenancy"
)

// MustContext constructs a tenancy.Context strictly for test suites.
// Accepting testing.TB prevents accidental import or invocation in production code.
func MustContext(tb testing.TB, tenantID, userID uuid.UUID, role tenancy.Role) tenancy.Context {
	tb.Helper()
	tc, err := tenancy.NewVerifiedContext(tenantID, userID, role)
	if err != nil {
		tb.Fatalf("tenancytest.MustContext: %v", err)
	}
	return tc
}
