package server

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/middleware"
	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

// Only an explicit "all" decides for any employee; an empty or unknown scope is the reporting line.
func TestLeaveApprovalOptions_FailClosed(t *testing.T) {
	assert.Empty(t, leaveApprovalOptions(auth.ScopeAll))
	for _, scope := range []string{"", auth.ScopeTeam, auth.ScopeOwn, "ALL", "bogus"} {
		assert.Len(t, leaveApprovalOptions(scope), 1, "scope %q must restrict the decision", scope)
	}
}

// The reporting line is the caller's, taken from the call context: without a caller the list
// is refused before it reaches the service.
func TestListLeaveRequests_ReportingLineNeedsACaller(t *testing.T) {
	srv := &HRGRPCServer{}
	ctx := context.WithValue(context.Background(), middleware.TenantIDKey, uuid.New())

	_, err := srv.ListLeaveRequests(ctx, &hrv1.ListLeaveRequestsReq{ReportingLineOnly: true})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}
