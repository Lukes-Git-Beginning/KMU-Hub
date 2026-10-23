package gateway

// The desktop client (api/inbox-client.ts) types GET /inbox/rules as
// RoutingRule[] and POST/PUT as RoutingRule; RoutingRulesEditor maps over the
// list body directly. These tests pin the wire shape against a real gRPC
// round-trip so a return to the wrapped proto response turns them red.

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/grpc"

	inboxv1 "github.com/kmuhub/kmuhub/proto/inbox/v1"
)

const shapeRuleID = "11111111-1111-1111-1111-111111111111"

type ruleShapeInboxServer struct {
	inboxv1.UnimplementedInboxServiceServer
}

func shapeRule() *inboxv1.RoutingRuleInfo {
	return &inboxv1.RoutingRuleInfo{Id: shapeRuleID, Name: "Support", Priority: 2, IsActive: true}
}

func (ruleShapeInboxServer) ListRoutingRules(context.Context, *inboxv1.ListRoutingRulesRequest) (*inboxv1.ListRoutingRulesResponse, error) {
	return &inboxv1.ListRoutingRulesResponse{Rules: []*inboxv1.RoutingRuleInfo{shapeRule()}}, nil
}

func (ruleShapeInboxServer) CreateRoutingRule(context.Context, *inboxv1.CreateRoutingRuleRequest) (*inboxv1.CreateRoutingRuleResponse, error) {
	return &inboxv1.CreateRoutingRuleResponse{Rule: shapeRule()}, nil
}

func (ruleShapeInboxServer) UpdateRoutingRule(context.Context, *inboxv1.UpdateRoutingRuleRequest) (*inboxv1.UpdateRoutingRuleResponse, error) {
	return &inboxv1.UpdateRoutingRuleResponse{Rule: shapeRule()}, nil
}

func newRuleShapeInboxRoutes(t *testing.T) *InboxRoutes {
	t.Helper()
	gs := grpc.NewServer()
	inboxv1.RegisterInboxServiceServer(gs, ruleShapeInboxServer{})
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() { _ = gs.Serve(l) }()
	t.Cleanup(gs.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("notification", l.Addr().String())
	t.Cleanup(reg.Close)
	return NewInboxRoutes(reg)
}

func TestHandleListRoutingRules_ReturnsBareArray(t *testing.T) {
	routes := newRuleShapeInboxRoutes(t)
	rec := httptest.NewRecorder()
	routes.HandleListRoutingRules(rec, httptest.NewRequest(http.MethodGet, "/api/v1/inbox/rules", nil))

	assertStatus(t, rec, http.StatusOK)
	var rules []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &rules); err != nil {
		t.Fatalf("body is not a JSON array: %v (%s)", err, rec.Body.String())
	}
	if len(rules) != 1 || rules[0]["id"] != shapeRuleID {
		t.Fatalf("unexpected rules: %s", rec.Body.String())
	}
}

func TestHandleCreateAndUpdateRoutingRule_ReturnBareRule(t *testing.T) {
	routes := newRuleShapeInboxRoutes(t)
	body := `{"name":"Support","conditions":{},"actions":[]}`

	for _, tc := range []struct {
		name   string
		status int
		call   func(*httptest.ResponseRecorder)
	}{
		{"create", http.StatusCreated, func(rec *httptest.ResponseRecorder) {
			routes.HandleCreateRoutingRule(rec, httptest.NewRequest(http.MethodPost, "/api/v1/inbox/rules", strings.NewReader(body)))
		}},
		{"update", http.StatusOK, func(rec *httptest.ResponseRecorder) {
			req := httptest.NewRequest(http.MethodPut, "/api/v1/inbox/rules/"+shapeRuleID, strings.NewReader(body))
			routes.HandleUpdateRoutingRule(rec, withChiURLParam(req, "id", shapeRuleID))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tc.call(rec)
			assertStatus(t, rec, tc.status)
			var rule map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &rule); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if rule["id"] != shapeRuleID || rule["name"] != "Support" {
				t.Fatalf("want the bare rule, got %s", rec.Body.String())
			}
		})
	}
}
