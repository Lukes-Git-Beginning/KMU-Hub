package server

// Coverage for internal/server/notification_grpc.go (BACKLOG b-cov-server-notification).
// notification_integration_test.go and notification_webhook_test.go already cover
// TestIntegrationConfig and HandlePlatformWebhook -- this file does not repeat either.
// It adds: (a) mapNotificationError as a full sentinel table, (b) all eight named
// proto<->domain converters (nil-optional vs populated), (c) the validation paths of
// the remaining 28 RPC handlers, and (d) the isolation guarantees that matter here --
// a notification listed for user A must never surface for user B, a mutation on a
// notification owned by another user is rejected, and quiet hours/config records are
// scoped to the requesting tenant.

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/event"
	"github.com/kmuhub/kmuhub/internal/notification/integration"
	"github.com/kmuhub/kmuhub/internal/notification/notification"
	"github.com/kmuhub/kmuhub/internal/notification/preference"
	notificationv1 "github.com/kmuhub/kmuhub/proto/notification/v1"
)

// ============================================================================
// Stub repositories
// ============================================================================

// stubNotifRepo implements notification.Repository. Only the methods the RPC
// handlers under test actually reach are implemented; the rest stay on the
// embedded nil interface and panic loudly if a future test reaches for them.
type stubNotifRepo struct {
	notification.Repository
	mu   sync.Mutex
	byID map[uuid.UUID]*models.Notification
}

func newStubNotifRepo() *stubNotifRepo {
	return &stubNotifRepo{byID: make(map[uuid.UUID]*models.Notification)}
}

func (r *stubNotifRepo) seed(n *models.Notification) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *n
	r.byID[n.ID] = &cp
}

func (r *stubNotifRepo) Create(_ context.Context, n *models.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[n.ID] = n
	return nil
}

func (r *stubNotifRepo) GetByID(_ context.Context, tenantID, id uuid.UUID) (*models.Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n, ok := r.byID[id]
	if !ok || n.TenantID != tenantID {
		return nil, notification.ErrNotificationNotFound
	}
	cp := *n
	return &cp, nil
}

// List mirrors the WHERE tenant_id = ? AND user_id = ? the real repository
// runs -- this is the enforcement point the cross-user test proves against.
func (r *stubNotifRepo) List(_ context.Context, filter notification.ListFilter, offset, limit int) ([]*models.Notification, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var matched []*models.Notification
	for _, n := range r.byID {
		if n.TenantID != filter.TenantID || n.UserID != filter.UserID {
			continue
		}
		if filter.ModuleID != nil && n.ModuleID != *filter.ModuleID {
			continue
		}
		if filter.IsRead != nil && n.IsRead != *filter.IsRead {
			continue
		}
		matched = append(matched, n)
	}
	total := len(matched)
	if offset >= total {
		return nil, total, nil
	}
	end := min(offset+limit, total)
	return matched[offset:end], total, nil
}

func (r *stubNotifRepo) GetUnreadCount(_ context.Context, tenantID, userID uuid.UUID) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, n := range r.byID {
		if n.TenantID == tenantID && n.UserID == userID && !n.IsRead {
			count++
		}
	}
	return count, nil
}

func (r *stubNotifRepo) MarkRead(_ context.Context, tenantID, id uuid.UUID, readAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	n, ok := r.byID[id]
	if !ok || n.TenantID != tenantID {
		return notification.ErrNotificationNotFound
	}
	n.IsRead = true
	n.ReadAt = &readAt
	return nil
}

func (r *stubNotifRepo) MarkAllRead(_ context.Context, tenantID, userID uuid.UUID, moduleID *string, readAt time.Time) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, n := range r.byID {
		if n.TenantID != tenantID || n.UserID != userID || n.IsRead {
			continue
		}
		if moduleID != nil && n.ModuleID != *moduleID {
			continue
		}
		n.IsRead = true
		n.ReadAt = &readAt
		count++
	}
	return count, nil
}

func (r *stubNotifRepo) SetPinned(_ context.Context, tenantID, id uuid.UUID, pinned bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	n, ok := r.byID[id]
	if !ok || n.TenantID != tenantID {
		return notification.ErrNotificationNotFound
	}
	n.IsPinned = pinned
	return nil
}

func (r *stubNotifRepo) SetDismissed(_ context.Context, tenantID, id uuid.UUID, dismissed bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	n, ok := r.byID[id]
	if !ok || n.TenantID != tenantID {
		return notification.ErrNotificationNotFound
	}
	n.IsDismissed = dismissed
	return nil
}

func (r *stubNotifRepo) Snooze(_ context.Context, tenantID, id uuid.UUID, until time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	n, ok := r.byID[id]
	if !ok || n.TenantID != tenantID {
		return notification.ErrNotificationNotFound
	}
	n.SnoozedUntil = &until
	return nil
}

// stubPrefRepo implements preference.Repository.
type stubPrefRepo struct {
	preference.Repository
	mu    sync.Mutex
	prefs []*models.NotificationPreference
	mutes map[uuid.UUID]*models.NotificationMute
	quiet map[string]*models.QuietHours // tenantID/userID -> record
}

func newStubPrefRepo() *stubPrefRepo {
	return &stubPrefRepo{
		mutes: make(map[uuid.UUID]*models.NotificationMute),
		quiet: make(map[string]*models.QuietHours),
	}
}

func quietKey(tenantID, userID uuid.UUID) string {
	return tenantID.String() + "/" + userID.String()
}

func (r *stubPrefRepo) ListPreferences(_ context.Context, tenantID, userID uuid.UUID, moduleID *string) ([]*models.NotificationPreference, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*models.NotificationPreference
	for _, p := range r.prefs {
		if p.TenantID != tenantID || p.UserID != userID {
			continue
		}
		if moduleID != nil && (p.ModuleID == nil || *p.ModuleID != *moduleID) {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

func (r *stubPrefRepo) UpsertPreference(_ context.Context, pref *models.NotificationPreference) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.prefs = append(r.prefs, pref)
	return nil
}

func (r *stubPrefRepo) IsResourceMuted(_ context.Context, tenantID, userID uuid.UUID, moduleID, resourceID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, m := range r.mutes {
		if m.TenantID == tenantID && m.UserID == userID && m.ModuleID == moduleID && m.ResourceID == resourceID {
			return true, nil
		}
	}
	return false, nil
}

func (r *stubPrefRepo) CreateMute(_ context.Context, mute *models.NotificationMute) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mutes[mute.ID] = mute
	return nil
}

func (r *stubPrefRepo) DeleteMute(_ context.Context, tenantID, userID, muteID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.mutes[muteID]
	if !ok || m.TenantID != tenantID || m.UserID != userID {
		return preference.ErrMuteNotFound
	}
	delete(r.mutes, muteID)
	return nil
}

func (r *stubPrefRepo) ListMutes(_ context.Context, tenantID, userID uuid.UUID, moduleID *string, offset, limit int) ([]*models.NotificationMute, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var matched []*models.NotificationMute
	for _, m := range r.mutes {
		if m.TenantID != tenantID || m.UserID != userID {
			continue
		}
		if moduleID != nil && m.ModuleID != *moduleID {
			continue
		}
		matched = append(matched, m)
	}
	total := len(matched)
	if offset >= total {
		return nil, total, nil
	}
	end := min(offset+limit, total)
	return matched[offset:end], total, nil
}

func (r *stubPrefRepo) GetQuietHours(_ context.Context, tenantID, userID uuid.UUID) (*models.QuietHours, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	qh, ok := r.quiet[quietKey(tenantID, userID)]
	if !ok {
		return nil, preference.ErrQuietHoursNotFound
	}
	return qh, nil
}

func (r *stubPrefRepo) UpsertQuietHours(_ context.Context, qh *models.QuietHours) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.quiet[quietKey(qh.TenantID, qh.UserID)] = qh
	return nil
}

