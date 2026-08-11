package action

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
	calendarv1 "github.com/kmuhub/kmuhub/proto/calendar/v1"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

// ─── Fakes ──────────────────────────────────────────────────────────────────
//
// Each fake embeds the generated *ServiceClient interface (nil at rest, so any
// unoverridden method panics loudly rather than silently succeeding) and
// overrides only the single method the action under test calls.

type fakeCalendarClient struct {
	calendarv1.CalendarServiceClient
	createEvent func(*calendarv1.CreateEventRequest) (*calendarv1.CreateEventResponse, error)
}

func (f *fakeCalendarClient) CreateEvent(_ context.Context, in *calendarv1.CreateEventRequest, _ ...grpc.CallOption) (*calendarv1.CreateEventResponse, error) {
	return f.createEvent(in)
}

type fakeFinanceClient struct {
	bizv1.FinanceServiceClient
	createInvoice func(*bizv1.CreateInvoiceRequest) (*bizv1.CreateInvoiceResponse, error)
	createDunning func(*bizv1.CreateDunningRequest) (*bizv1.CreateDunningResponse, error)
}

func (f *fakeFinanceClient) CreateInvoice(_ context.Context, in *bizv1.CreateInvoiceRequest, _ ...grpc.CallOption) (*bizv1.CreateInvoiceResponse, error) {
	return f.createInvoice(in)
}

func (f *fakeFinanceClient) CreateDunning(_ context.Context, in *bizv1.CreateDunningRequest, _ ...grpc.CallOption) (*bizv1.CreateDunningResponse, error) {
	return f.createDunning(in)
}

type fakeCRMClient struct {
	crmv1.CRMServiceClient
	moveDealToStage func(*crmv1.MoveDealToStageRequest) (*crmv1.MoveDealToStageResponse, error)
	updateDeal      func(*crmv1.UpdateDealRequest) (*crmv1.UpdateDealResponse, error)
	createContact   func(*crmv1.CreateContactRequest) (*crmv1.CreateContactResponse, error)
}

func (f *fakeCRMClient) MoveDealToStage(_ context.Context, in *crmv1.MoveDealToStageRequest, _ ...grpc.CallOption) (*crmv1.MoveDealToStageResponse, error) {
	return f.moveDealToStage(in)
}

func (f *fakeCRMClient) UpdateDeal(_ context.Context, in *crmv1.UpdateDealRequest, _ ...grpc.CallOption) (*crmv1.UpdateDealResponse, error) {
	return f.updateDeal(in)
}

func (f *fakeCRMClient) CreateContact(_ context.Context, in *crmv1.CreateContactRequest, _ ...grpc.CallOption) (*crmv1.CreateContactResponse, error) {
	return f.createContact(in)
}

// ─── Type() ─────────────────────────────────────────────────────────────────

func TestGRPCActions_Type(t *testing.T) {
	cases := []struct {
		name   string
		action ActionExecutor
		want   string
	}{
		{"calendar", NewCreateCalendarEventAction(nil), "calendar.create_event"},
		{"invoice", NewCreateInvoiceDraftAction(nil), "biz.create_invoice_draft"},
		{"dunning", NewCreateDunningAction(nil), "biz.create_dunning"},
		{"deal_field", NewUpdateDealFieldAction(nil), "crm.update_deal_field"},
		{"contact", NewCreateContactAction(nil), "crm.create_contact"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.action.Type())
		})
	}
}

// ─── nil-client guard ───────────────────────────────────────────────────────

func TestGRPCActions_NilClientFails(t *testing.T) {
	cases := []struct {
		name   string
		action ActionExecutor
	}{
		{"calendar", NewCreateCalendarEventAction(nil)},
		{"invoice", NewCreateInvoiceDraftAction(nil)},
		{"dunning", NewCreateDunningAction(nil)},
		{"deal_field", NewUpdateDealFieldAction(nil)},
		{"contact", NewCreateContactAction(nil)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := tc.action.Execute(context.Background(), json.RawMessage(`{}`), nil)
			require.Error(t, err)
			assert.False(t, res.Success)
			assert.NotEmpty(t, res.Error)
		})
	}
}

// ─── invalid config (shared json.Unmarshal codepath) ───────────────────────

