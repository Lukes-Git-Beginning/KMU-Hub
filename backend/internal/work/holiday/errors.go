package holiday

import "fmt"

var (
	ErrHolidayNotFound     = fmt.Errorf("holiday not found")
	ErrInvalidCountryCode  = fmt.Errorf("invalid country code: must be DE, AT, or CH")
	ErrInvalidYear         = fmt.Errorf("invalid year: must be between 2000 and 2100")
	ErrSeedFailed          = fmt.Errorf("failed to seed holidays")
	ErrNagerAPIUnavailable = fmt.Errorf("nager.date API client is not configured")
)

// validCountryCodes contains the supported DACH country codes
var validCountryCodes = map[string]bool{
	"DE": true,
	"AT": true,
	"CH": true,
}