// notifIntegStub implements integration.Repository for the config/mapping/
// account-link/link-token surface exercised by the RPCs under test.
type notifIntegStub struct {
	integration.Repository
	mu       sync.Mutex
	configs  map[uuid.UUID]*integration.IntegrationConfig
	mappings map[uuid.UUID]*integration.ChannelMapping
	links    map[string]*integration.AccountLink
	tokens   map[string]*integration.LinkToken
}

func newNotifIntegStub() *notifIntegStub {
	return &notifIntegStub{
		configs:  make(map[uuid.UUID]*integration.IntegrationConfig),
		mappings: make(map[uuid.UUID]*integration.ChannelMapping),
		links:    make(map[string]*integration.AccountLink),
		tokens:   make(map[string]*integration.LinkToken),
	}
}

func notifLinkKey(platform string, userID uuid.UUID) string {
	return platform + "/" + userID.String()
}

func (r *notifIntegStub) CreateConfig(_ context.Context, cfg *integration.IntegrationConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.configs[cfg.ID] = cfg
	return nil
}

func (r *notifIntegStub) GetConfigByPlatform(_ context.Context, platform string) (*integration.IntegrationConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.configs {
		if c.Platform == platform {
			return c, nil
		}
	}
	return nil, integration.ErrConfigNotFound
}

func (r *notifIntegStub) UpdateConfig(_ context.Context, cfg *integration.IntegrationConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.configs[cfg.ID]; !ok {
		return integration.ErrConfigNotFound
	}
	r.configs[cfg.ID] = cfg
	return nil
}

func (r *notifIntegStub) ListConfigs(_ context.Context) ([]*integration.IntegrationConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*integration.IntegrationConfig
	for _, c := range r.configs {
		out = append(out, c)
	}
	return out, nil
}

func (r *notifIntegStub) DeleteConfig(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.configs, id)
	return nil
}

func (r *notifIntegStub) CreateMapping(_ context.Context, m *integration.ChannelMapping) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mappings[m.ID] = m
	return nil
}

func (r *notifIntegStub) GetMapping(_ context.Context, id uuid.UUID) (*integration.ChannelMapping, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.mappings[id]
	if !ok {
		return nil, integration.ErrMappingNotFound
	}
	return m, nil
}

func (r *notifIntegStub) ListMappingsByConfig(_ context.Context, configID uuid.UUID) ([]*integration.ChannelMapping, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*integration.ChannelMapping
	for _, m := range r.mappings {
		if m.ConfigID == configID {
			out = append(out, m)
		}
	}
	return out, nil
}

func (r *notifIntegStub) UpdateMapping(_ context.Context, m *integration.ChannelMapping) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.mappings[m.ID]; !ok {
		return integration.ErrMappingNotFound
	}
	r.mappings[m.ID] = m
	return nil
}

func (r *notifIntegStub) DeleteMapping(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.mappings, id)
	return nil
}

func (r *notifIntegStub) CreateAccountLink(_ context.Context, link *integration.AccountLink) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.links[notifLinkKey(link.Platform, link.KMUHubUserID)] = link
	return nil
}

func (r *notifIntegStub) GetAccountLinkByKMUHubUser(_ context.Context, platform string, kmuhubUserID uuid.UUID) (*integration.AccountLink, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	link, ok := r.links[notifLinkKey(platform, kmuhubUserID)]
	if !ok {
		return nil, integration.ErrAccountLinkNotFound
	}
	return link, nil
}

func (r *notifIntegStub) DeleteAccountLink(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, l := range r.links {
		if l.ID == id {
			delete(r.links, k)
		}
	}
	return nil
}

func (r *notifIntegStub) CreateLinkToken(_ context.Context, token *integration.LinkToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tokens[token.TokenHash] = token
	return nil
}

func (r *notifIntegStub) GetLinkTokenByHash(_ context.Context, tokenHash string) (*integration.LinkToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tokens[tokenHash]
	if !ok {
		return nil, integration.ErrLinkTokenNotFound
	}
	return t, nil
}

func (r *notifIntegStub) MarkLinkTokenUsed(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.tokens {
		if t.ID == id {
			t.Used = true
		}
	}
	return nil
}

func (r *notifIntegStub) CleanupExpiredTokens(_ context.Context) (int, error) {
	return 0, nil
}

// ============================================================================
// Server construction helpers
// ============================================================================

// newTestNotificationServer returns a server with nil notifService/prefService.
// Usable ONLY for RPC paths that return before touching either -- every
// tenant/user_id parse error in notification_grpc.go runs before the service
// call, so this is safe for them and panics for anything reaching the happy
// path.
func newTestNotificationServer() *NotificationGRPCServer {
	return NewNotificationGRPCServer(nil, nil, nil)
}

func newNotificationServerWithRepos(notifRepo notification.Repository, prefRepo preference.Repository) *NotificationGRPCServer {
	prefSvc := preference.NewService(prefRepo)
	notifSvc := notification.NewService(notifRepo, prefSvc, nil, nil, nil)
	return NewNotificationGRPCServer(notifSvc, prefSvc, nil)
}

// newNotificationServerWithIntegration builds a server whose notifService/
// prefService validate-before-touch (nil repo is safe for UpdateQuietHours'
// timezone/time/day validation), wired with a working integration stub.
func newNotificationServerWithIntegration(repo integration.Repository, linkSvc *integration.AccountLinkService) *NotificationGRPCServer {
	prefSvc := preference.NewService(nil)
	return NewNotificationGRPCServer(nil, prefSvc, nil, WithIntegration(repo, linkSvc))
}

// ============================================================================
// mapNotificationError -- full sentinel table
// ============================================================================

