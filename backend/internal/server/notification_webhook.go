package server

import (
	"context"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/notification/integration"
	notificationv1 "github.com/kmuhub/kmuhub/proto/notification/v1"
)

// maxWebhookBody mirrors the gateway's cap. The gateway already truncates, so
// this is the second riegel for any other caller of the RPC.
const maxWebhookBody = 1 << 20 // 1 MiB

// HandlePlatformWebhook processes an inbound Slack or Teams request that the
// gateway tunneled here unchanged.
//
// The request arrives without a tenant on purpose: it is unauthenticated, and
// the processor resolves the tenant from the platform identity it can verify.
// Nothing in this method touches data — verification and tenant resolution both
// happen inside the platform processor, before its first repository call.
func (s *NotificationGRPCServer) HandlePlatformWebhook(
	ctx context.Context,
	req *notificationv1.HandlePlatformWebhookRequest,
) (*notificationv1.HandlePlatformWebhookResponse, error) {
	processor, ok := s.webhooks[req.GetPlatform()]
	if !ok || processor == nil {
		return nil, status.Errorf(codes.Unimplemented,
			"%s webhooks are not configured on this deployment", req.GetPlatform())
	}

	if len(req.GetBody()) > maxWebhookBody {
		return nil, status.Error(codes.InvalidArgument, "webhook body too large")
	}

	result := processor.Process(ctx, integration.WebhookRequest{
		Kind:    req.GetKind(),
		Body:    req.GetBody(),
		Headers: req.GetHeaders(),
	})

	statusCode := result.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	return &notificationv1.HandlePlatformWebhookResponse{
		StatusCode:  int32(statusCode),
		ContentType: result.ContentType,
		Body:        result.Body,
	}, nil
}