func TestGRPCActions_InvalidConfig(t *testing.T) {
	action := NewCreateCalendarEventAction(&fakeCalendarClient{})
	res, err := action.Execute(context.Background(), json.RawMessage(`{"title":`), nil)
	require.Error(t, err)
	assert.False(t, res.Success)
	assert.Contains(t, res.Error, "invalid config")
}

// ─── CreateCalendarEventAction ──────────────────────────────────────────────

func TestCreateCalendarEventAction_Success(t *testing.T) {
	var gotReq *calendarv1.CreateEventRequest
	client := &fakeCalendarClient{
		createEvent: func(req *calendarv1.CreateEventRequest) (*calendarv1.CreateEventResponse, error) {
			gotReq = req
			return &calendarv1.CreateEventResponse{Event: &calendarv1.CalendarEventProto{Id: "evt-1"}}, nil
		},
	}
	action := NewCreateCalendarEventAction(client)

	cfg, err := json.Marshal(map[string]any{
		"calendar_id": "{{cal.id}}",
		"title":       "{{deal.name}} Kickoff",
		"description": "{{deal.notes}}",
		"start_time":  "2026-08-20T10:00:00Z",
		"end_time":    "2026-08-20T11:00:00Z",
		"created_by":  "{{user.id}}",
	})
	require.NoError(t, err)

	res, err := action.Execute(context.Background(), cfg, map[string]any{
		"cal":  map[string]any{"id": "cal-42"},
		"deal": map[string]any{"name": "Acme", "notes": "wichtig"},
		"user": map[string]any{"id": "u-1"},
	})

	require.NoError(t, err)
	require.True(t, res.Success, "error: %s", res.Error)
	assert.Equal(t, "evt-1", res.Output["event_id"])
	require.NotNil(t, gotReq)
	assert.Equal(t, "cal-42", gotReq.CalendarId)
	assert.Equal(t, "Acme Kickoff", gotReq.Title)
	require.NotNil(t, gotReq.Description)
	assert.Equal(t, "wichtig", *gotReq.Description)
	assert.Equal(t, "u-1", gotReq.CreatedBy)
	assert.Equal(t, time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC), gotReq.StartTime.AsTime())
}

func TestCreateCalendarEventAction_OmitsEmptyDescription(t *testing.T) {
	var gotReq *calendarv1.CreateEventRequest
	client := &fakeCalendarClient{
		createEvent: func(req *calendarv1.CreateEventRequest) (*calendarv1.CreateEventResponse, error) {
			gotReq = req
			return &calendarv1.CreateEventResponse{Event: &calendarv1.CalendarEventProto{Id: "evt-2"}}, nil
		},
	}
	action := NewCreateCalendarEventAction(client)

	cfg, err := json.Marshal(map[string]any{
		"calendar_id": "cal-1",
		"title":       "Termin",
		"start_time":  "2026-08-20T10:00:00Z",
		"end_time":    "2026-08-20T11:00:00Z",
	})
	require.NoError(t, err)

	res, err := action.Execute(context.Background(), cfg, nil)

	require.NoError(t, err)
	require.True(t, res.Success)
	require.NotNil(t, gotReq)
	assert.Nil(t, gotReq.Description, "an empty description must stay unset, not an empty pointer")
}

func TestCreateCalendarEventAction_ClientError(t *testing.T) {
	wantErr := errors.New("calendar unavailable")
	client := &fakeCalendarClient{
		createEvent: func(*calendarv1.CreateEventRequest) (*calendarv1.CreateEventResponse, error) {
			return nil, wantErr
		},
	}
	action := NewCreateCalendarEventAction(client)

	cfg, err := json.Marshal(map[string]any{"calendar_id": "c", "title": "t", "start_time": "2026-01-01T00:00:00Z", "end_time": "2026-01-01T01:00:00Z"})
	require.NoError(t, err)

	res, err := action.Execute(context.Background(), cfg, nil)

	require.Error(t, err)
	assert.Equal(t, wantErr, err)
	assert.False(t, res.Success)
	assert.Equal(t, wantErr.Error(), res.Error)
}

// ─── parseTimeOrRelative ─────────────────────────────────────────────────────

