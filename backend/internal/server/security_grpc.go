package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/security/audit"
	"github.com/kmuhub/kmuhub/internal/security/gdpr"
	"github.com/kmuhub/kmuhub/internal/security/password"
	"github.com/kmuhub/kmuhub/internal/security/vault"
	"github.com/kmuhub/kmuhub/internal/security/vendoraccess"
	securityv1 "github.com/kmuhub/kmuhub/proto/security/v1"
)

// SecurityGRPCServer implements securityv1.SecurityServiceServer.
type SecurityGRPCServer struct {
	securityv1.UnimplementedSecurityServiceServer
	auditService        *audit.Service
	vaultService        *vault.Service
	gdprService         *gdpr.Service
	passwordService     *password.Service
	vendorAccessService *vendoraccess.Service
	pool                *pgxpool.Pool // for direct IP rule queries
}

// NewSecurityGRPCServer creates a new SecurityGRPCServer.
func NewSecurityGRPCServer(
	auditSvc *audit.Service,
	vaultSvc *vault.Service,
	gdprSvc *gdpr.Service,
	passwordSvc *password.Service,
	vendorAccessSvc *vendoraccess.Service,
	pool *pgxpool.Pool,
) *SecurityGRPCServer {
	return &SecurityGRPCServer{
		auditService:        auditSvc,
		vaultService:        vaultSvc,
		gdprService:         gdprSvc,
		passwordService:     passwordSvc,
		vendorAccessService: vendorAccessSvc,
		pool:                pool,
	}
}

// ============================================================================
// Audit RPCs
// ============================================================================

func (s *SecurityGRPCServer) CreateAuditEntry(ctx context.Context, req *securityv1.CreateAuditEntryRequest) (*securityv1.CreateAuditEntryResponse, error) {
	// tenantID defaults to sentinel if not present (system events without tenant context)
	tenantID, _ := middleware.GetTenantID(ctx)

	var userID *uuid.UUID
	if req.UserId != "" {
		parsed, err := uuid.Parse(req.UserId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid user_id")
		}
		userID = &parsed
	}

	s.auditService.LogEvent(ctx, tenantID, userID, req.Action, req.Target, req.TargetType,
		nil, req.IpAddress, req.UserAgent, req.Result)

	return &securityv1.CreateAuditEntryResponse{
		Entry: &securityv1.AuditEntry{
			UserId:    req.UserId,
			Action:    req.Action,
			Target:    req.Target,
			Result:    req.Result,
			Timestamp: timestamppb.Now(),
		},
	}, nil
}

func (s *SecurityGRPCServer) ListAuditEntries(ctx context.Context, req *securityv1.ListAuditEntriesRequest) (*securityv1.ListAuditEntriesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing tenant_id: %v", err)
	}

	filter := &models.AuditFilter{TenantID: tenantID}
	if req.Filter != nil {
		if req.Filter.DateFrom != nil {
			t := req.Filter.DateFrom.AsTime()
			filter.DateFrom = &t
		}
		if req.Filter.DateTo != nil {
			t := req.Filter.DateTo.AsTime()
			filter.DateTo = &t
		}
		if req.Filter.UserId != "" {
			parsed, err := uuid.Parse(req.Filter.UserId)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid user_id in filter")
			}
			filter.UserID = &parsed
		}
		filter.Action = req.Filter.Action
		filter.Result = req.Filter.Result
		filter.Offset = int(req.Filter.Offset)
		filter.Limit = int(req.Filter.Limit)
	}

	entries, total, err := s.auditService.ListEntries(ctx, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list audit entries")
	}

	pbEntries := make([]*securityv1.AuditEntry, 0, len(entries))
	for _, e := range entries {
		pbEntries = append(pbEntries, toProtoAuditEntry(e))
	}

	return &securityv1.ListAuditEntriesResponse{
		Entries: pbEntries,
		Total:   int32(total),
	}, nil
}

func (s *SecurityGRPCServer) ExportAuditLog(ctx context.Context, req *securityv1.ExportAuditLogRequest) (*securityv1.ExportAuditLogResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing tenant_id: %v", err)
	}

	filter := &models.AuditFilter{TenantID: tenantID}
	if req.Filter != nil {
		if req.Filter.DateFrom != nil {
			t := req.Filter.DateFrom.AsTime()
			filter.DateFrom = &t
		}
		if req.Filter.DateTo != nil {
			t := req.Filter.DateTo.AsTime()
			filter.DateTo = &t
		}
		if req.Filter.UserId != "" {
			parsed, err := uuid.Parse(req.Filter.UserId)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid user_id in filter")
			}
			filter.UserID = &parsed
		}
		filter.Action = req.Filter.Action
		filter.Result = req.Filter.Result
	}

	format := req.Format
	if format == "" {
		format = "csv"
	}

	data, err := s.auditService.ExportEntries(ctx, filter, format)
	if err != nil {
		return nil, mapSecurityError(err)
	}

	contentType := "text/csv"
	filename := "audit_log.csv"
	if format == "json" {
		contentType = "application/json"
		filename = "audit_log.json"
	}

	return &securityv1.ExportAuditLogResponse{
		Data:        data,
		ContentType: contentType,
		Filename:    filename,
	}, nil
}

