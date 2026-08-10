package caldav

import (
	"testing"
	"time"

	ical "github.com/emersion/go-ical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func childByName(comp *ical.Component, name string) *ical.Component {
	for _, c := range comp.Children {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func TestGenerateVTimezone_EuropeBerlin_StandardAndDaylightOffsets(t *testing.T) {
	comp, err := GenerateVTimezone("Europe/Berlin")
	require.NoError(t, err)

	assert.Equal(t, ical.CompTimezone, comp.Name)
	assert.Equal(t, "Europe/Berlin", comp.Props.Get(ical.PropTimezoneID).Value)
	require.Len(t, comp.Children, 2)

	standard := childByName(comp, ical.CompTimezoneStandard)
	require.NotNil(t, standard, "STANDARD subcomponent must be present")
	assert.Equal(t, "19701025T030000", standard.Props.Get("DTSTART").Value)
	assert.Equal(t, "FREQ=YEARLY;BYDAY=-1SU;BYMONTH=10", standard.Props.Get(ical.PropRecurrenceRule).Value)
	assert.Equal(t, "+0200", standard.Props.Get(ical.PropTimezoneOffsetFrom).Value)
	assert.Equal(t, "+0100", standard.Props.Get(ical.PropTimezoneOffsetTo).Value)
	assert.Equal(t, "CET", standard.Props.Get(ical.PropTimezoneName).Value)

	daylight := childByName(comp, ical.CompTimezoneDaylight)
	require.NotNil(t, daylight, "DAYLIGHT subcomponent must be present")
	assert.Equal(t, "19700329T020000", daylight.Props.Get("DTSTART").Value)
	assert.Equal(t, "FREQ=YEARLY;BYDAY=-1SU;BYMONTH=3", daylight.Props.Get(ical.PropRecurrenceRule).Value)
	assert.Equal(t, "+0100", daylight.Props.Get(ical.PropTimezoneOffsetFrom).Value)
	assert.Equal(t, "+0200", daylight.Props.Get(ical.PropTimezoneOffsetTo).Value)
	assert.Equal(t, "CEST", daylight.Props.Get(ical.PropTimezoneName).Value)
}

func TestGenerateVTimezone_DACHAliases_SameShapeAsBerlin(t *testing.T) {
	for _, tz := range []string{"Europe/Zurich", "Europe/Vienna"} {
		t.Run(tz, func(t *testing.T) {
			comp, err := GenerateVTimezone(tz)
			require.NoError(t, err)

			assert.Equal(t, tz, comp.Props.Get(ical.PropTimezoneID).Value)
			require.Len(t, comp.Children, 2)

			standard := childByName(comp, ical.CompTimezoneStandard)
			require.NotNil(t, standard)
			assert.Equal(t, "CET", standard.Props.Get(ical.PropTimezoneName).Value)

			daylight := childByName(comp, ical.CompTimezoneDaylight)
			require.NotNil(t, daylight)
			assert.Equal(t, "CEST", daylight.Props.Get(ical.PropTimezoneName).Value)
		})
	}
}

// TestGenerateVTimezone_Minimal_FixedOffsetZone covers a non-DACH timezone
// with no DST rule (JST is UTC+9 year-round), so the expected offset is
// deterministic regardless of when the test runs.
func TestGenerateVTimezone_Minimal_FixedOffsetZone(t *testing.T) {
	comp, err := GenerateVTimezone("Asia/Tokyo")
	require.NoError(t, err)

	assert.Equal(t, ical.CompTimezone, comp.Name)
	assert.Equal(t, "Asia/Tokyo", comp.Props.Get(ical.PropTimezoneID).Value)
	require.Len(t, comp.Children, 1, "minimal VTIMEZONE has only a STANDARD child")

	standard := comp.Children[0]
	assert.Equal(t, ical.CompTimezoneStandard, standard.Name)
	assert.Equal(t, "19700101T000000", standard.Props.Get("DTSTART").Value)
	assert.Equal(t, "+0900", standard.Props.Get(ical.PropTimezoneOffsetFrom).Value)
	assert.Equal(t, "+0900", standard.Props.Get(ical.PropTimezoneOffsetTo).Value)
	assert.Equal(t, "Asia/Tokyo", standard.Props.Get(ical.PropTimezoneName).Value)
}

func TestGenerateVTimezone_InvalidTimezone_ReturnsError(t *testing.T) {
	comp, err := GenerateVTimezone("Not/A_Real_Zone")

	require.Error(t, err)
	assert.Nil(t, comp)
}

// TestGenerateVTimezone_Caching_ReturnsSamePointer proves the sync.Map cache
// is actually consulted on the second call, not just populated and ignored.
// Uses a timezone name not touched by any other test in this file to avoid
// cross-test interference through the shared package-level cache.
func TestGenerateVTimezone_Caching_ReturnsSamePointer(t *testing.T) {
	first, err := GenerateVTimezone("Asia/Kolkata")
	require.NoError(t, err)

	second, err := GenerateVTimezone("Asia/Kolkata")
	require.NoError(t, err)

	assert.Same(t, first, second, "second call must return the cached component, not rebuild it")
}

func TestFormatUTCOffset(t *testing.T) {
	tests := []struct {
		name    string
		seconds int
		want    string
	}{
		{"zero", 0, "+0000"},
		{"one hour east", 3600, "+0100"},
		{"one hour west", -3600, "-0100"},
		{"five hours west", -18000, "-0500"},
		{"half hour east (India)", 19800, "+0530"},
		{"two hours east", 7200, "+0200"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, formatUTCOffset(tc.seconds))
		})
	}
}

// TestBuildMinimalTimezone_MatchesRuntimeOffset cross-checks
// buildMinimalTimezone's output directly against time.Now().In(loc).Zone()
// for a DST-observing zone, so the assertion holds no matter what time of
// year the suite runs.
func TestBuildMinimalTimezone_MatchesRuntimeOffset(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	comp := buildMinimalTimezone("America/New_York", loc)

	_, wantOffset := time.Now().In(loc).Zone()
	standard := comp.Children[0]
	assert.Equal(t, formatUTCOffset(wantOffset), standard.Props.Get(ical.PropTimezoneOffsetFrom).Value)
	assert.Equal(t, formatUTCOffset(wantOffset), standard.Props.Get(ical.PropTimezoneOffsetTo).Value)
}
