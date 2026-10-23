package formulare

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kmuhub/kmuhub/internal/idempotency"
)

// memIdempotency is an in-memory idempotency.Repository with the real
// Reserve semantics (fresh / in flight / replay / conflict).
type memIdempotency struct {
	mu   sync.Mutex
	recs map[string]*idempotency.Record
}

func newMemIdempotency() *memIdempotency {
	return &memIdempotency{recs: map[string]*idempotency.Record{}}
}

func (m *memIdempotency) id(tenantID uuid.UUID, key string) string { return tenantID.String() + "|" + key }

func (m *memIdempotency) Reserve(_ context.Context, key string, tenantID, userID uuid.UUID, method, path, hash string) (*idempotency.Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.recs[m.id(tenantID, key)]
	if !ok {
		m.recs[m.id(tenantID, key)] = &idempotency.Record{Key: key, TenantID: tenantID, UserID: userID, Method: method, Path: path, RequestHash: hash}
		return nil, nil
	}
	if rec.RequestHash != hash {
		return nil, idempotency.ErrConflict
	}
	if rec.CompletedAt == nil {
		return nil, idempotency.ErrInFlight
	}
	cp := *rec
	return &cp, nil
}

func (m *memIdempotency) Get(_ context.Context, tenantID uuid.UUID, key string) (*idempotency.Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.recs[m.id(tenantID, key)]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	cp := *rec
	return &cp, nil
}

func (m *memIdempotency) Complete(_ context.Context, tenantID uuid.UUID, key string, status int, body []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.recs[m.id(tenantID, key)]
	if !ok {
		return idempotency.ErrKeyMissing
	}
	now := time.Now()
	rec.ResponseStatus = &status
	rec.ResponseBody = body
	rec.CompletedAt = &now
	return nil
}

func (m *memIdempotency) Cleanup(context.Context) (int, error)                { return 0, nil }
func (m *memIdempotency) CleanupWithLock(context.Context, int64) (int, error) { return 0, nil }

func newIdempotentSvc() (*Service, *stubRepo, *memIdempotency) {
	svc, repo := newSvc()
	idem := newMemIdempotency()
	svc.SetIdempotency(idem)
	return svc, repo, idem
}

func TestSubmitByShareToken_SameKeyReplaysInsteadOfWritingTwice(t *testing.T) {
	svc, repo, _ := newIdempotentSvc()
	link := mintLink(t, svc, seedPublicSchema(t, repo, `[]`).ID)
	answers := []byte(`{"f1":"x"}`)

	first, err := svc.SubmitByShareToken(context.Background(), link.Token, answers, "", "retry-1")
	if err != nil {
		t.Fatalf("first submit: %v", err)
	}
	if first.Replayed {
		t.Fatal("first submit must not be a replay")
	}
	second, err := svc.SubmitByShareToken(context.Background(), link.Token, answers, "", "retry-1")
	if err != nil {
		t.Fatalf("second submit: %v", err)
	}
	if !second.Replayed || second.SubmissionID != first.SubmissionID {
		t.Fatalf("second = %+v, want replay of %s", second, first.SubmissionID)
	}
	if second.TenantID != testTenant || second.FormSchemaID != first.FormSchemaID {
		t.Fatalf("replay lost tenant/schema: %+v", second)
	}
	if len(repo.submissions) != 1 {
		t.Fatalf("submissions = %d, want 1", len(repo.submissions))
	}
	if got := repo.shareLinks[link.ID].SubmissionCount; got != 1 {
		t.Fatalf("submission_count = %d, want 1", got)
	}
}

func TestSubmitByShareToken_WithoutKeyEveryCallWrites(t *testing.T) {
	svc, repo, _ := newIdempotentSvc()
	link := mintLink(t, svc, seedPublicSchema(t, repo, `[]`).ID)

	for i := 0; i < 2; i++ {
		if _, err := svc.SubmitByShareToken(context.Background(), link.Token, []byte(`{"f1":"x"}`), "", ""); err != nil {
			t.Fatalf("submit %d: %v", i, err)
		}
	}
	if len(repo.submissions) != 2 {
		t.Fatalf("submissions = %d, want 2", len(repo.submissions))
	}
}

func TestSubmitByShareToken_SameKeyOtherAnswersIsRejected(t *testing.T) {
	svc, repo, _ := newIdempotentSvc()
	link := mintLink(t, svc, seedPublicSchema(t, repo, `[]`).ID)

	if _, err := svc.SubmitByShareToken(context.Background(), link.Token, []byte(`{"f1":"a"}`), "", "k"); err != nil {
		t.Fatalf("first submit: %v", err)
	}
	_, err := svc.SubmitByShareToken(context.Background(), link.Token, []byte(`{"f1":"b"}`), "", "k")
	if !errors.Is(err, ErrIdempotencyKeyReused) {
		t.Fatalf("err = %v, want ErrIdempotencyKeyReused", err)
	}
	if len(repo.submissions) != 1 {
		t.Fatalf("submissions = %d, want 1", len(repo.submissions))
	}
}

