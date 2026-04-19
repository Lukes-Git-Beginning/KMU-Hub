package export_test

import (
	"errors"
	"testing"

	"github.com/kmuhub/kmuhub/internal/berichte/export"
)

func TestNewExporter_knownFormats(t *testing.T) {
	t.Parallel()

	cases := []struct {
		format    string
		wantCT    string
		wantExt   string
	}{
		{"pdf", "application/pdf", ".pdf"},
		{"csv", "text/csv; charset=utf-8", ".csv"},
		{"xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", ".xlsx"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.format, func(t *testing.T) {
			t.Parallel()
			e, err := export.NewExporter(tc.format)
			if err != nil {
				t.Fatalf("NewExporter(%q) unexpected error: %v", tc.format, err)
			}
			if got := e.ContentType(); got != tc.wantCT {
				t.Errorf("ContentType = %q, want %q", got, tc.wantCT)
			}
			if got := e.FileExtension(); got != tc.wantExt {
				t.Errorf("FileExtension = %q, want %q", got, tc.wantExt)
			}
		})
	}
}

func TestNewExporter_unknownFormat(t *testing.T) {
	t.Parallel()

	_, err := export.NewExporter("docx")
	if err == nil {
		t.Fatal("expected error for unknown format, got nil")
	}
	if !errors.Is(err, export.ErrUnsupportedFormat) {
		t.Errorf("error should wrap ErrUnsupportedFormat, got: %v", err)
	}
}

func TestNewExporter_emptyFormat(t *testing.T) {
	t.Parallel()

	_, err := export.NewExporter("")
	if !errors.Is(err, export.ErrUnsupportedFormat) {
		t.Errorf("empty format should return ErrUnsupportedFormat, got: %v", err)
	}
}
