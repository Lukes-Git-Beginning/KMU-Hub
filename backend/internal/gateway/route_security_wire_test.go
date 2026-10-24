package gateway

// Audit, vault, password policy, IP rule and export review routes as the security pages read
// them (security-types.ts). A loopback gRPC server plays the service.

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	authv1 "github.com/kmuhub/kmuhub/proto/auth/v1"
	securityv1 "github.com/kmuhub/kmuhub/proto/security/v1"
)

var securityWireTime = timestamppb.New(time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC))

type securityWireStub struct {
	securityv1.UnimplementedSecurityServiceServer
	badDetails bool
	policy     *securityv1.PasswordPolicy

	gotListFilter   *securityv1.AuditFilter
	gotExportFilter *securityv1.AuditFilter
	gotSet          *securityv1.SetVaultSecretRequest
	gotPolicy       *securityv1.PasswordPolicy
	gotApproveNote  string
	gotDenyNote     string
}

func (s *securityWireStub) ListAuditEntries(_ context.Context, req *securityv1.ListAuditEntriesRequest) (*securityv1.ListAuditEntriesResponse, error) {
	s.gotListFilter = req.GetFilter()
	details := `{"old_value":"a","new_value":"b"}`
	if s.badDetails {
		details = ""
	}
	return &securityv1.ListAuditEntriesResponse{
		Entries: []*securityv1.AuditEntry{{
			Id: "e1", SequenceNum: 51, Timestamp: securityWireTime, UserId: "u1", Action: "user.update",
			Target: "t", TargetType: "user", Details: details, Result: "success", EntryHash: "h",
		}},
		Total: 120,
	}, nil
}

func (s *securityWireStub) ExportAuditLog(_ context.Context, req *securityv1.ExportAuditLogRequest) (*securityv1.ExportAuditLogResponse, error) {
	s.gotExportFilter = req.GetFilter()
	return &securityv1.ExportAuditLogResponse{Data: []byte("a,b"), ContentType: "text/csv", Filename: "audit_log.csv"}, nil
}

func (s *securityWireStub) GetVaultSecret(context.Context, *securityv1.GetVaultSecretRequest) (*securityv1.GetVaultSecretResponse, error) {
	return &securityv1.GetVaultSecretResponse{
		Secret:         &securityv1.VaultSecret{Id: "v1", KeyName: "smtp_password", KeyVersion: 1},
		DecryptedValue: "hunter2",
	}, nil
}

func (s *securityWireStub) SetVaultSecret(_ context.Context, req *securityv1.SetVaultSecretRequest) (*securityv1.SetVaultSecretResponse, error) {
	s.gotSet = req
	return &securityv1.SetVaultSecretResponse{
		Secret: &securityv1.VaultSecret{Id: "v1", KeyName: req.GetKeyName(), Description: req.GetDescription(), KeyVersion: 1},
	}, nil
}

func (s *securityWireStub) GetPasswordPolicy(context.Context, *securityv1.GetPasswordPolicyRequest) (*securityv1.GetPasswordPolicyResponse, error) {
	p := s.policy
	if p == nil {
		p = &securityv1.PasswordPolicy{
			Id: "p1", MinLength: 12, RequireUppercase: true, RequireDigit: true, MinEntropy: 40,
			PreventReuseCount: 5, UpdatedAt: securityWireTime,
		}
	}
	return &securityv1.GetPasswordPolicyResponse{Policy: p}, nil
}

func (s *securityWireStub) UpdatePasswordPolicy(_ context.Context, req *securityv1.UpdatePasswordPolicyRequest) (*securityv1.UpdatePasswordPolicyResponse, error) {
	s.gotPolicy = req.GetPolicy()
	return &securityv1.UpdatePasswordPolicyResponse{Policy: req.GetPolicy()}, nil
}

func (s *securityWireStub) ValidatePassword(_ context.Context, req *securityv1.ValidatePasswordRequest) (*securityv1.ValidatePasswordResponse, error) {
	if req.GetPassword() == "good-Passw0rd!" {
		return &securityv1.ValidatePasswordResponse{Valid: true, Entropy: 61.5}, nil
	}
	return &securityv1.ValidatePasswordResponse{Violations: []string{"too short"}, Entropy: 8}, nil
}

