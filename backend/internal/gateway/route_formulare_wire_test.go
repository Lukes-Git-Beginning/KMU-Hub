package gateway

// Formulare schemas and submissions as FormularePage reads them (formulare-types.ts): lists as
// `{items,total}` (empty -> `[]`), single entities flat, camelCase, `fields`/`answers` as JSON,
// status as a word. The request side takes the camelCase body and hands the RPC the JSON bytes.
// A loopback gRPC server plays the formulare service and records what reached it.

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	formularev1 "github.com/kmuhub/kmuhub/proto/formulare/v1"
)

const wireSchemaFields = `[{"id":"f1","type":"text","label":"Name","required":true}]`

type formulareWireStub struct {
	formularev1.UnimplementedFormulareServiceServer
	empty      bool
	createReq  *formularev1.CreateFormSchemaRequest
	updateReq  *formularev1.UpdateFormSchemaRequest
	dupReq     *formularev1.DuplicateFormSchemaRequest
	submitReq  *formularev1.CreateSubmissionRequest
	listSchema *formularev1.ListFormSchemasRequest
	updateHook *formularev1.UpdateWebhookRequest
	listDeliv  *formularev1.ListWebhookDeliveriesRequest
	linkReq    *formularev1.CreateFormShareLinkRequest
}

func wireSchema() *formularev1.FormSchema {
	now := timestamppb.New(time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC))
	return &formularev1.FormSchema{
		Id: "s1", TenantId: testTenantID.String(), Title: "Kontakt", Fields: []byte(wireSchemaFields),
		Status: formularev1.FormSchemaStatus_FORM_SCHEMA_STATUS_ACTIVE, IsTemplate: true,
		PageCount: 1, CreatedBy: testHRUserID, CreatedAt: now, UpdatedAt: now,
	}
}

func wireSubmission() *formularev1.FormSubmission {
	return &formularev1.FormSubmission{
		Id: "sub1", FormSchemaId: "s1", TenantId: testTenantID.String(), Answers: []byte(`{"f1":"Anna"}`),
		Status:      formularev1.FormSubmissionStatus_FORM_SUBMISSION_STATUS_READ,
		SubmittedAt: timestamppb.New(time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)),
	}
}

func (s *formulareWireStub) ListFormSchemas(_ context.Context, req *formularev1.ListFormSchemasRequest) (*formularev1.ListFormSchemasResponse, error) {
	s.listSchema = req
	if s.empty {
		return &formularev1.ListFormSchemasResponse{}, nil
	}
	return &formularev1.ListFormSchemasResponse{Schemas: []*formularev1.FormSchema{wireSchema()}, Total: 1}, nil
}

func (s *formulareWireStub) GetFormSchema(context.Context, *formularev1.GetFormSchemaRequest) (*formularev1.GetFormSchemaResponse, error) {
	return &formularev1.GetFormSchemaResponse{Schema: wireSchema()}, nil
}

func (s *formulareWireStub) CreateFormSchema(_ context.Context, req *formularev1.CreateFormSchemaRequest) (*formularev1.CreateFormSchemaResponse, error) {
	s.createReq = req
	return &formularev1.CreateFormSchemaResponse{Schema: wireSchema()}, nil
}

func (s *formulareWireStub) UpdateFormSchema(_ context.Context, req *formularev1.UpdateFormSchemaRequest) (*formularev1.UpdateFormSchemaResponse, error) {
	s.updateReq = req
	return &formularev1.UpdateFormSchemaResponse{Schema: wireSchema()}, nil
}

func (s *formulareWireStub) DuplicateFormSchema(_ context.Context, req *formularev1.DuplicateFormSchemaRequest) (*formularev1.DuplicateFormSchemaResponse, error) {
	s.dupReq = req
	return &formularev1.DuplicateFormSchemaResponse{Schema: wireSchema()}, nil
}

func (s *formulareWireStub) ListSubmissions(context.Context, *formularev1.ListSubmissionsRequest) (*formularev1.ListSubmissionsResponse, error) {
	if s.empty {
		return &formularev1.ListSubmissionsResponse{}, nil
	}
	return &formularev1.ListSubmissionsResponse{Submissions: []*formularev1.FormSubmission{wireSubmission()}, Total: 1}, nil
}

func (s *formulareWireStub) GetSubmission(context.Context, *formularev1.GetSubmissionRequest) (*formularev1.GetSubmissionResponse, error) {
	return &formularev1.GetSubmissionResponse{Submission: wireSubmission()}, nil
}

