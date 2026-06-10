package server

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/settings"
	settingsv1 "github.com/kmuhub/kmuhub/proto/settings/v1"
)

// SettingsGRPCServer implements settingsv1.SettingsServiceServer.
// It runs in the same binary as the auth service (port :50051) following the
// same co-location pattern as SecurityGRPCServer.
type SettingsGRPCServer struct {
	settingsv1.UnimplementedSettingsServiceServer
	svc *settings.Service
}

// NewSettingsGRPCServer creates a new Settings gRPC server.
func NewSettingsGRPCServer(svc *settings.Service) *SettingsGRPCServer {
	return &SettingsGRPCServer{svc: svc}
}

// ============================================================================
// Module-lead RPCs
// ============================================================================

func (s *SettingsGRPCServer) ListModuleLeads(ctx context.Context, req *settingsv1.ListModuleLeadsRequest) (*settingsv1.ListModuleLeadsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	var userIDPtr *uuid.UUID
	if req.UserId != nil {
		uid, err := uuid.Parse(req.GetUserId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
		}
		userIDPtr = &uid
	}

	leads, err := s.svc.ListModuleLeads(ctx, tenantID, userIDPtr, req.ModuleId)
	if err != nil {
		slog.Error("ListModuleLeads failed", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to list module leads")
	}

	resp := &settingsv1.ListModuleLeadsResponse{
		Leads: make([]*settingsv1.ModuleLead, 0, len(leads)),
	}
	for _, l := range leads {
		resp.Leads = append(resp.Leads, moduleleadToProto(l))
	}
	return resp, nil
}

func (s *SettingsGRPCServer) GrantModuleLead(ctx context.Context, req *settingsv1.GrantModuleLeadRequest) (*settingsv1.ModuleLead, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	targetUserID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}
	callerID, err := uuid.Parse(req.GetGrantedBy())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid granted_by: %v", err)
	}

	ml, err := s.svc.GrantModuleLead(ctx, tenantID, callerID, targetUserID, req.GetModuleId())
	if err != nil {
		if errors.Is(err, settings.ErrNotModuleLead) {
			return nil, status.Errorf(codes.PermissionDenied, "only admins may grant module-lead rights")
		}
		if errors.Is(err, settings.ErrInvalidModuleID) {
			return nil, status.Errorf(codes.InvalidArgument, "module_id must not be empty")
		}
		slog.Error("GrantModuleLead failed", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to grant module lead")
	}
	return moduleleadToProto(ml), nil
}

func (s *SettingsGRPCServer) RevokeModuleLead(ctx context.Context, req *settingsv1.RevokeModuleLeadRequest) (*settingsv1.RevokeModuleLeadResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	targetUserID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	// Caller is extracted from the gRPC context (set by TenantInboundUnaryInterceptor).
	callerIDStr := middleware.GetUserID(ctx)
	callerID, err := uuid.Parse(callerIDStr)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "caller not authenticated")
	}

	if err := s.svc.RevokeModuleLead(ctx, tenantID, callerID, targetUserID, req.GetModuleId()); err != nil {
		if errors.Is(err, settings.ErrNotModuleLead) {
			return nil, status.Errorf(codes.PermissionDenied, "only admins may revoke module-lead rights")
		}
		if errors.Is(err, settings.ErrInvalidModuleID) {
			return nil, status.Errorf(codes.InvalidArgument, "module_id must not be empty")
		}
		slog.Error("RevokeModuleLead failed", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to revoke module lead")
	}
	return &settingsv1.RevokeModuleLeadResponse{}, nil
}

