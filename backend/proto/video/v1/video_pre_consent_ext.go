// Package videov1 contains hand-written gRPC extensions for initiator pre-recording consent.
// These complement the protoc-generated code and will be replaced by regenerated proto output
// once the .proto file is compiled in CI (Sprint 4 proto-regen task).
package videov1

import (
	context "context"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// ============================================================================
// Method name constant
// ============================================================================

const VideoService_ConfirmInitiatorConsent_FullMethodName = "/video.v1.VideoService/ConfirmInitiatorConsent"

// ============================================================================
// Request / Response message types
// ============================================================================

// ConfirmInitiatorConsentRequest is sent by the gateway when the recording initiator
// acknowledges the pre-recording consent dialog before calling StartRecording.
type ConfirmInitiatorConsentRequest struct {
	// RecordingID is the UUID of the recording row to stamp.
	RecordingID string `json:"recording_id"`
	// UserID is the initiator's user UUID (must match started_by on the recording row).
	UserID string `json:"user_id"`
	// TenantID is extracted from the JWT claim for cross-tenant guard (Option-B ready).
	TenantID string `json:"tenant_id"`
}

// ConfirmInitiatorConsentResponse is returned on success.
type ConfirmInitiatorConsentResponse struct {
	// Stamped is true when pre_recording_consent_at was set successfully.
	Stamped bool `json:"stamped"`
}

// ============================================================================
// Client interface extension
// ============================================================================

// VideoServicePreConsentClient extends VideoServiceClient with the initiator
// pre-recording consent RPC (R2-P0.4 / Migration 000107).
type VideoServicePreConsentClient interface {
	VideoServiceClient
	ConfirmInitiatorConsent(ctx context.Context, in *ConfirmInitiatorConsentRequest, opts ...grpc.CallOption) (*ConfirmInitiatorConsentResponse, error)
}

type videoServicePreConsentClient struct {
	VideoServiceClient
	cc grpc.ClientConnInterface
}

// NewVideoServicePreConsentClient wraps a gRPC connection and returns a client
// that implements both the base VideoServiceClient and the pre-consent RPC.
func NewVideoServicePreConsentClient(cc grpc.ClientConnInterface) VideoServicePreConsentClient {
	return &videoServicePreConsentClient{
		VideoServiceClient: NewVideoServiceClient(cc),
		cc:                 cc,
	}
}

func (c *videoServicePreConsentClient) ConfirmInitiatorConsent(ctx context.Context, in *ConfirmInitiatorConsentRequest, opts ...grpc.CallOption) (*ConfirmInitiatorConsentResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ConfirmInitiatorConsentResponse)
	err := c.cc.Invoke(ctx, VideoService_ConfirmInitiatorConsent_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ============================================================================
// Server interface extension
// ============================================================================

// VideoServicePreConsentServer extends VideoServiceServer with the initiator
// pre-recording consent RPC.
type VideoServicePreConsentServer interface {
	ConfirmInitiatorConsent(context.Context, *ConfirmInitiatorConsentRequest) (*ConfirmInitiatorConsentResponse, error)
}

// UnimplementedVideoServicePreConsentServer returns "not implemented" for the RPC,
// allowing the server to compile before the method is wired.
type UnimplementedVideoServicePreConsentServer struct{}

func (UnimplementedVideoServicePreConsentServer) ConfirmInitiatorConsent(context.Context, *ConfirmInitiatorConsentRequest) (*ConfirmInitiatorConsentResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method ConfirmInitiatorConsent not implemented")
}

// ============================================================================
// gRPC handler function
// ============================================================================

func VideoService_ConfirmInitiatorConsent_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ConfirmInitiatorConsentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	preConsentSrv, ok := srv.(VideoServicePreConsentServer)
	if !ok {
		return nil, status.Error(codes.Unimplemented, "method ConfirmInitiatorConsent not implemented")
	}
	if interceptor == nil {
		return preConsentSrv.ConfirmInitiatorConsent(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: VideoService_ConfirmInitiatorConsent_FullMethodName}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return preConsentSrv.ConfirmInitiatorConsent(ctx, req.(*ConfirmInitiatorConsentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// ============================================================================
// Ensure emptypb is imported (used in sibling ext files)
// ============================================================================

var _ = (*emptypb.Empty)(nil)
