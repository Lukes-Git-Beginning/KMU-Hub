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

	"github.com/kmuhub/kmuhub/internal/automation/workflow"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	automationv1 "github.com/kmuhub/kmuhub/proto/automation/v1"
)

// AutomationGRPCServer implements the AutomationService gRPC server.
type AutomationGRPCServer struct {
	automationv1.UnimplementedAutomationServiceServer
	svc *workflow.Service
}

// NewAutomationGRPCServer creates a new automation gRPC server.
func NewAutomationGRPCServer(svc *workflow.Service) *AutomationGRPCServer {
	return &AutomationGRPCServer{svc: svc}
}

// ============================================================================
// CRUD (5 RPCs)
// ============================================================================

func (s *AutomationGRPCServer) CreateAutomation(ctx context.Context, req *automationv1.CreateAutomationRequest) (*automationv1.CreateAutomationResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing tenant_id: %v", err)
	}

	ownerID, err := uuid.Parse(req.GetOwnerId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid owner_id: %v", err)
	}

	auto := &models.Automation{
		TenantID:    tenantID,
		Name:        req.GetName(),
		Description: req.GetDescription(),
		Scope:       scopeFromProto(req.GetScope()),
		OwnerID:     ownerID,
		TriggerType: req.GetTriggerType(),
		MaxSteps:    int(req.GetMaxSteps()),
	}

	if req.GetTriggerConfig() != nil {
		auto.TriggerConfig = automationStructToJSON(req.GetTriggerConfig())
	}
	if req.GetConditions() != nil {
		auto.Conditions = automationStructToJSON(req.GetConditions())
	}
	if req.GetActions() != nil {
		auto.Actions = automationListToJSON(req.GetActions())
	}

	if err := s.svc.Create(ctx, auto); err != nil {
		return nil, mapDomainError(err)
	}

	return &automationv1.CreateAutomationResponse{
		Automation: automationToProto(auto),
	}, nil
}

func (s *AutomationGRPCServer) UpdateAutomation(ctx context.Context, req *automationv1.UpdateAutomationRequest) (*automationv1.UpdateAutomationResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing tenant_id: %v", err)
	}

	id, err := uuid.Parse(req.GetAutomationId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid automation_id: %v", err)
	}

	auto, err := s.svc.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, mapDomainError(err)
	}

	if req.Name != nil {
		auto.Name = *req.Name
	}
	if req.Description != nil {
		auto.Description = *req.Description
	}
	if req.Scope != nil {
		auto.Scope = scopeFromProto(*req.Scope)
	}
	if req.TriggerType != nil {
		auto.TriggerType = *req.TriggerType
	}
	if req.GetTriggerConfig() != nil {
		auto.TriggerConfig = automationStructToJSON(req.GetTriggerConfig())
	}
	if req.GetConditions() != nil {
		auto.Conditions = automationStructToJSON(req.GetConditions())
	}
	if req.GetActions() != nil {
		auto.Actions = automationListToJSON(req.GetActions())
	}
	if req.MaxSteps != nil {
		auto.MaxSteps = int(*req.MaxSteps)
	}

	if err := s.svc.Update(ctx, auto); err != nil {
		return nil, mapDomainError(err)
	}

	return &automationv1.UpdateAutomationResponse{
		Automation: automationToProto(auto),
	}, nil
}

func (s *AutomationGRPCServer) DeleteAutomation(ctx context.Context, req *automationv1.DeleteAutomationRequest) (*automationv1.DeleteAutomationResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing tenant_id: %v", err)
	}

	id, err := uuid.Parse(req.GetAutomationId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid automation_id: %v", err)
	}

	if err := s.svc.Delete(ctx, id, tenantID); err != nil {
		return nil, mapDomainError(err)
	}

	return &automationv1.DeleteAutomationResponse{}, nil
}

func (s *AutomationGRPCServer) GetAutomation(ctx context.Context, req *automationv1.GetAutomationRequest) (*automationv1.GetAutomationResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing tenant_id: %v", err)
	}

	id, err := uuid.Parse(req.GetAutomationId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid automation_id: %v", err)
	}

	auto, err := s.svc.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &automationv1.GetAutomationResponse{
		Automation: automationToProto(auto),
	}, nil
}

