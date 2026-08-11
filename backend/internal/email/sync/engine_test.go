package sync

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// newTestEngine builds an Engine without a real account.Service/MessageSyncer/
// FolderSyncer/AttachmentStorer wiring. NewEngine, GetStatus, Stop,
// StopWorker and TriggerSync never call into any of the four, so nil is
// sufficient here.
func newTestEngine() *Engine {
	return NewEngine(nil, nil, nil, nil)
}

func TestNewEngine_InitializesEmptyMaps(t *testing.T) {
	e := newTestEngine()

	assert.NotNil(t, e.workers)
	assert.Empty(t, e.workers)
	assert.NotNil(t, e.statuses)
	assert.Empty(t, e.statuses)
}

func TestEngine_GetStatus_UnknownAccountReturnsDefault(t *testing.T) {
	e := newTestEngine()
	id := uuid.New()

	status := e.GetStatus(id)

	assert.Equal(t, id, status.AccountID)
	assert.False(t, status.IsSyncing)
	assert.Nil(t, status.LastSync)
	assert.Empty(t, status.Error)
}

func TestEngine_GetStatus_KnownAccountReturnsStoredStatus(t *testing.T) {
	e := newTestEngine()
	id := uuid.New()
	stored := &SyncStatus{AccountID: id, IsSyncing: true, Error: "boom"}
	e.statuses[id] = stored

	status := e.GetStatus(id)

	assert.Same(t, stored, status)
}

func TestEngine_Stop_WithoutCancelFnOrWorkers_DoesNotPanic(t *testing.T) {
	e := newTestEngine()

	assert.NotPanics(t, func() { e.Stop() })
}

func TestEngine_Stop_StopsAndClearsAllWorkers(t *testing.T) {
	e := newTestEngine()
	id1, id2 := uuid.New(), uuid.New()
	var stopped1, stopped2 bool
	e.workers[id1] = &Worker{cancelFn: func() { stopped1 = true }}
	e.workers[id2] = &Worker{cancelFn: func() { stopped2 = true }}

	e.Stop()

	assert.True(t, stopped1)
	assert.True(t, stopped2)
	assert.Empty(t, e.workers)
}

func TestEngine_StopWorker_UnknownAccount_IsNoOp(t *testing.T) {
	e := newTestEngine()

	assert.NotPanics(t, func() { e.StopWorker(uuid.New()) })
	assert.Empty(t, e.workers)
}

func TestEngine_StopWorker_KnownAccount_StopsAndRemoves(t *testing.T) {
	e := newTestEngine()
	id := uuid.New()
	var stopped bool
	e.workers[id] = &Worker{cancelFn: func() { stopped = true }}

	e.StopWorker(id)

	assert.True(t, stopped)
	_, ok := e.workers[id]
	assert.False(t, ok)
}

func TestEngine_TriggerSync_UnknownAccount_IsNoOp(t *testing.T) {
	e := newTestEngine()

	assert.NotPanics(t, func() { e.TriggerSync(uuid.New()) })
}

func TestEngine_TriggerSync_KnownAccount_SignalsWorker(t *testing.T) {
	e := newTestEngine()
	id := uuid.New()
	w := &Worker{triggerCh: make(chan struct{}, 1)}
	e.workers[id] = w

	e.TriggerSync(id)

	select {
	case <-w.triggerCh:
	default:
		t.Fatal("expected TriggerSync to send on the worker's triggerCh")
	}
}
