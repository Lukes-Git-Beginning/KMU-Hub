package server

import (
	"errors"
	"testing"

	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/chat/channel"
	"github.com/kmuhub/kmuhub/internal/chat/file"
	"github.com/kmuhub/kmuhub/internal/chat/message"
	chatsearch "github.com/kmuhub/kmuhub/internal/chat/search"
)

func TestMapChatError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		// --- Channel errors ---
		{"channel not found", channel.ErrChannelNotFound, codes.NotFound},
		{"channel name required", channel.ErrChannelNameRequired, codes.InvalidArgument},
		{"channel archived", channel.ErrChannelArchived, codes.FailedPrecondition},
		{"user not found", channel.ErrUserNotFound, codes.NotFound},
		{"not channel member", channel.ErrNotChannelMember, codes.PermissionDenied},
		{"already member", channel.ErrAlreadyMember, codes.AlreadyExists},
		{"cannot join private", channel.ErrCannotJoinPrivate, codes.PermissionDenied},
		{"cannot leave owner", channel.ErrCannotLeaveOwner, codes.FailedPrecondition},
		{"not authorized (channel)", channel.ErrNotAuthorized, codes.PermissionDenied},
		{"cannot change owner", channel.ErrCannotChangeOwner, codes.FailedPrecondition},
		{"invalid role", channel.ErrInvalidRole, codes.InvalidArgument},
		{"cannot dm self", channel.ErrCannotDMSelf, codes.InvalidArgument},
		{"cannot join dm", channel.ErrCannotJoinDM, codes.FailedPrecondition},
		{"cannot leave dm", channel.ErrCannotLeaveDM, codes.FailedPrecondition},
		{"cannot delete dm", channel.ErrCannotDeleteDM, codes.FailedPrecondition},
		// --- Message errors ---
		{"message not found", message.ErrMessageNotFound, codes.NotFound},
		{"content required", message.ErrContentRequired, codes.InvalidArgument},
		{"channel not found (msg)", message.ErrChannelNotFound, codes.NotFound},
		{"channel archived (msg)", message.ErrChannelArchived, codes.FailedPrecondition},
		{"not channel member (msg)", message.ErrNotChannelMember, codes.PermissionDenied},
		{"not message author", message.ErrNotMessageAuthor, codes.PermissionDenied},
		{"not authorized (msg)", message.ErrNotAuthorized, codes.PermissionDenied},
		{"message deleted", message.ErrMessageDeleted, codes.FailedPrecondition},
		{"parent not in channel", message.ErrParentNotInChannel, codes.InvalidArgument},
		{"nested threads", message.ErrNestedThreads, codes.InvalidArgument},
		{"parent deleted", message.ErrParentDeleted, codes.FailedPrecondition},
		{"invalid mention", message.ErrInvalidMention, codes.InvalidArgument},
		// --- File errors ---
		{"not channel member (file)", file.ErrNotChannelMember, codes.PermissionDenied},
		{"channel archived (file)", file.ErrChannelArchived, codes.FailedPrecondition},
		{"file too large", file.ErrFileTooLarge, codes.InvalidArgument},
		{"invalid mime type", file.ErrInvalidMimeType, codes.InvalidArgument},
		{"quota exceeded", file.ErrQuotaExceeded, codes.ResourceExhausted},
		{"file not found", file.ErrFileNotFound, codes.NotFound},
		{"file deleted", file.ErrFileDeleted, codes.FailedPrecondition},
		{"not authorized (file)", file.ErrNotAuthorized, codes.PermissionDenied},
		{"scan failed", file.ErrScanFailed, codes.FailedPrecondition},
		// --- Search errors ---
		{"query too short", chatsearch.ErrQueryTooShort, codes.InvalidArgument},
		{"query too long", chatsearch.ErrQueryTooLong, codes.InvalidArgument},
		// --- Fallback ---
		{"unknown error", errors.New("boom"), codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireGRPCCode(t, mapChatError(tt.err), tt.wantCode)
		})
	}
}