func (s *SecurityGRPCServer) VerifyAuditChain(ctx context.Context, req *securityv1.VerifyAuditChainRequest) (*securityv1.VerifyAuditChainResponse, error) {
	fromSeq := int64(1)
	toSeq := int64(1000000)

	valid, firstInvalid, err := s.auditService.VerifyChainIntegrity(ctx, fromSeq, toSeq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "chain verification failed: %v", err)
	}

	resp := &securityv1.VerifyAuditChainResponse{
		Valid:                valid,
		FirstInvalidSequence: firstInvalid,
	}
	if !valid {
		resp.ErrorMessage = "hash chain integrity violation detected"
	}

	return resp, nil
}

// ============================================================================
// Vault RPCs
// ============================================================================

func (s *SecurityGRPCServer) GetVaultSecret(ctx context.Context, req *securityv1.GetVaultSecretRequest) (*securityv1.GetVaultSecretResponse, error) {
	if req.KeyName == "" {
		return nil, status.Error(codes.InvalidArgument, "key_name is required")
	}

	decryptedValue, err := s.vaultService.GetSecret(ctx, req.KeyName)
	if err != nil {
		return nil, mapSecurityError(err)
	}

	// Get metadata from list (or fetch the secret record)
	secrets, err := s.vaultService.ListSecrets(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get secret metadata")
	}

	var pbSecret *securityv1.VaultSecret
	for _, sec := range secrets {
		if sec.KeyName == req.KeyName {
			pbSecret = toProtoVaultSecret(sec)
			break
		}
	}

	return &securityv1.GetVaultSecretResponse{
		Secret:         pbSecret,
		DecryptedValue: decryptedValue,
	}, nil
}

func (s *SecurityGRPCServer) SetVaultSecret(ctx context.Context, req *securityv1.SetVaultSecretRequest) (*securityv1.SetVaultSecretResponse, error) {
	if req.KeyName == "" {
		return nil, status.Error(codes.InvalidArgument, "key_name is required")
	}
	if req.PlaintextValue == "" {
		return nil, status.Error(codes.InvalidArgument, "plaintext_value is required")
	}

	var createdBy uuid.UUID
	if req.CreatedBy != "" {
		parsed, err := uuid.Parse(req.CreatedBy)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid created_by")
		}
		createdBy = parsed
	}

	if err := s.vaultService.SetSecret(ctx, req.KeyName, req.PlaintextValue, req.Description, createdBy); err != nil {
		return nil, mapSecurityError(err)
	}

	// Return metadata
	secrets, _ := s.vaultService.ListSecrets(ctx)
	var pbSecret *securityv1.VaultSecret
	for _, sec := range secrets {
		if sec.KeyName == req.KeyName {
			pbSecret = toProtoVaultSecret(sec)
			break
		}
	}

	return &securityv1.SetVaultSecretResponse{
		Secret: pbSecret,
	}, nil
}

func (s *SecurityGRPCServer) ListVaultSecrets(ctx context.Context, _ *securityv1.ListVaultSecretsRequest) (*securityv1.ListVaultSecretsResponse, error) {
	secrets, err := s.vaultService.ListSecrets(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list secrets")
	}

	pbSecrets := make([]*securityv1.VaultSecret, 0, len(secrets))
	for _, sec := range secrets {
		pbSecrets = append(pbSecrets, toProtoVaultSecret(sec))
	}

	return &securityv1.ListVaultSecretsResponse{
		Secrets: pbSecrets,
	}, nil
}

func (s *SecurityGRPCServer) DeleteVaultSecret(ctx context.Context, req *securityv1.DeleteVaultSecretRequest) (*securityv1.DeleteVaultSecretResponse, error) {
	if req.KeyName == "" {
		return nil, status.Error(codes.InvalidArgument, "key_name is required")
	}

	// Find the secret by key name to get its ID
	secrets, err := s.vaultService.ListSecrets(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list secrets")
	}

	var secretID uuid.UUID
	found := false
	for _, sec := range secrets {
		if sec.KeyName == req.KeyName {
			secretID = sec.ID
			found = true
			break
		}
	}
	if !found {
		return nil, status.Error(codes.NotFound, "secret not found")
	}

	if err := s.vaultService.DeleteSecret(ctx, secretID); err != nil {
		return nil, mapSecurityError(err)
	}

	return &securityv1.DeleteVaultSecretResponse{}, nil
}

// ============================================================================
// GDPR RPCs
// ============================================================================

func (s *SecurityGRPCServer) RequestDataExport(ctx context.Context, req *securityv1.RequestDataExportRequest) (*securityv1.RequestDataExportResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	export, err := s.gdprService.RequestExport(ctx, userID)
	if err != nil {
		return nil, mapSecurityError(err)
	}

	return &securityv1.RequestDataExportResponse{
		ExportRequest: toProtoGDPRExport(export),
	}, nil
}

func (s *SecurityGRPCServer) ListDataExports(ctx context.Context, req *securityv1.ListDataExportsRequest) (*securityv1.ListDataExportsResponse, error) {
	var userID uuid.UUID
	if req.UserId != "" {
		parsed, err := uuid.Parse(req.UserId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid user_id")
		}
		userID = parsed
	}

	exports, err := s.gdprService.ListExports(ctx, userID, req.Status)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list exports")
	}

	pbExports := make([]*securityv1.GDPRExportRequest, 0, len(exports))
	for _, e := range exports {
		pbExports = append(pbExports, toProtoGDPRExport(e))
	}

	return &securityv1.ListDataExportsResponse{
		ExportRequests: pbExports,
	}, nil
}

