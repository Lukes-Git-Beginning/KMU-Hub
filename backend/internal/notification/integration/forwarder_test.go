package integration

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// fakeRepository is an in-memory Repository stand-in. Only the methods the
// forwarder actually calls (ListConfigs, ListMappingsByConfig, UpdateMapping,
// LogDelivery) carry real behavior; the rest exist to satisfy the interface.
type fakeRepository struct {
	mu sync.Mutex

	configs          []*IntegrationConfig
	mappingsByConfig map[uuid.UUID][]*ChannelMapping
	listConfigsErr   error

	updatedMappings  []*ChannelMapping
	updateMappingErr error

	deliveryLogs   []*DeliveryLogEntry
	logDeliveryErr error
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{mappingsByConfig: make(map[uuid.UUID][]*ChannelMapping)}
}

func (f *fakeRepository) CreateConfig(context.Context, *IntegrationConfig) error { return nil }
func (f *fakeRepository) GetConfigByPlatform(context.Context, string) (*IntegrationConfig, error) {
	return nil, nil
}
func (f *fakeRepository) UpdateConfig(context.Context, *IntegrationConfig) error { return nil }
func (f *fakeRepository) ListConfigs(context.Context) ([]*IntegrationConfig, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listConfigsErr != nil {
		return nil, f.listConfigsErr
	}
	return f.configs, nil
}
func (f *fakeRepository) DeleteConfig(context.Context, uuid.UUID) error { return nil }

func (f *fakeRepository) CreateMapping(context.Context, *ChannelMapping) error { return nil }
func (f *fakeRepository) GetMapping(context.Context, uuid.UUID) (*ChannelMapping, error) {
	return nil, nil
}
func (f *fakeRepository) ListMappingsByConfig(_ context.Context, configID uuid.UUID) ([]*ChannelMapping, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.mappingsByConfig[configID], nil
}
func (f *fakeRepository) ListActiveMappingsForModule(context.Context, string) ([]*ChannelMapping, error) {
	return nil, nil
}
func (f *fakeRepository) UpdateMapping(_ context.Context, m *ChannelMapping) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateMappingErr != nil {
		return f.updateMappingErr
	}
	f.updatedMappings = append(f.updatedMappings, m)
	return nil
}
func (f *fakeRepository) DeleteMapping(context.Context, uuid.UUID) error { return nil }

func (f *fakeRepository) CreateAccountLink(context.Context, *AccountLink) error { return nil }
func (f *fakeRepository) GetAccountLink(context.Context, string, string) (*AccountLink, error) {
	return nil, nil
}
func (f *fakeRepository) GetAccountLinkByKMUHubUser(context.Context, string, uuid.UUID) (*AccountLink, error) {
	return nil, nil
}
func (f *fakeRepository) DeleteAccountLink(context.Context, uuid.UUID) error { return nil }

func (f *fakeRepository) CreateLinkToken(context.Context, *LinkToken) error { return nil }
func (f *fakeRepository) GetLinkTokenByHash(context.Context, string) (*LinkToken, error) {
	return nil, nil
}
func (f *fakeRepository) MarkLinkTokenUsed(context.Context, uuid.UUID) error { return nil }
func (f *fakeRepository) CleanupExpiredTokens(context.Context) (int, error)  { return 0, nil }

func (f *fakeRepository) LogDelivery(_ context.Context, entry *DeliveryLogEntry) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.logDeliveryErr != nil {
		return f.logDeliveryErr
	}
	f.deliveryLogs = append(f.deliveryLogs, entry)
	return nil
}
func (f *fakeRepository) GetRecentFailures(context.Context, uuid.UUID, int) ([]*DeliveryLogEntry, error) {
	return nil, nil
}
func (f *fakeRepository) CleanupOldLogs(context.Context, time.Time) (int, error) { return 0, nil }

func (f *fakeRepository) snapshot() (updated []*ChannelMapping, logs []*DeliveryLogEntry) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*ChannelMapping{}, f.updatedMappings...), append([]*DeliveryLogEntry{}, f.deliveryLogs...)
}

// fakePoster is a scripted PlatformPoster: returns a canned result/error pair.
type fakePoster struct {
	calls  int
	result *DeliveryResult
	err    error
}

