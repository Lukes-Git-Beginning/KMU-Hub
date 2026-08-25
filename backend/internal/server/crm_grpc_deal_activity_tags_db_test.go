package server

// DB-backed proof for the four tag RPCs that used to be missing entirely, so
// the gateway handlers behind /api/v1/deals/{id}/tags and
// /api/v1/activities/{id}/tags answered a permanent 501
// (feat-crm-activity-deal-tag-rpcs).
//
// These tests go through CRMGRPCServer -- the same entry point the gateway
// reaches over gRPC -- with real postgres repositories, and then look at the
// join tables directly. A service-level test would not catch a missing or
// mis-parsed RPC field, and a mock repository would not catch the tenant_id
// the INSERT copies out of the parent row.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/crm/activity"
	"github.com/kmuhub/kmuhub/internal/crm/deal"
	"github.com/kmuhub/kmuhub/internal/testutil"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

// tagServerForTest builds a CRMGRPCServer with only the two services the tag
// RPCs touch; every other dependency stays nil, which is exactly what the
// methods under test require.
func tagServerForTest(pool *pgxpool.Pool) *CRMGRPCServer {
	return NewCRMGRPCServer(
		nil, nil, nil, nil, nil,
		deal.NewService(deal.NewPostgresRepository(pool)),
		activity.NewService(activity.NewPostgresRepository(pool)),
		nil, nil, nil, nil,
	)
}

// joinRowTenant returns the tenant_id stored on the join row, or uuid.Nil when
// no such row exists.
func joinRowTenant(t *testing.T, pool *pgxpool.Pool, table, idCol string, entityID, tagID uuid.UUID) uuid.UUID {
	t.Helper()
	ctx := testutil.WithSystemCtx(context.Background())
	var tenantID uuid.UUID
	err := pool.QueryRow(ctx,
		fmt.Sprintf(`SELECT tenant_id FROM %s WHERE %s = $1 AND tag_id = $2`, table, idCol),
		entityID, tagID,
	).Scan(&tenantID)
	if err != nil {
		return uuid.Nil
	}
	return tenantID
}

