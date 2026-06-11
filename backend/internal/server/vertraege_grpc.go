package server

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/vertraege"
	vertraegev1 "github.com/kmuhub/kmuhub/proto/vertraege/v1"
)

// VertraegeGRPCServer implements the VertraegeService gRPC server.
type VertraegeGRPCServer struct {
	vertraegev1.UnimplementedVertraegeServiceServer
	svc *vertraege.Service
}

// NewVertraegeGRPCServer creates a new Vertraege gRPC server.
func NewVertraegeGRPCServer(svc *vertraege.Service) *VertraegeGRPCServer {
	return &VertraegeGRPCServer{svc: svc}
}

// ============================================================================
// Contract RPCs
// ============================================================================

func (s *VertraegeGRPCServer) CreateContract(ctx context.Context, req *vertraegev1.CreateContractRequest) (*vertraegev1.ContractResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := vertraege.CreateContractInput{
		TenantID:       tenantID,
		ContractNumber: req.GetContractNumber(),
		Title:          req.GetTitle(),
		ContractType:   vertraege.ContractType(req.GetContractType()),
		Notes:          req.GetNotes(),
	}

	if req.GetStartsOn() != nil {
		input.StartsOn = req.GetStartsOn().AsTime()
	}
	if req.GetEndsOn() != nil {
		t := req.GetEndsOn().AsTime()
		input.EndsOn = &t
	}
	if req.DocumentUrl != nil {
		s := req.GetDocumentUrl()
		input.DocumentURL = &s
	}
	if req.CreatedBy != nil {
		id, parseErr := uuid.Parse(req.GetCreatedBy())
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid created_by: %v", parseErr)
		}
		input.CreatedBy = &id
	}

	c, createErr := s.svc.CreateContract(ctx, input)
	if createErr != nil {
		return nil, mapVertraegeError(createErr)
	}
	return &vertraegev1.ContractResponse{Contract: vertraegeContractToProto(c)}, nil
}

func (s *VertraegeGRPCServer) UpdateContract(ctx context.Context, req *vertraegev1.UpdateContractRequest) (*vertraegev1.ContractResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	contractID, err := uuid.Parse(req.GetContractId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract_id: %v", err)
	}

	input := vertraege.UpdateContractInput{
		TenantID:    tenantID,
		ContractID:  contractID,
		ClearEndsOn: req.GetClearEndsOn(),
	}

	if req.Title != nil {
		input.Title = req.Title
	}
	if req.ContractType != nil {
		ct := vertraege.ContractType(req.GetContractType())
		input.ContractType = &ct
	}
	if req.Status != nil {
		st := vertraege.ContractStatus(req.GetStatus())
		input.Status = &st
	}
	if req.StartsOn != nil {
		t := req.GetStartsOn().AsTime()
		input.StartsOn = &t
	}
	if req.EndsOn != nil && !req.GetClearEndsOn() {
		t := req.GetEndsOn().AsTime()
		input.EndsOn = &t
	}
	if req.DocumentUrl != nil {
		u := req.GetDocumentUrl()
		input.DocumentURL = &u
	}
	if req.Notes != nil {
		n := req.GetNotes()
		input.Notes = &n
	}

	c, updateErr := s.svc.UpdateContract(ctx, input)
	if updateErr != nil {
		return nil, mapVertraegeError(updateErr)
	}
	return &vertraegev1.ContractResponse{Contract: vertraegeContractToProto(c)}, nil
}

func (s *VertraegeGRPCServer) DeleteContract(ctx context.Context, req *vertraegev1.DeleteContractRequest) (*vertraegev1.DeleteContractResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	contractID, err := uuid.Parse(req.GetContractId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract_id: %v", err)
	}

	if delErr := s.svc.DeleteContract(ctx, tenantID, contractID); delErr != nil {
		return nil, mapVertraegeError(delErr)
	}
	return &vertraegev1.DeleteContractResponse{}, nil
}

func (s *VertraegeGRPCServer) GetContract(ctx context.Context, req *vertraegev1.GetContractRequest) (*vertraegev1.ContractResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	contractID, err := uuid.Parse(req.GetContractId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract_id: %v", err)
	}

	c, getErr := s.svc.GetContract(ctx, tenantID, contractID)
	if getErr != nil {
		return nil, mapVertraegeError(getErr)
	}
	return &vertraegev1.ContractResponse{Contract: vertraegeContractToProto(c)}, nil
}

