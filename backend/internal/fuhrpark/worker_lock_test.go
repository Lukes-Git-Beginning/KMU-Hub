package fuhrpark

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// Not parallel: the advisory lock key is global to the database.
func TestLockedRunner_SkipsWhileLockHeldAndRunsAfterRelease(t *testing.T) {
	pool := testutil.PoolFromEnv(t)
	ctx := context.Background()

	var runs atomic.Int32
	runner := LockedRunner(pool, func(context.Context) error {
		runs.Add(1)
		return nil
	})

	// Lock free → runs.
	require.NoError(t, runner(ctx))
	assert.EqualValues(t, 1, runs.Load())

	// Another "replica" holds the lock in an open transaction → skipped without error.
	holder, err := pool.Begin(ctx)
	require.NoError(t, err)
	released := false
	t.Cleanup(func() {
		if !released {
			_ = holder.Rollback(ctx)
		}
	})
	var got bool
	require.NoError(t, holder.QueryRow(ctx, `SELECT pg_try_advisory_xact_lock($1)`, LockKey()).Scan(&got))
	require.True(t, got, "test holder must obtain the lock")

	require.NoError(t, runner(ctx))
	assert.EqualValues(t, 1, runs.Load(), "run must be skipped while another transaction holds the lock")

	// Third pass after the foreign transaction ended → runs again.
	require.NoError(t, holder.Rollback(ctx))
	released = true
	require.NoError(t, runner(ctx))
	assert.EqualValues(t, 2, runs.Load())
}

func TestLockedRunner_ErrorIsReturnedAndReleasesLock(t *testing.T) {
	pool := testutil.PoolFromEnv(t)
	ctx := context.Background()

	boom := errors.New("boom")
	var runs atomic.Int32
	fail := true
	runner := LockedRunner(pool, func(context.Context) error {
		runs.Add(1)
		if fail {
			return boom
		}
		return nil
	})

	err := runner(ctx)
	require.ErrorIs(t, err, boom)

	// The rolled-back transaction must not keep the lock: the next pass runs.
	fail = false
	require.NoError(t, runner(ctx))
	assert.EqualValues(t, 2, runs.Load())
}

func TestTuevWorker_Run_TicksSurviveErrorsAndStopOnCancel(t *testing.T) {
	w := NewTuevWorker(nil, nil, nil)
	w.runInterval = 10 * time.Millisecond

	var calls atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- w.Run(ctx, func(context.Context) error {
			calls.Add(1)
			return errors.New("scan failed") // an error must not end the loop
		})
	}()

	require.Eventually(t, func() bool { return calls.Load() >= 3 }, 2*time.Second, 5*time.Millisecond,
		"loop must keep ticking after run errors")

	cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not stop after context cancel")
	}
}
