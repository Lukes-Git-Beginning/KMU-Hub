package video

import "errors"

var (
	ErrNotFound              = errors.New("call session not found")
	ErrCallEnded             = errors.New("call has already ended")
	ErrMaxParticipants       = errors.New("call has reached maximum participants")
	ErrLiveKitNotConfigured  = errors.New("livekit is not configured")
	ErrInvalidCallType       = errors.New("invalid call type")
	ErrNoTargetUsers         = errors.New("at least one target user is required")
)