func (s *VertraegeGRPCServer) ListContracts(ctx context.Context, req *vertraegev1.ListContractsRequest) (*vertraegev1.ListContractsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := vertraege.ListContractsInput{
		TenantID: tenantID,
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	}
	if req.Status != nil {
		st := vertraege.ContractStatus(req.GetStatus())
		input.Status = &st
	}
	if req.ContractType != nil {
		ct := vertraege.ContractType(req.GetContractType())
		input.Type = &ct
	}
	if req.StartsAfter != nil {
		t := req.GetStartsAfter().AsTime()
		input.StartsAfter = &t
	}
	if req.StartsBefore != nil {
		t := req.GetStartsBefore().AsTime()
		input.StartsBefore = &t
	}
	if req.EndsAfter != nil {
		t := req.GetEndsAfter().AsTime()
		input.EndsAfter = &t
	}
	if req.EndsBefore != nil {
		t := req.GetEndsBefore().AsTime()
		input.EndsBefore = &t
	}
	if req.ContactId != nil {
		cid, parseErr := uuid.Parse(req.GetContactId())
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid contact_id: %v", parseErr)
		}
		input.ContactID = &cid
	}

	contracts, total, listErr := s.svc.ListContracts(ctx, input)
	if listErr != nil {
		return nil, mapVertraegeError(listErr)
	}

	protoContracts := make([]*vertraegev1.Contract, len(contracts))
	for i, c := range contracts {
		protoContracts[i] = vertraegeContractToProto(c)
	}
	return &vertraegev1.ListContractsResponse{
		Contracts: protoContracts,
		Total:     int32(total),
	}, nil
}

// UploadDocument is deprecated. Use POST /api/v1/files/presign-upload with
// scope=vertraege instead — the client-side presign flow handles this entirely.
func (s *VertraegeGRPCServer) UploadDocument(_ context.Context, _ *vertraegev1.UploadDocumentRequest) (*vertraegev1.UploadDocumentResponse, error) {
	return nil, status.Error(codes.Unimplemented, "use POST /api/v1/files/presign-upload with scope=vertraege instead")
}

func (s *VertraegeGRPCServer) ExportContract(ctx context.Context, req *vertraegev1.ExportContractRequest) (*vertraegev1.ExportContractResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	contractID, err := uuid.Parse(req.GetContractId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract_id: %v", err)
	}

	data, exportErr := s.svc.ExportContract(ctx, tenantID, contractID)
	if exportErr != nil {
		return nil, mapVertraegeError(exportErr)
	}
	return &vertraegev1.ExportContractResponse{
		Payload:     data,
		ContentType: "text/plain; charset=utf-8",
		Filename:    "contract.txt",
	}, nil
}

// ============================================================================
// Signature RPC
// ============================================================================

func (s *VertraegeGRPCServer) SaveSignature(ctx context.Context, req *vertraegev1.SaveContractSignatureRequest) (*vertraegev1.ContractResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	contractID, err := uuid.Parse(req.GetContractId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract_id: %v", err)
	}

	contract, err := s.svc.SaveSignature(ctx, tenantID.String(), contractID.String(), req.GetSignatureData(), req.GetSignedBy())
	if err != nil {
		return nil, mapVertraegeError(err)
	}
	return &vertraegev1.ContractResponse{Contract: vertraegeContractToProto(contract)}, nil
}

// ============================================================================
// Party RPCs
// ============================================================================

func (s *VertraegeGRPCServer) AddParty(ctx context.Context, req *vertraegev1.AddPartyRequest) (*vertraegev1.PartyResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	contractID, err := uuid.Parse(req.GetContractId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract_id: %v", err)
	}

	input := vertraege.AddPartyInput{
		TenantID:       tenantID,
		ContractID:     contractID,
		PartyType:      vertraege.PartyType(req.GetPartyType()),
		RoleInContract: req.GetRoleInContract(),
		ExternalName:   req.ExternalName,
	}

	if req.ContactId != nil {
		id, parseErr := uuid.Parse(req.GetContactId())
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid contact_id: %v", parseErr)
		}
		input.ContactID = &id
	}
	if req.CompanyId != nil {
		id, parseErr := uuid.Parse(req.GetCompanyId())
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid company_id: %v", parseErr)
		}
		input.CompanyID = &id
	}
	if req.SignedOn != nil {
		t := req.GetSignedOn().AsTime()
		input.SignedOn = &t
	}

	p, addErr := s.svc.AddParty(ctx, input)
	if addErr != nil {
		return nil, mapVertraegeError(addErr)
	}
	return &vertraegev1.PartyResponse{Party: vertraegePartyToProto(p)}, nil
}

