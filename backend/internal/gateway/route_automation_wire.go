package gateway

// Wire shape of the automation routes as AutomatisierungPage reads them (automation-types.ts):
// single entities flat (no `{automation:…}` / `{execution:…}` envelope), enums as words
// (`scope`, execution `status`, template `complexity`), `trigger_config`/`trigger_event` as an
// object and `actions` as an array instead of `null`. The gRPC service keeps its proto types; the
// conversion sits at this edge only.

import (
	"encoding/json"
	"net/http"

	"google.golang.org/protobuf/proto"

	"github.com/kmuhub/kmuhub/internal/server/response"
	automationv1 "github.com/kmuhub/kmuhub/proto/automation/v1"
)

var (
	automationScopeWords      = map[string]string{"1": "personal", "2": "team", "3": "organization"}
	automationStatusWords     = map[string]string{"1": "running", "2": "completed", "3": "failed", "4": "skipped", "5": "aborted"}
	automationComplexityWords = map[string]string{"1": "einfach", "2": "mittel", "3": "fortgeschritten"}
)

// wireObject marshals msg with every field present and returns its members, so a handler can
// swap single members without a hand-built mirror struct per message.
func wireObject(msg proto.Message) (map[string]json.RawMessage, error) {
	b, err := response.MarshalProtoFull(msg)
	if err != nil {
		return nil, err
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(b, &obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// wireEnumWord replaces a numeric enum member by its word. An unknown number (a future enum
// value, or 0) stays a number rather than turning into a wrong word.
func wireEnumWord(obj map[string]json.RawMessage, key string, words map[string]string) {
	if word, ok := words[string(obj[key])]; ok {
		obj[key] = json.RawMessage(`"` + word + `"`)
	}
}

// wireDefault replaces a missing or `null` member by def (the editor iterates `actions` and
// reads keys of `trigger_config`).
func wireDefault(obj map[string]json.RawMessage, key, def string) {
	if v, ok := obj[key]; !ok || string(v) == "null" {
		obj[key] = json.RawMessage(def)
	}
}

func automationWire(a *automationv1.AutomationInfo) (map[string]json.RawMessage, error) {
	if a == nil {
		a = &automationv1.AutomationInfo{}
	}
	obj, err := wireObject(a)
	if err != nil {
		return nil, err
	}
	wireEnumWord(obj, "scope", automationScopeWords)
	wireDefault(obj, "trigger_config", "{}")
	wireDefault(obj, "actions", "[]")
	return obj, nil
}

func automationExecutionWire(e *automationv1.AutomationExecutionInfo) (map[string]json.RawMessage, error) {
	if e == nil {
		e = &automationv1.AutomationExecutionInfo{}
	}
	obj, err := wireObject(e)
	if err != nil {
		return nil, err
	}
	wireEnumWord(obj, "status", automationStatusWords)
	wireDefault(obj, "trigger_event", "{}")
	wireDefault(obj, "duration_ms", "0")
	return obj, nil
}

func automationTemplateWire(t *automationv1.AutomationTemplateInfo) (map[string]json.RawMessage, error) {
	obj, err := wireObject(t)
	if err != nil {
		return nil, err
	}
	wireEnumWord(obj, "complexity", automationComplexityWords)
	wireDefault(obj, "trigger_config", "{}")
	wireDefault(obj, "actions", "[]")
	return obj, nil
}

// wireList converts every element, keeping an empty list a `[]`.
func wireList[T any](items []T, conv func(T) (map[string]json.RawMessage, error)) ([]map[string]json.RawMessage, error) {
	out := make([]map[string]json.RawMessage, 0, len(items))
	for _, item := range items {
		obj, err := conv(item)
		if err != nil {
			return nil, err
		}
		out = append(out, obj)
	}
	return out, nil
}

// writeAutomationWire answers with v, or 500 when a message cannot be marshalled.
func writeAutomationWire(w http.ResponseWriter, status int, v any, err error) {
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.JSON(w, status, v)
}

func writeAutomation(w http.ResponseWriter, status int, a *automationv1.AutomationInfo) {
	obj, err := automationWire(a)
	writeAutomationWire(w, status, obj, err)
}

func writeAutomationExecution(w http.ResponseWriter, status int, e *automationv1.AutomationExecutionInfo) {
	obj, err := automationExecutionWire(e)
	writeAutomationWire(w, status, obj, err)
}

type automationListWire struct {
	Automations []map[string]json.RawMessage `json:"automations"`
	TotalCount  int32                        `json:"total_count"`
}

type automationExecutionListWire struct {
	Executions []map[string]json.RawMessage `json:"executions"`
	TotalCount int32                        `json:"total_count"`
}

type automationTemplateListWire struct {
	Templates []map[string]json.RawMessage `json:"templates"`
}

// automationStepsWire turns the simulated steps into the FE `ExecutionStep` shape (all members
// present, `input`/`output` as objects).
func automationStepsWire(steps []*automationv1.ExecutionStepInfo) ([]map[string]json.RawMessage, error) {
	return wireList(steps, func(s *automationv1.ExecutionStepInfo) (map[string]json.RawMessage, error) {
		obj, err := wireObject(s)
		if err != nil {
			return nil, err
		}
		wireDefault(obj, "input", "{}")
		wireDefault(obj, "output", "{}")
		return obj, nil
	})
}

// dryRunWire is the FE `DryRunResponse`. The proto has no `would_execute`: the automation would
// run when its condition matched and the simulation reported no error.
type dryRunWire struct {
	ConditionResult bool                         `json:"condition_result"`
	Steps           []map[string]json.RawMessage `json:"steps"`
	WouldExecute    bool                         `json:"would_execute"`
	ErrorMessage    string                       `json:"error_message"`
}

func toDryRunWire(resp *automationv1.DryRunAutomationResponse) (dryRunWire, error) {
	steps, err := automationStepsWire(resp.GetSimulatedSteps())
	if err != nil {
		return dryRunWire{}, err
	}
	return dryRunWire{
		ConditionResult: resp.GetConditionMatched(),
		Steps:           steps,
		WouldExecute:    resp.GetConditionMatched() && resp.GetErrorMessage() == "",
		ErrorMessage:    resp.GetErrorMessage(),
	}, nil
}

// testConditionWire is the FE `TestConditionResponse` (`error`, not `error_message`).
type testConditionWire struct {
	Matches bool   `json:"matches"`
	Error   string `json:"error"`
}
