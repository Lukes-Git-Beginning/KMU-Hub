package formulare

import "errors"

var (
	ErrSchemaNotFound       = errors.New("formulare schema not found")
	ErrSubmissionNotFound   = errors.New("formulare submission not found")
	ErrWebhookNotFound      = errors.New("formulare webhook not found")
	ErrDeliveryNotFound     = errors.New("formulare delivery not found")
	ErrSchemaDeleted        = errors.New("formulare schema is deleted")
	ErrInvalidFields        = errors.New("formulare fields JSON is invalid")
	ErrInvalidStatus        = errors.New("formulare status is invalid")
	ErrWebhookInactive      = errors.New("formulare webhook is inactive")
	ErrInvalidExportFormat  = errors.New("formulare export format is invalid")
	ErrInvalidURL           = errors.New("formulare webhook URL is invalid")

	// Share links (B8). ErrShareLinkNotFound is deliberately the single answer
	// to unknown, malformed, revoked, expired, used-up and schema-closed
	// links: the public route must not be an oracle for which tokens exist.
	ErrShareLinkNotFound             = errors.New("formulare share link not found")
	ErrInvalidShareLink              = errors.New("formulare share link parameters are invalid")
	ErrSchemaNotPublic               = errors.New("formulare schema is not public")
	ErrAnswersTooLarge               = errors.New("formulare submission answers are too large")
	ErrShareLinkFileFieldUnsupported = errors.New("formulare public submission does not support file fields")
)