func (s *SecurityGRPCServer) ApproveDataExport(ctx context.Context, req *securityv1.ApproveDataExportRequest) (*securityv1.ApproveDataExportResponse, error) {
	exportID, err := uuid.Parse(req.ExportId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid export_id")
	}

	reviewerID, err := uuid.Parse(req.ReviewedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid reviewed_by")
	}

	export, err := s.gdprService.ApproveExport(ctx, exportID, reviewerID, req.ReviewNote)
	if err != nil {
		return nil, mapSecurityError(err)
	}

	return &securityv1.ApproveDataExportResponse{
		ExportRequest: toProtoGDPRExport(export),
	}, nil
}

func (s *SecurityGRPCServer) DenyDataExport(ctx context.Context, req *securityv1.DenyDataExportRequest) (*securityv1.DenyDataExportResponse, error) {
	exportID, err := uuid.Parse(req.ExportId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid export_id")
	}

	reviewerID, err := uuid.Parse(req.ReviewedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid reviewed_by")
	}

	export, err := s.gdprService.DenyExport(ctx, exportID, reviewerID, req.ReviewNote)
	if err != nil {
		return nil, mapSecurityError(err)
	}

	return &securityv1.DenyDataExportResponse{
		ExportRequest: toProtoGDPRExport(export),
	}, nil
}

// ============================================================================
// Vendor Access RPCs (RBAC R-5 B, GDAP-light v3)
// ============================================================================

func (s *SecurityGRPCServer) ListVendorAccessRequests(ctx context.Context, req *securityv1.ListVendorAccessRequestsRequest) (*securityv1.ListVendorAccessRequestsResponse, error) {
	requests, err := s.vendorAccessService.List(ctx)
	if err != nil {
		return nil, mapSecurityError(err)
	}

	pbRequests := make([]*securityv1.VendorAccessRequest, 0, len(requests))
	for _, r := range requests {
		pbRequests = append(pbRequests, toProtoVendorAccessRequest(r))
	}

	return &securityv1.ListVendorAccessRequestsResponse{Requests: pbRequests}, nil
}

func (s *SecurityGRPCServer) ApproveVendorAccessRequest(ctx context.Context, req *securityv1.ApproveVendorAccessRequestRequest) (*securityv1.ApproveVendorAccessRequestResponse, error) {
	id, err := uuid.Parse(req.RequestId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request_id")
	}
	actorID, err := uuid.Parse(req.ActorId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid actor_id")
	}

	updated, err := s.vendorAccessService.Approve(ctx, id, actorID, req.SensitiveAck)
	if err != nil {
		return nil, mapSecurityError(err)
	}

	return &securityv1.ApproveVendorAccessRequestResponse{
		Request: toProtoVendorAccessRequest(updated),
	}, nil
}

func (s *SecurityGRPCServer) DeclineVendorAccessRequest(ctx context.Context, req *securityv1.DeclineVendorAccessRequestRequest) (*securityv1.DeclineVendorAccessRequestResponse, error) {
	id, err := uuid.Parse(req.RequestId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request_id")
	}

	updated, err := s.vendorAccessService.Decline(ctx, id)
	if err != nil {
		return nil, mapSecurityError(err)
	}

	return &securityv1.DeclineVendorAccessRequestResponse{
		Request: toProtoVendorAccessRequest(updated),
	}, nil
}

func (s *SecurityGRPCServer) CounterProposeVendorAccessRequest(ctx context.Context, req *securityv1.CounterProposeVendorAccessRequestRequest) (*securityv1.CounterProposeVendorAccessRequestResponse, error) {
	id, err := uuid.Parse(req.RequestId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request_id")
	}
	proposedStart, err := time.Parse("2006-01-02", req.ProposedStart)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid proposed_start, expected YYYY-MM-DD")
	}

	updated, err := s.vendorAccessService.CounterPropose(ctx, id, proposedStart)
	if err != nil {
		return nil, mapSecurityError(err)
	}

	return &securityv1.CounterProposeVendorAccessRequestResponse{
		Request: toProtoVendorAccessRequest(updated),
	}, nil
}

func (s *SecurityGRPCServer) RevokeVendorAccessRequest(ctx context.Context, req *securityv1.RevokeVendorAccessRequestRequest) (*securityv1.RevokeVendorAccessRequestResponse, error) {
	id, err := uuid.Parse(req.RequestId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request_id")
	}
	actorID, err := uuid.Parse(req.ActorId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid actor_id")
	}

	updated, err := s.vendorAccessService.Revoke(ctx, id, actorID)
	if err != nil {
		return nil, mapSecurityError(err)
	}

	return &securityv1.RevokeVendorAccessRequestResponse{
		Request: toProtoVendorAccessRequest(updated),
	}, nil
}

func (s *SecurityGRPCServer) GetExportDownload(ctx context.Context, req *securityv1.GetExportDownloadRequest) (*securityv1.GetExportDownloadResponse, error) {
	if req.DownloadToken == "" {
		return nil, status.Error(codes.InvalidArgument, "download_token is required")
	}

	data, filename, err := s.gdprService.GetExportDownload(ctx, req.DownloadToken)
	if err != nil {
		return nil, mapSecurityError(err)
	}

	return &securityv1.GetExportDownloadResponse{
		Data:     data,
		Filename: filename,
	}, nil
}

func (s *SecurityGRPCServer) PreviewErasure(ctx context.Context, req *securityv1.PreviewErasureRequest) (*securityv1.PreviewErasureResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	modules, total, err := s.gdprService.PreviewErasure(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "preview erasure failed")
	}

	pbModules := make([]*securityv1.ModuleErasurePreview, 0, len(modules))
	for _, m := range modules {
		pbModules = append(pbModules, &securityv1.ModuleErasurePreview{
			ModuleName:  m.ModuleName,
			RecordCount: int32(m.RecordCount),
			Action:      m.Action,
		})
	}

	return &securityv1.PreviewErasureResponse{
		Modules:      pbModules,
		TotalRecords: int32(total),
	}, nil
}

