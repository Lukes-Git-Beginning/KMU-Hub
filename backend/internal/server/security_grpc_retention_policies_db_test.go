package server

// DB-backed coverage for ListRetentionPolicies, which the main
// security_grpc_test.go file explicitly leaves out (it only has s.pool, no
// service to stub, same reasoning as ListIPRules). Before this test existed,
// ListRetentionPolicies had never run against a seeded row, so its
// nil-slice-on-empty-result bug (fix-security-grpc-nil-slice-wire-shape)
// went undetected by every prior test run.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
	securityv1 "github.com/kmuhub/kmuhub/proto/security/v1"
)

func TestSecurityGRPCServer_ListRetentionPolicies_EmptyIsEmptySliceNotNull(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Retention Policies Empty Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	srv := NewSecurityGRPCServer(nil, nil, nil, nil, nil, pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	resp, err := srv.ListRetentionPolicies(ctx, &securityv1.ListRetentionPoliciesRequest{})
	require.NoError(t, err)
	require.NotNil(t, resp.Policies)
	require.Empty(t, resp.Policies)
}

func TestSecurityGRPCServer_ListRetentionPolicies_ReturnsSeededPolicy(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Retention Policies List Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	srv := NewSecurityGRPCServer(nil, nil, nil, nil, nil, pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	created, err := srv.CreateRetentionPolicy(ctx, &securityv1.CreateRetentionPolicyRequest{
		ResourceType:  "audit_log",
		RetentionDays: 365,
		Action:        models.RetentionActionAnonymize,
	})
	require.NoError(t, err)
	policyID, err := uuid.Parse(created.Policy.Id)
	require.NoError(t, err)
	defer testutil.CleanupRow(t, pool, "retention_policies", policyID)

	resp, err := srv.ListRetentionPolicies(ctx, &securityv1.ListRetentionPoliciesRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Policies, 1)
	require.Equal(t, "audit_log", resp.Policies[0].ResourceType)
	require.Equal(t, int32(365), resp.Policies[0].RetentionDays)
}

// TestSecurityGRPCServer_CreateRetentionPolicy_AcceptsUnknownResourceType
// documents the current, deliberate state: resource_type is a free string at
// creation time (no enum/whitelist), and the gdpr.RetentionEngine reports an
// unrecognized value as "unmapped" at run time instead of silently doing
// nothing (see internal/security/gdpr/retention.go and
// TestRetentionEngine_EmptyRegistryReportsUnmapped). Introducing a whitelist
// here is an open product decision (BACKLOG-NEXT.yml), not something this
// test enforces -- it only locks in that creation does not reject a typo'd
// or not-yet-wired resource_type, so a later whitelist change is visible as
// an intentional diff, not a silent behavior shift.
func TestSecurityGRPCServer_CreateRetentionPolicy_AcceptsUnknownResourceType(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Retention Policies Unknown Type Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	srv := NewSecurityGRPCServer(nil, nil, nil, nil, nil, pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	created, err := srv.CreateRetentionPolicy(ctx, &securityv1.CreateRetentionPolicyRequest{
		ResourceType:  "kontakte_typo_not_a_real_type",
		RetentionDays: 90,
		Action:        models.RetentionActionDelete,
	})
	require.NoError(t, err)
	policyID, err := uuid.Parse(created.Policy.Id)
	require.NoError(t, err)
	defer testutil.CleanupRow(t, pool, "retention_policies", policyID)

	require.Equal(t, "kontakte_typo_not_a_real_type", created.Policy.ResourceType)
}
