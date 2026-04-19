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
)
