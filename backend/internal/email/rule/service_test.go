package rule

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ---------------------------------------------------------------------------
// Mock repository
// ---------------------------------------------------------------------------

type mockRepo struct {
	rules      []*models.EmailRule
	candidates []*models.EmailRuleCandidate
	folders    map[uuid.UUID]uuid.UUID // folder id -> tenant id
	labels     map[uuid.UUID]uuid.UUID // label id -> tenant id
	writes     int
}

func newMockRepo() *mockRepo {
	return &mockRepo{folders: map[uuid.UUID]uuid.UUID{}, labels: map[uuid.UUID]uuid.UUID{}}
}

func (m *mockRepo) Create(_ context.Context, r *models.EmailRule) error {
	m.rules = append(m.rules, r)
	return nil
}

func (m *mockRepo) GetByID(_ context.Context, id, tenantID uuid.UUID) (*models.EmailRule, error) {
	for _, r := range m.rules {
		if r.ID == id && r.TenantID == tenantID {
			return r, nil
		}
	}
	return nil, ErrRuleNotFound
}

func (m *mockRepo) List(_ context.Context, tenantID uuid.UUID) ([]*models.EmailRule, error) {
	out := make([]*models.EmailRule, 0)
	for _, r := range m.rules {
		if r.TenantID == tenantID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *mockRepo) Update(_ context.Context, r *models.EmailRule) error {
	for i, existing := range m.rules {
		if existing.ID == r.ID && existing.TenantID == r.TenantID {
			m.rules[i] = r
			return nil
		}
	}
	return ErrRuleNotFound
}

func (m *mockRepo) Delete(_ context.Context, id, tenantID uuid.UUID) error {
	for i, r := range m.rules {
		if r.ID == id && r.TenantID == tenantID {
			m.rules = slices.Delete(m.rules, i, i+1)
			return nil
		}
	}
	return ErrRuleNotFound
}

func (m *mockRepo) FolderBelongsToTenant(_ context.Context, folderID, tenantID uuid.UUID) (bool, error) {
	owner, ok := m.folders[folderID]
	return ok && owner == tenantID, nil
}

func (m *mockRepo) LabelBelongsToTenant(_ context.Context, labelID, tenantID uuid.UUID) (bool, error) {
	owner, ok := m.labels[labelID]
	return ok && owner == tenantID, nil
}

func (m *mockRepo) ListApplyCandidates(_ context.Context, _ uuid.UUID, limit int) ([]*models.EmailRuleCandidate, error) {
	if len(m.candidates) > limit {
		return m.candidates[:limit], nil
	}
	return m.candidates, nil
}

func (m *mockRepo) ApplyToMessage(_ context.Context, _, messageID, folderID uuid.UUID, labelIDs []uuid.UUID) error {
	m.writes++
	for _, c := range m.candidates {
		if c.ID == messageID {
			c.FolderID = folderID
			c.LabelIDs = labelIDs
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func ptr(s string) *string { return &s }

func validInput(target uuid.UUID) RuleInput {
	return RuleInput{
		Name:         ptr("Rechnungen markieren"),
		Field:        ptr(models.EmailRuleFieldSubject),
		Op:           ptr(models.EmailRuleOpContains),
		Value:        ptr("Rechnung"),
		ActionType:   ptr(models.EmailRuleActionLabel),
		ActionTarget: ptr(target.String()),
	}
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

func TestCreate_Valid(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := NewService(repo)
	tenant := uuid.New()
	label := uuid.New()
	repo.labels[label] = tenant

	r, err := svc.Create(context.Background(), tenant, validInput(label))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if r.TenantID != tenant {
		t.Errorf("tenant_id = %s, want %s", r.TenantID, tenant)
	}
	if r.ActionTarget != label {
		t.Errorf("action_target = %s, want %s", r.ActionTarget, label)
	}
	if r.Op != models.EmailRuleOpContains {
		t.Errorf("op = %q, want %q", r.Op, models.EmailRuleOpContains)
	}
}

func TestCreate_Rejects(t *testing.T) {
	t.Parallel()
	target := uuid.New()

	cases := []struct {
		name  string
		mutet func(*RuleInput)
		want  error
	}{
		{"blank name", func(in *RuleInput) { in.Name = ptr("   ") }, ErrNameRequired},
		{"unknown field", func(in *RuleInput) { in.Field = ptr("body") }, ErrInvalidField},
		{"unknown operator", func(in *RuleInput) { in.Op = ptr("matches") }, ErrInvalidOperator},
		{"blank value", func(in *RuleInput) { in.Value = ptr("  ") }, ErrValueRequired},
		{"unknown action", func(in *RuleInput) { in.ActionType = ptr("forward") }, ErrInvalidActionType},
		{"target not a uuid", func(in *RuleInput) { in.ActionTarget = ptr("lbl-rechnung") }, ErrTargetRequired},
		{"target nil uuid", func(in *RuleInput) { in.ActionTarget = ptr(uuid.Nil.String()) }, ErrTargetRequired},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			svc := NewService(newMockRepo())
			in := validInput(target)
			tc.mutet(&in)

			if _, err := svc.Create(context.Background(), uuid.New(), in); !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

// A move rule must not be able to push messages into another tenant's folder:
// the message keeps its own tenant_id, so RLS on email_messages would not
// notice the folder switch.
func TestCreate_MoveTargetMustBeOwnFolder(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := NewService(repo)
	tenant, other := uuid.New(), uuid.New()
	foreignFolder := uuid.New()
	repo.folders[foreignFolder] = other

	in := validInput(foreignFolder)
	in.ActionType = ptr(models.EmailRuleActionMove)

	if _, err := svc.Create(context.Background(), tenant, in); !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrTargetNotFound)
	}

	repo.folders[foreignFolder] = tenant
	if _, err := svc.Create(context.Background(), tenant, in); err != nil {
		t.Fatalf("own folder rejected: %v", err)
	}
}

// A label rule must not be able to stamp a foreign or nonexistent label id:
// label_ids has no foreign key, so an unvalidated target would silently
// corrupt the message's label set.
func TestCreate_LabelTargetMustBeOwnLabel(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := NewService(repo)
	tenant, other := uuid.New(), uuid.New()
	foreignLabel := uuid.New()
	repo.labels[foreignLabel] = other

	in := validInput(foreignLabel)

	if _, err := svc.Create(context.Background(), tenant, in); !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrTargetNotFound)
	}

	repo.labels[foreignLabel] = tenant
	if _, err := svc.Create(context.Background(), tenant, in); err != nil {
		t.Fatalf("own label rejected: %v", err)
	}
}

func TestUpdate_PartialKeepsUntouchedFields(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := NewService(repo)
	tenant, label := uuid.New(), uuid.New()
	repo.labels[label] = tenant

	created, err := svc.Create(context.Background(), tenant, validInput(label))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	updated, err := svc.Update(context.Background(), created.ID, tenant, RuleInput{Value: ptr("Mahnung")})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Value != "Mahnung" {
		t.Errorf("value = %q, want %q", updated.Value, "Mahnung")
	}
	if updated.Name != created.Name || updated.Field != created.Field || updated.ActionTarget != label {
		t.Errorf("untouched fields changed: %+v", updated)
	}
}

// Switching action_type to "move" without moving the target has to be rejected,
// because the label id it still carries is not a folder of this tenant.
func TestUpdate_ValidatesMergedRuleNotOnlyThePatch(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := NewService(repo)
	tenant, label := uuid.New(), uuid.New()
	repo.labels[label] = tenant

	created, err := svc.Create(context.Background(), tenant, validInput(label))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	_, err = svc.Update(context.Background(), created.ID, tenant, RuleInput{
		ActionType: ptr(models.EmailRuleActionMove),
	})
	if !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrTargetNotFound)
	}
}

func TestUpdate_ForeignTenantIsNotFound(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := NewService(repo)
	tenant, label := uuid.New(), uuid.New()
	repo.labels[label] = tenant

	created, err := svc.Create(context.Background(), tenant, validInput(label))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := svc.Update(context.Background(), created.ID, uuid.New(), RuleInput{Value: ptr("x")}); !errors.Is(err, ErrRuleNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrRuleNotFound)
	}
	if err := svc.Delete(context.Background(), created.ID, uuid.New()); !errors.Is(err, ErrRuleNotFound) {
		t.Fatalf("delete err = %v, want %v", err, ErrRuleNotFound)
	}
}

