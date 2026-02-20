package caldav

import (
	"fmt"
	"sync"
	"time"

	ical "github.com/emersion/go-ical"
)

// vtimezoneCache caches generated VTIMEZONE components for reuse.
var vtimezoneCache sync.Map

// GenerateVTimezone creates a VTIMEZONE component for the given IANA timezone name.
// For Europe/Berlin (Deutschland-First), a full VTIMEZONE with STANDARD and DAYLIGHT
// subcomponents is generated with proper CET/CEST transition rules. For other timezones,
// a minimal VTIMEZONE with the name and current UTC offset is generated.
// Results are cached in a sync.Map for performance.
func GenerateVTimezone(tzName string) (*ical.Component, error) {
	// Check cache first
	if cached, ok := vtimezoneCache.Load(tzName); ok {
		return cached.(*ical.Component), nil
	}

	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return nil, fmt.Errorf("load timezone %q: %w", tzName, err)
	}

	var comp *ical.Component

	switch tzName {
	case "Europe/Berlin", "Europe/Zurich", "Europe/Vienna":
		// Full VTIMEZONE for DACH region (all use CET/CEST)
		comp = buildDACHTimezone(tzName)
	default:
		comp = buildMinimalTimezone(tzName, loc)
	}

	vtimezoneCache.Store(tzName, comp)
	return comp, nil
}

// buildDACHTimezone creates a full VTIMEZONE for CET/CEST timezones.
// STANDARD: Last Sunday of October at 03:00 UTC -> CET (UTC+1)
// DAYLIGHT: Last Sunday of March at 02:00 UTC -> CEST (UTC+2)
func buildDACHTimezone(tzName string) *ical.Component {
	vtimezone := ical.NewComponent(ical.CompTimezone)
	tzidProp := ical.NewProp(ical.PropTimezoneID)
	tzidProp.Value = tzName
	vtimezone.Props.Set(tzidProp)

	// STANDARD subcomponent (CET: UTC+1, from last Sunday of October)
	standard := ical.NewComponent(ical.CompTimezoneStandard)
	setPropValue(standard, "DTSTART", "19701025T030000")
	setPropValue(standard, ical.PropRecurrenceRule, "FREQ=YEARLY;BYDAY=-1SU;BYMONTH=10")
	setPropValue(standard, ical.PropTimezoneOffsetFrom, "+0200")
	setPropValue(standard, ical.PropTimezoneOffsetTo, "+0100")
	setPropValue(standard, ical.PropTimezoneName, "CET")
	vtimezone.Children = append(vtimezone.Children, standard)

	// DAYLIGHT subcomponent (CEST: UTC+2, from last Sunday of March)
	daylight := ical.NewComponent(ical.CompTimezoneDaylight)
	setPropValue(daylight, "DTSTART", "19700329T020000")
	setPropValue(daylight, ical.PropRecurrenceRule, "FREQ=YEARLY;BYDAY=-1SU;BYMONTH=3")
	setPropValue(daylight, ical.PropTimezoneOffsetFrom, "+0100")
	setPropValue(daylight, ical.PropTimezoneOffsetTo, "+0200")
	setPropValue(daylight, ical.PropTimezoneName, "CEST")
	vtimezone.Children = append(vtimezone.Children, daylight)

	return vtimezone
}

// buildMinimalTimezone creates a minimal VTIMEZONE with the timezone name
// and a STANDARD subcomponent using the current UTC offset.
func buildMinimalTimezone(tzName string, loc *time.Location) *ical.Component {
	vtimezone := ical.NewComponent(ical.CompTimezone)
	tzidProp := ical.NewProp(ical.PropTimezoneID)
	tzidProp.Value = tzName
	vtimezone.Props.Set(tzidProp)

	// Use the current offset for a minimal STANDARD subcomponent
	_, offset := time.Now().In(loc).Zone()
	offsetStr := formatUTCOffset(offset)

	standard := ical.NewComponent(ical.CompTimezoneStandard)
	setPropValue(standard, "DTSTART", "19700101T000000")
	setPropValue(standard, ical.PropTimezoneOffsetFrom, offsetStr)
	setPropValue(standard, ical.PropTimezoneOffsetTo, offsetStr)
	setPropValue(standard, ical.PropTimezoneName, tzName)
	vtimezone.Children = append(vtimezone.Children, standard)

	return vtimezone
}

// setPropValue sets a property on a component with a raw string value.
func setPropValue(comp *ical.Component, name, value string) {
	prop := ical.NewProp(name)
	prop.Value = value
	comp.Props.Set(prop)
}

// formatUTCOffset formats seconds east of UTC as an iCalendar UTC offset string.
// Example: 3600 -> "+0100", -18000 -> "-0500"
func formatUTCOffset(seconds int) string {
	sign := "+"
	if seconds < 0 {
		sign = "-"
		seconds = -seconds
	}
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	return fmt.Sprintf("%s%02d%02d", sign, hours, minutes)
}