func (s *SecurityGRPCServer) ExecuteErasure(ctx context.Context, req *securityv1.ExecuteErasureRequest) (*securityv1.ExecuteErasureResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	executedBy, err := uuid.Parse(req.ExecutedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid executed_by")
	}

	entry, err := s.gdprService.ExecuteErasure(ctx, userID, executedBy)
	if err != nil {
		return nil, mapSecurityError(err)
	}

	pbModules := make([]*securityv1.ModuleErasurePreview, 0, len(entry.ModulesAffected))
	for moduleName, detail := range entry.ModulesAffected {
		pbModules = append(pbModules, &securityv1.ModuleErasurePreview{
			ModuleName: moduleName,
			Action:     detail,
		})
	}

	return &securityv1.ExecuteErasureResponse{
		AnonymizedLabel:  entry.AnonymizedLabel,
		ConfirmationHash: entry.ConfirmationHash,
		ModulesAffected:  pbModules,
	}, nil
}

// DSARSearch performs an Art. 15 GDPR cross-module lookup for a person within the
// caller's tenant. The tenant is taken from the authenticated context (not the
// request) so a caller cannot search another tenant's data.
func (s *SecurityGRPCServer) DSARSearch(ctx context.Context, req *securityv1.DSARSearchRequest) (*securityv1.DSARSearchResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing tenant_id: %v", err)
	}

	query := strings.TrimSpace(req.Query)
	if len([]rune(query)) < 2 {
		return nil, status.Error(codes.InvalidArgument, "query must be at least 2 characters")
	}

	persons, err := gdpr.SearchByQuery(ctx, s.pool, tenantID, query)
	if err != nil {
		slog.Error("dsar search failed", "error", err)
		return nil, status.Error(codes.Internal, "dsar search failed")
	}

	resp := &securityv1.DSARSearchResponse{Persons: make([]*securityv1.DSARPerson, 0, len(persons))}
	for _, p := range persons {
		pbPerson := &securityv1.DSARPerson{
			Id:      p.ID,
			Name:    p.Name,
			Email:   p.Email,
			Company: p.Company,
			Avatar:  p.Avatar,
			Modules: make([]*securityv1.DSARModule, 0, len(p.Modules)),
		}
		for _, m := range p.Modules {
			pbModule := &securityv1.DSARModule{
				Module:  m.Module,
				Columns: m.Columns,
				Records: make([]*securityv1.DSARRecord, 0, len(m.Records)),
			}
			for _, r := range m.Records {
				pbRecord := &securityv1.DSARRecord{Fields: make([]*securityv1.DSARField, 0, len(r.Fields))}
				for _, f := range r.Fields {
					pbRecord.Fields = append(pbRecord.Fields, &securityv1.DSARField{Key: f.Key, Value: f.Value})
				}
				pbModule.Records = append(pbModule.Records, pbRecord)
			}
			pbPerson.Modules = append(pbPerson.Modules, pbModule)
		}
		resp.Persons = append(resp.Persons, pbPerson)
	}
	return resp, nil
}

// ============================================================================
// Password Policy RPCs
// ============================================================================

func (s *SecurityGRPCServer) GetPasswordPolicy(ctx context.Context, _ *securityv1.GetPasswordPolicyRequest) (*securityv1.GetPasswordPolicyResponse, error) {
	policy, err := s.passwordService.GetPolicy(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get password policy")
	}

	return &securityv1.GetPasswordPolicyResponse{
		Policy: toProtoPasswordPolicy(policy),
	}, nil
}

func (s *SecurityGRPCServer) UpdatePasswordPolicy(ctx context.Context, req *securityv1.UpdatePasswordPolicyRequest) (*securityv1.UpdatePasswordPolicyResponse, error) {
	if req.Policy == nil {
		return nil, status.Error(codes.InvalidArgument, "policy is required")
	}

	updatedBy, err := uuid.Parse(req.UpdatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid updated_by")
	}

	policy := &models.PasswordPolicy{
		MinLength:         int(req.Policy.MinLength),
		RequireUppercase:  req.Policy.RequireUppercase,
		RequireLowercase:  req.Policy.RequireLowercase,
		RequireDigit:      req.Policy.RequireDigit,
		RequireSpecial:    req.Policy.RequireSpecial,
		MinEntropy:        req.Policy.MinEntropy,
		PreventReuseCount: int(req.Policy.PreventReuseCount),
	}

	if req.Policy.MaxAgeDays > 0 {
		days := int(req.Policy.MaxAgeDays)
		policy.MaxAgeDays = &days
	}

	// policy.ID is intentionally not taken from req.Policy.Id: UpdatePolicy resolves
	// the row to update server-side, scoped to the caller's tenant, so a client can
	// never target another tenant's policy row by supplying an arbitrary ID.
	if err := s.passwordService.UpdatePolicy(ctx, policy, updatedBy); err != nil {
		return nil, status.Error(codes.Internal, "failed to update password policy")
	}

	return &securityv1.UpdatePasswordPolicyResponse{
		Policy: toProtoPasswordPolicy(policy),
	}, nil
}

