package server

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/biz/hr/changerequest"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

// ============================================================================
// Profile Change Request RPCs
// ============================================================================

func toProtoChangeRequest(r *models.HRProfileChangeRequest) *hrv1.ProfileChangeRequest {
	out := &hrv1.ProfileChangeRequest{
		Id:            r.ID.String(),
		UserId:        r.UserID.String(),
		UserName:      r.UserName,
		Drawer:        r.Drawer,
		Field:         r.Field,
		FieldLabel:    r.FieldLabel,
		OldValue:      r.OldValue,
		NewValue:      r.NewValue,
		Status:        string(r.Status),
		Reason:        r.Reason,
		CreatedAt:     timestamppb.New(r.CreatedAt),
		DecidedByName: r.DecidedByName,
	}
	if r.DecidedAt != nil {
		out.DecidedAt = timestamppb.New(*r.DecidedAt)
	}
	return out
}

// changeRequestActors resolves the tenant from the context (never from the
// request, which the gateway fills from the same claims) and parses the actor.
func changeRequestActors(ctx context.Context, actorID string) (uuid.UUID, uuid.UUID, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return uuid.Nil, uuid.Nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}
	actor, err := uuid.Parse(actorID)
	if err != nil {
		return uuid.Nil, uuid.Nil, status.Error(codes.InvalidArgument, "invalid actor_id")
	}
	return tenantID, actor, nil
}

func (s *HRGRPCServer) ListProfileChangeRequests(ctx context.Context, req *hrv1.ListProfileChangeRequestsReq) (*hrv1.ListProfileChangeRequestsResp, error) {
	tenantID, actorID, err := changeRequestActors(ctx, req.GetActorId())
	if err != nil {
		return nil, err
	}

	requests, err := s.changeRequestService.List(ctx, changerequest.ListInput{
		TenantID:   tenantID,
		ActorID:    actorID,
		ActorScope: req.GetActorScope(),
		OwnOnly:    req.GetOwnOnly(),
		Status:     req.GetStatus(),
	})
	if err != nil {
		return nil, mapHRError(err)
	}

	out := make([]*hrv1.ProfileChangeRequest, 0, len(requests))
	for _, r := range requests {
		out = append(out, toProtoChangeRequest(r))
	}
	return &hrv1.ListProfileChangeRequestsResp{Requests: out, Total: int32(len(out))}, nil
}

func (s *HRGRPCServer) CreateProfileChangeRequest(ctx context.Context, req *hrv1.CreateProfileChangeRequestReq) (*hrv1.CreateProfileChangeRequestResp, error) {
	tenantID, actorID, err := changeRequestActors(ctx, req.GetActorId())
	if err != nil {
		return nil, err
	}

	created, err := s.changeRequestService.Create(ctx, changerequest.CreateInput{
		TenantID:   tenantID,
		ActorID:    actorID,
		Drawer:     req.GetDrawer(),
		Field:      req.GetField(),
		FieldLabel: req.GetFieldLabel(),
		OldValue:   req.GetOldValue(),
		NewValue:   req.GetNewValue(),
	})
	if err != nil {
		return nil, mapHRError(err)
	}
	return &hrv1.CreateProfileChangeRequestResp{Request: toProtoChangeRequest(created)}, nil
}

func (s *HRGRPCServer) ApproveProfileChangeRequest(ctx context.Context, req *hrv1.ApproveProfileChangeRequestReq) (*hrv1.ApproveProfileChangeRequestResp, error) {
	input, err := s.decideInput(ctx, req.GetActorId(), req.GetRequestId(), req.GetActorScope(), "")
	if err != nil {
		return nil, err
	}

	updated, err := s.changeRequestService.Approve(ctx, input)
	if err != nil {
		return nil, mapHRError(err)
	}
	return &hrv1.ApproveProfileChangeRequestResp{Request: toProtoChangeRequest(updated)}, nil
}

func (s *HRGRPCServer) RejectProfileChangeRequest(ctx context.Context, req *hrv1.RejectProfileChangeRequestReq) (*hrv1.RejectProfileChangeRequestResp, error) {
	input, err := s.decideInput(ctx, req.GetActorId(), req.GetRequestId(), req.GetActorScope(), req.GetReason())
	if err != nil {
		return nil, err
	}

	updated, err := s.changeRequestService.Reject(ctx, input)
	if err != nil {
		return nil, mapHRError(err)
	}
	return &hrv1.RejectProfileChangeRequestResp{Request: toProtoChangeRequest(updated)}, nil
}

func (s *HRGRPCServer) CancelProfileChangeRequest(ctx context.Context, req *hrv1.CancelProfileChangeRequestReq) (*hrv1.CancelProfileChangeRequestResp, error) {
	tenantID, actorID, err := changeRequestActors(ctx, req.GetActorId())
	if err != nil {
		return nil, err
	}
	requestID, err := uuid.Parse(req.GetRequestId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request_id")
	}

	updated, err := s.changeRequestService.Cancel(ctx, tenantID, actorID, requestID)
	if err != nil {
		return nil, mapHRError(err)
	}
	return &hrv1.CancelProfileChangeRequestResp{Request: toProtoChangeRequest(updated)}, nil
}

func (s *HRGRPCServer) decideInput(ctx context.Context, actorID, requestID, actorScope, reason string) (changerequest.DecideInput, error) {
	tenantID, actor, err := changeRequestActors(ctx, actorID)
	if err != nil {
		return changerequest.DecideInput{}, err
	}
	parsedRequestID, err := uuid.Parse(requestID)
	if err != nil {
		return changerequest.DecideInput{}, status.Error(codes.InvalidArgument, "invalid request_id")
	}
	return changerequest.DecideInput{
		TenantID:   tenantID,
		ActorID:    actor,
		ActorScope: actorScope,
		RequestID:  parsedRequestID,
		Reason:     reason,
	}, nil
}
