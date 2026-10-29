package action

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/event"
	"github.com/kmuhub/kmuhub/internal/security/safehttp"
	"github.com/kmuhub/kmuhub/internal/testutil"
	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
	workv1 "github.com/kmuhub/kmuhub/proto/work/v1"
)

type fakeEmailClient struct {
	emailv1.EmailServiceClient
	sendEmail func(*emailv1.SendEmailRequest) (*emailv1.SendEmailResponse, error)
}

func (f *fakeEmailClient) SendEmail(_ context.Context, in *emailv1.SendEmailRequest, _ ...grpc.CallOption) (*emailv1.SendEmailResponse, error) {
	return f.sendEmail(in)
}

type fakeWorkClient struct {
	workv1.WorkServiceClient
	createTask func(*workv1.CreateTaskRequest) (*workv1.CreateTaskResponse, error)
}

func (f *fakeWorkClient) CreateTask(_ context.Context, in *workv1.CreateTaskRequest, _ ...grpc.CallOption) (*workv1.CreateTaskResponse, error) {
	return f.createTask(in)
}

func mustMarshal(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

// ─── email.send ─────────────────────────────────────────────────────────────

func TestSendEmailAction_ResolvesTemplatesAndReturnsMessageID(t *testing.T) {
	var got *emailv1.SendEmailRequest
	client := &fakeEmailClient{sendEmail: func(req *emailv1.SendEmailRequest) (*emailv1.SendEmailResponse, error) {
		got = req
		return &emailv1.SendEmailResponse{Message: &emailv1.EmailMessageInfo{Id: "msg-7"}}, nil
	}}
	a := NewSendEmailAction(client)
	assert.Equal(t, "email.send", a.Type())

	env := map[string]any{
		"email_account_id": "acc-1",
		"deal":             map[string]any{"name": "Acme", "owner_email": "owner@example.com"},
	}
	res, err := a.Execute(context.Background(), mustMarshal(t, map[string]any{
		"account_id": "{{email_account_id}}",
		"to":         "{{deal.owner_email}}",
		"subject":    "Deal {{deal.name}}",
		"body":       "Hallo {{deal.name}}",
	}), env)

	require.NoError(t, err)
	assert.True(t, res.Success)
	assert.Equal(t, "msg-7", res.Output["message_id"])
	require.NotNil(t, got)
	assert.Equal(t, "acc-1", got.AccountId)
	require.Len(t, got.To, 1)
	assert.Equal(t, "owner@example.com", got.To[0].Email)
	assert.Equal(t, "Deal Acme", got.Subject)
	assert.Equal(t, "Hallo Acme", got.BodyHtml)
	assert.Equal(t, "Hallo Acme", got.BodyText)
}

func TestSendEmailAction_ClientErrorIsReported(t *testing.T) {
	wantErr := errors.New("smtp refused")
	a := NewSendEmailAction(&fakeEmailClient{sendEmail: func(*emailv1.SendEmailRequest) (*emailv1.SendEmailResponse, error) {
		return nil, wantErr
	}})
	res, err := a.Execute(context.Background(), json.RawMessage(`{"account_id":"a","to":"x@y.de"}`), nil)
	require.ErrorIs(t, err, wantErr)
	assert.False(t, res.Success)
	assert.Equal(t, "smtp refused", res.Error)
	assert.Empty(t, res.Output)
}

func TestSendEmailAction_NilClientAndBadConfig(t *testing.T) {
	res, err := NewSendEmailAction(nil).Execute(context.Background(), json.RawMessage(`{}`), nil)
	require.Error(t, err)
	assert.False(t, res.Success)
	assert.Equal(t, "email service unavailable", res.Error)

	called := false
	a := NewSendEmailAction(&fakeEmailClient{sendEmail: func(*emailv1.SendEmailRequest) (*emailv1.SendEmailResponse, error) {
		called = true
		return &emailv1.SendEmailResponse{}, nil
	}})
	res, err = a.Execute(context.Background(), json.RawMessage(`{"to":`), nil)
	require.Error(t, err)
	assert.False(t, res.Success)
	assert.True(t, strings.HasPrefix(res.Error, "invalid config: "))
	assert.False(t, called, "a broken config must not reach the email service")
}

// ─── work.create_task ───────────────────────────────────────────────────────

// These tests pin what the action maps from its config, including the creator
// the work service insists on (an empty created_by is rejected server-side).

func TestCreateTaskAction_MapsConfigToRequest(t *testing.T) {
	var got *workv1.CreateTaskRequest
	client := &fakeWorkClient{createTask: func(req *workv1.CreateTaskRequest) (*workv1.CreateTaskResponse, error) {
		got = req
		return &workv1.CreateTaskResponse{Task: &workv1.TaskProto{Id: "task-9"}}, nil
	}}
	a := NewCreateTaskAction(client)
	assert.Equal(t, "work.create_task", a.Type())

	before := time.Now()
	res, err := a.Execute(context.Background(), mustMarshal(t, map[string]any{
		"project_id":  "{{project_id}}",
		"title":       "Follow-up: {{deal.name}}",
		"assignee_id": "{{deal.owner_id}}",
		"created_by":  "{{creator}}",
		"priority":    "high",
		"due_days":    3,
	}), map[string]any{
		"creator":    "11111111-1111-4111-8111-111111111111",
		"actor_id":   "22222222-2222-4222-8222-222222222222",
		"project_id": "proj-1",
		"deal":       map[string]any{"name": "Acme", "owner_id": "user-5"},
	})

	require.NoError(t, err)
	assert.True(t, res.Success)
	assert.Equal(t, "task-9", res.Output["task_id"])
	require.NotNil(t, got)
	assert.Equal(t, "proj-1", got.ProjectId)
	assert.Equal(t, "Follow-up: Acme", got.Title)
	assert.Equal(t, "high", got.Priority)
	assert.Equal(t, "11111111-1111-4111-8111-111111111111", got.CreatedBy, "explicit created_by wins over the event actor")
	require.NotNil(t, got.AssigneeId)
	assert.Equal(t, "user-5", *got.AssigneeId)
	require.NotNil(t, got.DueDate)
	assert.WithinDuration(t, before.AddDate(0, 0, 3), got.DueDate.AsTime(), time.Minute)
}

func TestCreateTaskAction_DefaultsAndOptionalFields(t *testing.T) {
	var got *workv1.CreateTaskRequest
	client := &fakeWorkClient{createTask: func(req *workv1.CreateTaskRequest) (*workv1.CreateTaskResponse, error) {
		got = req
		return &workv1.CreateTaskResponse{}, nil
	}}
	// No priority and no due date; an assignee template that resolves to the
	// empty string behaves like no assignee.
	res, err := NewCreateTaskAction(client).Execute(context.Background(),
		json.RawMessage(`{"project_id":"p","title":"t","assignee_id":"{{missing}}","due_days":0}`),
		map[string]any{"missing": "", "actor_id": "22222222-2222-4222-8222-222222222222"})
	require.NoError(t, err)
	assert.True(t, res.Success)
	require.NotNil(t, got)
	assert.Equal(t, "normal", got.Priority)
	assert.Nil(t, got.AssigneeId)
	assert.Nil(t, got.DueDate)
	assert.Equal(t, "22222222-2222-4222-8222-222222222222", got.CreatedBy, "created_by defaults to the event actor")

	// A negative due_days is not a due date in the past.
	_, err = NewCreateTaskAction(client).Execute(context.Background(),
		json.RawMessage(`{"project_id":"p","title":"t","due_days":-2}`), map[string]any{"actor_id": "22222222-2222-4222-8222-222222222222"})
	require.NoError(t, err)
	assert.Nil(t, got.DueDate)
}

func TestCreateTaskAction_ErrorPaths(t *testing.T) {
	res, err := NewCreateTaskAction(nil).Execute(context.Background(), json.RawMessage(`{}`), nil)
	require.Error(t, err)
	assert.False(t, res.Success)
	assert.Equal(t, "work service unavailable", res.Error)

	wantErr := errors.New("project not found")
	a := NewCreateTaskAction(&fakeWorkClient{createTask: func(*workv1.CreateTaskRequest) (*workv1.CreateTaskResponse, error) {
		return nil, wantErr
	}})
	res, err = a.Execute(context.Background(), json.RawMessage(`{"project_id":"p","title":"t"}`),
		map[string]any{"actor_id": "22222222-2222-4222-8222-222222222222"})
	require.ErrorIs(t, err, wantErr)
	assert.False(t, res.Success)
	assert.Equal(t, "project not found", res.Error)

	res, err = a.Execute(context.Background(), json.RawMessage(`{"title":`), nil)
	require.Error(t, err)
	assert.False(t, res.Success)
	assert.True(t, strings.HasPrefix(res.Error, "invalid config: "))
}

// work.create_task must never reach the work service without a creator: the
// server rejects an empty or malformed created_by, which failed every run.
func TestCreateTaskAction_RejectsMissingCreator(t *testing.T) {
	called := false
	a := NewCreateTaskAction(&fakeWorkClient{createTask: func(*workv1.CreateTaskRequest) (*workv1.CreateTaskResponse, error) {
		called = true
		return &workv1.CreateTaskResponse{}, nil
	}})
	cfg := json.RawMessage(`{"project_id":"p","title":"t"}`)

	for name, env := range map[string]map[string]any{
		"no env":         nil,
		"empty actor":    {"actor_id": ""},
		"system actor":   {"actor_id": "system"},
		"nil uuid actor": {"actor_id": uuid.Nil.String()},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := a.Execute(context.Background(), cfg, env)
			require.Error(t, err)
			assert.False(t, res.Success)
			assert.Contains(t, res.Error, "created_by must be a user ID")
		})
	}
	assert.False(t, called, "a bad creator must not reach the work service")

	// An explicit created_by that resolves to garbage is rejected as well,
	// even though the event actor would have been valid.
	res, err := a.Execute(context.Background(),
		json.RawMessage(`{"project_id":"p","title":"t","created_by":"{{nope}}"}`),
		map[string]any{"actor_id": "22222222-2222-4222-8222-222222222222"})
	require.Error(t, err)
	assert.False(t, res.Success)
	assert.False(t, called)
}

