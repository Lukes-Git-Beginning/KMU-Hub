package message

import "errors"

var (
	ErrMessageNotFound   = errors.New("message not found")
	ErrContentRequired   = errors.New("message content is required")
	ErrChannelNotFound   = errors.New("channel not found")
	ErrChannelArchived   = errors.New("channel is archived")
	ErrNotChannelMember  = errors.New("user is not a member of this channel")
	ErrNotMessageAuthor  = errors.New("user is not the author of this message")
	ErrNotAuthorized     = errors.New("not authorized to perform this action")
	ErrMessageDeleted    = errors.New("message has been deleted")
)
