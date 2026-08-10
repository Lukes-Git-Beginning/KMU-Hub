package password

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
)

// mockRepository implements Repository entirely in memory so
// Service.ValidatePassword/CheckPasswordHistory/RecordPassword/GetPolicy/
// UpdatePolicy are testable without a database.
type mockRepository struct {
	policy    *models.PasswordPolicy
	policyErr error
	updateErr error

	history    []string
	historyErr error

	addHistoryErr   error
	addHistoryCalls []string
}

func (m *mockRepository) GetPolicy(_ context.Context, tenantID uuid.UUID) (*models.PasswordPolicy, error) {
	if m.policyErr != nil {
		return nil, m.policyErr
	}
	p := *m.policy
	p.TenantID = tenantID
	return &p, nil
}

func (m *mockRepository) UpdatePolicy(_ context.Context, _ uuid.UUID, policy *models.PasswordPolicy) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.policy = policy
	return nil
}

func (m *mockRepository) AddPasswordHistory(_ context.Context, _ uuid.UUID, passwordHash string) error {
	if m.addHistoryErr != nil {
		return m.addHistoryErr
	}
	m.addHistoryCalls = append(m.addHistoryCalls, passwordHash)
	return nil
}

func (m *mockRepository) GetPasswordHistory(_ context.Context, _ uuid.UUID, _ int) ([]string, error) {
	if m.historyErr != nil {
		return nil, m.historyErr
	}
	return m.history, nil
}

func tenantCtx() context.Context {
	return context.WithValue(context.Background(), middleware.TenantIDKey, uuid.New().String())
}

func containsSubstring(list []string, substr string) bool {
	for _, s := range list {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

// ============================================================================
// ValidatePassword
// ============================================================================

func TestValidatePassword_MinLength(t *testing.T) {
	repo := &mockRepository{policy: &models.PasswordPolicy{MinLength: 20, MinEntropy: 0}}
	svc := NewService(repo)

	valid, failures, err := svc.ValidatePassword(tenantCtx(), "Ab3!Ab3!Ab3!") // 12 chars, wants 20
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Fatal("expected invalid password (too short)")
	}
	if !containsSubstring(failures, "at least 20 characters") {
		t.Fatalf("expected a min-length failure, got %v", failures)
	}
}

func TestValidatePassword_MinLength_Pass(t *testing.T) {
	repo := &mockRepository{policy: &models.PasswordPolicy{MinLength: 8, MinEntropy: 0}}
	svc := NewService(repo)

	valid, failures, err := svc.ValidatePassword(tenantCtx(), "abcdefgh")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valid || len(failures) != 0 {
		t.Fatalf("expected valid password with no failures, got valid=%v failures=%v", valid, failures)
	}
}

func TestValidatePassword_MinEntropy(t *testing.T) {
	repo := &mockRepository{policy: &models.PasswordPolicy{MinLength: 0, MinEntropy: 999999}}
	svc := NewService(repo)

	valid, failures, err := svc.ValidatePassword(tenantCtx(), "abcdefgh12345678")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Fatal("expected invalid password (entropy far below an unreachable minimum)")
	}
	if !containsSubstring(failures, "entropy") {
		t.Fatalf("expected an entropy failure, got %v", failures)
	}
}

func TestValidatePassword_MinEntropy_Pass(t *testing.T) {
	repo := &mockRepository{policy: &models.PasswordPolicy{MinLength: 0, MinEntropy: 0}}
	svc := NewService(repo)

	valid, failures, err := svc.ValidatePassword(tenantCtx(), "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valid || len(failures) != 0 {
		t.Fatalf("expected valid password with a zero entropy floor, got valid=%v failures=%v", valid, failures)
	}
}

func TestValidatePassword_ComplexityRequirements(t *testing.T) {
	tests := []struct {
		name       string
		policy     models.PasswordPolicy
		password   string
		wantValid  bool
		wantSubstr string
	}{
		{"uppercase missing", models.PasswordPolicy{RequireUppercase: true}, "abcdefgh12", false, "uppercase"},
		{"uppercase present", models.PasswordPolicy{RequireUppercase: true}, "Abcdefgh12", true, ""},
		{"lowercase missing", models.PasswordPolicy{RequireLowercase: true}, "ABCDEFGH12", false, "lowercase"},
		{"lowercase present", models.PasswordPolicy{RequireLowercase: true}, "ABCDEFGh12", true, ""},
		{"digit missing", models.PasswordPolicy{RequireDigit: true}, "abcdefghij", false, "digit"},
		{"digit present", models.PasswordPolicy{RequireDigit: true}, "abcdefgh1j", true, ""},
		{"special missing", models.PasswordPolicy{RequireSpecial: true}, "abcdefgh12", false, "special character"},
		{"special present", models.PasswordPolicy{RequireSpecial: true}, "abcdefgh1!", true, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockRepository{policy: &tc.policy}
			svc := NewService(repo)

			valid, failures, err := svc.ValidatePassword(tenantCtx(), tc.password)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if valid != tc.wantValid {
				t.Fatalf("valid = %v, want %v (failures: %v)", valid, tc.wantValid, failures)
			}
			if tc.wantSubstr != "" && !containsSubstring(failures, tc.wantSubstr) {
				t.Fatalf("expected a failure containing %q, got %v", tc.wantSubstr, failures)
			}
		})
	}
}

