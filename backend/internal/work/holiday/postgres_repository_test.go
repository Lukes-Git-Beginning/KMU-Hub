package holiday_test

// postgres_repository_test.go covers the holiday repository against the real
// schema as kmuhub_app. public_holidays is a system-global table (no tenant_id,
// no RLS), so every test works in its own year and the unassigned country code
// "ZZ" to stay clear of seeded DE/AT/CH data and of parallel tests.

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
	"github.com/kmuhub/kmuhub/internal/work/holiday"
)

const testCountry = "ZZ"

// nextYear hands out a distinct year per test (2200+ is outside the 2000-2100
// service range, so no real seed can collide).
var yearCounter atomic.Int32

func nextYear() int { return 2200 + int(yearCounter.Add(1)) }

type holidayFixture struct {
	pool *pgxpool.Pool
	repo *holiday.PostgresRepository
	year int
	ctx  context.Context
}

func newHolidayFixture(t *testing.T) *holidayFixture {
	t.Helper()
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	f := &holidayFixture{
		pool: pool,
		repo: holiday.NewPostgresRepository(pool),
		year: nextYear(),
		ctx:  context.Background(),
	}
	t.Cleanup(func() {
		_, err := pool.Exec(context.Background(),
			`DELETE FROM public_holidays WHERE year = $1 AND country_code = $2`, f.year, testCountry)
		require.NoError(t, err)
	})
	return f
}

func (f *holidayFixture) day(month time.Month, d int) time.Time {
	return time.Date(f.year, month, d, 0, 0, 0, 0, time.UTC)
}

func (f *holidayFixture) holiday(month time.Month, d int, name string, global bool, subs ...string) models.PublicHoliday {
	return models.PublicHoliday{
		ID:               uuid.New(),
		Date:             f.day(month, d),
		Name:             name,
		LocalName:        name + " (lokal)",
		CountryCode:      testCountry,
		IsGlobal:         global,
		SubdivisionCodes: subs,
		HolidayType:      "public",
		Year:             f.year,
	}
}

func names(hs []models.PublicHoliday) []string {
	out := make([]string, 0, len(hs))
	for _, h := range hs {
		out = append(out, h.Name)
	}
	return out
}

func TestHolidayRepo_BulkUpsertEmptyIsNoop(t *testing.T) {
	f := newHolidayFixture(t)

	require.NoError(t, f.repo.BulkUpsert(f.ctx, nil))

	has, err := f.repo.HasDataForYear(f.ctx, f.year, testCountry)
	require.NoError(t, err)
	assert.False(t, has)
}

func TestHolidayRepo_BulkUpsertRoundtrip(t *testing.T) {
	f := newHolidayFixture(t)

	in := f.holiday(time.May, 1, "Labour Day", false, "ZZ-A", "ZZ-B")
	in.HolidayType = "bank"
	require.NoError(t, f.repo.BulkUpsert(f.ctx, []models.PublicHoliday{in}))

	got, err := f.repo.ListByDateRange(f.ctx, f.day(time.January, 1), f.day(time.December, 31), testCountry, nil)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, in.ID, got[0].ID)
	assert.Equal(t, in.Date.Format("2006-01-02"), got[0].Date.Format("2006-01-02"))
	assert.Equal(t, "Labour Day", got[0].Name)
	assert.Equal(t, "Labour Day (lokal)", got[0].LocalName)
	assert.Equal(t, testCountry, got[0].CountryCode)
	assert.False(t, got[0].IsGlobal)
	assert.Equal(t, []string{"ZZ-A", "ZZ-B"}, got[0].SubdivisionCodes)
	assert.Equal(t, "bank", got[0].HolidayType)
	assert.Equal(t, f.year, got[0].Year)
	assert.WithinDuration(t, time.Now(), got[0].CreatedAt, time.Minute, "zero CreatedAt falls back to now")
}

func TestHolidayRepo_BulkUpsertKeepsExplicitCreatedAt(t *testing.T) {
	f := newHolidayFixture(t)

	in := f.holiday(time.May, 1, "Old Row", true)
	in.CreatedAt = time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	require.NoError(t, f.repo.BulkUpsert(f.ctx, []models.PublicHoliday{in}))

	got, err := f.repo.ListByDateRange(f.ctx, f.day(time.January, 1), f.day(time.December, 31), testCountry, nil)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.True(t, in.CreatedAt.Equal(got[0].CreatedAt))
}

