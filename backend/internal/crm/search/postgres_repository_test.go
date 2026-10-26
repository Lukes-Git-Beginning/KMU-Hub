package search

// DB tests for the full-text repository. They run as kmuhub_app via
// DATABASE_URL, so RLS is live: a wrong tenant filter shows up as a wrong row.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// searchFixture is one tenant with an owning user and a pipeline stage, the
// two parents every searchable CRM row needs.
type searchFixture struct {
	tenantID uuid.UUID
	userID   uuid.UUID
	stageID  uuid.UUID
}

func newSearchFixture(t *testing.T, pool *pgxpool.Pool, name string) searchFixture {
	t.Helper()
	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, name)
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id": tenantID, "email": "search-" + uuid.NewString() + "@example.test",
		"password_hash": "hash", "first_name": "Search", "last_name": "Owner",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })
	stageID := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
		"tenant_id": tenantID, "name": "Search Stage",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "pipeline_stages", stageID) })
	return searchFixture{tenantID: tenantID, userID: userID, stageID: stageID}
}

func (f searchFixture) seed(t *testing.T, pool *pgxpool.Pool, table string, cols map[string]any) uuid.UUID {
	t.Helper()
	cols["tenant_id"] = f.tenantID
	cols["created_by"] = f.userID
	id := testutil.SeedRow(t, pool, table, cols)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, table, id) })
	return id
}

func newSearchRepo(t *testing.T) (*PostgresRepository, *pgxpool.Pool) {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	return NewPostgresRepository(pool), pool
}

func tenantCtx(f searchFixture) context.Context {
	return testutil.WithTenantCtx(context.Background(), f.tenantID)
}

func TestSearchContacts_MatchesNameAndMapsFields(t *testing.T) {
	repo, pool := newSearchRepo(t)
	f := newSearchFixture(t, pool, "Search Contacts Tenant")
	hit := f.seed(t, pool, "contacts", map[string]any{
		"first_name": "Erika", "last_name": "Quarkfix", "email": "erika@quarkfix.example",
	})
	f.seed(t, pool, "contacts", map[string]any{"first_name": "Hans", "last_name": "Unrelated"})
	noEmail := f.seed(t, pool, "contacts", map[string]any{"first_name": "Otto", "last_name": "Quarkfix"})

	got, err := repo.SearchContacts(tenantCtx(f), f.tenantID, "Quarkfix", 10)
	require.NoError(t, err)
	require.Len(t, got, 2, "only the two Quarkfix contacts match")

	byID := map[string]string{}
	for _, r := range got {
		assert.Equal(t, "contact", r.EntityType)
		assert.Positive(t, r.Score)
		byID[r.ID] = r.Title + "|" + r.Subtitle
	}
	assert.Equal(t, "Erika Quarkfix|erika@quarkfix.example", byID[hit.String()], "title is first+last name, subtitle the email")
	assert.Equal(t, "Otto Quarkfix|", byID[noEmail.String()], "NULL email becomes an empty subtitle")
}

func TestSearchContacts_RanksAndLimits(t *testing.T) {
	repo, pool := newSearchRepo(t)
	f := newSearchFixture(t, pool, "Search Rank Tenant")
	// Name is weight A, notes weight D: the name hit must come first.
	weak := f.seed(t, pool, "contacts", map[string]any{
		"first_name": "Anna", "last_name": "Meier", "notes": "Kennt Rankingwort aus der Messe",
	})
	strong := f.seed(t, pool, "contacts", map[string]any{"first_name": "Rankingwort", "last_name": "Schmidt"})

	got, err := repo.SearchContacts(tenantCtx(f), f.tenantID, "Rankingwort", 10)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, strong.String(), got[0].ID, "weight-A name hit outranks the weight-D notes hit")
	assert.Equal(t, weak.String(), got[1].ID)
	assert.Greater(t, got[0].Score, got[1].Score)

	limited, err := repo.SearchContacts(tenantCtx(f), f.tenantID, "Rankingwort", 1)
	require.NoError(t, err)
	require.Len(t, limited, 1, "LIMIT is applied")
	assert.Equal(t, strong.String(), limited[0].ID, "the limit keeps the best hit")
}

