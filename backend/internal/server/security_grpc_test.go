package server

// Coverage for internal/server/security_grpc.go (BACKLOG b-cov-server-security).
// The five sub-services (audit, vault, gdpr, password, vendoraccess) each have
// their own package-level service_test.go covering business logic -- this file
// covers the gRPC handler layer: request validation, proto<->domain conversion,
// mapSecurityError, and the security-relevant behaviors called out in the
// backlog notes (vault secrets never leak plaintext outside GetVaultSecret,
// DSGVO erasure is only exercised through its validation/error paths, never a
// real deletion run).
//
// Unlike helpdesk_grpc.go, most of these RPCs have little or no validation
// before the service call -- ListVaultSecrets, GetPasswordPolicy,
// VerifyAuditChain and ListVendorAccessRequests go straight to the service.
// A nil-service server would panic on those, so this file wires all five
// sub-services against small in-memory stub repositories instead of using a
// nil-service server. ListIPRules and ListRetentionPolicies query s.pool
// directly with zero validation beforehand -- ListRetentionPolicies is not
// reachable at all without a real DB pool and is intentionally left
// uncovered here (see JOURNAL.md for iteration 46); ListIPRules now has its
// own DB-backed test in security_grpc_ip_rules_db_test.go.

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/security/audit"
	"github.com/kmuhub/kmuhub/internal/security/gdpr"
	"github.com/kmuhub/kmuhub/internal/security/password"
	"github.com/kmuhub/kmuhub/internal/security/vault"
	"github.com/kmuhub/kmuhub/internal/security/vendoraccess"
	securityv1 "github.com/kmuhub/kmuhub/proto/security/v1"
)

// ============================================================================
// Stub repositories
// ============================================================================

type stubAuditRepo struct {
	mu         sync.Mutex
	entries    []*models.AuditEntry
	lastHash   string
	chainValid bool
	firstBad   int64
}

func newStubAuditRepo() *stubAuditRepo {
	return &stubAuditRepo{chainValid: true}
}

func (r *stubAuditRepo) Create(_ context.Context, entry *models.AuditEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	entry.SequenceNum = int64(len(r.entries) + 1)
	cp := *entry
	r.entries = append(r.entries, &cp)
	return nil
}

func (r *stubAuditRepo) List(_ context.Context, filter *models.AuditFilter) ([]*models.AuditEntry, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*models.AuditEntry
	for _, e := range r.entries {
		if e.TenantID != filter.TenantID {
			continue
		}
		if filter.UserID != nil && (e.UserID == nil || *e.UserID != *filter.UserID) {
			continue
		}
		if filter.Action != "" && e.Action != filter.Action {
			continue
		}
		if filter.Result != "" && e.Result != filter.Result {
			continue
		}
		cp := *e
		out = append(out, &cp)
	}
	return out, len(out), nil
}

func (r *stubAuditRepo) GetLastHash(_ context.Context) (string, error) {
	return r.lastHash, nil
}

func (r *stubAuditRepo) VerifyChain(_ context.Context, _, _ int64) (bool, int64, error) {
	return r.chainValid, r.firstBad, nil
}

type stubVaultRepo struct {
	mu      sync.Mutex
	secrets map[string]*models.VaultSecret // key: tenantID/keyName
}

func newStubVaultRepo() *stubVaultRepo {
	return &stubVaultRepo{secrets: make(map[string]*models.VaultSecret)}
}

func vaultRepoKey(tenantID uuid.UUID, keyName string) string {
	return tenantID.String() + "/" + keyName
}

func (r *stubVaultRepo) GetByKeyName(_ context.Context, tenantID uuid.UUID, keyName string) (*models.VaultSecret, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.secrets[vaultRepoKey(tenantID, keyName)]
	if !ok {
		return nil, vault.ErrSecretNotFound
	}
	cp := *s
	return &cp, nil
}

// List redacts encrypted_value the same way the real repository's contract
// documents (Repository.List doc comment) -- the service layer never
// re-decrypts for a list response, so this stub mirrors that by clearing it.
func (r *stubVaultRepo) List(_ context.Context, tenantID uuid.UUID) ([]*models.VaultSecret, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*models.VaultSecret
	for _, s := range r.secrets {
		if s.TenantID != tenantID {
			continue
		}
		cp := *s
		cp.EncryptedValue = ""
		out = append(out, &cp)
	}
	return out, nil
}

func (r *stubVaultRepo) Create(_ context.Context, secret *models.VaultSecret) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *secret
	r.secrets[vaultRepoKey(secret.TenantID, secret.KeyName)] = &cp
	return nil
}

func (r *stubVaultRepo) Update(_ context.Context, secret *models.VaultSecret) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *secret
	r.secrets[vaultRepoKey(secret.TenantID, secret.KeyName)] = &cp
	return nil
}

func (r *stubVaultRepo) Delete(_ context.Context, tenantID, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, s := range r.secrets {
		if s.TenantID == tenantID && s.ID == id {
			delete(r.secrets, k)
			return nil
		}
	}
	return vault.ErrSecretNotFound
}

type stubGDPRRepo struct {
	mu          sync.Mutex
	exports     map[uuid.UUID]*models.GDPRExportRequest
	erasureLogs []*models.GDPRErasureLog
	labelSeq    int
}

func newStubGDPRRepo() *stubGDPRRepo {
	return &stubGDPRRepo{exports: make(map[uuid.UUID]*models.GDPRExportRequest)}
}

var errGDPRExportNotFound = errors.New("stub: export request not found")

func (r *stubGDPRRepo) CreateExportRequest(_ context.Context, req *models.GDPRExportRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *req
	r.exports[req.ID] = &cp
	return nil
}

func (r *stubGDPRRepo) GetExportRequest(_ context.Context, _, id uuid.UUID) (*models.GDPRExportRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	req, ok := r.exports[id]
	if !ok {
		return nil, errGDPRExportNotFound
	}
	cp := *req
	return &cp, nil
}

func (r *stubGDPRRepo) ListExportRequests(_ context.Context, _, userID uuid.UUID, status string) ([]*models.GDPRExportRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*models.GDPRExportRequest
	for _, req := range r.exports {
		if userID != uuid.Nil && req.UserID != userID {
			continue
		}
		if status != "" && req.Status != status {
			continue
		}
		cp := *req
		out = append(out, &cp)
	}
	return out, nil
}

func (r *stubGDPRRepo) UpdateExportStatus(_ context.Context, _ uuid.UUID, req *models.GDPRExportRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.exports[req.ID]; !ok {
		return errGDPRExportNotFound
	}
	cp := *req
	r.exports[req.ID] = &cp
	return nil
}

