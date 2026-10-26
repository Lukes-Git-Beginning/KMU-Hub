package server

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/automation/action"
	workv1 "github.com/kmuhub/kmuhub/proto/work/v1"
)

// inProcessWorkClient routes the automation action's CreateTask call into the
// real WorkGRPCServer handler, so the request is parsed by the server and not
// by a fake that ignores created_by.
type inProcessWorkClient struct {
	workv1.WorkServiceClient
	srv *WorkGRPCServer
}

func (c inProcessWorkClient) CreateTask(ctx context.Context, in *workv1.CreateTaskRequest, _ ...grpc.CallOption) (*workv1.CreateTaskResponse, error) {
	return c.srv.CreateTask(ctx, in)
}

// work.create_task failed for every tenant because the action never set
// created_by and the server's first step is uuid.Parse(req.CreatedBy). The
// server has no services wired here, so a request that clears the created_by
// parse stops at the next check (missing tenant in the context) with
// Unauthenticated; a rejected creator would surface as InvalidArgument.
func TestCreateTaskAction_RequestPassesServerCreatedByParse(t *testing.T) {
	a := action.NewCreateTaskAction(inProcessWorkClient{srv: &WorkGRPCServer{}})
	cfg := json.RawMessage(`{"project_id":"` + uuid.NewString() + `","title":"Follow-up: Acme"}`)

	_, err := a.Execute(context.Background(), cfg, map[string]any{"actor_id": uuid.NewString()})

	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err), "created_by must clear the server parse: %v", err)
}