func (s *AutomationGRPCServer) ListAutomations(ctx context.Context, req *automationv1.ListAutomationsRequest) (*automationv1.ListAutomationsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing tenant_id: %v", err)
	}

	filter := workflow.ListFilter{
		TenantID: tenantID,
		Limit:    int(req.GetLimit()),
		Offset:   int(req.GetOffset()),
	}

	if req.GetOwnerId() != "" {
		ownerID, err := uuid.Parse(req.GetOwnerId())
		if err == nil {
			filter.OwnerID = &ownerID
		}
	}
	if req.GetScope() != automationv1.AutomationScope_SCOPE_UNSPECIFIED {
		scope := scopeFromProto(req.GetScope())
		filter.Scope = &scope
	}
	if req.GetTriggerType() != "" {
		tt := req.GetTriggerType()
		filter.TriggerType = &tt
	}
	if req.IsActive != nil {
		filter.IsActive = req.IsActive
	}

	autos, total, err := s.svc.List(ctx, filter)
	if err != nil {
		return nil, mapDomainError(err)
	}

	protoAutos := make([]*automationv1.AutomationInfo, len(autos))
	for i, a := range autos {
		protoAutos[i] = automationToProto(a)
	}

	return &automationv1.ListAutomationsResponse{
		Automations: protoAutos,
		TotalCount:  int32(total),
	}, nil
}

// ============================================================================
// State (2 RPCs)
// ============================================================================

func (s *AutomationGRPCServer) EnableAutomation(ctx context.Context, req *automationv1.EnableAutomationRequest) (*automationv1.EnableAutomationResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing tenant_id: %v", err)
	}

	id, err := uuid.Parse(req.GetAutomationId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid automation_id: %v", err)
	}

	if err := s.svc.SetActive(ctx, id, tenantID, true); err != nil {
		return nil, mapDomainError(err)
	}

	auto, err := s.svc.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &automationv1.EnableAutomationResponse{
		Automation: automationToProto(auto),
	}, nil
}

func (s *AutomationGRPCServer) DisableAutomation(ctx context.Context, req *automationv1.DisableAutomationRequest) (*automationv1.DisableAutomationResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing tenant_id: %v", err)
	}

	id, err := uuid.Parse(req.GetAutomationId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid automation_id: %v", err)
	}

	if err := s.svc.SetActive(ctx, id, tenantID, false); err != nil {
		return nil, mapDomainError(err)
	}

	auto, err := s.svc.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &automationv1.DisableAutomationResponse{
		Automation: automationToProto(auto),
	}, nil
}

// ============================================================================
// Execution Logs (2 RPCs)
// ============================================================================

func (s *AutomationGRPCServer) ListExecutions(ctx context.Context, req *automationv1.ListExecutionsRequest) (*automationv1.ListExecutionsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing tenant_id: %v", err)
	}

	filter := workflow.ExecutionFilter{
		TenantID: tenantID,
		Limit:    int(req.GetLimit()),
		Offset:   int(req.GetOffset()),
	}

	if req.GetAutomationId() != "" {
		id, err := uuid.Parse(req.GetAutomationId())
		if err == nil {
			filter.AutomationID = &id
		}
	}
	if req.GetStatus() != automationv1.ExecutionStatus_EXECUTION_STATUS_UNSPECIFIED {
		st := executionStatusFromProto(req.GetStatus())
		if st != "" {
			filter.Status = &st
		}
	}

	execs, total, err := s.svc.ListExecutions(ctx, filter)
	if err != nil {
		return nil, mapDomainError(err)
	}

	protoExecs := make([]*automationv1.AutomationExecutionInfo, len(execs))
	for i, e := range execs {
		protoExecs[i] = executionToProto(e)
	}

	return &automationv1.ListExecutionsResponse{
		Executions: protoExecs,
		TotalCount: int32(total),
	}, nil
}

func (s *AutomationGRPCServer) GetExecution(ctx context.Context, req *automationv1.GetExecutionRequest) (*automationv1.GetExecutionResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing tenant_id: %v", err)
	}

	id, err := uuid.Parse(req.GetExecutionId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid execution_id: %v", err)
	}

	exec, err := s.svc.GetExecution(ctx, id, tenantID)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &automationv1.GetExecutionResponse{
		Execution: executionToProto(exec),
	}, nil
}