func TestMapNotificationError_AllSentinels(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want codes.Code
	}{
		{"notification not found", notification.ErrNotificationNotFound, codes.NotFound},
		{"invalid priority", notification.ErrInvalidPriority, codes.InvalidArgument},
		{"unauthorized", notification.ErrUnauthorized, codes.PermissionDenied},
		{"invalid snooze time", notification.ErrInvalidSnoozeTime, codes.InvalidArgument},
		{"preference not found", preference.ErrPreferenceNotFound, codes.NotFound},
		{"quiet hours not found", preference.ErrQuietHoursNotFound, codes.NotFound},
		{"mute not found", preference.ErrMuteNotFound, codes.NotFound},
		{"mute already exists", preference.ErrMuteAlreadyExists, codes.AlreadyExists},
		{"invalid timezone", preference.ErrInvalidTimezone, codes.InvalidArgument},
		{"invalid time format", preference.ErrInvalidTimeFormat, codes.InvalidArgument},
		{"invalid day of week", preference.ErrInvalidDayOfWeek, codes.InvalidArgument},
		{"config not found", integration.ErrConfigNotFound, codes.NotFound},
		{"config already exists", integration.ErrConfigAlreadyExists, codes.AlreadyExists},
		{"mapping not found", integration.ErrMappingNotFound, codes.NotFound},
		{"account link not found", integration.ErrAccountLinkNotFound, codes.NotFound},
		{"account not linked", integration.ErrAccountNotLinked, codes.NotFound},
		{"link token expired", integration.ErrLinkTokenExpired, codes.DeadlineExceeded},
		{"link token used", integration.ErrLinkTokenUsed, codes.AlreadyExists},
		{"link token not found", integration.ErrLinkTokenNotFound, codes.NotFound},
		{"platform disabled", integration.ErrPlatformDisabled, codes.FailedPrecondition},
		{"invalid platform", integration.ErrInvalidPlatform, codes.InvalidArgument},
		{"tenant missing", integration.ErrTenantMissing, codes.Unauthenticated},
		{"nil error", nil, codes.Internal}, // errors.Is(nil, x) is always false -> falls to default
		{"unknown error", errors.New("boom"), codes.Internal},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := mapNotificationError(tt.err)
			requireGRPCCode(t, err, tt.want)
		})
	}
}

// ============================================================================
// Converters
// ============================================================================

func TestToNotificationInfo_NilOptionalFields(t *testing.T) {
	n := &models.Notification{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Title:     "t",
		Priority:  models.PriorityNormal,
		CreatedAt: time.Now(),
	}
	info := toNotificationInfo(n)
	assert.Nil(t, info.ActorId)
	assert.Nil(t, info.ActorName)
	assert.Nil(t, info.ResourceId)
	assert.Nil(t, info.Body)
	assert.Nil(t, info.DeepLink)
	assert.Nil(t, info.GroupKey)
	assert.Nil(t, info.ReadAt)
}

func TestToNotificationInfo_PopulatedFields(t *testing.T) {
	actorID := uuid.New()
	actorName := "Alice"
	resourceID := "res-1"
	body := "body text"
	deepLink := "/x/1"
	groupKey := "grp-1"
	readAt := time.Now()

	n := &models.Notification{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		Title:      "t",
		Priority:   models.PriorityUrgent,
		CreatedAt:  time.Now(),
		ActorID:    &actorID,
		ActorName:  &actorName,
		ResourceID: &resourceID,
		Body:       &body,
		DeepLink:   &deepLink,
		GroupKey:   &groupKey,
		ReadAt:     &readAt,
		IsRead:     true,
	}
	info := toNotificationInfo(n)
	require.NotNil(t, info.ActorId)
	assert.Equal(t, actorID.String(), *info.ActorId)
	require.NotNil(t, info.ActorName)
	assert.Equal(t, actorName, *info.ActorName)
	require.NotNil(t, info.ResourceId)
	assert.Equal(t, resourceID, *info.ResourceId)
	require.NotNil(t, info.Body)
	assert.Equal(t, body, *info.Body)
	require.NotNil(t, info.DeepLink)
	assert.Equal(t, deepLink, *info.DeepLink)
	require.NotNil(t, info.GroupKey)
	assert.Equal(t, groupKey, *info.GroupKey)
	require.NotNil(t, info.ReadAt)
	assert.Equal(t, notificationv1.Priority_PRIORITY_URGENT, info.Priority)
}