// ---------------------------------------------------------------------------
// ApplyAll
// ---------------------------------------------------------------------------

func TestApplyAll_NoRulesTouchesNothing(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	repo.candidates = []*models.EmailRuleCandidate{{ID: uuid.New(), Subject: "Rechnung 5"}}
	svc := NewService(repo)

	res, err := svc.ApplyAll(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("ApplyAll: %v", err)
	}
	if res.Affected != 0 || res.Scanned != 0 || repo.writes != 0 {
		t.Fatalf("res = %+v, writes = %d, want a no-op", res, repo.writes)
	}
}

func TestApplyAll_LabelAndMoveAndIdempotence(t *testing.T) {
	t.Parallel()
	tenant := uuid.New()
	label, inbox, archive := uuid.New(), uuid.New(), uuid.New()

	repo := newMockRepo()
	repo.folders[archive] = tenant
	repo.rules = []*models.EmailRule{
		{ID: uuid.New(), TenantID: tenant, Name: "label", Field: models.EmailRuleFieldSubject,
			Op: models.EmailRuleOpContains, Value: "rechnung",
			ActionType: models.EmailRuleActionLabel, ActionTarget: label},
		{ID: uuid.New(), TenantID: tenant, Name: "move", Field: models.EmailRuleFieldFrom,
			Op: models.EmailRuleOpContains, Value: "buchhaltung@",
			ActionType: models.EmailRuleActionMove, ActionTarget: archive},
	}

	matched := uuid.New()
	repo.candidates = []*models.EmailRuleCandidate{
		// Subject match is case-insensitive; the from match reads the address.
		{ID: matched, Subject: "Ihre RECHNUNG 2026-04", FromName: "Buchhaltung",
			FromEmail: "buchhaltung@example.de", FolderID: inbox},
		{ID: uuid.New(), Subject: "Mittagessen?", FromName: "Ben", FromEmail: "ben@example.de", FolderID: inbox},
	}
	svc := NewService(repo)

	res, err := svc.ApplyAll(context.Background(), tenant)
	if err != nil {
		t.Fatalf("ApplyAll: %v", err)
	}
	if res.Affected != 1 {
		t.Errorf("affected = %d, want 1", res.Affected)
	}
	if res.Scanned != 2 {
		t.Errorf("scanned = %d, want 2", res.Scanned)
	}

	got := repo.candidates[0]
	if got.ID != matched {
		t.Fatalf("candidate order changed")
	}
	if !slices.Contains(got.LabelIDs, label) {
		t.Errorf("label not applied: %v", got.LabelIDs)
	}
	if got.FolderID != archive {
		t.Errorf("folder = %s, want %s", got.FolderID, archive)
	}
	if repo.candidates[1].FolderID != inbox || len(repo.candidates[1].LabelIDs) != 0 {
		t.Errorf("non-matching message was touched: %+v", repo.candidates[1])
	}

	// Second run: everything already in place, so nothing may be written again.
	writesAfterFirst := repo.writes
	res2, err := svc.ApplyAll(context.Background(), tenant)
	if err != nil {
		t.Fatalf("second ApplyAll: %v", err)
	}
	if res2.Affected != 0 {
		t.Errorf("second run affected = %d, want 0", res2.Affected)
	}
	if repo.writes != writesAfterFirst {
		t.Errorf("second run wrote %d extra times, want 0", repo.writes-writesAfterFirst)
	}
}