func TestValidatePassword_AllRequirementsCombined(t *testing.T) {
	repo := &mockRepository{policy: &models.PasswordPolicy{
		MinLength:        8,
		MinEntropy:       0,
		RequireUppercase: true,
		RequireLowercase: true,
		RequireDigit:     true,
		RequireSpecial:   true,
	}}
	svc := NewService(repo)

	valid, failures, err := svc.ValidatePassword(tenantCtx(), "Abcdef1!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valid || len(failures) != 0 {
		t.Fatalf("expected a fully compliant password to pass, got valid=%v failures=%v", valid, failures)
	}
}

func TestValidatePassword_MissingTenant(t *testing.T) {
	repo := &mockRepository{policy: &models.PasswordPolicy{}}
	svc := NewService(repo)

	_, _, err := svc.ValidatePassword(context.Background(), "whatever")
	if err == nil || !strings.Contains(err.Error(), "tenant id missing") {
		t.Fatalf("expected a tenant-missing error, got %v", err)
	}
}

func TestValidatePassword_RepoError(t *testing.T) {
	repo := &mockRepository{policyErr: errors.New("db down")}
	svc := NewService(repo)

	_, _, err := svc.ValidatePassword(tenantCtx(), "whatever")
	if err == nil || !strings.Contains(err.Error(), "failed to load policy") {
		t.Fatalf("expected a load-policy error, got %v", err)
	}
}

// ============================================================================
// CheckPasswordHistory
// ============================================================================

func TestCheckPasswordHistory_ReuseDisabled_ShortCircuits(t *testing.T) {
	// historyErr proves GetPasswordHistory is never reached when
	// PreventReuseCount <= 0 -- a real query would surface historyErr.
	repo := &mockRepository{
		policy:     &models.PasswordPolicy{PreventReuseCount: 0},
		historyErr: errors.New("GetPasswordHistory should not be called"),
	}
	svc := NewService(repo)

	safe, err := svc.CheckPasswordHistory(tenantCtx(), uuid.New(), "new-password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !safe {
		t.Fatal("expected safe=true when reuse prevention is disabled")
	}
}

func TestCheckPasswordHistory_ReuseDetected(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("reused-pw"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt.GenerateFromPassword: %v", err)
	}
	repo := &mockRepository{
		policy:  &models.PasswordPolicy{PreventReuseCount: 5},
		history: []string{string(hash)},
	}
	svc := NewService(repo)

	safe, err := svc.CheckPasswordHistory(tenantCtx(), uuid.New(), "reused-pw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if safe {
		t.Fatal("expected safe=false for a password matching history")
	}
}

func TestCheckPasswordHistory_NoMatch(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("some-other-pw"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt.GenerateFromPassword: %v", err)
	}
	repo := &mockRepository{
		policy:  &models.PasswordPolicy{PreventReuseCount: 5},
		history: []string{string(hash)},
	}
	svc := NewService(repo)

	safe, err := svc.CheckPasswordHistory(tenantCtx(), uuid.New(), "brand-new-pw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !safe {
		t.Fatal("expected safe=true for a password not present in history")
	}
}

func TestCheckPasswordHistory_MissingTenant(t *testing.T) {
	repo := &mockRepository{policy: &models.PasswordPolicy{}}
	svc := NewService(repo)

	_, err := svc.CheckPasswordHistory(context.Background(), uuid.New(), "x")
	if err == nil || !strings.Contains(err.Error(), "tenant id missing") {
		t.Fatalf("expected a tenant-missing error, got %v", err)
	}
}

func TestCheckPasswordHistory_PolicyRepoError(t *testing.T) {
	repo := &mockRepository{policyErr: errors.New("db down")}
	svc := NewService(repo)

	_, err := svc.CheckPasswordHistory(tenantCtx(), uuid.New(), "x")
	if err == nil || !strings.Contains(err.Error(), "failed to load policy") {
		t.Fatalf("expected a load-policy error, got %v", err)
	}
}

