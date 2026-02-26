package lexware

import "errors"

var (
	ErrLexwareUnauthorized    = errors.New("lexware: unauthorized (invalid API key)")
	ErrLexwareRateLimited     = errors.New("lexware: rate limited (too many requests)")
	ErrLexwareNotFound        = errors.New("lexware: resource not found")
	ErrLexwareServerError     = errors.New("lexware: server error")
	ErrLexwareVersionConflict = errors.New("lexware: version conflict (HTTP 409, optimistic locking)")
)

var (
	ErrSyncConflict        = errors.New("lexware: sync conflict (concurrent modification)")
	ErrSyncAlreadyRunning  = errors.New("lexware: sync is already running for this config")
	ErrInvalidFieldMapping = errors.New("lexware: invalid field mapping configuration")
	ErrConfigNotFound      = errors.New("lexware: sync config not found")
	ErrMappingNotFound     = errors.New("lexware: entity mapping not found")
)
