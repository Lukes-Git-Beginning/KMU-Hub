package server

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/notification/integration"
	notificationv1 "github.com/kmuhub/kmuhub/proto/notification/v1"
)

type recordingProcessor struct {
	seen     integration.WebhookRequest
	response integration.WebhookResponse
}

func (p *recordingProcessor) Process(_ context.Context, req integration.WebhookRequest) integration.WebhookResponse {
	p.seen = req
	return p.response
}

// A platform without credentials has no processor. The RPC must say so instead
// of handling an inbound payload it cannot verify.
func TestHandlePlatformWebhook_UnconfiguredPlatform(t *testing.T) {
	s := NewNotificationGRPCServer(nil, nil, nil)

	_, err := s.HandlePlatformWebhook(context.Background(), &notificationv1.HandlePlatformWebhookRequest{
		Platform: integration.PlatformSlack,
		Kind:     integration.WebhookKindSlackCommand,
	})

	if status.Code(err) != codes.Unimplemented {
		t.Fatalf("unconfigured platform: got code %s, want %s", status.Code(err), codes.Unimplemented)
	}
}

// The tunnel must hand the processor exactly what arrived. Slack signs the raw
// body, so a single byte changed anywhere along the way invalidates the
// signature — this pins that the RPC layer changes nothing.
func TestHandlePlatformWebhook_PassesRawRequestThrough(t *testing.T) {
	body := []byte("token=xyz&team_id=T1&text=link")
	processor := &recordingProcessor{
		response: integration.WebhookResponse{
			StatusCode:  http.StatusOK,
			ContentType: "application/json",
			Body:        []byte(`{"response_type":"ephemeral"}`),
		},
	}
	s := NewNotificationGRPCServer(nil, nil, nil, WithPlatformWebhooks(
		map[string]integration.WebhookProcessor{integration.PlatformSlack: processor},
	))

	resp, err := s.HandlePlatformWebhook(context.Background(), &notificationv1.HandlePlatformWebhookRequest{
		Platform: integration.PlatformSlack,
		Kind:     integration.WebhookKindSlackCommand,
		Body:     body,
		Headers:  map[string]string{"X-Slack-Signature": "v0=abc"},
	})
	if err != nil {
		t.Fatalf("HandlePlatformWebhook: %v", err)
	}

	if !bytes.Equal(processor.seen.Body, body) {
		t.Fatalf("body: got %q, want %q", processor.seen.Body, body)
	}
	if processor.seen.Kind != integration.WebhookKindSlackCommand {
		t.Fatalf("kind: got %q, want %q", processor.seen.Kind, integration.WebhookKindSlackCommand)
	}
	if processor.seen.Headers["X-Slack-Signature"] != "v0=abc" {
		t.Fatalf("headers: signature header did not survive the tunnel: %v", processor.seen.Headers)
	}

	if resp.GetStatusCode() != http.StatusOK {
		t.Fatalf("status: got %d, want %d", resp.GetStatusCode(), http.StatusOK)
	}
	if resp.GetContentType() != "application/json" {
		t.Fatalf("content type: got %q, want application/json", resp.GetContentType())
	}
	if string(resp.GetBody()) != `{"response_type":"ephemeral"}` {
		t.Fatalf("body: got %q", resp.GetBody())
	}
}

// An oversized payload is refused before the processor sees it.
func TestHandlePlatformWebhook_RejectsOversizedBody(t *testing.T) {
	processor := &recordingProcessor{}
	s := NewNotificationGRPCServer(nil, nil, nil, WithPlatformWebhooks(
		map[string]integration.WebhookProcessor{integration.PlatformSlack: processor},
	))

	_, err := s.HandlePlatformWebhook(context.Background(), &notificationv1.HandlePlatformWebhookRequest{
		Platform: integration.PlatformSlack,
		Kind:     integration.WebhookKindSlackCommand,
		Body:     make([]byte, maxWebhookBody+1),
	})

	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("oversized body: got code %s, want %s", status.Code(err), codes.InvalidArgument)
	}
	if processor.seen.Body != nil {
		t.Fatal("oversized body reached the processor")
	}
}
