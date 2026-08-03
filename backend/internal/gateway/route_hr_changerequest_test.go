package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/middleware"
	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

func changeRequestListRequest(query string, perms []string, scopes map[string]string) *http.Request {
	r := httptest.NewRequest("GET", "/api/v1/hr/change-requests"+query, nil)
	ctx := context.WithValue(r.Context(), middleware.UserPermsKey, perms)
	ctx = context.WithValue(ctx, middleware.UserScopesKey, scopes)
	return r.WithContext(ctx)
}

// A proposer may read the route so they can see their own requests. If the
// handler honoured their scope parameter instead of forcing it, every employee
// could read the whole tenant's proposals — including salary-adjacent ones.
func TestChangeRequestListReach_ProposerIsAlwaysLimitedToOwnRows(t *testing.T) {
	for _, query := range []string{"", "?scope=mine", "?status=pending"} {
		r := changeRequestListRequest(query, []string{"team:self:propose"}, nil)
		ownOnly, scope := changeRequestListReach(r)
		if !ownOnly {
			t.Fatalf("query %q: ownOnly = false for a caller without team:data_personal:edit", query)
		}
		if scope != "" {
			t.Fatalf("query %q: actorScope = %q, want empty for a caller who cannot decide", query, scope)
		}
	}
}

func TestChangeRequestListReach_DeciderKeepsScopeAndHonoursTheQuery(t *testing.T) {
	perms := []string{"team:data_personal:edit"}
	scopes := map[string]string{"team:data_personal:edit": auth.ScopeTeam}

	ownOnly, scope := changeRequestListReach(changeRequestListRequest("", perms, scopes))
	if ownOnly {
		t.Fatal("inbox read collapsed to own rows for a caller who may decide")
	}
	if scope != auth.ScopeTeam {
		t.Fatalf("actorScope = %q, want %q", scope, auth.ScopeTeam)
	}

	ownOnly, _ = changeRequestListReach(changeRequestListRequest("?scope=mine", perms, scopes))
	if !ownOnly {
		t.Fatal("scope=mine ignored for a caller who may decide")
	}
}

// The desktop client reads camelCase (api/hr-change-requests.ts). The gateway's
// protojson options use proto names, so this mapping is hand-written — and a
// silent rename here is exactly the kind of contract break that only shows up
// as undefined in the UI.
func TestToChangeRequestBody_MatchesTheFrontendContract(t *testing.T) {
	created := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	decided := time.Date(2026, 8, 2, 9, 30, 0, 0, time.UTC)

	body := toChangeRequestBody(&hrv1.ProfileChangeRequest{
		Id:            "11111111-1111-1111-1111-111111111111",
		UserId:        "22222222-2222-2222-2222-222222222222",
		UserName:      "Markus Weber",
		Drawer:        "personal",
		Field:         "addressCity",
		FieldLabel:    "Wohnort",
		OldValue:      "Muenchen",
		NewValue:      "Hamburg",
		Status:        "approved",
		CreatedAt:     timestamppb.New(created),
		DecidedAt:     timestamppb.New(decided),
		DecidedByName: "Stefan Vogel",
	})

	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	want := map[string]any{
		"id":            "11111111-1111-1111-1111-111111111111",
		"userId":        "22222222-2222-2222-2222-222222222222",
		"userName":      "Markus Weber",
		"drawer":        "personal",
		"field":         "addressCity",
		"fieldLabel":    "Wohnort",
		"oldValue":      "Muenchen",
		"newValue":      "Hamburg",
		"status":        "approved",
		"createdAt":     "2026-08-01T10:00:00Z",
		"decidedAt":     "2026-08-02T09:30:00Z",
		"decidedByName": "Stefan Vogel",
	}
	for key, expected := range want {
		if decoded[key] != expected {
			t.Errorf("%s = %v, want %v", key, decoded[key], expected)
		}
	}

	// A pending request has no decision, and the frontend types decidedAt as
	// optional — it must be absent, not null.
	pending := toChangeRequestBody(&hrv1.ProfileChangeRequest{
		Id: "3", Status: "pending", CreatedAt: timestamppb.New(created),
	})
	rawPending, err := json.Marshal(pending)
	if err != nil {
		t.Fatalf("marshal pending: %v", err)
	}
	var decodedPending map[string]any
	if err := json.Unmarshal(rawPending, &decodedPending); err != nil {
		t.Fatalf("unmarshal pending: %v", err)
	}
	if _, present := decodedPending["decidedAt"]; present {
		t.Error("decidedAt present on a pending request")
	}
}

func TestChangeRequestHandlers_ServiceUnavailable(t *testing.T) {
	routes := NewHRRoutes(emptyRegistry(), nil)
	for name, handler := range map[string]http.HandlerFunc{
		"list":    routes.HandleListChangeRequests,
		"create":  routes.HandleCreateChangeRequest,
		"approve": routes.HandleApproveChangeRequest,
		"reject":  routes.HandleRejectChangeRequest,
		"cancel":  routes.HandleCancelChangeRequest,
	} {
		t.Run(name, func(t *testing.T) { testServiceUnavailable(t, handler) })
	}
}

func TestHandleListChangeRequests_RejectsAnUnknownStatusFilter(t *testing.T) {
	routes := NewHRRoutes(registryWithService("biz"), nil)
	rec := httptest.NewRecorder()
	req := changeRequestListRequest("?status=erledigt", []string{"team:data_personal:edit"}, nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.TenantIDKey, "00000000-0000-0000-0000-000000000001"))

	routes.HandleListChangeRequests(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}
