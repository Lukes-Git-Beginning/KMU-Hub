package meeting

// Covers the repository surface tenant_write_test.go/tenant_isolation_phase2_test.go
// leave open: attendee management, notes, recurring-series instance isolation,
// action items, chat, co-hosts, and breakout rooms/assignments — against the real
// schema, not mocks. See BACKLOG.yml unit c-cov-work-meeting-repo (Lauf 7).

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func seedMeetingFixtureTenant(t *testing.T, pool *pgxpool.Pool, label string) (tenantID, userID uuid.UUID) {
	t.Helper()
	tenantID = uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, label)
	userID = seedMeetingUser(t, pool, tenantID)
	return tenantID, userID
}

func newTestMeeting(tenantID, organizerID uuid.UUID, title string, start time.Time) *Meeting {
	return &Meeting{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Title:          title,
		OrganizerID:    organizerID,
		Status:         MeetingStatusScheduled,
		ScheduledStart: start,
		ScheduledEnd:   start.Add(time.Hour),
		CreatedAt:      start,
		UpdatedAt:      start,
	}
}

// TestAttendees_AddUpdateRemoveAndGet covers Add/Update/Remove/Get across the
// attendee lifecycle, including AddAttendee's ON CONFLICT DO NOTHING idempotency
// and UpdateAttendeeRSVP's not-found error path.
func TestAttendees_AddUpdateRemoveAndGet(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, organizerOwn := seedMeetingFixtureTenant(t, pool, "Meeting Attendees Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", organizerOwn)
	attendeeUser := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("wp-meeting-attendee-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"first_name":    "Ada",
		"last_name":     "Attendee",
	})
	defer testutil.CleanupRow(t, pool, "users", attendeeUser)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	m := newTestMeeting(tenantOwn, organizerOwn, "Attendee-Test", time.Now().UTC())
	if err := repo.CreateMeeting(ctxOwn, m); err != nil {
		t.Fatalf("CreateMeeting: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "meetings", m.ID)

	// UpdateAttendeeRSVP on a non-existent attendee row must return ErrNotFound.
	if err := repo.UpdateAttendeeRSVP(ctxOwn, m.ID, attendeeUser, MeetingRSVPAccepted); !errors.Is(err, ErrNotFound) {
		t.Fatalf("UpdateAttendeeRSVP (not invited): expected ErrNotFound, got %v", err)
	}

	if err := repo.AddAttendee(ctxOwn, m.ID, attendeeUser); err != nil {
		t.Fatalf("AddAttendee: %v", err)
	}
	// Idempotent: a second Add must not error and must not reset rsvp_status.
	if err := repo.UpdateAttendeeRSVP(ctxOwn, m.ID, attendeeUser, MeetingRSVPTentative); err != nil {
		t.Fatalf("UpdateAttendeeRSVP: %v", err)
	}
	if err := repo.AddAttendee(ctxOwn, m.ID, attendeeUser); err != nil {
		t.Fatalf("AddAttendee (duplicate, must be idempotent): %v", err)
	}

	attendees, err := repo.GetAttendees(ctxOwn, m.ID)
	if err != nil {
		t.Fatalf("GetAttendees: %v", err)
	}
	if len(attendees) != 1 {
		t.Fatalf("GetAttendees: expected 1 attendee (dup Add must not insert a second row), got %d", len(attendees))
	}
	if attendees[0].UserID != attendeeUser || attendees[0].RSVPStatus != MeetingRSVPTentative {
		t.Fatalf("GetAttendees: unexpected row %+v", attendees[0])
	}
	if attendees[0].FirstName != "Ada" || attendees[0].LastName != "Attendee" {
		t.Fatalf("GetAttendees: expected denormalized user name to be joined, got first=%q last=%q", attendees[0].FirstName, attendees[0].LastName)
	}

	if err := repo.RemoveAttendee(ctxOwn, m.ID, attendeeUser); err != nil {
		t.Fatalf("RemoveAttendee: %v", err)
	}
	attendees, err = repo.GetAttendees(ctxOwn, m.ID)
	if err != nil {
		t.Fatalf("GetAttendees (after remove): %v", err)
	}
	if len(attendees) != 0 {
		t.Fatalf("GetAttendees (after remove): expected empty, got %d", len(attendees))
	}
}

// TestListMeetings_FiltersAndCrossTenant covers ListMeetings' filter
// combinations (organizer, attendee EXISTS-subquery, status, time range,
// pagination/order) and proves that RLS blocks a foreign-tenant session even
// when the caller explicitly passes the victim's TenantID in the filter —
// ListMeetings has no ctx-derived tenant check of its own, only the RLS
// session and the explicit filter, so both must independently agree.
func TestListMeetings_FiltersAndCrossTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, organizerOwn := seedMeetingFixtureTenant(t, pool, "Meeting List Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", organizerOwn)
	tenantOther, organizerOther := seedMeetingFixtureTenant(t, pool, "Meeting List Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)
	defer testutil.CleanupRow(t, pool, "users", organizerOther)
	attendeeUser := seedMeetingUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", attendeeUser)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	base := time.Now().UTC().Truncate(time.Second)

	scheduled := newTestMeeting(tenantOwn, organizerOwn, "List-Match Scheduled "+uuid.New().String()[:8], base)
	if err := repo.CreateMeeting(ctxOwn, scheduled); err != nil {
		t.Fatalf("CreateMeeting (scheduled): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "meetings", scheduled.ID)
	if err := repo.AddAttendee(ctxOwn, scheduled.ID, attendeeUser); err != nil {
		t.Fatalf("AddAttendee: %v", err)
	}

	cancelled := newTestMeeting(tenantOwn, organizerOwn, "List-Match Cancelled "+uuid.New().String()[:8], base.Add(2*time.Hour))
	cancelled.Status = MeetingStatusCancelled
	if err := repo.CreateMeeting(ctxOwn, cancelled); err != nil {
		t.Fatalf("CreateMeeting (cancelled): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "meetings", cancelled.ID)

	// Foreign-tenant meeting sharing the same organizer/status/title substring —
	// must never appear in tenantOwn's results or leak via the attendee filter.
	foreign := newTestMeeting(tenantOther, organizerOther, "List-Match Scheduled foreign", base)
	if err := repo.CreateMeeting(ctxOther, foreign); err != nil {
		t.Fatalf("CreateMeeting (foreign): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "meetings", foreign.ID)

	results, err := repo.ListMeetings(ctxOwn, MeetingFilter{TenantID: tenantOwn, OrganizerID: &organizerOwn, Limit: 50})
	if err != nil {
		t.Fatalf("ListMeetings (organizer): %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("ListMeetings (organizer): expected 2, got %d (%+v)", len(results), results)
	}
	// ORDER BY scheduled_start DESC.
	if results[0].ID != cancelled.ID || results[1].ID != scheduled.ID {
		t.Fatalf("ListMeetings (organizer): wrong order, got [%s, %s]", results[0].ID, results[1].ID)
	}

	status := MeetingStatusCancelled
	results, err = repo.ListMeetings(ctxOwn, MeetingFilter{TenantID: tenantOwn, Status: &status, Limit: 50})
	if err != nil {
		t.Fatalf("ListMeetings (status): %v", err)
	}
	if len(results) != 1 || results[0].ID != cancelled.ID {
		t.Fatalf("ListMeetings (status=cancelled): expected exactly cancelled, got %+v", results)
	}

	results, err = repo.ListMeetings(ctxOwn, MeetingFilter{TenantID: tenantOwn, AttendeeID: &attendeeUser, Limit: 50})
	if err != nil {
		t.Fatalf("ListMeetings (attendee): %v", err)
	}
	if len(results) != 1 || results[0].ID != scheduled.ID {
		t.Fatalf("ListMeetings (attendee): expected exactly scheduled, got %+v", results)
	}

	results, err = repo.ListMeetings(ctxOwn, MeetingFilter{TenantID: tenantOwn, StartAfter: &cancelled.ScheduledStart, Limit: 50})
	if err != nil {
		t.Fatalf("ListMeetings (startAfter): %v", err)
	}
	if len(results) != 1 || results[0].ID != cancelled.ID {
		t.Fatalf("ListMeetings (startAfter): expected exactly cancelled, got %+v", results)
	}

	before := scheduled.ScheduledStart
	results, err = repo.ListMeetings(ctxOwn, MeetingFilter{TenantID: tenantOwn, StartBefore: &before, Limit: 50})
	if err != nil {
		t.Fatalf("ListMeetings (startBefore): %v", err)
	}
	if len(results) != 1 || results[0].ID != scheduled.ID {
		t.Fatalf("ListMeetings (startBefore): expected exactly scheduled, got %+v", results)
	}

	// Pagination: Limit=1 Offset=1 on the DESC-ordered full set returns the
	// second-newest row (scheduled), not the newest.
	results, err = repo.ListMeetings(ctxOwn, MeetingFilter{TenantID: tenantOwn, Limit: 1, Offset: 1})
	if err != nil {
		t.Fatalf("ListMeetings (paginated): %v", err)
	}
	if len(results) != 1 || results[0].ID != scheduled.ID {
		t.Fatalf("ListMeetings (limit=1 offset=1): expected exactly scheduled, got %+v", results)
	}

	// Cross-tenant: ctxOther's RLS session must block rows even though the
	// filter explicitly asks for tenantOwn's data.
	results, err = repo.ListMeetings(ctxOther, MeetingFilter{TenantID: tenantOwn, Limit: 50})
	if err != nil {
		t.Fatalf("ListMeetings (foreign ctx, victim filter): %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("ListMeetings (foreign ctx, victim filter): expected 0 rows under RLS, got %d", len(results))
	}
}

// TestGetMeetingByRoomName_And_ListStaleMeetings covers the two cross-tenant
// system-context queries, which must be called under WithSystemContext to
// bypass RLS entirely (no tenant filter in the SQL).
func TestGetMeetingByRoomName_And_ListStaleMeetings(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, organizerOwn := seedMeetingFixtureTenant(t, pool, "Meeting System Query Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", organizerOwn)

	repo := NewPostgresRepository(pool)
	sysCtx := testutil.WithSystemCtx(context.Background())
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	room := "room-" + uuid.New().String()[:12]
	past := time.Now().UTC().Add(-3 * time.Hour)
	stale := newTestMeeting(tenantOwn, organizerOwn, "Stale-Test", past)
	stale.Status = MeetingStatusInProgress
	stale.ScheduledEnd = past.Add(30 * time.Minute)
	stale.RoomName = &room
	if err := repo.CreateMeeting(ctxOwn, stale); err != nil {
		t.Fatalf("CreateMeeting: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "meetings", stale.ID)

	got, err := repo.GetMeetingByRoomName(sysCtx, room)
	if err != nil {
		t.Fatalf("GetMeetingByRoomName: %v", err)
	}
	if got.ID != stale.ID {
		t.Fatalf("GetMeetingByRoomName: expected %s, got %s", stale.ID, got.ID)
	}

	if _, err := repo.GetMeetingByRoomName(sysCtx, "does-not-exist-"+uuid.New().String()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetMeetingByRoomName (missing): expected ErrNotFound, got %v", err)
	}

	staleList, err := repo.ListStaleMeetings(sysCtx, time.Now().UTC())
	if err != nil {
		t.Fatalf("ListStaleMeetings: %v", err)
	}
	found := false
	for _, m := range staleList {
		if m.ID == stale.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("ListStaleMeetings: expected to find in_progress meeting past scheduled_end")
	}

	// A cutoff before scheduled_end must not include it.
	staleList, err = repo.ListStaleMeetings(sysCtx, past)
	if err != nil {
		t.Fatalf("ListStaleMeetings (early cutoff): %v", err)
	}
	for _, m := range staleList {
		if m.ID == stale.ID {
			t.Fatalf("ListStaleMeetings (early cutoff): must not include a meeting whose scheduled_end is after cutoff")
		}
	}
}

// TestNotes_SeriesIsolationAndSaveNotesUpsert is the "Serientermin-Ausnahme"
// case: it builds a three-meeting recurring series (one anchor, two dated
// instances sharing recurring_meeting_id), proves that mutating a single
// instance via UpdateMeeting leaves its siblings untouched, and exercises
// GetPreviousMeetingNotes' date-scoped lookup across the series.
//
// It also exercises SaveNotes' upsert behavior against the partial unique
// indexes added in migration 000309 (meeting_notes_meeting_author_public_unique/
// _private_unique, matching SaveNotes' `ON CONFLICT (meeting_id, author_id)
// WHERE is_private = $5` target): a first call creates the row, a second call
// for the same (meeting_id, author_id, is_private) updates it in place instead
// of erroring or duplicating, and a public and a private note for the same
// meeting/author coexist as two separate rows.
func TestNotes_SeriesIsolationAndSaveNotesUpsert(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, organizerOwn := seedMeetingFixtureTenant(t, pool, "Meeting Series Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", organizerOwn)
	tenantOther := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOther, "Meeting Series Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	base := time.Now().UTC().Truncate(time.Second)
	anchor := newTestMeeting(tenantOwn, organizerOwn, "Weekly Sync", base)
	if err := repo.CreateMeeting(ctxOwn, anchor); err != nil {
		t.Fatalf("CreateMeeting (anchor): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "meetings", anchor.ID)

	// SaveNotes upsert: first call creates the public note.
	publicNoteID := uuid.New()
	if err := repo.SaveNotes(ctxOwn, &MeetingNotes{
		ID:        publicNoteID,
		MeetingID: anchor.ID,
		AuthorID:  organizerOwn,
		Content:   "first draft",
		IsPrivate: false,
		CreatedAt: base,
		UpdatedAt: base,
	}); err != nil {
		t.Fatalf("SaveNotes (create public): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "meeting_notes", publicNoteID)

	// Second call for the same (meeting_id, author_id, is_private) updates the
	// existing row in place — new ID is ignored, content is overwritten, no
	// duplicate row is created.
	if err := repo.SaveNotes(ctxOwn, &MeetingNotes{
		ID:        uuid.New(),
		MeetingID: anchor.ID,
		AuthorID:  organizerOwn,
		Content:   "revised draft",
		IsPrivate: false,
		CreatedAt: base,
		UpdatedAt: base.Add(time.Minute),
	}); err != nil {
		t.Fatalf("SaveNotes (update public): %v", err)
	}
	anchorNotes, err := repo.GetAllNotes(ctxOwn, anchor.ID)
	if err != nil {
		t.Fatalf("GetAllNotes (anchor after upsert): %v", err)
	}
	if len(anchorNotes) != 1 || anchorNotes[0].ID != publicNoteID || anchorNotes[0].Content != "revised draft" {
		t.Fatalf("SaveNotes: expected exactly one updated public row, got %+v", anchorNotes)
	}

	// A private note for the same meeting/author is a distinct row, not a
	// conflict with the public one (separate partial unique index).
	privateAnchorNoteID := uuid.New()
	if err := repo.SaveNotes(ctxOwn, &MeetingNotes{
		ID:        privateAnchorNoteID,
		MeetingID: anchor.ID,
		AuthorID:  organizerOwn,
		Content:   "private scratch",
		IsPrivate: true,
		CreatedAt: base,
		UpdatedAt: base,
	}); err != nil {
		t.Fatalf("SaveNotes (create private): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "meeting_notes", privateAnchorNoteID)
	if anchorNotes, err = repo.GetAllNotes(ctxOwn, anchor.ID); err != nil {
		t.Fatalf("GetAllNotes (anchor after private save): %v", err)
	} else if len(anchorNotes) != 1 || anchorNotes[0].ID != publicNoteID {
		t.Fatalf("GetAllNotes: private note must not appear in the public listing, got %+v", anchorNotes)
	}

	inst1 := newTestMeeting(tenantOwn, organizerOwn, "Weekly Sync — Week 1", base.Add(7*24*time.Hour))
	inst1.RecurringMeetingID = &anchor.ID
	inst1.Status = MeetingStatusCompleted
	if err := repo.CreateMeeting(ctxOwn, inst1); err != nil {
		t.Fatalf("CreateMeeting (inst1): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "meetings", inst1.ID)

	inst2 := newTestMeeting(tenantOwn, organizerOwn, "Weekly Sync — Week 2", base.Add(14*24*time.Hour))
	inst2.RecurringMeetingID = &anchor.ID
	inst2.Status = MeetingStatusCompleted
	if err := repo.CreateMeeting(ctxOwn, inst2); err != nil {
		t.Fatalf("CreateMeeting (inst2): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "meetings", inst2.ID)

	note1 := testutil.SeedRow(t, pool, "meeting_notes", map[string]any{
		"tenant_id":  tenantOwn,
		"meeting_id": inst1.ID,
		"author_id":  organizerOwn,
		"content":    "Notes from week 1",
		"is_private": false,
	})
	defer testutil.CleanupRow(t, pool, "meeting_notes", note1)
	note2 := testutil.SeedRow(t, pool, "meeting_notes", map[string]any{
		"tenant_id":  tenantOwn,
		"meeting_id": inst2.ID,
		"author_id":  organizerOwn,
		"content":    "Notes from week 2",
		"is_private": false,
	})
	defer testutil.CleanupRow(t, pool, "meeting_notes", note2)
	// A private note on inst2, dated even later — GetAllNotes must exclude it.
	privateNote := testutil.SeedRow(t, pool, "meeting_notes", map[string]any{
		"tenant_id":  tenantOwn,
		"meeting_id": inst2.ID,
		"author_id":  organizerOwn,
		"content":    "Private scratchpad",
		"is_private": true,
	})
	defer testutil.CleanupRow(t, pool, "meeting_notes", privateNote)

	got, err := repo.GetNotes(ctxOwn, inst1.ID, organizerOwn)
	if err != nil {
		t.Fatalf("GetNotes: %v", err)
	}
	if got.ID != note1 || got.Content != "Notes from week 1" {
		t.Fatalf("GetNotes: unexpected row %+v", got)
	}
	if _, err := repo.GetNotes(ctxOwn, uuid.New(), organizerOwn); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetNotes (missing): expected ErrNotFound, got %v", err)
	}

	all, err := repo.GetAllNotes(ctxOwn, inst2.ID)
	if err != nil {
		t.Fatalf("GetAllNotes: %v", err)
	}
	if len(all) != 1 || all[0].ID != note2 {
		t.Fatalf("GetAllNotes: expected exactly the public note, got %+v", all)
	}

	// GetPreviousMeetingNotes/GetNotes/GetAllNotes carry no tenant_id predicate
	// of their own — RLS alone must block a foreign-tenant session.
	if _, err := repo.GetNotes(ctxOther, inst1.ID, organizerOwn); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetNotes (foreign ctx): expected ErrNotFound under RLS, got %v", err)
	}
	if all, err := repo.GetAllNotes(ctxOther, inst2.ID); err != nil || len(all) != 0 {
		t.Fatalf("GetAllNotes (foreign ctx): expected empty under RLS, got %+v err=%v", all, err)
	}

	// Between week 1 and week 2 — only week 1's note qualifies.
	prev, err := repo.GetPreviousMeetingNotes(ctxOwn, anchor.ID, base.Add(10*24*time.Hour))
	if err != nil {
		t.Fatalf("GetPreviousMeetingNotes (between): %v", err)
	}
	if prev.ID != note1 {
		t.Fatalf("GetPreviousMeetingNotes (between): expected week 1's note, got %+v", prev)
	}

	// After both — the most recent (week 2) wins.
	prev, err = repo.GetPreviousMeetingNotes(ctxOwn, anchor.ID, base.Add(20*24*time.Hour))
	if err != nil {
		t.Fatalf("GetPreviousMeetingNotes (after both): %v", err)
	}
	if prev.ID != note2 {
		t.Fatalf("GetPreviousMeetingNotes (after both): expected week 2's note (most recent), got %+v", prev)
	}

	// Before any instance — none qualify.
	if _, err := repo.GetPreviousMeetingNotes(ctxOwn, anchor.ID, base); !errors.Is(err, ErrNoPreviousNotes) {
		t.Fatalf("GetPreviousMeetingNotes (before all): expected ErrNoPreviousNotes, got %v", err)
	}

	// Serientermin-Ausnahme: mutating inst1 must not ripple to inst2 or anchor.
	renamed := *inst1
	renamed.Title = "Week 1 — Rescheduled"
	renamed.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateMeeting(ctxOwn, &renamed); err != nil {
		t.Fatalf("UpdateMeeting (inst1): %v", err)
	}
	if gotAnchor, err := repo.GetMeeting(ctxOwn, anchor.ID, tenantOwn); err != nil || gotAnchor.Title != anchor.Title {
		t.Fatalf("UpdateMeeting (inst1) leaked into anchor: %+v err=%v", gotAnchor, err)
	}
	if gotInst2, err := repo.GetMeeting(ctxOwn, inst2.ID, tenantOwn); err != nil || gotInst2.Title != inst2.Title {
		t.Fatalf("UpdateMeeting (inst1) leaked into inst2: %+v err=%v", gotInst2, err)
	}
	if gotInst1, err := repo.GetMeeting(ctxOwn, inst1.ID, tenantOwn); err != nil || gotInst1.Title != renamed.Title {
		t.Fatalf("UpdateMeeting (inst1) did not persist on the target instance: %+v err=%v", gotInst1, err)
	}
}

// TestActionItems_CRUDAndTaskIDLink covers Create/Get/Update/Delete/List plus
// UpdateActionItemTaskID, including every method's not-found error path.
func TestActionItems_CRUDAndTaskIDLink(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, organizerOwn := seedMeetingFixtureTenant(t, pool, "Meeting ActionItems Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", organizerOwn)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	m := newTestMeeting(tenantOwn, organizerOwn, "ActionItems-Test", time.Now().UTC())
	if err := repo.CreateMeeting(ctxOwn, m); err != nil {
		t.Fatalf("CreateMeeting: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "meetings", m.ID)

	now := time.Now().UTC()
	item1 := &MeetingActionItem{ID: uuid.New(), TenantID: tenantOwn, MeetingID: m.ID, Description: "Send follow-up", SortOrder: 2, CreatedAt: now}
	item2 := &MeetingActionItem{ID: uuid.New(), TenantID: tenantOwn, MeetingID: m.ID, Description: "Book room", SortOrder: 1, CreatedAt: now}
	if err := repo.CreateActionItem(ctxOwn, item1); err != nil {
		t.Fatalf("CreateActionItem (1): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "meeting_action_items", item1.ID)
	if err := repo.CreateActionItem(ctxOwn, item2); err != nil {
		t.Fatalf("CreateActionItem (2): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "meeting_action_items", item2.ID)

	got, err := repo.GetActionItemByID(ctxOwn, item1.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetActionItemByID: %v", err)
	}
	if got.Description != item1.Description {
		t.Fatalf("GetActionItemByID: unexpected row %+v", got)
	}
	if _, err := repo.GetActionItemByID(ctxOwn, uuid.New(), tenantOwn); !errors.Is(err, ErrActionItemNotFound) {
		t.Fatalf("GetActionItemByID (missing): expected ErrActionItemNotFound, got %v", err)
	}

	list, err := repo.ListActionItems(ctxOwn, m.ID, tenantOwn)
	if err != nil {
		t.Fatalf("ListActionItems: %v", err)
	}
	if len(list) != 2 || list[0].ID != item2.ID || list[1].ID != item1.ID {
		t.Fatalf("ListActionItems: expected [item2, item1] ordered by sort_order, got %+v", list)
	}

	item1.Description = "Send follow-up email"
	item1.IsCompleted = true
	if err := repo.UpdateActionItem(ctxOwn, item1, tenantOwn); err != nil {
		t.Fatalf("UpdateActionItem: %v", err)
	}
	ghost := &MeetingActionItem{ID: uuid.New(), Description: "ghost"}
	if err := repo.UpdateActionItem(ctxOwn, ghost, tenantOwn); !errors.Is(err, ErrActionItemNotFound) {
		t.Fatalf("UpdateActionItem (missing): expected ErrActionItemNotFound, got %v", err)
	}

	taskID := uuid.New()
	if err := repo.UpdateActionItemTaskID(ctxOwn, item1.ID, taskID); err != nil {
		t.Fatalf("UpdateActionItemTaskID: %v", err)
	}
	got, err = repo.GetActionItemByID(ctxOwn, item1.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetActionItemByID (after task link): %v", err)
	}
	if got.TaskID == nil || *got.TaskID != taskID || !got.IsCompleted {
		t.Fatalf("GetActionItemByID (after task link): unexpected row %+v", got)
	}
	if err := repo.UpdateActionItemTaskID(ctxOwn, uuid.New(), taskID); !errors.Is(err, ErrActionItemNotFound) {
		t.Fatalf("UpdateActionItemTaskID (missing): expected ErrActionItemNotFound, got %v", err)
	}

	if err := repo.DeleteActionItem(ctxOwn, item2.ID, tenantOwn); err != nil {
		t.Fatalf("DeleteActionItem: %v", err)
	}
	if err := repo.DeleteActionItem(ctxOwn, item2.ID, tenantOwn); !errors.Is(err, ErrActionItemNotFound) {
		t.Fatalf("DeleteActionItem (already deleted): expected ErrActionItemNotFound, got %v", err)
	}
}

// TestChatMessages_SaveListLimitAndCrossTenant covers SaveChatMessage,
// ListChatMessages' limit clamp (<=0 or >500 falls back to 100) and its
// explicit tenant_id parameter (unlike notes/attendees, chat DOES carry one).
func TestChatMessages_SaveListLimitAndCrossTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, organizerOwn := seedMeetingFixtureTenant(t, pool, "Meeting Chat Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", organizerOwn)
	tenantOther := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOther, "Meeting Chat Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	m := newTestMeeting(tenantOwn, organizerOwn, "Chat-Test", time.Now().UTC())
	if err := repo.CreateMeeting(ctxOwn, m); err != nil {
		t.Fatalf("CreateMeeting: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "meetings", m.ID)

	base := time.Now().UTC()
	for i := range 3 {
		msg := &MeetingChatMessage{
			ID:         uuid.New(),
			TenantID:   tenantOwn,
			MeetingID:  m.ID,
			SenderID:   organizerOwn,
			SenderName: "Organizer",
			Message:    fmt.Sprintf("message %d", i),
			CreatedAt:  base.Add(time.Duration(i) * time.Second),
		}
		if err := repo.SaveChatMessage(ctxOwn, msg); err != nil {
			t.Fatalf("SaveChatMessage (%d): %v", i, err)
		}
		defer testutil.CleanupRow(t, pool, "meeting_chat", msg.ID)
	}

	msgs, err := repo.ListChatMessages(ctxOwn, m.ID, tenantOwn, 2)
	if err != nil {
		t.Fatalf("ListChatMessages (limit=2): %v", err)
	}
	if len(msgs) != 2 || msgs[0].Message != "message 0" || msgs[1].Message != "message 1" {
		t.Fatalf("ListChatMessages (limit=2): expected first two in ASC order, got %+v", msgs)
	}

	// limit <= 0 clamps to 100 — all three come back.
	msgs, err = repo.ListChatMessages(ctxOwn, m.ID, tenantOwn, 0)
	if err != nil {
		t.Fatalf("ListChatMessages (limit=0): %v", err)
	}
	if len(msgs) != 3 {
		t.Fatalf("ListChatMessages (limit=0 clamps to 100): expected all 3, got %d", len(msgs))
	}

	// limit > 500 also clamps to 100.
	msgs, err = repo.ListChatMessages(ctxOwn, m.ID, tenantOwn, 501)
	if err != nil {
		t.Fatalf("ListChatMessages (limit=501): %v", err)
	}
	if len(msgs) != 3 {
		t.Fatalf("ListChatMessages (limit=501 clamps to 100): expected all 3, got %d", len(msgs))
	}

	// Explicit foreign tenantID parameter (not just RLS) must yield nothing.
	msgs, err = repo.ListChatMessages(ctxOther, m.ID, tenantOther, 100)
	if err != nil {
		t.Fatalf("ListChatMessages (foreign tenant param): %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("ListChatMessages (foreign tenant param): expected 0, got %d", len(msgs))
	}
}

// TestCoHosts_AddIsListRemoveIdempotent covers the co-host lifecycle,
// including AddCoHost/RemoveCoHost idempotency (both are explicitly documented
// as safe to call redundantly).
func TestCoHosts_AddIsListRemoveIdempotent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, organizerOwn := seedMeetingFixtureTenant(t, pool, "Meeting CoHosts Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", organizerOwn)
	coHostUser := seedMeetingUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", coHostUser)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	m := newTestMeeting(tenantOwn, organizerOwn, "CoHost-Test", time.Now().UTC())
	if err := repo.CreateMeeting(ctxOwn, m); err != nil {
		t.Fatalf("CreateMeeting: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "meetings", m.ID)

	isCoHost, err := repo.IsCoHost(ctxOwn, tenantOwn, m.ID, coHostUser)
	if err != nil {
		t.Fatalf("IsCoHost (before): %v", err)
	}
	if isCoHost {
		t.Fatalf("IsCoHost (before): expected false")
	}

	if err := repo.AddCoHost(ctxOwn, tenantOwn, m.ID, coHostUser, organizerOwn); err != nil {
		t.Fatalf("AddCoHost: %v", err)
	}
	// Idempotent re-add must not error.
	if err := repo.AddCoHost(ctxOwn, tenantOwn, m.ID, coHostUser, organizerOwn); err != nil {
		t.Fatalf("AddCoHost (duplicate): %v", err)
	}

	isCoHost, err = repo.IsCoHost(ctxOwn, tenantOwn, m.ID, coHostUser)
	if err != nil {
		t.Fatalf("IsCoHost (after): %v", err)
	}
	if !isCoHost {
		t.Fatalf("IsCoHost (after): expected true")
	}

	list, err := repo.ListCoHosts(ctxOwn, tenantOwn, m.ID)
	if err != nil {
		t.Fatalf("ListCoHosts: %v", err)
	}
	if len(list) != 1 || list[0].UserID != coHostUser {
		t.Fatalf("ListCoHosts: expected exactly one co-host, got %+v", list)
	}

	if err := repo.RemoveCoHost(ctxOwn, tenantOwn, m.ID, coHostUser); err != nil {
		t.Fatalf("RemoveCoHost: %v", err)
	}
	// Idempotent: removing an absent co-host must not error.
	if err := repo.RemoveCoHost(ctxOwn, tenantOwn, m.ID, coHostUser); err != nil {
		t.Fatalf("RemoveCoHost (already removed): %v", err)
	}
	isCoHost, err = repo.IsCoHost(ctxOwn, tenantOwn, m.ID, coHostUser)
	if err != nil {
		t.Fatalf("IsCoHost (after remove): %v", err)
	}
	if isCoHost {
		t.Fatalf("IsCoHost (after remove): expected false")
	}
}

// TestBreakoutRoomsAndAssignments_FullLifecycle covers room creation/listing/
// lookup/close-all plus the assignment upsert/get/list/clear surface,
// including the "not assigned" nil,nil contract on GetBreakoutAssignmentForUser.
func TestBreakoutRoomsAndAssignments_FullLifecycle(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, organizerOwn := seedMeetingFixtureTenant(t, pool, "Meeting Breakout Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", organizerOwn)
	attendeeUser := seedMeetingUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", attendeeUser)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	m := newTestMeeting(tenantOwn, organizerOwn, "Breakout-Test", time.Now().UTC())
	if err := repo.CreateMeeting(ctxOwn, m); err != nil {
		t.Fatalf("CreateMeeting: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "meetings", m.ID)

	now := time.Now().UTC()
	roomA := &BreakoutRoom{ID: uuid.New(), MeetingID: m.ID, RoomName: "brk-" + uuid.New().String(), Label: "Room A", SortIndex: 1, Status: "open", CreatedBy: organizerOwn, CreatedAt: now}
	roomB := &BreakoutRoom{ID: uuid.New(), MeetingID: m.ID, RoomName: "brk-" + uuid.New().String(), Label: "Room B", SortIndex: 0, Status: "open", CreatedBy: organizerOwn, CreatedAt: now}
	if err := repo.CreateBreakoutRoom(ctxOwn, roomA); err != nil {
		t.Fatalf("CreateBreakoutRoom (A): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "meeting_breakout_rooms", roomA.ID)
	if err := repo.CreateBreakoutRoom(ctxOwn, roomB); err != nil {
		t.Fatalf("CreateBreakoutRoom (B): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "meeting_breakout_rooms", roomB.ID)

	rooms, err := repo.ListBreakoutRooms(ctxOwn, m.ID, tenantOwn)
	if err != nil {
		t.Fatalf("ListBreakoutRooms: %v", err)
	}
	if len(rooms) != 2 || rooms[0].ID != roomB.ID || rooms[1].ID != roomA.ID {
		t.Fatalf("ListBreakoutRooms: expected [B, A] ordered by sort_index, got %+v", rooms)
	}

	got, err := repo.GetBreakoutRoom(ctxOwn, roomA.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetBreakoutRoom: %v", err)
	}
	if got.Label != "Room A" {
		t.Fatalf("GetBreakoutRoom: unexpected row %+v", got)
	}
	if _, err := repo.GetBreakoutRoom(ctxOwn, uuid.New(), tenantOwn); !errors.Is(err, ErrBreakoutRoomNotFound) {
		t.Fatalf("GetBreakoutRoom (missing): expected ErrBreakoutRoomNotFound, got %v", err)
	}

	assignment, err := repo.GetBreakoutAssignmentForUser(ctxOwn, m.ID, attendeeUser, tenantOwn)
	if err != nil {
		t.Fatalf("GetBreakoutAssignmentForUser (none yet): %v", err)
	}
	if assignment != nil {
		t.Fatalf("GetBreakoutAssignmentForUser (none yet): expected nil, nil, got %+v", assignment)
	}

	if err := repo.UpsertBreakoutAssignment(ctxOwn, m.ID, roomA.ID, attendeeUser, organizerOwn); err != nil {
		t.Fatalf("UpsertBreakoutAssignment (A): %v", err)
	}
	assignment, err = repo.GetBreakoutAssignmentForUser(ctxOwn, m.ID, attendeeUser, tenantOwn)
	if err != nil {
		t.Fatalf("GetBreakoutAssignmentForUser (A): %v", err)
	}
	if assignment == nil || assignment.ID != roomA.ID {
		t.Fatalf("GetBreakoutAssignmentForUser (A): expected roomA, got %+v", assignment)
	}

	// Re-assign to room B — ON CONFLICT DO UPDATE must move the assignment.
	if err := repo.UpsertBreakoutAssignment(ctxOwn, m.ID, roomB.ID, attendeeUser, organizerOwn); err != nil {
		t.Fatalf("UpsertBreakoutAssignment (B): %v", err)
	}
	assignment, err = repo.GetBreakoutAssignmentForUser(ctxOwn, m.ID, attendeeUser, tenantOwn)
	if err != nil {
		t.Fatalf("GetBreakoutAssignmentForUser (B): %v", err)
	}
	if assignment == nil || assignment.ID != roomB.ID {
		t.Fatalf("GetBreakoutAssignmentForUser (B): expected reassignment to roomB, got %+v", assignment)
	}

	assignments, err := repo.ListBreakoutAssignments(ctxOwn, m.ID, tenantOwn)
	if err != nil {
		t.Fatalf("ListBreakoutAssignments: %v", err)
	}
	if len(assignments) != 1 || assignments[0].BreakoutRoomID != roomB.ID {
		t.Fatalf("ListBreakoutAssignments: expected exactly one row in roomB, got %+v", assignments)
	}

	if err := repo.ClearBreakoutAssignment(ctxOwn, m.ID, attendeeUser, tenantOwn); err != nil {
		t.Fatalf("ClearBreakoutAssignment: %v", err)
	}
	assignment, err = repo.GetBreakoutAssignmentForUser(ctxOwn, m.ID, attendeeUser, tenantOwn)
	if err != nil {
		t.Fatalf("GetBreakoutAssignmentForUser (after clear): %v", err)
	}
	if assignment != nil {
		t.Fatalf("GetBreakoutAssignmentForUser (after clear): expected nil, got %+v", assignment)
	}

	// Re-assign once more so ClearAllBreakoutAssignments has something to clear.
	if err := repo.UpsertBreakoutAssignment(ctxOwn, m.ID, roomA.ID, attendeeUser, organizerOwn); err != nil {
		t.Fatalf("UpsertBreakoutAssignment (reassign for ClearAll): %v", err)
	}
	if err := repo.ClearAllBreakoutAssignments(ctxOwn, m.ID, tenantOwn); err != nil {
		t.Fatalf("ClearAllBreakoutAssignments: %v", err)
	}
	assignments, err = repo.ListBreakoutAssignments(ctxOwn, m.ID, tenantOwn)
	if err != nil {
		t.Fatalf("ListBreakoutAssignments (after clear all): %v", err)
	}
	if len(assignments) != 0 {
		t.Fatalf("ListBreakoutAssignments (after clear all): expected empty, got %+v", assignments)
	}

	names, err := repo.CloseAllBreakoutRooms(ctxOwn, m.ID, tenantOwn)
	if err != nil {
		t.Fatalf("CloseAllBreakoutRooms: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("CloseAllBreakoutRooms: expected both rooms closed, got %v", names)
	}
	rooms, err = repo.ListBreakoutRooms(ctxOwn, m.ID, tenantOwn)
	if err != nil {
		t.Fatalf("ListBreakoutRooms (after close all): %v", err)
	}
	for _, r := range rooms {
		if r.Status != "closed" || r.ClosedAt == nil {
			t.Fatalf("ListBreakoutRooms (after close all): expected all closed, got %+v", r)
		}
	}

	// A second CloseAllBreakoutRooms call has nothing open left to close.
	names, err = repo.CloseAllBreakoutRooms(ctxOwn, m.ID, tenantOwn)
	if err != nil {
		t.Fatalf("CloseAllBreakoutRooms (idempotent): %v", err)
	}
	if len(names) != 0 {
		t.Fatalf("CloseAllBreakoutRooms (idempotent): expected nothing left to close, got %v", names)
	}
}

// TestSetLocked_And_UpdateAISummary_ErrorPaths covers the two remaining
// single-column update helpers and their not-found paths.
func TestSetLocked_And_UpdateAISummary_ErrorPaths(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, organizerOwn := seedMeetingFixtureTenant(t, pool, "Meeting Lock Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", organizerOwn)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	m := newTestMeeting(tenantOwn, organizerOwn, "Lock-Test", time.Now().UTC())
	if err := repo.CreateMeeting(ctxOwn, m); err != nil {
		t.Fatalf("CreateMeeting: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "meetings", m.ID)

	if err := repo.SetLocked(ctxOwn, tenantOwn, m.ID, true); err != nil {
		t.Fatalf("SetLocked: %v", err)
	}
	got, err := repo.GetMeeting(ctxOwn, m.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetMeeting: %v", err)
	}
	if !got.Locked {
		t.Fatalf("SetLocked: expected locked=true")
	}
	if err := repo.SetLocked(ctxOwn, tenantOwn, uuid.New(), true); !errors.Is(err, ErrNotFound) {
		t.Fatalf("SetLocked (missing): expected ErrNotFound, got %v", err)
	}

	summaryAt := time.Now().UTC()
	if err := repo.UpdateAISummary(ctxOwn, tenantOwn, m.ID, "AI generated summary", summaryAt); err != nil {
		t.Fatalf("UpdateAISummary: %v", err)
	}
	got, err = repo.GetMeeting(ctxOwn, m.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetMeeting (after summary): %v", err)
	}
	if got.AISummary == nil || *got.AISummary != "AI generated summary" {
		t.Fatalf("UpdateAISummary: unexpected row %+v", got)
	}
	if err := repo.UpdateAISummary(ctxOwn, tenantOwn, uuid.New(), "x", summaryAt); !errors.Is(err, ErrNotFound) {
		t.Fatalf("UpdateAISummary (missing): expected ErrNotFound, got %v", err)
	}
}