// ============================================================================
// Catalog (2 RPCs)
// ============================================================================

func (s *AutomationGRPCServer) ListTriggerDefinitions(_ context.Context, _ *automationv1.ListTriggerDefinitionsRequest) (*automationv1.ListTriggerDefinitionsResponse, error) {
	triggers := s.svc.ListTriggers()

	protoDefs := make([]*automationv1.TriggerDefinition, 0, len(triggers))
	for _, t := range triggers {
		protoDefs = append(protoDefs, triggerDefToProto(t))
	}

	return &automationv1.ListTriggerDefinitionsResponse{
		Triggers: protoDefs,
	}, nil
}

func (s *AutomationGRPCServer) ListActionDefinitions(_ context.Context, _ *automationv1.ListActionDefinitionsRequest) (*automationv1.ListActionDefinitionsResponse, error) {
	actions := s.svc.ListActions()

	protoDefs := make([]*automationv1.ActionDefinition, 0, len(actions))
	for _, a := range actions {
		protoDefs = append(protoDefs, actionDefToProto(a))
	}

	return &automationv1.ListActionDefinitionsResponse{
		Actions: protoDefs,
	}, nil
}

// ============================================================================
// Templates (2 RPCs)
// ============================================================================

func (s *AutomationGRPCServer) ListTemplates(ctx context.Context, req *automationv1.ListTemplatesRequest) (*automationv1.ListTemplatesResponse, error) {
	var category *string
	if req.GetCategory() != "" {
		c := req.GetCategory()
		category = &c
	}

	templates, err := s.svc.ListTemplates(ctx, category)
	if err != nil {
		return nil, mapDomainError(err)
	}

	protoTemplates := make([]*automationv1.AutomationTemplateInfo, len(templates))
	for i, t := range templates {
		protoTemplates[i] = templateToProto(t)
	}

	return &automationv1.ListTemplatesResponse{
		Templates: protoTemplates,
	}, nil
}

func (s *AutomationGRPCServer) CreateFromTemplate(ctx context.Context, req *automationv1.CreateFromTemplateRequest) (*automationv1.CreateFromTemplateResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing tenant_id: %v", err)
	}

	ownerID, err := uuid.Parse(req.GetOwnerId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid owner_id: %v", err)
	}

	auto, err := s.svc.CreateFromTemplate(ctx, req.GetTemplateId(), tenantID, ownerID, req.GetName())
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &automationv1.CreateFromTemplateResponse{
		Automation: automationToProto(auto),
	}, nil
}

// ============================================================================
// Testing (2 RPCs)
// ============================================================================

func (s *AutomationGRPCServer) TestCondition(ctx context.Context, req *automationv1.TestConditionRequest) (*automationv1.TestConditionResponse, error) {
	condJSON := automationStructToJSON(req.GetConditions())
	sampleEnv := automationStructToMap(req.GetSampleEvent())

	result, err := s.svc.TestCondition(ctx, condJSON, sampleEnv)
	if err != nil {
		return &automationv1.TestConditionResponse{
			Matches:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &automationv1.TestConditionResponse{
		Matches: result,
	}, nil
}

func (s *AutomationGRPCServer) DryRunAutomation(ctx context.Context, req *automationv1.DryRunAutomationRequest) (*automationv1.DryRunAutomationResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing tenant_id: %v", err)
	}

	id, err := uuid.Parse(req.GetAutomationId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid automation_id: %v", err)
	}

	sampleEnv := automationStructToMap(req.GetSampleEvent())

	result, err := s.svc.DryRun(ctx, id, tenantID, sampleEnv)
	if err != nil {
		return &automationv1.DryRunAutomationResponse{
			ConditionMatched: false,
			ErrorMessage:     err.Error(),
		}, nil
	}

	steps := make([]*automationv1.ExecutionStepInfo, len(result.Steps))
	for i, step := range result.Steps {
		steps[i] = &automationv1.ExecutionStepInfo{
			ActionType: step.ActionType,
		}
		if step.Error != "" {
			steps[i].Error = step.Error
		}
	}

	return &automationv1.DryRunAutomationResponse{
		ConditionMatched: result.ConditionResult,
		SimulatedSteps:   steps,
	}, nil
}

// ============================================================================
// Stats (1 RPC)
// ============================================================================

func (s *AutomationGRPCServer) GetAutomationStats(ctx context.Context, _ *automationv1.GetAutomationStatsRequest) (*automationv1.GetAutomationStatsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing tenant_id: %v", err)
	}

	stats, err := s.svc.GetStats(ctx, tenantID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get stats: %v", err)
	}

	return &automationv1.GetAutomationStatsResponse{
		TotalAutomations:     int32(stats.TotalAutomations),
		ActiveAutomations:    int32(stats.ActiveAutomations),
		TotalExecutions:      int32(stats.TotalExecutions),
		SuccessfulExecutions: int32(stats.SuccessfulExecs),
		FailedExecutions:     int32(stats.FailedExecs),
		SkippedExecutions:    int32(stats.SkippedExecs),
	}, nil
}

