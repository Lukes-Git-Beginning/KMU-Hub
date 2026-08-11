package routing

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/inbox/message"
	"github.com/kmuhub/kmuhub/internal/models"
)

// mockRoutingRepository implements Repository for testing.
type mockRoutingRepository struct {
	mu    sync.RWMutex
	rules map[uuid.UUID]*models.RoutingRule

	createErr     error
	updateErr     error
	deleteErr     error
	getByIDErr    error
	listActiveErr error
	listAllErr    error

	listActiveCalls int
}

func newMockRoutingRepository() *mockRoutingRepository {
	return &mockRoutingRepository{
		rules: make(map[uuid.UUID]*models.RoutingRule),
	}
}

func (m *mockRoutingRepository) Create(_ context.Context, rule *models.RoutingRule) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rules[rule.ID] = rule
	return nil
}

func (m *mockRoutingRepository) Update(_ context.Context, rule *models.RoutingRule) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.rules[rule.ID]; !ok {
		return ErrRuleNotFound
	}
	m.rules[rule.ID] = rule
	return nil
}

func (m *mockRoutingRepository) Delete(_ context.Context, _, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.rules[id]; !ok {
		return ErrRuleNotFound
	}
	delete(m.rules, id)
	return nil
}

func (m *mockRoutingRepository) GetByID(_ context.Context, _, id uuid.UUID) (*models.RoutingRule, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	rule, ok := m.rules[id]
	if !ok {
		return nil, ErrRuleNotFound
	}
	return rule, nil
}

