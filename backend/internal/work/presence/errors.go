package presence

import "errors"

var (
	ErrInvalidManualStatus = errors.New("only online, away, and dnd can be set manually")
	ErrInvalidAwayTimeout  = errors.New("away timeout must be between 60 and 3600 seconds")
)
