package gateway

// Automation routes as AutomatisierungPage reads them (automation-types.ts): single entities flat,
// enums as words (`scope`, execution `status`, template `complexity`), lists keep their wrapper,
// the dry run carries `condition_result`/`steps`/`would_execute` and the condition test `error`.
// A loopback gRPC server plays the automation service.

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
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	automationv1 "github.com/kmuhub/kmuhub/proto/automation/v1"
)

type automationWireStub struct {
	automationv1.UnimplementedAutomationServiceServer
	empty        bool
	matched      bool
	dryRunErr    string
	conditionErr string
}

var automationWireTime = timestamppb.New(time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC))

func wireAutomation(active bool) *automationv1.AutomationInfo {
	cfg, _ := structpb.NewStruct(map[string]any{"field": "status"})
	actions, _ := structpb.NewList([]any{map[string]any{"type": "send_email"}})
	return &automationv1.AutomationInfo{
		Id: "a1", Name: "Willkommen", Scope: automationv1.AutomationScope_SCOPE_TEAM, OwnerId: "u1",
		TriggerType: "contact.created", TriggerConfig: cfg, Actions: actions, IsActive: active,
		CreatedAt: automationWireTime, UpdatedAt: automationWireTime,
	}
}

func wireExecution() *automationv1.AutomationExecutionInfo {
	return &automationv1.AutomationExecutionInfo{
		Id: "e1", AutomationId: "a1", ConditionResult: true,
		Status:    automationv1.ExecutionStatus_EXECUTION_STATUS_COMPLETED,
		Steps:     []*automationv1.ExecutionStepInfo{{ActionType: "send_email", DurationMs: 12}},
		StartedAt: automationWireTime,
	}
}

func (s *automationWireStub) CreateAutomation(context.Context, *automationv1.CreateAutomationRequest) (*automationv1.CreateAutomationResponse, error) {
	return &automationv1.CreateAutomationResponse{Automation: wireAutomation(false)}, nil
}

func (s *automationWireStub) GetAutomation(context.Context, *automationv1.GetAutomationRequest) (*automationv1.GetAutomationResponse, error) {
	return &automationv1.GetAutomationResponse{Automation: wireAutomation(false)}, nil
}

func (s *automationWireStub) UpdateAutomation(context.Context, *automationv1.UpdateAutomationRequest) (*automationv1.UpdateAutomationResponse, error) {
	return &automationv1.UpdateAutomationResponse{Automation: wireAutomation(false)}, nil
}

func (s *automationWireStub) EnableAutomation(context.Context, *automationv1.EnableAutomationRequest) (*automationv1.EnableAutomationResponse, error) {
	return &automationv1.EnableAutomationResponse{Automation: wireAutomation(true)}, nil
}

func (s *automationWireStub) DisableAutomation(context.Context, *automationv1.DisableAutomationRequest) (*automationv1.DisableAutomationResponse, error) {
	return &automationv1.DisableAutomationResponse{Automation: wireAutomation(false)}, nil
}

func (s *automationWireStub) ListAutomations(context.Context, *automationv1.ListAutomationsRequest) (*automationv1.ListAutomationsResponse, error) {
	if s.empty {
		return &automationv1.ListAutomationsResponse{}, nil
	}
	return &automationv1.ListAutomationsResponse{Automations: []*automationv1.AutomationInfo{wireAutomation(true)}, TotalCount: 1}, nil
}

func (s *automationWireStub) ListExecutions(context.Context, *automationv1.ListExecutionsRequest) (*automationv1.ListExecutionsResponse, error) {
	if s.empty {
		return &automationv1.ListExecutionsResponse{}, nil
	}
	return &automationv1.ListExecutionsResponse{Executions: []*automationv1.AutomationExecutionInfo{wireExecution()}, TotalCount: 1}, nil
}

func (s *automationWireStub) GetExecution(context.Context, *automationv1.GetExecutionRequest) (*automationv1.GetExecutionResponse, error) {
	return &automationv1.GetExecutionResponse{Execution: wireExecution()}, nil
}

