package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/plugin"
	pluginv1 "github.com/kmuhub/kmuhub/proto/plugin/v1"
)

// PluginGRPCServer implements the PluginService gRPC service
type PluginGRPCServer struct {
	pluginv1.UnimplementedPluginServiceServer
	svc *plugin.Service
}

// NewPluginGRPCServer creates a new PluginGRPCServer
func NewPluginGRPCServer(svc *plugin.Service) *PluginGRPCServer {
	return &PluginGRPCServer{svc: svc}
}

// ============================================================================
// Manifest CRUD
// ============================================================================

func (s *PluginGRPCServer) CreateManifest(ctx context.Context, req *pluginv1.CreateManifestRequest) (*pluginv1.CreateManifestResponse, error) {
	if req.GetSlug() == "" {
		return nil, status.Error(codes.InvalidArgument, "slug is required")
	}
	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	hookRegs := make([]models.HookRegistration, 0, len(req.GetHookRegistrations()))
	for _, hr := range req.GetHookRegistrations() {
		hookRegs = append(hookRegs, models.HookRegistration{
			HookType:   hr.GetHookType(),
			Module:     hr.GetModule(),
			EntityType: hr.GetEntityType(),
			Priority:   int(hr.GetPriority()),
		})
	}

	m := &models.PluginManifest{
		Slug:              req.GetSlug(),
		Name:              req.GetName(),
		Description:       req.GetDescription(),
		Version:           req.GetVersion(),
		Author:            req.GetAuthor(),
		SettingsSchema:    json.RawMessage(req.GetSettingsSchema()),
		Permissions:       req.GetPermissions(),
		HookRegistrations: hookRegs,
		WASMBinary:        req.GetWasmBinary(),
		PluginType:        models.PluginType(req.GetPluginType()),
	}

	result, err := s.svc.CreateManifest(ctx, m)
	if err != nil {
		return nil, mapPluginError(err)
	}

	return &pluginv1.CreateManifestResponse{Manifest: manifestToProto(result)}, nil
}

func (s *PluginGRPCServer) GetManifest(ctx context.Context, req *pluginv1.GetManifestRequest) (*pluginv1.GetManifestResponse, error) {
	id, err := uuid.Parse(req.GetManifestId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid manifest_id")
	}

	m, err := s.svc.GetManifest(ctx, id)
	if err != nil {
		return nil, mapPluginError(err)
	}

	return &pluginv1.GetManifestResponse{Manifest: manifestToProto(m)}, nil
}

func (s *PluginGRPCServer) ListManifests(ctx context.Context, req *pluginv1.ListManifestsRequest) (*pluginv1.ListManifestsResponse, error) {
	manifests, err := s.svc.ListManifests(ctx, req.GetPluginType())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	result := make([]*pluginv1.PluginManifestMsg, 0, len(manifests))
	for _, m := range manifests {
		result = append(result, manifestToProto(m))
	}
	return &pluginv1.ListManifestsResponse{Manifests: result}, nil
}

func (s *PluginGRPCServer) DeleteManifest(ctx context.Context, req *pluginv1.DeleteManifestRequest) (*pluginv1.DeleteManifestResponse, error) {
	id, err := uuid.Parse(req.GetManifestId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid manifest_id")
	}

	if err := s.svc.DeleteManifest(ctx, id); err != nil {
		return nil, mapPluginError(err)
	}

	return &pluginv1.DeleteManifestResponse{Success: true}, nil
}

// ============================================================================
// Installation Lifecycle
// ============================================================================

func (s *PluginGRPCServer) InstallPlugin(ctx context.Context, req *pluginv1.InstallPluginRequest) (*pluginv1.InstallPluginResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	manifestID, err := uuid.Parse(req.GetManifestId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid manifest_id")
	}
	installedBy, err := uuid.Parse(req.GetInstalledBy())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid installed_by")
	}

	inst, err := s.svc.InstallPlugin(ctx, tenantID, manifestID, installedBy)
	if err != nil {
		return nil, mapPluginError(err)
	}

	return &pluginv1.InstallPluginResponse{Installation: installationToProto(inst)}, nil
}

