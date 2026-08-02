package export_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/berichte"
	"github.com/kmuhub/kmuhub/internal/berichte/export"
)

func fixedResult() *berichte.ReportResult {
	r := sampleResult()
	r.Meta = berichte.ReportMeta{
		GeneratedAt:  time.Date(2026, 8, 2, 6, 30, 0, 0, time.UTC),
		RowCount:     2,
		DefinitionID: uuid.New(),
	}
	return r
}

func TestRender_FormatsAndFilenames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		format          string
		wantContentType string
		wantFilename    string
	}{
		{"csv", "text/csv; charset=utf-8", "umsatzuebersicht-q3_2026-08-02.csv"},
		{"pdf", "application/pdf", "umsatzuebersicht-q3_2026-08-02.pdf"},
		{"xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "umsatzuebersicht-q3_2026-08-02.xlsx"},
	}

	for _, tc := range tests {
		t.Run(tc.format, func(t *testing.T) {
			t.Parallel()

			out, err := export.Render(fixedResult(), "Umsatzübersicht Q3", tc.format)
			if err != nil {
				t.Fatalf("Render(%s): %v", tc.format, err)
			}
			if len(out.Payload) == 0 {
				t.Error("payload is empty")
			}
			if out.ContentType != tc.wantContentType {
				t.Errorf("ContentType = %q, want %q", out.ContentType, tc.wantContentType)
			}
			if out.Filename != tc.wantFilename {
				t.Errorf("Filename = %q, want %q", out.Filename, tc.wantFilename)
			}
		})
	}
}

func TestRender_UnsupportedFormat(t *testing.T) {
	t.Parallel()

	if _, err := export.Render(fixedResult(), "Bericht", "docx"); err == nil {
		t.Fatal("expected an error for an unsupported format")
	}
}

func TestRender_NilResult(t *testing.T) {
	t.Parallel()

	if _, err := export.Render(nil, "Bericht", "csv"); err == nil {
		t.Fatal("expected an error for a nil result")
	}
}

// A schedule may carry a name that reduces to nothing printable; the filename
// still has to be usable.
func TestRender_EmptyNameFallsBack(t *testing.T) {
	t.Parallel()

	out, err := export.Render(fixedResult(), "  ///  ", "csv")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.HasPrefix(out.Filename, "bericht_") {
		t.Errorf("Filename = %q, want the bericht_ fallback stem", out.Filename)
	}
}