// ─── notification.send ──────────────────────────────────────────────────────

// capturePublisher records every event the action publishes and fails with err.
type capturePublisher struct {
	events []models.EventPayload
	err    error
}

func (c *capturePublisher) publish(_ context.Context, p models.EventPayload) error {
	c.events = append(c.events, p)
	return c.err
}

func TestSendNotificationAction_PublishesOneEventForTheTarget(t *testing.T) {
	pub := &capturePublisher{}
	a := NewSendNotificationAction(pub.publish)
	assert.Equal(t, "notification.send", a.Type())

	tenantID, ownerID := uuid.New(), uuid.NewString()
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	res, err := a.Execute(ctx, mustMarshal(t, map[string]any{
		"target_user_id": "{{deal.owner_id}}",
		"title":          "Deal {{deal.name}}",
		"body":           "Neuer Stand",
		"deep_link":      "/crm/deals/{{deal.id}}",
	}), map[string]any{"deal": map[string]any{"owner_id": ownerID, "name": "Acme", "id": "d-7"}})

	require.NoError(t, err)
	assert.True(t, res.Success)
	require.Len(t, pub.events, 1)
	evt := pub.events[0]
	assert.Equal(t, EventTypeAutomationNotification, evt.Type)
	assert.Equal(t, event.ModuleAutomation, evt.ModuleID, "the automation consumer skips this module: no workflow loop")
	assert.Equal(t, tenantID, evt.TenantID)
	assert.Equal(t, []string{ownerID}, evt.TargetUserIDs)
	assert.Equal(t, "Deal Acme", evt.Title)
	assert.Equal(t, "Neuer Stand", evt.Body)
	assert.Equal(t, "/crm/deals/d-7", evt.DeepLink)
	assert.False(t, evt.Timestamp.IsZero())

	assert.Equal(t, ownerID, res.Output["target_user_id"])
	assert.Equal(t, evt.ResourceID, res.Output["notification_id"], "the reported id is the published event's id")
	_, perr := uuid.Parse(evt.ResourceID)
	assert.NoError(t, perr, "notification_id must be a UUID")

	res2, err := a.Execute(ctx, mustMarshal(t, map[string]any{"target_user_id": ownerID}), nil)
	require.NoError(t, err)
	assert.NotEqual(t, res.Output["notification_id"], res2.Output["notification_id"], "each execution gets its own id")
}

