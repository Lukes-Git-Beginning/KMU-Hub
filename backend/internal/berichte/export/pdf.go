package export

import (
	"fmt"
	"io"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/consts/pagesize"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"

	"github.com/kmuhub/kmuhub/internal/berichte"
)

// PDFExporter renders a ReportResult as an A4 PDF using maroto/v2.
// Header row is bold; data rows alternate between white and light-grey backgrounds.
type PDFExporter struct{}

func (e *PDFExporter) ContentType() string  { return "application/pdf" }
func (e *PDFExporter) FileExtension() string { return ".pdf" }

// Export writes the PDF representation of result into w.
func (e *PDFExporter) Export(result *berichte.ReportResult, w io.Writer) error {
	cfg := config.NewBuilder().
		WithPageSize(pagesize.A4).
		WithLeftMargin(10).
		WithTopMargin(10).
		WithRightMargin(10).
		Build()

	m := maroto.New(cfg)
	m.AddRows(resultRows(result)...)

	doc, err := m.Generate()
	if err != nil {
		return fmt.Errorf("pdf generate: %w", err)
	}

	_, err = w.Write(doc.GetBytes())
	return err
}

// resultRows renders a ReportResult as a header row (bold, dark background)
// plus alternating-shade data rows. Shared with DocumentPDFExporter, which
// embeds the same table for chart/table blocks that resolve to a saved
// definition (see document_pdf.go).
func resultRows(result *berichte.ReportResult) []core.Row {
	// Distribute columns evenly across the 12-grid.
	numCols := len(result.Columns)
	if numCols == 0 {
		numCols = 1
	}
	colWidth := 12 / numCols
	if colWidth < 1 {
		colWidth = 1
	}

	// Header row — bold text, dark background.
	headerCols := make([]core.Col, 0, numCols)
	for _, c := range result.Columns {
		label := c.Label
		if label == "" {
			label = c.Name
		}
		headerCols = append(headerCols, col.New(colWidth).Add(
			text.New(label, props.Text{
				Style: fontstyle.Bold,
				Size:  9,
				Align: align.Center,
				Color: &props.Color{Red: 255, Green: 255, Blue: 255},
			}),
		))
	}
	// Handle column count not divisible by 12 — widen the last column.
	if len(headerCols) > 0 {
		used := colWidth * numCols
		if used < 12 {
			headerCols[len(headerCols)-1] = col.New(colWidth + (12 - used)).Add(
				text.New(func() string {
					last := result.Columns[len(result.Columns)-1]
					if last.Label != "" {
						return last.Label
					}
					return last.Name
				}(), props.Text{
					Style: fontstyle.Bold,
					Size:  9,
					Align: align.Center,
					Color: &props.Color{Red: 255, Green: 255, Blue: 255},
				}),
			)
		}
	}

	headerBg := &props.Color{Red: 50, Green: 50, Blue: 50}
	rows := []core.Row{row.New(8).WithStyle(&props.Cell{BackgroundColor: headerBg}).Add(headerCols...)}

	// Guard: no data rows → show placeholder.
	if len(result.Rows) == 0 {
		rows = append(rows, row.New(8).Add(
			col.New(12).Add(
				text.New("Keine Daten", props.Text{
					Size:  9,
					Align: align.Center,
					Color: &props.Color{Red: 120, Green: 120, Blue: 120},
				}),
			),
		))
	}

	lightGrey := &props.Color{Red: 245, Green: 245, Blue: 245}

	for i, dataRow := range result.Rows {
		dataCols := make([]core.Col, 0, numCols)
		for _, c := range result.Columns {
			val := fmt.Sprintf("%v", dataRow[c.Name])
			dataCols = append(dataCols, col.New(colWidth).Add(
				text.New(val, props.Text{Size: 8, Align: align.Left}),
			))
		}
		// Widen last column if needed (same logic as header).
		if len(dataCols) > 0 {
			used := colWidth * numCols
			if used < 12 {
				lastC := result.Columns[len(result.Columns)-1]
				val := fmt.Sprintf("%v", dataRow[lastC.Name])
				dataCols[len(dataCols)-1] = col.New(colWidth + (12 - used)).Add(
					text.New(val, props.Text{Size: 8, Align: align.Left}),
				)
			}
		}

		r := row.New(7)
		// Alternate row shading for readability.
		if i%2 == 1 {
			r = r.WithStyle(&props.Cell{BackgroundColor: lightGrey})
		}
		rows = append(rows, r.Add(dataCols...))
	}

	return rows
}
