package gateway

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
)

// produktionRouter registers the produktion routes on a fresh chi router the
// same way cmd/gateway/main.go does (flag on, auth middleware passed through).
func produktionRouter(t *testing.T) *chi.Mux {
	t.Helper()
	r := chi.NewRouter()
	routes := NewProduktionRoutes(emptyRegistry(), produktionFlagsON())
	routes.RegisterRoutes(r, func(next http.Handler) http.Handler { return next })
	return r
}

// TestProduktionExtRoutes_Registered walks the router and asserts that the BOM,
// work step, machine and quality check patterns are actually mounted. They were
// fully implemented — handlers, RPCs and openapi.yaml entries — but the
// registration function had no caller, so every one of these was a 404 against a
// real gateway. Walking the router is the check; "the code exists" is not.
func TestProduktionExtRoutes_Registered(t *testing.T) {
	r := produktionRouter(t)

	registered := make(map[string]bool)
	err := chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if len(route) > 1 && route[len(route)-1] == '/' {
			route = route[:len(route)-1]
		}
		registered[method+" "+route] = true
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk failed: %v", err)
	}

	want := []string{
		"GET /api/v1/produktion/boms",
		"POST /api/v1/produktion/boms",
		"GET /api/v1/produktion/boms/{bomId}",
		"PATCH /api/v1/produktion/boms/{bomId}",
		"DELETE /api/v1/produktion/boms/{bomId}",
		"GET /api/v1/produktion/orders/{orderId}/steps",
		"POST /api/v1/produktion/orders/{orderId}/steps",
		"PATCH /api/v1/produktion/orders/{orderId}/steps/{stepId}",
		"DELETE /api/v1/produktion/orders/{orderId}/steps/{stepId}",
		"GET /api/v1/produktion/machines",
		"POST /api/v1/produktion/machines",
		"GET /api/v1/produktion/machines/{machineId}",
		"PATCH /api/v1/produktion/machines/{machineId}",
		"DELETE /api/v1/produktion/machines/{machineId}",
		"GET /api/v1/produktion/quality",
		"POST /api/v1/produktion/quality",
		"GET /api/v1/produktion/quality/{checkId}",
	}
	for _, w := range want {
		if !registered[w] {
			t.Errorf("route not registered: %s", w)
		}
	}
}

// TestProduktionExtRoutes_MatchAgainstOrderSubtree pins the routing decision
// that made this wiring delicate: /orders/{id} and /orders/{orderId}/steps share
// the same chi param position. A request must reach the step routes and expose
// the order id under the name the handler reads (validateUUIDParam "orderId"),
// while the plain order route keeps matching its own pattern.
func TestProduktionExtRoutes_MatchAgainstOrderSubtree(t *testing.T) {
	r := produktionRouter(t)
	const orderID = "550e8400-e29b-41d4-a716-446655440000"
	const stepID = "550e8400-e29b-41d4-a716-446655440001"

	tests := []struct {
		name       string
		method     string
		path       string
		wantParams map[string]string
	}{
		{
			name:       "list steps",
			method:     http.MethodGet,
			path:       "/api/v1/produktion/orders/" + orderID + "/steps",
			wantParams: map[string]string{"orderId": orderID},
		},
		{
			name:       "update step",
			method:     http.MethodPatch,
			path:       "/api/v1/produktion/orders/" + orderID + "/steps/" + stepID,
			wantParams: map[string]string{"orderId": orderID, "stepId": stepID},
		},
		{
			name:       "get order still resolves",
			method:     http.MethodGet,
			path:       "/api/v1/produktion/orders/" + orderID,
			wantParams: map[string]string{"id": orderID},
		},
		{
			name:       "start order still resolves",
			method:     http.MethodPost,
			path:       "/api/v1/produktion/orders/" + orderID + "/start",
			wantParams: map[string]string{"id": orderID},
		},
		{
			name:       "get bom",
			method:     http.MethodGet,
			path:       "/api/v1/produktion/boms/" + orderID,
			wantParams: map[string]string{"bomId": orderID},
		},
		{
			name:       "get quality check",
			method:     http.MethodGet,
			path:       "/api/v1/produktion/quality/" + orderID,
			wantParams: map[string]string{"checkId": orderID},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rctx := chi.NewRouteContext()
			if !r.Match(rctx, tt.method, tt.path) {
				t.Fatalf("no route matched %s %s", tt.method, tt.path)
			}
			for key, want := range tt.wantParams {
				if got := rctx.URLParam(key); got != want {
					t.Errorf("URLParam(%q) = %q, want %q (matched pattern %q)",
						key, got, want, rctx.RoutePattern())
				}
			}
		})
	}
}