func (s *SettingsGRPCServer) GetMyModuleLeads(ctx context.Context, req *settingsv1.GetMyModuleLeadsRequest) (*settingsv1.GetMyModuleLeadsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	moduleIDs, isAdmin, err := s.svc.GetMyModuleLeads(ctx, tenantID, userID)
	if err != nil {
		slog.Error("GetMyModuleLeads failed", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get module leads")
	}

	if moduleIDs == nil {
		moduleIDs = []string{}
	}
	return &settingsv1.GetMyModuleLeadsResponse{
		ModuleIds: moduleIDs,
		IsAdmin:   isAdmin,
	}, nil
}

// ============================================================================
// Settings RPCs
// ============================================================================

func (s *SettingsGRPCServer) GetResolvedSettings(ctx context.Context, req *settingsv1.GetResolvedSettingsRequest) (*settingsv1.GetResolvedSettingsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	entries, err := s.svc.GetResolvedSettings(ctx, tenantID, userID, req.GetModuleId())
	if err != nil {
		if errors.Is(err, settings.ErrInvalidModuleID) {
			return nil, status.Errorf(codes.InvalidArgument, "module_id must not be empty")
		}
		slog.Error("GetResolvedSettings failed", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get resolved settings")
	}

	return &settingsv1.GetResolvedSettingsResponse{
		Entries: entriesToProto(entries),
	}, nil
}

func (s *SettingsGRPCServer) GetTenantSettings(ctx context.Context, req *settingsv1.GetTenantSettingsRequest) (*settingsv1.GetTenantSettingsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	entries, err := s.svc.GetTenantSettings(ctx, tenantID, req.GetModuleId())
	if err != nil {
		if errors.Is(err, settings.ErrInvalidModuleID) {
			return nil, status.Errorf(codes.InvalidArgument, "module_id must not be empty")
		}
		slog.Error("GetTenantSettings failed", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get tenant settings")
	}

	return &settingsv1.GetTenantSettingsResponse{
		Entries: entriesToProto(entries),
	}, nil
}

func (s *SettingsGRPCServer) PutTenantSettings(ctx context.Context, req *settingsv1.PutTenantSettingsRequest) (*settingsv1.PutTenantSettingsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	callerID, err := uuid.Parse(req.GetUpdatedBy())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid updated_by: %v", err)
	}

	domainEntries, err := protoToEntries(req.GetEntries())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid setting entry: %v", err)
	}

	updated, err := s.svc.PutTenantSettings(ctx, tenantID, callerID, req.GetModuleId(), domainEntries)
	if err != nil {
		if errors.Is(err, settings.ErrNotModuleLead) {
			return nil, status.Errorf(codes.PermissionDenied, "only module-leads or admins may update tenant settings")
		}
		if errors.Is(err, settings.ErrInvalidModuleID) {
			return nil, status.Errorf(codes.InvalidArgument, "module_id must not be empty")
		}
		if errors.Is(err, settings.ErrInvalidKey) {
			return nil, status.Errorf(codes.InvalidArgument, "setting key must not be empty")
		}
		slog.Error("PutTenantSettings failed", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to update tenant settings")
	}

	return &settingsv1.PutTenantSettingsResponse{
		Entries: entriesToProto(updated),
	}, nil
}

func (s *SettingsGRPCServer) GetUserSettings(ctx context.Context, req *settingsv1.GetUserSettingsRequest) (*settingsv1.GetUserSettingsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	entries, err := s.svc.GetUserSettings(ctx, tenantID, userID, req.GetModuleId())
	if err != nil {
		if errors.Is(err, settings.ErrInvalidModuleID) {
			return nil, status.Errorf(codes.InvalidArgument, "module_id must not be empty")
		}
		slog.Error("GetUserSettings failed", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get user settings")
	}

	return &settingsv1.GetUserSettingsResponse{
		Entries: entriesToProto(entries),
	}, nil
}

func (s *SettingsGRPCServer) PutUserSettings(ctx context.Context, req *settingsv1.PutUserSettingsRequest) (*settingsv1.PutUserSettingsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	domainEntries, err := protoToEntries(req.GetEntries())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid setting entry: %v", err)
	}

	updated, err := s.svc.PutUserSettings(ctx, tenantID, userID, req.GetModuleId(), domainEntries)
	if err != nil {
		if errors.Is(err, settings.ErrInvalidModuleID) {
			return nil, status.Errorf(codes.InvalidArgument, "module_id must not be empty")
		}
		if errors.Is(err, settings.ErrInvalidKey) {
			return nil, status.Errorf(codes.InvalidArgument, "setting key must not be empty")
		}
		slog.Error("PutUserSettings failed", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to update user settings")
	}

	return &settingsv1.PutUserSettingsResponse{
		Entries: entriesToProto(updated),
	}, nil
}

// ============================================================================
// Conversion helpers
// ============================================================================

func moduleleadToProto(ml *settings.ModuleLead) *settingsv1.ModuleLead {
	p := &settingsv1.ModuleLead{
		TenantId:  ml.TenantID.String(),
		UserId:    ml.UserID.String(),
		ModuleId:  ml.ModuleID,
		GrantedAt: timestamppb.New(ml.GrantedAt),
	}
	if ml.GrantedBy != nil {
		p.GrantedBy = ml.GrantedBy.String()
	}
	return p
}

func entriesToProto(entries []*settings.SettingEntry) []*settingsv1.SettingEntry {
	out := make([]*settingsv1.SettingEntry, 0, len(entries))
	for _, e := range entries {
		protoVal := &structpb.Value{}
		if err := protoVal.UnmarshalJSON(e.Value); err != nil {
			// Fallback: wrap as string if JSON unmarshal fails (should never happen
			// for well-formed JSONB from Postgres).
			protoVal = structpb.NewStringValue(string(e.Value))
		}
		out = append(out, &settingsv1.SettingEntry{
			Key:   e.Key,
			Value: protoVal,
		})
	}
	return out
}

func protoToEntries(protoEntries []*settingsv1.SettingEntry) ([]*settings.SettingEntry, error) {
	out := make([]*settings.SettingEntry, 0, len(protoEntries))
	for _, pe := range protoEntries {
		raw, err := pe.GetValue().MarshalJSON()
		if err != nil {
			return nil, err
		}
		out = append(out, &settings.SettingEntry{
			Key:   pe.GetKey(),
			Value: raw,
		})
	}
	return out, nil
}