func (p *fakePoster) PostNotification(context.Context, *ChannelMapping, *models.Notification, []ActionType) (*DeliveryResult, error) {
	p.calls++
	return p.result, p.err
}

func newTestMapping(channelID string, modules []string) *ChannelMapping {
	return &ChannelMapping{
		ID:          uuid.New(),
		ConfigID:    uuid.New(),
		ChannelID:   channelID,
		ChannelName: channelID,
		Modules:     modules,
		IsActive:    true,
	}
}

// --- selectMostSpecific -----------------------------------------------------

func TestSelectMostSpecific(t *testing.T) {
	t.Parallel()

	t.Run("exact match wins over wildcard", func(t *testing.T) {
		exact := newTestMapping("C-exact", []string{"crm"})
		wildcard := newTestMapping("C-wild", nil)

		got := selectMostSpecific([]*ChannelMapping{wildcard, exact}, "crm")

		if len(got) != 1 || got[0] != exact {
			t.Fatalf("want only the exact match, got %+v", got)
		}
	})

	t.Run("no exact match falls back to wildcard", func(t *testing.T) {
		wildcard := newTestMapping("C-wild", nil)
		other := newTestMapping("C-other", []string{"hr"})

		got := selectMostSpecific([]*ChannelMapping{other, wildcard}, "crm")

		if len(got) != 1 || got[0] != wildcard {
			t.Fatalf("want only the wildcard mapping, got %+v", got)
		}
	})

	t.Run("multiple exact matches on the same channel dedupe to the first", func(t *testing.T) {
		first := newTestMapping("C-dup", []string{"crm", "work"})
		second := newTestMapping("C-dup", []string{"crm"})
		second.ID = first.ID // same logical mapping row, distinct pointer, same ChannelID

		got := selectMostSpecific([]*ChannelMapping{first, second}, "crm")

		if len(got) != 1 || got[0] != first {
			t.Fatalf("want dedup to keep the first match, got %+v", got)
		}
	})

	t.Run("no mappings match either exact or wildcard", func(t *testing.T) {
		other := newTestMapping("C-other", []string{"hr"})

		got := selectMostSpecific([]*ChannelMapping{other}, "crm")

		if len(got) != 0 {
			t.Fatalf("want no selection, got %+v", got)
		}
	})
}

// --- buildActionSet ----------------------------------------------------------

func TestBuildActionSet(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		eventTypeKey string
		want         []ActionType
	}{
		{"hr leave event gets approve/reject", "hr.leave.requested", []ActionType{ActionAcknowledge, ActionApprove, ActionReject}},
		{"finance event gets acknowledge only", "biz.invoice.overdue", []ActionType{ActionAcknowledge}},
		{"work task event gets reply", "work.task.assigned", []ActionType{ActionAcknowledge, ActionReply}},
		{"crm event gets reply", "crm.contact.created", []ActionType{ActionAcknowledge, ActionReply}},
		{"unrecognized event falls back to acknowledge only", "system.unknown.thing", []ActionType{ActionAcknowledge}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := buildActionSet(tc.eventTypeKey)
			if len(got) != len(tc.want) {
				t.Fatalf("buildActionSet(%q) = %v, want %v", tc.eventTypeKey, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("buildActionSet(%q) = %v, want %v", tc.eventTypeKey, got, tc.want)
				}
			}
		})
	}
}

// --- failure tracking ---------------------------------------------------------

func TestForwarder_TrackFailure_DisablesAfterThreshold(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	f := &Forwarder{repo: repo, failures: make(map[uuid.UUID]int)}
	mapping := newTestMapping("C-flaky", []string{"crm"})
	ctx := context.Background()

	for range 10 {
		f.trackFailure(ctx, mapping)
	}
	if updated, _ := repo.snapshot(); len(updated) != 0 {
		t.Fatalf("want no UpdateMapping call at 10 consecutive failures, got %d", len(updated))
	}
	if !mapping.IsActive {
		t.Fatalf("mapping must stay active at 10 failures")
	}

	f.trackFailure(ctx, mapping) // 11th: crosses the >10 threshold

	if mapping.IsActive {
		t.Fatalf("want mapping disabled after 11 consecutive failures")
	}
	updated, _ := repo.snapshot()
	if len(updated) != 1 || updated[0].ID != mapping.ID {
		t.Fatalf("want exactly one UpdateMapping call for the disabled mapping, got %+v", updated)
	}

	f.failureMu.Lock()
	_, stillTracked := f.failures[mapping.ID]
	f.failureMu.Unlock()
	if stillTracked {
		t.Fatalf("want the failure counter reset (deleted) after auto-disable")
	}
}