func (s *PluginGRPCServer) UninstallPlugin(ctx context.Context, req *pluginv1.UninstallPluginRequest) (*pluginv1.UninstallPluginResponse, error) {
	id, err := uuid.Parse(req.GetInstallationId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid installation_id")
	}

	if err := s.svc.UninstallPlugin(ctx, id); err != nil {
		return nil, mapPluginError(err)
	}
	return &pluginv1.UninstallPluginResponse{Success: true}, nil
}

func (s *PluginGRPCServer) EnablePlugin(ctx context.Context, req *pluginv1.EnablePluginRequest) (*pluginv1.EnablePluginResponse, error) {
	id, err := uuid.Parse(req.GetInstallationId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid installation_id")
	}

	inst, err := s.svc.EnablePlugin(ctx, id)
	if err != nil {
		return nil, mapPluginError(err)
	}
	return &pluginv1.EnablePluginResponse{Installation: installationToProto(inst)}, nil
}

func (s *PluginGRPCServer) DisablePlugin(ctx context.Context, req *pluginv1.DisablePluginRequest) (*pluginv1.DisablePluginResponse, error) {
	id, err := uuid.Parse(req.GetInstallationId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid installation_id")
	}

	inst, err := s.svc.DisablePlugin(ctx, id)
	if err != nil {
		return nil, mapPluginError(err)
	}
	return &pluginv1.DisablePluginResponse{Installation: installationToProto(inst)}, nil
}

func (s *PluginGRPCServer) GetInstallation(ctx context.Context, req *pluginv1.GetInstallationRequest) (*pluginv1.GetInstallationResponse, error) {
	id, err := uuid.Parse(req.GetInstallationId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid installation_id")
	}

	inst, err := s.svc.GetInstallation(ctx, id)
	if err != nil {
		return nil, mapPluginError(err)
	}
	return &pluginv1.GetInstallationResponse{Installation: installationToProto(inst)}, nil
}

func (s *PluginGRPCServer) ListInstallations(ctx context.Context, req *pluginv1.ListInstallationsRequest) (*pluginv1.ListInstallationsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	installations, err := s.svc.ListInstallations(ctx, tenantID, req.GetStatus())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	result := make([]*pluginv1.PluginInstallationMsg, 0, len(installations))
	for _, inst := range installations {
		result = append(result, installationWithManifestToProto(inst))
	}
	return &pluginv1.ListInstallationsResponse{Installations: result}, nil
}

// ============================================================================
// Permissions
// ============================================================================

func (s *PluginGRPCServer) ApprovePermissions(ctx context.Context, req *pluginv1.ApprovePermissionsRequest) (*pluginv1.ApprovePermissionsResponse, error) {
	installationID, err := uuid.Parse(req.GetInstallationId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid installation_id")
	}
	grantedBy, err := uuid.Parse(req.GetGrantedBy())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid granted_by")
	}

	if err := s.svc.ApprovePermissions(ctx, installationID, req.GetPermissions(), grantedBy); err != nil {
		return nil, mapPluginError(err)
	}
	return &pluginv1.ApprovePermissionsResponse{Success: true}, nil
}