func TestCheckPasswordHistory_HistoryRepoError(t *testing.T) {
	repo := &mockRepository{
		policy:     &models.PasswordPolicy{PreventReuseCount: 5},
		historyErr: errors.New("query timeout"),
	}
	svc := NewService(repo)

	_, err := svc.CheckPasswordHistory(tenantCtx(), uuid.New(), "x")
	if err == nil || !strings.Contains(err.Error(), "failed to load history") {
		t.Fatalf("expected a load-history error, got %v", err)
	}
}

// ============================================================================
// RecordPassword
// ============================================================================

func TestRecordPassword_Success(t *testing.T) {
	repo := &mockRepository{}
	svc := NewService(repo)

	if err := svc.RecordPassword(context.Background(), uuid.New(), "some-hash"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.addHistoryCalls) != 1 || repo.addHistoryCalls[0] != "some-hash" {
		t.Fatalf("expected AddPasswordHistory to be called with the hash, got %v", repo.addHistoryCalls)
	}
}

func TestRecordPassword_RepoError(t *testing.T) {
	repo := &mockRepository{addHistoryErr: errors.New("insert failed")}
	svc := NewService(repo)

	err := svc.RecordPassword(context.Background(), uuid.New(), "some-hash")
	if err == nil || !strings.Contains(err.Error(), "failed to record history") {
		t.Fatalf("expected a record-history error, got %v", err)
	}
}

// ============================================================================
// GetPolicy / UpdatePolicy
// ============================================================================

func TestGetPolicy_MissingTenant(t *testing.T) {
	repo := &mockRepository{policy: &models.PasswordPolicy{}}
	svc := NewService(repo)

	_, err := svc.GetPolicy(context.Background())
	if err == nil || !strings.Contains(err.Error(), "tenant id missing") {
		t.Fatalf("expected a tenant-missing error, got %v", err)
	}
}

func TestGetPolicy_ReturnsRepoPolicy(t *testing.T) {
	repo := &mockRepository{policy: &models.PasswordPolicy{MinLength: 16}}
	svc := NewService(repo)

	got, err := svc.GetPolicy(tenantCtx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.MinLength != 16 {
		t.Fatalf("expected the repo's policy to be returned unmodified, got MinLength=%d", got.MinLength)
	}
}

func TestUpdatePolicy_MissingTenant(t *testing.T) {
	repo := &mockRepository{policy: &models.PasswordPolicy{}}
	svc := NewService(repo)

	err := svc.UpdatePolicy(context.Background(), &models.PasswordPolicy{}, uuid.New())
	if err == nil || !strings.Contains(err.Error(), "tenant id missing") {
		t.Fatalf("expected a tenant-missing error, got %v", err)
	}
}

func TestUpdatePolicy_CurrentLookupError(t *testing.T) {
	repo := &mockRepository{policyErr: errors.New("db down")}
	svc := NewService(repo)

	err := svc.UpdatePolicy(tenantCtx(), &models.PasswordPolicy{}, uuid.New())
	if err == nil || !strings.Contains(err.Error(), "failed to load current policy") {
		t.Fatalf("expected a load-current-policy error, got %v", err)
	}
}

func TestUpdatePolicy_ResolvesIDServerSide(t *testing.T) {
	existingID := uuid.New()
	repo := &mockRepository{policy: &models.PasswordPolicy{ID: existingID}}
	svc := NewService(repo)

	ctx := tenantCtx()
	tenantID, _ := middleware.GetTenantID(ctx)
	updatedBy := uuid.New()

	// A caller-supplied ID pointing at a different row must be ignored --
	// UpdatePolicy always resolves the row server-side via GetPolicy.
	spoofedID := uuid.New()
	err := svc.UpdatePolicy(ctx, &models.PasswordPolicy{ID: spoofedID, MinLength: 20}, updatedBy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.policy.ID != existingID {
		t.Fatalf("expected the server-resolved ID %s to be used, got %s", existingID, repo.policy.ID)
	}
	if repo.policy.TenantID != tenantID {
		t.Fatalf("expected TenantID=%s, got %s", tenantID, repo.policy.TenantID)
	}
	if repo.policy.UpdatedBy == nil || *repo.policy.UpdatedBy != updatedBy {
		t.Fatalf("expected UpdatedBy=%s, got %v", updatedBy, repo.policy.UpdatedBy)
	}
	if repo.policy.MinLength != 20 {
		t.Fatalf("expected MinLength=20 to pass through, got %d", repo.policy.MinLength)
	}
}

func TestUpdatePolicy_RepoError(t *testing.T) {
	repo := &mockRepository{
		policy:    &models.PasswordPolicy{ID: uuid.New()},
		updateErr: errors.New("update failed"),
	}
	svc := NewService(repo)

	err := svc.UpdatePolicy(tenantCtx(), &models.PasswordPolicy{}, uuid.New())
	if err == nil || !strings.Contains(err.Error(), "failed to update policy") {
		t.Fatalf("expected an update-policy error, got %v", err)
	}
}
