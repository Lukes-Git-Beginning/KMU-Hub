package server

// DB-backed coverage for the ListIPRules/CreateIPRule pair, which the main
// security_grpc_test.go file explicitly leaves out (it only has s.pool, no
// service to stub). ip_cidr is a NOT NULL CIDR column; before this test
// existed, ListIPRules had never run against a seeded row, so the pgx
// scan-type mismatch (fix-security-ip-access-rules-cidr-scan) went
// undetected by every prior test run.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
	securityv1 "github.com/kmuhub/kmuhub/proto/security/v1"
)

func TestSecurityGRPCServer_ListIPRules_ReturnsSeededCIDR(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "IP Rules List Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	srv := NewSecurityGRPCServer(nil, nil, nil, nil, nil, pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	created, err := srv.CreateIPRule(ctx, &securityv1.CreateIPRuleRequest{
		IpCidr:      "10.0.0.0/8",
		RuleType:    models.IPRuleAllow,
		Description: "office network",
	})
	require.NoError(t, err)
	ruleID, err := uuid.Parse(created.Rule.Id)
	require.NoError(t, err)
	defer testutil.CleanupRow(t, pool, "ip_access_rules", ruleID)

	resp, err := srv.ListIPRules(ctx, &securityv1.ListIPRulesRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Rules, 1)
	require.Equal(t, "10.0.0.0/8", resp.Rules[0].IpCidr)
	require.Equal(t, models.IPRuleAllow, resp.Rules[0].RuleType)
	require.Equal(t, "office network", resp.Rules[0].Description)
}

// TestSecurityGRPCServer_ListIPRules_PreservesHostBit proves the fix uses a
// text cast, not host(), which would silently strip the CIDR prefix (e.g.
// "10.0.0.0/8" -> "10.0.0.0") and corrupt the stored network range.
func TestSecurityGRPCServer_ListIPRules_PreservesHostBit(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "IP Rules Preserve Prefix Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	srv := NewSecurityGRPCServer(nil, nil, nil, nil, nil, pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	created, err := srv.CreateIPRule(ctx, &securityv1.CreateIPRuleRequest{
		IpCidr:   "192.168.1.1/32",
		RuleType: models.IPRuleBlock,
	})
	require.NoError(t, err)
	ruleID, err := uuid.Parse(created.Rule.Id)
	require.NoError(t, err)
	defer testutil.CleanupRow(t, pool, "ip_access_rules", ruleID)

	resp, err := srv.ListIPRules(ctx, &securityv1.ListIPRulesRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Rules, 1)
	require.Equal(t, "192.168.1.1/32", resp.Rules[0].IpCidr)
}
