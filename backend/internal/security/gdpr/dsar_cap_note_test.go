package gdpr

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// Most DSAR builders stop at LIMIT dsarMaxRows without knowing whether more
// rows exist; the admin must see that the list may be incomplete.
func TestMarkCappedModules_NotesOnlyModulesAtTheCap(t *testing.T) {
	rows := func(n int) []DSARRecord {
		out := make([]DSARRecord, n)
		for i := range out {
			out[i] = DSARRecord{Fields: []DSARField{{Key: "A", Value: "x"}, {Key: "B", Value: "y"}}}
		}
		return out
	}
	persons := []DSARPerson{{Modules: []DSARModule{
		{Module: "full", Columns: []string{"A", "B"}, Records: rows(dsarMaxRows)},
		{Module: "short", Columns: []string{"A", "B"}, Records: rows(dsarMaxRows - 1)},
		{Module: "exact", Columns: []string{"A", "B"}, Records: rows(dsarMaxRows + 1), capExact: true},
	}}}

	markCappedModules(persons)

	mods := persons[0].Modules
	require.Len(t, mods[0].Records, dsarMaxRows+1)
	note := mods[0].Records[dsarMaxRows].Fields
	require.Len(t, note, 2, "the note fills every column")
	assert.Contains(t, note[0].Value, "es können weitere existieren")
	assert.Equal(t, "B", note[1].Key)
	assert.Len(t, mods[1].Records, dsarMaxRows-1, "a list below the cap is complete")
	assert.Len(t, mods[2].Records, dsarMaxRows+1, "a builder that states its own cut is left alone")
}

// previewSQL must mark a cut value and leave a short one untouched.
func TestPreviewSQL_MarksOnlyCutValues_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	long := strings.Repeat("a", jsonPreviewChars+1)
	var cut, short string
	require.NoError(t, pool.QueryRow(context.Background(),
		"SELECT "+previewSQL("$1::text")+", "+previewSQL("$2::text"), long, "kurz").Scan(&cut, &short))

	assert.Equal(t, strings.Repeat("a", jsonPreviewChars)+" …", cut)
	assert.Equal(t, "kurz", short)
}
