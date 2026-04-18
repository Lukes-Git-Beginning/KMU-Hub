// Package videov1 contains gRPC extensions for LiveKit Egress completion callbacks.
// These hand-written stubs complement the protoc-generated code and will be replaced
// by regenerated proto output once the .proto file is compiled in CI.
package videov1

import (
	context "context"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// ============================================================================
// Request / Response message types
// ============================================================================

// CompleteRecordingByEgressRequest is sent by the gateway when a LiveKit
// egress_ended webhook is received with a successful outcome.
type CompleteRecordingByEgressRequest struct {
	// EgressID is the LiveKit egress ID stored in the recordings table.
	EgressID string `json:"egress_id"`
	// FileURL is the S3/MinIO location of the completed recording file.
	FileURL string `json:"file_url"`
	// FileSizeBytes is the size of the recording file in bytes (0 if unknown).
	FileSizeBytes int64 `json:"file_size_bytes"`
	// DurationSeconds is the recording duration, converted from LiveKit's nanoseconds.
	DurationSeconds int32 `json:"duration_seconds"`
}

// FailRecordingByEgressRequest is sent by the gateway when a LiveKit
// egress_ended webhook is received with a failure status.
type FailRecordingByEgressRequest struct {
	// EgressID is the LiveKit egress ID stored in the recordings table.
	EgressID string `json:"egress_id"`
	// Reason is a human-readable failure description from the EgressInfo.Error field.
	Reason string `json:"reason"`
}

// ============================================================================
// gRPC method names
// ============================================================================

const (
	VideoService_CompleteRecordingByEgress_FullMethodName = "/video.v1.VideoService/CompleteRecordingByEgress"
	VideoService_FailRecordingByEgress_FullMethodName     = "/video.v1.VideoService/FailRecordingByEgress"
)

// ============================================================================
// Client interface extension
//
// videoServiceClientEgress wraps an existing VideoServiceClient and adds the
// two egress-completion RPCs. The gateway calls this instead of the raw
// VideoServiceClient interface for webhook handling.
// ============================================================================

// VideoServiceEgressClient extends VideoServiceClient with egress callbacks.
type VideoServiceEgressClient interface {
	VideoServiceClient
	CompleteRecordingByEgress(ctx context.Context, in *CompleteRecordingByEgressRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	FailRecordingByEgress(ctx context.Context, in *FailRecordingByEgressRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
}

type videoServiceEgressClient struct {
	VideoServiceClient
	cc grpc.ClientConnInterface
}

// NewVideoServiceEgressClient wraps a gRPC connection and returns a client
// that implements both the base VideoServiceClient interface and the two
// egress-completion RPCs.
func NewVideoServiceEgressClient(cc grpc.ClientConnInterface) VideoServiceEgressClient {
	return &videoServiceEgressClient{
		VideoServiceClient: NewVideoServiceClient(cc),
		cc:                 cc,
	}
}

func (c *videoServiceEgressClient) CompleteRecordingByEgress(ctx context.Context, in *CompleteRecordingByEgressRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, VideoService_CompleteRecordingByEgress_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *videoServiceEgressClient) FailRecordingByEgress(ctx context.Context, in *FailRecordingByEgressRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, VideoService_FailRecordingByEgress_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ============================================================================
// Server interface extension (for video_grpc.go / R2-B)
// ============================================================================

// VideoServiceEgressServer extends VideoServiceServer with egress callbacks.
// Embed this in the gRPC server struct alongside UnimplementedVideoServiceServer.
type VideoServiceEgressServer interface {
	CompleteRecordingByEgress(context.Context, *CompleteRecordingByEgressRequest) (*emptypb.Empty, error)
	FailRecordingByEgress(context.Context, *FailRecordingByEgressRequest) (*emptypb.Empty, error)
}

// UnimplementedVideoServiceEgressServer provides default "not implemented"
// responses so that VideoGRPCServer compiles before the methods are wired up.
type UnimplementedVideoServiceEgressServer struct{}

func (UnimplementedVideoServiceEgressServer) CompleteRecordingByEgress(context.Context, *CompleteRecordingByEgressRequest) (*emptypb.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "method CompleteRecordingByEgress not implemented")
}

func (UnimplementedVideoServiceEgressServer) FailRecordingByEgress(context.Context, *FailRecordingByEgressRequest) (*emptypb.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "method FailRecordingByEgress not implemented")
}

// ============================================================================
// gRPC handler functions (registered on the existing ServiceDesc by R2-B)
// ============================================================================

func VideoService_CompleteRecordingByEgress_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CompleteRecordingByEgressRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	egressSrv, ok := srv.(VideoServiceEgressServer)
	if !ok {
		return nil, status.Error(codes.Unimplemented, "method CompleteRecordingByEgress not implemented")
	}
	if interceptor == nil {
		return egressSrv.CompleteRecordingByEgress(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: VideoService_CompleteRecordingByEgress_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return egressSrv.CompleteRecordingByEgress(ctx, req.(*CompleteRecordingByEgressRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func VideoService_FailRecordingByEgress_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FailRecordingByEgressRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	egressSrv, ok := srv.(VideoServiceEgressServer)
	if !ok {
		return nil, status.Error(codes.Unimplemented, "method FailRecordingByEgress not implemented")
	}
	if interceptor == nil {
		return egressSrv.FailRecordingByEgress(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: VideoService_FailRecordingByEgress_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return egressSrv.FailRecordingByEgress(ctx, req.(*FailRecordingByEgressRequest))
	}
	return interceptor(ctx, in, info, handler)
}