func (r *stubGDPRRepo) StoreExportResult(_ context.Context, _, id uuid.UUID, data []byte, token string, expiresAt any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	req, ok := r.exports[id]
	if !ok {
		return errGDPRExportNotFound
	}
	req.ExportData = data
	req.DownloadToken = token
	if t, ok := expiresAt.(time.Time); ok {
		req.DownloadExpiresAt = &t
	}
	return nil
}

func (r *stubGDPRRepo) GetExportByToken(_ context.Context, _ uuid.UUID, token string) (*models.GDPRExportRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, req := range r.exports {
		if req.DownloadToken == token {
			cp := *req
			return &cp, nil
		}
	}
	return nil, errGDPRExportNotFound
}

func (r *stubGDPRRepo) MarkDownloaded(_ context.Context, _, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	req, ok := r.exports[id]
	if !ok {
		return errGDPRExportNotFound
	}
	now := time.Now().UTC()
	req.DownloadedAt = &now
	return nil
}

func (r *stubGDPRRepo) CreateErasureLog(_ context.Context, log *models.GDPRErasureLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.erasureLogs = append(r.erasureLogs, log)
	return nil
}

func (r *stubGDPRRepo) GetNextAnonymizedLabel(_ context.Context) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.labelSeq++
	return fmt.Sprintf("Geloeschter Benutzer #%03d", r.labelSeq), nil
}

type stubPasswordRepo struct {
	mu       sync.Mutex
	policies map[uuid.UUID]*models.PasswordPolicy
	history  map[uuid.UUID][]string
}

func newStubPasswordRepo() *stubPasswordRepo {
	return &stubPasswordRepo{
		policies: make(map[uuid.UUID]*models.PasswordPolicy),
		history:  make(map[uuid.UUID][]string),
	}
}

func (r *stubPasswordRepo) GetPolicy(_ context.Context, tenantID uuid.UUID) (*models.PasswordPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.policies[tenantID]
	if !ok {
		p = &models.PasswordPolicy{
			ID:         uuid.New(),
			TenantID:   tenantID,
			MinLength:  12,
			MinEntropy: 40,
			UpdatedAt:  time.Now().UTC(),
		}
		r.policies[tenantID] = p
	}
	cp := *p
	return &cp, nil
}

func (r *stubPasswordRepo) UpdatePolicy(_ context.Context, tenantID uuid.UUID, policy *models.PasswordPolicy) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *policy
	r.policies[tenantID] = &cp
	return nil
}

func (r *stubPasswordRepo) AddPasswordHistory(_ context.Context, userID uuid.UUID, passwordHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.history[userID] = append(r.history[userID], passwordHash)
	return nil
}

func (r *stubPasswordRepo) GetPasswordHistory(_ context.Context, userID uuid.UUID, limit int) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	h := r.history[userID]
	if len(h) > limit {
		h = h[len(h)-limit:]
	}
	out := make([]string, len(h))
	copy(out, h)
	return out, nil
}

type stubVendorAccessRepo struct {
	mu       sync.Mutex
	requests map[uuid.UUID]*models.VendorAccessRequest
}

func newStubVendorAccessRepo() *stubVendorAccessRepo {
	return &stubVendorAccessRepo{requests: make(map[uuid.UUID]*models.VendorAccessRequest)}
}

func (r *stubVendorAccessRepo) seed(req *models.VendorAccessRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *req
	r.requests[req.ID] = &cp
}

func (r *stubVendorAccessRepo) CreateRequest(_ context.Context, req *models.VendorAccessRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *req
	r.requests[req.ID] = &cp
	return nil
}

func (r *stubVendorAccessRepo) GetRequest(_ context.Context, tenantID, id uuid.UUID) (*models.VendorAccessRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	req, ok := r.requests[id]
	if !ok || req.TenantID != tenantID {
		return nil, vendoraccess.ErrRequestNotFound
	}
	cp := *req
	return &cp, nil
}

func (r *stubVendorAccessRepo) ListRequests(_ context.Context, tenantID uuid.UUID) ([]*models.VendorAccessRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*models.VendorAccessRequest
	for _, req := range r.requests {
		if req.TenantID == tenantID {
			cp := *req
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *stubVendorAccessRepo) UpdateStatus(_ context.Context, tenantID uuid.UUID, req *models.VendorAccessRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.requests[req.ID]
	if !ok || existing.TenantID != tenantID {
		return vendoraccess.ErrRequestNotFound
	}
	cp := *req
	r.requests[req.ID] = &cp
	return nil
}

// ============================================================================
// Test server wiring
// ============================================================================

const testVaultMasterSecret = "test-vault-master-secret-min-32-chars!!"

type securityTestRepos struct {
	audit        *stubAuditRepo
	vault        *stubVaultRepo
	gdpr         *stubGDPRRepo
	password     *stubPasswordRepo
	vendorAccess *stubVendorAccessRepo
}

// newTestSecurityGRPCServer wires all five sub-services against in-memory stub
// repositories. pool stays nil: ListIPRules, CreateIPRule/DeleteIPRule (beyond
// validation), ListRetentionPolicies and Create/Update/DeleteRetentionPolicy
// (beyond validation) all touch s.pool directly and are not exercised here.
func newTestSecurityGRPCServer(t *testing.T) (*SecurityGRPCServer, *securityTestRepos) {
	t.Helper()
	repos := &securityTestRepos{
		audit:        newStubAuditRepo(),
		vault:        newStubVaultRepo(),
		gdpr:         newStubGDPRRepo(),
		password:     newStubPasswordRepo(),
		vendorAccess: newStubVendorAccessRepo(),
	}

	vaultSvc, err := vault.NewService(repos.vault, testVaultMasterSecret)
	require.NoError(t, err)

	srv := NewSecurityGRPCServer(
		audit.NewService(repos.audit),
		vaultSvc,
		gdpr.NewService(repos.gdpr),
		password.NewService(repos.password),
		vendoraccess.NewService(repos.vendorAccess),
		nil, // pool
	)
	return srv, repos
}

// ctxWithTenant is defined in dialer_grpc_test.go.

// ============================================================================
// mapSecurityError -- full sentinel table
// ============================================================================

func TestMapSecurityError_AllSentinels(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want codes.Code
	}{
		{"nil", nil, codes.OK},
		{"vault_secret_not_found", vault.ErrSecretNotFound, codes.NotFound},
		{"vault_empty_key_name", vault.ErrEmptyKeyName, codes.InvalidArgument},
		{"vault_empty_payload", vault.ErrEmptyPayload, codes.InvalidArgument},
		{"gdpr_export_already_pending", gdpr.ErrExportAlreadyPending, codes.AlreadyExists},
		{"gdpr_invalid_export_status", gdpr.ErrInvalidExportStatus, codes.FailedPrecondition},
		{"gdpr_export_expired", gdpr.ErrExportExpired, codes.FailedPrecondition},
		{"gdpr_export_not_ready", gdpr.ErrExportNotReady, codes.FailedPrecondition},
		{"vendoraccess_request_not_found", vendoraccess.ErrRequestNotFound, codes.NotFound},
		{"vendoraccess_invalid_status", vendoraccess.ErrInvalidStatus, codes.FailedPrecondition},
		{"vendoraccess_sensitive_ack_required", vendoraccess.ErrSensitiveAckRequired, codes.OutOfRange},
		{"unknown_error", errors.New("boom"), codes.Internal},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mapSecurityError(tc.err)
			if tc.err == nil {
				assert.NoError(t, got)
				return
			}
			requireGRPCCode(t, got, tc.want)
		})
	}
}

