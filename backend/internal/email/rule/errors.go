package rule

import "errors"

var (
	// ErrRuleNotFound is returned when a rule does not exist within the tenant.
	ErrRuleNotFound = errors.New("email rule not found")

	// ErrInvalidField is returned for a condition field the matcher cannot read.
	ErrInvalidField = errors.New("invalid rule field")

	// ErrInvalidOperator is returned for an operator outside the supported set.
	ErrInvalidOperator = errors.New("invalid rule operator")

	// ErrInvalidActionType is returned for an action other than label or move.
	ErrInvalidActionType = errors.New("invalid rule action type")

	// ErrNameRequired is returned when the rule name is blank.
	ErrNameRequired = errors.New("rule name is required")

	// ErrValueRequired is returned when the match value is blank. An empty
	// needle would match every message.
	ErrValueRequired = errors.New("rule value is required")

	// ErrTargetRequired is returned when the action target is missing.
	ErrTargetRequired = errors.New("rule action target is required")

	// ErrTargetNotFound is returned when a move action points at a folder that
	// does not belong to the caller's tenant.
	ErrTargetNotFound = errors.New("rule action target not found")
)