func (s *formulareWireStub) CreateSubmission(_ context.Context, req *formularev1.CreateSubmissionRequest) (*formularev1.CreateSubmissionResponse, error) {
	s.submitReq = req
	return &formularev1.CreateSubmissionResponse{Submission: wireSubmission()}, nil
}

func (s *formulareWireStub) UpdateSubmissionStatus(context.Context, *formularev1.UpdateSubmissionStatusRequest) (*formularev1.UpdateSubmissionStatusResponse, error) {
	return &formularev1.UpdateSubmissionStatusResponse{Submission: wireSubmission()}, nil
}

func newFormulareWireRoutes(t *testing.T, stub *formulareWireStub) *FormulareRoutes {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer()
	formularev1.RegisterFormulareServiceServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("formulare", lis.Addr().String())
	t.Cleanup(reg.Close)
	return newFormulareRoutes(reg)
}

func formulareWireCall(t *testing.T, h http.HandlerFunc, method string, body any, want int) map[string]any {
	t.Helper()
	req := httptest.NewRequest(method, "/x", nil)
	if body != nil {
		req = httptest.NewRequest(method, "/x", jsonBody(t, body))
	}
	req = withAuth(req, testHRUserID, testTenantID)
	rec := httptest.NewRecorder()
	h(rec, req)
	assertStatus(t, rec, want)
	return decodeBody(t, rec)
}

func assertWireSchema(t *testing.T, got map[string]any) {
	t.Helper()
	assert.Equal(t, "s1", got["id"])
	assert.Equal(t, "active", got["status"])
	assert.Equal(t, true, got["isTemplate"])
	assert.Equal(t, false, got["isPublic"])
	assert.Equal(t, float64(1), got["pageCount"])
	assert.Equal(t, float64(0), got["submissionCount"])
	assert.Equal(t, testHRUserID, got["createdBy"])
	assert.Equal(t, "2026-10-23T12:00:00Z", got["createdAt"])
	assert.Nil(t, got["deletedAt"])
	fields, ok := got["fields"].([]any)
	require.True(t, ok, "fields must be a JSON array, got %v", got["fields"])
	require.Len(t, fields, 1)
	assert.Equal(t, "Name", fields[0].(map[string]any)["label"])
	assert.NotContains(t, got, "schema")
}

func TestFormulareWire_ListSchemasIsItemsAndTotal(t *testing.T) {
	routes := newFormulareWireRoutes(t, &formulareWireStub{})
	body := formulareWireCall(t, routes.HandleListFormSchemas, http.MethodGet, nil, http.StatusOK)

	assert.Equal(t, float64(1), body["total"])
	items, ok := body["items"].([]any)
	require.True(t, ok, "%v", body)
	require.Len(t, items, 1)
	assertWireSchema(t, items[0].(map[string]any))
	assert.NotContains(t, body, "schemas")
}

func TestFormulareWire_EmptyListsAreArrays(t *testing.T) {
	routes := newFormulareWireRoutes(t, &formulareWireStub{empty: true})

	for name, h := range map[string]http.HandlerFunc{
		"schemas":     routes.HandleListFormSchemas,
		"submissions": routes.HandleListSubmissions,
	} {
		t.Run(name, func(t *testing.T) {
			req := withAuth(httptest.NewRequest(http.MethodGet, "/x", nil), testHRUserID, testTenantID)
			if name == "submissions" {
				req = withChiURLParam(req, "id", testSchemaID)
			}
			rec := httptest.NewRecorder()
			h(rec, req)
			assertStatus(t, rec, http.StatusOK)
			body := decodeBody(t, rec)
			assert.Equal(t, []any{}, body["items"])
			assert.Equal(t, float64(0), body["total"])
		})
	}
}

func TestFormulareWire_ListSchemasDefaultPageHoldsEveryForm(t *testing.T) {
	stub := &formulareWireStub{}
	routes := newFormulareWireRoutes(t, stub)
	formulareWireCall(t, routes.HandleListFormSchemas, http.MethodGet, nil, http.StatusOK)
	assert.Equal(t, int32(100), stub.listSchema.GetLimit())
}

func TestFormulareWire_SingleSchemaIsFlat(t *testing.T) {
	routes := newFormulareWireRoutes(t, &formulareWireStub{})

	get := withChiURLParam(withAuth(httptest.NewRequest(http.MethodGet, "/x", nil), testHRUserID, testTenantID), "id", testSchemaID)
	rec := httptest.NewRecorder()
	routes.HandleGetFormSchema(rec, get)
	assertStatus(t, rec, http.StatusOK)
	assertWireSchema(t, decodeBody(t, rec))
}

