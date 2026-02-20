package action

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	calendarv1 "github.com/kmuhub/kmuhub/proto/calendar/v1"
)

// CreateCalendarEventAction creates a calendar event via the Calendar gRPC service.
type CreateCalendarEventAction struct {
	client calendarv1.CalendarServiceClient
}

// NewCreateCalendarEventAction creates a new CreateCalendarEventAction.
func NewCreateCalendarEventAction(client calendarv1.CalendarServiceClient) *CreateCalendarEventAction {
	return &CreateCalendarEventAction{client: client}
}

func (a *CreateCalendarEventAction) Type() string { return "calendar.create_event" }

type createCalendarEventConfig struct {
	CalendarID  string `json:"calendar_id"`  // template
	Title       string `json:"title"`        // template
	Description string `json:"description"`  // template (optional)
	IsAllDay    bool   `json:"is_all_day"`   // boolean
	StartTime   string `json:"start_time"`   // template: RFC3339 or relative like "+1h"
	EndTime     string `json:"end_time"`     // template: RFC3339 or relative like "+2h"
	CreatedBy   string `json:"created_by"`   // template: user ID
}

func (a *CreateCalendarEventAction) Execute(ctx context.Context, config json.RawMessage, env map[string]any) (ActionResult, error) {
	if a.client == nil {
		return ActionResult{Success: false, Error: "Calendar service unavailable"}, fmt.Errorf("Calendar gRPC client is nil")
	}

	var cfg createCalendarEventConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return ActionResult{Success: false, Error: "invalid config: " + err.Error()}, err
	}

	calendarID := resolveTemplate(cfg.CalendarID, env)
	title := resolveTemplate(cfg.Title, env)
	description := resolveTemplate(cfg.Description, env)
	createdBy := resolveTemplate(cfg.CreatedBy, env)
	startTimeStr := resolveTemplate(cfg.StartTime, env)
	endTimeStr := resolveTemplate(cfg.EndTime, env)

	startTime := parseTimeOrRelative(startTimeStr)
	endTime := parseTimeOrRelative(endTimeStr)

	req := &calendarv1.CreateEventRequest{
		CalendarId: calendarID,
		Title:      title,
		IsAllDay:   cfg.IsAllDay,
		StartTime:  timestamppb.New(startTime),
		EndTime:    timestamppb.New(endTime),
		CreatedBy:  createdBy,
	}

	if description != "" {
		req.Description = &description
	}

	resp, err := a.client.CreateEvent(ctx, req)
	if err != nil {
		return ActionResult{Success: false, Error: err.Error()}, err
	}

	return ActionResult{
		Success: true,
		Output: map[string]any{
			"event_id": resp.GetEvent().GetId(),
		},
	}, nil
}

// parseTimeOrRelative parses a time string as RFC3339 or as a relative offset
// like "+1h", "+30m", "+7d". Falls back to time.Now() on parse failure.
func parseTimeOrRelative(s string) time.Time {
	// Try RFC3339 first
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}

	// Try relative offset: +Nh, +Nm, +Nd
	if len(s) > 1 && s[0] == '+' {
		unit := s[len(s)-1]
		numStr := s[1 : len(s)-1]
		n, err := strconv.Atoi(numStr)
		if err == nil {
			now := time.Now()
			switch unit {
			case 'h':
				return now.Add(time.Duration(n) * time.Hour)
			case 'm':
				return now.Add(time.Duration(n) * time.Minute)
			case 'd':
				return now.AddDate(0, 0, n)
			}
		}
	}

	return time.Now()
}

// CreateCalendarEventDefinition returns the action definition for the catalog.
func CreateCalendarEventDefinition() *ActionDefinition {
	return &ActionDefinition{
		Type:        "calendar.create_event",
		Module:      "calendar",
		Name:        "Kalenderereignis erstellen",
		Description: "Erstellt ein neues Ereignis im Kalender",
		Params: []ConfigParam{
			{Key: "calendar_id", Label: "Kalender-ID", Type: "template", Required: true},
			{Key: "title", Label: "Titel", Type: "template", Required: true},
			{Key: "description", Label: "Beschreibung", Type: "template", Required: false},
			{Key: "is_all_day", Label: "Ganztaegig", Type: "boolean", Required: false, Default: false},
			{Key: "start_time", Label: "Startzeit", Type: "template", Required: true},
			{Key: "end_time", Label: "Endzeit", Type: "template", Required: true},
			{Key: "created_by", Label: "Erstellt von", Type: "template", Required: true},
		},
		OutputFields: []OutputField{
			{Key: "event_id", Label: "Ereignis-ID", Type: "string"},
		},
	}
}