// ============================================================================
// Proto <-> domain converters (nil-optional vs populated)
// ============================================================================

func TestToProtoAuditEntry(t *testing.T) {
	base := &models.AuditEntry{
		ID:           uuid.New(),
		SequenceNum:  7,
		Timestamp:    time.Now().UTC(),
		Action:       "login",
		Target:       "user",
		TargetType:   "auth",
		Details:      "{}",
		IPAddress:    "10.0.0.1",
		UserAgent:    "test-agent",
		Result:       "success",
		PreviousHash: "prev",
		EntryHash:    "hash",
	}

	t.Run("nil_user_id", func(t *testing.T) {
		pb := toProtoAuditEntry(base)
		assert.Empty(t, pb.UserId)
		assert.Equal(t, "login", pb.Action)
		assert.Equal(t, int64(7), pb.SequenceNum)
	})

	t.Run("populated_user_id", func(t *testing.T) {
		userID := uuid.New()
		withUser := *base
		withUser.UserID = &userID
		pb := toProtoAuditEntry(&withUser)
		assert.Equal(t, userID.String(), pb.UserId)
	})
}

func TestToProtoVaultSecret(t *testing.T) {
	base := &models.VaultSecret{
		ID:          uuid.New(),
		KeyName:     "smtp_password",
		KeyVersion:  1,
		Description: "SMTP creds",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	t.Run("nil_created_by", func(t *testing.T) {
		pb := toProtoVaultSecret(base)
		assert.Empty(t, pb.CreatedBy)
		assert.Equal(t, "smtp_password", pb.KeyName)
	})

	t.Run("populated_created_by", func(t *testing.T) {
		createdBy := uuid.New()
		withCreator := *base
		withCreator.CreatedBy = &createdBy
		pb := toProtoVaultSecret(&withCreator)
		assert.Equal(t, createdBy.String(), pb.CreatedBy)
	})
}

func TestToProtoGDPRExport(t *testing.T) {
	base := &models.GDPRExportRequest{
		ID:          uuid.New(),
		UserID:      uuid.New(),
		Status:      models.ExportStatusPending,
		RequestedAt: time.Now().UTC(),
	}

	t.Run("nil_optionals", func(t *testing.T) {
		pb := toProtoGDPRExport(base)
		assert.Empty(t, pb.ReviewedBy)
		assert.Nil(t, pb.ReviewedAt)
		assert.Empty(t, pb.DownloadToken)
		assert.Nil(t, pb.DownloadExpiresAt)
	})

	t.Run("populated_optionals", func(t *testing.T) {
		reviewer := uuid.New()
		reviewedAt := time.Now().UTC()
		expiresAt := time.Now().UTC().Add(7 * 24 * time.Hour)
		withReview := *base
		withReview.ReviewedBy = &reviewer
		withReview.ReviewedAt = &reviewedAt
		withReview.DownloadToken = "tok123"
		withReview.DownloadExpiresAt = &expiresAt
		pb := toProtoGDPRExport(&withReview)
		assert.Equal(t, reviewer.String(), pb.ReviewedBy)
		require.NotNil(t, pb.ReviewedAt)
		assert.Equal(t, "tok123", pb.DownloadToken)
		require.NotNil(t, pb.DownloadExpiresAt)
	})
}

func TestToProtoVendorAccessRequest(t *testing.T) {
	base := &models.VendorAccessRequest{
		ID:             uuid.New(),
		Reason:         "incident support",
		Description:    "desc",
		Agents:         []models.VendorAccessAgent{{Name: "Alice"}},
		Scope:          []string{"logs"},
		RequestedStart: time.Now().UTC(),
		DurationDays:   3,
		ExpiresAt:      time.Now().UTC().Add(3 * 24 * time.Hour),
		Status:         models.VendorAccessStatusPending,
		CreatedAt:      time.Now().UTC(),
	}

	t.Run("nil_optionals", func(t *testing.T) {
		pb := toProtoVendorAccessRequest(base)
		assert.Empty(t, pb.CounterProposedStart)
		assert.Nil(t, pb.ApprovedAt)
		assert.False(t, pb.SensitiveAck)
		assert.Nil(t, pb.RevokedAt)
		assert.Nil(t, pb.CompletedAt)
	})

	t.Run("populated_optionals", func(t *testing.T) {
		counterStart := time.Now().UTC()
		approvedAt := time.Now().UTC()
		ack := true
		revokedAt := time.Now().UTC()
		completedAt := time.Now().UTC()
		withAll := *base
		withAll.CounterProposedStart = &counterStart
		withAll.ApprovedAt = &approvedAt
		withAll.SensitiveAck = &ack
		withAll.RevokedAt = &revokedAt
		withAll.CompletedAt = &completedAt
		pb := toProtoVendorAccessRequest(&withAll)
		assert.NotEmpty(t, pb.CounterProposedStart)
		require.NotNil(t, pb.ApprovedAt)
		assert.True(t, pb.SensitiveAck)
		require.NotNil(t, pb.RevokedAt)
		require.NotNil(t, pb.CompletedAt)
	})
}

func TestToProtoPasswordPolicy(t *testing.T) {
	base := &models.PasswordPolicy{
		ID:                uuid.New(),
		MinLength:         12,
		RequireUppercase:  true,
		MinEntropy:        40,
		PreventReuseCount: 5,
		UpdatedAt:         time.Now().UTC(),
	}

	t.Run("nil_optionals", func(t *testing.T) {
		pb := toProtoPasswordPolicy(base)
		assert.Equal(t, int32(0), pb.MaxAgeDays)
		assert.Empty(t, pb.UpdatedBy)
	})

	t.Run("populated_optionals", func(t *testing.T) {
		days := 90
		updatedBy := uuid.New()
		withOpts := *base
		withOpts.MaxAgeDays = &days
		withOpts.UpdatedBy = &updatedBy
		pb := toProtoPasswordPolicy(&withOpts)
		assert.Equal(t, int32(90), pb.MaxAgeDays)
		assert.Equal(t, updatedBy.String(), pb.UpdatedBy)
	})
}

func TestToProtoIPRule(t *testing.T) {
	base := &models.IPAccessRule{
		ID:        uuid.New(),
		IPCIDR:    "10.0.0.0/8",
		RuleType:  models.IPRuleAllow,
		CreatedAt: time.Now().UTC(),
	}

	t.Run("nil_created_by", func(t *testing.T) {
		pb := toProtoIPRule(base)
		assert.Empty(t, pb.CreatedBy)
	})

	t.Run("populated_created_by", func(t *testing.T) {
		createdBy := uuid.New()
		withCreator := *base
		withCreator.CreatedBy = &createdBy
		pb := toProtoIPRule(&withCreator)
		assert.Equal(t, createdBy.String(), pb.CreatedBy)
	})
}

func TestToProtoRetentionPolicy(t *testing.T) {
	base := &models.RetentionPolicy{
		ID:            uuid.New(),
		ResourceType:  "audit_log",
		RetentionDays: 365,
		Action:        models.RetentionActionAnonymize,
		Enabled:       true,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	t.Run("nil_created_by", func(t *testing.T) {
		pb := toProtoRetentionPolicy(base)
		assert.Empty(t, pb.CreatedBy)
	})

	t.Run("populated_created_by", func(t *testing.T) {
		createdBy := uuid.New()
		withCreator := *base
		withCreator.CreatedBy = &createdBy
		pb := toProtoRetentionPolicy(&withCreator)
		assert.Equal(t, createdBy.String(), pb.CreatedBy)
	})
}

// ============================================================================
// Validation-path table: parse/tenant/enum errors reachable without touching
// s.pool (ListIPRules and ListRetentionPolicies have no pre-validation at all
// and are therefore not representable here -- see file header).
// ============================================================================

func TestSecurityGRPC_ValidationErrors(t *testing.T) {
	srv, _ := newTestSecurityGRPCServer(t)
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)
	bgCtx := context.Background() // no tenant

	cases := []struct {
		name string
		want codes.Code
		call func() error
	}{
		// Audit
		{"CreateAuditEntry/invalid_user_id", codes.InvalidArgument, func() error {
			_, err := srv.CreateAuditEntry(ctx, &securityv1.CreateAuditEntryRequest{UserId: "not-a-uuid"})
			return err
		}},
		{"ListAuditEntries/missing_tenant", codes.InvalidArgument, func() error {
			_, err := srv.ListAuditEntries(bgCtx, &securityv1.ListAuditEntriesRequest{})
			return err
		}},
		{"ListAuditEntries/invalid_filter_user_id", codes.InvalidArgument, func() error {
			_, err := srv.ListAuditEntries(ctx, &securityv1.ListAuditEntriesRequest{
				Filter: &securityv1.AuditFilter{UserId: "not-a-uuid"},
			})
			return err
		}},
		{"ExportAuditLog/missing_tenant", codes.InvalidArgument, func() error {
			_, err := srv.ExportAuditLog(bgCtx, &securityv1.ExportAuditLogRequest{})
			return err
		}},
		{"ExportAuditLog/invalid_filter_user_id", codes.InvalidArgument, func() error {
			_, err := srv.ExportAuditLog(ctx, &securityv1.ExportAuditLogRequest{
				Filter: &securityv1.AuditFilter{UserId: "not-a-uuid"},
			})
			return err
		}},

		// Vault
		{"GetVaultSecret/empty_key_name", codes.InvalidArgument, func() error {
			_, err := srv.GetVaultSecret(ctx, &securityv1.GetVaultSecretRequest{})
			return err
		}},
		{"GetVaultSecret/not_found", codes.NotFound, func() error {
			_, err := srv.GetVaultSecret(ctx, &securityv1.GetVaultSecretRequest{KeyName: "does-not-exist"})
			return err
		}},
		{"SetVaultSecret/empty_key_name", codes.InvalidArgument, func() error {
			_, err := srv.SetVaultSecret(ctx, &securityv1.SetVaultSecretRequest{PlaintextValue: "x"})
			return err
		}},
		{"SetVaultSecret/empty_plaintext", codes.InvalidArgument, func() error {
			_, err := srv.SetVaultSecret(ctx, &securityv1.SetVaultSecretRequest{KeyName: "k"})
			return err
		}},
		{"SetVaultSecret/invalid_created_by", codes.InvalidArgument, func() error {
			_, err := srv.SetVaultSecret(ctx, &securityv1.SetVaultSecretRequest{
				KeyName: "k", PlaintextValue: "v", CreatedBy: "not-a-uuid",
			})
			return err
		}},
		{"DeleteVaultSecret/empty_key_name", codes.InvalidArgument, func() error {
			_, err := srv.DeleteVaultSecret(ctx, &securityv1.DeleteVaultSecretRequest{})
			return err
		}},
		{"DeleteVaultSecret/not_found", codes.NotFound, func() error {
			_, err := srv.DeleteVaultSecret(ctx, &securityv1.DeleteVaultSecretRequest{KeyName: "does-not-exist"})
			return err
		}},

		// GDPR
		{"RequestDataExport/invalid_user_id", codes.InvalidArgument, func() error {
			_, err := srv.RequestDataExport(ctx, &securityv1.RequestDataExportRequest{UserId: "not-a-uuid"})
			return err
		}},
		{"ListDataExports/invalid_user_id", codes.InvalidArgument, func() error {
			_, err := srv.ListDataExports(ctx, &securityv1.ListDataExportsRequest{UserId: "not-a-uuid"})
			return err
		}},
		{"ApproveDataExport/invalid_export_id", codes.InvalidArgument, func() error {
			_, err := srv.ApproveDataExport(ctx, &securityv1.ApproveDataExportRequest{
				ExportId: "not-a-uuid", ReviewedBy: uuid.New().String(),
			})
			return err
		}},
		{"ApproveDataExport/invalid_reviewed_by", codes.InvalidArgument, func() error {
			_, err := srv.ApproveDataExport(ctx, &securityv1.ApproveDataExportRequest{
				ExportId: uuid.New().String(), ReviewedBy: "not-a-uuid",
			})
			return err
		}},
		{"DenyDataExport/invalid_export_id", codes.InvalidArgument, func() error {
			_, err := srv.DenyDataExport(ctx, &securityv1.DenyDataExportRequest{
				ExportId: "not-a-uuid", ReviewedBy: uuid.New().String(),
			})
			return err
		}},
		{"DenyDataExport/invalid_reviewed_by", codes.InvalidArgument, func() error {
			_, err := srv.DenyDataExport(ctx, &securityv1.DenyDataExportRequest{
				ExportId: uuid.New().String(), ReviewedBy: "not-a-uuid",
			})
			return err
		}},
		{"GetExportDownload/empty_token", codes.InvalidArgument, func() error {
			_, err := srv.GetExportDownload(ctx, &securityv1.GetExportDownloadRequest{})
			return err
		}},
		{"PreviewErasure/invalid_user_id", codes.InvalidArgument, func() error {
			_, err := srv.PreviewErasure(ctx, &securityv1.PreviewErasureRequest{UserId: "not-a-uuid"})
			return err
		}},
		{"ExecuteErasure/invalid_user_id", codes.InvalidArgument, func() error {
			_, err := srv.ExecuteErasure(ctx, &securityv1.ExecuteErasureRequest{
				UserId: "not-a-uuid", ExecutedBy: uuid.New().String(),
			})
			return err
		}},
		{"ExecuteErasure/invalid_executed_by", codes.InvalidArgument, func() error {
			_, err := srv.ExecuteErasure(ctx, &securityv1.ExecuteErasureRequest{
				UserId: uuid.New().String(), ExecutedBy: "not-a-uuid",
			})
			return err
		}},
		{"DSARSearch/missing_tenant", codes.InvalidArgument, func() error {
			_, err := srv.DSARSearch(bgCtx, &securityv1.DSARSearchRequest{Query: "ab"})
			return err
		}},
		{"DSARSearch/query_too_short", codes.InvalidArgument, func() error {
			_, err := srv.DSARSearch(ctx, &securityv1.DSARSearchRequest{Query: "a"})
			return err
		}},

		// Password
		{"UpdatePasswordPolicy/nil_policy", codes.InvalidArgument, func() error {
			_, err := srv.UpdatePasswordPolicy(ctx, &securityv1.UpdatePasswordPolicyRequest{
				UpdatedBy: uuid.New().String(),
			})
			return err
		}},
		{"UpdatePasswordPolicy/invalid_updated_by", codes.InvalidArgument, func() error {
			_, err := srv.UpdatePasswordPolicy(ctx, &securityv1.UpdatePasswordPolicyRequest{
				Policy: &securityv1.PasswordPolicy{MinLength: 12}, UpdatedBy: "not-a-uuid",
			})
			return err
		}},
		{"ValidatePassword/empty_password", codes.InvalidArgument, func() error {
			_, err := srv.ValidatePassword(ctx, &securityv1.ValidatePasswordRequest{})
			return err
		}},

		// IP rules (validation only -- CreateIPRule/DeleteIPRule touch s.pool
		// right after these checks, so nothing beyond this is reachable here)
		{"CreateIPRule/empty_cidr", codes.InvalidArgument, func() error {
			_, err := srv.CreateIPRule(ctx, &securityv1.CreateIPRuleRequest{RuleType: models.IPRuleAllow})
			return err
		}},
		{"CreateIPRule/invalid_rule_type", codes.InvalidArgument, func() error {
			_, err := srv.CreateIPRule(ctx, &securityv1.CreateIPRuleRequest{IpCidr: "10.0.0.0/8", RuleType: "maybe"})
			return err
		}},
		{"DeleteIPRule/invalid_rule_id", codes.InvalidArgument, func() error {
			_, err := srv.DeleteIPRule(ctx, &securityv1.DeleteIPRuleRequest{RuleId: "not-a-uuid"})
			return err
		}},

		// Retention policies (validation only -- same pool caveat as IP rules)
		{"CreateRetentionPolicy/empty_resource_type", codes.InvalidArgument, func() error {
			_, err := srv.CreateRetentionPolicy(ctx, &securityv1.CreateRetentionPolicyRequest{
				RetentionDays: 30, Action: models.RetentionActionDelete,
			})
			return err
		}},
		{"CreateRetentionPolicy/zero_retention_days", codes.InvalidArgument, func() error {
			_, err := srv.CreateRetentionPolicy(ctx, &securityv1.CreateRetentionPolicyRequest{
				ResourceType: "audit_log", Action: models.RetentionActionDelete,
			})
			return err
		}},
		{"CreateRetentionPolicy/invalid_action", codes.InvalidArgument, func() error {
			_, err := srv.CreateRetentionPolicy(ctx, &securityv1.CreateRetentionPolicyRequest{
				ResourceType: "audit_log", RetentionDays: 30, Action: "shred",
			})
			return err
		}},
		{"UpdateRetentionPolicy/invalid_id", codes.InvalidArgument, func() error {
			_, err := srv.UpdateRetentionPolicy(ctx, &securityv1.UpdateRetentionPolicyRequest{
				Id: "not-a-uuid", RetentionDays: 30, Action: models.RetentionActionDelete,
			})
			return err
		}},
		{"UpdateRetentionPolicy/zero_retention_days", codes.InvalidArgument, func() error {
			_, err := srv.UpdateRetentionPolicy(ctx, &securityv1.UpdateRetentionPolicyRequest{
				Id: uuid.New().String(), Action: models.RetentionActionDelete,
			})
			return err
		}},
		{"UpdateRetentionPolicy/invalid_action", codes.InvalidArgument, func() error {
			_, err := srv.UpdateRetentionPolicy(ctx, &securityv1.UpdateRetentionPolicyRequest{
				Id: uuid.New().String(), RetentionDays: 30, Action: "shred",
			})
			return err
		}},
		{"DeleteRetentionPolicy/invalid_id", codes.InvalidArgument, func() error {
			_, err := srv.DeleteRetentionPolicy(ctx, &securityv1.DeleteRetentionPolicyRequest{Id: "not-a-uuid"})
			return err
		}},

		// Vendor access
		{"ApproveVendorAccessRequest/invalid_request_id", codes.InvalidArgument, func() error {
			_, err := srv.ApproveVendorAccessRequest(ctx, &securityv1.ApproveVendorAccessRequestRequest{
				RequestId: "not-a-uuid", ActorId: uuid.New().String(),
			})
			return err
		}},
		{"ApproveVendorAccessRequest/invalid_actor_id", codes.InvalidArgument, func() error {
			_, err := srv.ApproveVendorAccessRequest(ctx, &securityv1.ApproveVendorAccessRequestRequest{
				RequestId: uuid.New().String(), ActorId: "not-a-uuid",
			})
			return err
		}},
		{"DeclineVendorAccessRequest/invalid_request_id", codes.InvalidArgument, func() error {
			_, err := srv.DeclineVendorAccessRequest(ctx, &securityv1.DeclineVendorAccessRequestRequest{
				RequestId: "not-a-uuid",
			})
			return err
		}},
		{"CounterProposeVendorAccessRequest/invalid_request_id", codes.InvalidArgument, func() error {
			_, err := srv.CounterProposeVendorAccessRequest(ctx, &securityv1.CounterProposeVendorAccessRequestRequest{
				RequestId: "not-a-uuid", ProposedStart: "2026-01-01",
			})
			return err
		}},
		{"CounterProposeVendorAccessRequest/invalid_date", codes.InvalidArgument, func() error {
			_, err := srv.CounterProposeVendorAccessRequest(ctx, &securityv1.CounterProposeVendorAccessRequestRequest{
				RequestId: uuid.New().String(), ProposedStart: "not-a-date",
			})
			return err
		}},
		{"RevokeVendorAccessRequest/invalid_request_id", codes.InvalidArgument, func() error {
			_, err := srv.RevokeVendorAccessRequest(ctx, &securityv1.RevokeVendorAccessRequestRequest{
				RequestId: "not-a-uuid", ActorId: uuid.New().String(),
			})
			return err
		}},
		{"RevokeVendorAccessRequest/invalid_actor_id", codes.InvalidArgument, func() error {
			_, err := srv.RevokeVendorAccessRequest(ctx, &securityv1.RevokeVendorAccessRequestRequest{
				RequestId: uuid.New().String(), ActorId: "not-a-uuid",
			})
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requireGRPCCode(t, tc.call(), tc.want)
		})
	}
}

// ============================================================================
// Vault secret masking -- the highest-value test in this file per the backlog
// notes. Everything except GetVaultSecret's decrypted_value field must never
// carry the plaintext.
// ============================================================================

func TestVaultSecret_PlaintextNeverLeaksOutsideGet(t *testing.T) {
	srv, _ := newTestSecurityGRPCServer(t)
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)

	const plaintext = "sk_live_mysupersecretvalue123"

	setResp, err := srv.SetVaultSecret(ctx, &securityv1.SetVaultSecretRequest{
		KeyName:        "stripe_key",
		PlaintextValue: plaintext,
		Description:    "Stripe API key",
	})
	require.NoError(t, err)
	require.NotNil(t, setResp.Secret)
	assert.NotEqual(t, plaintext, setResp.Secret.KeyName)
	assert.NotEqual(t, plaintext, setResp.Secret.Description)

	listResp, err := srv.ListVaultSecrets(ctx, &securityv1.ListVaultSecretsRequest{})
	require.NoError(t, err)
	require.Len(t, listResp.Secrets, 1)
	for _, s := range listResp.Secrets {
		assert.NotEqual(t, plaintext, s.KeyName)
		assert.NotEqual(t, plaintext, s.Description)
	}

	// Update (SetVaultSecret again for the same key_name) must not leak either.
	updateResp, err := srv.SetVaultSecret(ctx, &securityv1.SetVaultSecretRequest{
		KeyName:        "stripe_key",
		PlaintextValue: "sk_live_rotatedvalue456",
		Description:    "Stripe API key (rotated)",
	})
	require.NoError(t, err)
	assert.NotEqual(t, "sk_live_rotatedvalue456", updateResp.Secret.KeyName)
	assert.NotEqual(t, "sk_live_rotatedvalue456", updateResp.Secret.Description)

	// GetVaultSecret is the one RPC whose entire purpose is returning the
	// decrypted value -- confirm it actually does, and confirm the metadata
	// side of that same response still carries no plaintext field.
	getResp, err := srv.GetVaultSecret(ctx, &securityv1.GetVaultSecretRequest{KeyName: "stripe_key"})
	require.NoError(t, err)
	assert.Equal(t, "sk_live_rotatedvalue456", getResp.DecryptedValue)
	assert.NotEqual(t, "sk_live_rotatedvalue456", getResp.Secret.KeyName)

	deleteResp, err := srv.DeleteVaultSecret(ctx, &securityv1.DeleteVaultSecretRequest{KeyName: "stripe_key"})
	require.NoError(t, err)
	require.NotNil(t, deleteResp)
}