func (s *PluginGRPCServer) ListGrantedPermissions(ctx context.Context, req *pluginv1.ListGrantedPermissionsRequest) (*pluginv1.ListGrantedPermissionsResponse, error) {
	installationID, err := uuid.Parse(req.GetInstallationId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid installation_id")
	}

	perms, err := s.svc.ListGrantedPermissions(ctx, installationID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pluginv1.ListGrantedPermissionsResponse{Permissions: perms}, nil
}

// ============================================================================
// Settings
// ============================================================================

func (s *PluginGRPCServer) GetPluginSettings(ctx context.Context, req *pluginv1.GetPluginSettingsRequest) (*pluginv1.GetPluginSettingsResponse, error) {
	installationID, err := uuid.Parse(req.GetInstallationId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid installation_id")
	}

	settings, err := s.svc.GetPluginSettings(ctx, installationID)
	if err != nil {
		return nil, mapPluginError(err)
	}
	return &pluginv1.GetPluginSettingsResponse{Settings: string(settings)}, nil
}

func (s *PluginGRPCServer) UpdatePluginSettings(ctx context.Context, req *pluginv1.UpdatePluginSettingsRequest) (*pluginv1.UpdatePluginSettingsResponse, error) {
	installationID, err := uuid.Parse(req.GetInstallationId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid installation_id")
	}

	if err := s.svc.UpdatePluginSettings(ctx, installationID, json.RawMessage(req.GetSettings())); err != nil {
		return nil, mapPluginError(err)
	}
	return &pluginv1.UpdatePluginSettingsResponse{Success: true}, nil
}

func (s *PluginGRPCServer) GetPluginSettingsSchema(ctx context.Context, req *pluginv1.GetPluginSettingsSchemaRequest) (*pluginv1.GetPluginSettingsSchemaResponse, error) {
	installationID, err := uuid.Parse(req.GetInstallationId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid installation_id")
	}

	schema, err := s.svc.GetPluginSettingsSchema(ctx, installationID)
	if err != nil {
		return nil, mapPluginError(err)
	}
	return &pluginv1.GetPluginSettingsSchemaResponse{Schema: string(schema)}, nil
}

// ============================================================================
// Validation Rules
// ============================================================================

// ruleTenant resolves the caller's tenant for the validation_rules and
// workflow_rules paths from the propagated x-tenant-id metadata rather than
// from the request body. Both tables are under RLS since migration 000269, so
// a body-supplied foreign tenant now fails the policy's WITH CHECK anyway —
// but it fails as an opaque database error deep inside the repository. Reading
// the tenant here keeps the rejection at the service boundary, where it can say
// what is wrong, and it makes the reads (which pass the tenant as a plain
// predicate) consistent with the writes.
//
// The tenant_id field on the requests stays in the proto for compatibility;
// the gateway already fills it from the JWT (route_plugin.go), so no caller
// loses anything by it being ignored here.
func ruleTenant(ctx context.Context) (uuid.UUID, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return uuid.Nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	return tenantID, nil
}

func (s *PluginGRPCServer) CreateValidationRule(ctx context.Context, req *pluginv1.CreateValidationRuleRequest) (*pluginv1.CreateValidationRuleResponse, error) {
	tenantID, err := ruleTenant(ctx)
	if err != nil {
		return nil, err
	}

	rule := &models.ValidationRule{
		TenantID:     tenantID,
		Name:         req.GetName(),
		Description:  req.GetDescription(),
		EntityType:   req.GetEntityType(),
		FieldName:    req.GetFieldName(),
		RuleType:     models.ValidationRuleType(req.GetRuleType()),
		RuleConfig:   json.RawMessage(req.GetRuleConfig()),
		ErrorMessage: req.GetErrorMessage(),
		Priority:     int(req.GetPriority()),
	}

	if req.GetInstallationId() != "" {
		instID, parseErr := uuid.Parse(req.GetInstallationId())
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid installation_id")
		}
		rule.InstallationID = &instID
	}

	result, err := s.svc.CreateValidationRule(ctx, rule)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pluginv1.CreateValidationRuleResponse{Rule: validationRuleToProto(result)}, nil
}

func (s *PluginGRPCServer) ListValidationRules(ctx context.Context, req *pluginv1.ListValidationRulesRequest) (*pluginv1.ListValidationRulesResponse, error) {
	tenantID, err := ruleTenant(ctx)
	if err != nil {
		return nil, err
	}

	rules, err := s.svc.ListValidationRules(ctx, tenantID, req.GetEntityType())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	result := make([]*pluginv1.ValidationRuleMsg, 0, len(rules))
	for _, r := range rules {
		result = append(result, validationRuleToProto(r))
	}
	return &pluginv1.ListValidationRulesResponse{Rules: result}, nil
}

func (s *PluginGRPCServer) UpdateValidationRule(ctx context.Context, req *pluginv1.UpdateValidationRuleRequest) (*pluginv1.UpdateValidationRuleResponse, error) {
	ruleID, err := uuid.Parse(req.GetRuleId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid rule_id")
	}

	rule := &models.ValidationRule{
		ID:           ruleID,
		Name:         req.GetName(),
		Description:  req.GetDescription(),
		RuleConfig:   json.RawMessage(req.GetRuleConfig()),
		ErrorMessage: req.GetErrorMessage(),
		Priority:     int(req.GetPriority()),
		Enabled:      req.GetEnabled(),
	}

	result, err := s.svc.UpdateValidationRule(ctx, rule)
	if err != nil {
		return nil, mapPluginError(err)
	}
	return &pluginv1.UpdateValidationRuleResponse{Rule: validationRuleToProto(result)}, nil
}

