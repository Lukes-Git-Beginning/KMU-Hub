package settings

import "errors"

// ErrNotFound is returned when a module-lead or setting record does not exist.
var ErrNotFound = errors.New("settings: record not found")

// ErrNotModuleLead is returned when a user attempts to write tenant-scope settings
// but is neither an admin nor a Modul-Leiter for the module.
var ErrNotModuleLead = errors.New("settings: caller is not a module-lead for this module")

// ErrInvalidModuleID is returned when the module_id is empty.
var ErrInvalidModuleID = errors.New("settings: module_id must not be empty")

// ErrInvalidKey is returned when a setting key is empty.
var ErrInvalidKey = errors.New("settings: setting key must not be empty")

// ErrNotAdmin is returned when a non-admin attempts an admin-only operation
// (e.g. granting or revoking module access).
var ErrNotAdmin = errors.New("settings: caller is not an admin")

// ErrInvalidValueSet is returned when a value-set payload is malformed as a
// whole (empty name, no options, too many options).
var ErrInvalidValueSet = errors.New("settings: invalid value-set")

// ErrInvalidValueSetKey is returned when a value-set key does not match the
// slug pattern shared with migration 000295.
var ErrInvalidValueSetKey = errors.New("settings: invalid value-set key")

// ErrInvalidValueSetOption is returned when a single option is malformed
// (bad key, duplicate key, empty or oversized label, oversized color).
var ErrInvalidValueSetOption = errors.New("settings: invalid value-set option")

// ErrUnknownModule is returned when a module id is not in the catalogue
// (internal/modules). Guessing would let a typo create an activation row that
// no UI can ever show.
var ErrUnknownModule = errors.New("settings: unknown module id")
