package gateway

// Formulare webhooks, deliveries and share links as the FE reads them (formulare-types.ts):
// lists as bare arrays (empty -> `[]`), single entities flat and camelCase, delivery status as a
// word. The request side takes the camelCase share-link body, the delivery filters the FE sends
// and replaces the webhook events when the FE sends them. A loopback gRPC server plays the
// formulare service.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	formularev1 "github.com/kmuhub/kmuhub/proto/formulare/v1"
)

var wireHookTime = time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)

func wireWebhook() *formularev1.FormWebhook {
	last := timestamppb.New(wireHookTime)
	return &formularev1.FormWebhook{
		Id: "w1", FormSchemaId: "s1", TenantId: testTenantID.String(), Url: "https://example.org/hook",
		Secret: "....ab12", Events: []string{"submission.created"}, Active: true,
		LastTriggeredAt: last, LastStatus: "delivered", CreatedAt: last, UpdatedAt: last,
	}
}

func wireDelivery() *formularev1.WebhookDelivery {
	at := timestamppb.New(wireHookTime)
	return &formularev1.WebhookDelivery{
		Id: "d1", WebhookId: "w1", SubmissionId: "sub1", TenantId: testTenantID.String(),
		Payload: []byte(`{"answers":{"f1":"Anna"}}`), Status: formularev1.WebhookDeliveryStatus_WEBHOOK_DELIVERY_STATUS_FAILED,
		AttemptCount: 2, MaxAttempts: 5, NextAttemptAt: at, LastError: "timeout", LastResponseCode: 502, CreatedAt: at,
	}
}

func wireShareLink() *formularev1.FormShareLink {
	createdBy := testHRUserID
	return &formularev1.FormShareLink{
		Id: "l1", TenantId: testTenantID.String(), FormSchemaId: "s1", Token: "tok", MaxSubmissions: 10,
		SubmissionCount: 3, CreatedBy: &createdBy, CreatedAt: timestamppb.New(wireHookTime), Status: "active",
	}
}

func (s *formulareWireStub) ListWebhooks(context.Context, *formularev1.ListWebhooksRequest) (*formularev1.ListWebhooksResponse, error) {
	if s.empty {
		return &formularev1.ListWebhooksResponse{}, nil
	}
	return &formularev1.ListWebhooksResponse{Webhooks: []*formularev1.FormWebhook{wireWebhook()}}, nil
}

func (s *formulareWireStub) CreateWebhook(context.Context, *formularev1.CreateWebhookRequest) (*formularev1.CreateWebhookResponse, error) {
	return &formularev1.CreateWebhookResponse{Webhook: wireWebhook()}, nil
}

func (s *formulareWireStub) GetWebhook(context.Context, *formularev1.GetWebhookRequest) (*formularev1.GetWebhookResponse, error) {
	return &formularev1.GetWebhookResponse{Webhook: wireWebhook()}, nil
}

func (s *formulareWireStub) UpdateWebhook(_ context.Context, req *formularev1.UpdateWebhookRequest) (*formularev1.UpdateWebhookResponse, error) {
	s.updateHook = req
	return &formularev1.UpdateWebhookResponse{Webhook: wireWebhook()}, nil
}

func (s *formulareWireStub) ListWebhookDeliveries(_ context.Context, req *formularev1.ListWebhookDeliveriesRequest) (*formularev1.ListWebhookDeliveriesResponse, error) {
	s.listDeliv = req
	if s.empty {
		return &formularev1.ListWebhookDeliveriesResponse{}, nil
	}
	return &formularev1.ListWebhookDeliveriesResponse{Deliveries: []*formularev1.WebhookDelivery{wireDelivery()}, Total: 1}, nil
}

func (s *formulareWireStub) ListFormShareLinks(context.Context, *formularev1.ListFormShareLinksRequest) (*formularev1.ListFormShareLinksResponse, error) {
	if s.empty {
		return &formularev1.ListFormShareLinksResponse{}, nil
	}
	return &formularev1.ListFormShareLinksResponse{ShareLinks: []*formularev1.FormShareLink{wireShareLink()}, Total: 1}, nil
}

func (s *formulareWireStub) CreateFormShareLink(_ context.Context, req *formularev1.CreateFormShareLinkRequest) (*formularev1.CreateFormShareLinkResponse, error) {
	s.linkReq = req
	return &formularev1.CreateFormShareLinkResponse{ShareLink: wireShareLink()}, nil
}

// formulareHookCall runs a handler with the schema/webhook id as chi param and returns the raw body.
func formulareHookCall(t *testing.T, h http.HandlerFunc, method, query string, body any, want int) []byte {
	t.Helper()
	req := httptest.NewRequest(method, "/x"+query, nil)
	if body != nil {
		req = httptest.NewRequest(method, "/x"+query, jsonBody(t, body))
	}
	req = withChiURLParam(withAuth(req, testHRUserID, testTenantID), "id", testSchemaID)
	rec := httptest.NewRecorder()
	h(rec, req)
	assertStatus(t, rec, want)
	return rec.Body.Bytes()
}