func (s *automationWireStub) ListTemplates(context.Context, *automationv1.ListTemplatesRequest) (*automationv1.ListTemplatesResponse, error) {
	if s.empty {
		return &automationv1.ListTemplatesResponse{}, nil
	}
	return &automationv1.ListTemplatesResponse{Templates: []*automationv1.AutomationTemplateInfo{{
		Id: "t1", Name: "Nachfassen", Category: "vertrieb", Complexity: automationv1.TemplateComplexity_COMPLEXITY_MEDIUM,
		TriggerType: "deal.won",
	}}}, nil
}

func (s *automationWireStub) CreateFromTemplate(context.Context, *automationv1.CreateFromTemplateRequest) (*automationv1.CreateFromTemplateResponse, error) {
	return &automationv1.CreateFromTemplateResponse{Automation: wireAutomation(false)}, nil
}

func (s *automationWireStub) TestCondition(context.Context, *automationv1.TestConditionRequest) (*automationv1.TestConditionResponse, error) {
	return &automationv1.TestConditionResponse{Matches: s.matched, ErrorMessage: s.conditionErr}, nil
}

func (s *automationWireStub) DryRunAutomation(context.Context, *automationv1.DryRunAutomationRequest) (*automationv1.DryRunAutomationResponse, error) {
	return &automationv1.DryRunAutomationResponse{
		ConditionMatched: s.matched, ErrorMessage: s.dryRunErr,
		SimulatedSteps: []*automationv1.ExecutionStepInfo{{ActionType: "send_email"}},
	}, nil
}

func (s *automationWireStub) ListTriggerDefinitions(context.Context, *automationv1.ListTriggerDefinitionsRequest) (*automationv1.ListTriggerDefinitionsResponse, error) {
	if s.empty {
		return &automationv1.ListTriggerDefinitionsResponse{Triggers: []*automationv1.TriggerDefinition{{Type: "crm.deal.won"}}}, nil
	}
	return &automationv1.ListTriggerDefinitionsResponse{Triggers: []*automationv1.TriggerDefinition{{
		Type: "crm.deal.stage_changed", EventType: "crm.deal.stage_changed",
		Fields: []*automationv1.FieldDefinition{{Key: "deal.value", Label: "Deal-Wert", Type: "number", Operators: []string{"equals"}}},
	}}}, nil
}

func (s *automationWireStub) ListActionDefinitions(context.Context, *automationv1.ListActionDefinitionsRequest) (*automationv1.ListActionDefinitionsResponse, error) {
	if s.empty {
		return &automationv1.ListActionDefinitionsResponse{Actions: []*automationv1.ActionDefinition{{Type: "add_tag"}}}, nil
	}
	return &automationv1.ListActionDefinitionsResponse{Actions: []*automationv1.ActionDefinition{{
		Type:         "send_email",
		Params:       []*automationv1.ConfigParam{{Key: "subject", Label: "Betreff", Type: "template", Required: true}},
		OutputFields: []*automationv1.FieldDefinition{{Key: "message_id", Label: "Nachrichten-ID", Type: "string"}},
	}}}, nil
}

func newAutomationWireRoutes(t *testing.T, stub *automationWireStub) *AutomationRoutes {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer()
	automationv1.RegisterAutomationServiceServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("automation", lis.Addr().String())
	t.Cleanup(reg.Close)
	return NewAutomationRoutes(reg)
}

// automationWireCall runs a handler (id, executionId and templateId all point at the same UUID)
// and returns the decoded body.
func automationWireCall(t *testing.T, h http.HandlerFunc, method string, body any, want int) map[string]any {
	t.Helper()
	req := httptest.NewRequest(method, "/x", nil)
	if body != nil {
		req = httptest.NewRequest(method, "/x", jsonBody(t, body))
	}
	req = withAuth(req, testHRUserID, testTenantID)
	for _, key := range []string{"id", "executionId", "templateId"} {
		req = withChiURLParam(req, key, testSchemaID)
	}
	rec := httptest.NewRecorder()
	h(rec, req)
	assertStatus(t, rec, want)
	return decodeBody(t, rec)
}

func assertWireAutomation(t *testing.T, got map[string]any, active bool) {
	t.Helper()
	assert.Equal(t, "a1", got["id"])
	assert.Equal(t, "team", got["scope"], "scope must be a word")
	assert.Equal(t, active, got["is_active"])
	assert.Equal(t, float64(0), got["max_steps"], "zero counter must stay in the body")
	assert.Equal(t, "2026-10-23T12:00:00Z", got["created_at"])
	assert.Nil(t, got["template_id"])
	assert.Nil(t, got["last_triggered_at"])
	assert.Equal(t, map[string]any{"field": "status"}, got["trigger_config"])
	assert.Len(t, got["actions"], 1)
	assert.NotContains(t, got, "automation", "single entity must not be wrapped")
}

