package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/middleware"
)

func NewPostgresPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}
	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute
	config.HealthCheckPeriod = time.Minute
	config.ConnConfig.ConnectTimeout = 10 * time.Second

	// PrepareConn stamps the connection's session GUCs (app.tenant_id,
	// app.user_id, app.role, app.user_roles) from the caller context. RLS
	// policies installed by enable_tenant_rls() filter against these GUCs —
	// without them the policies admit nothing.
	//
	// Behaviour:
	//   - System context (database.WithSystemContext): app.role='system'.
	//     Policies bypass the tenant filter via is_system_context().
	//   - Authenticated request: tenant via middleware.GetTenantID(ctx),
	//     user via middleware.GetUserID(ctx), roles via
	//     middleware.GetUserRoles(ctx). app.role='user'.
	//   - Unauthenticated/missing tenant outside system mode: returns
	//     (true, nil), GUCs stay at AfterRelease's '' default. RLS policies
	//     then filter every row away — caller sees empty results, which is
	//     the safe default. Login, Register, and other pre-JWT paths must
	//     wrap with WithSystemContext.
	//
	// Scope is session (third arg false to set_config); AfterRelease resets
	// before the connection returns to the pool. BeginRLSTx still applies
	// LOCAL-scoped overrides for explicit multi-statement transactions.
	config.PrepareConn = func(ctx context.Context, conn *pgx.Conn) (bool, error) {
		if IsSystemContext(ctx) {
			if _, err := conn.Exec(ctx,
				`SELECT set_config('app.tenant_id',  '', false),
                        set_config('app.user_id',    '', false),
                        set_config('app.role',       'system', false),
                        set_config('app.user_roles', '', false)`); err != nil {
				return false, fmt.Errorf("set system-context guc: %w", err)
			}
			return true, nil
		}

		tenantID, terr := middleware.GetTenantID(ctx)
		if terr != nil {
			return true, nil
		}

		userStr := ""
		if uid := middleware.GetUserID(ctx); uid != "" {
			if _, perr := uuid.Parse(uid); perr == nil {
				userStr = uid
			}
		}

		rolesStr := ""
		if roles := middleware.GetUserRoles(ctx); len(roles) > 0 {
			rolesStr = strings.Join(roles, ",")
		}

		if _, err := conn.Exec(ctx,
			`SELECT set_config('app.tenant_id',  $1, false),
                    set_config('app.user_id',    $2, false),
                    set_config('app.role',       'user', false),
                    set_config('app.user_roles', $3, false)`,
			tenantID.String(), userStr, rolesStr); err != nil {
			return false, fmt.Errorf("set tenant-context guc: %w", err)
		}
		return true, nil
	}

	// AfterRelease clears the RLS session GUCs before the connection
	// returns to the pool. BeforeAcquire stamps fresh values on the next
	// caller, so without this reset a connection could leak a previous
	// caller's tenant. Returning false discards the connection on reset
	// failure rather than handing back a possibly-tainted one.
	config.AfterRelease = func(conn *pgx.Conn) bool {
		_, err := conn.Exec(context.Background(),
			`SELECT set_config('app.tenant_id',  '', false),
                    set_config('app.user_id',    '', false),
                    set_config('app.role',       '', false),
                    set_config('app.user_roles', '', false)`)
		return err == nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}
