package main

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/work/meeting"
)

func TestRunPeriodically_WaitsForDelayThenTicksAndStopsOnCancel(t *testing.T) {
	var calls atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		runPeriodically(ctx, 60*time.Millisecond, 10*time.Millisecond, func() { calls.Add(1) })
	}()

	time.Sleep(20 * time.Millisecond)
	assert.Zero(t, calls.Load(), "no run before the startup delay has passed")

	require.Eventually(t, func() bool { return calls.Load() >= 3 }, 2*time.Second, 5*time.Millisecond,
		"initial run plus ticker runs after the delay")

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runPeriodically did not stop after context cancel")
	}
}

func TestRunPeriodically_CancelDuringDelayNeverRuns(t *testing.T) {
	var calls atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	runPeriodically(ctx, time.Hour, time.Hour, func() { calls.Add(1) })
	assert.Zero(t, calls.Load())
}

type fakeRecordingCleaner struct {
	calls atomic.Int32
	err   error
}

func (f *fakeRecordingCleaner) CleanupExpiredRecordings(context.Context) (int, error) {
	f.calls.Add(1)
	return 0, f.err
}

func TestCleanupExpiredRecordings_ErrorDoesNotStopTheLoop(t *testing.T) {
	cleaner := &fakeRecordingCleaner{err: errors.New("db down")}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go runPeriodically(ctx, 0, 10*time.Millisecond, func() { cleanupExpiredRecordings(ctx, cleaner) })

	require.Eventually(t, func() bool { return cleaner.calls.Load() >= 3 }, 2*time.Second, 5*time.Millisecond,
		"a failing cleanup must not end the loop")
}

type fakeSweepService struct {
	mu        sync.Mutex
	stale     []meeting.Meeting
	listErr   error
	completeE map[string]error
	completed []string
}

func (f *fakeSweepService) ListStaleMeetings(context.Context, time.Time) ([]meeting.Meeting, error) {
	return f.stale, f.listErr
}

func (f *fakeSweepService) CompleteMeetingByRoom(_ context.Context, room string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.completeE[room]; err != nil {
		return err
	}
	f.completed = append(f.completed, room)
	return nil
}

type fakeRooms struct {
	participants map[string][]string
	errs         map[string]error
}

func (f *fakeRooms) ListParticipants(_ context.Context, room string) ([]string, error) {
	return f.participants[room], f.errs[room]
}

func staleMeeting(room *string) meeting.Meeting {
	return meeting.Meeting{ID: uuid.New(), RoomName: room}
}

func TestSweepStaleMeetings_ClosesOnlyEmptyRoomsAndSurvivesPerMeetingErrors(t *testing.T) {
	str := func(s string) *string { return &s }
	svc := &fakeSweepService{
		stale: []meeting.Meeting{
			staleMeeting(nil),                 // no room → skipped
			staleMeeting(str("")),             // empty room name → skipped
			staleMeeting(str("busy")),         // participants present → kept open
			staleMeeting(str("lk-error")),     // LiveKit lookup failed → skipped
			staleMeeting(str("complete-err")), // empty, but closing fails → next one still handled
			staleMeeting(str("empty")),        // empty → closed
		},
		completeE: map[string]error{"complete-err": errors.New("db down")},
	}
	rooms := &fakeRooms{
		participants: map[string][]string{"busy": {"alice"}},
		errs:         map[string]error{"lk-error": errors.New("livekit unreachable")},
	}

	sweepStaleMeetings(context.Background(), svc, rooms, 15*time.Minute)

	assert.Equal(t, []string{"empty"}, svc.completed)
}

func TestSweepStaleMeetings_ListErrorClosesNothing(t *testing.T) {
	svc := &fakeSweepService{listErr: errors.New("db down")}
	sweepStaleMeetings(context.Background(), svc, &fakeRooms{}, time.Minute)
	assert.Empty(t, svc.completed)
}
