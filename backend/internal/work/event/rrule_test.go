package event

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandRecurrence_MonthEnd_SkipsShortMonths(t *testing.T) {
	// BYMONTHDAY=31 must only fire in months that actually have a 31st —
	// RFC 5545 skips the occurrence rather than clamping it to month-end.
	dtstart := time.Date(2026, 1, 31, 10, 0, 0, 0, time.UTC)
	windowEnd := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	dates, err := ExpandRecurrence("FREQ=MONTHLY;BYMONTHDAY=31;COUNT=6", dtstart, dtstart, windowEnd)

	require.NoError(t, err)
	want := []time.Time{
		time.Date(2026, 1, 31, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 31, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 31, 10, 0, 0, 0, time.UTC),
	}
	require.Len(t, dates, len(want))
	for i, w := range want {
		assert.True(t, w.Equal(dates[i]), "index %d: want %v, got %v", i, w, dates[i])
	}
}

func TestExpandRecurrence_LeapYear_Feb29OnlyFiresOnLeapYears(t *testing.T) {
	// FREQ=YEARLY on Feb 29 must skip non-leap years entirely, not roll to Mar 1.
	dtstart := time.Date(2024, 2, 29, 9, 0, 0, 0, time.UTC)
	windowEnd := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)

	dates, err := ExpandRecurrence("FREQ=YEARLY;COUNT=4", dtstart, dtstart, windowEnd)

	require.NoError(t, err)
	want := []time.Time{
		time.Date(2024, 2, 29, 9, 0, 0, 0, time.UTC),
		time.Date(2028, 2, 29, 9, 0, 0, 0, time.UTC),
	}
	require.Len(t, dates, len(want))
	for i, w := range want {
		assert.True(t, w.Equal(dates[i]), "index %d: want %v, got %v", i, w, dates[i])
	}
}

func TestExpandRecurrence_DSTSpringForward_ShiftsPastMissingHour(t *testing.T) {
	// 2026-03-29 is the EU spring-forward date (02:00 -> 03:00 CEST in
	// Europe/Berlin). A weekly 02:30 occurrence falls inside the skipped
	// hour and must resolve to 03:30 CEST, not silently vanish or stay at
	// the (nonexistent) 02:30 CET.
	loc, err := time.LoadLocation("Europe/Berlin")
	require.NoError(t, err)
	dtstart := time.Date(2026, 3, 15, 2, 30, 0, 0, loc)
	windowEnd := time.Date(2026, 4, 15, 0, 0, 0, 0, loc)

	dates, err := ExpandRecurrence("FREQ=WEEKLY;COUNT=4", dtstart, dtstart, windowEnd)

	require.NoError(t, err)
	require.Len(t, dates, 4)

	preDST := dates[1]
	assert.Equal(t, 2, preDST.Hour())
	_, offset := preDST.Zone()
	assert.Equal(t, 3600, offset, "expected CET (+1h) before the DST switch")

	postDST := dates[2]
	assert.Equal(t, time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC).Day(), postDST.Day())
	assert.Equal(t, 3, postDST.Hour(), "02:30 falls inside the skipped hour and must shift to 03:30 CEST")
	_, offset = postDST.Zone()
	assert.Equal(t, 7200, offset, "expected CEST (+2h) after the DST switch")
}

func TestValidateRRule_EmptyString(t *testing.T) {
	err := ValidateRRule("")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidRRule)
}

func TestSetUntil_ReplacesExistingUntilAndRemovesCount(t *testing.T) {
	// SetUntil must handle a rule that already carries both UNTIL and
	// COUNT — RFC 5545 forbids the combination, so the existing UNTIL is
	// replaced and COUNT is still dropped, not left behind.
	until := time.Date(2027, 1, 15, 12, 0, 0, 0, time.UTC)

	result, err := SetUntil("FREQ=WEEKLY;UNTIL=20260101T000000Z;COUNT=5;BYDAY=MO", until)

	require.NoError(t, err)
	assert.Contains(t, result, "UNTIL=20270115T120000Z")
	assert.NotContains(t, result, "UNTIL=20260101T000000Z")
	assert.NotContains(t, result, "COUNT=")
	assert.Contains(t, result, "BYDAY=MO")
}