func (s *SecurityGRPCServer) ValidatePassword(ctx context.Context, req *securityv1.ValidatePasswordRequest) (*securityv1.ValidatePasswordResponse, error) {
	if req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	valid, violations, err := s.passwordService.ValidatePassword(ctx, req.Password)
	if err != nil {
		return nil, status.Error(codes.Internal, "password validation failed")
	}

	// If user_id is provided, also check password history
	if valid && req.UserId != "" {
		userID, parseErr := uuid.Parse(req.UserId)
		if parseErr == nil {
			notReused, histErr := s.passwordService.CheckPasswordHistory(ctx, userID, req.Password)
			if histErr != nil {
				slog.Warn("password history check failed", "error", histErr)
			} else if !notReused {
				valid = false
				violations = append(violations, "password was recently used")
			}
		}
	}

	return &securityv1.ValidatePasswordResponse{
		Valid:      valid,
		Violations: violations,
	}, nil
}

// ============================================================================
// IP Access Rules RPCs (direct pgx queries, no separate service)
// ============================================================================

func (s *SecurityGRPCServer) ListIPRules(ctx context.Context, req *securityv1.ListIPRulesRequest) (*securityv1.ListIPRulesResponse, error) {
	query := `SELECT id, ip_cidr::text, rule_type, description, created_by, created_at FROM ip_access_rules`
	var args []interface{}

	if req.RuleType != "" {
		query += ` WHERE rule_type = $1`
		args = append(args, req.RuleType)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list IP rules")
	}
	defer rows.Close()

	rules := make([]*securityv1.IPAccessRule, 0)
	for rows.Next() {
		var rule models.IPAccessRule
		if scanErr := rows.Scan(&rule.ID, &rule.IPCIDR, &rule.RuleType, &rule.Description, &rule.CreatedBy, &rule.CreatedAt); scanErr != nil {
			return nil, status.Error(codes.Internal, "failed to scan IP rule")
		}
		rules = append(rules, toProtoIPRule(&rule))
	}
	if err := rows.Err(); err != nil {
		return nil, status.Error(codes.Internal, "failed to iterate IP rules")
	}

	return &securityv1.ListIPRulesResponse{
		Rules: rules,
	}, nil
}

// CreateIPRule persists an allow/block rule. It does NOT check whether the
// calling admin's own IP would still pass the resulting rule set --
// gateway.IPFilterMiddleware enforces rules on every request once loaded, so
// a "block 0.0.0.0/0" or an "allow" rule that excludes the admin's own
// network can lock every admin out (verified: no self-lockout guard exists
// anywhere in this path). Fixing that is a product decision (soft warning?
// hard block? break-glass bypass?) and is out of scope here; documented
// per cov-gateway-security-retention-ip-password-routes (2026-08-23).
func (s *SecurityGRPCServer) CreateIPRule(ctx context.Context, req *securityv1.CreateIPRuleRequest) (*securityv1.CreateIPRuleResponse, error) {
	if req.IpCidr == "" {
		return nil, status.Error(codes.InvalidArgument, "ip_cidr is required")
	}
	// Postgres' CIDR column type rejects any value with host bits set (e.g.
	// "192.168.1.5/24" -- valid inet, invalid cidr). Mirror that here so the
	// rejection surfaces as 400 instead of a generic 500 from the INSERT.
	if ip, network, err := net.ParseCIDR(req.IpCidr); err != nil || !ip.Equal(network.IP) {
		return nil, status.Error(codes.InvalidArgument, "ip_cidr must be valid CIDR notation with no host bits set")
	}
	if req.RuleType != models.IPRuleAllow && req.RuleType != models.IPRuleBlock {
		return nil, status.Error(codes.InvalidArgument, "rule_type must be 'allow' or 'block'")
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to determine tenant")
	}

	rule := &models.IPAccessRule{
		ID:          uuid.New(),
		TenantID:    tenantID,
		IPCIDR:      req.IpCidr,
		RuleType:    req.RuleType,
		Description: req.Description,
		CreatedAt:   time.Now().UTC(),
	}

	if req.CreatedBy != "" {
		parsed, err := uuid.Parse(req.CreatedBy)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid created_by")
		}
		rule.CreatedBy = &parsed
	}

	_, err = s.pool.Exec(ctx,
		`INSERT INTO ip_access_rules (id, tenant_id, ip_cidr, rule_type, description, created_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		rule.ID, rule.TenantID, rule.IPCIDR, rule.RuleType, rule.Description, rule.CreatedBy, rule.CreatedAt,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create IP rule")
	}

	slog.Info("IP access rule created",
		"rule_id", rule.ID,
		"ip_cidr", rule.IPCIDR,
		"rule_type", rule.RuleType,
	)

	return &securityv1.CreateIPRuleResponse{
		Rule: toProtoIPRule(rule),
	}, nil
}

func (s *SecurityGRPCServer) DeleteIPRule(ctx context.Context, req *securityv1.DeleteIPRuleRequest) (*securityv1.DeleteIPRuleResponse, error) {
	ruleID, err := uuid.Parse(req.RuleId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid rule_id")
	}

	result, err := s.pool.Exec(ctx, `DELETE FROM ip_access_rules WHERE id = $1`, ruleID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to delete IP rule")
	}
	if result.RowsAffected() == 0 {
		return nil, status.Error(codes.NotFound, "IP rule not found")
	}

	slog.Info("IP access rule deleted", "rule_id", ruleID)

	return &securityv1.DeleteIPRuleResponse{}, nil
}

// ============================================================================
// Retention Policy RPCs (direct pgx queries, same pattern as IP rules)
// ============================================================================

func (s *SecurityGRPCServer) ListRetentionPolicies(ctx context.Context, _ *securityv1.ListRetentionPoliciesRequest) (*securityv1.ListRetentionPoliciesResponse, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, resource_type, retention_days, action, enabled, description, created_by, created_at, updated_at
		   FROM retention_policies
		  ORDER BY resource_type ASC`,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list retention policies")
	}
	defer rows.Close()

	policies := make([]*securityv1.RetentionPolicy, 0)
	for rows.Next() {
		var p models.RetentionPolicy
		if scanErr := rows.Scan(
			&p.ID, &p.ResourceType, &p.RetentionDays, &p.Action,
			&p.Enabled, &p.Description, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
		); scanErr != nil {
			return nil, status.Error(codes.Internal, "failed to scan retention policy")
		}
		policies = append(policies, toProtoRetentionPolicy(&p))
	}
	if err := rows.Err(); err != nil {
		return nil, status.Error(codes.Internal, "failed to iterate retention policies")
	}

	return &securityv1.ListRetentionPoliciesResponse{Policies: policies}, nil
}