func (s *PluginGRPCServer) DeleteValidationRule(ctx context.Context, req *pluginv1.DeleteValidationRuleRequest) (*pluginv1.DeleteValidationRuleResponse, error) {
	ruleID, err := uuid.Parse(req.GetRuleId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid rule_id")
	}

	if err := s.svc.DeleteValidationRule(ctx, ruleID); err != nil {
		return nil, mapPluginError(err)
	}
	return &pluginv1.DeleteValidationRuleResponse{Success: true}, nil
}

// ============================================================================
// Workflow Rules
// ============================================================================

func (s *PluginGRPCServer) CreateWorkflowRule(ctx context.Context, req *pluginv1.CreateWorkflowRuleRequest) (*pluginv1.CreateWorkflowRuleResponse, error) {
	tenantID, err := ruleTenant(ctx)
	if err != nil {
		return nil, err
	}

	rule := &models.WorkflowRule{
		TenantID:     tenantID,
		Name:         req.GetName(),
		Description:  req.GetDescription(),
		TriggerEvent: req.GetTriggerEvent(),
		Conditions:   json.RawMessage(req.GetConditions()),
		Actions:      json.RawMessage(req.GetActions()),
		Priority:     int(req.GetPriority()),
	}

	if req.GetInstallationId() != "" {
		instID, parseErr := uuid.Parse(req.GetInstallationId())
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid installation_id")
		}
		rule.InstallationID = &instID
	}

	result, err := s.svc.CreateWorkflowRule(ctx, rule)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pluginv1.CreateWorkflowRuleResponse{Rule: workflowRuleToProto(result)}, nil
}

func (s *PluginGRPCServer) ListWorkflowRules(ctx context.Context, req *pluginv1.ListWorkflowRulesRequest) (*pluginv1.ListWorkflowRulesResponse, error) {
	tenantID, err := ruleTenant(ctx)
	if err != nil {
		return nil, err
	}

	rules, err := s.svc.ListWorkflowRules(ctx, tenantID, req.GetTriggerEvent())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	result := make([]*pluginv1.WorkflowRuleMsg, 0, len(rules))
	for _, r := range rules {
		result = append(result, workflowRuleToProto(r))
	}
	return &pluginv1.ListWorkflowRulesResponse{Rules: result}, nil
}

func (s *PluginGRPCServer) UpdateWorkflowRule(ctx context.Context, req *pluginv1.UpdateWorkflowRuleRequest) (*pluginv1.UpdateWorkflowRuleResponse, error) {
	ruleID, err := uuid.Parse(req.GetRuleId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid rule_id")
	}

	rule := &models.WorkflowRule{
		ID:          ruleID,
		Name:        req.GetName(),
		Description: req.GetDescription(),
		Conditions:  json.RawMessage(req.GetConditions()),
		Actions:     json.RawMessage(req.GetActions()),
		Priority:    int(req.GetPriority()),
		Enabled:     req.GetEnabled(),
	}

	result, err := s.svc.UpdateWorkflowRule(ctx, rule)
	if err != nil {
		return nil, mapPluginError(err)
	}
	return &pluginv1.UpdateWorkflowRuleResponse{Rule: workflowRuleToProto(result)}, nil
}

func (s *PluginGRPCServer) DeleteWorkflowRule(ctx context.Context, req *pluginv1.DeleteWorkflowRuleRequest) (*pluginv1.DeleteWorkflowRuleResponse, error) {
	ruleID, err := uuid.Parse(req.GetRuleId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid rule_id")
	}

	if err := s.svc.DeleteWorkflowRule(ctx, ruleID); err != nil {
		return nil, mapPluginError(err)
	}
	return &pluginv1.DeleteWorkflowRuleResponse{Success: true}, nil
}

// ============================================================================
// Industry Templates
// ============================================================================

func (s *PluginGRPCServer) ListIndustryTemplates(ctx context.Context, req *pluginv1.ListIndustryTemplatesRequest) (*pluginv1.ListIndustryTemplatesResponse, error) {
	templates, err := s.svc.ListIndustryTemplates(ctx, req.GetIndustry())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	result := make([]*pluginv1.IndustryTemplateMsg, 0, len(templates))
	for _, t := range templates {
		result = append(result, industryTemplateToProto(t))
	}
	return &pluginv1.ListIndustryTemplatesResponse{Templates: result}, nil
}