func (s *securityWireStub) CreateIPRule(_ context.Context, req *securityv1.CreateIPRuleRequest) (*securityv1.CreateIPRuleResponse, error) {
	return &securityv1.CreateIPRuleResponse{
		Rule: &securityv1.IPAccessRule{Id: "r1", IpCidr: req.GetIpCidr(), RuleType: req.GetRuleType(), CreatedAt: securityWireTime},
	}, nil
}

func (s *securityWireStub) ApproveDataExport(_ context.Context, req *securityv1.ApproveDataExportRequest) (*securityv1.ApproveDataExportResponse, error) {
	s.gotApproveNote = req.GetReviewNote()
	return &securityv1.ApproveDataExportResponse{}, nil
}

func (s *securityWireStub) DenyDataExport(_ context.Context, req *securityv1.DenyDataExportRequest) (*securityv1.DenyDataExportResponse, error) {
	s.gotDenyNote = req.GetReviewNote()
	return &securityv1.DenyDataExportResponse{}, nil
}

func newSecurityWireRoutes(t *testing.T, stub *securityWireStub) *SecurityRoutes {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer()
	securityv1.RegisterSecurityServiceServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("auth", lis.Addr().String())
	t.Cleanup(reg.Close)
	return NewSecurityRoutes(reg)
}

func securityWireCall(t *testing.T, h http.HandlerFunc, method, target string, body any, params ...string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	if body != nil {
		req = httptest.NewRequest(method, target, jsonBody(t, body))
	}
	req = withAuth(req, testHRUserID, testTenantID)
	for i := 0; i+1 < len(params); i += 2 {
		req = withChiURLParam(req, params[i], params[i+1])
	}
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func TestSecurityWire_AuditEntryDetailsAreAnObject(t *testing.T) {
	sr := newSecurityWireRoutes(t, &securityWireStub{})
	rec := securityWireCall(t, sr.HandleListAuditEntries, http.MethodGet, "/audit", nil)
	assertStatus(t, rec, http.StatusOK)

	got := decodeBody(t, rec)
	assert.EqualValues(t, 120, got["total"])
	entries := got["entries"].([]any)
	require.Len(t, entries, 1)
	entry := entries[0].(map[string]any)
	assert.Equal(t, map[string]any{"old_value": "a", "new_value": "b"}, entry["details"])
	assert.Equal(t, "51", entry["sequence_num"])
	assert.Equal(t, "2026-10-23T12:00:00Z", entry["timestamp"])
}

func TestSecurityWire_AuditEntryWithoutDetailsGetsEmptyObject(t *testing.T) {
	sr := newSecurityWireRoutes(t, &securityWireStub{badDetails: true})
	rec := securityWireCall(t, sr.HandleListAuditEntries, http.MethodGet, "/audit", nil)
	assertStatus(t, rec, http.StatusOK)
	entry := decodeBody(t, rec)["entries"].([]any)[0].(map[string]any)
	assert.Equal(t, map[string]any{}, entry["details"])
}

func TestSecurityWire_AuditOffsetLimitSelectsSecondPage(t *testing.T) {
	stub := &securityWireStub{}
	sr := newSecurityWireRoutes(t, stub)

	rec := securityWireCall(t, sr.HandleListAuditEntries, http.MethodGet, "/audit?offset=50&limit=25", nil)
	assertStatus(t, rec, http.StatusOK)
	assert.EqualValues(t, 50, stub.gotListFilter.GetOffset())
	assert.EqualValues(t, 25, stub.gotListFilter.GetLimit())

	// callers that page by number still work
	securityWireCall(t, sr.HandleListAuditEntries, http.MethodGet, "/audit?page=3&page_size=20", nil)
	assert.EqualValues(t, 40, stub.gotListFilter.GetOffset())
	assert.EqualValues(t, 20, stub.gotListFilter.GetLimit())

	// the roles tab asks for 200 rows and gets them; more than the ceiling is clamped to it,
	// not replaced by the default window (the tab used to come back with 50 rows, mostly empty)
	securityWireCall(t, sr.HandleListAuditEntries, http.MethodGet, "/audit?offset=10&limit=200", nil)
	assert.EqualValues(t, 10, stub.gotListFilter.GetOffset())
	assert.EqualValues(t, 200, stub.gotListFilter.GetLimit())
	securityWireCall(t, sr.HandleListAuditEntries, http.MethodGet, "/audit?offset=10&limit=100000", nil)
	assert.EqualValues(t, 10, stub.gotListFilter.GetOffset())
	assert.EqualValues(t, auditPageSizeMax, stub.gotListFilter.GetLimit())

	// an unparsable or non-positive limit still falls back to the default window
	for _, q := range []string{"limit=abc", "limit=0", "limit=-5"} {
		securityWireCall(t, sr.HandleListAuditEntries, http.MethodGet, "/audit?"+q, nil)
		assert.EqualValues(t, 50, stub.gotListFilter.GetLimit(), q)
	}
}

func TestSecurityWire_AuditDateFiltersReachTheServiceAndExport(t *testing.T) {
	stub := &securityWireStub{}
	sr := newSecurityWireRoutes(t, stub)

	q := "?date_from=2026-10-01&date_to=2026-10-23&action=user.update&result=failure"
	rec := securityWireCall(t, sr.HandleListAuditEntries, http.MethodGet, "/audit"+q, nil)
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), stub.gotListFilter.GetDateFrom().AsTime())
	// a bare day is inclusive: the end bound is the last instant of the 23rd
	assert.Equal(t, time.Date(2026, 10, 23, 23, 59, 59, 999999999, time.UTC), stub.gotListFilter.GetDateTo().AsTime())
	assert.Equal(t, "user.update", stub.gotListFilter.GetAction())
	assert.Equal(t, "failure", stub.gotListFilter.GetResult())

	rec = securityWireCall(t, sr.HandleExportAuditLog, http.MethodGet, "/audit/export"+q+"&format=json", nil)
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), stub.gotExportFilter.GetDateFrom().AsTime())
	assert.Equal(t, "failure", stub.gotExportFilter.GetResult())
}