func TestFormulareWire_CreateTakesCamelCaseBodyAndJSONFields(t *testing.T) {
	stub := &formulareWireStub{}
	routes := newFormulareWireRoutes(t, stub)

	body := formulareWireCall(t, routes.HandleCreateFormSchema, http.MethodPost, map[string]any{
		"title":          "Kontakt",
		"fields":         []map[string]any{{"id": "f1", "type": "text", "label": "Name", "required": true}},
		"status":         "active",
		"isTemplate":     true,
		"isPublic":       true,
		"pageCount":      2,
		"intakeTargetId": "helpdesk_ticket",
	}, http.StatusCreated)

	assertWireSchema(t, body)
	require.NotNil(t, stub.createReq)
	assert.JSONEq(t, wireSchemaFields, string(stub.createReq.GetFields()))
	assert.Equal(t, formularev1.FormSchemaStatus_FORM_SCHEMA_STATUS_ACTIVE, stub.createReq.GetStatus())
	assert.True(t, stub.createReq.GetIsTemplate())
	assert.True(t, stub.createReq.GetIsPublic())
	assert.Equal(t, int32(2), stub.createReq.GetPageCount())
	assert.Equal(t, "helpdesk_ticket", stub.createReq.GetIntakeTargetId())
}

func TestFormulareWire_CreateStillTakesSnakeCaseBody(t *testing.T) {
	stub := &formulareWireStub{}
	routes := newFormulareWireRoutes(t, stub)

	formulareWireCall(t, routes.HandleCreateFormSchema, http.MethodPost, map[string]any{
		"title": "Kontakt", "is_template": true, "page_count": 3, "intake_target_id": "helpdesk_ticket",
	}, http.StatusCreated)

	assert.True(t, stub.createReq.GetIsTemplate())
	assert.Equal(t, int32(3), stub.createReq.GetPageCount())
	assert.Equal(t, "helpdesk_ticket", stub.createReq.GetIntakeTargetId())
	assert.Equal(t, formularev1.FormSchemaStatus_FORM_SCHEMA_STATUS_DRAFT, stub.createReq.GetStatus())
}

func TestFormulareWire_ClosedStatusIsRejectedNotStoredAsSomethingElse(t *testing.T) {
	stub := &formulareWireStub{}
	routes := newFormulareWireRoutes(t, stub)

	formulareWireCall(t, routes.HandleCreateFormSchema, http.MethodPost,
		map[string]any{"title": "Kontakt", "status": "closed"}, http.StatusBadRequest)
	assert.Nil(t, stub.createReq)
}

func TestFormulareWire_UpdateCarriesStatusAndCamelCaseFlags(t *testing.T) {
	stub := &formulareWireStub{}
	routes := newFormulareWireRoutes(t, stub)

	req := withChiURLParam(withAuth(httptest.NewRequest(http.MethodPatch, "/x", jsonBody(t, map[string]any{
		"status": "archived", "isPublic": true, "pageCount": 4,
		"fields": []map[string]any{{"id": "f1", "type": "text", "label": "Name", "required": true}},
	})), testHRUserID, testTenantID), "id", testSchemaID)
	rec := httptest.NewRecorder()
	routes.HandleUpdateFormSchema(rec, req)
	assertStatus(t, rec, http.StatusOK)
	assertWireSchema(t, decodeBody(t, rec))

	require.NotNil(t, stub.updateReq)
	require.NotNil(t, stub.updateReq.Status)
	assert.Equal(t, formularev1.FormSchemaStatus_FORM_SCHEMA_STATUS_ARCHIVED, stub.updateReq.GetStatus())
	assert.True(t, stub.updateReq.GetIsPublic())
	assert.Equal(t, int32(4), stub.updateReq.GetPageCount())
	assert.JSONEq(t, wireSchemaFields, string(stub.updateReq.GetFields()))
	assert.Nil(t, stub.updateReq.IsTemplate, "an unsent flag stays unset")
}

func TestFormulareWire_UpdateWithoutStatusLeavesItUnset(t *testing.T) {
	stub := &formulareWireStub{}
	routes := newFormulareWireRoutes(t, stub)

	req := withChiURLParam(withAuth(httptest.NewRequest(http.MethodPatch, "/x", jsonBody(t, map[string]any{"title": "Neu"})), testHRUserID, testTenantID), "id", testSchemaID)
	routes.HandleUpdateFormSchema(httptest.NewRecorder(), req)

	require.NotNil(t, stub.updateReq)
	assert.Nil(t, stub.updateReq.Status)
}

