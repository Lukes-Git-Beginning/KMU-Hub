package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kmuhub/kmuhub/internal/server/response"
	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

func offboardRequest(body string) *http.Request {
	r := httptest.NewRequest("POST", "/api/v1/hr/employees/11111111-1111-1111-1111-111111111111/offboard",
		strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestHandleOffboardEmployee_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	testServiceUnavailable(t, routes.HandleOffboardEmployee)
}

// The dialog posts camelCase (OffboardEmployeeDialog -> hr-client.ts), so a
// snake_case-only binding would arrive with every field empty and the required
// ones would fail validation instead of the request going through.
func TestHandleOffboardEmployee_RejectsAnUnknownExitTypeAtTheBorder(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)

	for name, body := range map[string]string{
		"unknown exit type": `{"exitDate":"2026-08-31","exitType":"fired_by_email"}`,
		"missing exit date": `{"exitType":"resignation"}`,
		"malformed date":    `{"exitDate":"31.08.2026","exitType":"resignation"}`,
		"snake_case body":   `{"exit_date":"2026-08-31","exit_type":"resignation"}`,
	} {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			routes.HandleOffboardEmployee(rec, offboardRequest(body))
			assertStatus(t, rec, http.StatusBadRequest)
		})
	}
}

// fixed_term_expired is the one exit type the desktop dialog offers that the
// backlog's scope did not list. Refusing it would leave a value the UI can
// produce failing at the border.
func TestHandleOffboardEmployee_AcceptsEveryExitTypeTheDialogOffers(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)

	for _, exitType := range []string{
		"resignation", "termination", "fixed_term_expired",
		"mutual_termination", "retirement",
	} {
		t.Run(exitType, func(t *testing.T) {
			rec := httptest.NewRecorder()
			routes.HandleOffboardEmployee(rec,
				offboardRequest(`{"exitDate":"2026-08-31","exitType":"`+exitType+`"}`))
			// The registry is empty, so a body that passes validation reaches
			// the client lookup and stops at 503 — anything else means the
			// value was refused.
			assertStatus(t, rec, http.StatusServiceUnavailable)
		})
	}
}

// The client reads `{employee}` (hr-client.ts:943), so the handler writes the
// whole response message rather than resp.Employee. A bare profile would leave
// adaptEmployee reading undefined.
func TestOffboardResponse_IsWrappedInEmployee(t *testing.T) {
	rec := httptest.NewRecorder()
	response.Proto(rec, http.StatusOK, &hrv1.OffboardEmployeeResp{
		Employee: &hrv1.EmployeeProfile{
			Id:       "11111111-1111-1111-1111-111111111111",
			UserId:   "22222222-2222-2222-2222-222222222222",
			Status:   "inactive",
			ExitDate: "2026-08-31",
			ExitType: "resignation",
		},
	})

	var decoded map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	employee, ok := decoded["employee"].(map[string]any)
	if !ok {
		t.Fatalf("response is not wrapped in employee: %s", rec.Body.String())
	}
	// snake_case, matching the rest of the HR profile contract.
	for key, want := range map[string]any{
		"status": "inactive", "exit_date": "2026-08-31", "exit_type": "resignation",
		"user_id": "22222222-2222-2222-2222-222222222222",
	} {
		if employee[key] != want {
			t.Errorf("employee.%s = %v, want %v", key, employee[key], want)
		}
	}
}
