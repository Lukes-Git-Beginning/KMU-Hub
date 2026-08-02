package export

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kmuhub/kmuhub/internal/berichte"
)

// Rendered is a fully materialised report: the encoded bytes plus the MIME
// metadata a transport (mail attachment, HTTP response) needs.
type Rendered struct {
	Payload     []byte
	ContentType string
	Filename    string
}

// Render encodes result in the given format and derives a download filename
// from the report name and the result's generation date, so a recipient who
// gets the same scheduled report every week can tell the runs apart.
func Render(result *berichte.ReportResult, name, format string) (*Rendered, error) {
	if result == nil {
		return nil, errors.New("export: nil report result")
	}
	exporter, err := NewExporter(format)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := exporter.Export(result, &buf); err != nil {
		return nil, fmt.Errorf("export %s: %w", format, err)
	}

	stamp := result.Meta.GeneratedAt
	if stamp.IsZero() {
		stamp = time.Now()
	}

	return &Rendered{
		Payload:     buf.Bytes(),
		ContentType: exporter.ContentType(),
		Filename:    fmt.Sprintf("%s_%s%s", filenameSlug(name), stamp.Format("2006-01-02"), exporter.FileExtension()),
	}, nil
}

// filenameSlug reduces a report name to a safe filename stem. German umlauts
// are transliterated rather than dropped -- "Umsatzübersicht" must not become
// "umsatzbersicht".
func filenameSlug(name string) string {
	var b strings.Builder
	prevSep := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch r {
		case 'ä':
			b.WriteString("ae")
			prevSep = false
			continue
		case 'ö':
			b.WriteString("oe")
			prevSep = false
			continue
		case 'ü':
			b.WriteString("ue")
			prevSep = false
			continue
		case 'ß':
			b.WriteString("ss")
			prevSep = false
			continue
		}
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevSep = false
		default:
			if !prevSep {
				b.WriteRune('-')
				prevSep = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "bericht"
	}
	return slug
}