func TestToPreferenceInfo(t *testing.T) {
	p := &models.NotificationPreference{
		ID:          uuid.New(),
		UserID:      uuid.New(),
		InApp:       true,
		DesktopPush: false,
		Email:       true,
		SMS:         false,
		Sound:       "chime",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	info := toPreferenceInfo(p)
	assert.Equal(t, p.ID.String(), info.Id)
	assert.True(t, info.InApp)
	assert.False(t, info.DesktopPush)
	assert.True(t, info.Email)
	assert.Equal(t, "chime", info.Sound)
}

func TestToMuteInfo(t *testing.T) {
	m := &models.NotificationMute{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		ModuleID:   "crm",
		ResourceID: "contact-1",
		CreatedAt:  time.Now(),
	}
	info := toMuteInfo(m)
	assert.Equal(t, m.ID.String(), info.Id)
	assert.Equal(t, "crm", info.ModuleId)
	assert.Equal(t, "contact-1", info.ResourceId)
}

func TestToQuietHoursInfo_NilManualDNDUntil(t *testing.T) {
	qh := &models.QuietHours{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		StartTime:  "22:00",
		EndTime:    "06:00",
		Timezone:   "Europe/Berlin",
		DaysOfWeek: []int{1, 2, 3},
		Enabled:    true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	info := toQuietHoursInfo(qh)
	assert.Nil(t, info.ManualDndUntil)
	assert.Equal(t, []int32{1, 2, 3}, info.DaysOfWeek)
}

func TestToQuietHoursInfo_PopulatedManualDNDUntil(t *testing.T) {
	until := time.Now().Add(time.Hour)
	qh := &models.QuietHours{
		ID:             uuid.New(),
		UserID:         uuid.New(),
		ManualDND:      true,
		ManualDNDUntil: &until,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	info := toQuietHoursInfo(qh)
	require.NotNil(t, info.ManualDndUntil)
	assert.True(t, info.ManualDnd)
}

func TestToEventTypeInfo_NilDescription(t *testing.T) {
	et := &models.EventType{
		ID:              uuid.New(),
		ModuleID:        "crm",
		EventKey:        "contact.created",
		DisplayName:     "Contact Created",
		DefaultPriority: models.PriorityNormal,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	info := toEventTypeInfo(et)
	assert.Nil(t, info.Description)
}

func TestToEventTypeInfo_PopulatedDescription(t *testing.T) {
	desc := "fires when a contact is created"
	et := &models.EventType{
		ID:              uuid.New(),
		ModuleID:        "crm",
		EventKey:        "contact.created",
		DisplayName:     "Contact Created",
		Description:     &desc,
		DefaultPriority: models.PriorityLow,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	info := toEventTypeInfo(et)
	require.NotNil(t, info.Description)
	assert.Equal(t, desc, *info.Description)
	assert.Equal(t, notificationv1.Priority_PRIORITY_LOW, info.DefaultPriority)
}

func TestToPriority_AllValues(t *testing.T) {
	cases := []struct {
		in   string
		want notificationv1.Priority
	}{
		{models.PriorityUrgent, notificationv1.Priority_PRIORITY_URGENT},
		{models.PriorityNormal, notificationv1.Priority_PRIORITY_NORMAL},
		{models.PriorityLow, notificationv1.Priority_PRIORITY_LOW},
		{"garbage", notificationv1.Priority_PRIORITY_UNSPECIFIED},
		{"", notificationv1.Priority_PRIORITY_UNSPECIFIED},
	}
	for _, tt := range cases {
		assert.Equal(t, tt.want, toPriority(tt.in), "input %q", tt.in)
	}
}

func TestToIntegrationConfigInfo_NilMetadata(t *testing.T) {
	cfg := &integration.IntegrationConfig{
		ID:        uuid.New(),
		Platform:  integration.PlatformSlack,
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	info := toIntegrationConfigInfo(cfg)
	assert.Equal(t, "{}", info.Metadata)
}

func TestToIntegrationConfigInfo_PopulatedMetadata(t *testing.T) {
	cfg := &integration.IntegrationConfig{
		ID:        uuid.New(),
		Platform:  integration.PlatformTeams,
		Metadata:  []byte(`{"workspace":"zentria"}`),
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	info := toIntegrationConfigInfo(cfg)
	assert.Equal(t, `{"workspace":"zentria"}`, info.Metadata)
}

func TestToChannelMappingInfo(t *testing.T) {
	m := &integration.ChannelMapping{
		ID:          uuid.New(),
		ConfigID:    uuid.New(),
		ChannelID:   "C123",
		ChannelName: "#general",
		Modules:     []string{"crm", "hr"},
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	info := toChannelMappingInfo(m)
	assert.Equal(t, m.ID.String(), info.Id)
	assert.Equal(t, m.ConfigID.String(), info.ConfigId)
	assert.Equal(t, []string{"crm", "hr"}, info.Modules)
	assert.True(t, info.IsActive)
}

// ============================================================================
// Validation paths -- table test across the RPCs that validate before
// touching notifService/prefService/integrationRepo/linkService.
// ============================================================================

func TestNotificationGRPC_ValidationErrors(t *testing.T) {
	srv := newTestNotificationServer()
	integSrv := NewNotificationGRPCServer(nil, nil, nil, WithIntegration(newNotifIntegStub(), integration.NewAccountLinkService(newNotifIntegStub())))
	tenantID := uuid.New()
	ctx := helpdeskTenantContext(tenantID)
	bgCtx := context.Background()

	cases := []struct {
		name string
		want codes.Code
		call func() error
	}{
		{"ListNotifications/missing_tenant", codes.InvalidArgument, func() error {
			_, err := srv.ListNotifications(bgCtx, &notificationv1.ListNotificationsRequest{UserId: uuid.NewString()})
			return err
		}},
		{"ListNotifications/invalid_user_id", codes.InvalidArgument, func() error {
			_, err := srv.ListNotifications(ctx, &notificationv1.ListNotificationsRequest{UserId: "not-a-uuid"})
			return err
		}},
		{"GetUnreadCount/missing_tenant", codes.InvalidArgument, func() error {
			_, err := srv.GetUnreadCount(bgCtx, &notificationv1.GetUnreadCountRequest{UserId: uuid.NewString()})
			return err
		}},
		{"GetUnreadCount/invalid_user_id", codes.InvalidArgument, func() error {
			_, err := srv.GetUnreadCount(ctx, &notificationv1.GetUnreadCountRequest{UserId: "not-a-uuid"})
			return err
		}},
		{"MarkNotificationRead/invalid_notification_id", codes.InvalidArgument, func() error {
			_, err := srv.MarkNotificationRead(ctx, &notificationv1.MarkNotificationReadRequest{NotificationId: "x", UserId: uuid.NewString()})
			return err
		}},
		{"MarkNotificationRead/invalid_user_id", codes.InvalidArgument, func() error {
			_, err := srv.MarkNotificationRead(ctx, &notificationv1.MarkNotificationReadRequest{NotificationId: uuid.NewString(), UserId: "x"})
			return err
		}},
		{"MarkAllNotificationsRead/invalid_user_id", codes.InvalidArgument, func() error {
			_, err := srv.MarkAllNotificationsRead(ctx, &notificationv1.MarkAllNotificationsReadRequest{UserId: "x"})
			return err
		}},
		{"PinNotification/invalid_notification_id", codes.InvalidArgument, func() error {
			_, err := srv.PinNotification(ctx, &notificationv1.PinNotificationRequest{NotificationId: "x", UserId: uuid.NewString()})
			return err
		}},
		{"UnpinNotification/invalid_notification_id", codes.InvalidArgument, func() error {
			_, err := srv.UnpinNotification(ctx, &notificationv1.UnpinNotificationRequest{NotificationId: "x", UserId: uuid.NewString()})
			return err
		}},
		{"DismissNotification/invalid_notification_id", codes.InvalidArgument, func() error {
			_, err := srv.DismissNotification(ctx, &notificationv1.DismissNotificationRequest{NotificationId: "x", UserId: uuid.NewString()})
			return err
		}},
		{"SnoozeNotification/invalid_notification_id", codes.InvalidArgument, func() error {
			_, err := srv.SnoozeNotification(ctx, &notificationv1.SnoozeNotificationRequest{NotificationId: "x", UserId: uuid.NewString()})
			return err
		}},
		{"SnoozeNotification/missing_until", codes.InvalidArgument, func() error {
			_, err := srv.SnoozeNotification(ctx, &notificationv1.SnoozeNotificationRequest{NotificationId: uuid.NewString(), UserId: uuid.NewString()})
			return err
		}},
		{"GetNotificationPreferences/invalid_user_id", codes.InvalidArgument, func() error {
			_, err := srv.GetNotificationPreferences(ctx, &notificationv1.GetNotificationPreferencesRequest{UserId: "x"})
			return err
		}},
		{"UpdateNotificationPreference/invalid_user_id", codes.InvalidArgument, func() error {
			_, err := srv.UpdateNotificationPreference(ctx, &notificationv1.UpdateNotificationPreferenceRequest{UserId: "x"})
			return err
		}},
		{"MuteResource/invalid_user_id", codes.InvalidArgument, func() error {
			_, err := srv.MuteResource(ctx, &notificationv1.MuteResourceRequest{UserId: "x"})
			return err
		}},
		{"UnmuteResource/invalid_user_id", codes.InvalidArgument, func() error {
			_, err := srv.UnmuteResource(ctx, &notificationv1.UnmuteResourceRequest{UserId: "x", MuteId: uuid.NewString()})
			return err
		}},
		{"UnmuteResource/invalid_mute_id", codes.InvalidArgument, func() error {
			_, err := srv.UnmuteResource(ctx, &notificationv1.UnmuteResourceRequest{UserId: uuid.NewString(), MuteId: "x"})
			return err
		}},
		{"ListMutedResources/invalid_user_id", codes.InvalidArgument, func() error {
			_, err := srv.ListMutedResources(ctx, &notificationv1.ListMutedResourcesRequest{UserId: "x"})
			return err
		}},
		{"GetQuietHours/invalid_user_id", codes.InvalidArgument, func() error {
			_, err := srv.GetQuietHours(ctx, &notificationv1.GetQuietHoursRequest{UserId: "x"})
			return err
		}},
		{"ToggleManualDND/invalid_user_id", codes.InvalidArgument, func() error {
			_, err := srv.ToggleManualDND(ctx, &notificationv1.ToggleManualDNDRequest{UserId: "x"})
			return err
		}},

		// Integration surface without integrationRepo/linkService configured.
		{"ListIntegrationConfigs/unavailable", codes.Unavailable, func() error {
			_, err := srv.ListIntegrationConfigs(ctx, &notificationv1.ListIntegrationConfigsRequest{})
			return err
		}},
		{"GetIntegrationConfig/unavailable", codes.Unavailable, func() error {
			_, err := srv.GetIntegrationConfig(ctx, &notificationv1.GetIntegrationConfigRequest{Platform: "slack"})
			return err
		}},
		{"CreateIntegrationConfig/unavailable", codes.Unavailable, func() error {
			_, err := srv.CreateIntegrationConfig(ctx, &notificationv1.CreateIntegrationConfigRequest{Platform: "slack"})
			return err
		}},
		{"UpdateIntegrationConfig/unavailable", codes.Unavailable, func() error {
			_, err := srv.UpdateIntegrationConfig(ctx, &notificationv1.UpdateIntegrationConfigRequest{Id: uuid.NewString()})
			return err
		}},
		{"DeleteIntegrationConfig/unavailable", codes.Unavailable, func() error {
			_, err := srv.DeleteIntegrationConfig(ctx, &notificationv1.DeleteIntegrationConfigRequest{Id: uuid.NewString()})
			return err
		}},
		{"ListChannelMappings/unavailable", codes.Unavailable, func() error {
			_, err := srv.ListChannelMappings(ctx, &notificationv1.ListChannelMappingsRequest{ConfigId: uuid.NewString()})
			return err
		}},
		{"CreateChannelMapping/unavailable", codes.Unavailable, func() error {
			_, err := srv.CreateChannelMapping(ctx, &notificationv1.CreateChannelMappingRequest{ConfigId: uuid.NewString()})
			return err
		}},
		{"UpdateChannelMapping/unavailable", codes.Unavailable, func() error {
			_, err := srv.UpdateChannelMapping(ctx, &notificationv1.UpdateChannelMappingRequest{Id: uuid.NewString()})
			return err
		}},
		{"DeleteChannelMapping/unavailable", codes.Unavailable, func() error {
			_, err := srv.DeleteChannelMapping(ctx, &notificationv1.DeleteChannelMappingRequest{Id: uuid.NewString()})
			return err
		}},
		{"LinkAccount/unavailable", codes.Unavailable, func() error {
			_, err := srv.LinkAccount(ctx, &notificationv1.LinkAccountRequest{Token: "t", UserId: uuid.NewString()})
			return err
		}},
		{"UnlinkAccount/unavailable", codes.Unavailable, func() error {
			_, err := srv.UnlinkAccount(ctx, &notificationv1.UnlinkAccountRequest{Platform: "slack", UserId: uuid.NewString()})
			return err
		}},
		{"GetAccountLinkStatus/unavailable", codes.Unavailable, func() error {
			_, err := srv.GetAccountLinkStatus(ctx, &notificationv1.GetAccountLinkStatusRequest{UserId: uuid.NewString()})
			return err
		}},

		// Integration surface configured, but request itself is invalid.
		{"GetIntegrationConfig/invalid_config_id", codes.InvalidArgument, func() error {
			_, err := integSrv.CreateChannelMapping(ctx, &notificationv1.CreateChannelMappingRequest{ConfigId: "not-a-uuid"})
			return err
		}},
		{"CreateIntegrationConfig/invalid_platform", codes.InvalidArgument, func() error {
			_, err := integSrv.CreateIntegrationConfig(ctx, &notificationv1.CreateIntegrationConfigRequest{Platform: "discord", CreatedBy: uuid.NewString()})
			return err
		}},
		{"CreateIntegrationConfig/invalid_created_by", codes.InvalidArgument, func() error {
			_, err := integSrv.CreateIntegrationConfig(ctx, &notificationv1.CreateIntegrationConfigRequest{Platform: "slack", CreatedBy: "not-a-uuid"})
			return err
		}},
		{"UpdateIntegrationConfig/invalid_id", codes.InvalidArgument, func() error {
			_, err := integSrv.UpdateIntegrationConfig(ctx, &notificationv1.UpdateIntegrationConfigRequest{Id: "not-a-uuid"})
			return err
		}},
		{"UpdateIntegrationConfig/not_found", codes.NotFound, func() error {
			_, err := integSrv.UpdateIntegrationConfig(ctx, &notificationv1.UpdateIntegrationConfigRequest{Id: uuid.NewString()})
			return err
		}},
		{"DeleteIntegrationConfig/invalid_id", codes.InvalidArgument, func() error {
			_, err := integSrv.DeleteIntegrationConfig(ctx, &notificationv1.DeleteIntegrationConfigRequest{Id: "not-a-uuid"})
			return err
		}},
		{"ListChannelMappings/invalid_config_id", codes.InvalidArgument, func() error {
			_, err := integSrv.ListChannelMappings(ctx, &notificationv1.ListChannelMappingsRequest{ConfigId: "not-a-uuid"})
			return err
		}},
		{"CreateChannelMapping/invalid_config_id", codes.InvalidArgument, func() error {
			_, err := integSrv.CreateChannelMapping(ctx, &notificationv1.CreateChannelMappingRequest{ConfigId: "not-a-uuid"})
			return err
		}},
		{"UpdateChannelMapping/invalid_id", codes.InvalidArgument, func() error {
			_, err := integSrv.UpdateChannelMapping(ctx, &notificationv1.UpdateChannelMappingRequest{Id: "not-a-uuid"})
			return err
		}},
		{"UpdateChannelMapping/not_found", codes.NotFound, func() error {
			_, err := integSrv.UpdateChannelMapping(ctx, &notificationv1.UpdateChannelMappingRequest{Id: uuid.NewString()})
			return err
		}},
		{"DeleteChannelMapping/invalid_id", codes.InvalidArgument, func() error {
			_, err := integSrv.DeleteChannelMapping(ctx, &notificationv1.DeleteChannelMappingRequest{Id: "not-a-uuid"})
			return err
		}},
		{"LinkAccount/invalid_user_id", codes.InvalidArgument, func() error {
			_, err := integSrv.LinkAccount(ctx, &notificationv1.LinkAccountRequest{Token: "t", UserId: "x"})
			return err
		}},
		{"LinkAccount/unknown_token", codes.NotFound, func() error {
			_, err := integSrv.LinkAccount(ctx, &notificationv1.LinkAccountRequest{Token: "does-not-exist", UserId: uuid.NewString()})
			return err
		}},
		{"UnlinkAccount/invalid_user_id", codes.InvalidArgument, func() error {
			_, err := integSrv.UnlinkAccount(ctx, &notificationv1.UnlinkAccountRequest{Platform: "slack", UserId: "x"})
			return err
		}},
		{"GetAccountLinkStatus/invalid_user_id", codes.InvalidArgument, func() error {
			_, err := integSrv.GetAccountLinkStatus(ctx, &notificationv1.GetAccountLinkStatusRequest{UserId: "x"})
			return err
		}},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			requireGRPCCode(t, tt.call(), tt.want)
		})
	}
}

// ============================================================================
// ListEventTypes -- registry passthrough, no tenant needed
// ============================================================================

func TestListEventTypes(t *testing.T) {
	registry := event.NewEventTypeRegistry()
	registry.Register(models.EventType{EventKey: "contact.created", ModuleID: "crm", DisplayName: "Contact Created"})
	registry.Register(models.EventType{EventKey: "invoice.overdue", ModuleID: "biz", DisplayName: "Invoice Overdue"})
	srv := NewNotificationGRPCServer(nil, nil, registry)

	resp, err := srv.ListEventTypes(context.Background(), &notificationv1.ListEventTypesRequest{})
	requireGRPCOK(t, err)
	assert.Len(t, resp.EventTypes, 2)

	moduleID := "crm"
	resp, err = srv.ListEventTypes(context.Background(), &notificationv1.ListEventTypesRequest{ModuleId: &moduleID})
	requireGRPCOK(t, err)
	require.Len(t, resp.EventTypes, 1)
	assert.Equal(t, "contact.created", resp.EventTypes[0].EventKey)
}

// ============================================================================
// ListNotifications -- the case that matters: user A never sees user B's rows
// ============================================================================

func TestListNotifications_ScopedToRequestingUser(t *testing.T) {
	notifRepo := newStubNotifRepo()
	tenantID := uuid.New()
	userA, userB := uuid.New(), uuid.New()

	notifRepo.seed(&models.Notification{ID: uuid.New(), TenantID: tenantID, UserID: userA, Title: "for A", CreatedAt: time.Now()})
	notifRepo.seed(&models.Notification{ID: uuid.New(), TenantID: tenantID, UserID: userB, Title: "for B", CreatedAt: time.Now()})

	srv := newNotificationServerWithRepos(notifRepo, newStubPrefRepo())
	ctx := helpdeskTenantContext(tenantID)

	resp, err := srv.ListNotifications(ctx, &notificationv1.ListNotificationsRequest{UserId: userA.String()})
	requireGRPCOK(t, err)
	require.Len(t, resp.Notifications, 1)
	assert.Equal(t, "for A", resp.Notifications[0].Title)
	assert.Equal(t, int32(1), resp.Total)
}

func TestListNotifications_CrossTenantYieldsEmpty(t *testing.T) {
	notifRepo := newStubNotifRepo()
	userA := uuid.New()
	notifRepo.seed(&models.Notification{ID: uuid.New(), TenantID: uuid.New(), UserID: userA, Title: "other tenant", CreatedAt: time.Now()})

	srv := newNotificationServerWithRepos(notifRepo, newStubPrefRepo())
	ctx := helpdeskTenantContext(uuid.New()) // different tenant than the seeded row

	resp, err := srv.ListNotifications(ctx, &notificationv1.ListNotificationsRequest{UserId: userA.String()})
	requireGRPCOK(t, err)
	assert.Empty(t, resp.Notifications)
	assert.Equal(t, int32(0), resp.Total)
}

// ============================================================================
// Ownership guard on mutations -- MarkRead of another user's notification
// ============================================================================

func TestMarkNotificationRead_RejectsOtherUsersNotification(t *testing.T) {
	notifRepo := newStubNotifRepo()
	tenantID := uuid.New()
	owner, requester := uuid.New(), uuid.New()
	notifID := uuid.New()
	notifRepo.seed(&models.Notification{ID: notifID, TenantID: tenantID, UserID: owner, Title: "t", CreatedAt: time.Now()})

	srv := newNotificationServerWithRepos(notifRepo, newStubPrefRepo())
	ctx := helpdeskTenantContext(tenantID)

	_, err := srv.MarkNotificationRead(ctx, &notificationv1.MarkNotificationReadRequest{
		NotificationId: notifID.String(),
		UserId:         requester.String(),
	})
	requireGRPCCode(t, err, codes.PermissionDenied)
}

func TestSnoozeNotification_HappyPathAndPastDeadline(t *testing.T) {
	notifRepo := newStubNotifRepo()
	tenantID, userID := uuid.New(), uuid.New()
	notifID := uuid.New()
	notifRepo.seed(&models.Notification{ID: notifID, TenantID: tenantID, UserID: userID, Title: "t", CreatedAt: time.Now()})

	srv := newNotificationServerWithRepos(notifRepo, newStubPrefRepo())
	ctx := helpdeskTenantContext(tenantID)

	future := time.Now().Add(time.Hour)
	resp, err := srv.SnoozeNotification(ctx, &notificationv1.SnoozeNotificationRequest{
		NotificationId: notifID.String(),
		UserId:         userID.String(),
		Until:          timestamppb.New(future),
	})
	requireGRPCOK(t, err)
	assert.Equal(t, notifID.String(), resp.Id)

	_, err = srv.SnoozeNotification(ctx, &notificationv1.SnoozeNotificationRequest{
		NotificationId: notifID.String(),
		UserId:         userID.String(),
		Until:          timestamppb.New(time.Now().Add(-time.Hour)),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestMarkNotificationRead_HappyPath(t *testing.T) {
	notifRepo := newStubNotifRepo()
	tenantID := uuid.New()
	userID := uuid.New()
	notifID := uuid.New()
	notifRepo.seed(&models.Notification{ID: notifID, TenantID: tenantID, UserID: userID, Title: "t", CreatedAt: time.Now()})

	srv := newNotificationServerWithRepos(notifRepo, newStubPrefRepo())
	ctx := helpdeskTenantContext(tenantID)

	resp, err := srv.MarkNotificationRead(ctx, &notificationv1.MarkNotificationReadRequest{
		NotificationId: notifID.String(),
		UserId:         userID.String(),
	})
	requireGRPCOK(t, err)
	assert.True(t, resp.Notification.IsRead)
}

func TestMarkAllNotificationsRead_HappyPath(t *testing.T) {
	notifRepo := newStubNotifRepo()
	tenantID, userID := uuid.New(), uuid.New()
	notifRepo.seed(&models.Notification{ID: uuid.New(), TenantID: tenantID, UserID: userID, ModuleID: "crm", Title: "a", CreatedAt: time.Now()})
	notifRepo.seed(&models.Notification{ID: uuid.New(), TenantID: tenantID, UserID: userID, ModuleID: "hr", Title: "b", CreatedAt: time.Now()})

	srv := newNotificationServerWithRepos(notifRepo, newStubPrefRepo())
	ctx := helpdeskTenantContext(tenantID)

	resp, err := srv.MarkAllNotificationsRead(ctx, &notificationv1.MarkAllNotificationsReadRequest{UserId: userID.String()})
	requireGRPCOK(t, err)
	assert.Equal(t, int32(2), resp.UpdatedCount)
}

func TestPinUnpinDismissNotification_HappyPath(t *testing.T) {
	notifRepo := newStubNotifRepo()
	tenantID, userID := uuid.New(), uuid.New()
	notifID := uuid.New()
	notifRepo.seed(&models.Notification{ID: notifID, TenantID: tenantID, UserID: userID, Title: "t", CreatedAt: time.Now()})

	srv := newNotificationServerWithRepos(notifRepo, newStubPrefRepo())
	ctx := helpdeskTenantContext(tenantID)

	pinResp, err := srv.PinNotification(ctx, &notificationv1.PinNotificationRequest{NotificationId: notifID.String(), UserId: userID.String()})
	requireGRPCOK(t, err)
	assert.True(t, pinResp.Notification.IsPinned)

	unpinResp, err := srv.UnpinNotification(ctx, &notificationv1.UnpinNotificationRequest{NotificationId: notifID.String(), UserId: userID.String()})
	requireGRPCOK(t, err)
	assert.False(t, unpinResp.Notification.IsPinned)

	dismissResp, err := srv.DismissNotification(ctx, &notificationv1.DismissNotificationRequest{NotificationId: notifID.String(), UserId: userID.String()})
	requireGRPCOK(t, err)
	assert.True(t, dismissResp.Notification.IsDismissed)
}

func TestGetAndUpdateNotificationPreference_HappyPath(t *testing.T) {
	prefRepo := newStubPrefRepo()
	srv := newNotificationServerWithRepos(newStubNotifRepo(), prefRepo)
	tenantID, userID := uuid.New(), uuid.New()
	ctx := helpdeskTenantContext(tenantID)

	eventKey := "contact.created"
	updateResp, err := srv.UpdateNotificationPreference(ctx, &notificationv1.UpdateNotificationPreferenceRequest{
		UserId: userID.String(), EventTypeKey: &eventKey, InApp: true, Email: true, Sound: "chime",
	})
	requireGRPCOK(t, err)
	assert.True(t, updateResp.Preference.InApp)

	listResp, err := srv.GetNotificationPreferences(ctx, &notificationv1.GetNotificationPreferencesRequest{UserId: userID.String()})
	requireGRPCOK(t, err)
	require.Len(t, listResp.Preferences, 1)
	assert.Equal(t, "chime", listResp.Preferences[0].Sound)
}

func TestListMutedResources_HappyPath(t *testing.T) {
	prefRepo := newStubPrefRepo()
	srv := newNotificationServerWithRepos(newStubNotifRepo(), prefRepo)
	tenantID, userID := uuid.New(), uuid.New()
	ctx := helpdeskTenantContext(tenantID)

	_, err := srv.MuteResource(ctx, &notificationv1.MuteResourceRequest{UserId: userID.String(), ModuleId: "crm", ResourceId: "contact-1"})
	requireGRPCOK(t, err)

	listResp, err := srv.ListMutedResources(ctx, &notificationv1.ListMutedResourcesRequest{UserId: userID.String()})
	requireGRPCOK(t, err)
	require.Len(t, listResp.Mutes, 1)
	assert.Equal(t, int32(1), listResp.Total)
}

func TestToggleManualDND_HappyPath(t *testing.T) {
	prefRepo := newStubPrefRepo()
	srv := newNotificationServerWithRepos(newStubNotifRepo(), prefRepo)
	tenantID, userID := uuid.New(), uuid.New()
	ctx := helpdeskTenantContext(tenantID)

	until := timestamppb.New(time.Now().Add(time.Hour))
	resp, err := srv.ToggleManualDND(ctx, &notificationv1.ToggleManualDNDRequest{UserId: userID.String(), Enabled: true, Until: until})
	requireGRPCOK(t, err)
	assert.True(t, resp.QuietHours.ManualDnd)
	require.NotNil(t, resp.QuietHours.ManualDndUntil)

	resp, err = srv.ToggleManualDND(ctx, &notificationv1.ToggleManualDNDRequest{UserId: userID.String(), Enabled: false})
	requireGRPCOK(t, err)
	assert.False(t, resp.QuietHours.ManualDnd)
}

func TestListIntegrationConfigs_HappyPath(t *testing.T) {
	repo := newNotifIntegStub()
	srv := newNotificationServerWithIntegration(repo, nil)
	ctx := helpdeskTenantContext(uuid.New())

	_, err := srv.CreateIntegrationConfig(ctx, &notificationv1.CreateIntegrationConfigRequest{Platform: integration.PlatformSlack, CreatedBy: uuid.NewString()})
	requireGRPCOK(t, err)
	_, err = srv.CreateIntegrationConfig(ctx, &notificationv1.CreateIntegrationConfigRequest{Platform: integration.PlatformTeams, CreatedBy: uuid.NewString()})
	requireGRPCOK(t, err)

	list, err := srv.ListIntegrationConfigs(ctx, &notificationv1.ListIntegrationConfigsRequest{})
	requireGRPCOK(t, err)
	assert.Len(t, list.Configs, 2)
}

// ============================================================================
// GetQuietHours -- the "no record yet" branch returns a disabled default
// instead of a 404. This is the special-case in the handler, not the service.
// ============================================================================

func TestGetQuietHours_NoRecordYetReturnsDisabledDefault(t *testing.T) {
	srv := newNotificationServerWithRepos(newStubNotifRepo(), newStubPrefRepo())
	ctx := helpdeskTenantContext(uuid.New())

	resp, err := srv.GetQuietHours(ctx, &notificationv1.GetQuietHoursRequest{UserId: uuid.NewString()})
	requireGRPCOK(t, err)
	require.NotNil(t, resp.QuietHours)
	assert.False(t, resp.QuietHours.Enabled)
	assert.Empty(t, resp.QuietHours.Id)
}

func TestGetQuietHours_ExistingRecord(t *testing.T) {
	prefRepo := newStubPrefRepo()
	tenantID, userID := uuid.New(), uuid.New()
	prefRepo.quiet[quietKey(tenantID, userID)] = &models.QuietHours{
		ID: uuid.New(), TenantID: tenantID, UserID: userID, Enabled: true,
		StartTime: "22:00", EndTime: "06:00", Timezone: "Europe/Berlin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	srv := newNotificationServerWithRepos(newStubNotifRepo(), prefRepo)
	ctx := helpdeskTenantContext(tenantID)

	resp, err := srv.GetQuietHours(ctx, &notificationv1.GetQuietHoursRequest{UserId: userID.String()})
	requireGRPCOK(t, err)
	assert.True(t, resp.QuietHours.Enabled)
}

// ============================================================================
// UpdateQuietHours -- validation is in the service, mapped by the handler
// ============================================================================

func TestUpdateQuietHours_ValidationErrors(t *testing.T) {
	srv := newNotificationServerWithIntegration(newNotifIntegStub(), nil)
	ctx := helpdeskTenantContext(uuid.New())

	cases := []struct {
		name string
		req  *notificationv1.UpdateQuietHoursRequest
		want codes.Code
	}{
		{"invalid timezone", &notificationv1.UpdateQuietHoursRequest{UserId: uuid.NewString(), StartTime: "22:00", EndTime: "06:00", Timezone: "Mars/Colony"}, codes.InvalidArgument},
		{"invalid start time", &notificationv1.UpdateQuietHoursRequest{UserId: uuid.NewString(), StartTime: "25:99", EndTime: "06:00", Timezone: "UTC"}, codes.InvalidArgument},
		{"invalid day of week", &notificationv1.UpdateQuietHoursRequest{UserId: uuid.NewString(), StartTime: "22:00", EndTime: "06:00", Timezone: "UTC", DaysOfWeek: []int32{9}}, codes.InvalidArgument},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := srv.UpdateQuietHours(ctx, tt.req)
			requireGRPCCode(t, err, tt.want)
		})
	}
}

// ============================================================================
// MuteResource -- double-mute is rejected, matching the frontend's mute toggle
// ============================================================================

func TestMuteResource_AlreadyMutedRejected(t *testing.T) {
	prefRepo := newStubPrefRepo()
	srv := newNotificationServerWithRepos(newStubNotifRepo(), prefRepo)
	tenantID, userID := uuid.New(), uuid.New()
	ctx := helpdeskTenantContext(tenantID)

	req := &notificationv1.MuteResourceRequest{UserId: userID.String(), ModuleId: "crm", ResourceId: "contact-1"}
	_, err := srv.MuteResource(ctx, req)
	requireGRPCOK(t, err)

	_, err = srv.MuteResource(ctx, req)
	requireGRPCCode(t, err, codes.AlreadyExists)
}

func TestUnmuteResource_CrossUserRejected(t *testing.T) {
	prefRepo := newStubPrefRepo()
	srv := newNotificationServerWithRepos(newStubNotifRepo(), prefRepo)
	tenantID, owner, other := uuid.New(), uuid.New(), uuid.New()
	ctx := helpdeskTenantContext(tenantID)

	muteResp, err := srv.MuteResource(ctx, &notificationv1.MuteResourceRequest{UserId: owner.String(), ModuleId: "crm", ResourceId: "contact-1"})
	requireGRPCOK(t, err)

	_, err = srv.UnmuteResource(ctx, &notificationv1.UnmuteResourceRequest{UserId: other.String(), MuteId: muteResp.Mute.Id})
	requireGRPCCode(t, err, codes.NotFound)
}

// ============================================================================
// Integration config / channel mapping CRUD -- happy paths against the stub
// ============================================================================

func TestIntegrationConfigCRUD_HappyPath(t *testing.T) {
	repo := newNotifIntegStub()
	srv := newNotificationServerWithIntegration(repo, nil)
	tenantID := uuid.New()
	ctx := helpdeskTenantContext(tenantID)

	created, err := srv.CreateIntegrationConfig(ctx, &notificationv1.CreateIntegrationConfigRequest{
		Platform: integration.PlatformSlack, CredentialsVaultKey: "vault/slack", CreatedBy: uuid.NewString(),
	})
	requireGRPCOK(t, err)
	require.False(t, created.Config.IsActive) // starts inactive until tested

	got, err := srv.GetIntegrationConfig(ctx, &notificationv1.GetIntegrationConfigRequest{Platform: integration.PlatformSlack})
	requireGRPCOK(t, err)
	assert.Equal(t, created.Config.Id, got.Config.Id)

	newKey := "vault/slack-2"
	updated, err := srv.UpdateIntegrationConfig(ctx, &notificationv1.UpdateIntegrationConfigRequest{
		Id: created.Config.Id, IsActive: true, CredentialsVaultKey: &newKey, Metadata: "{}",
	})
	requireGRPCOK(t, err)
	assert.True(t, updated.Config.IsActive)

	_, err = srv.DeleteIntegrationConfig(ctx, &notificationv1.DeleteIntegrationConfigRequest{Id: created.Config.Id})
	requireGRPCOK(t, err)

	_, err = srv.GetIntegrationConfig(ctx, &notificationv1.GetIntegrationConfigRequest{Platform: integration.PlatformSlack})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestChannelMappingCRUD_HappyPath(t *testing.T) {
	repo := newNotifIntegStub()
	srv := newNotificationServerWithIntegration(repo, nil)
	ctx := helpdeskTenantContext(uuid.New())
	configID := uuid.NewString()

	created, err := srv.CreateChannelMapping(ctx, &notificationv1.CreateChannelMappingRequest{
		ConfigId: configID, ChannelId: "C1", ChannelName: "#general", Modules: []string{"crm"},
	})
	requireGRPCOK(t, err)

	list, err := srv.ListChannelMappings(ctx, &notificationv1.ListChannelMappingsRequest{ConfigId: configID})
	requireGRPCOK(t, err)
	require.Len(t, list.Mappings, 1)

	updated, err := srv.UpdateChannelMapping(ctx, &notificationv1.UpdateChannelMappingRequest{
		Id: created.Mapping.Id, ChannelId: "C1", ChannelName: "#general-renamed", Modules: []string{"crm", "hr"}, IsActive: false,
	})
	requireGRPCOK(t, err)
	assert.Equal(t, "#general-renamed", updated.Mapping.ChannelName)
	assert.False(t, updated.Mapping.IsActive)

	_, err = srv.DeleteChannelMapping(ctx, &notificationv1.DeleteChannelMappingRequest{Id: created.Mapping.Id})
	requireGRPCOK(t, err)

	list, err = srv.ListChannelMappings(ctx, &notificationv1.ListChannelMappingsRequest{ConfigId: configID})
	requireGRPCOK(t, err)
	assert.Empty(t, list.Mappings)
}

// ============================================================================
// Account linking -- full flow through the real AccountLinkService
// ============================================================================

func TestLinkAccount_HappyPathAndDomainErrors(t *testing.T) {
	repo := newNotifIntegStub()
	linkSvc := integration.NewAccountLinkService(repo)
	srv := newNotificationServerWithIntegration(repo, linkSvc)
	ctx := helpdeskTenantContext(uuid.New())
	userID := uuid.New()

	rawToken, err := linkSvc.GenerateLinkToken(ctx, integration.PlatformSlack, "U-EXTERNAL-1")
	require.NoError(t, err)

	resp, err := srv.LinkAccount(ctx, &notificationv1.LinkAccountRequest{Token: rawToken, UserId: userID.String()})
	requireGRPCOK(t, err)
	assert.Equal(t, integration.PlatformSlack, resp.Platform)

	// Re-using the same (now-used) token must fail as AlreadyExists.
	_, err = srv.LinkAccount(ctx, &notificationv1.LinkAccountRequest{Token: rawToken, UserId: userID.String()})
	requireGRPCCode(t, err, codes.AlreadyExists)

	status, err := srv.GetAccountLinkStatus(ctx, &notificationv1.GetAccountLinkStatusRequest{UserId: userID.String()})
	requireGRPCOK(t, err)
	require.Len(t, status.Links, 1)
	assert.Equal(t, integration.PlatformSlack, status.Links[0].Platform)

	_, err = srv.UnlinkAccount(ctx, &notificationv1.UnlinkAccountRequest{Platform: integration.PlatformSlack, UserId: userID.String()})
	requireGRPCOK(t, err)

	status, err = srv.GetAccountLinkStatus(ctx, &notificationv1.GetAccountLinkStatusRequest{UserId: userID.String()})
	requireGRPCOK(t, err)
	assert.Empty(t, status.Links)

	// Unlinking again finds nothing.
	_, err = srv.UnlinkAccount(ctx, &notificationv1.UnlinkAccountRequest{Platform: integration.PlatformSlack, UserId: userID.String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestLinkAccount_ExpiredToken(t *testing.T) {
	repo := newNotifIntegStub()
	linkSvc := integration.NewAccountLinkService(repo)
	srv := newNotificationServerWithIntegration(repo, linkSvc)
	ctx := helpdeskTenantContext(uuid.New())

	rawToken, err := linkSvc.GenerateLinkToken(ctx, integration.PlatformTeams, "U-EXTERNAL-2")
	require.NoError(t, err)

	// Force the stored token into the past without touching the service.
	for _, tok := range repo.tokens {
		tok.ExpiresAt = time.Now().Add(-time.Minute)
	}

	_, err = srv.LinkAccount(ctx, &notificationv1.LinkAccountRequest{Token: rawToken, UserId: uuid.NewString()})
	requireGRPCCode(t, err, codes.DeadlineExceeded)
}