// A message hit by two label rules collects both, and the from haystack covers
// the display name as well as the address.
func TestApplyAll_LabelsAccumulateAndFromMatchesDisplayName(t *testing.T) {
	t.Parallel()
	tenant := uuid.New()
	labelA, labelB := uuid.New(), uuid.New()

	repo := newMockRepo()
	repo.rules = []*models.EmailRule{
		{ID: uuid.New(), TenantID: tenant, Name: "a", Field: models.EmailRuleFieldFrom,
			Op: models.EmailRuleOpContains, Value: "Musterfirma",
			ActionType: models.EmailRuleActionLabel, ActionTarget: labelA},
		{ID: uuid.New(), TenantID: tenant, Name: "b", Field: models.EmailRuleFieldSubject,
			Op: models.EmailRuleOpContains, Value: "angebot",
			ActionType: models.EmailRuleActionLabel, ActionTarget: labelB},
	}
	repo.candidates = []*models.EmailRuleCandidate{
		{ID: uuid.New(), Subject: "Angebot 42", FromName: "Musterfirma GmbH", FromEmail: "info@example.de"},
	}
	svc := NewService(repo)

	res, err := svc.ApplyAll(context.Background(), tenant)
	if err != nil {
		t.Fatalf("ApplyAll: %v", err)
	}
	if res.Affected != 1 {
		t.Fatalf("affected = %d, want 1", res.Affected)
	}
	got := repo.candidates[0].LabelIDs
	if len(got) != 2 || !slices.Contains(got, labelA) || !slices.Contains(got, labelB) {
		t.Errorf("labels = %v, want both %s and %s", got, labelA, labelB)
	}
}