func TestAutomationWire_SingleEntitiesAreFlatWithScopeWord(t *testing.T) {
	routes := newAutomationWireRoutes(t, &automationWireStub{})
	body := map[string]any{"name": "Willkommen", "trigger_type": "contact.created"}

	assertWireAutomation(t, automationWireCall(t, routes.HandleCreateAutomation, "POST", body, http.StatusCreated), false)
	assertWireAutomation(t, automationWireCall(t, routes.HandleGetAutomation, "GET", nil, http.StatusOK), false)
	assertWireAutomation(t, automationWireCall(t, routes.HandleUpdateAutomation, "PUT", map[string]any{"name": "x"}, http.StatusOK), false)
	assertWireAutomation(t, automationWireCall(t, routes.HandleEnableAutomation, "POST", nil, http.StatusOK), true)
	assertWireAutomation(t, automationWireCall(t, routes.HandleDisableAutomation, "POST", nil, http.StatusOK), false)
	assertWireAutomation(t, automationWireCall(t, routes.HandleCreateFromTemplate, "POST", map[string]any{"name": "x"}, http.StatusCreated), false)
}

func TestAutomationWire_ListKeepsWrapperWithScopeWord(t *testing.T) {
	routes := newAutomationWireRoutes(t, &automationWireStub{})
	got := automationWireCall(t, routes.HandleListAutomations, "GET", nil, http.StatusOK)
	assert.Equal(t, float64(1), got["total_count"])
	items, ok := got["automations"].([]any)
	require.True(t, ok)
	require.Len(t, items, 1)
	assertWireAutomation(t, items[0].(map[string]any), true)
}

func TestAutomationWire_EmptyListsAreArrays(t *testing.T) {
	routes := newAutomationWireRoutes(t, &automationWireStub{empty: true})
	cases := map[string]struct {
		h   http.HandlerFunc
		key string
	}{
		"automations": {routes.HandleListAutomations, "automations"},
		"executions":  {routes.HandleListExecutions, "executions"},
		"templates":   {routes.HandleListTemplates, "templates"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got := automationWireCall(t, c.h, "GET", nil, http.StatusOK)
			assert.Equal(t, []any{}, got[c.key], "empty list must be [] not null")
		})
	}
}

func TestAutomationWire_ExecutionStatusIsWord(t *testing.T) {
	routes := newAutomationWireRoutes(t, &automationWireStub{})

	single := automationWireCall(t, routes.HandleGetExecution, "GET", nil, http.StatusOK)
	assert.Equal(t, "completed", single["status"])
	assert.Equal(t, true, single["condition_result"])
	assert.Equal(t, map[string]any{}, single["trigger_event"])
	assert.Equal(t, float64(0), single["duration_ms"])
	assert.NotContains(t, single, "execution", "single entity must not be wrapped")

	list := automationWireCall(t, routes.HandleListExecutions, "GET", nil, http.StatusOK)
	assert.Equal(t, float64(1), list["total_count"])
	items := list["executions"].([]any)
	require.Len(t, items, 1)
	assert.Equal(t, "completed", items[0].(map[string]any)["status"])
	steps := items[0].(map[string]any)["steps"].([]any)
	require.Len(t, steps, 1)
	assert.Equal(t, float64(12), steps[0].(map[string]any)["duration_ms"])
}

func TestAutomationWire_TemplateComplexityIsWord(t *testing.T) {
	routes := newAutomationWireRoutes(t, &automationWireStub{})
	got := automationWireCall(t, routes.HandleListTemplates, "GET", nil, http.StatusOK)
	items := got["templates"].([]any)
	require.Len(t, items, 1)
	tpl := items[0].(map[string]any)
	assert.Equal(t, "mittel", tpl["complexity"])
	assert.Equal(t, "vertrieb", tpl["category"])
	assert.Equal(t, []any{}, tpl["actions"], "unset actions must be [] not null")
}

