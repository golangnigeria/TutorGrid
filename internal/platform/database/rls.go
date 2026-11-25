package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

const tenantSettingKey = "app.current_tenant_id"

var ErrMissingTenantContext = errors.New("database: valid tenant context is required")

// TenantScope is satisfied by tenancy.Context without creating a package cycle
// between platform infrastructure and domain packages.
type TenantScope interface {
	TenantID() uuid.UUID
	Valid() bool
}

// RunInTenantTx executes fn inside a PostgreSQL transaction with transaction-scoped
// RLS tenant context set via set_config('app.current_tenant_id', tenant_id, true).
// Because is_local is true, PostgreSQL resets the setting automatically on commit
// or rollback, preventing tenant context leakage across pooled connections.
func RunInTenantTx(
	ctx context.Context,
	db *bun.DB,
	tc TenantScope,
	fn func(ctx context.Context, tx bun.Tx) error,
) error {
	if db == nil {
		return errors.New("database: db handle is nil")
	}
	if tc == nil || !tc.Valid() {
		return ErrMissingTenantContext
	}
	if fn == nil {
		return errors.New("database: transaction callback is nil")
	}

	return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.ExecContext(ctx, "SELECT set_config(?, ?, true)", tenantSettingKey, tc.TenantID().String()); err != nil {
			return fmt.Errorf("database: set tenant RLS context: %w", err)
		}
		return fn(ctx, tx)
	})
}
