package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/vermietung"
	vermietungv1 "github.com/kmuhub/kmuhub/proto/vermietung/v1"
)

// A legacy/pre-migration inspection row (or any repository call that bypasses
// Service.CreateInspection's normalization) has Checklist == nil. The wire
// response must still carry an empty list, never JSON null.
func TestChecklistToProto_NilBecomesEmptySlice(t *testing.T) {
	proto := checklistToProto(nil)

	require.NotNil(t, proto)
	assert.Empty(t, proto)
}

func TestChecklistToProto_RoundTrip(t *testing.T) {
	items := []vermietung.ChecklistItem{
		{Label: "Windschutzscheibe", Condition: "intakt"},
		{Label: "Reifen", Condition: "beschaedigt", Remark: "vorne links"},
	}

	proto := checklistToProto(items)
	require.Len(t, proto, 2)
	assert.Equal(t, "Windschutzscheibe", proto[0].GetLabel())
	assert.Equal(t, "vorne links", proto[1].GetRemark())

	back := checklistFromProto(proto)
	assert.Equal(t, items, back)
}

func TestChecklistFromProto_EmptyStaysNil(t *testing.T) {
	// nil/empty on the wire means "no checklist in this request" — must stay
	// nil, not become an empty slice, so UpdateInspectionInput.Checklist == nil
	// is distinguishable from an explicit empty replacement.
	assert.Nil(t, checklistFromProto(nil))
	assert.Nil(t, checklistFromProto([]*vermietungv1.ChecklistItem{}))
}
