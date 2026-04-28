package rapporte

import "errors"

var (
	ErrReportNotFound          = errors.New("work report not found")
	ErrLineNotFound            = errors.New("report line not found")
	ErrAttachmentNotFound      = errors.New("report attachment not found")
	ErrInvalidInput            = errors.New("rapporte input is invalid")
	ErrAlreadyApproved         = errors.New("report is already approved and cannot be modified")
	ErrInvalidStateTransition  = errors.New("report state transition is not allowed")
)
