package file

import "errors"

// Sentinel errors for file operations.
var (
	ErrFileNotFound      = errors.New("file not found")
	ErrFilenameRequired  = errors.New("filename is required")
	ErrFilenameTooLong   = errors.New("filename must be 255 characters or less")
	ErrFileSizeZero      = errors.New("file size must be greater than zero")
	ErrFileTooLarge      = errors.New("file exceeds maximum allowed size")
	ErrStorageKeyMissing = errors.New("storage key is required")
	ErrVersionNotFound   = errors.New("file version not found")
	ErrVersionConflict   = errors.New("version conflict: file has been updated")
	ErrFileDeleted       = errors.New("file has been deleted")
	ErrNoWritePermission = errors.New("write permission required for this operation")
	ErrInvalidEntityType = errors.New("invalid entity type for linking")

	ErrCommentNotFound           = errors.New("comment not found")
	ErrCommentContentRequired    = errors.New("comment content is required")
	ErrCommentContentTooLong     = errors.New("comment content must be 10000 characters or less")
	ErrCannotEditOthersComment   = errors.New("only the author can edit this comment")
	ErrCannotDeleteOthersComment = errors.New("only the author or an admin can delete this comment")

	// ErrShareLinkNotFound covers every way a share link fails on the
	// authenticated management side (unknown id, wrong tenant).
	ErrShareLinkNotFound = errors.New("share link not found")
	// ErrShareLinkInvalid is the single answer the unauthenticated public
	// redemption route gives for an unknown token, a revoked or expired link,
	// a missing password and a wrong password alike — see
	// Service.RedeemShareLink for why these must not be told apart.
	ErrShareLinkInvalid       = errors.New("share link is invalid or has expired")
	ErrShareLinkExpiryInvalid = errors.New("share link expiry is invalid")
	ErrSharePasswordTooLong   = errors.New("share link password is too long")
)

// Allowed entity types for document entity links.
var AllowedEntityTypes = map[string]bool{
	"contact": true,
	"company": true,
	"deal":    true,
	"project": true,
	"task":    true,
}
