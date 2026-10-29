package engine

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/automation/action"
	"github.com/kmuhub/kmuhub/internal/automation/trigger"
	"github.com/kmuhub/kmuhub/internal/automation/workflow"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// listingWorkflowRepo serves a fixed set of active automations to the trigger
// matcher and counts how often the matcher asked.
type listingWorkflowRepo struct {
	mockWorkflowRepo
	autos     []*models.Automation
	listCalls atomic.Int32
}

func (r *listingWorkflowRepo) List(_ context.Context, _ workflow.ListFilter) ([]*models.Automation, int, error) {
	r.listCalls.Add(1)
	return r.autos, len(r.autos), nil
}

// TestNotificationSendEvent_DoesNotTriggerAWorkflow pins the loop guard for the
// notification.send action: the automation service listens on "*" and would
// otherwise pick up the event the action itself publishes. A workflow
// triggered by that event type must not run for it -- and the control event
// (same type, other module) proves the workflow would have matched.
func TestNotificationSendEvent_DoesNotTriggerAWorkflow(t *testing.T) {
	tenantID := uuid.New()

	var published []models.EventPayload
	send := action.NewSendNotificationAction(func(_ context.Context, p models.EventPayload) error {
		published = append(published, p)
		return nil
	})
	res, err := send.Execute(testutil.WithTenantCtx(context.Background(), tenantID),
		[]byte(`{"target_user_id":"`+uuid.NewString()+`","title":"t","body":"b"}`), nil)
	require.NoError(t, err)
	require.True(t, res.Success)
	require.Len(t, published, 1)

	triggers := trigger.NewTriggerRegistry()
	triggers.Register(&trigger.TriggerDefinition{Type: "test.on_notification", EventType: action.EventTypeAutomationNotification})
	auto := makeAutomation(tenantID)
	auto.TriggerType = "test.on_notification"
	repo := &listingWorkflowRepo{autos: []*models.Automation{&auto}}

	started := make(chan struct{}, 1)
	gate := make(chan struct{})
	close(gate)
	consumer := NewAutomationConsumer(trigger.NewTriggerMatcher(repo, triggers),
		newTestEngine(&blockingExecutor{started: started, gate: gate}))

	require.NoError(t, consumer.HandleEvent(context.Background(), published[0]))
	require.Zero(t, repo.listCalls.Load(), "the action's own event reached the trigger matcher")

	control := published[0]
	control.ModuleID = "crm"
	require.NoError(t, consumer.HandleEvent(context.Background(), control))
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("control event did not run the workflow; the test set-up does not match")
	}
}