func TestSecurityWire_AuditMalformedDateIs400(t *testing.T) {
	sr := newSecurityWireRoutes(t, &securityWireStub{})
	for _, h := range []http.HandlerFunc{sr.HandleListAuditEntries, sr.HandleExportAuditLog} {
		rec := securityWireCall(t, h, http.MethodGet, "/audit?date_from=gestern", nil)
		assertStatus(t, rec, http.StatusBadRequest)
		assertErrorContains(t, rec, "date_from")
	}
}

func TestSecurityWire_VaultPutByKeyThenGetReadsKeyAndValue(t *testing.T) {
	stub := &securityWireStub{}
	sr := newSecurityWireRoutes(t, stub)

	rec := securityWireCall(t, sr.HandleSetVaultSecretByKey, http.MethodPut, "/vault/smtp_password",
		map[string]any{"value": "hunter2", "description": "Mailserver"}, "keyName", "smtp_password")
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, "smtp_password", stub.gotSet.GetKeyName())
	assert.Equal(t, "hunter2", stub.gotSet.GetPlaintextValue())
	assert.Equal(t, testHRUserID, stub.gotSet.GetCreatedBy())
	saved := decodeBody(t, rec)
	assert.Equal(t, "smtp_password", saved["key_name"])
	assert.NotContains(t, saved, "secret")

	rec = securityWireCall(t, sr.HandleGetVaultSecret, http.MethodGet, "/vault/smtp_password", nil, "keyName", "smtp_password")
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, map[string]any{"key_name": "smtp_password", "value": "hunter2"}, decodeBody(t, rec))
}