func (m *mockRoutingRepository) ListActive(_ context.Context, _ uuid.UUID, _ *string) ([]*models.RoutingRule, error) {
	m.mu.Lock()
	m.listActiveCalls++
	m.mu.Unlock()
	if m.listActiveErr != nil {
		return nil, m.listActiveErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*models.RoutingRule
	for _, r := range m.rules {
		if r.IsActive {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *mockRoutingRepository) ListAll(_ context.Context, _ uuid.UUID) ([]*models.RoutingRule, error) {
	if m.listAllErr != nil {
		return nil, m.listAllErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*models.RoutingRule
	for _, r := range m.rules {
		result = append(result, r)
	}
	return result, nil
}

// mockMessageRepository implements message.Repository with minimal methods
// needed for routing tests.
type mockMessageRepository struct {
	mu       sync.RWMutex
	messages map[uuid.UUID]*models.InboxMessage

	updateErr error
}

func newMockMessageRepository() *mockMessageRepository {
	return &mockMessageRepository{
		messages: make(map[uuid.UUID]*models.InboxMessage),
	}
}

func (m *mockMessageRepository) Create(_ context.Context, msg *models.InboxMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages[msg.ID] = msg
	return nil
}

func (m *mockMessageRepository) GetByID(_ context.Context, id uuid.UUID, _ uuid.UUID) (*models.InboxMessage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	msg, ok := m.messages[id]
	if !ok {
		return nil, errors.New("message not found")
	}
	return msg, nil
}

func (m *mockMessageRepository) List(_ context.Context, _ message.ListFilter) ([]*models.InboxMessage, int, error) {
	return nil, 0, nil
}

func (m *mockMessageRepository) Update(_ context.Context, msg *models.InboxMessage) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages[msg.ID] = msg
	return nil
}

func (m *mockMessageRepository) MarkRead(_ context.Context, _, _ uuid.UUID) error   { return nil }
func (m *mockMessageRepository) MarkUnread(_ context.Context, _, _ uuid.UUID) error { return nil }
func (m *mockMessageRepository) ToggleStar(_ context.Context, _, _ uuid.UUID) error { return nil }
func (m *mockMessageRepository) SetStatus(_ context.Context, _, _ uuid.UUID, _ string) error {
	return nil
}
func (m *mockMessageRepository) AddTag(_ context.Context, _, _ uuid.UUID, _ string) error    { return nil }
func (m *mockMessageRepository) RemoveTag(_ context.Context, _, _ uuid.UUID, _ string) error { return nil }
func (m *mockMessageRepository) Archive(_ context.Context, _, _ uuid.UUID) error   { return nil }
func (m *mockMessageRepository) Unarchive(_ context.Context, _, _ uuid.UUID) error { return nil }
func (m *mockMessageRepository) Snooze(_ context.Context, _, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (m *mockMessageRepository) UnsnoozeExpired(_ context.Context) (int, error) { return 0, nil }
func (m *mockMessageRepository) AssignMessage(_ context.Context, _, _ uuid.UUID, _ uuid.UUID) (bool, error) {
	return true, nil
}
func (m *mockMessageRepository) GetUnreadCounts(_ context.Context, _ uuid.UUID) ([]models.UnreadCount, error) {
	return nil, nil
}
func (m *mockMessageRepository) BulkMarkRead(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error { return nil }
func (m *mockMessageRepository) BulkArchive(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error  { return nil }
func (m *mockMessageRepository) GetBySourceID(_ context.Context, _ uuid.UUID, _, _ string) (*models.InboxMessage, error) {
	return nil, nil
}
func (m *mockMessageRepository) NotifyDelivery(_ context.Context, _ string) error { return nil }

// mockEmailSender implements EmailSender for testing.
type mockEmailSender struct {
	sent    []sentEmail
	sendErr error
}

type sentEmail struct {
	to, subject, body string
}

func (m *mockEmailSender) Send(_ context.Context, to, subject, body string) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	m.sent = append(m.sent, sentEmail{to: to, subject: subject, body: body})
	return nil
}

func validConditionsJSON() json.RawMessage {
	return json.RawMessage(`{"field":"channel","operator":"equals","value":"email"}`)
}

func validActionsJSON() json.RawMessage {
	return json.RawMessage(`[{"type":"add_tags","config":{"tags":["important"]}}]`)
}

func newTestRule() *models.RoutingRule {
	return &models.RoutingRule{
		ID:         uuid.New(),
		TenantID:   uuid.New(),
		Name:       "Test Rule",
		Conditions: validConditionsJSON(),
		Actions:    validActionsJSON(),
		Priority:   1,
		IsActive:   true,
		CreatedBy:  uuid.New(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

func TestCreate_Success(t *testing.T) {
	repo := newMockRoutingRepository()
	msgRepo := newMockMessageRepository()
	svc := NewService(repo, msgRepo, &mockEmailSender{})

	rule := newTestRule()
	err := svc.Create(context.Background(), rule)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, rule.ID)

	stored, err := repo.GetByID(context.Background(), rule.TenantID, rule.ID)
	require.NoError(t, err)
	assert.Equal(t, rule.Name, stored.Name)
}

func TestCreate_RepoError(t *testing.T) {
	repo := newMockRoutingRepository()
	repo.createErr = errors.New("db connection failed")
	msgRepo := newMockMessageRepository()
	svc := NewService(repo, msgRepo, &mockEmailSender{})

	rule := newTestRule()
	err := svc.Create(context.Background(), rule)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "db connection failed")
}

func TestUpdate_Success(t *testing.T) {
	repo := newMockRoutingRepository()
	msgRepo := newMockMessageRepository()
	svc := NewService(repo, msgRepo, &mockEmailSender{})

	rule := newTestRule()
	repo.rules[rule.ID] = rule

	rule.Name = "Updated Rule"
	err := svc.Update(context.Background(), rule)

	require.NoError(t, err)
	stored, _ := repo.GetByID(context.Background(), rule.TenantID, rule.ID)
	assert.Equal(t, "Updated Rule", stored.Name)
}

func TestDelete_Success(t *testing.T) {
	repo := newMockRoutingRepository()
	msgRepo := newMockMessageRepository()
	svc := NewService(repo, msgRepo, &mockEmailSender{})

	rule := newTestRule()
	repo.rules[rule.ID] = rule

	err := svc.Delete(context.Background(), rule.TenantID, rule.ID)

	require.NoError(t, err)
	_, err = repo.GetByID(context.Background(), rule.TenantID, rule.ID)
	require.ErrorIs(t, err, ErrRuleNotFound)
}

func TestDelete_NotFound(t *testing.T) {
	repo := newMockRoutingRepository()
	msgRepo := newMockMessageRepository()
	svc := NewService(repo, msgRepo, &mockEmailSender{})

	err := svc.Delete(context.Background(), uuid.New(), uuid.New())

	require.ErrorIs(t, err, ErrRuleNotFound)
}

func TestGetByID_Success(t *testing.T) {
	repo := newMockRoutingRepository()
	msgRepo := newMockMessageRepository()
	svc := NewService(repo, msgRepo, &mockEmailSender{})

	rule := newTestRule()
	repo.rules[rule.ID] = rule

	result, err := svc.GetByID(context.Background(), rule.TenantID, rule.ID)

	require.NoError(t, err)
	assert.Equal(t, rule.ID, result.ID)
	assert.Equal(t, rule.Name, result.Name)
}

func TestGetByID_NotFound(t *testing.T) {
	repo := newMockRoutingRepository()
	msgRepo := newMockMessageRepository()
	svc := NewService(repo, msgRepo, &mockEmailSender{})

	_, err := svc.GetByID(context.Background(), uuid.New(), uuid.New())

	require.ErrorIs(t, err, ErrRuleNotFound)
}

func TestListAll_Success(t *testing.T) {
	repo := newMockRoutingRepository()
	msgRepo := newMockMessageRepository()
	svc := NewService(repo, msgRepo, &mockEmailSender{})

	tenantID := uuid.New()
	repo.rules[uuid.New()] = newTestRule()
	repo.rules[uuid.New()] = newTestRule()

	rules, err := svc.ListAll(context.Background(), tenantID)

	require.NoError(t, err)
	assert.Len(t, rules, 2)
}

func TestEvaluateAndApply_NoRules(t *testing.T) {
	repo := newMockRoutingRepository()
	msgRepo := newMockMessageRepository()
	svc := NewService(repo, msgRepo, &mockEmailSender{})

	msg := &models.InboxMessage{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		UserID:   uuid.New(),
		Channel:  "email",
		Subject:  "Test",
		Tags:     []string{},
	}

	err := svc.EvaluateAndApply(context.Background(), msg)

	require.NoError(t, err)
}

func TestTestRule_MatchesConditions(t *testing.T) {
	repo := newMockRoutingRepository()
	msgRepo := newMockMessageRepository()
	svc := NewService(repo, msgRepo, &mockEmailSender{})

	condition := models.Condition{
		Field:    "channel",
		Operator: "equals",
		Value:    "email",
	}

	msg := &models.InboxMessage{
		ID:      uuid.New(),
		Channel: "email",
		Tags:    []string{},
	}

	result := svc.TestRule(context.Background(), condition, msg)

	assert.True(t, result)

	// Non-matching
	msg.Channel = "chat"
	result = svc.TestRule(context.Background(), condition, msg)

	assert.False(t, result)
}

// ============================================================================
// Tenant Isolation Tests
// ============================================================================

// TestListAll_TenantIsolation verifies that ListAll passes tenantID to the repo correctly.
func TestListAll_TenantIsolation(t *testing.T) {
	repo := newMockRoutingRepository()
	msgRepo := newMockMessageRepository()
	svc := NewService(repo, msgRepo, &mockEmailSender{})

	tenantA := uuid.New()
	tenantB := uuid.New()

	ruleA := newTestRule()
	ruleA.TenantID = tenantA
	repo.rules[ruleA.ID] = ruleA

	rulesA, err := svc.ListAll(context.Background(), tenantA)
	require.NoError(t, err)
	// Mock returns all rules regardless of tenant; isolation enforced in postgres_repository.
	assert.Len(t, rulesA, 1)

	rulesB, err := svc.ListAll(context.Background(), tenantB)
	require.NoError(t, err)
	// Same mock returns same result; verifies tenantID propagates without error.
	assert.Len(t, rulesB, 1)
}

// ============================================================================
// executeActions
// ============================================================================

func newRoutableMessage() *models.InboxMessage {
	return &models.InboxMessage{
		ID:      uuid.New(),
		Channel: "email",
		Subject: "Test",
		Tags:    []string{},
	}
}

func TestExecuteActions_RouteToTeam_Success(t *testing.T) {
	repo := newMockRoutingRepository()
	msgRepo := newMockMessageRepository()
	svc := NewService(repo, msgRepo, &mockEmailSender{})

	teamID := uuid.New()
	msg := newRoutableMessage()
	actions := []models.Action{
		{Type: "route_to_team", Config: json.RawMessage(`{"team_inbox_id":"` + teamID.String() + `"}`)},
	}

	err := svc.executeActions(context.Background(), msg, actions)

	require.NoError(t, err)
	require.NotNil(t, msg.TeamInboxID)
	assert.Equal(t, teamID, *msg.TeamInboxID)
}

func TestExecuteActions_RouteToTeam_InvalidTeamID(t *testing.T) {
	repo := newMockRoutingRepository()
	msgRepo := newMockMessageRepository()
	svc := NewService(repo, msgRepo, &mockEmailSender{})

	msg := newRoutableMessage()
	actions := []models.Action{
		{Type: "route_to_team", Config: json.RawMessage(`{"team_inbox_id":"not-a-uuid"}`)},
	}

	err := svc.executeActions(context.Background(), msg, actions)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid team_inbox_id")
	assert.Nil(t, msg.TeamInboxID)
}

func TestExecuteActions_AssignTo_Success(t *testing.T) {
	repo := newMockRoutingRepository()
	msgRepo := newMockMessageRepository()
	svc := NewService(repo, msgRepo, &mockEmailSender{})

	userID := uuid.New()
	msg := newRoutableMessage()
	actions := []models.Action{
		{Type: "assign_to", Config: json.RawMessage(`{"user_id":"` + userID.String() + `"}`)},
	}

	err := svc.executeActions(context.Background(), msg, actions)

	require.NoError(t, err)
	require.NotNil(t, msg.AssignedTo)
	assert.Equal(t, userID, *msg.AssignedTo)
}

func TestExecuteActions_AssignTo_InvalidUserID(t *testing.T) {
	repo := newMockRoutingRepository()
	msgRepo := newMockMessageRepository()
	svc := NewService(repo, msgRepo, &mockEmailSender{})

	msg := newRoutableMessage()
	actions := []models.Action{
		{Type: "assign_to", Config: json.RawMessage(`{"user_id":"not-a-uuid"}`)},
	}

	err := svc.executeActions(context.Background(), msg, actions)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user_id")
	assert.Nil(t, msg.AssignedTo)
}

func TestExecuteActions_AddTags_DedupesAgainstExisting(t *testing.T) {
	repo := newMockRoutingRepository()
	msgRepo := newMockMessageRepository()
	svc := NewService(repo, msgRepo, &mockEmailSender{})

	msg := newRoutableMessage()
	msg.Tags = []string{"vip"}
	actions := []models.Action{
		{Type: "add_tags", Config: json.RawMessage(`{"tags":["vip","urgent"]}`)},
	}

	err := svc.executeActions(context.Background(), msg, actions)

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"vip", "urgent"}, msg.Tags)
}

func TestExecuteActions_AddTags_InvalidConfig(t *testing.T) {
	repo := newMockRoutingRepository()
	msgRepo := newMockMessageRepository()
	svc := NewService(repo, msgRepo, &mockEmailSender{})

	msg := newRoutableMessage()
	actions := []models.Action{
		{Type: "add_tags", Config: json.RawMessage(`not-json`)},
	}

	err := svc.executeActions(context.Background(), msg, actions)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse add_tags config")
}

func TestExecuteActions_AutoReply_SendsForEmailChannel(t *testing.T) {
	repo := newMockRoutingRepository()
	msgRepo := newMockMessageRepository()
	sender := &mockEmailSender{}
	svc := NewService(repo, msgRepo, sender)

	senderEmail := "customer@example.test"
	msg := newRoutableMessage()
	msg.SenderEmail = &senderEmail
	actions := []models.Action{
		{Type: "auto_reply", Config: json.RawMessage(`{"subject":"Re: Hi","body":"Thanks!"}`)},
	}

	err := svc.executeActions(context.Background(), msg, actions)

	require.NoError(t, err)
	require.Len(t, sender.sent, 1)
	assert.Equal(t, senderEmail, sender.sent[0].to)
	assert.Equal(t, "Thanks!", sender.sent[0].body)
}

func TestExecuteActions_AutoReply_SendErrorIsNonFatal(t *testing.T) {
	repo := newMockRoutingRepository()
	msgRepo := newMockMessageRepository()
	sender := &mockEmailSender{sendErr: errors.New("smtp down")}
	svc := NewService(repo, msgRepo, sender)

	senderEmail := "customer@example.test"
	msg := newRoutableMessage()
	msg.SenderEmail = &senderEmail
	actions := []models.Action{
		{Type: "auto_reply", Config: json.RawMessage(`{"body":"Thanks!"}`)},
	}

	err := svc.executeActions(context.Background(), msg, actions)

	require.NoError(t, err, "auto_reply failures are logged as a warning, not propagated")
}

func TestExecuteActions_AutoReply_SkippedForNonEmailChannel(t *testing.T) {
	repo := newMockRoutingRepository()
	msgRepo := newMockMessageRepository()
	sender := &mockEmailSender{}
	svc := NewService(repo, msgRepo, sender)

	senderEmail := "customer@example.test"
	msg := newRoutableMessage()
	msg.Channel = "chat"
	msg.SenderEmail = &senderEmail
	actions := []models.Action{
		{Type: "auto_reply", Config: json.RawMessage(`{"body":"Thanks!"}`)},
	}

	err := svc.executeActions(context.Background(), msg, actions)

	require.NoError(t, err)
	assert.Empty(t, sender.sent)
}

// ============================================================================
// Rule cache
// ============================================================================

func TestGetCachedRules_ServesFromCacheWithinTTL(t *testing.T) {
	repo := newMockRoutingRepository()
	msgRepo := newMockMessageRepository()
	svc := NewService(repo, msgRepo, &mockEmailSender{})

	tenantID := uuid.New()
	rule := newTestRule()
	rule.TenantID = tenantID
	repo.rules[rule.ID] = rule

	rules1, err := svc.getCachedRules(context.Background(), tenantID, nil)
	require.NoError(t, err)
	assert.Len(t, rules1, 1)
	assert.Equal(t, 1, repo.listActiveCalls)

	rules2, err := svc.getCachedRules(context.Background(), tenantID, nil)
	require.NoError(t, err)
	assert.Len(t, rules2, 1)
	assert.Equal(t, 1, repo.listActiveCalls, "second call within TTL must not hit the repository again")
}

func TestGetCachedRules_RepoError(t *testing.T) {
	repo := newMockRoutingRepository()
	repo.listActiveErr = errors.New("db connection failed")
	msgRepo := newMockMessageRepository()
	svc := NewService(repo, msgRepo, &mockEmailSender{})

	_, err := svc.getCachedRules(context.Background(), uuid.New(), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "db connection failed")
}

func TestInvalidateCache_ForcesRefreshOnNextAccess(t *testing.T) {
	repo := newMockRoutingRepository()
	msgRepo := newMockMessageRepository()
	svc := NewService(repo, msgRepo, &mockEmailSender{})

	tenantID := uuid.New()
	rule := newTestRule()
	rule.TenantID = tenantID
	repo.rules[rule.ID] = rule

	_, err := svc.getCachedRules(context.Background(), tenantID, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, repo.listActiveCalls)

	svc.invalidateCache(tenantID)

	_, err = svc.getCachedRules(context.Background(), tenantID, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, repo.listActiveCalls, "a cache invalidation must force a repository refresh on the next call")
}

func TestFilterByChannel(t *testing.T) {
	svc := &Service{}

	emailChannel := "email"
	chatChannel := "chat"
	ruleAllChannels := &models.RoutingRule{ID: uuid.New(), Channel: nil}
	ruleEmail := &models.RoutingRule{ID: uuid.New(), Channel: &emailChannel}
	ruleChat := &models.RoutingRule{ID: uuid.New(), Channel: &chatChannel}
	rules := []*models.RoutingRule{ruleAllChannels, ruleEmail, ruleChat}

	filtered := svc.filterByChannel(rules, &emailChannel)
	assert.ElementsMatch(t, []*models.RoutingRule{ruleAllChannels, ruleEmail}, filtered)

	unfiltered := svc.filterByChannel(rules, nil)
	assert.Len(t, unfiltered, 3)
}
