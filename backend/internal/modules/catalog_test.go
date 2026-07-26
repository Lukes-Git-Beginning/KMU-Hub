package modules_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/featureflag"
	"github.com/kmuhub/kmuhub/internal/modules"
)

// A flag key that no longer exists in the registry would silently resolve to
// false, and the module would be permanently unactivatable with no error
// anywhere. This test is the only thing standing between a flag rename and that.
func TestCatalog_FlagKeysExistInRegistry(t *testing.T) {
	known := featureflag.NewRegistry().All()

	for _, m := range modules.Catalog {
		if m.FlagKey == "" {
			continue
		}
		_, ok := known[m.FlagKey]
		assert.True(t, ok, "module %s references unknown feature flag %s", m.ID, m.FlagKey)
	}
}

func TestCatalog_IDsAreUniqueAndGrouped(t *testing.T) {
	validGroups := map[modules.Group]bool{
		modules.GroupCore: true, modules.GroupComm: true, modules.GroupTeam: true,
		modules.GroupIndustry: true, modules.GroupTools: true,
	}

	seen := make(map[string]bool, len(modules.Catalog))
	for _, m := range modules.Catalog {
		require.NotEmpty(t, m.ID)
		assert.False(t, seen[m.ID], "duplicate module id %s", m.ID)
		seen[m.ID] = true
		assert.True(t, validGroups[m.Group], "module %s has unknown group %q", m.ID, m.Group)
	}
}

func TestGet(t *testing.T) {
	m, ok := modules.Get("finance")
	require.True(t, ok)
	assert.Equal(t, modules.GroupCore, m.Group)
	assert.Equal(t, "modules.buchhaltung", modules.FlagKey("finance"))

	_, ok = modules.Get("not-a-module")
	assert.False(t, ok)
	assert.Empty(t, modules.FlagKey("not-a-module"))
}
