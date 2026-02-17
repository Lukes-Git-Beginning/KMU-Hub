package search

import (
	"context"
	"io"
	"log/slog"
	"strings"

	"code.sajari.com/docconv/v2"
)

// DefaultMaxIndexBytes is the default maximum number of bytes to index from
// extracted text. 1 MB covers most business documents while keeping the
// tsvector index manageable.
const DefaultMaxIndexBytes int64 = 1048576

// extractableMIMETypes maps MIME types that the extractor can process to true.
// O(1) lookup via map.
var extractableMIMETypes = map[string]bool{
	"application/pdf":          true,
	"application/msword":       true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   true,
	"application/vnd.ms-excel": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         true,
	"application/vnd.ms-powerpoint": true,
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
	"text/plain":    true,
	"text/csv":      true,
	"text/markdown": true,
	"text/html":     true,
	"application/rtf": true,
}

// Extractor wraps docconv for text extraction from multiple file formats.
type Extractor struct {
	maxBytes int64 // max bytes to index (truncate beyond this)
}

// NewExtractor creates a new Extractor with the given maximum index size in bytes.
// If maxIndexBytes is <= 0, DefaultMaxIndexBytes is used.
func NewExtractor(maxIndexBytes int64) *Extractor {
	if maxIndexBytes <= 0 {
		maxIndexBytes = DefaultMaxIndexBytes
	}
	return &Extractor{
		maxBytes: maxIndexBytes,
	}
}

// CanExtract returns true if the given MIME type is supported for text extraction.
// The mimeType is normalized to lowercase before checking.
func (e *Extractor) CanExtract(mimeType string) bool {
	return extractableMIMETypes[strings.ToLower(strings.TrimSpace(mimeType))]
}

// Extract reads content from the reader and extracts searchable text using docconv.
// The result is truncated to maxBytes. On error, an empty string is returned and
// the error is logged at Warn level (extraction failures should not block uploads).
func (e *Extractor) Extract(ctx context.Context, reader io.Reader, mimeType string) (string, error) {
	normalizedMIME := strings.ToLower(strings.TrimSpace(mimeType))
	if !extractableMIMETypes[normalizedMIME] {
		slog.Warn("unsupported MIME type for extraction",
			"mime_type", mimeType,
		)
		return "", ErrUnsupportedFormat
	}

	res, err := docconv.Convert(reader, normalizedMIME, false)
	if err != nil {
		slog.Warn("text extraction failed",
			"mime_type", normalizedMIME,
			"error", err,
		)
		return "", nil
	}

	body := res.Body
	if int64(len(body)) > e.maxBytes {
		body = body[:e.maxBytes]
	}

	return body, nil
}

// SupportedMIMETypes returns a list of all MIME types that the extractor supports.
func (e *Extractor) SupportedMIMETypes() []string {
	types := make([]string, 0, len(extractableMIMETypes))
	for mt := range extractableMIMETypes {
		types = append(types, mt)
	}
	return types
}