// A secret set for one tenant must not surface in another tenant's list --
// the vault handlers resolve tenant from context, never from the request.
func TestVaultSecret_CrossTenantIsolation(t *testing.T) {
	srv, _ := newTestSecurityGRPCServer(t)
	tenantA := ctxWithTenant(uuid.New())
	tenantB := ctxWithTenant(uuid.New())

	_, err := srv.SetVaultSecret(tenantA, &securityv1.SetVaultSecretRequest{
		KeyName: "tenant_a_only", PlaintextValue: "secret-a",
	})
	require.NoError(t, err)

	listB, err := srv.ListVaultSecrets(tenantB, &securityv1.ListVaultSecretsRequest{})
	require.NoError(t, err)
	assert.Empty(t, listB.Secrets)

	_, err = srv.GetVaultSecret(tenantB, &securityv1.GetVaultSecretRequest{KeyName: "tenant_a_only"})
	requireGRPCCode(t, err, codes.NotFound)
}

// ============================================================================
// Happy-path and domain-error coverage per RPC group
// ============================================================================

func TestAuditRPCs_HappyPaths(t *testing.T) {
	srv, _ := newTestSecurityGRPCServer(t)
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)

	createResp, err := srv.CreateAuditEntry(ctx, &securityv1.CreateAuditEntryRequest{
		Action: "login", Target: "user", Result: models.AuditResultSuccess,
	})
	require.NoError(t, err)
	assert.Equal(t, "login", createResp.Entry.Action)

	listResp, err := srv.ListAuditEntries(ctx, &securityv1.ListAuditEntriesRequest{})
	require.NoError(t, err)
	assert.Equal(t, int32(1), listResp.Total)

	csvResp, err := srv.ExportAuditLog(ctx, &securityv1.ExportAuditLogRequest{})
	require.NoError(t, err)
	assert.Equal(t, "text/csv", csvResp.ContentType)
	assert.NotEmpty(t, csvResp.Data)

	jsonResp, err := srv.ExportAuditLog(ctx, &securityv1.ExportAuditLogRequest{Format: "json"})
	require.NoError(t, err)
	assert.Equal(t, "application/json", jsonResp.ContentType)

	verifyResp, err := srv.VerifyAuditChain(ctx, &securityv1.VerifyAuditChainRequest{})
	require.NoError(t, err)
	assert.True(t, verifyResp.Valid)
	assert.Empty(t, verifyResp.ErrorMessage)
}