func TestSendNotificationAction_PublishFailureIsNoSuccess(t *testing.T) {
	pub := &capturePublisher{err: errors.New("pg_notify: connection refused")}
	res, err := NewSendNotificationAction(pub.publish).Execute(testutil.WithTenantCtx(context.Background(), uuid.New()),
		mustMarshal(t, map[string]any{"target_user_id": uuid.NewString(), "title": "t"}), nil)
	require.Error(t, err)
	assert.False(t, res.Success)
	assert.Contains(t, res.Error, "connection refused")
	assert.Len(t, pub.events, 1)
}

func TestSendNotificationAction_RejectsWithoutPublishing(t *testing.T) {
	tenantCtx := testutil.WithTenantCtx(context.Background(), uuid.New())
	cases := []struct {
		name   string
		ctx    context.Context
		target string
		pub    bool
	}{
		{"unresolved template", tenantCtx, "{{deal.owner_id}}", true},
		{"not a uuid", tenantCtx, "user-5", true},
		{"nil uuid", tenantCtx, uuid.Nil.String(), true},
		{"no tenant", context.Background(), uuid.NewString(), true},
		{"no publisher", tenantCtx, uuid.NewString(), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pub := &capturePublisher{}
			var fn EventPublisher
			if tc.pub {
				fn = pub.publish
			}
			res, err := NewSendNotificationAction(fn).Execute(tc.ctx,
				mustMarshal(t, map[string]any{"target_user_id": tc.target, "title": "t"}), map[string]any{})
			require.Error(t, err)
			assert.False(t, res.Success)
			assert.Empty(t, pub.events)
		})
	}
}