func decodeArray(t *testing.T, raw []byte) []map[string]any {
	t.Helper()
	var arr []map[string]any
	require.NoError(t, json.Unmarshal(raw, &arr), "body must be a bare JSON array: %s", raw)
	return arr
}

func assertWireWebhook(t *testing.T, got map[string]any) {
	t.Helper()
	assert.Equal(t, "w1", got["id"])
	assert.Equal(t, "s1", got["formSchemaId"])
	assert.Equal(t, "https://example.org/hook", got["url"])
	assert.Equal(t, "....ab12", got["secret"])
	assert.Equal(t, []any{"submission.created"}, got["events"])
	assert.Equal(t, true, got["active"])
	assert.Equal(t, "2026-10-23T12:00:00Z", got["lastTriggeredAt"])
	assert.Equal(t, "delivered", got["lastStatus"])
	assert.NotContains(t, got, "webhook")
}

func TestFormulareHookWire_WebhookListIsBareArray(t *testing.T) {
	routes := newFormulareWireRoutes(t, &formulareWireStub{})
	arr := decodeArray(t, formulareHookCall(t, routes.HandleListWebhooks, http.MethodGet, "", nil, http.StatusOK))
	require.Len(t, arr, 1)
	assertWireWebhook(t, arr[0])
}

func TestFormulareHookWire_EmptyListsAreArrays(t *testing.T) {
	routes := newFormulareWireRoutes(t, &formulareWireStub{empty: true})
	for name, h := range map[string]http.HandlerFunc{
		"webhooks":     routes.HandleListWebhooks,
		"deliveries":   routes.HandleListWebhookDeliveries,
		"webhook-dlvr": routes.HandleListWebhookDeliveriesForWebhook,
		"share-links":  routes.HandleListShareLinks,
	} {
		raw := formulareHookCall(t, h, http.MethodGet, "", nil, http.StatusOK)
		assert.JSONEq(t, `[]`, string(raw), name)
	}
}

func TestFormulareHookWire_SingleWebhookIsFlat(t *testing.T) {
	routes := newFormulareWireRoutes(t, &formulareWireStub{})
	for name, call := range map[string]func() []byte{
		"create": func() []byte {
			return formulareHookCall(t, routes.HandleCreateWebhook, http.MethodPost, "",
				map[string]any{"url": "https://example.org/hook", "events": []string{"submission.created"}, "active": true}, http.StatusCreated)
		},
		"get": func() []byte {
			return formulareHookCall(t, routes.HandleGetWebhook, http.MethodGet, "", nil, http.StatusOK)
		},
		"update": func() []byte {
			return formulareHookCall(t, routes.HandleUpdateWebhook, http.MethodPatch, "", map[string]any{"active": true}, http.StatusOK)
		},
	} {
		var got map[string]any
		require.NoError(t, json.Unmarshal(call(), &got), name)
		assertWireWebhook(t, got)
	}
}

func TestFormulareHookWire_UpdateWebhookReplacesEventsWhenSent(t *testing.T) {
	stub := &formulareWireStub{}
	routes := newFormulareWireRoutes(t, stub)

	formulareHookCall(t, routes.HandleUpdateWebhook, http.MethodPatch, "", map[string]any{"events": []string{"submission.created"}}, http.StatusOK)
	require.NotNil(t, stub.updateHook)
	assert.True(t, stub.updateHook.GetEventsSet(), "a sent events list replaces the stored one")
	assert.Equal(t, []string{"submission.created"}, stub.updateHook.GetEvents())

	formulareHookCall(t, routes.HandleUpdateWebhook, http.MethodPatch, "", map[string]any{"events": []string{}}, http.StatusOK)
	assert.True(t, stub.updateHook.GetEventsSet(), "an empty list clears the events")

	formulareHookCall(t, routes.HandleUpdateWebhook, http.MethodPatch, "", map[string]any{"active": false}, http.StatusOK)
	assert.False(t, stub.updateHook.GetEventsSet(), "no events sent leaves them alone")
}

func TestFormulareHookWire_DeliveriesBareArrayWithWordsAndNulls(t *testing.T) {
	routes := newFormulareWireRoutes(t, &formulareWireStub{})
	arr := decodeArray(t, formulareHookCall(t, routes.HandleListWebhookDeliveries, http.MethodGet, "", nil, http.StatusOK))
	require.Len(t, arr, 1)
	d := arr[0]
	assert.Equal(t, "d1", d["id"])
	assert.Equal(t, "w1", d["webhookId"])
	assert.Equal(t, "sub1", d["submissionId"])
	assert.Equal(t, "failed", d["status"])
	assert.Equal(t, float64(2), d["attemptCount"])
	assert.Equal(t, float64(5), d["maxAttempts"])
	assert.Equal(t, "timeout", d["lastError"])
	assert.Equal(t, float64(502), d["lastResponseCode"])
	assert.Equal(t, "2026-10-23T12:00:00Z", d["nextAttemptAt"])
	assert.Nil(t, d["deliveredAt"])
	assert.Equal(t, map[string]any{"answers": map[string]any{"f1": "Anna"}}, d["payload"], "payload is JSON, not base64")
}

