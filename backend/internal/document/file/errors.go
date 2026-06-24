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
)

// Allowed entity types for document entity links.
var AllowedEntityTypes = map[string]bool{
	"contact": true,
	"company": true,
	"deal":    true,
	"project": true,
	"task":    true,
}
