package export

import (
	"errors"
	"fmt"
	"io"

	"github.com/kmuhub/kmuhub/internal/berichte"
)

// ErrUnsupportedFormat is returned when NewExporter receives an unknown format key.
var ErrUnsupportedFormat = errors.New("export: unsupported format")

// Exporter writes a ReportResult into a target writer for a specific format.
type Exporter interface {
	Export(result *berichte.ReportResult, w io.Writer) error
	ContentType() string
	FileExtension() string
}

// NewExporter returns an Exporter for the given format key ("pdf", "csv", "xlsx").
func NewExporter(format string) (Exporter, error) {
	switch format {
	case "pdf":
		return &PDFExporter{}, nil
	case "csv":
		return &CSVExporter{}, nil
	case "xlsx":
		return &XLSXExporter{}, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, format)
}