// TestExportAuditLog_UnsupportedFormatIsInvalidArgument proves the fix for a
// bug found while building cov-gateway-security-audit-and-vault-routes:
// ExportAuditLog used to wrap ANY error from Service.ExportEntries (including
// the audit.ErrUnsupportedFormat client-input error) as codes.Internal, so an
// unknown ?format=xml query returned a bare 500 "internal error" through
// respondGRPCError instead of a clear 400. It now routes the error through
// mapSecurityError, which maps audit.ErrUnsupportedFormat to InvalidArgument.
func TestExportAuditLog_UnsupportedFormatIsInvalidArgument(t *testing.T) {
	srv, _ := newTestSecurityGRPCServer(t)
	ctx := ctxWithTenant(uuid.New())

	_, err := srv.ExportAuditLog(ctx, &securityv1.ExportAuditLogRequest{Format: "xml"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// TestVerifyAuditChain_BrokenChainReportsError proves a broken hash chain is
// reported as such in the response body (Valid=false, ErrorMessage set) and
// not silently returned as a successful, empty-result verification.
func TestVerifyAuditChain_BrokenChainReportsError(t *testing.T) {
	srv, repos := newTestSecurityGRPCServer(t)
	ctx := ctxWithTenant(uuid.New())

	repos.audit.chainValid = false
	repos.audit.firstBad = 5

	resp, err := srv.VerifyAuditChain(ctx, &securityv1.VerifyAuditChainRequest{})
	require.NoError(t, err)
	assert.False(t, resp.Valid)
	assert.Equal(t, int64(5), resp.FirstInvalidSequence)
	assert.NotEmpty(t, resp.ErrorMessage)
}

func TestGDPRExportRPCs_HappyPathAndDomainErrors(t *testing.T) {
	srv, repos := newTestSecurityGRPCServer(t)
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)
	userID := uuid.New()

	reqResp, err := srv.RequestDataExport(ctx, &securityv1.RequestDataExportRequest{UserId: userID.String()})
	require.NoError(t, err)
	exportID := reqResp.ExportRequest.Id
	assert.Equal(t, models.ExportStatusPending, reqResp.ExportRequest.Status)

	// A second request for the same user while one is pending must be rejected
	// -- this is the ErrExportAlreadyPending sentinel, not a generic error.
	_, err = srv.RequestDataExport(ctx, &securityv1.RequestDataExportRequest{UserId: userID.String()})
	requireGRPCCode(t, err, codes.AlreadyExists)

	listResp, err := srv.ListDataExports(ctx, &securityv1.ListDataExportsRequest{})
	require.NoError(t, err)
	assert.Len(t, listResp.ExportRequests, 1)

	reviewer := uuid.New()
	approveResp, err := srv.ApproveDataExport(ctx, &securityv1.ApproveDataExportRequest{
		ExportId: exportID, ReviewedBy: reviewer.String(), ReviewNote: "ok",
	})
	require.NoError(t, err)
	assert.Equal(t, models.ExportStatusApproved, approveResp.ExportRequest.Status)

	// Approving again from a non-pending state must surface ErrInvalidExportStatus.
	_, err = srv.ApproveDataExport(ctx, &securityv1.ApproveDataExportRequest{
		ExportId: exportID, ReviewedBy: reviewer.String(),
	})
	requireGRPCCode(t, err, codes.FailedPrecondition)

	// Denying an approved (non-pending) request hits the same sentinel.
	_, err = srv.DenyDataExport(ctx, &securityv1.DenyDataExportRequest{
		ExportId: exportID, ReviewedBy: reviewer.String(),
	})
	requireGRPCCode(t, err, codes.FailedPrecondition)

	// GetExportDownload before the async ZIP generation has stored a token:
	// the export is not ready yet.
	repos.gdpr.mu.Lock()
	req := repos.gdpr.exports[uuid.MustParse(exportID)]
	req.Status = models.ExportStatusReady
	req.DownloadToken = "tok-ready"
	repos.gdpr.mu.Unlock()

	dlResp, err := srv.GetExportDownload(ctx, &securityv1.GetExportDownloadRequest{DownloadToken: "tok-ready"})
	require.NoError(t, err)
	assert.NotEmpty(t, dlResp.Filename)

	// An expired download must map to FailedPrecondition (ErrExportExpired).
	repos.gdpr.mu.Lock()
	past := time.Now().UTC().Add(-time.Hour)
	req.Status = models.ExportStatusReady
	req.DownloadExpiresAt = &past
	repos.gdpr.mu.Unlock()
	_, err = srv.GetExportDownload(ctx, &securityv1.GetExportDownloadRequest{DownloadToken: "tok-ready"})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestGDPRErasure_PreviewIsSafeExecuteIsValidationOnly(t *testing.T) {
	srv, _ := newTestSecurityGRPCServer(t)
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)
	userID := uuid.New()

	// PreviewErasure is read-only (no registered erasure handlers in this
	// server), safe to exercise end to end.
	previewResp, err := srv.PreviewErasure(ctx, &securityv1.PreviewErasureRequest{UserId: userID.String()})
	require.NoError(t, err)
	assert.Equal(t, int32(0), previewResp.TotalRecords)
	require.NotNil(t, previewResp.Modules)
	assert.Empty(t, previewResp.Modules)

	// ExecuteErasure is destructive by design -- per the backlog notes, this
	// file only exercises its validation/error paths (covered in the table
	// test above) plus this server's wire-shape guarantee, never a real
	// deletion run. With no erasure handlers registered here, it performs no
	// module writes and its ModulesAffected map stays empty.
	executeResp, err := srv.ExecuteErasure(ctx, &securityv1.ExecuteErasureRequest{
		UserId: userID.String(), ExecutedBy: uuid.New().String(),
	})
	require.NoError(t, err)
	require.NotNil(t, executeResp.ModulesAffected)
	assert.Empty(t, executeResp.ModulesAffected)
}

// TestSecurityGRPC_EmptyListsAreEmptySliceNotNull proves the four
// stub-repo-backed list RPCs serialize an empty result to [] rather than
// null: ListAuditEntries, ListVaultSecrets, ListDataExports and
// PreviewErasure (covered separately above) all used to declare their
// response slice with a nil `var`, which json/protojson renders as `null`.
func TestSecurityGRPC_EmptyListsAreEmptySliceNotNull(t *testing.T) {
	srv, _ := newTestSecurityGRPCServer(t)
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)

	auditResp, err := srv.ListAuditEntries(ctx, &securityv1.ListAuditEntriesRequest{})
	require.NoError(t, err)
	require.NotNil(t, auditResp.Entries)
	assert.Empty(t, auditResp.Entries)

	vaultResp, err := srv.ListVaultSecrets(ctx, &securityv1.ListVaultSecretsRequest{})
	require.NoError(t, err)
	require.NotNil(t, vaultResp.Secrets)
	assert.Empty(t, vaultResp.Secrets)

	exportsResp, err := srv.ListDataExports(ctx, &securityv1.ListDataExportsRequest{})
	require.NoError(t, err)
	require.NotNil(t, exportsResp.ExportRequests)
	assert.Empty(t, exportsResp.ExportRequests)
}

func TestPasswordRPCs_HappyPaths(t *testing.T) {
	srv, _ := newTestSecurityGRPCServer(t)
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)

	getResp, err := srv.GetPasswordPolicy(ctx, &securityv1.GetPasswordPolicyRequest{})
	require.NoError(t, err)
	assert.Equal(t, int32(12), getResp.Policy.MinLength)

	updatedBy := uuid.New()
	updateResp, err := srv.UpdatePasswordPolicy(ctx, &securityv1.UpdatePasswordPolicyRequest{
		Policy: &securityv1.PasswordPolicy{
			MinLength: 16, RequireDigit: true, MinEntropy: 50, PreventReuseCount: 3,
		},
		UpdatedBy: updatedBy.String(),
	})
	require.NoError(t, err)
	assert.Equal(t, int32(16), updateResp.Policy.MinLength)
	assert.Equal(t, updatedBy.String(), updateResp.Policy.UpdatedBy)

	// A short password against the now-16-char-minimum policy must fail.
	validateResp, err := srv.ValidatePassword(ctx, &securityv1.ValidatePasswordRequest{Password: "short1"})
	require.NoError(t, err)
	assert.False(t, validateResp.Valid)
	assert.NotEmpty(t, validateResp.Violations)

	// A long, high-entropy password satisfying every rule must pass.
	validateResp2, err := srv.ValidatePassword(ctx, &securityv1.ValidatePasswordRequest{
		Password: "Tr0ub4dor&3-Zephyr-Quokka!",
	})
	require.NoError(t, err)
	assert.True(t, validateResp2.Valid)
	assert.Empty(t, validateResp2.Violations)
}

