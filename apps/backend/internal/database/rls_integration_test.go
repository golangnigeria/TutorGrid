package database_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/tutorgrid/tutorgrid/internal/database"
)

func TestPgxPoolRLSTenantIsolationAndConcurrency(t *testing.T) {
	migrationDSN := os.Getenv("TUTORGRID_TEST_DATABASE_MIGRATION_URL")
	appDSN := os.Getenv("TUTORGRID_TEST_DATABASE_URL")
	if migrationDSN == "" || appDSN == "" {
		t.Skip("set TUTORGRID_TEST_DATABASE_MIGRATION_URL and TUTORGRID_TEST_DATABASE_URL to run pgxpool RLS integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	migratorConn, err := pgx.Connect(ctx, migrationDSN)
	if err != nil {
		t.Fatalf("connect migrator: %v", err)
	}
	defer migratorConn.Close(ctx)

	nopLog := zerolog.Nop()
	if err := database.MigrateConn(ctx, &nopLog, migratorConn); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}

	// Open tight pgxpool (MaxConns=2) as non-superuser tutorgrid_app to force connection reuse.
	poolCfg, err := pgxpool.ParseConfig(appDSN)
	if err != nil {
		t.Fatalf("parse app DSN: %v", err)
	}
	poolCfg.MaxConns = 2
	appPool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		t.Fatalf("open app pool: %v", err)
	}
	defer appPool.Close()

	// 1. Verify app role is strictly NOSUPERUSER NOBYPASSRLS.
	var rolSuper, rolBypassRLS bool
	if err := appPool.QueryRow(ctx,
		"SELECT rolsuper, rolbypassrls FROM pg_roles WHERE rolname = current_user",
	).Scan(&rolSuper, &rolBypassRLS); err != nil {
		t.Fatalf("query app role: %v", err)
	}
	if rolSuper || rolBypassRLS {
		t.Fatalf("app role must be NOSUPERUSER NOBYPASSRLS, got rolsuper=%v rolbypassrls=%v", rolSuper, rolBypassRLS)
	}

	// 2. Seed Tenant A and Tenant B fixtures via migrator.
	tenantA := uuid.New()
	tenantB := uuid.New()
	userA := uuid.New()
	userB := uuid.New()
	membershipA := uuid.New()
	membershipB := uuid.New()

	seeds := []struct {
		sql  string
		args []any
	}{
		{"TRUNCATE tenant_memberships, users, tenants CASCADE", nil},
		{"INSERT INTO tenants (id, name, slug) VALUES ($1, $2, $3)", []any{tenantA, "Academy A", "academy-a"}},
		{"INSERT INTO tenants (id, name, slug) VALUES ($1, $2, $3)", []any{tenantB, "Academy B", "academy-b"}},
		{"INSERT INTO users (id, external_subject, display_name, email) VALUES ($1, $2, $3, $4)", []any{userA, "limen:user-a", "Teacher A", "a@example.com"}},
		{"INSERT INTO users (id, external_subject, display_name, email) VALUES ($1, $2, $3, $4)", []any{userB, "limen:user-b", "Teacher B", "b@example.com"}},
		{"INSERT INTO tenant_memberships (id, tenant_id, user_id, role) VALUES ($1, $2, $3, $4)", []any{membershipA, tenantA, userA, "teacher"}},
		{"INSERT INTO tenant_memberships (id, tenant_id, user_id, role) VALUES ($1, $2, $3, $4)", []any{membershipB, tenantB, userB, "teacher"}},
	}
	for _, s := range seeds {
		if _, err := migratorConn.Exec(ctx, s.sql, s.args...); err != nil {
			t.Fatalf("seed %q: %v", s.sql, err)
		}
	}

	// 3. Fail-closed when tenant context is unset.
	t.Run("fail-closed without tenant context", func(t *testing.T) {
		var count int
		if err := appPool.QueryRow(ctx, "SELECT COUNT(*) FROM tenant_memberships").Scan(&count); err != nil {
			t.Fatalf("count memberships: %v", err)
		}
		if count != 0 {
			t.Fatalf("expected 0 memberships without tenant context, got %d", count)
		}

		_, err := appPool.Exec(ctx,
			"INSERT INTO tenant_memberships (id, tenant_id, user_id, role) VALUES ($1, $2, $3, $4)",
			uuid.New(), tenantA, userB, "learner",
		)
		if err == nil {
			t.Fatal("expected INSERT without tenant context to fail RLS")
		}
	})

	// 4. Cross-tenant isolation (read, update, delete, insert).
	t.Run("tenant A isolated from tenant B", func(t *testing.T) {
		err := database.RunInTenantTx(ctx, appPool, tenantA, func(ctx context.Context, tx pgx.Tx) error {
			var gotID uuid.UUID
			if err := tx.QueryRow(ctx, "SELECT id FROM tenants").Scan(&gotID); err != nil {
				return err
			}
			if gotID != tenantA {
				t.Fatalf("expected tenant A, got %s", gotID)
			}

			tag, err := tx.Exec(ctx, "UPDATE tenant_memberships SET role = 'org_admin' WHERE id = $1", membershipB)
			if err != nil {
				return err
			}
			if tag.RowsAffected() != 0 {
				t.Fatalf("expected 0 rows affected updating tenant B, got %d", tag.RowsAffected())
			}

			tag, err = tx.Exec(ctx, "DELETE FROM tenant_memberships WHERE id = $1", membershipB)
			if err != nil {
				return err
			}
			if tag.RowsAffected() != 0 {
				t.Fatalf("expected 0 rows affected deleting tenant B, got %d", tag.RowsAffected())
			}
			return nil
		})
		if err != nil {
			t.Fatalf("RunInTenantTx tenant A: %v", err)
		}

		// Cross-tenant INSERT must fail RLS WITH CHECK.
		err = database.RunInTenantTx(ctx, appPool, tenantA, func(ctx context.Context, tx pgx.Tx) error {
			_, err := tx.Exec(ctx,
				"INSERT INTO tenant_memberships (id, tenant_id, user_id, role) VALUES ($1, $2, $3, $4)",
				uuid.New(), tenantB, userA, "learner",
			)
			return err
		})
		if err == nil {
			t.Fatal("expected cross-tenant INSERT into tenant B under tenant A to fail RLS")
		}
	})

	// 5. Concurrent pooled connection safety (20 workers x 25 tx = 500 interleaved tx over MaxConns=2).
	t.Run("zero cross-tenant leakage under concurrent pgxpool reuse", func(t *testing.T) {
		const workers = 20
		const iterations = 25

		var wg sync.WaitGroup
		errCh := make(chan error, workers*iterations)
		deliberateRollback := errors.New("deliberate rollback")

		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				for i := 0; i < iterations; i++ {
					expectedTenant := tenantA
					expectedMembership := membershipA
					if (workerID+i)%2 == 1 {
						expectedTenant = tenantB
						expectedMembership = membershipB
					}

					err := database.RunInTenantTx(ctx, appPool, expectedTenant, func(ctx context.Context, tx pgx.Tx) error {
						var tid uuid.UUID
						if err := tx.QueryRow(ctx, "SELECT id FROM tenants").Scan(&tid); err != nil {
							return err
						}
						if tid != expectedTenant {
							return errors.New("cross-tenant leak in tenants")
						}

						var mid uuid.UUID
						if err := tx.QueryRow(ctx, "SELECT id FROM tenant_memberships").Scan(&mid); err != nil {
							return err
						}
						if mid != expectedMembership {
							return errors.New("cross-tenant leak in tenant_memberships")
						}

						if i%5 == 0 {
							return deliberateRollback
						}
						return nil
					})
					if err != nil && !errors.Is(err, deliberateRollback) {
						errCh <- err
						return
					}
				}
			}(w)
		}

		wg.Wait()
		close(errCh)

		for err := range errCh {
			t.Fatalf("concurrent pooled RLS violation: %v", err)
		}

		for i := 0; i < 6; i++ {
			var count int
			if err := appPool.QueryRow(ctx, "SELECT COUNT(*) FROM tenant_memberships").Scan(&count); err != nil {
				t.Fatalf("post-pool check %d: %v", i, err)
			}
			if count != 0 {
				t.Fatalf("residual tenant context leaked onto pooled pgx connection: saw %d rows", count)
			}
		}
	})
}