func TestFormulareHookWire_DeliveryFiltersAcceptFEWordsAndLimitOffset(t *testing.T) {
	stub := &formulareWireStub{}
	routes := newFormulareWireRoutes(t, stub)

	formulareHookCall(t, routes.HandleListWebhookDeliveries, http.MethodGet,
		"?status=dead&webhook_id=w1&submission_id=sub1&limit=25&offset=50", nil, http.StatusOK)
	require.NotNil(t, stub.listDeliv)
	require.NotNil(t, stub.listDeliv.Status)
	assert.Equal(t, formularev1.WebhookDeliveryStatus_WEBHOOK_DELIVERY_STATUS_DEAD, stub.listDeliv.GetStatus())
	assert.Equal(t, "w1", stub.listDeliv.GetWebhookId())
	assert.Equal(t, "sub1", stub.listDeliv.GetSubmissionId())
	assert.Equal(t, int32(25), stub.listDeliv.GetLimit())
	assert.Equal(t, int32(50), stub.listDeliv.GetOffset())

	formulareHookCall(t, routes.HandleListWebhookDeliveries, http.MethodGet, "?status=2&page=3&page_size=10", nil, http.StatusOK)
	assert.Equal(t, formularev1.WebhookDeliveryStatus_WEBHOOK_DELIVERY_STATUS_DELIVERED, stub.listDeliv.GetStatus(), "the proto number still works")
	assert.Equal(t, int32(10), stub.listDeliv.GetLimit())
	assert.Equal(t, int32(20), stub.listDeliv.GetOffset())

	formulareHookCall(t, routes.HandleListWebhookDeliveries, http.MethodGet, "?status=bogus", nil, http.StatusOK)
	assert.Nil(t, stub.listDeliv.Status, "an unknown word is no filter")
}

func assertWireShareLink(t *testing.T, got map[string]any) {
	t.Helper()
	assert.Equal(t, "l1", got["id"])
	assert.Equal(t, "s1", got["formSchemaId"])
	assert.Equal(t, "tok", got["token"])
	assert.Equal(t, "link", got["channel"])
	assert.Equal(t, "", got["url"], "no public page exists yet, so no invented URL")
	assert.Equal(t, float64(10), got["maxSubmissions"])
	assert.Equal(t, float64(3), got["submissions"])
	assert.Equal(t, float64(0), got["views"])
	assert.Equal(t, "active", got["status"])
	assert.Equal(t, testHRUserID, got["createdBy"])
	assert.Nil(t, got["expiresAt"])
	assert.NotContains(t, got, "share_link")
	assert.NotContains(t, got, "shareLink")
}

func TestFormulareHookWire_ShareLinkListAndCreateAreFlat(t *testing.T) {
	routes := newFormulareWireRoutes(t, &formulareWireStub{})
	arr := decodeArray(t, formulareHookCall(t, routes.HandleListShareLinks, http.MethodGet, "", nil, http.StatusOK))
	require.Len(t, arr, 1)
	assertWireShareLink(t, arr[0])

	var created map[string]any
	require.NoError(t, json.Unmarshal(formulareHookCall(t, routes.HandleCreateShareLink, http.MethodPost, "",
		map[string]any{"channel": "link"}, http.StatusCreated), &created))
	assertWireShareLink(t, created)
}

func TestFormulareHookWire_ShareLinkUnlimitedIsNull(t *testing.T) {
	raw, err := json.Marshal(toFormShareLinkWire(&formularev1.FormShareLink{Id: "l2", Status: "active"}))
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Nil(t, got["maxSubmissions"], "0 means unlimited; the FE type is number | null")
	assert.Nil(t, got["expiresAt"])
	assert.Nil(t, got["createdBy"])
}

func TestFormulareHookWire_CreateShareLinkTakesCamelCaseBody(t *testing.T) {
	stub := &formulareWireStub{}
	routes := newFormulareWireRoutes(t, stub)

	formulareHookCall(t, routes.HandleCreateShareLink, http.MethodPost, "",
		map[string]any{"channel": "embed", "expiresAt": "2026-12-31T00:00:00Z", "maxSubmissions": 7}, http.StatusCreated)
	require.NotNil(t, stub.linkReq)
	assert.Equal(t, int32(7), stub.linkReq.GetMaxSubmissions())
	require.NotNil(t, stub.linkReq.ExpiresAt)
	assert.Equal(t, "2026-12-31T00:00:00Z", stub.linkReq.GetExpiresAt().AsTime().Format(time.RFC3339))

	formulareHookCall(t, routes.HandleCreateShareLink, http.MethodPost, "",
		map[string]any{"channel": "link", "expiresAt": nil, "maxSubmissions": nil}, http.StatusCreated)
	assert.Equal(t, int32(0), stub.linkReq.GetMaxSubmissions(), "null = unlimited")
	assert.Nil(t, stub.linkReq.ExpiresAt)

	formulareHookCall(t, routes.HandleCreateShareLink, http.MethodPost, "",
		map[string]any{"max_submissions": 4}, http.StatusCreated)
	assert.Equal(t, int32(4), stub.linkReq.GetMaxSubmissions(), "the snake_case body still works")
}
