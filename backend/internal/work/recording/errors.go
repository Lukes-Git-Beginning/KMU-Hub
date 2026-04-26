package recording

import "errors"

var (
	ErrNotFound             = errors.New("recording not found")
	ErrConsentPending       = errors.New("recording cannot start: not all participants have responded to consent")
	ErrEgressNotConfigured  = errors.New("egress/recording is not configured")
	ErrRecordingNotActive   = errors.New("recording is not in active state")
	ErrInvalidStatus        = errors.New("invalid recording status transition")
	ErrNoCallOrMeeting      = errors.New("recording must be associated with a call or meeting")
	ErrNoParticipants       = errors.New("recording requires at least one participant")
	ErrNotExpired           = errors.New("recording retention period has not elapsed")
)