func TestParseTimeOrRelative(t *testing.T) {
	t.Run("RFC3339", func(t *testing.T) {
		got := parseTimeOrRelative("2026-08-20T10:00:00Z")
		assert.Equal(t, time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC), got)
	})

	t.Run("relative hours", func(t *testing.T) {
		before := time.Now()
		got := parseTimeOrRelative("+1h")
		assert.WithinDuration(t, before.Add(time.Hour), got, 2*time.Second)
	})

	t.Run("relative minutes", func(t *testing.T) {
		before := time.Now()
		got := parseTimeOrRelative("+30m")
		assert.WithinDuration(t, before.Add(30*time.Minute), got, 2*time.Second)
	})

	t.Run("relative days", func(t *testing.T) {
		before := time.Now()
		got := parseTimeOrRelative("+7d")
		assert.WithinDuration(t, before.AddDate(0, 0, 7), got, 2*time.Second)
	})

	t.Run("invalid format falls back to now", func(t *testing.T) {
		before := time.Now()
		got := parseTimeOrRelative("not-a-time")
		assert.WithinDuration(t, before, got, 2*time.Second)
	})
}

// ─── CreateInvoiceDraftAction ────────────────────────────────────────────────

func TestCreateInvoiceDraftAction_Success(t *testing.T) {
	var gotReq *bizv1.CreateInvoiceRequest
	client := &fakeFinanceClient{
		createInvoice: func(req *bizv1.CreateInvoiceRequest) (*bizv1.CreateInvoiceResponse, error) {
			gotReq = req
			return &bizv1.CreateInvoiceResponse{Invoice: &bizv1.Invoice{Id: "inv-1"}}, nil
		},
	}
	action := NewCreateInvoiceDraftAction(client)

	cfg, err := json.Marshal(map[string]any{
		"tenant_id":  "{{tenant.id}}",
		"created_by": "{{user.id}}",
		"notes":      "aus {{deal.name}}",
	})
	require.NoError(t, err)

	res, err := action.Execute(context.Background(), cfg, map[string]any{
		"tenant": map[string]any{"id": "t-1"},
		"user":   map[string]any{"id": "u-1"},
		"deal":   map[string]any{"name": "Acme"},
	})

	require.NoError(t, err)
	require.True(t, res.Success, "error: %s", res.Error)
	assert.Equal(t, "inv-1", res.Output["invoice_id"])
	require.NotNil(t, gotReq)
	assert.Equal(t, "t-1", gotReq.TenantId)
	assert.Equal(t, "u-1", gotReq.CreatedBy)
	assert.Equal(t, "aus Acme", gotReq.Notes)
}

func TestCreateInvoiceDraftAction_ClientError(t *testing.T) {
	wantErr := errors.New("finance unavailable")
	client := &fakeFinanceClient{
		createInvoice: func(*bizv1.CreateInvoiceRequest) (*bizv1.CreateInvoiceResponse, error) {
			return nil, wantErr
		},
	}
	action := NewCreateInvoiceDraftAction(client)

	res, err := action.Execute(context.Background(), json.RawMessage(`{}`), nil)

	require.Error(t, err)
	assert.Equal(t, wantErr, err)
	assert.False(t, res.Success)
	assert.Equal(t, wantErr.Error(), res.Error)
}

// ─── CreateDunningAction ─────────────────────────────────────────────────────

func TestCreateDunningAction_Success(t *testing.T) {
	var gotReq *bizv1.CreateDunningRequest
	client := &fakeFinanceClient{
		createDunning: func(req *bizv1.CreateDunningRequest) (*bizv1.CreateDunningResponse, error) {
			gotReq = req
			return &bizv1.CreateDunningResponse{Dunning: &bizv1.DunningRecord{Id: "dun-1"}}, nil
		},
	}
	action := NewCreateDunningAction(client)

	cfg, err := json.Marshal(map[string]any{
		"tenant_id":  "t-1",
		"invoice_id": "{{invoice.id}}",
		"level":      "{{invoice.dunning_level}}",
		"created_by": "u-1",
	})
	require.NoError(t, err)

	res, err := action.Execute(context.Background(), cfg, map[string]any{
		"invoice": map[string]any{"id": "inv-9", "dunning_level": "2"},
	})

	require.NoError(t, err)
	require.True(t, res.Success, "error: %s", res.Error)
	assert.Equal(t, "dun-1", res.Output["dunning_id"])
	require.NotNil(t, gotReq)
	assert.Equal(t, "inv-9", gotReq.InvoiceId)
	assert.Equal(t, int32(2), gotReq.Level)
}

