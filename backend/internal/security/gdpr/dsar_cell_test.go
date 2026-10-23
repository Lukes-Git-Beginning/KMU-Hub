package gdpr

import (
	"math/big"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// pgx returns NUMERIC and UUID columns from rows.Values() as pgtype.Numeric and
// [16]byte; the DSAR must show them as a number and a uuid, not as Go dumps.
func TestDSARCell_RendersPgxValuesReadably(t *testing.T) {
	id := uuid.MustParse("6f1c2a4e-3b5d-4c7e-9f00-112233445566")
	at := time.Date(2026, 10, 23, 9, 30, 0, 0, time.UTC)

	cases := []struct {
		name string
		in   any
		want string
	}{
		{"numeric", pgtype.Numeric{Int: big.NewInt(500), Exp: -2, Valid: true}, "5.00"},
		{"null numeric", pgtype.Numeric{}, ""},
		{"uuid", [16]byte(id), id.String()},
		{"nil", nil, ""},
		{"time", at, "2026-10-23 09:30"},
		{"true", true, "ja"},
		{"false", false, "nein"},
		{"text", "Mahnstufe 2", "Mahnstufe 2"},
		{"int", int64(42), "42"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := dsarCell(tc.in); got != tc.want {
				t.Errorf("dsarCell(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