func TestSearchContacts_NoMatchIsEmptyNotError(t *testing.T) {
	repo, pool := newSearchRepo(t)
	f := newSearchFixture(t, pool, "Search Empty Tenant")
	f.seed(t, pool, "contacts", map[string]any{"first_name": "Erika", "last_name": "Mustermann"})

	got, err := repo.SearchContacts(tenantCtx(f), f.tenantID, "Nirgendwoanzutreffen", 10)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestSearchCompanies_MatchesAndMapsFields(t *testing.T) {
	repo, pool := newSearchRepo(t)
	f := newSearchFixture(t, pool, "Search Companies Tenant")
	hit := f.seed(t, pool, "companies", map[string]any{"name": "Quarkwerke GmbH", "industry": "Lebensmittel"})
	noIndustry := f.seed(t, pool, "companies", map[string]any{"name": "Quarkhandel AG"})
	f.seed(t, pool, "companies", map[string]any{"name": "Stahlbau KG"})

	got, err := repo.SearchCompanies(tenantCtx(f), f.tenantID, "Quarkwerke", 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, hit.String(), got[0].ID)
	assert.Equal(t, "company", got[0].EntityType)
	assert.Equal(t, "Quarkwerke GmbH", got[0].Title)
	assert.Equal(t, "Lebensmittel", got[0].Subtitle)

	got, err = repo.SearchCompanies(tenantCtx(f), f.tenantID, "Quarkhandel", 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, noIndustry.String(), got[0].ID)
	assert.Equal(t, "", got[0].Subtitle, "NULL industry becomes an empty subtitle")
}

func TestSearchDeals_MatchesAndFormatsValueAndCurrency(t *testing.T) {
	repo, pool := newSearchRepo(t)
	f := newSearchFixture(t, pool, "Search Deals Tenant")
	hit := f.seed(t, pool, "deals", map[string]any{
		"name": "Rahmenvertrag Quarkfix", "value": 1500.5, "currency": "CHF", "stage_id": f.stageID,
	})
	f.seed(t, pool, "deals", map[string]any{"name": "Anderes Geschaeft", "stage_id": f.stageID})

	got, err := repo.SearchDeals(tenantCtx(f), f.tenantID, "Quarkfix", 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, hit.String(), got[0].ID)
	assert.Equal(t, "deal", got[0].EntityType)
	assert.Equal(t, "Rahmenvertrag Quarkfix", got[0].Title)
	assert.Equal(t, "1500.50 CHF", got[0].Subtitle, "numeric(15,2) text plus currency")
}

func TestSearchActivities_MatchesAndShowsType(t *testing.T) {
	repo, pool := newSearchRepo(t)
	f := newSearchFixture(t, pool, "Search Activities Tenant")
	hit := f.seed(t, pool, "activities", map[string]any{"activity_type": "meeting", "subject": "Quarkfix Workshop"})
	byBody := f.seed(t, pool, "activities", map[string]any{
		"activity_type": "call", "subject": "Rueckruf", "description": "Wegen Quarkfix nachfassen",
	})
	f.seed(t, pool, "activities", map[string]any{"activity_type": "note", "subject": "Unbeteiligt"})

	got, err := repo.SearchActivities(tenantCtx(f), f.tenantID, "Quarkfix", 10)
	require.NoError(t, err)
	require.Len(t, got, 2, "subject (A) and description (B) both feed the vector")
	assert.Equal(t, hit.String(), got[0].ID, "the subject hit outranks the description hit")
	assert.Equal(t, byBody.String(), got[1].ID)
	assert.Equal(t, "activity", got[0].EntityType)
	assert.Equal(t, "Quarkfix Workshop", got[0].Title)
	assert.Equal(t, "meeting", got[0].Subtitle, "enum cast to text")
	assert.Equal(t, "call", got[1].Subtitle)
}

// TestSearch_IsTenantScoped plants the same searchable word in a second tenant
// and checks that every entity search returns only the caller's own rows. The
// searches also run under the foreign tenant's context to pin both layers:
// the explicit tenant_id predicate and RLS must agree.
func TestSearch_IsTenantScoped(t *testing.T) {
	repo, pool := newSearchRepo(t)
	own := newSearchFixture(t, pool, "Search Own Tenant")
	foreign := newSearchFixture(t, pool, "Search Foreign Tenant")

	ownIDs := map[string]uuid.UUID{
		"contact":  own.seed(t, pool, "contacts", map[string]any{"first_name": "Scopewort", "last_name": "Eigen"}),
		"company":  own.seed(t, pool, "companies", map[string]any{"name": "Scopewort Eigen"}),
		"deal":     own.seed(t, pool, "deals", map[string]any{"name": "Scopewort Eigen", "stage_id": own.stageID}),
		"activity": own.seed(t, pool, "activities", map[string]any{"activity_type": "note", "subject": "Scopewort Eigen"}),
	}
	foreignIDs := map[string]uuid.UUID{
		"contact":  foreign.seed(t, pool, "contacts", map[string]any{"first_name": "Scopewort", "last_name": "Fremd"}),
		"company":  foreign.seed(t, pool, "companies", map[string]any{"name": "Scopewort Fremd"}),
		"deal":     foreign.seed(t, pool, "deals", map[string]any{"name": "Scopewort Fremd", "stage_id": foreign.stageID}),
		"activity": foreign.seed(t, pool, "activities", map[string]any{"activity_type": "note", "subject": "Scopewort Fremd"}),
	}

	searches := map[string]func(context.Context, uuid.UUID) ([]string, error){}
	collect := func(f func(context.Context, uuid.UUID, string, int) ([]*models.SearchResult, error)) func(context.Context, uuid.UUID) ([]string, error) {
		return func(ctx context.Context, tenantID uuid.UUID) ([]string, error) {
			rows, err := f(ctx, tenantID, "Scopewort", 50)
			ids := make([]string, 0, len(rows))
			for _, r := range rows {
				ids = append(ids, r.ID)
			}
			return ids, err
		}
	}
	searches["contact"] = collect(repo.SearchContacts)
	searches["company"] = collect(repo.SearchCompanies)
	searches["deal"] = collect(repo.SearchDeals)
	searches["activity"] = collect(repo.SearchActivities)

	for entity, run := range searches {
		ids, err := run(tenantCtx(own), own.tenantID)
		require.NoError(t, err, entity)
		assert.Equal(t, []string{ownIDs[entity].String()}, ids, "%s: own tenant sees exactly its own row", entity)

		ids, err = run(tenantCtx(foreign), foreign.tenantID)
		require.NoError(t, err, entity)
		assert.Equal(t, []string{foreignIDs[entity].String()}, ids, "%s: foreign tenant sees exactly its own row", entity)

		// Mismatch between the context tenant and the filter argument: RLS
		// and the predicate cannot both be satisfied, so nothing comes back.
		ids, err = run(tenantCtx(own), foreign.tenantID)
		require.NoError(t, err, entity)
		assert.Empty(t, ids, "%s: a foreign tenant id under our context leaks nothing", entity)
	}
}

func TestSearch_ClosedPoolSurfacesErrors(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	repo := NewPostgresRepository(pool)
	pool.Close()
	ctx := testutil.WithTenantCtx(context.Background(), uuid.New())

	_, err := repo.SearchContacts(ctx, uuid.New(), "irgendwas", 5)
	assert.Error(t, err)
	_, err = repo.SearchCompanies(ctx, uuid.New(), "irgendwas", 5)
	assert.Error(t, err)
	_, err = repo.SearchDeals(ctx, uuid.New(), "irgendwas", 5)
	assert.Error(t, err)
	_, err = repo.SearchActivities(ctx, uuid.New(), "irgendwas", 5)
	assert.Error(t, err)
}
