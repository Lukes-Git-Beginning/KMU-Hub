package server

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/email/label"
	"github.com/kmuhub/kmuhub/internal/email/rule"
	"github.com/kmuhub/kmuhub/internal/email/signature"
	"github.com/kmuhub/kmuhub/internal/email/template"
	"github.com/kmuhub/kmuhub/internal/models"
	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
)

// ---------------------------------------------------------------------------
// stubSignatureRepo implements signature.Repository over an in-memory map.
// ---------------------------------------------------------------------------

type stubSignatureRepo struct {
	sigs map[uuid.UUID]*models.EmailSignature
}

func newStubSignatureRepo() *stubSignatureRepo {
	return &stubSignatureRepo{sigs: make(map[uuid.UUID]*models.EmailSignature)}
}

func (r *stubSignatureRepo) Create(_ context.Context, sig *models.EmailSignature) error {
	r.sigs[sig.ID] = sig
	return nil
}

func (r *stubSignatureRepo) GetByID(_ context.Context, id, tenantID uuid.UUID) (*models.EmailSignature, error) {
	s, ok := r.sigs[id]
	if !ok || s.TenantID != tenantID {
		return nil, signature.ErrSignatureNotFound
	}
	return s, nil
}

func (r *stubSignatureRepo) GetDefault(_ context.Context, userID, tenantID uuid.UUID) (*models.EmailSignature, error) {
	for _, s := range r.sigs {
		if s.UserID == userID && s.TenantID == tenantID && s.IsDefault {
			return s, nil
		}
	}
	return nil, signature.ErrSignatureNotFound
}