func (s *VertraegeGRPCServer) RemoveParty(ctx context.Context, req *vertraegev1.RemovePartyRequest) (*vertraegev1.RemovePartyResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	partyID, err := uuid.Parse(req.GetPartyId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid party_id: %v", err)
	}

	if removeErr := s.svc.RemoveParty(ctx, tenantID, partyID); removeErr != nil {
		return nil, mapVertraegeError(removeErr)
	}
	return &vertraegev1.RemovePartyResponse{}, nil
}

func (s *VertraegeGRPCServer) ListParties(ctx context.Context, req *vertraegev1.ListPartiesRequest) (*vertraegev1.ListPartiesResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	contractID, err := uuid.Parse(req.GetContractId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract_id: %v", err)
	}

	parties, listErr := s.svc.ListParties(ctx, tenantID, contractID)
	if listErr != nil {
		return nil, mapVertraegeError(listErr)
	}

	protoParties := make([]*vertraegev1.ContractParty, len(parties))
	for i, p := range parties {
		protoParties[i] = vertraegePartyToProto(p)
	}
	return &vertraegev1.ListPartiesResponse{Parties: protoParties}, nil
}

// ============================================================================
// Reminder RPCs
// ============================================================================

func (s *VertraegeGRPCServer) CreateReminder(ctx context.Context, req *vertraegev1.CreateReminderRequest) (*vertraegev1.ReminderResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	contractID, err := uuid.Parse(req.GetContractId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract_id: %v", err)
	}

	var remindAt time.Time
	if req.GetRemindAt() != nil {
		remindAt = req.GetRemindAt().AsTime()
	}

	rem, createErr := s.svc.CreateReminder(ctx, vertraege.CreateReminderInput{
		TenantID:     tenantID,
		ContractID:   contractID,
		RemindAt:     remindAt,
		ReminderType: vertraege.ReminderType(req.GetReminderType()),
		Subject:      req.GetSubject(),
		Message:      req.GetMessage(),
	})
	if createErr != nil {
		return nil, mapVertraegeError(createErr)
	}
	return &vertraegev1.ReminderResponse{Reminder: vertraegeReminderToProto(rem)}, nil
}

func (s *VertraegeGRPCServer) UpdateReminder(ctx context.Context, req *vertraegev1.UpdateReminderRequest) (*vertraegev1.ReminderResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	reminderID, err := uuid.Parse(req.GetReminderId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid reminder_id: %v", err)
	}

	input := vertraege.UpdateReminderInput{
		TenantID:   tenantID,
		ReminderID: reminderID,
	}
	if req.RemindAt != nil {
		t := req.GetRemindAt().AsTime()
		input.RemindAt = &t
	}
	if req.ReminderType != nil {
		rt := vertraege.ReminderType(req.GetReminderType())
		input.ReminderType = &rt
	}
	if req.Subject != nil {
		input.Subject = req.Subject
	}
	if req.Message != nil {
		input.Message = req.Message
	}
	if req.Status != nil {
		st := vertraege.ReminderStatus(req.GetStatus())
		input.Status = &st
	}

	rem, updateErr := s.svc.UpdateReminder(ctx, input)
	if updateErr != nil {
		return nil, mapVertraegeError(updateErr)
	}
	return &vertraegev1.ReminderResponse{Reminder: vertraegeReminderToProto(rem)}, nil
}

func (s *VertraegeGRPCServer) DeleteReminder(ctx context.Context, req *vertraegev1.DeleteReminderRequest) (*vertraegev1.DeleteReminderResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	reminderID, err := uuid.Parse(req.GetReminderId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid reminder_id: %v", err)
	}

	if delErr := s.svc.DeleteReminder(ctx, tenantID, reminderID); delErr != nil {
		return nil, mapVertraegeError(delErr)
	}
	return &vertraegev1.DeleteReminderResponse{}, nil
}

