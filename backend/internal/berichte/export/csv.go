package export

import (
	"fmt"
	"io"
	"strings"

	"github.com/kmuhub/kmuhub/internal/berichte"
)

// CSVExporter writes a ReportResult as UTF-8 CSV with:
//   - BOM prefix (0xEF 0xBB 0xBF) for Excel/DATEV compatibility
//   - Semicolon separator (DATEV standard; comma causes field splits in German locales)
type CSVExporter struct{}

func (e *CSVExporter) ContentType() string  { return "text/csv; charset=utf-8" }
func (e *CSVExporter) FileExtension() string { return ".csv" }

// Export writes the CSV representation of result into w.
func (e *CSVExporter) Export(result *berichte.ReportResult, w io.Writer) error {
	// UTF-8 BOM — required for Excel and DATEV import to detect encoding correctly.
	if _, err := w.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return fmt.Errorf("csv write bom: %w", err)
	}

	var sb strings.Builder

	// Header line.
	headers := make([]string, 0, len(result.Columns))
	for _, c := range result.Columns {
		label := c.Label
		if label == "" {
			label = c.Name
		}
		headers = append(headers, escapeCSV(label))
	}
	sb.WriteString(strings.Join(headers, ";"))
	sb.WriteString("\n")

	// Data lines.
	for _, dataRow := range result.Rows {
		fields := make([]string, 0, len(result.Columns))
		for _, c := range result.Columns {
			val := fmt.Sprintf("%v", dataRow[c.Name])
			fields = append(fields, escapeCSV(val))
		}
		sb.WriteString(strings.Join(fields, ";"))
		sb.WriteString("\n")
	}

	_, err := io.WriteString(w, sb.String())
	return err
}

// escapeCSV wraps a field in double-quotes if it contains a semicolon, double-quote,
// or newline. Internal double-quotes are doubled per RFC 4180.
func escapeCSV(s string) string {
	if strings.ContainsAny(s, ";\"\r\n") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}
