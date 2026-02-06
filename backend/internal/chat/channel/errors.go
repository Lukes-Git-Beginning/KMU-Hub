package channel

import "errors"

var (
	ErrChannelNotFound     = errors.New("channel not found")
	ErrChannelNameRequired = errors.New("channel name is required")
	ErrChannelArchived     = errors.New("channel is archived")
	ErrUserNotFound        = errors.New("user not found")
	ErrNotChannelMember    = errors.New("user is not a member of this channel")
	ErrAlreadyMember       = errors.New("user is already a member of this channel")
	ErrCannotJoinPrivate   = errors.New("cannot join private channel without invitation")
	ErrCannotLeaveOwner    = errors.New("channel owner cannot leave the channel")
	ErrNotAuthorized       = errors.New("not authorized to perform this action")
	ErrCannotChangeOwner   = errors.New("cannot change owner role")
	ErrInvalidRole         = errors.New("invalid channel role")
)
