package pipelinestage

import "errors"

var (
	ErrStageNotFound      = errors.New("pipeline stage not found")
	ErrNameRequired       = errors.New("stage name is required")
	ErrInvalidColor       = errors.New("invalid color format, must be hex like #3b82f6")
	ErrInvalidProbability = errors.New("probability must be between 0 and 100")
	ErrStageHasDeals      = errors.New("stage has deals and cannot be deleted")
	ErrWonStageExists     = errors.New("a won stage already exists")
	ErrLostStageExists    = errors.New("a lost stage already exists")
	ErrInvalidReorder     = errors.New("invalid reorder: all stage IDs must be provided exactly once")
)
