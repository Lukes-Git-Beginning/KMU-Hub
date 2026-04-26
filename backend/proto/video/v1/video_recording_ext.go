// Package videov1 contains hand-written gRPC extensions for recording tagging RPCs.
// These complement the protoc-generated code and will be replaced by regenerated
// proto output once the .proto file is compiled in CI (Sprint 4 proto-regen task).
package videov1

import (
	context "context"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// ============================================================================
// Method name constants
// ============================================================================

const (
	VideoService_GetRecordingConsents_FullMethodName     = "/video.v1.VideoService/GetRecordingConsents"
	VideoService_TagRecordingWithConsents_FullMethodName = "/video.v1.VideoService/TagRecordingWithConsents"
	VideoService_UpdateRecordingMetadata_FullMethodName  = "/video.v1.VideoService/UpdateRecordingMetadata"
	VideoService_ListRecordingsByMeeting_FullMethodName  = "/video.v1.VideoService/ListRecordingsByMeeting"
	VideoService_GetRecordingStatus_FullMethodName       = "/video.v1.VideoService/GetRecordingStatus"
	VideoService_CleanupExpiredRecording_FullMethodName  = "/video.v1.VideoService/CleanupExpiredRecording"
)

// ============================================================================
// Request / Response message types
// ============================================================================

// GetRecordingConsentsRequest requests the consent snapshot + live consent rows
// for a recording.
type GetRecordingConsentsRequest struct {
	RecordingID string `json:"recording_id"`
}

// GetRecordingConsentsResponse returns the frozen snapshot and live consent status.
type GetRecordingConsentsResponse struct {
	RecordingID  string              `json:"recording_id"`
	Consents     []*RecordingConsent `json:"consents"`
	AllConsented bool                `json:"all_consented"`
}

// ConsentSnapshotEntry represents a single participant in the consent snapshot.
type ConsentSnapshotEntry struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
	JoinedAt    string `json:"joined_at"`
}

// TagRecordingWithConsentsRequest overwrites the consent_snapshot on a recording.
type TagRecordingWithConsentsRequest struct {
	RecordingID string                  `json:"recording_id"`
	Snapshot    []*ConsentSnapshotEntry `json:"snapshot"`
}

// UpdateRecordingMetadataRequest updates mutable fields on a recording.
type UpdateRecordingMetadataRequest struct {
	RecordingID     string  `json:"recording_id"`
	FileURL         *string `json:"file_url,omitempty"`
	FileSizeBytes   *int64  `json:"file_size_bytes,omitempty"`
	DurationSeconds *int32  `json:"duration_seconds,omitempty"`
	Status          *string `json:"status,omitempty"`
}

// ListRecordingsByMeetingRequest lists all recordings for a meeting with pagination.
type ListRecordingsByMeetingRequest struct {
	MeetingID string `json:"meeting_id"`
	TenantID  string `json:"tenant_id"`
	Page      int32  `json:"page"`
	PageSize  int32  `json:"page_size"`
}

// ListRecordingsByMeetingResponse returns paginated recordings for a meeting.
type ListRecordingsByMeetingResponse struct {
	Recordings []*Recording `json:"recordings"`
	Total      int32        `json:"total"`
}

// GetRecordingStatusRequest fetches a single recording for status polling.
type GetRecordingStatusRequest struct {
	RecordingID string `json:"recording_id"`
}

// CleanupExpiredRecordingRequest requests deletion of a single expired recording.
type CleanupExpiredRecordingRequest struct {
	RecordingID string `json:"recording_id"`
}

// ============================================================================
// Client interface extension
// ============================================================================