func TestForwarder_ResetFailures_ClearsCounter(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	f := &Forwarder{repo: repo, failures: make(map[uuid.UUID]int)}
	mapping := newTestMapping("C-recovers", []string{"crm"})
	ctx := context.Background()

	f.trackFailure(ctx, mapping)
	f.trackFailure(ctx, mapping)
	f.failureMu.Lock()
	count := f.failures[mapping.ID]
	f.failureMu.Unlock()
	if count != 2 {
		t.Fatalf("want failure count 2 before reset, got %d", count)
	}

	f.resetFailures(mapping.ID)

	f.failureMu.Lock()
	_, tracked := f.failures[mapping.ID]
	f.failureMu.Unlock()
	if tracked {
		t.Fatalf("want no entry for mapping after resetFailures")
	}
}

// --- dispatchToMapping: the real per-notification decision logic -------------

func TestForwarder_DispatchToMapping(t *testing.T) {
	t.Parallel()

	t.Run("success resets failures and logs sent", func(t *testing.T) {
		t.Parallel()
		repo := newFakeRepository()
		cfg := &IntegrationConfig{ID: uuid.New(), Platform: PlatformSlack, IsActive: true}
		mapping := newTestMapping("C-ok", []string{"crm"})
		mapping.ConfigID = cfg.ID
		repo.configs = []*IntegrationConfig{cfg}
		repo.mappingsByConfig[cfg.ID] = []*ChannelMapping{mapping}

		slack := &fakePoster{result: &DeliveryResult{PlatformMessageID: "msg-1", Success: true}}
		f := &Forwarder{
			repo:     repo,
			slack:    slack,
			limiter:  NewRateLimiter(),
			cache:    NewMappingCache(repo, time.Minute),
			failures: map[uuid.UUID]int{mapping.ID: 3},
		}
		notif := &models.Notification{ID: uuid.New(), ModuleID: "crm", EventTypeKey: "crm.contact.created"}

		f.dispatchToMapping(context.Background(), mapping, notif, buildActionSet(notif.EventTypeKey))

		if slack.calls != 1 {
			t.Fatalf("want PostNotification called once, got %d", slack.calls)
		}
		f.failureMu.Lock()
		_, tracked := f.failures[mapping.ID]
		f.failureMu.Unlock()
		if tracked {
			t.Fatalf("want failure counter cleared on success")
		}
		_, logs := repo.snapshot()
		if len(logs) != 1 || logs[0].Status != DeliveryStatusSent || logs[0].PlatformMessageID != "msg-1" {
			t.Fatalf("want one sent log entry with the platform message id, got %+v", logs)
		}
	})

	t.Run("platform error logs failed and tracks a failure", func(t *testing.T) {
		t.Parallel()
		repo := newFakeRepository()
		cfg := &IntegrationConfig{ID: uuid.New(), Platform: PlatformTeams, IsActive: true}
		mapping := newTestMapping("C-fail", []string{"crm"})
		mapping.ConfigID = cfg.ID
		repo.configs = []*IntegrationConfig{cfg}
		repo.mappingsByConfig[cfg.ID] = []*ChannelMapping{mapping}

		teams := &fakePoster{err: errors.New("webhook unreachable")}
		f := &Forwarder{
			repo:     repo,
			teams:    teams,
			limiter:  NewRateLimiter(),
			cache:    NewMappingCache(repo, time.Minute),
			failures: make(map[uuid.UUID]int),
		}
		notif := &models.Notification{ID: uuid.New(), ModuleID: "crm", EventTypeKey: "crm.contact.created"}

		f.dispatchToMapping(context.Background(), mapping, notif, buildActionSet(notif.EventTypeKey))

		f.failureMu.Lock()
		count := f.failures[mapping.ID]
		f.failureMu.Unlock()
		if count != 1 {
			t.Fatalf("want failure count 1 after a failed delivery, got %d", count)
		}
		_, logs := repo.snapshot()
		if len(logs) != 1 || logs[0].Status != DeliveryStatusFailed {
			t.Fatalf("want one failed log entry, got %+v", logs)
		}
	})

	t.Run("unconfigured platform is a silent no-op, not a panic", func(t *testing.T) {
		t.Parallel()
		repo := newFakeRepository()
		cfg := &IntegrationConfig{ID: uuid.New(), Platform: PlatformSlack, IsActive: true}
		mapping := newTestMapping("C-unconfigured", []string{"crm"})
		mapping.ConfigID = cfg.ID
		repo.configs = []*IntegrationConfig{cfg}
		repo.mappingsByConfig[cfg.ID] = []*ChannelMapping{mapping}

		f := &Forwarder{
			repo:     repo, // slack == nil: platform resolves but no client is wired
			limiter:  NewRateLimiter(),
			cache:    NewMappingCache(repo, time.Minute),
			failures: make(map[uuid.UUID]int),
		}
		notif := &models.Notification{ID: uuid.New(), ModuleID: "crm"}

		f.dispatchToMapping(context.Background(), mapping, notif, nil)

		_, logs := repo.snapshot()
		if len(logs) != 0 {
			t.Fatalf("want no delivery log for an unconfigured platform, got %+v", logs)
		}
	})

	t.Run("rate limited mapping is logged without calling the poster", func(t *testing.T) {
		t.Parallel()
		repo := newFakeRepository()
		cfg := &IntegrationConfig{ID: uuid.New(), Platform: PlatformSlack, IsActive: true}
		mapping := newTestMapping("C-limited", []string{"crm"})
		mapping.ConfigID = cfg.ID
		repo.configs = []*IntegrationConfig{cfg}
		repo.mappingsByConfig[cfg.ID] = []*ChannelMapping{mapping}

		slack := &fakePoster{result: &DeliveryResult{Success: true}}
		limiter := NewRateLimiter()
		limiter.Allow(PlatformSlack, mapping.ChannelID) // consume the only slot in this window

		f := &Forwarder{
			repo:     repo,
			slack:    slack,
			limiter:  limiter,
			cache:    NewMappingCache(repo, time.Minute),
			failures: make(map[uuid.UUID]int),
		}
		notif := &models.Notification{ID: uuid.New(), ModuleID: "crm"}

		f.dispatchToMapping(context.Background(), mapping, notif, nil)

		if slack.calls != 0 {
			t.Fatalf("want PostNotification not called while rate limited, got %d calls", slack.calls)
		}
		_, logs := repo.snapshot()
		if len(logs) != 1 || logs[0].Status != DeliveryStatusRateLimited {
			t.Fatalf("want one rate_limited log entry, got %+v", logs)
		}
	})
}