func TestAutomationWire_DryRunCarriesFEFieldNames(t *testing.T) {
	routes := newAutomationWireRoutes(t, &automationWireStub{matched: true})
	body := map[string]any{"automation_id": testSchemaID}

	got := automationWireCall(t, routes.HandleDryRun, "POST", body, http.StatusOK)
	assert.Equal(t, true, got["condition_result"])
	assert.Equal(t, true, got["would_execute"])
	steps, ok := got["steps"].([]any)
	require.True(t, ok)
	require.Len(t, steps, 1)
	assert.Equal(t, "send_email", steps[0].(map[string]any)["action_type"])
	assert.NotContains(t, got, "condition_matched")
	assert.NotContains(t, got, "simulated_steps")
}

func TestAutomationWire_DryRunWouldNotExecuteOnMissOrError(t *testing.T) {
	body := map[string]any{"automation_id": testSchemaID}

	miss := newAutomationWireRoutes(t, &automationWireStub{matched: false})
	got := automationWireCall(t, miss.HandleDryRun, "POST", body, http.StatusOK)
	assert.Equal(t, false, got["condition_result"])
	assert.Equal(t, false, got["would_execute"])

	failed := newAutomationWireRoutes(t, &automationWireStub{matched: true, dryRunErr: "action invalid"})
	got = automationWireCall(t, failed.HandleDryRun, "POST", body, http.StatusOK)
	assert.Equal(t, true, got["condition_result"])
	assert.Equal(t, false, got["would_execute"])
	assert.Equal(t, "action invalid", got["error_message"])
}

func TestAutomationWire_TestConditionCarriesError(t *testing.T) {
	routes := newAutomationWireRoutes(t, &automationWireStub{conditionErr: "bad operator"})
	got := automationWireCall(t, routes.HandleTestCondition, "POST", map[string]any{"condition": map[string]any{}}, http.StatusOK)
	assert.Equal(t, false, got["matches"], "zero value must stay in the body")
	assert.Equal(t, "bad operator", got["error"])
	assert.NotContains(t, got, "error_message")
}

func TestWireEnumWord_UnknownNumberStaysNumber(t *testing.T) {
	obj, err := automationWire(&automationv1.AutomationInfo{Scope: automationv1.AutomationScope(9)})
	require.NoError(t, err)
	assert.Equal(t, "9", string(obj["scope"]))
}

// The condition builder reads `fields[].key/label/operators`, the action editor `params[].key/label`
// and `output_fields[].key`; an action without parameters must still carry `params: []` because
// the editor calls `.length` / `.map` on it.
func TestAutomationWire_CatalogCarriesFieldKeysLabelsAndOperators(t *testing.T) {
	routes := newAutomationWireRoutes(t, &automationWireStub{})

	triggers := automationWireCall(t, routes.HandleListTriggers, "GET", nil, http.StatusOK)["triggers"].([]any)
	fields := triggers[0].(map[string]any)["fields"].([]any)
	field := fields[0].(map[string]any)
	assert.Equal(t, "deal.value", field["key"])
	assert.Equal(t, "Deal-Wert", field["label"])
	assert.Equal(t, []any{"equals"}, field["operators"])
	assert.Equal(t, []any{}, triggers[0].(map[string]any)["config_params"])

	actions := automationWireCall(t, routes.HandleListActions, "GET", nil, http.StatusOK)["actions"].([]any)
	action := actions[0].(map[string]any)
	param := action["params"].([]any)[0].(map[string]any)
	assert.Equal(t, "subject", param["key"])
	assert.Equal(t, "Betreff", param["label"])
	assert.Equal(t, true, param["required"])
	assert.Equal(t, "message_id", action["output_fields"].([]any)[0].(map[string]any)["key"])
}

func TestAutomationWire_CatalogEntryWithoutFieldsStillHasArrays(t *testing.T) {
	routes := newAutomationWireRoutes(t, &automationWireStub{empty: true})

	trigger := automationWireCall(t, routes.HandleListTriggers, "GET", nil, http.StatusOK)["triggers"].([]any)[0].(map[string]any)
	assert.Equal(t, []any{}, trigger["fields"])

	action := automationWireCall(t, routes.HandleListActions, "GET", nil, http.StatusOK)["actions"].([]any)[0].(map[string]any)
	assert.Equal(t, []any{}, action["params"])
	assert.Equal(t, []any{}, action["output_fields"])
}
