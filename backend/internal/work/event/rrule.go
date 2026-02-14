package event

import (
	"fmt"
	"strings"
	"time"

	"github.com/teambition/rrule-go"
)

// ExpandRecurrence expands an RRULE string into concrete dates within a time window.
// dtstart is the original event start time. Returns dates in the event's timezone.
func ExpandRecurrence(rruleStr string, dtstart time.Time, windowStart, windowEnd time.Time) ([]time.Time, error) {
	option, err := rrule.StrToROption(rruleStr)
	if err != nil {
		return nil, fmt.Errorf("parse RRULE: %w", err)
	}

	option.Dtstart = dtstart

	r, err := rrule.NewRRule(*option)
	if err != nil {
		return nil, fmt.Errorf("create RRULE: %w", err)
	}

	dates := r.Between(windowStart, windowEnd, true)
	return dates, nil
}

// ValidateRRule checks if an RRULE string is valid RFC 5545.
func ValidateRRule(rruleStr string) error {
	_, err := rrule.StrToROption(rruleStr)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidRRule, err.Error())
	}
	return nil
}

// SetUntil modifies an RRULE string to add/replace the UNTIL parameter.
// Used for "this and future events" split.
func SetUntil(rruleStr string, until time.Time) (string, error) {
	if _, err := rrule.StrToROption(rruleStr); err != nil {
		return "", fmt.Errorf("%w: %s", ErrInvalidRRule, err.Error())
	}

	untilStr := until.UTC().Format("20060102T150405Z")

	parts := strings.Split(rruleStr, ";")
	var result []string
	found := false
	for _, part := range parts {
		if strings.HasPrefix(strings.ToUpper(part), "UNTIL=") {
			result = append(result, "UNTIL="+untilStr)
			found = true
		} else if strings.HasPrefix(strings.ToUpper(part), "COUNT=") {
			// Remove COUNT when setting UNTIL (they're mutually exclusive in RFC 5545)
			continue
		} else {
			result = append(result, part)
		}
	}
	if !found {
		result = append(result, "UNTIL="+untilStr)
	}

	return strings.Join(result, ";"), nil
}

// RemoveUntil removes the UNTIL parameter from an RRULE string.
func RemoveUntil(rruleStr string) (string, error) {
	if _, err := rrule.StrToROption(rruleStr); err != nil {
		return "", fmt.Errorf("%w: %s", ErrInvalidRRule, err.Error())
	}

	parts := strings.Split(rruleStr, ";")
	var result []string
	for _, part := range parts {
		if strings.HasPrefix(strings.ToUpper(part), "UNTIL=") {
			continue
		}
		result = append(result, part)
	}

	return strings.Join(result, ";"), nil
}