// ApplyIndustryTemplate writes into validation_rules and workflow_rules, so it
// takes the tenant from the context for the same reason the rule handlers do.
func (s *PluginGRPCServer) ApplyIndustryTemplate(ctx context.Context, req *pluginv1.ApplyIndustryTemplateRequest) (*pluginv1.ApplyIndustryTemplateResponse, error) {
	tenantID, err := ruleTenant(ctx)
	if err != nil {
		return nil, err
	}
	templateID, err := uuid.Parse(req.GetTemplateId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid template_id")
	}
	appliedBy, err := uuid.Parse(req.GetAppliedBy())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid applied_by")
	}

	if err := s.svc.ApplyIndustryTemplate(ctx, tenantID, templateID, appliedBy); err != nil {
		return nil, mapPluginError(err)
	}
	return &pluginv1.ApplyIndustryTemplateResponse{Success: true}, nil
}

// ============================================================================
// Execution Logs
// ============================================================================

func (s *PluginGRPCServer) ListExecutionLogs(ctx context.Context, req *pluginv1.ListExecutionLogsRequest) (*pluginv1.ListExecutionLogsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	var installationID *uuid.UUID
	if req.GetInstallationId() != "" {
		id, parseErr := uuid.Parse(req.GetInstallationId())
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid installation_id")
		}
		installationID = &id
	}

	logs, err := s.svc.ListExecutionLogs(ctx, tenantID, installationID, int(req.GetLimit()))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	result := make([]*pluginv1.ExecutionLogMsg, 0, len(logs))
	for _, l := range logs {
		result = append(result, executionLogToProto(l))
	}
	return &pluginv1.ListExecutionLogsResponse{Logs: result}, nil
}

// ============================================================================
// KV Store
// ============================================================================

func (s *PluginGRPCServer) KVGet(ctx context.Context, req *pluginv1.KVGetRequest) (*pluginv1.KVGetResponse, error) {
	installationID, err := uuid.Parse(req.GetInstallationId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid installation_id")
	}

	value, found, err := s.svc.KVGet(ctx, installationID, req.GetKey())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pluginv1.KVGetResponse{Value: string(value), Found: found}, nil
}

func (s *PluginGRPCServer) KVSet(ctx context.Context, req *pluginv1.KVSetRequest) (*pluginv1.KVSetResponse, error) {
	installationID, err := uuid.Parse(req.GetInstallationId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid installation_id")
	}

	if err := s.svc.KVSet(ctx, installationID, req.GetKey(), json.RawMessage(req.GetValue())); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pluginv1.KVSetResponse{Success: true}, nil
}

func (s *PluginGRPCServer) KVDelete(ctx context.Context, req *pluginv1.KVDeleteRequest) (*pluginv1.KVDeleteResponse, error) {
	installationID, err := uuid.Parse(req.GetInstallationId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid installation_id")
	}

	if err := s.svc.KVDelete(ctx, installationID, req.GetKey()); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pluginv1.KVDeleteResponse{Success: true}, nil
}

func (s *PluginGRPCServer) KVList(ctx context.Context, req *pluginv1.KVListRequest) (*pluginv1.KVListResponse, error) {
	installationID, err := uuid.Parse(req.GetInstallationId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid installation_id")
	}

	entries, err := s.svc.KVList(ctx, installationID, req.GetKeyPrefix())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	result := make(map[string]string, len(entries))
	for k, v := range entries {
		result[k] = string(v)
	}
	return &pluginv1.KVListResponse{Entries: result}, nil
}

// ============================================================================
// Hook Execution
// ============================================================================

func (s *PluginGRPCServer) ExecuteHooks(ctx context.Context, req *pluginv1.ExecuteHooksRequest) (*pluginv1.ExecuteHooksResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	// Find active installations for this hook
	installations, err := s.svc.ListActiveInstallationsForHook(ctx, tenantID, req.GetHookType(), req.GetModule(), req.GetEntityType())
	if err != nil {
		slog.Warn("failed to list active installations for hook",
			"error", err,
			"hook_type", req.GetHookType(),
		)
		return &pluginv1.ExecuteHooksResponse{Modified: false}, nil
	}

	if len(installations) == 0 {
		return &pluginv1.ExecuteHooksResponse{Modified: false}, nil
	}

	// For config-based plugins, run validation rules
	validationErrors := s.svc.ValidateEntity(ctx, tenantID, req.GetEntityType(), structToMap(req.GetEntityData()))

	var results []*pluginv1.HookExecutionResult
	for _, inst := range installations {
		results = append(results, &pluginv1.HookExecutionResult{
			PluginSlug: inst.ManifestID.String(),
			Status:     "success",
		})
	}

	if len(validationErrors) > 0 {
		return &pluginv1.ExecuteHooksResponse{
			Modified: false,
			Results:  results,
		}, nil
	}

	return &pluginv1.ExecuteHooksResponse{
		Modified:     false,
		ModifiedData: req.GetEntityData(),
		Results:      results,
	}, nil
}

