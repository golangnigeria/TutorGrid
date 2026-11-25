package database_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/tutorgrid/tutorgrid/internal/platform/database"
	"github.com/tutorgrid/tutorgrid/internal/tenancy"
	"github.com/tutorgrid/tutorgrid/internal/tenancy/tenancytest"
)

func TestPostgreSQLRLSTenantIsolationAndPoolSafety(t *testing.T) {
	migrationDSN := os.Getenv("TEST_DATABASE_MIGRATION_URL")
	appDSN := os.Getenv("TEST_DATABASE_URL")
	if migrationDSN == "" || appDSN == "" {
		t.Skip("set TEST_DATABASE_MIGRATION_URL and TEST_DATABASE_URL to run PostgreSQL RLS integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	migratorDB, err := database.Open(database.PoolConfig{
		DSN:          migrationDSN,
		MaxOpenConns: 2,
		MaxIdleConns: 1,
	})
	if err != nil {
		t.Fatalf("open migrator db: %v", err)
	}
	defer migratorDB.Close()

	// 1. Prove migrations apply up, roll down cleanly, and re-apply up.
	if err := database.MigrateUp(ctx, migratorDB.DB); err != nil {
		t.Fatalf("initial migrate up: %v", err)
	}
	if err := database.MigrateDownTo(ctx, migratorDB.DB, 0); err != nil {
		t.Fatalf("migrate down to 0: %v", err)
	}
	if err := database.MigrateUp(ctx, migratorDB.DB); err != nil {
		t.Fatalf("re-apply migrate up: %v", err)
	}

	// Open application pool with tight concurrency (MaxOpenConns=2) to force connection reuse.
	appDB, err := database.Open(database.PoolConfig{
		DSN:             appDSN,
		MaxOpenConns:    2,
		MaxIdleConns:    2,
		ConnMaxLifetime: 5 * time.Minute,
	})
	if err != nil {
		t.Fatalf("open app db: %v", err)
	}
	defer appDB.Close()

	// 2. Verify app role is strictly non-superuser and NOBYPASSRLS.
	var rolSuper, rolBypassRLS bool
	if err := appDB.QueryRowContext(ctx,
		"SELECT rolsuper, rolbypassrls FROM pg_roles WHERE rolname = current_user",
	).Scan(&rolSuper, &rolBypassRLS); err != nil {
		t.Fatalf("query app role attributes: %v", err)
	}
	if rolSuper || rolBypassRLS {
		t.Fatalf("app role must be NOSUPERUSER NOBYPASSRLS, got rolsuper=%v rolbypassrls=%v", rolSuper, rolBypassRLS)
	}

	// 3. Seed isolated fixtures for Tenant A and Tenant B via migrator connection.
	tenantA := uuid.New()
	tenantB := uuid.New()
	userA := uuid.New()
	userB := uuid.New()
	membershipA := uuid.New()
	membershipB := uuid.New()

	seedQueries := []struct {
		query string
		args  []any
	}{
		{"TRUNCATE tenant_memberships, users, tenants CASCADE", nil},
		{"INSERT INTO tenants (id, name, slug) VALUES (?, ?, ?)", []any{tenantA, "Academy A", "academy-a"}},
		{"INSERT INTO tenants (id, name, slug) VALUES (?, ?, ?)", []any{tenantB, "Academy B", "academy-b"}},
		{"INSERT INTO users (id, external_subject, display_name, email) VALUES (?, ?, ?, ?)", []any{userA, "google:user-a", "Teacher A", "a@example.com"}},
		{"INSERT INTO users (id, external_subject, display_name, email) VALUES (?, ?, ?, ?)", []any{userB, "google:user-b", "Teacher B", "b@example.com"}},
		{"INSERT INTO tenant_memberships (id, tenant_id, user_id, role) VALUES (?, ?, ?, ?)", []any{membershipA, tenantA, userA, string(tenancy.RoleTeacher)}},
		{"INSERT INTO tenant_memberships (id, tenant_id, user_id, role) VALUES (?, ?, ?, ?)", []any{membershipB, tenantB, userB, string(tenancy.RoleTeacher)}},
	}
	for _, sq := range seedQueries {
		if _, err := migratorDB.ExecContext(ctx, sq.query, sq.args...); err != nil {
			t.Fatalf("seed fixture %q: %v", sq.query, err)
		}
	}

	ctxA := tenancytest.MustContext(t, tenantA, userA, tenancy.RoleTeacher)
	ctxB := tenancytest.MustContext(t, tenantB, userB, tenancy.RoleTeacher)

	// 4. Fail-closed check: queries without tenant context return zero rows and reject writes.
	t.Run("fail-closed when tenant context is unset", func(t *testing.T) {
		var count int
		if err := appDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM tenant_memberships").Scan(&count); err != nil {
			t.Fatalf("count memberships without context: %v", err)
		}
		if count != 0 {
			t.Fatalf("expected 0 visible memberships without tenant context, got %d", count)
		}

		if err := appDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM tenants").Scan(&count); err != nil {
			t.Fatalf("count tenants without context: %v", err)
		}
		if count != 0 {
			t.Fatalf("expected 0 visible tenants without tenant context, got %d", count)
		}

		_, err := appDB.ExecContext(ctx,
			"INSERT INTO tenant_memberships (id, tenant_id, user_id, role) VALUES (?, ?, ?, ?)",
			uuid.New(), tenantA, userB, string(tenancy.RoleLearner),
		)
		if err == nil {
			t.Fatal("expected INSERT without tenant context to fail RLS check")
		}
	})

	// 5. Cross-tenant read, update, delete, and insert isolation.
	t.Run("tenant A cannot read, update, delete, or insert tenant B rows", func(t *testing.T) {
		err := database.RunInTenantTx(ctx, appDB, ctxA, func(ctx context.Context, tx bun.Tx) error {
			var visibleTenantIDs []uuid.UUID
			if err := tx.NewSelect().Table("tenants").Column("id").Scan(ctx, &visibleTenantIDs); err != nil {
				return err
			}
			if len(visibleTenantIDs) != 1 || visibleTenantIDs[0] != tenantA {
				t.Fatalf("expected only tenant A (%s), got %v", tenantA, visibleTenantIDs)
			}

			var visibleMembershipIDs []uuid.UUID
			if err := tx.NewSelect().Table("tenant_memberships").Column("id").Scan(ctx, &visibleMembershipIDs); err != nil {
				return err
			}
			if len(visibleMembershipIDs) != 1 || visibleMembershipIDs[0] != membershipA {
				t.Fatalf("expected only membership A (%s), got %v", membershipA, visibleMembershipIDs)
			}

			// Attempt cross-tenant UPDATE on Tenant B's membership.
			res, err := tx.ExecContext(ctx, "UPDATE tenant_memberships SET role = 'org_admin' WHERE id = ?", membershipB)
			if err != nil {
				return err
			}
			affected, _ := res.RowsAffected()
			if affected != 0 {
				t.Fatalf("expected 0 rows affected updating tenant B, got %d", affected)
			}

			// Attempt cross-tenant DELETE on Tenant B's membership.
			res, err = tx.ExecContext(ctx, "DELETE FROM tenant_memberships WHERE id = ?", membershipB)
			if err != nil {
				return err
			}
			affected, _ = res.RowsAffected()
			if affected != 0 {
				t.Fatalf("expected 0 rows affected deleting tenant B, got %d", affected)
			}

			return nil
		})
		if err != nil {
			t.Fatalf("RunInTenantTx tenant A failed: %v", err)
		}

		// Attempt cross-tenant INSERT for Tenant B while scoped to Tenant A.
		err = database.RunInTenantTx(ctx, appDB, ctxA, func(ctx context.Context, tx bun.Tx) error {
			_, err := tx.ExecContext(ctx,
				"INSERT INTO tenant_memberships (id, tenant_id, user_id, role) VALUES (?, ?, ?, ?)",
				uuid.New(), tenantB, userA, string(tenancy.RoleLearner),
			)
			return err
		})
		if err == nil {
			t.Fatal("expected cross-tenant INSERT into tenant B under tenant A context to fail RLS WITH CHECK")
		}
	})

	// 6. Pooled connection concurrency proof: 20 goroutines x 25 tx = 500 interleaved tx over 2 pooled conns.
	t.Run("zero cross-tenant leakage under concurrent pooled transactions", func(t *testing.T) {
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
					tc := ctxA
					expectedTenant := tenantA
					expectedMembership := membershipA
					if (workerID+i)%2 == 1 {
						tc = ctxB
						expectedTenant = tenantB
						expectedMembership = membershipB
					}

					err := database.RunInTenantTx(ctx, appDB, tc, func(ctx context.Context, tx bun.Tx) error {
						var tenantIDs []uuid.UUID
						if err := tx.NewSelect().Table("tenants").Column("id").Scan(ctx, &tenantIDs); err != nil {
							return err
						}
						if len(tenantIDs) != 1 || tenantIDs[0] != expectedTenant {
							return errors.New("cross-tenant leak detected in tenants table")
						}

						var membershipIDs []uuid.UUID
						if err := tx.NewSelect().Table("tenant_memberships").Column("id").Scan(ctx, &membershipIDs); err != nil {
							return err
						}
						if len(membershipIDs) != 1 || membershipIDs[0] != expectedMembership {
							return errors.New("cross-tenant leak detected in tenant_memberships table")
						}

						// Exercise rollback path on every 5th iteration to confirm pooled connection resets cleanly.
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

		// After all pooled transactions complete, confirm no residual tenant setting leaks on reused connections.
		for i := 0; i < 6; i++ {
			var count int
			if err := appDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM tenant_memberships").Scan(&count); err != nil {
				t.Fatalf("post-pool count check %d: %v", i, err)
			}
			if count != 0 {
				t.Fatalf("residual tenant context leaked onto pooled connection: saw %d rows", count)
			}
		}
	})
}