// VideoServiceRecordingTagClient extends VideoServiceClient with recording
// tagging RPCs added in S1.7 (Sprint 2 / Welle 1.B).
type VideoServiceRecordingTagClient interface {
	VideoServiceClient
	GetRecordingConsents(ctx context.Context, in *GetRecordingConsentsRequest, opts ...grpc.CallOption) (*GetRecordingConsentsResponse, error)
	TagRecordingWithConsents(ctx context.Context, in *TagRecordingWithConsentsRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	UpdateRecordingMetadata(ctx context.Context, in *UpdateRecordingMetadataRequest, opts ...grpc.CallOption) (*Recording, error)
	ListRecordingsByMeeting(ctx context.Context, in *ListRecordingsByMeetingRequest, opts ...grpc.CallOption) (*ListRecordingsByMeetingResponse, error)
	GetRecordingStatus(ctx context.Context, in *GetRecordingStatusRequest, opts ...grpc.CallOption) (*Recording, error)
	CleanupExpiredRecording(ctx context.Context, in *CleanupExpiredRecordingRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
}

type videoServiceRecordingTagClient struct {
	VideoServiceClient
	cc grpc.ClientConnInterface
}

// NewVideoServiceRecordingTagClient wraps a gRPC connection and returns a client
// that implements both the base VideoServiceClient and the recording tagging RPCs.
func NewVideoServiceRecordingTagClient(cc grpc.ClientConnInterface) VideoServiceRecordingTagClient {
	return &videoServiceRecordingTagClient{
		VideoServiceClient: NewVideoServiceClient(cc),
		cc:                 cc,
	}
}

func (c *videoServiceRecordingTagClient) GetRecordingConsents(ctx context.Context, in *GetRecordingConsentsRequest, opts ...grpc.CallOption) (*GetRecordingConsentsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetRecordingConsentsResponse)
	err := c.cc.Invoke(ctx, VideoService_GetRecordingConsents_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *videoServiceRecordingTagClient) TagRecordingWithConsents(ctx context.Context, in *TagRecordingWithConsentsRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, VideoService_TagRecordingWithConsents_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *videoServiceRecordingTagClient) UpdateRecordingMetadata(ctx context.Context, in *UpdateRecordingMetadataRequest, opts ...grpc.CallOption) (*Recording, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Recording)
	err := c.cc.Invoke(ctx, VideoService_UpdateRecordingMetadata_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *videoServiceRecordingTagClient) ListRecordingsByMeeting(ctx context.Context, in *ListRecordingsByMeetingRequest, opts ...grpc.CallOption) (*ListRecordingsByMeetingResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ListRecordingsByMeetingResponse)
	err := c.cc.Invoke(ctx, VideoService_ListRecordingsByMeeting_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *videoServiceRecordingTagClient) GetRecordingStatus(ctx context.Context, in *GetRecordingStatusRequest, opts ...grpc.CallOption) (*Recording, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Recording)
	err := c.cc.Invoke(ctx, VideoService_GetRecordingStatus_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *videoServiceRecordingTagClient) CleanupExpiredRecording(ctx context.Context, in *CleanupExpiredRecordingRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, VideoService_CleanupExpiredRecording_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ============================================================================
// Server interface extension
// ============================================================================

// VideoServiceRecordingTagServer extends VideoServiceServer with recording
// tagging RPCs.
type VideoServiceRecordingTagServer interface {
	GetRecordingConsents(context.Context, *GetRecordingConsentsRequest) (*GetRecordingConsentsResponse, error)
	TagRecordingWithConsents(context.Context, *TagRecordingWithConsentsRequest) (*emptypb.Empty, error)
	UpdateRecordingMetadata(context.Context, *UpdateRecordingMetadataRequest) (*Recording, error)
	ListRecordingsByMeeting(context.Context, *ListRecordingsByMeetingRequest) (*ListRecordingsByMeetingResponse, error)
	GetRecordingStatus(context.Context, *GetRecordingStatusRequest) (*Recording, error)
	CleanupExpiredRecording(context.Context, *CleanupExpiredRecordingRequest) (*emptypb.Empty, error)
}

// UnimplementedVideoServiceRecordingTagServer returns "not implemented" for all
// recording tagging RPCs, allowing the server to compile before all methods are wired.
type UnimplementedVideoServiceRecordingTagServer struct{}

func (UnimplementedVideoServiceRecordingTagServer) GetRecordingConsents(context.Context, *GetRecordingConsentsRequest) (*GetRecordingConsentsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method GetRecordingConsents not implemented")
}

func (UnimplementedVideoServiceRecordingTagServer) TagRecordingWithConsents(context.Context, *TagRecordingWithConsentsRequest) (*emptypb.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "method TagRecordingWithConsents not implemented")
}

func (UnimplementedVideoServiceRecordingTagServer) UpdateRecordingMetadata(context.Context, *UpdateRecordingMetadataRequest) (*Recording, error) {
	return nil, status.Error(codes.Unimplemented, "method UpdateRecordingMetadata not implemented")
}

func (UnimplementedVideoServiceRecordingTagServer) ListRecordingsByMeeting(context.Context, *ListRecordingsByMeetingRequest) (*ListRecordingsByMeetingResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method ListRecordingsByMeeting not implemented")
}

func (UnimplementedVideoServiceRecordingTagServer) GetRecordingStatus(context.Context, *GetRecordingStatusRequest) (*Recording, error) {
	return nil, status.Error(codes.Unimplemented, "method GetRecordingStatus not implemented")
}

func (UnimplementedVideoServiceRecordingTagServer) CleanupExpiredRecording(context.Context, *CleanupExpiredRecordingRequest) (*emptypb.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "method CleanupExpiredRecording not implemented")
}

// ============================================================================
// gRPC handler functions
// ============================================================================

func VideoService_GetRecordingConsents_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRecordingConsentsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	tagSrv, ok := srv.(VideoServiceRecordingTagServer)
	if !ok {
		return nil, status.Error(codes.Unimplemented, "method GetRecordingConsents not implemented")
	}
	if interceptor == nil {
		return tagSrv.GetRecordingConsents(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: VideoService_GetRecordingConsents_FullMethodName}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return tagSrv.GetRecordingConsents(ctx, req.(*GetRecordingConsentsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func VideoService_TagRecordingWithConsents_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TagRecordingWithConsentsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	tagSrv, ok := srv.(VideoServiceRecordingTagServer)
	if !ok {
		return nil, status.Error(codes.Unimplemented, "method TagRecordingWithConsents not implemented")
	}
	if interceptor == nil {
		return tagSrv.TagRecordingWithConsents(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: VideoService_TagRecordingWithConsents_FullMethodName}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return tagSrv.TagRecordingWithConsents(ctx, req.(*TagRecordingWithConsentsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func VideoService_UpdateRecordingMetadata_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateRecordingMetadataRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	tagSrv, ok := srv.(VideoServiceRecordingTagServer)
	if !ok {
		return nil, status.Error(codes.Unimplemented, "method UpdateRecordingMetadata not implemented")
	}
	if interceptor == nil {
		return tagSrv.UpdateRecordingMetadata(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: VideoService_UpdateRecordingMetadata_FullMethodName}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return tagSrv.UpdateRecordingMetadata(ctx, req.(*UpdateRecordingMetadataRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func VideoService_ListRecordingsByMeeting_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListRecordingsByMeetingRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	tagSrv, ok := srv.(VideoServiceRecordingTagServer)
	if !ok {
		return nil, status.Error(codes.Unimplemented, "method ListRecordingsByMeeting not implemented")
	}
	if interceptor == nil {
		return tagSrv.ListRecordingsByMeeting(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: VideoService_ListRecordingsByMeeting_FullMethodName}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return tagSrv.ListRecordingsByMeeting(ctx, req.(*ListRecordingsByMeetingRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func VideoService_GetRecordingStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRecordingStatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	tagSrv, ok := srv.(VideoServiceRecordingTagServer)
	if !ok {
		return nil, status.Error(codes.Unimplemented, "method GetRecordingStatus not implemented")
	}
	if interceptor == nil {
		return tagSrv.GetRecordingStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: VideoService_GetRecordingStatus_FullMethodName}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return tagSrv.GetRecordingStatus(ctx, req.(*GetRecordingStatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func VideoService_CleanupExpiredRecording_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CleanupExpiredRecordingRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	tagSrv, ok := srv.(VideoServiceRecordingTagServer)
	if !ok {
		return nil, status.Error(codes.Unimplemented, "method CleanupExpiredRecording not implemented")
	}
	if interceptor == nil {
		return tagSrv.CleanupExpiredRecording(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: VideoService_CleanupExpiredRecording_FullMethodName}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return tagSrv.CleanupExpiredRecording(ctx, req.(*CleanupExpiredRecordingRequest))
	}
	return interceptor(ctx, in, info, handler)
}
