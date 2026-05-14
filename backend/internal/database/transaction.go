package database

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/middleware"
)

// BeginRLSTx starts a transaction whose session-level GUCs (app.tenant_id,
// app.user_id, app.role, app.user_roles) match the caller context. RLS
// policies installed by enable_tenant_rls() filter against these GUCs —
// without them the connection sees an empty tenant and policies reject every
// row.
//
// Behaviour:
//   - System context (WithSystemContext): sets app.role='system'. Policies
//     admit every row via is_system_context().
//   - Authenticated request: pulls tenant via middleware.GetTenantID(ctx)
//     and user via middleware.GetUserID(ctx). Sets app.role='user'. The
//     user's role list (e.g. "admin,hr_admin,manager") is exposed via
//     app.user_roles for role-aware policies such as hr_document_access
//     (mig 000127).
//   - Unauthenticated context outside system mode: returns
//     ErrMissingTenantContext. Callers must wrap with WithSystemContext or
//     run on a non-RLS code path.
//
// All SET calls use the LOCAL scope (third argument true to set_config),
// so the GUCs revert at COMMIT/ROLLBACK and never leak via pool reuse.
// The AfterRelease hook in postgres.go is a defence-in-depth reset for
// any non-transactional code path.
//
// Caller is responsible for tx.Commit(ctx) or tx.Rollback(ctx).
func BeginRLSTx(ctx context.Context, pool *pgxpool.Pool) (pgx.Tx, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}

	role := "user"
	tenantStr := ""
	userStr := ""
	rolesStr := ""

	if IsSystemContext(ctx) {
		role = "system"
	} else {
		tenantID, terr := middleware.GetTenantID(ctx)
		if terr != nil {
			_ = tx.Rollback(ctx)
			return nil, fmt.Errorf("%w: %v", ErrMissingTenantContext, terr)
		}
		tenantStr = tenantID.String()
		if uid := middleware.GetUserID(ctx); uid != "" {
			if _, perr := uuid.Parse(uid); perr == nil {
				userStr = uid
			}
		}
		if roles := middleware.GetUserRoles(ctx); len(roles) > 0 {
			rolesStr = strings.Join(roles, ",")
		}
	}

	_, err = tx.Exec(ctx,
		`SELECT set_config('app.tenant_id',  $1, true),
                set_config('app.user_id',    $2, true),
                set_config('app.role',       $3, true),
                set_config('app.user_roles', $4, true)`,
		tenantStr, userStr, role, rolesStr,
	)
	if err != nil {
		_ = tx.Rollback(ctx)
		return nil, fmt.Errorf("set rls session vars: %w", err)
	}
	return tx, nil
}

// ErrMissingTenantContext is returned by BeginRLSTx when the context
// carries neither a tenant id nor the system-context marker.
var ErrMissingTenantContext = errors.New("missing tenant context for RLS transaction")