func TestSubmitByShareToken_KeyInFlightIsReportedNotWritten(t *testing.T) {
	svc, repo, idem := newIdempotentSvc()
	link := mintLink(t, svc, seedPublicSchema(t, repo, `[]`).ID)
	answers := []byte(`{"f1":"x"}`)

	// A first request holds the key but has not completed.
	held := newSubmitIdempotency(link, "k", answers)
	if _, err := idem.Reserve(context.Background(), held.key, held.tenantID, held.linkID, "POST", "p", held.hash); err != nil {
		t.Fatalf("seed reservation: %v", err)
	}

	_, err := svc.SubmitByShareToken(context.Background(), link.Token, answers, "", "k")
	if !errors.Is(err, ErrIdempotencyInFlight) {
		t.Fatalf("err = %v, want ErrIdempotencyInFlight", err)
	}
	if len(repo.submissions) != 0 {
		t.Fatalf("submissions = %d, want 0", len(repo.submissions))
	}
}

func TestSubmitByShareToken_ReplaySurvivesAnExhaustedQuota(t *testing.T) {
	svc, repo, _ := newIdempotentSvc()
	schema := seedPublicSchema(t, repo, `[]`)
	one := 1
	link, err := svc.CreateShareLink(context.Background(), CreateShareLinkInput{
		TenantID: testTenant, FormSchemaID: schema.ID, MaxSubmissions: &one,
	})
	if err != nil {
		t.Fatalf("CreateShareLink: %v", err)
	}
	answers := []byte(`{"a":1}`)

	first, err := svc.SubmitByShareToken(context.Background(), link.Token, answers, "", "k")
	if err != nil {
		t.Fatalf("first submit: %v", err)
	}
	// The link is used up now; the visitor's own retry must still get its
	// success, a different visitor still gets the 404.
	replay, err := svc.SubmitByShareToken(context.Background(), link.Token, answers, "", "k")
	if err != nil || !replay.Replayed || replay.SubmissionID != first.SubmissionID {
		t.Fatalf("replay = %+v, err = %v, want replay of %s", replay, err, first.SubmissionID)
	}
	if _, err := svc.SubmitByShareToken(context.Background(), link.Token, answers, "", "other"); !errors.Is(err, ErrShareLinkNotFound) {
		t.Fatalf("other key err = %v, want ErrShareLinkNotFound", err)
	}
}

func TestSubmitByShareToken_SameKeyOnOtherLinkDoesNotCollide(t *testing.T) {
	svc, repo, _ := newIdempotentSvc()
	schema := seedPublicSchema(t, repo, `[]`)
	linkA := mintLink(t, svc, schema.ID)
	linkB := mintLink(t, svc, schema.ID)
	answers := []byte(`{"f1":"x"}`)

	a, err := svc.SubmitByShareToken(context.Background(), linkA.Token, answers, "", "k")
	if err != nil {
		t.Fatalf("link A: %v", err)
	}
	b, err := svc.SubmitByShareToken(context.Background(), linkB.Token, answers, "", "k")
	if err != nil {
		t.Fatalf("link B: %v", err)
	}
	if b.Replayed || b.SubmissionID == a.SubmissionID {
		t.Fatalf("link B got link A's submission: %+v", b)
	}
	if len(repo.submissions) != 2 {
		t.Fatalf("submissions = %d, want 2", len(repo.submissions))
	}
}

func TestSubmitByShareToken_RefusedLinkLeavesNoReservation(t *testing.T) {
	svc, repo, idem := newIdempotentSvc()
	link := mintLink(t, svc, seedPublicSchema(t, repo, `[]`).ID)
	if err := svc.RevokeShareLink(context.Background(), link.ID, testTenant); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	_, err := svc.SubmitByShareToken(context.Background(), link.Token, []byte(`{"f1":"x"}`), "", "k")
	if !errors.Is(err, ErrShareLinkNotFound) {
		t.Fatalf("err = %v, want ErrShareLinkNotFound", err)
	}
	if len(idem.recs) != 0 {
		t.Fatalf("reservations = %d, want 0 (a refused request must not block a later one)", len(idem.recs))
	}
}

func TestSubmitByShareToken_RejectsAnOverLongKey(t *testing.T) {
	svc, repo, _ := newIdempotentSvc()
	link := mintLink(t, svc, seedPublicSchema(t, repo, `[]`).ID)

	key := string(make([]byte, maxIdempotencyKeyLength+1))
	_, err := svc.SubmitByShareToken(context.Background(), link.Token, []byte(`{"f1":"x"}`), "", key)
	if !errors.Is(err, ErrInvalidFields) {
		t.Fatalf("err = %v, want ErrInvalidFields", err)
	}
}
