package rapporte

import (
	"fmt"

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
)

// renderReportPDF renders a work report and its line items as an A4 PDF.
func renderReportPDF(report *WorkReport, lines []*ReportLine) ([]byte, error) {
	cfg := config.NewBuilder().
		WithPageSize(pagesize.A4).
		WithLeftMargin(15).
		WithTopMargin(15).
		WithRightMargin(15).
		WithPageNumber().
		Build()

	m := maroto.New(cfg)

	m.AddRows(row.New(10).Add(
		col.New(8).Add(text.New("Arbeitsbericht", props.Text{Size: 16, Style: fontstyle.Bold})),
		col.New(4).Add(text.New(report.ReportDate.Format("02.01.2006"), props.Text{Size: 9, Align: align.Right})),
	))
	m.AddRows(text.NewRow(6, report.Title, props.Text{Size: 11, Style: fontstyle.Bold}))
	m.AddRows(text.NewRow(6, fmt.Sprintf("Status: %s", reportStatusLabel(report.Status)), props.Text{Size: 9}))
	if report.ProjectName != nil && *report.ProjectName != "" {
		m.AddRows(text.NewRow(6, fmt.Sprintf("Projekt: %s", *report.ProjectName), props.Text{Size: 9}))
	}
	m.AddRows(row.New(3))

	if report.Description != "" {
		m.AddRows(text.NewRow(6, report.Description, props.Text{Size: 9}))
		m.AddRows(row.New(3))
	}

	for _, meta := range reportMetaLines(report) {
		m.AddRows(text.NewRow(5, meta, props.Text{Size: 8}))
	}
	if len(report.Workers) > 0 || report.WorkStart != nil || report.Weather != nil {
		m.AddRows(row.New(3))
	}

	if len(report.Workers) > 0 {
		m.AddRows(text.NewRow(6, "Mitarbeiter", props.Text{Size: 10, Style: fontstyle.Bold}))
		for _, w := range report.Workers {
			m.AddRows(text.NewRow(5, workerLine(w), props.Text{Size: 8}))
		}
		m.AddRows(row.New(3))
	}

	if len(lines) > 0 {
		m.AddRows(buildRapportLineItemHeader())
		for _, l := range lines {
			m.AddRows(buildRapportLineItemRow(l))
		}
		m.AddRows(row.New(3))
	}

	m.AddRows(row.New(10))
	m.AddRows(text.NewRow(6, signatureLine(report), props.Text{Size: 8, Style: fontstyle.Italic}))

	doc, err := m.Generate()
	if err != nil {
		return nil, fmt.Errorf("generate report pdf: %w", err)
	}
	return doc.GetBytes(), nil
}

func reportStatusLabel(status ReportStatus) string {
	switch status {
	case StatusDraft:
		return "Entwurf"
	case StatusSubmitted:
		return "Eingereicht"
	case StatusApproved:
		return "Freigegeben"
	case StatusRejected:
		return "Abgelehnt"
	default:
		return string(status)
	}
}

// reportMetaLines renders the optional work-time/weather fields as one line each.
func reportMetaLines(report *WorkReport) []string {
	var meta []string
	if report.WorkStart != nil && report.WorkEnd != nil {
		meta = append(meta, fmt.Sprintf("Arbeitszeit: %s - %s", *report.WorkStart, *report.WorkEnd))
	}
	if report.BreakMinutes != nil {
		meta = append(meta, fmt.Sprintf("Pause: %d min", *report.BreakMinutes))
	}
	if report.Weather != nil && *report.Weather != "" {
		meta = append(meta, fmt.Sprintf("Wetter: %s", *report.Weather))
	}
	if report.Temperature != nil {
		meta = append(meta, fmt.Sprintf("Temperatur: %.1f C", *report.Temperature))
	}
	return meta
}

func workerLine(w Worker) string {
	line := w.Name
	if w.Role.Valid && w.Role.String != "" {
		line += " (" + w.Role.String + ")"
	}
	if w.Hours.Valid {
		line += fmt.Sprintf(" - %.1f Std.", w.Hours.Float64)
	}
	return line
}

// signatureLine describes the customer-signature state. Embedding the actual
// PNG/SVG signature image is left for later — no maroto image embedding
// exists anywhere in this codebase yet.
// lean: text-only signature line; embed the stored image once a customer asks for it in the PDF.
func signatureLine(report *WorkReport) string {
	if report.SignedBy == nil || *report.SignedBy == "" {
		return "Kundenunterschrift: ausstehend"
	}
	signedAt := ""
	if report.SignedAt != nil {
		signedAt = " am " + report.SignedAt.Format("02.01.2006 15:04")
	}
	return fmt.Sprintf("Unterschrieben von %s%s", *report.SignedBy, signedAt)
}

func buildRapportLineItemHeader() core.Row {
	return row.New(6).Add(
		col.New(1).Add(text.New("Pos", props.Text{Size: 8, Style: fontstyle.Bold})),
		col.New(5).Add(text.New("Beschreibung", props.Text{Size: 8, Style: fontstyle.Bold})),
		col.New(2).Add(text.New("Menge", props.Text{Size: 8, Style: fontstyle.Bold, Align: align.Right})),
		col.New(2).Add(text.New("Einheit", props.Text{Size: 8, Style: fontstyle.Bold})),
		col.New(2).Add(text.New("Notiz", props.Text{Size: 8, Style: fontstyle.Bold})),
	)
}

func buildRapportLineItemRow(l *ReportLine) core.Row {
	return row.New(6).Add(
		col.New(1).Add(text.New(fmt.Sprintf("%d", l.Position), props.Text{Size: 8})),
		col.New(5).Add(text.New(l.Description, props.Text{Size: 8})),
		col.New(2).Add(text.New(fmt.Sprintf("%.2f", l.Quantity), props.Text{Size: 8, Align: align.Right})),
		col.New(2).Add(text.New(l.Unit, props.Text{Size: 8})),
		col.New(2).Add(text.New(l.Notes, props.Text{Size: 8})),
	)
}