func TestVendorAccessRPCs_HappyPathAndDomainErrors(t *testing.T) {
	srv, repos := newTestSecurityGRPCServer(t)
	tenantID := uuid.New()
	ctx := ctxWithTenant(tenantID)

	pending := &models.VendorAccessRequest{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Reason:         "support",
		Description:    "investigate ticket",
		Agents:         []models.VendorAccessAgent{{Name: "Bob"}},
		Scope:          []string{"logs"},
		RequestedStart: time.Now().UTC(),
		DurationDays:   2,
		ExpiresAt:      time.Now().UTC().Add(2 * 24 * time.Hour),
		Status:         models.VendorAccessStatusPending,
		CreatedAt:      time.Now().UTC(),
	}
	repos.vendorAccess.seed(pending)

	listResp, err := srv.ListVendorAccessRequests(ctx, &securityv1.ListVendorAccessRequestsRequest{})
	require.NoError(t, err)
	require.Len(t, listResp.Requests, 1)

	// Non-sensitive scope: approve succeeds without sensitive_ack.
	actorID := uuid.New()
	approveResp, err := srv.ApproveVendorAccessRequest(ctx, &securityv1.ApproveVendorAccessRequestRequest{
		RequestId: pending.ID.String(), ActorId: actorID.String(),
	})
	require.NoError(t, err)
	assert.Equal(t, models.VendorAccessStatusActive, approveResp.Request.Status)

	// Approving an already-active request hits ErrInvalidStatus.
	_, err = srv.ApproveVendorAccessRequest(ctx, &securityv1.ApproveVendorAccessRequestRequest{
		RequestId: pending.ID.String(), ActorId: actorID.String(),
	})
	requireGRPCCode(t, err, codes.FailedPrecondition)

	// Revoke the now-active request.
	revokeResp, err := srv.RevokeVendorAccessRequest(ctx, &securityv1.RevokeVendorAccessRequestRequest{
		RequestId: pending.ID.String(), ActorId: actorID.String(),
	})
	require.NoError(t, err)
	assert.Equal(t, models.VendorAccessStatusRevoked, revokeResp.Request.Status)

	// A sensitive scope requires sensitive_ack -- OutOfRange without it.
	sensitive := &models.VendorAccessRequest{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Reason:         "payroll fix",
		Description:    "adjust salary field",
		Agents:         []models.VendorAccessAgent{{Name: "Carol"}},
		Scope:          []string{"salary"},
		RequestedStart: time.Now().UTC(),
		DurationDays:   1,
		ExpiresAt:      time.Now().UTC().Add(24 * time.Hour),
		Status:         models.VendorAccessStatusPending,
		CreatedAt:      time.Now().UTC(),
	}
	repos.vendorAccess.seed(sensitive)

	_, err = srv.ApproveVendorAccessRequest(ctx, &securityv1.ApproveVendorAccessRequestRequest{
		RequestId: sensitive.ID.String(), ActorId: actorID.String(), SensitiveAck: false,
	})
	requireGRPCCode(t, err, codes.OutOfRange)

	ackResp, err := srv.ApproveVendorAccessRequest(ctx, &securityv1.ApproveVendorAccessRequestRequest{
		RequestId: sensitive.ID.String(), ActorId: actorID.String(), SensitiveAck: true,
	})
	require.NoError(t, err)
	assert.Equal(t, models.VendorAccessStatusActive, ackResp.Request.Status)

	// Decline and counter-propose against a fresh pending request.
	third := &models.VendorAccessRequest{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Reason:         "onboarding",
		Description:    "setup",
		RequestedStart: time.Now().UTC(),
		DurationDays:   1,
		ExpiresAt:      time.Now().UTC().Add(24 * time.Hour),
		Status:         models.VendorAccessStatusPending,
		CreatedAt:      time.Now().UTC(),
	}
	repos.vendorAccess.seed(third)

	counterResp, err := srv.CounterProposeVendorAccessRequest(ctx, &securityv1.CounterProposeVendorAccessRequestRequest{
		RequestId: third.ID.String(), ProposedStart: "2026-12-01",
	})
	require.NoError(t, err)
	assert.Equal(t, models.VendorAccessStatusCounterProposed, counterResp.Request.Status)
	assert.Equal(t, "2026-12-01", counterResp.Request.CounterProposedStart)

	fourth := &models.VendorAccessRequest{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Reason:         "cleanup",
		Description:    "remove stale rows",
		RequestedStart: time.Now().UTC(),
		DurationDays:   1,
		ExpiresAt:      time.Now().UTC().Add(24 * time.Hour),
		Status:         models.VendorAccessStatusPending,
		CreatedAt:      time.Now().UTC(),
	}
	repos.vendorAccess.seed(fourth)

	declineResp, err := srv.DeclineVendorAccessRequest(ctx, &securityv1.DeclineVendorAccessRequestRequest{
		RequestId: fourth.ID.String(),
	})
	require.NoError(t, err)
	assert.Equal(t, models.VendorAccessStatusDeclined, declineResp.Request.Status)
}

// A vendor-access request created for one tenant must not resolve under a
// different tenant's context -- GetRequest in the stub repo enforces this,
// mirroring the real postgres_repository's tenant-scoped lookup.
func TestVendorAccessRequest_CrossTenantIsolation(t *testing.T) {
	srv, repos := newTestSecurityGRPCServer(t)
	tenantA := uuid.New()
	tenantB := uuid.New()

	req := &models.VendorAccessRequest{
		ID:             uuid.New(),
		TenantID:       tenantA,
		Reason:         "support",
		Description:    "desc",
		RequestedStart: time.Now().UTC(),
		DurationDays:   1,
		ExpiresAt:      time.Now().UTC().Add(24 * time.Hour),
		Status:         models.VendorAccessStatusPending,
		CreatedAt:      time.Now().UTC(),
	}
	repos.vendorAccess.seed(req)

	_, err := srv.ApproveVendorAccessRequest(ctxWithTenant(tenantB), &securityv1.ApproveVendorAccessRequestRequest{
		RequestId: req.ID.String(), ActorId: uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.NotFound)
}
