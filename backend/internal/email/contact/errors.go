package contact

import "errors"

var (
	// ErrImportFailed is returned when a contact import operation fails.
	ErrImportFailed = errors.New("contact import failed")

	// ErrInvalidCSV is returned when the CSV file is malformed or has invalid structure.
	ErrInvalidCSV = errors.New("invalid CSV file format")

	// ErrInvalidVCard is returned when the vCard file is malformed or has invalid structure.
	ErrInvalidVCard = errors.New("invalid vCard file format")

	// ErrInvalidXLSX is returned when the XLSX workbook is malformed or has invalid structure.
	ErrInvalidXLSX = errors.New("invalid XLSX file format")
)
