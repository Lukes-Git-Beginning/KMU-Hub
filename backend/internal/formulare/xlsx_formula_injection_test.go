package formulare

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestBuildXLSX_FormulaLikeAnswerIsStoredAsPlainString locks in that buildXLSX's use of
// excelize.SetCellValue never produces a live formula cell for user-controlled answer text,
// even when that text starts with a formula-trigger character (=, +, -, @). Unlike CSV, a
// native XLSX cell carries its type in the sheet XML (t="s" for a shared string); Excel only
// auto-detects "=..." as a formula during text/CSV import, not when the cell is explicitly
// typed as a string. If a future change swaps SetCellValue for SetCellFormula or otherwise
// changes this, this test must fail.
func TestBuildXLSX_FormulaLikeAnswerIsStoredAsPlainString(t *testing.T) {
	answers, err := json.Marshal(map[string]any{"comment": "=cmd|'/c calc'!A1"})
	if err != nil {
		t.Fatal(err)
	}
	submissions := []*FormSubmission{
		{
			ID:          uuid.New(),
			Answers:     answers,
			Status:      FormSubmissionStatusNew,
			SubmittedAt: time.Now().UTC(),
		},
	}

	content, err := buildXLSX(submissions, []string{"comment"})
	if err != nil {
		t.Fatalf("buildXLSX: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("open xlsx as zip: %v", err)
	}

	var sheetXML, sharedStringsXML []byte
	for _, zf := range zr.File {
		switch zf.Name {
		case "xl/worksheets/sheet1.xml":
			sheetXML = readZipFile(t, zf)
		case "xl/sharedStrings.xml":
			sharedStringsXML = readZipFile(t, zf)
		}
	}
	if sheetXML == nil {
		t.Fatal("sheet1.xml not found in generated workbook")
	}

	sheetStr := string(sheetXML)
	if bytes.Contains(sheetXML, []byte(`<f>`)) {
		t.Fatalf("sheet contains a live formula element <f>, expected plain string cells only:\n%s", sheetStr)
	}
	// Column F holds the "comment" answer (A-E are id/submitted_at/status/submitted_by/ip_address).
	if !bytes.Contains(sheetXML, []byte(`<c r="F2" t="s">`)) {
		t.Fatalf("expected F2 to be typed as a shared string (t=\"s\"), got:\n%s", sheetStr)
	}

	if sharedStringsXML == nil {
		t.Fatal("sharedStrings.xml not found in generated workbook")
	}
	if !bytes.Contains(sharedStringsXML, []byte(`=cmd|`)) {
		t.Fatalf("expected the raw answer text to be preserved verbatim in sharedStrings.xml, got:\n%s", string(sharedStringsXML))
	}
}

func readZipFile(t *testing.T, zf *zip.File) []byte {
	t.Helper()
	rc, err := zf.Open()
	if err != nil {
		t.Fatalf("open %s: %v", zf.Name, err)
	}
	defer rc.Close()
	b, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read %s: %v", zf.Name, err)
	}
	return b
}