// ============================================================================
// Webhook Trigger (1 RPC)
// ============================================================================

// TriggerWebhook is deliberately the only RPC on this service that does not
// call middleware.GetTenantID: the caller is the gateway's unauthenticated
// public webhook route, which has no tenant to offer. workflow.Service
// resolves the tenant off the automation row itself once the signature has
// been verified (see workflow.Service.TriggerWebhook's doc comment).
func (s *AutomationGRPCServer) TriggerWebhook(ctx context.Context, req *automationv1.TriggerWebhookRequest) (*automationv1.TriggerWebhookResponse, error) {
	automationID, err := uuid.Parse(req.GetAutomationId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid automation_id: %v", err)
	}

	result, err := s.svc.TriggerWebhook(ctx, automationID, req.GetBody(), req.GetSignature(), req.GetIdempotencyKey())
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &automationv1.TriggerWebhookResponse{
		Duplicate: result.Duplicate,
	}, nil
}

// ============================================================================
// Converters
// ============================================================================

func automationToProto(a *models.Automation) *automationv1.AutomationInfo {
	info := &automationv1.AutomationInfo{
		Id:          a.ID.String(),
		Name:        a.Name,
		Description: a.Description,
		Scope:       scopeToProto(a.Scope),
		OwnerId:     a.OwnerID.String(),
		TriggerType: a.TriggerType,
		IsActive:    a.IsActive,
		MaxSteps:    int32(a.MaxSteps),
		CreatedAt:   timestamppb.New(a.CreatedAt),
		UpdatedAt:   timestamppb.New(a.UpdatedAt),
	}

	if a.TriggerConfig != nil {
		info.TriggerConfig = automationJSONToStruct(a.TriggerConfig)
	}
	if a.Conditions != nil {
		info.Conditions = automationJSONToStruct(a.Conditions)
	}
	if a.Actions != nil {
		info.Actions = automationJSONToList(a.Actions)
	}
	if a.TemplateID != nil {
		info.TemplateId = a.TemplateID
	}
	if a.LastTriggeredAt != nil {
		info.LastTriggeredAt = timestamppb.New(*a.LastTriggeredAt)
	}

	return info
}

func executionToProto(e *models.AutomationExecution) *automationv1.AutomationExecutionInfo {
	info := &automationv1.AutomationExecutionInfo{
		Id:              e.ID.String(),
		AutomationId:    e.AutomationID.String(),
		ChainId:         e.ChainID.String(),
		ConditionResult: e.ConditionResult,
		Status:          executionStatusToProto(e.Status),
		StartedAt:       timestamppb.New(e.StartedAt),
	}

	if e.TriggerEvent != nil {
		info.TriggerEvent = automationJSONToStruct(e.TriggerEvent)
	}
	if e.Steps != nil {
		var steps []models.ExecutionStep
		if json.Unmarshal(e.Steps, &steps) == nil {
			for _, step := range steps {
				protoStep := &automationv1.ExecutionStepInfo{
					ActionType: step.ActionType,
					DurationMs: int32(step.DurationMs),
				}
				if step.Input != nil {
					protoStep.Input = automationJSONToStruct(step.Input)
				}
				if step.Output != nil {
					protoStep.Output = automationJSONToStruct(step.Output)
				}
				if step.Error != nil {
					protoStep.Error = *step.Error
				}
				info.Steps = append(info.Steps, protoStep)
			}
		}
	}
	if e.ErrorMessage != nil {
		info.ErrorMessage = *e.ErrorMessage
	}
	if e.CompletedAt != nil {
		info.CompletedAt = timestamppb.New(*e.CompletedAt)
	}
	if e.DurationMs != nil {
		d := int32(*e.DurationMs)
		info.DurationMs = &d
	}

	return info
}

