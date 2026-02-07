package file

import (
	"context"
	"io"
)

// FileScanner scans files for malware/viruses
type FileScanner interface {
	Scan(ctx context.Context, reader io.Reader, filename string) error
}

// NoOpScanner is the default scanner that does no scanning
// Replace with ClamAV implementation later
type NoOpScanner struct{}

// Scan always returns nil (no scanning)
func (s *NoOpScanner) Scan(_ context.Context, _ io.Reader, _ string) error {
	return nil
}