func (s *PluginGRPCServer) ValidateEntity(ctx context.Context, req *pluginv1.ValidateEntityRequest) (*pluginv1.ValidateEntityResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	data := structToMap(req.GetEntityData())
	fieldErrors := s.svc.ValidateEntity(ctx, tenantID, req.GetEntityType(), data)

	var protoErrors []*pluginv1.ValidationError
	for _, fe := range fieldErrors {
		protoErrors = append(protoErrors, &pluginv1.ValidationError{
			Field:   fe.Field,
			Rule:    fe.Rule,
			Message: fe.Message,
		})
	}

	return &pluginv1.ValidateEntityResponse{
		Valid:  len(protoErrors) == 0,
		Errors: protoErrors,
	}, nil
}

// ============================================================================
// Helpers
// ============================================================================

func structToMap(s *structpb.Struct) map[string]any {
	if s == nil {
		return make(map[string]any)
	}
	return s.AsMap()
}

func manifestToProto(m *models.PluginManifest) *pluginv1.PluginManifestMsg {
	hookRegs := make([]*pluginv1.HookRegistrationMsg, 0, len(m.HookRegistrations))
	for _, hr := range m.HookRegistrations {
		hookRegs = append(hookRegs, &pluginv1.HookRegistrationMsg{
			HookType:   hr.HookType,
			Module:     hr.Module,
			EntityType: hr.EntityType,
			Priority:   int32(hr.Priority),
		})
	}

	return &pluginv1.PluginManifestMsg{
		Id:                m.ID.String(),
		Slug:              m.Slug,
		Name:              m.Name,
		Description:       m.Description,
		Version:           m.Version,
		Author:            m.Author,
		SettingsSchema:    string(m.SettingsSchema),
		Permissions:       m.Permissions,
		HookRegistrations: hookRegs,
		WasmBinaryHash:    m.WASMBinaryHash,
		PluginType:        string(m.PluginType),
		CreatedAt:         timestamppb.New(m.CreatedAt),
		UpdatedAt:         timestamppb.New(m.UpdatedAt),
	}
}

func installationToProto(inst *models.PluginInstallation) *pluginv1.PluginInstallationMsg {
	msg := &pluginv1.PluginInstallationMsg{
		Id:          inst.ID.String(),
		TenantId:    inst.TenantID.String(),
		ManifestId:  inst.ManifestID.String(),
		Status:      string(inst.Status),
		Settings:    string(inst.Settings),
		InstalledBy: inst.InstalledBy.String(),
		CreatedAt:   timestamppb.New(inst.CreatedAt),
		UpdatedAt:   timestamppb.New(inst.UpdatedAt),
	}
	if inst.ErrorMessage != nil {
		msg.ErrorMessage = *inst.ErrorMessage
	}
	return msg
}

func installationWithManifestToProto(inst *models.PluginInstallationWithManifest) *pluginv1.PluginInstallationMsg {
	msg := installationToProto(&inst.PluginInstallation)
	msg.ManifestSlug = inst.ManifestSlug
	msg.ManifestName = inst.ManifestName
	msg.ManifestVersion = inst.ManifestVersion
	msg.PluginType = string(inst.PluginType)
	msg.RequiredPermissions = inst.Permissions
	msg.GrantedPermissions = inst.Granted
	msg.SettingsSchema = string(inst.SettingsSchema)
	return msg
}