func templateToProto(t *models.AutomationTemplate) *automationv1.AutomationTemplateInfo {
	info := &automationv1.AutomationTemplateInfo{
		Id:          t.ID,
		Name:        t.Name,
		Description: t.Description,
		Category:    t.Category,
		Complexity:  complexityToProto(t.Complexity),
		TriggerType: t.TriggerType,
		CreatedAt:   timestamppb.New(t.CreatedAt),
	}

	if t.TriggerConfig != nil {
		info.TriggerConfig = automationJSONToStruct(t.TriggerConfig)
	}
	if t.Conditions != nil {
		info.Conditions = automationJSONToStruct(t.Conditions)
	}
	if t.Actions != nil {
		info.Actions = automationJSONToList(t.Actions)
	}

	return info
}

// triggerDefToProto converts an opaque trigger definition to proto.
// The service returns []any to avoid import cycles; we type-assert here.
func triggerDefToProto(v any) *automationv1.TriggerDefinition {
	// The value is a map or struct - marshal to JSON and use proto Struct
	data, err := json.Marshal(v)
	if err != nil {
		return &automationv1.TriggerDefinition{}
	}

	var m map[string]any
	if json.Unmarshal(data, &m) != nil {
		return &automationv1.TriggerDefinition{}
	}

	def := &automationv1.TriggerDefinition{}
	if t, ok := m["Type"].(string); ok {
		def.Type = t
	}
	if mod, ok := m["Module"].(string); ok {
		def.Module = mod
	}
	if n, ok := m["Name"].(string); ok {
		def.Name = n
	}
	if d, ok := m["Description"].(string); ok {
		def.Description = d
	}

	return def
}

// actionDefToProto converts an opaque action definition to proto.
func actionDefToProto(v any) *automationv1.ActionDefinition {
	data, err := json.Marshal(v)
	if err != nil {
		return &automationv1.ActionDefinition{}
	}

	var m map[string]any
	if json.Unmarshal(data, &m) != nil {
		return &automationv1.ActionDefinition{}
	}

	def := &automationv1.ActionDefinition{}
	if t, ok := m["Type"].(string); ok {
		def.Type = t
	}
	if mod, ok := m["Module"].(string); ok {
		def.Module = mod
	}
	if n, ok := m["Name"].(string); ok {
		def.Name = n
	}
	if d, ok := m["Description"].(string); ok {
		def.Description = d
	}

	return def
}

// ============================================================================
// Enum converters
// ============================================================================

func scopeFromProto(s automationv1.AutomationScope) string {
	switch s {
	case automationv1.AutomationScope_SCOPE_PERSONAL:
		return models.AutomationScopePersonal
	case automationv1.AutomationScope_SCOPE_TEAM:
		return models.AutomationScopeTeam
	case automationv1.AutomationScope_SCOPE_ORGANIZATION:
		return models.AutomationScopeOrganization
	default:
		return models.AutomationScopePersonal
	}
}

func scopeToProto(s string) automationv1.AutomationScope {
	switch s {
	case models.AutomationScopePersonal:
		return automationv1.AutomationScope_SCOPE_PERSONAL
	case models.AutomationScopeTeam:
		return automationv1.AutomationScope_SCOPE_TEAM
	case models.AutomationScopeOrganization:
		return automationv1.AutomationScope_SCOPE_ORGANIZATION
	default:
		return automationv1.AutomationScope_SCOPE_PERSONAL
	}
}

