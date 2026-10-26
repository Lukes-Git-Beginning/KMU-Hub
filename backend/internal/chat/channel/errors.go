package channel

import "errors"

var (
	ErrChannelNotFound        = errors.New("channel not found")
	ErrChannelNameRequired    = errors.New("channel name is required")
	ErrChannelArchived        = errors.New("channel is archived")
	ErrUserNotFound           = errors.New("user not found")
	ErrNotChannelMember       = errors.New("user is not a member of this channel")
	ErrAlreadyMember          = errors.New("user is already a member of this channel")
	ErrCannotJoinPrivate      = errors.New("cannot join private channel without invitation")
	ErrCannotLeaveOwner       = errors.New("channel owner cannot leave the channel")
	ErrNotAuthorized          = errors.New("not authorized to perform this action")
	ErrCannotChangeOwner      = errors.New("cannot change owner role")
	ErrInvalidRole            = errors.New("invalid channel role")
	ErrCannotDMSelf           = errors.New("cannot create DM with yourself")
	ErrCannotJoinDM           = errors.New("cannot join a DM channel")
	ErrCannotLeaveDM          = errors.New("cannot leave a DM channel")
	ErrCannotDeleteDM         = errors.New("cannot delete a DM channel")
	ErrChannelHasCallSessions = errors.New("channel has archived call sessions and cannot be deleted")
	ErrTooManyDMParticipants  = errors.New("too many participants for a group DM")
	ErrDMNameTooLong          = errors.New("group DM name is too long")
	// ErrGroupDMExists is returned by CreateGroupDMChannel when a concurrent
	// request created the same member set first; the service then re-reads it.
	ErrGroupDMExists = errors.New("group DM for this member set already exists")
)
