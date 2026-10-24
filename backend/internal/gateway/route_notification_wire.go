package gateway

import (
	"log/slog"
	"net/http"

	"github.com/kmuhub/kmuhub/internal/server/response"
	notificationv1 "github.com/kmuhub/kmuhub/proto/notification/v1"
)

// The desktop settings read booleans with `pref?.in_app ?? eventType.default_in_app ?? true`,
// so a field protojson drops because it is false makes the checkbox jump back on after
// the user switched it off. Every preference and event-type object therefore carries its
// booleans explicitly.

const (
	defaultQuietStart    = "22:00"
	defaultQuietEnd      = "07:00"
	defaultQuietTimezone = "Europe/Berlin"
)

// desktopDaysToISO converts the desktop weekday mask (0=Sunday..6=Saturday) to the
// service's ISO weekdays (1=Monday..7=Sunday). ok is false for a value outside 0..6.
func desktopDaysToISO(days []int) (iso []int32, ok bool) {
	iso = make([]int32, 0, len(days))
	for _, d := range days {
		if d < 0 || d > 6 {
			return nil, false
		}
		if d == 0 {
			d = 7
		}
		iso = append(iso, int32(d))
	}
	return iso, true
}

// isoDaysToDesktop is the inverse of desktopDaysToISO.
func isoDaysToDesktop(iso []int32) []int {
	days := make([]int, 0, len(iso))
	for _, d := range iso {
		if d == 7 {
			d = 0
		}
		days = append(days, int(d))
	}
	return days
}

// quietHoursObject renders a quiet-hours record with both vocabularies: the service's
// (`days_of_week` ISO, `enabled`) and the desktop's (`days` 0=Sunday, `is_active`).
// A user without a record gets the defaults the settings form starts from, with the
// schedule switched off.
func quietHoursObject(qh *notificationv1.QuietHoursInfo) (jsonObject, error) {
	if qh == nil || qh.GetStartTime() == "" {
		qh = &notificationv1.QuietHoursInfo{
			StartTime:  defaultQuietStart,
			EndTime:    defaultQuietEnd,
			Timezone:   defaultQuietTimezone,
			DaysOfWeek: []int32{1, 2, 3, 4, 5, 6, 7},
		}
	}
	obj, err := protoObject(qh)
	if err != nil {
		return nil, err
	}
	setJSON(obj, "days_of_week", append([]int32{}, qh.GetDaysOfWeek()...))
	setJSON(obj, "days", isoDaysToDesktop(qh.GetDaysOfWeek()))
	setJSON(obj, "enabled", qh.GetEnabled())
	setJSON(obj, "is_active", qh.GetEnabled())
	setJSON(obj, "manual_dnd", qh.GetManualDnd())
	return obj, nil
}

func preferenceObject(p *notificationv1.NotificationPreferenceInfo) (jsonObject, error) {
	obj, err := protoObject(p)
	if err != nil {
		return nil, err
	}
	setJSON(obj, "in_app", p.GetInApp())
	setJSON(obj, "desktop_push", p.GetDesktopPush())
	setJSON(obj, "email", p.GetEmail())
	setJSON(obj, "sms", p.GetSms())
	setJSON(obj, "sound", p.GetSound())
	return obj, nil
}

func eventTypeObject(et *notificationv1.EventTypeInfo) (jsonObject, error) {
	obj, err := protoObject(et)
	if err != nil {
		return nil, err
	}
	setJSON(obj, "default_in_app", et.GetDefaultInApp())
	setJSON(obj, "default_desktop_push", et.GetDefaultDesktopPush())
	setJSON(obj, "default_sound", et.GetDefaultSound())
	return obj, nil
}

// writeNotificationEnvelope writes {key: value}; a marshal failure is a 500.
func writeNotificationEnvelope(w http.ResponseWriter, status int, key string, value any, err error) {
	if err != nil {
		slog.Error("notification response marshal failed", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.JSON(w, status, map[string]any{key: value})
}

func writeNotificationList[T any](w http.ResponseWriter, key string, items []T, fn func(T) (jsonObject, error)) {
	out := make([]jsonObject, 0, len(items))
	for _, it := range items {
		obj, err := fn(it)
		if err != nil {
			writeNotificationEnvelope(w, http.StatusOK, key, nil, err)
			return
		}
		out = append(out, obj)
	}
	writeNotificationEnvelope(w, http.StatusOK, key, out, nil)
}
