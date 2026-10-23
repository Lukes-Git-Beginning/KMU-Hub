package gateway

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/server/response"
	calv1 "github.com/kmuhub/kmuhub/proto/calendar/v1"
)

// flattenExpandedEvents renders ExpandedEventProto items with the event fields on
// the element level ({id, start_time, title, calendar_name, display_color, ...}).
// The desktop types ExpandedEvent as CalendarEvent plus the extras and reads
// event.id / event.start_time directly; the raw proto JSON nests them under
// `event`, so no appointment showed up. Never returns nil, so an empty list
// serializes as [].
func flattenExpandedEvents(events []*calv1.ExpandedEventProto) ([]json.RawMessage, error) {
	out := make([]json.RawMessage, 0, len(events))
	for _, e := range events {
		flat, err := response.ProtoFlatten(e, "event")
		if err != nil {
			return nil, fmt.Errorf("flatten expanded event: %w", err)
		}
		out = append(out, flat)
	}
	return out, nil
}

// parseRangeEnd parses the upper bound of a range query. The desktop sends the
// LAST VISIBLE DAY as a bare date (YYYY-MM-DD) while the repository filters with
// start_time < end, so a bare date is read as inclusive: it becomes the start of
// the following day. Full timestamps are taken as given.
func parseRangeEnd(s string) (*timestamppb.Timestamp, error) {
	if d, err := time.Parse("2006-01-02", s); err == nil {
		return timestamppb.New(d.AddDate(0, 0, 1)), nil
	}
	return parseTimestamp(s)
}

// holidayRange resolves the holiday window: explicit start/end win (end follows
// the inclusive-date rule of the repository, `date <= end`); otherwise `year`
// (what the desktop sends) selects 1 Jan - 31 Dec. Without any of them the old
// handler filtered 1970-01-01..1970-01-01 and always answered empty.
func holidayRange(q url.Values) (start, end *timestamppb.Timestamp, err error) {
	if s := q.Get("start"); s != "" {
		if start, err = parseTimestamp(s); err != nil {
			return nil, nil, errors.New("invalid start time format")
		}
	}
	if e := q.Get("end"); e != "" {
		if end, err = parseTimestamp(e); err != nil {
			return nil, nil, errors.New("invalid end time format")
		}
	}
	if start != nil && end != nil {
		return start, end, nil
	}
	y := q.Get("year")
	if y == "" {
		return nil, nil, errors.New("year or start and end query parameters are required")
	}
	year, convErr := strconv.Atoi(y)
	if convErr != nil || year < 2000 || year > 2100 {
		return nil, nil, errors.New("invalid year")
	}
	if start == nil {
		start = timestamppb.New(time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC))
	}
	if end == nil {
		end = timestamppb.New(time.Date(year, time.December, 31, 0, 0, 0, 0, time.UTC))
	}
	return start, end, nil
}