func (r *stubSignatureRepo) ListByUser(_ context.Context, userID, tenantID uuid.UUID) ([]*models.EmailSignature, error) {
	out := make([]*models.EmailSignature, 0)
	for _, s := range r.sigs {
		if s.UserID == userID && s.TenantID == tenantID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (r *stubSignatureRepo) Update(_ context.Context, sig *models.EmailSignature) error {
	if _, ok := r.sigs[sig.ID]; !ok {
		return signature.ErrSignatureNotFound
	}
	r.sigs[sig.ID] = sig
	return nil
}

func (r *stubSignatureRepo) Delete(_ context.Context, id, tenantID uuid.UUID) error {
	s, ok := r.sigs[id]
	if !ok || s.TenantID != tenantID {
		return signature.ErrSignatureNotFound
	}
	delete(r.sigs, id)
	return nil
}

func (r *stubSignatureRepo) ClearDefaultForUser(_ context.Context, userID, tenantID uuid.UUID) error {
	for _, s := range r.sigs {
		if s.UserID == userID && s.TenantID == tenantID {
			s.IsDefault = false
		}
	}
	return nil
}

func (r *stubSignatureRepo) CountByUser(_ context.Context, userID, tenantID uuid.UUID) (int, error) {
	count := 0
	for _, s := range r.sigs {
		if s.UserID == userID && s.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

func seedSignature(repo *stubSignatureRepo, tenantID, userID uuid.UUID, name string, isDefault bool) *models.EmailSignature {
	now := time.Now().UTC()
	s := &models.EmailSignature{
		ID: uuid.New(), TenantID: tenantID, UserID: userID, Name: name, HTMLContent: "<p>Hi</p>",
		IsDefault: isDefault, CreatedAt: now, UpdatedAt: now,
	}
	repo.sigs[s.ID] = s
	return s
}

// ---------------------------------------------------------------------------
// stubRuleRepo implements rule.Repository over in-memory maps. folders/labels
// record which tenant an id belongs to, mirroring the real repository's
// membership check (a foreign id must not validate).
// ---------------------------------------------------------------------------

type stubRuleRepo struct {
	rules      map[uuid.UUID]*models.EmailRule
	folders    map[uuid.UUID]uuid.UUID // folder id -> owning tenant
	labels     map[uuid.UUID]uuid.UUID // label id -> owning tenant
	candidates []*models.EmailRuleCandidate
	writes     int
}

func newStubRuleRepo() *stubRuleRepo {
	return &stubRuleRepo{
		rules:   make(map[uuid.UUID]*models.EmailRule),
		folders: make(map[uuid.UUID]uuid.UUID),
		labels:  make(map[uuid.UUID]uuid.UUID),
	}
}

func (r *stubRuleRepo) Create(_ context.Context, rl *models.EmailRule) error {
	r.rules[rl.ID] = rl
	return nil
}

func (r *stubRuleRepo) GetByID(_ context.Context, id, tenantID uuid.UUID) (*models.EmailRule, error) {
	rl, ok := r.rules[id]
	if !ok || rl.TenantID != tenantID {
		return nil, rule.ErrRuleNotFound
	}
	return rl, nil
}

func (r *stubRuleRepo) List(_ context.Context, tenantID uuid.UUID) ([]*models.EmailRule, error) {
	out := make([]*models.EmailRule, 0)
	for _, rl := range r.rules {
		if rl.TenantID == tenantID {
			out = append(out, rl)
		}
	}
	return out, nil
}

func (r *stubRuleRepo) Update(_ context.Context, rl *models.EmailRule) error {
	if _, ok := r.rules[rl.ID]; !ok {
		return rule.ErrRuleNotFound
	}
	r.rules[rl.ID] = rl
	return nil
}

func (r *stubRuleRepo) Delete(_ context.Context, id, tenantID uuid.UUID) error {
	rl, ok := r.rules[id]
	if !ok || rl.TenantID != tenantID {
		return rule.ErrRuleNotFound
	}
	delete(r.rules, id)
	return nil
}

func (r *stubRuleRepo) FolderBelongsToTenant(_ context.Context, folderID, tenantID uuid.UUID) (bool, error) {
	owner, ok := r.folders[folderID]
	return ok && owner == tenantID, nil
}

func (r *stubRuleRepo) LabelBelongsToTenant(_ context.Context, labelID, tenantID uuid.UUID) (bool, error) {
	owner, ok := r.labels[labelID]
	return ok && owner == tenantID, nil
}

func (r *stubRuleRepo) ListApplyCandidates(_ context.Context, _ uuid.UUID, limit int) ([]*models.EmailRuleCandidate, error) {
	if len(r.candidates) > limit {
		return r.candidates[:limit], nil
	}
	return r.candidates, nil
}

func (r *stubRuleRepo) ApplyToMessage(_ context.Context, _, messageID, folderID uuid.UUID, labelIDs []uuid.UUID) error {
	r.writes++
	for _, c := range r.candidates {
		if c.ID == messageID {
			c.FolderID = folderID
			c.LabelIDs = labelIDs
		}
	}
	return nil
}

func seedRule(repo *stubRuleRepo, tenantID uuid.UUID, field, value, actionType string, target uuid.UUID) *models.EmailRule {
	now := time.Now().UTC()
	rl := &models.EmailRule{
		ID: uuid.New(), TenantID: tenantID, Name: "Regel", Field: field,
		Op: models.EmailRuleOpContains, Value: value, ActionType: actionType, ActionTarget: target,
		CreatedAt: now, UpdatedAt: now,
	}
	repo.rules[rl.ID] = rl
	return rl
}

// ---------------------------------------------------------------------------
// stubLabelRepo implements label.Repository over an in-memory map.
// ---------------------------------------------------------------------------

type stubLabelRepo struct {
	labels   map[uuid.UUID]*models.EmailLabel
	assigned map[uuid.UUID][]uuid.UUID // messageID -> labelIDs, last AssignToMessage call
}

func newStubLabelRepo() *stubLabelRepo {
	return &stubLabelRepo{
		labels:   make(map[uuid.UUID]*models.EmailLabel),
		assigned: make(map[uuid.UUID][]uuid.UUID),
	}
}

func (r *stubLabelRepo) Create(_ context.Context, l *models.EmailLabel) error {
	r.labels[l.ID] = l
	return nil
}

func (r *stubLabelRepo) GetByID(_ context.Context, id, tenantID uuid.UUID) (*models.EmailLabel, error) {
	l, ok := r.labels[id]
	if !ok || l.TenantID != tenantID {
		return nil, label.ErrNotFound
	}
	return l, nil
}

func (r *stubLabelRepo) List(_ context.Context, tenantID uuid.UUID) ([]*models.EmailLabel, error) {
	out := make([]*models.EmailLabel, 0)
	for _, l := range r.labels {
		if l.TenantID == tenantID {
			out = append(out, l)
		}
	}
	return out, nil
}

func (r *stubLabelRepo) Update(_ context.Context, l *models.EmailLabel) error {
	if _, ok := r.labels[l.ID]; !ok {
		return label.ErrNotFound
	}
	r.labels[l.ID] = l
	return nil
}

func (r *stubLabelRepo) Delete(_ context.Context, id, tenantID uuid.UUID) error {
	l, ok := r.labels[id]
	if !ok || l.TenantID != tenantID {
		return label.ErrNotFound
	}
	delete(r.labels, id)
	return nil
}

func (r *stubLabelRepo) LabelIDsBelongToTenant(_ context.Context, tenantID uuid.UUID, ids []uuid.UUID) (bool, error) {
	for _, id := range ids {
		l, ok := r.labels[id]
		if !ok || l.TenantID != tenantID {
			return false, nil
		}
	}
	return true, nil
}

func (r *stubLabelRepo) AssignToMessage(_ context.Context, _, messageID uuid.UUID, labelIDs []uuid.UUID) error {
	r.assigned[messageID] = labelIDs
	return nil
}

func seedLabel(repo *stubLabelRepo, tenantID uuid.UUID, name, color string) *models.EmailLabel {
	now := time.Now().UTC()
	l := &models.EmailLabel{ID: uuid.New(), TenantID: tenantID, Name: name, Color: color, CreatedAt: now, UpdatedAt: now}
	repo.labels[l.ID] = l
	return l
}

// stubLabelMessageReader implements label.MessageReader over an in-memory map
// of pre-seeded messages -- AssignMessageLabels only needs a message to exist
// after the assignment call, it does not need it to reflect the just-written
// label set (that is the repository's job, asserted separately via
// stubLabelRepo.assigned).
type stubLabelMessageReader struct {
	messages map[uuid.UUID]*models.EmailMessage
}

func newStubLabelMessageReader() *stubLabelMessageReader {
	return &stubLabelMessageReader{messages: make(map[uuid.UUID]*models.EmailMessage)}
}

func (r *stubLabelMessageReader) GetByID(_ context.Context, id, tenantID uuid.UUID) (*models.EmailMessage, error) {
	m, ok := r.messages[id]
	if !ok || m.TenantID != tenantID {
		return nil, label.ErrMessageNotFound
	}
	return m, nil
}

func seedLabelableMessage(reader *stubLabelMessageReader, tenantID uuid.UUID) *models.EmailMessage {
	now := time.Now().UTC()
	m := &models.EmailMessage{
		ID: uuid.New(), TenantID: tenantID, AccountID: uuid.New(), FolderID: uuid.New(),
		FromName: "Sender", FromEmail: "sender@example.com", Subject: "Hallo",
		Date: now, CreatedAt: now, UpdatedAt: now,
	}
	reader.messages[m.ID] = m
	return m
}

// ---------------------------------------------------------------------------
// stubTemplateRepo implements template.Repository over an in-memory map.
// ---------------------------------------------------------------------------

type stubTemplateRepo struct {
	templates map[uuid.UUID]*models.EmailTemplate
}

func newStubTemplateRepo() *stubTemplateRepo {
	return &stubTemplateRepo{templates: make(map[uuid.UUID]*models.EmailTemplate)}
}

func (r *stubTemplateRepo) Create(_ context.Context, tpl *models.EmailTemplate) error {
	r.templates[tpl.ID] = tpl
	return nil
}

func (r *stubTemplateRepo) visible(tpl *models.EmailTemplate, userID uuid.UUID, isAdmin bool) bool {
	if tpl.Visibility == template.VisibilityShared || isAdmin {
		return true
	}
	return tpl.OwnerID != nil && *tpl.OwnerID == userID
}

func (r *stubTemplateRepo) GetByID(_ context.Context, id, tenantID, userID uuid.UUID, isAdmin bool) (*models.EmailTemplate, error) {
	tpl, ok := r.templates[id]
	if !ok || tpl.TenantID != tenantID || !r.visible(tpl, userID, isAdmin) {
		return nil, template.ErrTemplateNotFound
	}
	return tpl, nil
}

func (r *stubTemplateRepo) ListVisible(_ context.Context, tenantID, userID uuid.UUID, isAdmin bool) ([]*models.EmailTemplate, error) {
	out := make([]*models.EmailTemplate, 0)
	for _, tpl := range r.templates {
		if tpl.TenantID == tenantID && r.visible(tpl, userID, isAdmin) {
			out = append(out, tpl)
		}
	}
	return out, nil
}

func (r *stubTemplateRepo) Update(_ context.Context, tpl *models.EmailTemplate) error {
	if _, ok := r.templates[tpl.ID]; !ok {
		return template.ErrTemplateNotFound
	}
	r.templates[tpl.ID] = tpl
	return nil
}

func (r *stubTemplateRepo) Delete(_ context.Context, id, tenantID uuid.UUID) error {
	tpl, ok := r.templates[id]
	if !ok || tpl.TenantID != tenantID {
		return template.ErrTemplateNotFound
	}
	delete(r.templates, id)
	return nil
}

func seedTemplate(repo *stubTemplateRepo, tenantID uuid.UUID, owner *uuid.UUID, visibility, subject string) *models.EmailTemplate {
	now := time.Now().UTC()
	tpl := &models.EmailTemplate{
		ID: uuid.New(), TenantID: tenantID, OwnerID: owner, Visibility: visibility,
		Name: "Vorlage", Subject: subject, BodyHTML: "<p>" + subject + "</p>", BodyText: subject,
		CreatedAt: now, UpdatedAt: now,
	}
	repo.templates[tpl.ID] = tpl
	return tpl
}

// ---------------------------------------------------------------------------
// Server + fixture wiring
// ---------------------------------------------------------------------------

type testEmailSigRuleLabelTplFixture struct {
	srv          *EmailGRPCServer
	sigRepo      *stubSignatureRepo
	ruleRepo     *stubRuleRepo
	labelRepo    *stubLabelRepo
	templateRepo *stubTemplateRepo
	messages     *stubLabelMessageReader
}

// newEmailSigRuleLabelTplFixture wires an EmailGRPCServer for the signature/
// rule/label/template RPCs. accountService, messageService, sendService,
// attachmentService, syncEngine, linkRepo, contactService and companyService
// are all nil because no RPC under test reaches them.
func newEmailSigRuleLabelTplFixture() *testEmailSigRuleLabelTplFixture {
	sigRepo := newStubSignatureRepo()
	ruleRepo := newStubRuleRepo()
	labelRepo := newStubLabelRepo()
	templateRepo := newStubTemplateRepo()
	messages := newStubLabelMessageReader()

	sigSvc := signature.NewService(sigRepo)
	ruleSvc := rule.NewService(ruleRepo)
	labelSvc := label.NewService(labelRepo, messages)
	tplSvc := template.NewService(templateRepo)

	srv := NewEmailGRPCServer(nil, nil, nil, sigSvc, nil, nil, nil, nil, nil, ruleSvc, labelSvc, tplSvc)

	return &testEmailSigRuleLabelTplFixture{
		srv: srv, sigRepo: sigRepo, ruleRepo: ruleRepo, labelRepo: labelRepo,
		templateRepo: templateRepo, messages: messages,
	}
}

// ---------------------------------------------------------------------------
// Signatures
// ---------------------------------------------------------------------------

func TestCreateSignature_FirstSignatureBecomesDefault(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	tenantID, userID := uuid.New(), uuid.New()

	resp, err := f.srv.CreateSignature(ctxWithTenant(tenantID), &emailv1.CreateSignatureRequest{
		UserId: userID.String(), Name: "Standard", HtmlContent: "<p>Gruss</p>",
	})
	requireGRPCOK(t, err)
	require.True(t, resp.Signature.IsDefault, "first signature for a user must become the default")
	require.Equal(t, "Standard", resp.Signature.Name)
}

func TestCreateSignature_InvalidUserID(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	_, err := f.srv.CreateSignature(ctxWithTenant(uuid.New()), &emailv1.CreateSignatureRequest{UserId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetSignature(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	tenantID, userID := uuid.New(), uuid.New()
	sig := seedSignature(f.sigRepo, tenantID, userID, "Standard", true)

	resp, err := f.srv.GetSignature(ctxWithTenant(tenantID), &emailv1.GetSignatureRequest{Id: sig.ID.String()})
	requireGRPCOK(t, err)
	require.Equal(t, sig.ID.String(), resp.Signature.Id)
}

func TestGetSignature_InvalidID(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	_, err := f.srv.GetSignature(ctxWithTenant(uuid.New()), &emailv1.GetSignatureRequest{Id: "nope"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetSignature_NotFound(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	_, err := f.srv.GetSignature(ctxWithTenant(uuid.New()), &emailv1.GetSignatureRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestListSignatures(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	tenantID, userID := uuid.New(), uuid.New()
	seedSignature(f.sigRepo, tenantID, userID, "A", true)
	seedSignature(f.sigRepo, tenantID, userID, "B", false)

	resp, err := f.srv.ListSignatures(ctxWithTenant(tenantID), &emailv1.ListSignaturesRequest{UserId: userID.String()})
	requireGRPCOK(t, err)
	require.Len(t, resp.Signatures, 2)

	// Wire-shape: a user with no signatures gets [] back, not null.
	empty, err := f.srv.ListSignatures(ctxWithTenant(tenantID), &emailv1.ListSignaturesRequest{UserId: uuid.New().String()})
	requireGRPCOK(t, err)
	require.NotNil(t, empty.Signatures)
	require.Empty(t, empty.Signatures)
}

func TestListSignatures_InvalidUserID(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	_, err := f.srv.ListSignatures(ctxWithTenant(uuid.New()), &emailv1.ListSignaturesRequest{UserId: "nope"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdateSignature(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	tenantID, userID := uuid.New(), uuid.New()
	sig := seedSignature(f.sigRepo, tenantID, userID, "Alt", false)
	newName := "Neu"

	resp, err := f.srv.UpdateSignature(ctxWithTenant(tenantID), &emailv1.UpdateSignatureRequest{
		Id: sig.ID.String(), Name: &newName,
	})
	requireGRPCOK(t, err)
	require.Equal(t, "Neu", resp.Signature.Name)
}

func TestUpdateSignature_InvalidID(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	_, err := f.srv.UpdateSignature(ctxWithTenant(uuid.New()), &emailv1.UpdateSignatureRequest{Id: "nope"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestDeleteSignature(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	tenantID, userID := uuid.New(), uuid.New()
	sig := seedSignature(f.sigRepo, tenantID, userID, "Weg", false)

	_, err := f.srv.DeleteSignature(ctxWithTenant(tenantID), &emailv1.DeleteSignatureRequest{Id: sig.ID.String()})
	requireGRPCOK(t, err)
	_, stillThere := f.sigRepo.sigs[sig.ID]
	require.False(t, stillThere)
}

func TestDeleteSignature_InvalidID(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	_, err := f.srv.DeleteSignature(ctxWithTenant(uuid.New()), &emailv1.DeleteSignatureRequest{Id: "nope"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestSetDefaultSignature(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	tenantID, userID := uuid.New(), uuid.New()
	first := seedSignature(f.sigRepo, tenantID, userID, "A", true)
	second := seedSignature(f.sigRepo, tenantID, userID, "B", false)

	resp, err := f.srv.SetDefaultSignature(ctxWithTenant(tenantID), &emailv1.SetDefaultSignatureRequest{
		Id: second.ID.String(), UserId: userID.String(),
	})
	requireGRPCOK(t, err)
	require.True(t, resp.Signature.IsDefault)
	require.False(t, f.sigRepo.sigs[first.ID].IsDefault, "the previous default must be cleared")
}

func TestSetDefaultSignature_NotFound(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	_, err := f.srv.SetDefaultSignature(ctxWithTenant(uuid.New()), &emailv1.SetDefaultSignatureRequest{
		Id: uuid.New().String(), UserId: uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.NotFound)
}

// ---------------------------------------------------------------------------
// Rules
// ---------------------------------------------------------------------------

func TestListEmailRules(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	tenantID := uuid.New()
	seedRule(f.ruleRepo, tenantID, models.EmailRuleFieldSubject, "Rechnung", models.EmailRuleActionLabel, uuid.New())

	resp, err := f.srv.ListEmailRules(ctxWithTenant(tenantID), &emailv1.ListEmailRulesRequest{})
	requireGRPCOK(t, err)
	require.Len(t, resp.Rules, 1)

	// Wire-shape: a tenant with no rules gets [] back, not null.
	empty, err := f.srv.ListEmailRules(ctxWithTenant(uuid.New()), &emailv1.ListEmailRulesRequest{})
	requireGRPCOK(t, err)
	require.NotNil(t, empty.Rules)
	require.Empty(t, empty.Rules)
}

func TestListEmailRules_MissingTenant(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	_, err := f.srv.ListEmailRules(context.Background(), &emailv1.ListEmailRulesRequest{})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateEmailRule(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	tenantID := uuid.New()
	target := uuid.New()
	f.ruleRepo.labels[target] = tenantID

	resp, err := f.srv.CreateEmailRule(ctxWithTenant(tenantID), &emailv1.CreateEmailRuleRequest{
		Name: "Rechnungen", Field: models.EmailRuleFieldSubject, Op: models.EmailRuleOpContains,
		Value: "Rechnung", ActionType: models.EmailRuleActionLabel, ActionTarget: target.String(),
	})
	requireGRPCOK(t, err)
	require.Equal(t, target.String(), resp.Rule.ActionTarget)
}

func TestCreateEmailRule_InvalidField(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	target := uuid.New()
	tenantID := uuid.New()
	f.ruleRepo.labels[target] = tenantID

	_, err := f.srv.CreateEmailRule(ctxWithTenant(tenantID), &emailv1.CreateEmailRuleRequest{
		Name: "Kaputt", Field: "cc", Op: models.EmailRuleOpContains,
		Value: "x", ActionType: models.EmailRuleActionLabel, ActionTarget: target.String(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdateEmailRule(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	tenantID := uuid.New()
	target := uuid.New()
	f.ruleRepo.labels[target] = tenantID
	rl := seedRule(f.ruleRepo, tenantID, models.EmailRuleFieldSubject, "Alt", models.EmailRuleActionLabel, target)
	newValue := "Neu"

	resp, err := f.srv.UpdateEmailRule(ctxWithTenant(tenantID), &emailv1.UpdateEmailRuleRequest{
		Id: rl.ID.String(), Value: &newValue,
	})
	requireGRPCOK(t, err)
	require.Equal(t, "Neu", resp.Rule.Value)
}

func TestUpdateEmailRule_InvalidID(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	_, err := f.srv.UpdateEmailRule(ctxWithTenant(uuid.New()), &emailv1.UpdateEmailRuleRequest{Id: "nope"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestDeleteEmailRule(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	tenantID := uuid.New()
	rl := seedRule(f.ruleRepo, tenantID, models.EmailRuleFieldSubject, "x", models.EmailRuleActionLabel, uuid.New())

	resp, err := f.srv.DeleteEmailRule(ctxWithTenant(tenantID), &emailv1.DeleteEmailRuleRequest{Id: rl.ID.String()})
	requireGRPCOK(t, err)
	require.True(t, resp.Success)
}

func TestDeleteEmailRule_NotFound(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	_, err := f.srv.DeleteEmailRule(ctxWithTenant(uuid.New()), &emailv1.DeleteEmailRuleRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestApplyEmailRules(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	tenantID := uuid.New()
	labelTarget := uuid.New()
	f.ruleRepo.labels[labelTarget] = tenantID
	seedRule(f.ruleRepo, tenantID, models.EmailRuleFieldSubject, "rechnung", models.EmailRuleActionLabel, labelTarget)

	matched := uuid.New()
	f.ruleRepo.candidates = []*models.EmailRuleCandidate{
		{ID: matched, Subject: "Ihre Rechnung 04/2026"},
		{ID: uuid.New(), Subject: "Mittagessen?"},
	}

	resp, err := f.srv.ApplyEmailRules(ctxWithTenant(tenantID), &emailv1.ApplyEmailRulesRequest{})
	requireGRPCOK(t, err)
	require.EqualValues(t, 1, resp.Affected)
	require.EqualValues(t, 2, resp.Scanned)
}

func TestApplyEmailRules_MissingTenant(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	_, err := f.srv.ApplyEmailRules(context.Background(), &emailv1.ApplyEmailRulesRequest{})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ---------------------------------------------------------------------------
// Labels
// ---------------------------------------------------------------------------

func TestListEmailLabels(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	tenantID := uuid.New()
	seedLabel(f.labelRepo, tenantID, "Wichtig", "#FF0000")

	resp, err := f.srv.ListEmailLabels(ctxWithTenant(tenantID), &emailv1.ListEmailLabelsRequest{})
	requireGRPCOK(t, err)
	require.Len(t, resp.Labels, 1)

	// Wire-shape: a tenant with no labels gets [] back, not null.
	empty, err := f.srv.ListEmailLabels(ctxWithTenant(uuid.New()), &emailv1.ListEmailLabelsRequest{})
	requireGRPCOK(t, err)
	require.NotNil(t, empty.Labels)
	require.Empty(t, empty.Labels)
}

func TestListEmailLabels_MissingTenant(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	_, err := f.srv.ListEmailLabels(context.Background(), &emailv1.ListEmailLabelsRequest{})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateEmailLabel_DefaultsColorWhenEmpty(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	resp, err := f.srv.CreateEmailLabel(ctxWithTenant(uuid.New()), &emailv1.CreateEmailLabelRequest{Name: "Wichtig"})
	requireGRPCOK(t, err)
	require.Equal(t, "#6B7280", resp.Label.Color)
}

func TestCreateEmailLabel_InvalidColor(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	_, err := f.srv.CreateEmailLabel(ctxWithTenant(uuid.New()), &emailv1.CreateEmailLabelRequest{
		Name: "Wichtig", Color: "notahex",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdateEmailLabel(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	tenantID := uuid.New()
	l := seedLabel(f.labelRepo, tenantID, "Alt", "#6B7280")
	newName := "Neu"

	resp, err := f.srv.UpdateEmailLabel(ctxWithTenant(tenantID), &emailv1.UpdateEmailLabelRequest{
		Id: l.ID.String(), Name: &newName,
	})
	requireGRPCOK(t, err)
	require.Equal(t, "Neu", resp.Label.Name)
}

func TestUpdateEmailLabel_InvalidID(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	_, err := f.srv.UpdateEmailLabel(ctxWithTenant(uuid.New()), &emailv1.UpdateEmailLabelRequest{Id: "nope"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestDeleteEmailLabel(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	tenantID := uuid.New()
	l := seedLabel(f.labelRepo, tenantID, "Weg", "#6B7280")

	resp, err := f.srv.DeleteEmailLabel(ctxWithTenant(tenantID), &emailv1.DeleteEmailLabelRequest{Id: l.ID.String()})
	requireGRPCOK(t, err)
	require.True(t, resp.Success)
}

func TestDeleteEmailLabel_InvalidID(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	_, err := f.srv.DeleteEmailLabel(ctxWithTenant(uuid.New()), &emailv1.DeleteEmailLabelRequest{Id: "nope"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestAssignMessageLabels(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	tenantID := uuid.New()
	l1 := seedLabel(f.labelRepo, tenantID, "A", "#6B7280")
	l2 := seedLabel(f.labelRepo, tenantID, "B", "#FF0000")
	msg := seedLabelableMessage(f.messages, tenantID)

	resp, err := f.srv.AssignMessageLabels(ctxWithTenant(tenantID), &emailv1.AssignMessageLabelsRequest{
		MessageId: msg.ID.String(), LabelIds: []string{l1.ID.String(), l2.ID.String()},
	})
	requireGRPCOK(t, err)
	require.Equal(t, msg.ID.String(), resp.Message.Id)
	require.ElementsMatch(t, []uuid.UUID{l1.ID, l2.ID}, f.labelRepo.assigned[msg.ID])
}

func TestAssignMessageLabels_InvalidLabelID(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	tenantID := uuid.New()
	msg := seedLabelableMessage(f.messages, tenantID)

	_, err := f.srv.AssignMessageLabels(ctxWithTenant(tenantID), &emailv1.AssignMessageLabelsRequest{
		MessageId: msg.ID.String(), LabelIds: []string{"not-a-uuid"},
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ---------------------------------------------------------------------------
// Templates
// ---------------------------------------------------------------------------

func TestListEmailTemplates(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	tenantID, userID := uuid.New(), uuid.New()
	seedTemplate(f.templateRepo, tenantID, nil, template.VisibilityShared, "Willkommen")

	resp, err := f.srv.ListEmailTemplates(ctxWithTenant(tenantID), &emailv1.ListEmailTemplatesRequest{UserId: userID.String()})
	requireGRPCOK(t, err)
	require.Len(t, resp.Templates, 1)
}

func TestListEmailTemplates_InvalidUserID(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	_, err := f.srv.ListEmailTemplates(ctxWithTenant(uuid.New()), &emailv1.ListEmailTemplatesRequest{UserId: "nope"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetEmailTemplate(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	tenantID, userID := uuid.New(), uuid.New()
	tpl := seedTemplate(f.templateRepo, tenantID, nil, template.VisibilityShared, "Willkommen")

	resp, err := f.srv.GetEmailTemplate(ctxWithTenant(tenantID), &emailv1.GetEmailTemplateRequest{
		Id: tpl.ID.String(), UserId: userID.String(),
	})
	requireGRPCOK(t, err)
	require.Equal(t, tpl.ID.String(), resp.Template.Id)
}

func TestGetEmailTemplate_InvalidID(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	_, err := f.srv.GetEmailTemplate(ctxWithTenant(uuid.New()), &emailv1.GetEmailTemplateRequest{
		Id: "nope", UserId: uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateEmailTemplate_DefaultsToPersonalAndOwned(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	userID := uuid.New()

	resp, err := f.srv.CreateEmailTemplate(ctxWithTenant(uuid.New()), &emailv1.CreateEmailTemplateRequest{
		UserId: userID.String(), Name: "Standard", Subject: "Hallo {{contact_first_name}}",
	})
	requireGRPCOK(t, err)
	require.Equal(t, template.VisibilityPersonal, resp.Template.Visibility)
	require.Equal(t, userID.String(), resp.Template.OwnerId)
}

func TestCreateEmailTemplate_InvalidVisibility(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	_, err := f.srv.CreateEmailTemplate(ctxWithTenant(uuid.New()), &emailv1.CreateEmailTemplateRequest{
		UserId: uuid.New().String(), Name: "Standard", Visibility: "public",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdateEmailTemplate_SwitchToSharedClearsOwner(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	tenantID, userID := uuid.New(), uuid.New()
	tpl := seedTemplate(f.templateRepo, tenantID, &userID, template.VisibilityPersonal, "Hallo")
	shared := template.VisibilityShared

	resp, err := f.srv.UpdateEmailTemplate(ctxWithTenant(tenantID), &emailv1.UpdateEmailTemplateRequest{
		Id: tpl.ID.String(), UserId: userID.String(), Visibility: &shared,
	})
	requireGRPCOK(t, err)
	require.Equal(t, "", resp.Template.OwnerId)
}

func TestUpdateEmailTemplate_InvalidID(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	_, err := f.srv.UpdateEmailTemplate(ctxWithTenant(uuid.New()), &emailv1.UpdateEmailTemplateRequest{
		Id: "nope", UserId: uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestDeleteEmailTemplate(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	tenantID, userID := uuid.New(), uuid.New()
	tpl := seedTemplate(f.templateRepo, tenantID, &userID, template.VisibilityPersonal, "Weg")

	_, err := f.srv.DeleteEmailTemplate(ctxWithTenant(tenantID), &emailv1.DeleteEmailTemplateRequest{
		Id: tpl.ID.String(), UserId: userID.String(),
	})
	requireGRPCOK(t, err)
	_, stillThere := f.templateRepo.templates[tpl.ID]
	require.False(t, stillThere)
}

func TestDeleteEmailTemplate_InvalidUserID(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	_, err := f.srv.DeleteEmailTemplate(ctxWithTenant(uuid.New()), &emailv1.DeleteEmailTemplateRequest{
		Id: uuid.New().String(), UserId: "nope",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestRenderEmailTemplate(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	tenantID, userID := uuid.New(), uuid.New()
	tpl := seedTemplate(f.templateRepo, tenantID, &userID, template.VisibilityPersonal, "Hallo {{contact_first_name}}!")

	resp, err := f.srv.RenderEmailTemplate(ctxWithTenant(tenantID), &emailv1.RenderEmailTemplateRequest{
		Id: tpl.ID.String(), UserId: userID.String(),
		Values: map[string]string{"contact_first_name": "Max"},
	})
	requireGRPCOK(t, err)
	require.Equal(t, "Hallo Max!", resp.Subject)
}

func TestRenderEmailTemplate_InvalidID(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	_, err := f.srv.RenderEmailTemplate(ctxWithTenant(uuid.New()), &emailv1.RenderEmailTemplateRequest{
		Id: "nope", UserId: uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// TestRenderEmailTemplate_UnknownPlaceholderKeyIgnored documents the actual
// behavior of template.Service.Render (see AllowedPlaceholders in
// internal/email/template/service.go): a values key outside the fixed
// allow-list is silently ignored, not rejected. There is no error path for
// an unrecognized placeholder field -- the render always succeeds and simply
// leaves unmatched "{{token}}" occurrences in the output untouched.
func TestRenderEmailTemplate_UnknownPlaceholderKeyIgnored(t *testing.T) {
	f := newEmailSigRuleLabelTplFixture()
	tenantID, userID := uuid.New(), uuid.New()
	tpl := seedTemplate(f.templateRepo, tenantID, &userID, template.VisibilityPersonal,
		"Hallo {{contact_first_name}}, ref {{bogus_field}}")

	resp, err := f.srv.RenderEmailTemplate(ctxWithTenant(tenantID), &emailv1.RenderEmailTemplateRequest{
		Id: tpl.ID.String(), UserId: userID.String(),
		Values: map[string]string{"contact_first_name": "Max", "bogus_field": "should-not-appear"},
	})
	requireGRPCOK(t, err)
	require.Equal(t, "Hallo Max, ref {{bogus_field}}", resp.Subject)
}