func TestHolidayRepo_BulkUpsertConflictUpdatesMutableColumns(t *testing.T) {
	f := newHolidayFixture(t)

	first := f.holiday(time.May, 1, "Labour Day", false, "ZZ-A")
	require.NoError(t, f.repo.BulkUpsert(f.ctx, []models.PublicHoliday{first}))

	// Same (date, country, name) -> conflict path; a new ID must not replace the row ID.
	second := f.holiday(time.May, 1, "Labour Day", true)
	second.LocalName = "Tag der Arbeit"
	second.SubdivisionCodes = []string{"ZZ-C"}
	second.HolidayType = "optional"
	require.NoError(t, f.repo.BulkUpsert(f.ctx, []models.PublicHoliday{second}))

	got, err := f.repo.ListByDateRange(f.ctx, f.day(time.January, 1), f.day(time.December, 31), testCountry, nil)
	require.NoError(t, err)
	require.Len(t, got, 1, "conflict must update in place, not duplicate")
	assert.Equal(t, first.ID, got[0].ID)
	assert.Equal(t, "Tag der Arbeit", got[0].LocalName)
	assert.True(t, got[0].IsGlobal)
	assert.Equal(t, []string{"ZZ-C"}, got[0].SubdivisionCodes)
	assert.Equal(t, "optional", got[0].HolidayType)
}

func TestHolidayRepo_BulkUpsertRollsBackOnError(t *testing.T) {
	f := newHolidayFixture(t)

	good := f.holiday(time.May, 1, "Good", true)
	bad := f.holiday(time.May, 2, "Bad", true)
	bad.CountryCode = "TOOLONG" // CHAR(2) rejects it

	err := f.repo.BulkUpsert(f.ctx, []models.PublicHoliday{good, bad})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `upsert holiday "Bad"`)

	has, err := f.repo.HasDataForYear(f.ctx, f.year, testCountry)
	require.NoError(t, err)
	assert.False(t, has, "the first row must be rolled back with the failing batch")
}