func TestCRMGRPCServer_DealTags_AddThenRemoveHitsJoinTable(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CRM Deal Tags Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id": tenantID, "email": fmt.Sprintf("crm-deal-tags-%s@tenant.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	stageID := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"tenant_id": tenantID, "name": fmt.Sprintf("Deal-Tags-Stage-%s", uuid.New().String()[:8]),
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageID)

	dealID := testutil.SeedRow(t, pool, "deals", map[string]any{
		"tenant_id": tenantID, "name": "Tagged Deal", "stage_id": stageID, "created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "deals", dealID)

	tagID := testutil.SeedRow(t, pool, "tags", map[string]any{
		"tenant_id": tenantID, "name": fmt.Sprintf("deal-tag-%s", uuid.New().String()[:8]), "entity_type": "deal",
	})
	defer testutil.CleanupRow(t, pool, "tags", tagID)

	srv := tagServerForTest(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	added, err := srv.AddDealTags(ctx, &crmv1.AddDealTagsRequest{
		DealId: dealID.String(),
		TagIds: []string{tagID.String()},
	})
	require.NoError(t, err)
	require.NotNil(t, added.Deal)
	require.Equal(t, dealID.String(), added.Deal.Id)

	// The join row exists AND carries the parent tenant, not NULL and not a
	// foreign one -- the INSERT copies it out of `deals`.
	require.Equal(t, tenantID, joinRowTenant(t, pool, "deal_tags", "deal_id", dealID, tagID),
		"deal_tags row missing or stamped with the wrong tenant after AddDealTags")

	removed, err := srv.RemoveDealTags(ctx, &crmv1.RemoveDealTagsRequest{
		DealId: dealID.String(),
		TagIds: []string{tagID.String()},
	})
	require.NoError(t, err)
	require.NotNil(t, removed.Deal)
	require.Equal(t, uuid.Nil, joinRowTenant(t, pool, "deal_tags", "deal_id", dealID, tagID),
		"deal_tags row still present after RemoveDealTags")
}

// TestCRMGRPCServer_AddDealTags_ForeignTenantGetsNotFound proves the RPC reads
// the tenant from the context and not from the request: the deal exists, but
// not for the caller's tenant, so the lookup in Service.AddTags must fail
// before anything is written.
func TestCRMGRPCServer_AddDealTags_ForeignTenantGetsNotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID, foreignTenantID := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CRM Deal Tags Owner")
	testutil.EnsureTenant(t, pool, foreignTenantID, "CRM Deal Tags Intruder")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	defer testutil.CleanupRow(t, pool, "tenants", foreignTenantID)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id": tenantID, "email": fmt.Sprintf("crm-deal-tags-x-%s@tenant.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	stageID := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"tenant_id": tenantID, "name": fmt.Sprintf("Deal-Tags-XStage-%s", uuid.New().String()[:8]),
	})
	defer testutil.CleanupRow(t, pool, "pipeline_stages", stageID)

	dealID := testutil.SeedRow(t, pool, "deals", map[string]any{
		"tenant_id": tenantID, "name": "Foreign Deal", "stage_id": stageID, "created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "deals", dealID)

	tagID := testutil.SeedRow(t, pool, "tags", map[string]any{
		"tenant_id": tenantID, "name": fmt.Sprintf("deal-xtag-%s", uuid.New().String()[:8]), "entity_type": "deal",
	})
	defer testutil.CleanupRow(t, pool, "tags", tagID)

	srv := tagServerForTest(pool)
	foreignCtx := testutil.WithTenantCtx(context.Background(), foreignTenantID)

	_, err := srv.AddDealTags(foreignCtx, &crmv1.AddDealTagsRequest{
		DealId: dealID.String(),
		TagIds: []string{tagID.String()},
	})
	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
	require.Equal(t, uuid.Nil, joinRowTenant(t, pool, "deal_tags", "deal_id", dealID, tagID),
		"a foreign tenant managed to write a deal_tags row")
}

func TestCRMGRPCServer_ActivityTags_AddThenRemoveHitsJoinTable(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CRM Activity Tags Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id": tenantID, "email": fmt.Sprintf("crm-act-tags-%s@tenant.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	activityID := testutil.SeedRow(t, pool, "activities", map[string]any{
		"tenant_id": tenantID, "activity_type": "note", "subject": "Tagged Activity", "created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "activities", activityID)

	tagID := testutil.SeedRow(t, pool, "tags", map[string]any{
		"tenant_id": tenantID, "name": fmt.Sprintf("act-tag-%s", uuid.New().String()[:8]), "entity_type": "activity",
	})
	defer testutil.CleanupRow(t, pool, "tags", tagID)

	srv := tagServerForTest(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	added, err := srv.AddActivityTags(ctx, &crmv1.AddActivityTagsRequest{
		ActivityId: activityID.String(),
		TagIds:     []string{tagID.String()},
	})
	require.NoError(t, err)
	require.NotNil(t, added.Activity)
	require.Equal(t, activityID.String(), added.Activity.Id)
	require.Equal(t, tenantID, joinRowTenant(t, pool, "activity_tags", "activity_id", activityID, tagID),
		"activity_tags row missing or stamped with the wrong tenant after AddActivityTags")

	removed, err := srv.RemoveActivityTags(ctx, &crmv1.RemoveActivityTagsRequest{
		ActivityId: activityID.String(),
		TagIds:     []string{tagID.String()},
	})
	require.NoError(t, err)
	require.NotNil(t, removed.Activity)
	require.Equal(t, uuid.Nil, joinRowTenant(t, pool, "activity_tags", "activity_id", activityID, tagID),
		"activity_tags row still present after RemoveActivityTags")
}

// TestCRMGRPCServer_AddActivityTags_WrongEntityTypeTagIsRejected proves the tag
// existence check in the service is reached through the RPC: a tag id that
// exists but for the wrong entity type must not produce a join row.
func TestCRMGRPCServer_AddActivityTags_WrongEntityTypeTagIsRejected(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CRM Activity Tags Wrong Type Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id": tenantID, "email": fmt.Sprintf("crm-act-tags-w-%s@tenant.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	activityID := testutil.SeedRow(t, pool, "activities", map[string]any{
		"tenant_id": tenantID, "activity_type": "note", "subject": "Wrong Tag Type", "created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "activities", activityID)

	// entity_type 'deal', deliberately not 'activity'.
	dealTagID := testutil.SeedRow(t, pool, "tags", map[string]any{
		"tenant_id": tenantID, "name": fmt.Sprintf("wrong-type-%s", uuid.New().String()[:8]), "entity_type": "deal",
	})
	defer testutil.CleanupRow(t, pool, "tags", dealTagID)

	srv := tagServerForTest(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	_, err := srv.AddActivityTags(ctx, &crmv1.AddActivityTagsRequest{
		ActivityId: activityID.String(),
		TagIds:     []string{dealTagID.String()},
	})
	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
	require.Equal(t, uuid.Nil, joinRowTenant(t, pool, "activity_tags", "activity_id", activityID, dealTagID),
		"a tag of the wrong entity_type was written to activity_tags")
}

// TestCRMGRPCServer_TagRPCs_InvalidIDsAreRejected covers the two parse paths in
// the RPCs themselves, which no DB round-trip reaches.
func TestCRMGRPCServer_TagRPCs_InvalidIDsAreRejected(t *testing.T) {
	t.Parallel()

	srv := tagServerForTest(nil)
	ctx := testutil.WithTenantCtx(context.Background(), uuid.New())

	_, err := srv.AddDealTags(ctx, &crmv1.AddDealTagsRequest{DealId: "not-a-uuid"})
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = srv.AddDealTags(ctx, &crmv1.AddDealTagsRequest{
		DealId: uuid.New().String(), TagIds: []string{"not-a-uuid"},
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = srv.AddActivityTags(ctx, &crmv1.AddActivityTagsRequest{
		ActivityId: uuid.New().String(), TagIds: []string{"not-a-uuid"},
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = srv.RemoveDealTags(ctx, &crmv1.RemoveDealTagsRequest{DealId: "not-a-uuid"})
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = srv.RemoveActivityTags(ctx, &crmv1.RemoveActivityTagsRequest{ActivityId: "not-a-uuid"})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

// TestCRMGRPCServer_TagRPCs_RequireTenantContext proves all four RPCs refuse a
// context without a tenant instead of falling back to a default.
func TestCRMGRPCServer_TagRPCs_RequireTenantContext(t *testing.T) {
	t.Parallel()

	srv := tagServerForTest(nil)
	ctx := context.Background()
	id := uuid.New().String()

	_, err := srv.AddDealTags(ctx, &crmv1.AddDealTagsRequest{DealId: id})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
	_, err = srv.RemoveDealTags(ctx, &crmv1.RemoveDealTagsRequest{DealId: id})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
	_, err = srv.AddActivityTags(ctx, &crmv1.AddActivityTagsRequest{ActivityId: id})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
	_, err = srv.RemoveActivityTags(ctx, &crmv1.RemoveActivityTagsRequest{ActivityId: id})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}
