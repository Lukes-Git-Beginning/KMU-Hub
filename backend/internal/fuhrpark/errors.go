package fuhrpark

import "errors"

var (
	ErrVehicleNotFound       = errors.New("vehicle not found")
	ErrServiceNotFound       = errors.New("vehicle service not found")
	ErrDamageNotFound        = errors.New("vehicle damage not found")
	ErrFuelLogNotFound       = errors.New("fuel log not found")
	ErrTripLogNotFound       = errors.New("trip log not found")
	ErrDocumentNotFound      = errors.New("vehicle document not found")
	ErrDriverLicenseNotFound = errors.New("driver license check not found")
	ErrDriverNotFound        = errors.New("driver not found in this tenant")
	ErrInvalidInput          = errors.New("fuhrpark input is invalid")
	ErrPlateTaken            = errors.New("license plate already taken for this tenant")
	ErrInvalidTransition     = errors.New("status transition is not allowed")
)