func (s *VertraegeGRPCServer) ListReminders(ctx context.Context, req *vertraegev1.ListRemindersRequest) (*vertraegev1.ListRemindersResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	contractID, err := uuid.Parse(req.GetContractId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract_id: %v", err)
	}

	reminders, listErr := s.svc.ListReminders(ctx, tenantID, contractID, req.GetOnlyPending())
	if listErr != nil {
		return nil, mapVertraegeError(listErr)
	}

	protoReminders := make([]*vertraegev1.ContractReminder, len(reminders))
	for i, r := range reminders {
		protoReminders[i] = vertraegeReminderToProto(r)
	}
	return &vertraegev1.ListRemindersResponse{Reminders: protoReminders}, nil
}

// ============================================================================
// Conversion helpers
// ============================================================================

func vertraegeContractToProto(c *vertraege.Contract) *vertraegev1.Contract {
	if c == nil {
		return nil
	}
	proto := &vertraegev1.Contract{
		Id:             c.ID.String(),
		TenantId:       c.TenantID.String(),
		ContractNumber: c.ContractNumber,
		Title:          c.Title,
		ContractType:   string(c.ContractType),
		Status:         string(c.Status),
		StartsOn:       timestamppb.New(c.StartsOn),
		Notes:          c.Notes,
		CreatedAt:      timestamppb.New(c.CreatedAt),
		UpdatedAt:      timestamppb.New(c.UpdatedAt),
	}
	if c.EndsOn != nil {
		proto.EndsOn = timestamppb.New(*c.EndsOn)
	}
	if c.DocumentURL != nil {
		proto.DocumentUrl = c.DocumentURL
	}
	if c.CreatedBy != nil {
		s := c.CreatedBy.String()
		proto.CreatedBy = &s
	}
	if c.SignatureProvider != nil {
		proto.SignatureProvider = c.SignatureProvider
	}

	for _, p := range c.Parties {
		proto.Parties = append(proto.Parties, vertraegePartyToProto(p))
	}
	for _, r := range c.Reminders {
		proto.Reminders = append(proto.Reminders, vertraegeReminderToProto(r))
	}
	return proto
}

func vertraegePartyToProto(p *vertraege.ContractParty) *vertraegev1.ContractParty {
	if p == nil {
		return nil
	}
	proto := &vertraegev1.ContractParty{
		Id:             p.ID.String(),
		TenantId:       p.TenantID.String(),
		ContractId:     p.ContractID.String(),
		PartyType:      string(p.PartyType),
		RoleInContract: p.RoleInContract,
		CreatedAt:      timestamppb.New(p.CreatedAt),
	}
	if p.ContactID != nil {
		s := p.ContactID.String()
		proto.ContactId = &s
	}
	if p.CompanyID != nil {
		s := p.CompanyID.String()
		proto.CompanyId = &s
	}
	if p.ExternalName != nil {
		proto.ExternalName = p.ExternalName
	}
	if p.SignedOn != nil {
		proto.SignedOn = timestamppb.New(*p.SignedOn)
	}
	return proto
}

func vertraegeReminderToProto(r *vertraege.ContractReminder) *vertraegev1.ContractReminder {
	if r == nil {
		return nil
	}
	proto := &vertraegev1.ContractReminder{
		Id:           r.ID.String(),
		TenantId:     r.TenantID.String(),
		ContractId:   r.ContractID.String(),
		RemindAt:     timestamppb.New(r.RemindAt),
		ReminderType: string(r.ReminderType),
		Subject:      r.Subject,
		Message:      r.Message,
		Status:       string(r.Status),
		CreatedAt:    timestamppb.New(r.CreatedAt),
	}
	if r.SentAt != nil {
		proto.SentAt = timestamppb.New(*r.SentAt)
	}
	return proto
}

// ============================================================================
// Error mapping
// ============================================================================

func mapVertraegeError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, vertraege.ErrContractNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, vertraege.ErrPartyNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, vertraege.ErrReminderNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, vertraege.ErrContractNumberTaken):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, vertraege.ErrDeleteNonDraft):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, vertraege.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		slog.Error("unhandled vertraege service error", "error", err)
		return status.Error(codes.Internal, "internal error")
	}
}
