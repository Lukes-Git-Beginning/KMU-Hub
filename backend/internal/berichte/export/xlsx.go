package export

import (
	"fmt"
	"io"

	"github.com/xuri/excelize/v2"

	"github.com/kmuhub/kmuhub/internal/berichte"
)

// XLSXExporter writes a ReportResult as an XLSX workbook using excelize v2.
// Sheet name: "Bericht". Row 1 is a bold header; data starts at row 2.
type XLSXExporter struct{}

func (e *XLSXExporter) ContentType() string  { return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" }
func (e *XLSXExporter) FileExtension() string { return ".xlsx" }

// Export writes the XLSX representation of result into w.
func (e *XLSXExporter) Export(result *berichte.ReportResult, w io.Writer) error {
	f := excelize.NewFile()
	defer f.Close() //nolint:errcheck // close on cleanup only; write errors captured below

	const sheet = "Bericht"
	// Rename the default "Sheet1".
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return fmt.Errorf("xlsx set sheet name: %w", err)
	}

	// Bold header style.
	bold, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
	})
	if err != nil {
		return fmt.Errorf("xlsx create style: %w", err)
	}

	// Write header row (row 1).
	for colIdx, c := range result.Columns {
		label := c.Label
		if label == "" {
			label = c.Name
		}
		cell, cErr := excelize.CoordinatesToCellName(colIdx+1, 1)
		if cErr != nil {
			return fmt.Errorf("xlsx coordinates header: %w", cErr)
		}
		if err := f.SetCellValue(sheet, cell, label); err != nil {
			return fmt.Errorf("xlsx set header cell: %w", err)
		}
		if err := f.SetCellStyle(sheet, cell, cell, bold); err != nil {
			return fmt.Errorf("xlsx set header style: %w", err)
		}
	}

	// Write data rows (starting at row 2).
	for rowIdx, dataRow := range result.Rows {
		xlRow := rowIdx + 2
		for colIdx, c := range result.Columns {
			cell, cErr := excelize.CoordinatesToCellName(colIdx+1, xlRow)
			if cErr != nil {
				return fmt.Errorf("xlsx coordinates data: %w", cErr)
			}
			if err := f.SetCellValue(sheet, cell, fmt.Sprintf("%v", dataRow[c.Name])); err != nil {
				return fmt.Errorf("xlsx set data cell: %w", err)
			}
		}
	}

	if err := f.Write(w); err != nil {
		return fmt.Errorf("xlsx write: %w", err)
	}
	return nil
}
