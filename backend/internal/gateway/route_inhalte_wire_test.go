package gateway

// Integration and plugin routes as the desktop reads them (integration-types.ts, plugin-client.ts):
// a channel mapping keeps `is_active:false` and `modules:[]`, an integration config keeps
// `is_active:false`, and the plugin lists sit under the key the client selects. A loopback gRPC
// server plays the notification and the plugin service.

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	notificationv1 "github.com/kmuhub/kmuhub/proto/notification/v1"
	pluginv1 "github.com/kmuhub/kmuhub/proto/plugin/v1"
)

type inhalteNotificationStub struct {
	notificationv1.UnimplementedNotificationServiceServer
}

func (inhalteNotificationStub) ListIntegrationConfigs(context.Context, *notificationv1.ListIntegrationConfigsRequest) (*notificationv1.ListIntegrationConfigsResponse, error) {
	return &notificationv1.ListIntegrationConfigsResponse{
		Configs: []*notificationv1.IntegrationConfigInfo{{Id: "c1", Platform: "slack"}},
	}, nil
}

func (inhalteNotificationStub) GetIntegrationConfig(context.Context, *notificationv1.GetIntegrationConfigRequest) (*notificationv1.GetIntegrationConfigResponse, error) {
	return &notificationv1.GetIntegrationConfigResponse{Config: &notificationv1.IntegrationConfigInfo{Id: "c1", Platform: "slack"}}, nil
}

func (inhalteNotificationStub) ListChannelMappings(context.Context, *notificationv1.ListChannelMappingsRequest) (*notificationv1.ListChannelMappingsResponse, error) {
	return &notificationv1.ListChannelMappingsResponse{
		Mappings: []*notificationv1.ChannelMappingInfo{{Id: "m1", ConfigId: "c1", ChannelId: "C1", ChannelName: "general"}},
	}, nil
}

type inhaltePluginStub struct {
	pluginv1.UnimplementedPluginServiceServer
}

func (inhaltePluginStub) ListManifests(context.Context, *pluginv1.ListManifestsRequest) (*pluginv1.ListManifestsResponse, error) {
	return &pluginv1.ListManifestsResponse{Manifests: []*pluginv1.PluginManifestMsg{{Id: "p1"}}}, nil
}

func (inhaltePluginStub) ListInstallations(context.Context, *pluginv1.ListInstallationsRequest) (*pluginv1.ListInstallationsResponse, error) {
	return &pluginv1.ListInstallationsResponse{Installations: []*pluginv1.PluginInstallationMsg{{Id: "i1"}}}, nil
}

func (inhaltePluginStub) ListValidationRules(context.Context, *pluginv1.ListValidationRulesRequest) (*pluginv1.ListValidationRulesResponse, error) {
	return &pluginv1.ListValidationRulesResponse{Rules: []*pluginv1.ValidationRuleMsg{{Id: "v1"}}}, nil
}

func (inhaltePluginStub) ListWorkflowRules(context.Context, *pluginv1.ListWorkflowRulesRequest) (*pluginv1.ListWorkflowRulesResponse, error) {
	return &pluginv1.ListWorkflowRulesResponse{Rules: []*pluginv1.WorkflowRuleMsg{{Id: "w1"}}}, nil
}

func (inhaltePluginStub) ListIndustryTemplates(context.Context, *pluginv1.ListIndustryTemplatesRequest) (*pluginv1.ListIndustryTemplatesResponse, error) {
	return &pluginv1.ListIndustryTemplatesResponse{Templates: []*pluginv1.IndustryTemplateMsg{{Id: "t1"}}}, nil
}

func (inhaltePluginStub) ListExecutionLogs(context.Context, *pluginv1.ListExecutionLogsRequest) (*pluginv1.ListExecutionLogsResponse, error) {
	return &pluginv1.ListExecutionLogsResponse{Logs: []*pluginv1.ExecutionLogMsg{{Id: "l1"}}}, nil
}

func inhalteRegistry(t *testing.T, name string, register func(*grpc.Server)) *ServiceRegistry {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer()
	register(srv)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register(name, lis.Addr().String())
	t.Cleanup(reg.Close)
	return reg
}

func inhalteCall(t *testing.T, h http.HandlerFunc, params map[string]string) map[string]any {
	t.Helper()
	req := withAuth(httptest.NewRequest(http.MethodGet, "/x", nil), testHRUserID, testTenantID)
	for k, v := range params {
		req = withChiURLParam(req, k, v)
	}
	rec := httptest.NewRecorder()
	h(rec, req)
	assertStatus(t, rec, http.StatusOK)
	return decodeBody(t, rec)
}

func TestInhalteWire_IntegrationConfigAndMappingKeepZeroValues(t *testing.T) {
	reg := inhalteRegistry(t, "notification", func(s *grpc.Server) {
		notificationv1.RegisterNotificationServiceServer(s, inhalteNotificationStub{})
	})
	routes := NewIntegrationRoutes(reg)

	list := inhalteCall(t, routes.HandleListConfigs, nil)
	cfg := list["configs"].([]any)[0].(map[string]any)
	assert.Equal(t, false, cfg["is_active"], "inactive config must say so")

	got := inhalteCall(t, routes.HandleGetConfig, map[string]string{"platform": "slack"})
	assert.Equal(t, false, got["config"].(map[string]any)["is_active"])

	mappings := inhalteCall(t, routes.HandleListMappings, map[string]string{"platform": "slack"})
	m := mappings["mappings"].([]any)[0].(map[string]any)
	assert.Equal(t, false, m["is_active"], "inactive mapping must say so")
	assert.Equal(t, []any{}, m["modules"], "a mapping without modules is [], not absent")
}

func TestInhalteWire_PluginListsSitUnderTheKeyTheClientSelects(t *testing.T) {
	reg := inhalteRegistry(t, "plugin", func(s *grpc.Server) {
		pluginv1.RegisterPluginServiceServer(s, inhaltePluginStub{})
	})
	routes := NewPluginRoutes(reg, nil)

	for _, tc := range []struct {
		key string
		h   http.HandlerFunc
	}{
		{"manifests", routes.HandleListManifests},
		{"installations", routes.HandleListInstallations},
		{"rules", routes.HandleListValidationRules},
		{"rules", routes.HandleListWorkflowRules},
		{"templates", routes.HandleListTemplates},
		{"logs", routes.HandleListExecutionLogs},
	} {
		got := inhalteCall(t, tc.h, nil)
		assert.Len(t, got[tc.key], 1, "list must sit under %q", tc.key)
	}
}