func TestCreateDunningAction_UnparsableLevelDefaultsToOne(t *testing.T) {
	var gotReq *bizv1.CreateDunningRequest
	client := &fakeFinanceClient{
		createDunning: func(req *bizv1.CreateDunningRequest) (*bizv1.CreateDunningResponse, error) {
			gotReq = req
			return &bizv1.CreateDunningResponse{Dunning: &bizv1.DunningRecord{Id: "dun-2"}}, nil
		},
	}
	action := NewCreateDunningAction(client)

	cfg, err := json.Marshal(map[string]any{
		"tenant_id":  "t-1",
		"invoice_id": "inv-1",
		"level":      "not-a-number",
		"created_by": "u-1",
	})
	require.NoError(t, err)

	res, err := action.Execute(context.Background(), cfg, nil)

	require.NoError(t, err)
	require.True(t, res.Success)
	require.NotNil(t, gotReq)
	assert.Equal(t, int32(1), gotReq.Level, "an unparsable level string must default to 1, not zero or an error")
}

func TestCreateDunningAction_ClientError(t *testing.T) {
	wantErr := errors.New("dunning failed")
	client := &fakeFinanceClient{
		createDunning: func(*bizv1.CreateDunningRequest) (*bizv1.CreateDunningResponse, error) {
			return nil, wantErr
		},
	}
	action := NewCreateDunningAction(client)

	res, err := action.Execute(context.Background(), json.RawMessage(`{}`), nil)

	require.Error(t, err)
	assert.False(t, res.Success)
	assert.Equal(t, wantErr.Error(), res.Error)
}

// ─── UpdateDealFieldAction ───────────────────────────────────────────────────

func TestUpdateDealFieldAction_StageBranch(t *testing.T) {
	var gotReq *crmv1.MoveDealToStageRequest
	client := &fakeCRMClient{
		moveDealToStage: func(req *crmv1.MoveDealToStageRequest) (*crmv1.MoveDealToStageResponse, error) {
			gotReq = req
			return &crmv1.MoveDealToStageResponse{Deal: &crmv1.DealInfo{Id: "deal-1"}}, nil
		},
	}
	action := NewUpdateDealFieldAction(client)

	cfg, err := json.Marshal(map[string]any{
		"deal_id": "{{deal.id}}",
		"field":   "stage",
		"value":   "{{stage.id}}",
	})
	require.NoError(t, err)

	res, err := action.Execute(context.Background(), cfg, map[string]any{
		"deal":  map[string]any{"id": "deal-1"},
		"stage": map[string]any{"id": "won"},
	})

	require.NoError(t, err)
	require.True(t, res.Success, "error: %s", res.Error)
	assert.Equal(t, "deal-1", res.Output["deal_id"])
	assert.Equal(t, "stage", res.Output["updated_field"])
	require.NotNil(t, gotReq)
	assert.Equal(t, "won", gotReq.StageId)
}

func TestUpdateDealFieldAction_AssigneeBranch(t *testing.T) {
	var gotReq *crmv1.UpdateDealRequest
	client := &fakeCRMClient{
		updateDeal: func(req *crmv1.UpdateDealRequest) (*crmv1.UpdateDealResponse, error) {
			gotReq = req
			return &crmv1.UpdateDealResponse{Deal: &crmv1.DealInfo{Id: "deal-2"}}, nil
		},
	}
	action := NewUpdateDealFieldAction(client)

	cfg, err := json.Marshal(map[string]any{
		"deal_id": "deal-2",
		"field":   "assignee",
		"value":   "{{user.id}}",
	})
	require.NoError(t, err)

	res, err := action.Execute(context.Background(), cfg, map[string]any{
		"user": map[string]any{"id": "u-7"},
	})

	require.NoError(t, err)
	require.True(t, res.Success, "error: %s", res.Error)
	assert.Equal(t, "deal-2", res.Output["deal_id"])
	assert.Equal(t, "assignee", res.Output["updated_field"])
	require.NotNil(t, gotReq)
	require.NotNil(t, gotReq.OwnerId)
	assert.Equal(t, "u-7", *gotReq.OwnerId)
}

