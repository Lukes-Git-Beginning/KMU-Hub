package middleware

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/kmuhub/kmuhub/internal/metrics"
	securityv1 "github.com/kmuhub/kmuhub/proto/security/v1"
)

// A panic while writing one audit event must cost only that event: the worker
// stays alive, the loss is counted, and the next event is still written.
func TestAuditLogger_WorkerSurvivesPanicInProcessEvent(t *testing.T) {
	reg := metrics.NewRegistry()
	client := &fakeSecurityClient{err: errors.New("ignored")}
	calls := 0
	a := NewAuditLogger(func() (securityv1.SecurityServiceClient, error) {
		calls++
		if calls == 1 {
			panic("boom in client lookup")
		}
		return client, nil
	})
	a.SetMetrics(reg)
	a.Start(1)
	t.Cleanup(a.Close)

	for i := 0; i < 2; i++ {
		a.events <- auditEvent{method: http.MethodPost, path: "/api/v1/contacts", statusCode: 200}
	}

	deadline := time.After(2 * time.Second)
	for client.calls.Load() < 1 {
		select {
		case <-deadline:
			t.Fatal("second event never reached the client: worker died after the panic")
		case <-time.After(5 * time.Millisecond):
		}
	}
	if got := counterValue(t, reg.AuditEventsDropped.WithLabelValues(dropWorkerPanic)); got != 1 {
		t.Fatalf("worker_panic counter = %v, want 1", got)
	}
}