// --- MappingCache: TTL and the wildcard gap -----------------------------------

func TestMappingCache_TTLRefresh(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	cfg := &IntegrationConfig{ID: uuid.New(), Platform: PlatformSlack, IsActive: true}
	repo.configs = []*IntegrationConfig{cfg}
	v1 := newTestMapping("C-v1", []string{"crm"})
	v1.ConfigID = cfg.ID
	repo.mappingsByConfig[cfg.ID] = []*ChannelMapping{v1}

	ttl := 30 * time.Millisecond
	cache := NewMappingCache(repo, ttl)
	ctx := context.Background()

	got, err := cache.GetMappingsForModule(ctx, "crm")
	if err != nil {
		t.Fatalf("GetMappingsForModule: %v", err)
	}
	if len(got) != 1 || got[0].ChannelID != "C-v1" {
		t.Fatalf("want the initial mapping, got %+v", got)
	}

	// Simulate the underlying data changing without advancing the cache clock.
	v2 := newTestMapping("C-v2", []string{"crm"})
	v2.ConfigID = cfg.ID
	repo.mu.Lock()
	repo.mappingsByConfig[cfg.ID] = []*ChannelMapping{v2}
	repo.mu.Unlock()

	got, err = cache.GetMappingsForModule(ctx, "crm")
	if err != nil {
		t.Fatalf("GetMappingsForModule (still cached): %v", err)
	}
	if len(got) != 1 || got[0].ChannelID != "C-v1" {
		t.Fatalf("want the stale cached mapping before TTL expiry, got %+v", got)
	}

	time.Sleep(ttl + 10*time.Millisecond)

	got, err = cache.GetMappingsForModule(ctx, "crm")
	if err != nil {
		t.Fatalf("GetMappingsForModule (after TTL): %v", err)
	}
	if len(got) != 1 || got[0].ChannelID != "C-v2" {
		t.Fatalf("want the fresh mapping after TTL expiry, got %+v", got)
	}
}