func (s *SecurityGRPCServer) CreateRetentionPolicy(ctx context.Context, req *securityv1.CreateRetentionPolicyRequest) (*securityv1.CreateRetentionPolicyResponse, error) {
	if req.ResourceType == "" {
		return nil, status.Error(codes.InvalidArgument, "resource_type is required")
	}
	if req.RetentionDays <= 0 {
		return nil, status.Error(codes.InvalidArgument, "retention_days must be greater than 0")
	}
	if req.Action != models.RetentionActionDelete && req.Action != models.RetentionActionAnonymize {
		return nil, status.Error(codes.InvalidArgument, "action must be 'delete' or 'anonymize'")
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to determine tenant")
	}

	p := &models.RetentionPolicy{
		ID:            uuid.New(),
		TenantID:      tenantID,
		ResourceType:  req.ResourceType,
		RetentionDays: int(req.RetentionDays),
		Action:        req.Action,
		Enabled:       req.Enabled,
		Description:   req.Description,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	if req.CreatedBy != "" {
		parsed, err := uuid.Parse(req.CreatedBy)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid created_by")
		}
		p.CreatedBy = &parsed
	}

	_, err = s.pool.Exec(ctx,
		`INSERT INTO retention_policies
		   (id, tenant_id, resource_type, retention_days, action, enabled, description, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		p.ID, p.TenantID, p.ResourceType, p.RetentionDays, p.Action,
		p.Enabled, p.Description, p.CreatedBy, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "uq_retention_policies_tenant_resource") {
			return nil, status.Error(codes.AlreadyExists, "retention policy for this resource_type already exists")
		}
		return nil, status.Error(codes.Internal, "failed to create retention policy")
	}

	slog.Info("retention policy created",
		"policy_id", p.ID,
		"resource_type", p.ResourceType,
		"retention_days", p.RetentionDays,
		"action", p.Action,
	)

	return &securityv1.CreateRetentionPolicyResponse{Policy: toProtoRetentionPolicy(p)}, nil
}

func (s *SecurityGRPCServer) UpdateRetentionPolicy(ctx context.Context, req *securityv1.UpdateRetentionPolicyRequest) (*securityv1.UpdateRetentionPolicyResponse, error) {
	policyID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid policy id")
	}
	if req.RetentionDays <= 0 {
		return nil, status.Error(codes.InvalidArgument, "retention_days must be greater than 0")
	}
	if req.Action != models.RetentionActionDelete && req.Action != models.RetentionActionAnonymize {
		return nil, status.Error(codes.InvalidArgument, "action must be 'delete' or 'anonymize'")
	}

	now := time.Now().UTC()
	var p models.RetentionPolicy
	row := s.pool.QueryRow(ctx,
		`UPDATE retention_policies
		    SET retention_days = $1, action = $2, enabled = $3, description = $4, updated_at = $5
		  WHERE id = $6
		  RETURNING id, resource_type, retention_days, action, enabled, description, created_by, created_at, updated_at`,
		req.RetentionDays, req.Action, req.Enabled, req.Description, now, policyID,
	)
	if scanErr := row.Scan(
		&p.ID, &p.ResourceType, &p.RetentionDays, &p.Action,
		&p.Enabled, &p.Description, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
	); scanErr != nil {
		if strings.Contains(scanErr.Error(), "no rows") {
			return nil, status.Error(codes.NotFound, "retention policy not found")
		}
		return nil, status.Error(codes.Internal, "failed to update retention policy")
	}

	slog.Info("retention policy updated",
		"policy_id", p.ID,
		"retention_days", p.RetentionDays,
		"action", p.Action,
		"enabled", p.Enabled,
	)

	return &securityv1.UpdateRetentionPolicyResponse{Policy: toProtoRetentionPolicy(&p)}, nil
}

func (s *SecurityGRPCServer) DeleteRetentionPolicy(ctx context.Context, req *securityv1.DeleteRetentionPolicyRequest) (*securityv1.DeleteRetentionPolicyResponse, error) {
	policyID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid policy id")
	}

	result, err := s.pool.Exec(ctx, `DELETE FROM retention_policies WHERE id = $1`, policyID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to delete retention policy")
	}
	if result.RowsAffected() == 0 {
		return nil, status.Error(codes.NotFound, "retention policy not found")
	}

	slog.Info("retention policy deleted", "policy_id", policyID)

	return &securityv1.DeleteRetentionPolicyResponse{}, nil
}

// GetLatestRetentionRun reads the run log written by the scheduler
// (internal/security/gdpr/retention_scheduler.go) -- it has no engine
// reference of its own and never triggers a run, it only shows what the
// last one did. RLS on retention_runs/retention_run_items (both created in
// 000316) scopes both queries to the caller's tenant automatically, the
// same pattern ListRetentionPolicies relies on.
func (s *SecurityGRPCServer) GetLatestRetentionRun(ctx context.Context, _ *securityv1.GetLatestRetentionRunRequest) (*securityv1.GetLatestRetentionRunResponse, error) {
	var (
		runID                                                          uuid.UUID
		mode, runStatus, triggeredBy                                   string
		policiesTotal, recordsMatched, recordsAffected, recordsSkipped int
		runError                                                       *string
		startedAt                                                      time.Time
		finishedAt                                                     *time.Time
	)
	err := s.pool.QueryRow(ctx,
		`SELECT id, mode, status, triggered_by, policies_total, records_matched,
		        records_affected, records_skipped, error, started_at, finished_at
		   FROM retention_runs
		  ORDER BY started_at DESC
		  LIMIT 1`,
	).Scan(&runID, &mode, &runStatus, &triggeredBy, &policiesTotal,
		&recordsMatched, &recordsAffected, &recordsSkipped, &runError, &startedAt, &finishedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return &securityv1.GetLatestRetentionRunResponse{HasRun: false}, nil
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to load latest retention run")
	}

	pbRun := &securityv1.RetentionRun{
		Id:              runID.String(),
		Mode:            mode,
		Status:          runStatus,
		TriggeredBy:     triggeredBy,
		PoliciesTotal:   int32(policiesTotal),
		RecordsMatched:  int32(recordsMatched),
		RecordsAffected: int32(recordsAffected),
		RecordsSkipped:  int32(recordsSkipped),
		StartedAt:       timestamppb.New(startedAt),
	}
	if runError != nil {
		pbRun.Error = *runError
	}
	if finishedAt != nil {
		pbRun.FinishedAt = timestamppb.New(*finishedAt)
	}

	rows, err := s.pool.Query(ctx,
		`SELECT id, resource_type, action, retention_days, cutoff, status,
		        matched, affected, skipped, skip_reasons, message
		   FROM retention_run_items
		  WHERE run_id = $1
		  ORDER BY resource_type ASC`,
		runID,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to load retention run items")
	}
	defer rows.Close()

	items := make([]*securityv1.RetentionRunItem, 0)
	for rows.Next() {
		var (
			itemID                            uuid.UUID
			resourceType, action, itemStatus  string
			retentionDays                     int
			cutoff                            time.Time
			matched, affected, skipped        int
			skipReasonsRaw                    []byte
			message                           *string
		)
		if scanErr := rows.Scan(&itemID, &resourceType, &action, &retentionDays, &cutoff,
			&itemStatus, &matched, &affected, &skipped, &skipReasonsRaw, &message,
		); scanErr != nil {
			return nil, status.Error(codes.Internal, "failed to scan retention run item")
		}

		var skipReasons []struct {
			RecordID uuid.UUID `json:"record_id"`
			Reason   string    `json:"reason"`
		}
		if unmarshalErr := json.Unmarshal(skipReasonsRaw, &skipReasons); unmarshalErr != nil {
			return nil, status.Error(codes.Internal, "failed to parse retention skip reasons")
		}
		pbSkips := make([]*securityv1.RetentionSkipReason, 0, len(skipReasons))
		for _, sr := range skipReasons {
			pbSkips = append(pbSkips, &securityv1.RetentionSkipReason{
				RecordId: sr.RecordID.String(),
				Reason:   sr.Reason,
			})
		}

		pbItem := &securityv1.RetentionRunItem{
			Id:            itemID.String(),
			ResourceType:  resourceType,
			Action:        action,
			RetentionDays: int32(retentionDays),
			Cutoff:        timestamppb.New(cutoff),
			Status:        itemStatus,
			Matched:       int32(matched),
			Affected:      int32(affected),
			Skipped:       int32(skipped),
			SkipReasons:   pbSkips,
		}
		if message != nil {
			pbItem.Message = *message
		}
		items = append(items, pbItem)
	}
	if err := rows.Err(); err != nil {
		return nil, status.Error(codes.Internal, "failed to iterate retention run items")
	}

	return &securityv1.GetLatestRetentionRunResponse{
		HasRun: true,
		Run:    pbRun,
		Items:  items,
	}, nil
}

// ============================================================================
// Proto type conversions
// ============================================================================

func toProtoAuditEntry(e *models.AuditEntry) *securityv1.AuditEntry {
	pb := &securityv1.AuditEntry{
		Id:           e.ID.String(),
		SequenceNum:  e.SequenceNum,
		Timestamp:    timestamppb.New(e.Timestamp),
		Action:       e.Action,
		Target:       e.Target,
		TargetType:   e.TargetType,
		Details:      e.Details,
		IpAddress:    e.IPAddress,
		UserAgent:    e.UserAgent,
		Result:       e.Result,
		PreviousHash: e.PreviousHash,
		EntryHash:    e.EntryHash,
	}
	if e.UserID != nil {
		pb.UserId = e.UserID.String()
	}
	return pb
}

func toProtoVaultSecret(s *models.VaultSecret) *securityv1.VaultSecret {
	pb := &securityv1.VaultSecret{
		Id:          s.ID.String(),
		KeyName:     s.KeyName,
		KeyVersion:  int32(s.KeyVersion),
		Description: s.Description,
		CreatedAt:   timestamppb.New(s.CreatedAt),
		UpdatedAt:   timestamppb.New(s.UpdatedAt),
	}
	if s.CreatedBy != nil {
		pb.CreatedBy = s.CreatedBy.String()
	}
	return pb
}

func toProtoGDPRExport(e *models.GDPRExportRequest) *securityv1.GDPRExportRequest {
	pb := &securityv1.GDPRExportRequest{
		Id:          e.ID.String(),
		UserId:      e.UserID.String(),
		Status:      e.Status,
		RequestedAt: timestamppb.New(e.RequestedAt),
		ReviewNote:  e.ReviewNote,
	}
	if e.ReviewedBy != nil {
		pb.ReviewedBy = e.ReviewedBy.String()
	}
	if e.ReviewedAt != nil {
		pb.ReviewedAt = timestamppb.New(*e.ReviewedAt)
	}
	if e.DownloadToken != "" {
		pb.DownloadToken = e.DownloadToken
	}
	if e.DownloadExpiresAt != nil {
		pb.DownloadExpiresAt = timestamppb.New(*e.DownloadExpiresAt)
	}
	return pb
}

func toProtoVendorAccessRequest(r *models.VendorAccessRequest) *securityv1.VendorAccessRequest {
	agents := make([]*securityv1.VendorAccessAgent, 0, len(r.Agents))
	for _, a := range r.Agents {
		agents = append(agents, &securityv1.VendorAccessAgent{Name: a.Name})
	}

	pb := &securityv1.VendorAccessRequest{
		Id:             r.ID.String(),
		Reason:         r.Reason,
		Description:    r.Description,
		TicketRef:      r.TicketRef,
		Agents:         agents,
		Scope:          r.Scope,
		RequestedStart: r.RequestedStart.Format("2006-01-02"),
		DurationDays:   int32(r.DurationDays),
		ExpiresAt:      r.ExpiresAt.Format("2006-01-02"),
		Status:         r.Status,
		ApprovedBy:     r.ApprovedByName,
		RevokedBy:      r.RevokedByName,
		CreatedAt:      timestamppb.New(r.CreatedAt),
	}
	if r.CounterProposedStart != nil {
		pb.CounterProposedStart = r.CounterProposedStart.Format("2006-01-02")
	}
	if r.ApprovedAt != nil {
		pb.ApprovedAt = timestamppb.New(*r.ApprovedAt)
	}
	if r.SensitiveAck != nil {
		pb.SensitiveAck = *r.SensitiveAck
	}
	if r.RevokedAt != nil {
		pb.RevokedAt = timestamppb.New(*r.RevokedAt)
	}
	if r.CompletedAt != nil {
		pb.CompletedAt = timestamppb.New(*r.CompletedAt)
	}
	return pb
}

func toProtoPasswordPolicy(p *models.PasswordPolicy) *securityv1.PasswordPolicy {
	pb := &securityv1.PasswordPolicy{
		Id:                p.ID.String(),
		MinLength:         int32(p.MinLength),
		RequireUppercase:  p.RequireUppercase,
		RequireLowercase:  p.RequireLowercase,
		RequireDigit:      p.RequireDigit,
		RequireSpecial:    p.RequireSpecial,
		MinEntropy:        p.MinEntropy,
		PreventReuseCount: int32(p.PreventReuseCount),
		UpdatedAt:         timestamppb.New(p.UpdatedAt),
	}
	if p.MaxAgeDays != nil {
		pb.MaxAgeDays = int32(*p.MaxAgeDays)
	}
	if p.UpdatedBy != nil {
		pb.UpdatedBy = p.UpdatedBy.String()
	}
	return pb
}

func toProtoIPRule(r *models.IPAccessRule) *securityv1.IPAccessRule {
	pb := &securityv1.IPAccessRule{
		Id:          r.ID.String(),
		IpCidr:      r.IPCIDR,
		RuleType:    r.RuleType,
		Description: r.Description,
		CreatedAt:   timestamppb.New(r.CreatedAt),
	}
	if r.CreatedBy != nil {
		pb.CreatedBy = r.CreatedBy.String()
	}
	return pb
}

func toProtoRetentionPolicy(p *models.RetentionPolicy) *securityv1.RetentionPolicy {
	pb := &securityv1.RetentionPolicy{
		Id:            p.ID.String(),
		ResourceType:  p.ResourceType,
		RetentionDays: int32(p.RetentionDays),
		Action:        p.Action,
		Enabled:       p.Enabled,
		Description:   p.Description,
		CreatedAt:     timestamppb.New(p.CreatedAt),
		UpdatedAt:     timestamppb.New(p.UpdatedAt),
	}
	if p.CreatedBy != nil {
		pb.CreatedBy = p.CreatedBy.String()
	}
	return pb
}

// mapSecurityError maps domain errors to gRPC status errors.
func mapSecurityError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, audit.ErrUnsupportedFormat):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, vault.ErrSecretNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, vault.ErrEmptyKeyName),
		errors.Is(err, vault.ErrEmptyPayload):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, gdpr.ErrExportNotFound),
		errors.Is(err, gdpr.ErrTokenNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, gdpr.ErrExportAlreadyPending):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, gdpr.ErrInvalidExportStatus):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, gdpr.ErrExportExpired):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, gdpr.ErrExportNotReady):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, vendoraccess.ErrRequestNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, vendoraccess.ErrInvalidStatus):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, vendoraccess.ErrSensitiveAckRequired):
		return status.Error(codes.OutOfRange, err.Error())
	default:
		slog.Error("unhandled security service error", "error", err)
		return status.Error(codes.Internal, "internal error")
	}
}
