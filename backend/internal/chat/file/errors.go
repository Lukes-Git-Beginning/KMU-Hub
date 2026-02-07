package file

import "errors"

var (
	ErrNotChannelMember = errors.New("user is not a member of this channel")
	ErrChannelArchived  = errors.New("channel is archived")
	ErrFileTooLarge     = errors.New("file exceeds maximum size limit")
	ErrInvalidMimeType  = errors.New("file type is not allowed")
	ErrQuotaExceeded    = errors.New("storage quota exceeded")
	ErrFileNotFound     = errors.New("file not found")
	ErrFileDeleted      = errors.New("file has been deleted")
	ErrNotAuthorized    = errors.New("not authorized to perform this action")
	ErrScanFailed       = errors.New("file failed security scan")
)