func TestUpdateDealFieldAction_UnknownFieldReturnsUnsuccessfulWithoutError(t *testing.T) {
	action := NewUpdateDealFieldAction(&fakeCRMClient{})

	cfg, err := json.Marshal(map[string]any{
		"deal_id": "deal-3",
		"field":   "bogus",
		"value":   "x",
	})
	require.NoError(t, err)

	res, err := action.Execute(context.Background(), cfg, nil)

	require.NoError(t, err, "an unsupported field is a config problem for the automation author, not a system error")
	assert.False(t, res.Success)
	assert.Contains(t, res.Error, "bogus")
}

func TestUpdateDealFieldAction_StageBranchClientError(t *testing.T) {
	wantErr := errors.New("crm unavailable")
	client := &fakeCRMClient{
		moveDealToStage: func(*crmv1.MoveDealToStageRequest) (*crmv1.MoveDealToStageResponse, error) {
			return nil, wantErr
		},
	}
	action := NewUpdateDealFieldAction(client)

	cfg, err := json.Marshal(map[string]any{"deal_id": "d", "field": "stage", "value": "won"})
	require.NoError(t, err)

	res, err := action.Execute(context.Background(), cfg, nil)

	require.Error(t, err)
	assert.False(t, res.Success)
	assert.Equal(t, wantErr.Error(), res.Error)
}

// ─── CreateContactAction ─────────────────────────────────────────────────────

func TestCreateContactAction_Success(t *testing.T) {
	var gotReq *crmv1.CreateContactRequest
	client := &fakeCRMClient{
		createContact: func(req *crmv1.CreateContactRequest) (*crmv1.CreateContactResponse, error) {
			gotReq = req
			return &crmv1.CreateContactResponse{Contact: &crmv1.ContactInfo{Id: "contact-1"}}, nil
		},
	}
	action := NewCreateContactAction(client)

	cfg, err := json.Marshal(map[string]any{
		"first_name": "{{lead.first}}",
		"last_name":  "{{lead.last}}",
		"email":      "{{lead.email}}",
	})
	require.NoError(t, err)

	res, err := action.Execute(context.Background(), cfg, map[string]any{
		"lead": map[string]any{"first": "Max", "last": "Muster", "email": "max@example.com"},
	})

	require.NoError(t, err)
	require.True(t, res.Success, "error: %s", res.Error)
	assert.Equal(t, "contact-1", res.Output["contact_id"])
	require.NotNil(t, gotReq)
	assert.Equal(t, "Max", gotReq.FirstName)
	assert.Equal(t, "Muster", gotReq.LastName)
	assert.Equal(t, "max@example.com", gotReq.Email)
}

func TestCreateContactAction_ClientError(t *testing.T) {
	wantErr := errors.New("crm unavailable")
	client := &fakeCRMClient{
		createContact: func(*crmv1.CreateContactRequest) (*crmv1.CreateContactResponse, error) {
			return nil, wantErr
		},
	}
	action := NewCreateContactAction(client)

	res, err := action.Execute(context.Background(), json.RawMessage(`{}`), nil)

	require.Error(t, err)
	assert.False(t, res.Success)
	assert.Equal(t, wantErr.Error(), res.Error)
}

// ─── Definitions (catalog agreement, same pattern as http_actions_test.go) ──

func TestGRPCActionDefinitions(t *testing.T) {
	cases := []struct {
		name string
		def  *ActionDefinition
		want string
	}{
		{"calendar", CreateCalendarEventDefinition(), (&CreateCalendarEventAction{}).Type()},
		{"invoice", CreateInvoiceDraftDefinition(), (&CreateInvoiceDraftAction{}).Type()},
		{"dunning", CreateDunningDefinition(), (&CreateDunningAction{}).Type()},
		{"deal_field", UpdateDealFieldDefinition(), (&UpdateDealFieldAction{}).Type()},
		{"contact", CreateContactDefinition(), (&CreateContactAction{}).Type()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.def.Type, "catalog entry and executor must agree")
			require.NotEmpty(t, tc.def.Params)
			require.NotEmpty(t, tc.def.OutputFields)
		})
	}
}