func validationRuleToProto(r *models.ValidationRule) *pluginv1.ValidationRuleMsg {
	msg := &pluginv1.ValidationRuleMsg{
		Id:           r.ID.String(),
		TenantId:     r.TenantID.String(),
		Name:         r.Name,
		Description:  r.Description,
		EntityType:   r.EntityType,
		FieldName:    r.FieldName,
		RuleType:     string(r.RuleType),
		RuleConfig:   string(r.RuleConfig),
		ErrorMessage: r.ErrorMessage,
		Priority:     int32(r.Priority),
		Enabled:      r.Enabled,
		CreatedAt:    timestamppb.New(r.CreatedAt),
		UpdatedAt:    timestamppb.New(r.UpdatedAt),
	}
	if r.InstallationID != nil {
		msg.InstallationId = r.InstallationID.String()
	}
	return msg
}

func workflowRuleToProto(r *models.WorkflowRule) *pluginv1.WorkflowRuleMsg {
	msg := &pluginv1.WorkflowRuleMsg{
		Id:           r.ID.String(),
		TenantId:     r.TenantID.String(),
		Name:         r.Name,
		Description:  r.Description,
		TriggerEvent: r.TriggerEvent,
		Conditions:   string(r.Conditions),
		Actions:      string(r.Actions),
		Priority:     int32(r.Priority),
		Enabled:      r.Enabled,
		CreatedAt:    timestamppb.New(r.CreatedAt),
		UpdatedAt:    timestamppb.New(r.UpdatedAt),
	}
	if r.InstallationID != nil {
		msg.InstallationId = r.InstallationID.String()
	}
	return msg
}

func industryTemplateToProto(t *models.IndustryTemplate) *pluginv1.IndustryTemplateMsg {
	return &pluginv1.IndustryTemplateMsg{
		Id:              t.ID.String(),
		Slug:            t.Slug,
		Name:            t.Name,
		Description:     t.Description,
		Industry:        t.Industry,
		Icon:            t.Icon,
		CustomFields:    string(t.CustomFields),
		ValidationRules: string(t.ValidationRules),
		WorkflowRules:   string(t.WorkflowRules),
		DefaultSettings: string(t.DefaultSettings),
	}
}

func executionLogToProto(l *models.PluginExecutionLog) *pluginv1.ExecutionLogMsg {
	msg := &pluginv1.ExecutionLogMsg{
		Id:             l.ID.String(),
		InstallationId: l.InstallationID.String(),
		HookType:       l.HookType,
		Module:         l.Module,
		EntityType:     l.EntityType,
		DurationMs:     int32(l.DurationMs),
		Status:         string(l.Status),
		CreatedAt:      timestamppb.New(l.CreatedAt),
	}
	if l.EntityID != nil {
		msg.EntityId = l.EntityID.String()
	}
	if l.ErrorMessage != nil {
		msg.ErrorMessage = *l.ErrorMessage
	}
	return msg
}

func mapPluginError(err error) error {
	switch {
	case err == nil:
		return nil
	case isNotFound(err):
		return status.Error(codes.NotFound, err.Error())
	case isAlreadyExists(err):
		return status.Error(codes.AlreadyExists, err.Error())
	case isInvalidArgument(err):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, plugin.ErrManifestImmutable):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, plugin.ErrPluginHasInstallations):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		slog.Error("unhandled plugin service error", "error", err)
		return status.Error(codes.Internal, "internal error")
	}
}

func isNotFound(err error) bool {
	return errors.Is(err, plugin.ErrManifestNotFound) ||
		errors.Is(err, plugin.ErrInstallationNotFound) ||
		errors.Is(err, plugin.ErrValidationRuleNotFound) ||
		errors.Is(err, plugin.ErrWorkflowRuleNotFound) ||
		errors.Is(err, plugin.ErrTemplateNotFound) ||
		errors.Is(err, plugin.ErrKVKeyNotFound)
}

func isAlreadyExists(err error) bool {
	return errors.Is(err, plugin.ErrManifestSlugExists) ||
		errors.Is(err, plugin.ErrAlreadyInstalled)
}

func isInvalidArgument(err error) bool {
	return errors.Is(err, plugin.ErrInvalidSettings) ||
		errors.Is(err, plugin.ErrInvalidSettingsSchema) ||
		errors.Is(err, plugin.ErrInvalidRuleConfig) ||
		errors.Is(err, plugin.ErrPermissionNotDeclared) ||
		errors.Is(err, plugin.ErrWASMBinaryRequired) ||
		errors.Is(err, plugin.ErrWASMBinaryNotAllowed)
}