func TestSendNotificationAction_InvalidConfig(t *testing.T) {
	res, err := NewSendNotificationAction(nil).Execute(context.Background(), json.RawMessage(`{"title":`), nil)
	require.Error(t, err)
	assert.False(t, res.Success)
	assert.True(t, strings.HasPrefix(res.Error, "invalid config: "))
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "abc", truncate("abc", 3))
	assert.Equal(t, "abc...", truncate("abcd", 3))
	assert.Equal(t, "", truncate("", 5))
}

func TestSendNotificationAction_LogsOnlyAPreview(t *testing.T) {
	// slog.Default is process-wide; this test does not run in parallel.
	var buf strings.Builder
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	long := strings.Repeat("x", 150)
	pub := &capturePublisher{}
	_, err := NewSendNotificationAction(pub.publish).Execute(testutil.WithTenantCtx(context.Background(), uuid.New()),
		mustMarshal(t, map[string]any{"target_user_id": uuid.NewString(), "body": long}), nil)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), strings.Repeat("x", 100)+"...")
	assert.NotContains(t, buf.String(), strings.Repeat("x", 101))
}

// ─── Registry ───────────────────────────────────────────────────────────────

func TestActionRegistry_RegisterGetAndCatalog(t *testing.T) {
	r := NewActionRegistry()
	assert.Equal(t, 0, r.Count())
	assert.Empty(t, r.AllDefinitions())
	_, ok := r.Get("email.send")
	assert.False(t, ok)

	email := NewSendEmailAction(nil)
	r.Register(email, SendEmailDefinition())
	r.Register(NewSendNotificationAction(nil), SendNotificationDefinition())

	assert.Equal(t, 2, r.Count())
	got, ok := r.Get("email.send")
	require.True(t, ok)
	assert.Same(t, email, got)

	types := map[string]bool{}
	for _, d := range r.AllDefinitions() {
		types[d.Type] = true
	}
	assert.Equal(t, map[string]bool{"email.send": true, "notification.send": true}, types)

	// Registering the same type again replaces the executor instead of
	// growing the registry.
	other := NewSendEmailAction(&fakeEmailClient{})
	r.Register(other, SendEmailDefinition())
	assert.Equal(t, 2, r.Count())
	got, _ = r.Get("email.send")
	assert.Same(t, other, got)
}

func TestDefinitions_AgreeWithExecutors(t *testing.T) {
	cases := []struct {
		def  *ActionDefinition
		typ  string
		mod  string
		req  []string
		outs []string
	}{
		{SendEmailDefinition(), NewSendEmailAction(nil).Type(), "email", []string{"account_id", "to", "subject", "body"}, []string{"message_id"}},
		{CreateTaskDefinition(), NewCreateTaskAction(nil).Type(), "work", []string{"project_id", "title"}, []string{"task_id"}},
		{SendNotificationDefinition(), NewSendNotificationAction(nil).Type(), "notification", []string{"target_user_id", "title", "body"}, []string{"notification_id", "target_user_id"}},
	}
	for _, tc := range cases {
		t.Run(tc.typ, func(t *testing.T) {
			assert.Equal(t, tc.typ, tc.def.Type)
			assert.Equal(t, tc.mod, tc.def.Module)
			var required, outs []string
			for _, p := range tc.def.Params {
				if p.Required {
					required = append(required, p.Key)
				}
			}
			for _, o := range tc.def.OutputFields {
				outs = append(outs, o.Key)
			}
			assert.Equal(t, tc.req, required)
			assert.Equal(t, tc.outs, outs)
		})
	}
}

// ─── http_actions helpers ───────────────────────────────────────────────────

func TestValidateHeader(t *testing.T) {
	cases := []struct {
		name, key, value, wantErr string
	}{
		{"ok", "X-Token", "abc", ""},
		{"empty name", "", "v", "must not be empty"},
		{"forbidden framing header", "Content-Length", "1", "cannot be set"},
		{"space in name", "X Token", "v", "invalid character"},
		{"newline in name", "X-A\nB", "v", "invalid character"},
		{"newline in value", "X-Token", "a\r\nInjected: 1", "line break"},
		{"nul in value", "X-Token", "a\x00b", "line break"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateHeader(tc.key, tc.value)
			if tc.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
			assert.NotContains(t, err.Error(), "Injected", "the value must not be echoed into the stored error text")
		})
	}
}

func TestRequestErrorMessage(t *testing.T) {
	assert.Equal(t, "target resolves to a non-public address and was blocked",
		requestErrorMessage(fmt.Errorf("dial: %w", safehttp.ErrBlockedAddress)))
	assert.Equal(t, "redirect target uses an unsupported scheme",
		requestErrorMessage(fmt.Errorf("redirect: %w", safehttp.ErrUnsupportedScheme)))
	assert.Equal(t, "request timed out", requestErrorMessage(context.DeadlineExceeded))
	assert.Equal(t, "boom", requestErrorMessage(errors.New("boom")))
}
