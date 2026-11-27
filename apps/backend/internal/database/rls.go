package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	tenantSettingKey          = "app.current_tenant_id"
	userSettingKey            = "app.current_user_id"
	userEmailSettingKey       = "app.current_user_email"
	inviteTokenHashSettingKey = "app.current_invite_token_hash"
)

var (
	ErrInvalidTenantID = errors.New("database: valid tenant UUID is required")
	ErrInvalidUserID   = errors.New("database: valid user UUID is required")
)

// RunInTenantTx executes fn inside a pgx transaction with transaction-scoped
// PostgreSQL Row-Level Security context via set_config('app.current_tenant_id', $1, true).
func RunInTenantTx(
	ctx context.Context,
	pool *pgxpool.Pool,
	tenantID uuid.UUID,
	fn func(ctx context.Context, tx pgx.Tx) error,
) error {
	if tenantID == uuid.Nil {
		return ErrInvalidTenantID
	}
	return runWithSettings(ctx, pool, map[string]string{
		tenantSettingKey: tenantID.String(),
	}, fn)
}

// RunInUserTx executes fn inside a pgx transaction with transaction-scoped
// app.current_user_id set for user-level Dual RLS lookups (memberships, active tenant, tenant creation).
func RunInUserTx(
	ctx context.Context,
	pool *pgxpool.Pool,
	userID uuid.UUID,
	fn func(ctx context.Context, tx pgx.Tx) error,
) error {
	if userID == uuid.Nil {
		return ErrInvalidUserID
	}
	return runWithSettings(ctx, pool, map[string]string{
		userSettingKey: userID.String(),
	}, fn)
}

// RunInUserAndTenantTx sets both app.current_user_id and app.current_tenant_id transaction-scoped.
func RunInUserAndTenantTx(
	ctx context.Context,
	pool *pgxpool.Pool,
	userID, tenantID uuid.UUID,
	fn func(ctx context.Context, tx pgx.Tx) error,
) error {
	if userID == uuid.Nil {
		return ErrInvalidUserID
	}
	if tenantID == uuid.Nil {
		return ErrInvalidTenantID
	}
	return runWithSettings(ctx, pool, map[string]string{
		userSettingKey:   userID.String(),
		tenantSettingKey: tenantID.String(),
	}, fn)
}

// RunInUserEmailTx sets app.current_user_id and app.current_user_email transaction-scoped
// for verified-email pending invitation discovery and acceptance under RLS.
func RunInUserEmailTx(
	ctx context.Context,
	pool *pgxpool.Pool,
	userID uuid.UUID,
	userEmail string,
	fn func(ctx context.Context, tx pgx.Tx) error,
) error {
	if userID == uuid.Nil {
		return ErrInvalidUserID
	}
	if userEmail == "" {
		return errors.New("database: verified user email is required")
	}
	return runWithSettings(ctx, pool, map[string]string{
		userSettingKey:      userID.String(),
		userEmailSettingKey: userEmail,
	}, fn)
}

// RunInInviteTx sets app.current_user_id and app.current_invite_token_hash transaction-scoped
// for atomic invitation acceptance under RLS.
func RunInInviteTx(
	ctx context.Context,
	pool *pgxpool.Pool,
	userID uuid.UUID,
	tokenHashHex string,
	fn func(ctx context.Context, tx pgx.Tx) error,
) error {
	if userID == uuid.Nil {
		return ErrInvalidUserID
	}
	if tokenHashHex == "" {
		return errors.New("database: invite token hash is required")
	}
	return runWithSettings(ctx, pool, map[string]string{
		userSettingKey:            userID.String(),
		inviteTokenHashSettingKey: tokenHashHex,
	}, fn)
}

// SetTxTenantID sets app.current_tenant_id transaction-locally inside an already-open pgx.Tx.
func SetTxTenantID(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) error {
	if tenantID == uuid.Nil {
		return ErrInvalidTenantID
	}
	_, err := tx.Exec(ctx, "SELECT set_config(@key, @value, true)", pgx.StrictNamedArgs{
		"key":   tenantSettingKey,
		"value": tenantID.String(),
	})
	return err
}

func runWithSettings(
	ctx context.Context,
	pool *pgxpool.Pool,
	settings map[string]string,
	fn func(ctx context.Context, tx pgx.Tx) error,
) error {
	if pool == nil {
		return errors.New("database: pgxpool is nil")
	}
	if fn == nil {
		return errors.New("database: transaction callback is nil")
	}

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("database: begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	for k, v := range settings {
		if _, err := tx.Exec(ctx, "SELECT set_config(@key, @value, true)", pgx.StrictNamedArgs{
			"key":   k,
			"value": v,
		}); err != nil {
			return fmt.Errorf("database: set RLS context %s: %w", k, err)
		}
	}

	if err := fn(ctx, tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("database: commit tx: %w", err)
	}
	return nil
}