// Later rules win on a move; that is the frontend's demo behaviour and the
// order the repository returns (creation order).
func TestApplyAll_LastMoveWins(t *testing.T) {
	t.Parallel()
	tenant := uuid.New()
	first, second := uuid.New(), uuid.New()

	repo := newMockRepo()
	repo.rules = []*models.EmailRule{
		{ID: uuid.New(), TenantID: tenant, Name: "1", Field: models.EmailRuleFieldSubject,
			Op: models.EmailRuleOpContains, Value: "x",
			ActionType: models.EmailRuleActionMove, ActionTarget: first},
		{ID: uuid.New(), TenantID: tenant, Name: "2", Field: models.EmailRuleFieldSubject,
			Op: models.EmailRuleOpContains, Value: "x",
			ActionType: models.EmailRuleActionMove, ActionTarget: second},
	}
	repo.candidates = []*models.EmailRuleCandidate{{ID: uuid.New(), Subject: "xyz", FolderID: uuid.New()}}
	svc := NewService(repo)

	if _, err := svc.ApplyAll(context.Background(), tenant); err != nil {
		t.Fatalf("ApplyAll: %v", err)
	}
	if repo.candidates[0].FolderID != second {
		t.Errorf("folder = %s, want the later rule's target %s", repo.candidates[0].FolderID, second)
	}
}

func TestApplyAll_RespectsScanLimit(t *testing.T) {
	t.Parallel()
	tenant := uuid.New()
	label := uuid.New()

	repo := newMockRepo()
	repo.rules = []*models.EmailRule{{
		ID: uuid.New(), TenantID: tenant, Name: "all", Field: models.EmailRuleFieldSubject,
		Op: models.EmailRuleOpContains, Value: "a",
		ActionType: models.EmailRuleActionLabel, ActionTarget: label,
	}}
	for range applyScanLimit + 10 {
		repo.candidates = append(repo.candidates, &models.EmailRuleCandidate{ID: uuid.New(), Subject: "aaa"})
	}
	svc := NewService(repo)

	res, err := svc.ApplyAll(context.Background(), tenant)
	if err != nil {
		t.Fatalf("ApplyAll: %v", err)
	}
	if res.Scanned != applyScanLimit {
		t.Errorf("scanned = %d, want the cap %d", res.Scanned, applyScanLimit)
	}
}