func TestFormulareWire_DuplicateTakesTitle(t *testing.T) {
	stub := &formulareWireStub{}
	routes := newFormulareWireRoutes(t, stub)

	req := withChiURLParam(withAuth(httptest.NewRequest(http.MethodPost, "/x", jsonBody(t, map[string]any{"title": "Kopie"})), testHRUserID, testTenantID), "id", testSchemaID)
	rec := httptest.NewRecorder()
	routes.HandleDuplicateFormSchema(rec, req)
	assertStatus(t, rec, http.StatusCreated)
	assertWireSchema(t, decodeBody(t, rec))
	assert.Equal(t, "Kopie", stub.dupReq.GetNewTitle())
}

func assertWireSubmission(t *testing.T, got map[string]any) {
	t.Helper()
	assert.Equal(t, "sub1", got["id"])
	assert.Equal(t, "s1", got["formSchemaId"])
	assert.Equal(t, "read", got["status"])
	assert.Equal(t, map[string]any{"f1": "Anna"}, got["answers"])
	assert.Nil(t, got["submittedBy"])
	assert.Nil(t, got["ipAddress"])
	assert.Equal(t, "2026-10-23T12:00:00Z", got["submittedAt"])
	assert.NotContains(t, got, "submission")
}

func TestFormulareWire_SubmissionsListAndSingle(t *testing.T) {
	routes := newFormulareWireRoutes(t, &formulareWireStub{})

	list := withChiURLParam(withAuth(httptest.NewRequest(http.MethodGet, "/x", nil), testHRUserID, testTenantID), "id", testSchemaID)
	rec := httptest.NewRecorder()
	routes.HandleListSubmissions(rec, list)
	assertStatus(t, rec, http.StatusOK)
	body := decodeBody(t, rec)
	assert.Equal(t, float64(1), body["total"])
	items, ok := body["items"].([]any)
	require.True(t, ok, "%v", body)
	require.Len(t, items, 1)
	assertWireSubmission(t, items[0].(map[string]any))

	get := withChiURLParam(withAuth(httptest.NewRequest(http.MethodGet, "/x", nil), testHRUserID, testTenantID), "id", testSchemaID)
	rec = httptest.NewRecorder()
	routes.HandleGetSubmission(rec, get)
	assertStatus(t, rec, http.StatusOK)
	assertWireSubmission(t, decodeBody(t, rec))

	patch := withChiURLParam(withAuth(httptest.NewRequest(http.MethodPatch, "/x", jsonBody(t, map[string]any{"status": "read"})), testHRUserID, testTenantID), "id", testSchemaID)
	rec = httptest.NewRecorder()
	routes.HandleUpdateSubmissionStatus(rec, patch)
	assertStatus(t, rec, http.StatusOK)
	assertWireSubmission(t, decodeBody(t, rec))
}

func TestFormulareWire_CreateSubmissionTakesAnswersAsObject(t *testing.T) {
	stub := &formulareWireStub{}
	routes := newFormulareWireRoutes(t, stub)

	req := withChiURLParam(withAuth(httptest.NewRequest(http.MethodPost, "/x", jsonBody(t, map[string]any{
		"answers": map[string]any{"f1": "Anna"}, "submittedBy": testHRUserID,
	})), testHRUserID, testTenantID), "id", testSchemaID)
	rec := httptest.NewRecorder()
	routes.HandleCreateSubmission(rec, req)
	assertStatus(t, rec, http.StatusCreated)
	assertWireSubmission(t, decodeBody(t, rec))

	require.NotNil(t, stub.submitReq)
	assert.JSONEq(t, `{"f1":"Anna"}`, string(stub.submitReq.GetAnswers()))
	assert.Equal(t, testHRUserID, stub.submitReq.GetSubmittedBy())
}

func TestJSONBlob(t *testing.T) {
	assert.Nil(t, jsonBlob(nil))
	assert.Nil(t, jsonBlob([]byte("null")))
	assert.Equal(t, `[{"id":"f1"}]`, string(jsonBlob([]byte(`[{"id":"f1"}]`))))
	// the body shape before the camelCase contract: a base64 string
	assert.Equal(t, `{"f1":"x"}`, string(jsonBlob([]byte(`"eyJmMSI6IngifQ=="`))))
}