func TestSecurityWire_VaultPutByKeyRequiresValue(t *testing.T) {
	sr := newSecurityWireRoutes(t, &securityWireStub{})
	rec := securityWireCall(t, sr.HandleSetVaultSecretByKey, http.MethodPut, "/vault/k",
		map[string]any{"description": "x"}, "keyName", "k")
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestSecurityWire_VaultOldPutKeepsWorkingAndAnswersFlat(t *testing.T) {
	stub := &securityWireStub{}
	sr := newSecurityWireRoutes(t, stub)
	rec := securityWireCall(t, sr.HandleSetVaultSecret, http.MethodPut, "/vault",
		map[string]any{"key_name": "k", "plaintext_value": "v"})
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, "k", stub.gotSet.GetKeyName())
	assert.Equal(t, "k", decodeBody(t, rec)["key_name"])
}

func TestSecurityWire_PasswordValidateCarriesFailures(t *testing.T) {
	sr := newSecurityWireRoutes(t, &securityWireStub{})

	rec := securityWireCall(t, sr.HandleValidatePassword, http.MethodPost, "/password/validate", map[string]any{"password": "abc"})
	assertStatus(t, rec, http.StatusOK)
	got := decodeBody(t, rec)
	assert.Equal(t, []any{"too short"}, got["failures"])
	assert.Equal(t, []any{"too short"}, got["violations"])
	assert.Equal(t, false, got["valid"])

	rec = securityWireCall(t, sr.HandleValidatePassword, http.MethodPost, "/password/validate", map[string]any{"password": "good-Passw0rd!"})
	got = decodeBody(t, rec)
	assert.Equal(t, []any{}, got["failures"])
	assert.Equal(t, true, got["valid"])
}

func TestSecurityWire_PasswordPolicyIsFlatWithNullMaxAge(t *testing.T) {
	stub := &securityWireStub{}
	sr := newSecurityWireRoutes(t, stub)

	rec := securityWireCall(t, sr.HandleGetPasswordPolicy, http.MethodGet, "/password/policy", nil)
	assertStatus(t, rec, http.StatusOK)
	policy := decodeBody(t, rec)["policy"].(map[string]any)
	assert.EqualValues(t, 12, policy["min_length"])
	assert.Nil(t, policy["max_age_days"])
	assert.Contains(t, policy, "max_age_days")
	assert.Equal(t, false, policy["require_special"])

	body := map[string]any{
		"min_length": 14, "min_entropy": 50, "max_age_days": nil, "prevent_reuse_count": 3,
		"require_uppercase": true, "require_lowercase": true, "require_digit": false, "require_special": true,
	}
	rec = securityWireCall(t, sr.HandleUpdatePasswordPolicy, http.MethodPut, "/password/policy", body)
	assertStatus(t, rec, http.StatusOK)
	got := decodeBody(t, rec)
	assert.NotContains(t, got, "policy")
	assert.EqualValues(t, 14, got["min_length"])
	assert.Nil(t, got["max_age_days"])
	assert.EqualValues(t, 0, stub.gotPolicy.GetMaxAgeDays())

	body["max_age_days"] = 90
	rec = securityWireCall(t, sr.HandleUpdatePasswordPolicy, http.MethodPut, "/password/policy", body)
	assert.EqualValues(t, 90, decodeBody(t, rec)["max_age_days"])
	assert.EqualValues(t, 90, stub.gotPolicy.GetMaxAgeDays())
}

func TestSecurityWire_PasswordPolicyPartialBodyKeepsStoredValues(t *testing.T) {
	stub := &securityWireStub{policy: &securityv1.PasswordPolicy{
		Id: "p1", MinLength: 12, RequireUppercase: true, RequireDigit: true, MinEntropy: 40,
		MaxAgeDays: 60, PreventReuseCount: 5,
	}}
	sr := newSecurityWireRoutes(t, stub)

	rec := securityWireCall(t, sr.HandleUpdatePasswordPolicy, http.MethodPut, "/password/policy",
		map[string]any{"require_special": true})
	assertStatus(t, rec, http.StatusOK)
	p := stub.gotPolicy
	assert.EqualValues(t, 12, p.GetMinLength())
	assert.True(t, p.GetRequireUppercase())
	assert.True(t, p.GetRequireDigit())
	assert.True(t, p.GetRequireSpecial())
	assert.EqualValues(t, 40, p.GetMinEntropy())
	assert.EqualValues(t, 60, p.GetMaxAgeDays())
	assert.EqualValues(t, 5, p.GetPreventReuseCount())
}