func executionStatusToProto(s string) automationv1.ExecutionStatus {
	switch s {
	case models.ExecutionStatusRunning:
		return automationv1.ExecutionStatus_EXECUTION_STATUS_RUNNING
	case models.ExecutionStatusCompleted:
		return automationv1.ExecutionStatus_EXECUTION_STATUS_COMPLETED
	case models.ExecutionStatusFailed:
		return automationv1.ExecutionStatus_EXECUTION_STATUS_FAILED
	case models.ExecutionStatusSkipped:
		return automationv1.ExecutionStatus_EXECUTION_STATUS_SKIPPED
	case models.ExecutionStatusAborted:
		return automationv1.ExecutionStatus_EXECUTION_STATUS_ABORTED
	default:
		return automationv1.ExecutionStatus_EXECUTION_STATUS_UNSPECIFIED
	}
}

// executionStatusFromProto converts a proto ExecutionStatus to the domain string.
func executionStatusFromProto(s automationv1.ExecutionStatus) string {
	switch s {
	case automationv1.ExecutionStatus_EXECUTION_STATUS_RUNNING:
		return models.ExecutionStatusRunning
	case automationv1.ExecutionStatus_EXECUTION_STATUS_COMPLETED:
		return models.ExecutionStatusCompleted
	case automationv1.ExecutionStatus_EXECUTION_STATUS_FAILED:
		return models.ExecutionStatusFailed
	case automationv1.ExecutionStatus_EXECUTION_STATUS_SKIPPED:
		return models.ExecutionStatusSkipped
	case automationv1.ExecutionStatus_EXECUTION_STATUS_ABORTED:
		return models.ExecutionStatusAborted
	default:
		return ""
	}
}

func complexityToProto(c string) automationv1.TemplateComplexity {
	switch c {
	case models.ComplexitySimple:
		return automationv1.TemplateComplexity_COMPLEXITY_SIMPLE
	case models.ComplexityMedium:
		return automationv1.TemplateComplexity_COMPLEXITY_MEDIUM
	case models.ComplexityAdvanced:
		return automationv1.TemplateComplexity_COMPLEXITY_ADVANCED
	default:
		return automationv1.TemplateComplexity_COMPLEXITY_UNSPECIFIED
	}
}

// ============================================================================
// JSON / Struct helpers
// ============================================================================

func automationStructToJSON(s *structpb.Struct) json.RawMessage {
	if s == nil {
		return nil
	}
	data, err := s.MarshalJSON()
	if err != nil {
		return nil
	}
	return data
}

func automationJSONToStruct(data json.RawMessage) *structpb.Struct {
	if len(data) == 0 {
		return nil
	}
	s := &structpb.Struct{}
	if err := s.UnmarshalJSON(data); err != nil {
		return nil
	}
	return s
}

// automationListToJSON converts the actions field, which carries a JSON
// array (see models.ActionConfig), unlike trigger_config/conditions which
// carry a JSON object and stay google.protobuf.Struct.
func automationListToJSON(l *structpb.ListValue) json.RawMessage {
	if l == nil {
		return nil
	}
	data, err := l.MarshalJSON()
	if err != nil {
		return nil
	}
	return data
}

func automationJSONToList(data json.RawMessage) *structpb.ListValue {
	if len(data) == 0 {
		return nil
	}
	l := &structpb.ListValue{}
	if err := l.UnmarshalJSON(data); err != nil {
		return nil
	}
	return l
}

func automationStructToMap(s *structpb.Struct) map[string]any {
	if s == nil {
		return map[string]any{}
	}
	return s.AsMap()
}

// ============================================================================
// Error mapping
// ============================================================================

func mapDomainError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, workflow.ErrAutomationNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, workflow.ErrExecutionNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, workflow.ErrTemplateNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, workflow.ErrInvalidCondition):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, workflow.ErrInvalidAction):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, workflow.ErrAutomationInactive):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, workflow.ErrLoopDetected):
		return status.Error(codes.Aborted, err.Error())
	case errors.Is(err, workflow.ErrCircuitBreakerOpen):
		return status.Error(codes.ResourceExhausted, err.Error())
	case errors.Is(err, workflow.ErrWebhookNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, workflow.ErrWebhookSignatureInvalid):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, workflow.ErrWebhookPayloadTooLarge):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, workflow.ErrWebhookIdempotencyConflict):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		slog.Error("unhandled automation service error", "error", err)
		return status.Error(codes.Internal, "internal error")
	}
}
