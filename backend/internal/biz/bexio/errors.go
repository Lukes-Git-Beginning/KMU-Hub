package bexio

import "errors"

// API errors returned by the Bexio client.
var (
	ErrBexioUnauthorized = errors.New("bexio: unauthorized (invalid or expired token)")
	ErrBexioRateLimited  = errors.New("bexio: rate limited (too many requests)")
	ErrBexioNotFound     = errors.New("bexio: resource not found")
	ErrBexioServerError  = errors.New("bexio: server error")
)

// Sync errors.
var (
	ErrSyncConflict       = errors.New("bexio: sync conflict (concurrent modification)")
	ErrSyncAlreadyRunning = errors.New("bexio: sync is already running for this config")
	ErrInvalidFieldMapping = errors.New("bexio: invalid field mapping configuration")
	ErrConfigNotFound     = errors.New("bexio: sync config not found")
	ErrMappingNotFound    = errors.New("bexio: entity mapping not found")
)