func TestMappingCache_SkipsInactiveConfigsAndMappings(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	activeCfg := &IntegrationConfig{ID: uuid.New(), Platform: PlatformSlack, IsActive: true}
	inactiveCfg := &IntegrationConfig{ID: uuid.New(), Platform: PlatformTeams, IsActive: false}
	repo.configs = []*IntegrationConfig{activeCfg, inactiveCfg}

	activeMapping := newTestMapping("C-active", []string{"crm"})
	activeMapping.ConfigID = activeCfg.ID
	inactiveMapping := newTestMapping("C-inactive", []string{"crm"})
	inactiveMapping.ConfigID = activeCfg.ID
	inactiveMapping.IsActive = false
	repo.mappingsByConfig[activeCfg.ID] = []*ChannelMapping{activeMapping, inactiveMapping}
	// inactiveCfg has mappings too, but the whole config must be skipped first.
	repo.mappingsByConfig[inactiveCfg.ID] = []*ChannelMapping{newTestMapping("C-orphan", []string{"crm"})}

	cache := NewMappingCache(repo, time.Minute)
	got, err := cache.GetMappingsForModule(context.Background(), "crm")
	if err != nil {
		t.Fatalf("GetMappingsForModule: %v", err)
	}
	if len(got) != 1 || got[0].ChannelID != "C-active" {
		t.Fatalf("want only the active mapping under the active config, got %+v", got)
	}

	if _, err := cache.GetConfigForMapping(context.Background(), inactiveCfg.ID); !errors.Is(err, ErrConfigNotFound) {
		t.Fatalf("want ErrConfigNotFound for an inactive config, got %v", err)
	}
}

func TestMappingCache_RefreshPropagatesListConfigsError(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	repo.listConfigsErr = errors.New("db unavailable")
	cache := NewMappingCache(repo, time.Minute)

	if _, err := cache.GetMappingsForModule(context.Background(), "crm"); !errors.Is(err, repo.listConfigsErr) {
		t.Fatalf("want the ListConfigs error propagated, got %v", err)
	}
}

// TestMappingCache_WildcardMappingNeverReturned documents a real bug found
// while writing this coverage: refresh() files a wildcard mapping (empty
// Modules, meaning "all modules") under the literal cache key "*"
// (forwarder.go:370), with a comment promising it is merged in "at query
// time in GetMappingsForModule" - but GetMappingsForModule only ever reads
// c.modules[moduleID] (forwarder.go:293/306), never c.modules["*"]. A
// wildcard channel mapping - the simplest possible setup, "forward every
// notification to this Slack channel" - is loaded into the cache and then
// never handed to any real module lookup. See the new backlog fix unit
// `fix-notification-wildcard-mapping-never-delivered`.
func TestMappingCache_WildcardMappingNeverReturned_DocumentsCurrentGap(t *testing.T) {
	t.Parallel()

	repo := newFakeRepository()
	cfg := &IntegrationConfig{ID: uuid.New(), Platform: PlatformSlack, IsActive: true}
	repo.configs = []*IntegrationConfig{cfg}
	wildcard := newTestMapping("C-everything", nil) // empty Modules == wildcard
	repo.mappingsByConfig[cfg.ID] = []*ChannelMapping{wildcard}

	cache := NewMappingCache(repo, time.Minute)

	got, err := cache.GetMappingsForModule(context.Background(), "crm")
	if err != nil {
		t.Fatalf("GetMappingsForModule: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("documents current gap: expected the wildcard mapping to be lost, got %+v (bug fixed? update this test and the backlog unit)", got)
	}
}