func TestHolidayRepo_ListByDateRange(t *testing.T) {
	f := newHolidayFixture(t)

	require.NoError(t, f.repo.BulkUpsert(f.ctx, []models.PublicHoliday{
		f.holiday(time.March, 10, "Late March", true),
		f.holiday(time.January, 1, "New Year", true),
		f.holiday(time.February, 15, "Regional A", false, "ZZ-A"),
		f.holiday(time.February, 20, "Regional B", false, "ZZ-B"),
		f.holiday(time.June, 1, "Out Of Range", true),
	}))
	// Same date in another country must stay invisible.
	other := f.holiday(time.January, 1, "Other Country", true)
	other.CountryCode = "ZY"
	require.NoError(t, f.repo.BulkUpsert(f.ctx, []models.PublicHoliday{other}))
	t.Cleanup(func() {
		_, err := f.pool.Exec(context.Background(), `DELETE FROM public_holidays WHERE id = $1`, other.ID)
		require.NoError(t, err)
	})

	start, end := f.day(time.January, 1), f.day(time.March, 10)

	all, err := f.repo.ListByDateRange(f.ctx, start, end, testCountry, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"New Year", "Regional A", "Regional B", "Late March"}, names(all),
		"ordered by date, bounds inclusive, country-scoped, no subdivision filter")

	sub := "ZZ-A"
	filtered, err := f.repo.ListByDateRange(f.ctx, start, end, testCountry, &sub)
	require.NoError(t, err)
	assert.Equal(t, []string{"New Year", "Regional A", "Late March"}, names(filtered),
		"global holidays plus the requested subdivision only")

	none := "ZZ-Z"
	onlyGlobal, err := f.repo.ListByDateRange(f.ctx, start, end, testCountry, &none)
	require.NoError(t, err)
	assert.Equal(t, []string{"New Year", "Late March"}, names(onlyGlobal))

	empty, err := f.repo.ListByDateRange(f.ctx, f.day(time.July, 1), f.day(time.July, 31), testCountry, nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestHolidayRepo_HasDataForYearAndDeleteByYear(t *testing.T) {
	f := newHolidayFixture(t)

	require.NoError(t, f.repo.BulkUpsert(f.ctx, []models.PublicHoliday{
		f.holiday(time.January, 1, "New Year", true),
		f.holiday(time.May, 1, "Labour Day", true),
	}))

	has, err := f.repo.HasDataForYear(f.ctx, f.year, testCountry)
	require.NoError(t, err)
	assert.True(t, has)

	hasOtherCountry, err := f.repo.HasDataForYear(f.ctx, f.year, "ZY")
	require.NoError(t, err)
	assert.False(t, hasOtherCountry)

	hasOtherYear, err := f.repo.HasDataForYear(f.ctx, f.year+1000, testCountry)
	require.NoError(t, err)
	assert.False(t, hasOtherYear)

	// Delete scoped to another country leaves the rows alone.
	require.NoError(t, f.repo.DeleteByYear(f.ctx, f.year, "ZY"))
	has, err = f.repo.HasDataForYear(f.ctx, f.year, testCountry)
	require.NoError(t, err)
	assert.True(t, has)

	require.NoError(t, f.repo.DeleteByYear(f.ctx, f.year, testCountry))
	has, err = f.repo.HasDataForYear(f.ctx, f.year, testCountry)
	require.NoError(t, err)
	assert.False(t, has)

	// Deleting nothing is not an error.
	require.NoError(t, f.repo.DeleteByYear(f.ctx, f.year, testCountry))
}

func TestHolidayRepo_ErrorPathsOnClosedPool(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	repo := holiday.NewPostgresRepository(pool)
	pool.Close()

	ctx := context.Background()
	year := nextYear()
	d := time.Date(year, time.May, 1, 0, 0, 0, 0, time.UTC)

	err := repo.BulkUpsert(ctx, []models.PublicHoliday{{ID: uuid.New(), Date: d, Name: "x", CountryCode: testCountry, Year: year}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "begin tx")

	_, err = repo.ListByDateRange(ctx, d, d, testCountry, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list holidays")

	_, err = repo.HasDataForYear(ctx, year, testCountry)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "check holiday data")

	err = repo.DeleteByYear(ctx, year, testCountry)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete holidays")
}

// Service on the real repository: seeds from a fake fetcher, then serves from the DB.
type fakeFetcher struct {
	calls int
	data  []holiday.NagerHoliday
}

func (ff *fakeFetcher) FetchHolidays(_ context.Context, _ int, _ string) ([]holiday.NagerHoliday, error) {
	ff.calls++
	return ff.data, nil
}

func TestHolidayService_SeedsOnceAndServesFromDB(t *testing.T) {
	f := newHolidayFixture(t)

	// validateInput only admits DE/AT/CH and 2000-2100, so the service runs on DE rows
	// in a far-future year that no real seed touches; they are removed before and after.
	year := 2090 + int(yearCounter.Load()%9)
	_, err := f.pool.Exec(f.ctx, `DELETE FROM public_holidays WHERE year = $1 AND country_code = 'DE'`, year)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, delErr := f.pool.Exec(context.Background(), `DELETE FROM public_holidays WHERE year = $1 AND country_code = 'DE'`, year)
		require.NoError(t, delErr)
	})

	ff := &fakeFetcher{data: []holiday.NagerHoliday{
		{Date: time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02"), Name: "New Year's Day", LocalName: "Neujahr", CountryCode: "DE", Global: true, Types: []string{"Public"}},
		{Date: time.Date(year, time.January, 6, 0, 0, 0, 0, time.UTC).Format("2006-01-02"), Name: "Epiphany", LocalName: "Heilige Drei Könige", CountryCode: "DE", Global: false, Counties: []string{"DE-BW", "DE-BY"}, Types: []string{"Public"}},
	}}
	svc := holiday.NewService(f.repo, ff)

	start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(year, time.January, 31, 0, 0, 0, 0, time.UTC)

	got, err := svc.ListHolidays(f.ctx, start, end, "DE", nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"New Year's Day", "Epiphany"}, names(got))
	assert.Equal(t, 1, ff.calls)

	bw := "DE-BW"
	withSub, err := svc.ListHolidays(f.ctx, start, end, "DE", &bw)
	require.NoError(t, err)
	assert.Len(t, withSub, 2)

	nrw := "DE-NW"
	withoutSub, err := svc.ListHolidays(f.ctx, start, end, "DE", &nrw)
	require.NoError(t, err)
	assert.Equal(t, []string{"New Year's Day"}, names(withoutSub))

	assert.Equal(t, 1, ff.calls, "data for the year exists now: no second fetch")
}
