package berichte

import "errors"

var (
	ErrDefinitionNotFound  = errors.New("berichte definition not found")
	ErrScheduleNotFound    = errors.New("berichte schedule not found")
	ErrInvalidQueryConfig  = errors.New("berichte query config is invalid")
	ErrInvalidCron         = errors.New("berichte cron expression is invalid")
	ErrInvalidFormat       = errors.New("berichte report format is invalid")
	ErrInvalidModule       = errors.New("berichte module is invalid")
	ErrInvalidKind         = errors.New("berichte kind is invalid")
	ErrCacheMiss           = errors.New("berichte cache miss")
	ErrExecutorUnavailable = errors.New("berichte executor unavailable")
	ErrDocumentNotFound    = errors.New("berichte document not found")
	ErrInvalidTitle        = errors.New("berichte document title is required")
	ErrInvalidStatus       = errors.New("berichte document status is invalid")
	ErrInvalidRows         = errors.New("berichte document rows must be a JSON array")
	ErrInvalidSettings     = errors.New("berichte document settings must be a JSON object")
	ErrDocumentTooLarge    = errors.New("berichte document exceeds the size limit")
)