func TestSecurityWire_PasswordPolicyRejectsZeroLengthAndBadMaxAge(t *testing.T) {
	sr := newSecurityWireRoutes(t, &securityWireStub{})
	rec := securityWireCall(t, sr.HandleUpdatePasswordPolicy, http.MethodPut, "/password/policy", map[string]any{"min_length": 0})
	assertStatus(t, rec, http.StatusBadRequest)

	rec = securityWireCall(t, sr.HandleUpdatePasswordPolicy, http.MethodPut, "/password/policy", map[string]any{"max_age_days": "soon"})
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestSecurityWire_ExportReviewNoteReachesTheService(t *testing.T) {
	stub := &securityWireStub{}
	sr := newSecurityWireRoutes(t, stub)
	id := "11111111-1111-1111-1111-111111111111"

	rec := securityWireCall(t, sr.HandleDenyDataExport, http.MethodPost, "/x", map[string]any{"note": "kein Rechtsgrund"}, "id", id)
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, "kein Rechtsgrund", stub.gotDenyNote)

	securityWireCall(t, sr.HandleApproveDataExport, http.MethodPost, "/x", map[string]any{"note": "ok"}, "id", id)
	assert.Equal(t, "ok", stub.gotApproveNote)

	// the older name still wins when both are sent
	securityWireCall(t, sr.HandleApproveDataExport, http.MethodPost, "/x", map[string]any{"review_note": "alt", "note": "neu"}, "id", id)
	assert.Equal(t, "alt", stub.gotApproveNote)
}

func TestSecurityWire_CreateIPRuleAnswersFlat(t *testing.T) {
	sr := newSecurityWireRoutes(t, &securityWireStub{})
	rec := securityWireCall(t, sr.HandleCreateIPRule, http.MethodPost, "/ip-rules",
		map[string]any{"ip_cidr": "10.0.0.0/8", "rule_type": "allow"})
	assertStatus(t, rec, http.StatusCreated)
	got := decodeBody(t, rec)
	assert.NotContains(t, got, "rule")
	assert.Equal(t, "10.0.0.0/8", got["ip_cidr"])
	assert.Equal(t, "allow", got["rule_type"])
}

type twoFactorPolicyStub struct {
	authv1.UnimplementedAuthServiceServer
}

func (twoFactorPolicyStub) UpdateTwoFactorPolicy(_ context.Context, req *authv1.UpdateTwoFactorPolicyRequest) (*authv1.UpdateTwoFactorPolicyResponse, error) {
	return &authv1.UpdateTwoFactorPolicyResponse{
		Policy: &authv1.TwoFactorPolicy{RoleName: req.GetRoleName(), Enforced: req.GetEnforced()},
	}, nil
}

func TestSecurityWire_TwoFactorPolicyUpdateAnswersFlat(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer()
	authv1.RegisterAuthServiceServer(srv, twoFactorPolicyStub{})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)
	reg := NewServiceRegistry(nil)
	reg.Register("auth", lis.Addr().String())
	t.Cleanup(reg.Close)
	ar := NewAuthRoutes(reg)

	rec := securityWireCall(t, ar.HandleUpdateTwoFactorPolicy, http.MethodPut, "/auth/2fa/policy",
		map[string]any{"role_name": "admin", "enforced": false, "grace_period_days": 7})
	assertStatus(t, rec, http.StatusOK)
	got := decodeBody(t, rec)
	assert.NotContains(t, got, "policy")
	assert.Equal(t, "admin", got["role_name"])
	assert.Equal(t, false, got["enforced"])
	assert.Contains(t, got, "grace_period_days")
}
