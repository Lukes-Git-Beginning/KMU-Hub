// Package pdf provides PDF generation for financial documents using maroto/v2.
// Generates clean A4 documents with Pflichtangaben in footer following the
// Stripe/Lexoffice minimal aesthetic defined in the CONTEXT.md.
package pdf

import (
	"fmt"

	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/line"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/biz/tax"
	"github.com/kmuhub/kmuhub/internal/models"
)

// parseColor converts a hex color string (#RRGGBB) to maroto props.Color.
// Falls back to a dark blue accent if the string is invalid.
func parseColor(hex string) *props.Color {
	if len(hex) == 7 && hex[0] == '#' {
		r := hexToByte(hex[1:3])
		g := hexToByte(hex[3:5])
		b := hexToByte(hex[5:7])
		return &props.Color{Red: r, Green: g, Blue: b}
	}
	// Default accent: professional dark blue
	return &props.Color{Red: 31, Green: 78, Blue: 121}
}

func hexToByte(s string) int {
	n := 0
	for _, c := range s {
		n *= 16
		switch {
		case c >= '0' && c <= '9':
			n += int(c - '0')
		case c >= 'a' && c <= 'f':
			n += int(c-'a') + 10
		case c >= 'A' && c <= 'F':
			n += int(c-'A') + 10
		}
	}
	return n
}

// buildHeader creates the document header with company info.
func buildHeader(settings models.CompanySettings, accentColor *props.Color) core.Row {
	return row.New(25).Add(
		col.New(6).Add(
			text.New(settings.Name, props.Text{
				Size:  12,
				Style: fontstyle.Bold,
				Color: accentColor,
			}),
			text.New(settings.Street, props.Text{
				Top:  6,
				Size: 8,
			}),
			text.New(fmt.Sprintf("%s %s", settings.PLZ, settings.City), props.Text{
				Top:  10,
				Size: 8,
			}),
		),
		col.New(6).Add(
			text.New(settings.Handelsregister, props.Text{
				Size:  8,
				Align: align.Right,
			}),
			text.New(fmt.Sprintf("Steuernr.: %s", settings.Steuernummer), props.Text{
				Top:   4,
				Size:  8,
				Align: align.Right,
			}),
			text.New(fmt.Sprintf("USt-IdNr.: %s", settings.UStIDNr), props.Text{
				Top:   8,
				Size:  8,
				Align: align.Right,
			}),
		),
	)
}

// buildAccentLine creates a thin colored horizontal rule.
func buildAccentLine(accentColor *props.Color) core.Row {
	return row.New(2).Add(
		line.NewCol(12, props.Line{
			Color:     accentColor,
			Thickness: 0.5,
		}),
	)
}

// buildRecipient creates the recipient address block, including the buyer's
// USt-IdNr. when known. The invoice XML (BT-48, einvoice/generator_doc.go
// buildBuyerParty) always carries CustomerUStIDNr when set -- the PDF must
// not print something different for the same document. Omitted entirely
// when empty, like the other optional blocks in this file (buildNotesSection,
// the ValidUntil row in GenerateQuotePDF), so no dangling "USt-IdNr.: " line
// appears for buyers without one.
func buildRecipient(name, address, vatID string) core.Row {
	lines := []core.Component{
		text.New(name, props.Text{
			Size:  10,
			Style: fontstyle.Bold,
		}),
		text.New(address, props.Text{
			Top:  5,
			Size: 9,
		}),
	}
	if vatID != "" {
		lines = append(lines, text.New(fmt.Sprintf("USt-IdNr.: %s", vatID), props.Text{
			Top:  10,
			Size: 8,
		}))
	}
	return row.New(20).Add(
		col.New(8).Add(lines...),
		col.New(4),
	)
}

// buildDocumentTitle creates the document title row with number and date.
func buildDocumentTitle(title, number, date string, accentColor *props.Color) core.Row {
	return row.New(15).Add(
		col.New(8).Add(
			text.New(title, props.Text{
				Size:  14,
				Style: fontstyle.Bold,
				Color: accentColor,
			}),
		),
		col.New(4).Add(
			text.New(fmt.Sprintf("Nr. %s", number), props.Text{
				Size:  9,
				Align: align.Right,
				Style: fontstyle.Bold,
			}),
			text.New(fmt.Sprintf("Datum: %s", date), props.Text{
				Top:   5,
				Size:  9,
				Align: align.Right,
			}),
		),
	)
}

// buildLineItemHeader creates the table header row for line items.
func buildLineItemHeader() core.Row {
	gray := &props.Color{Red: 240, Green: 240, Blue: 240}
	return row.New(7).Add(
		text.NewCol(1, "Pos.", props.Text{Size: 8, Style: fontstyle.Bold, Align: align.Center}),
		text.NewCol(5, "Bezeichnung", props.Text{Size: 8, Style: fontstyle.Bold}),
		text.NewCol(2, "Menge", props.Text{Size: 8, Style: fontstyle.Bold, Align: align.Right}),
		text.NewCol(2, "Einzelpreis", props.Text{Size: 8, Style: fontstyle.Bold, Align: align.Right}),
		text.NewCol(2, "Gesamtpreis", props.Text{Size: 8, Style: fontstyle.Bold, Align: align.Right}),
	).WithStyle(&props.Cell{BackgroundColor: gray})
}

// buildLineItemRow creates a single line item row.
func buildLineItemRow(position int, item models.LineItem, isAlternate bool) core.Row {
	// Recomputed rather than read from item.LineTotal so a stale stored value cannot
	// print. Routed through tax.LineTotal for one definition of the formula, not to
	// change output: formatEUR's StringFixed(2) already rounds half away from zero,
	// so the printed cents were and stay the same.
	lineTotal := tax.LineTotal(item.Quantity, item.UnitPrice)
	r := row.New(6).Add(
		text.NewCol(1, fmt.Sprintf("%d", position), props.Text{Size: 8, Align: align.Center}),
		text.NewCol(5, item.Description, props.Text{Size: 8}),
		text.NewCol(2, item.Quantity.StringFixed(2), props.Text{Size: 8, Align: align.Right}),
		text.NewCol(2, formatEUR(item.UnitPrice), props.Text{Size: 8, Align: align.Right}),
		text.NewCol(2, formatEUR(lineTotal), props.Text{Size: 8, Align: align.Right}),
	)
	if isAlternate {
		altGray := &props.Color{Red: 248, Green: 248, Blue: 248}
		r.WithStyle(&props.Cell{BackgroundColor: altGray})
	}
	return r
}

// buildTotalsSection creates the subtotal, tax, and grand total rows.
func buildTotalsSection(breakdown models.TaxBreakdown, taxMode string) []core.Row {
	var rows []core.Row

	// Spacer
	rows = append(rows, row.New(4))

	// Thin line above totals
	rows = append(rows, row.New(1).Add(
		col.New(7),
		line.NewCol(5, props.Line{Thickness: 0.3}),
	))

	// Subtotal
	rows = append(rows, row.New(6).Add(
		col.New(7),
		text.NewCol(3, "Zwischensumme", props.Text{Size: 8, Align: align.Right}),
		text.NewCol(2, formatEUR(breakdown.Subtotal), props.Text{Size: 8, Align: align.Right}),
	))

	// Tax lines per rate
	for rate, taxAmount := range breakdown.TaxByRate {
		rows = append(rows, row.New(5).Add(
			col.New(7),
			text.NewCol(3, fmt.Sprintf("MwSt %s%%", rate), props.Text{Size: 8, Align: align.Right}),
			text.NewCol(2, formatEUR(taxAmount), props.Text{Size: 8, Align: align.Right}),
		))
	}

	// Grand total (bold)
	rows = append(rows, row.New(1).Add(
		col.New(7),
		line.NewCol(5, props.Line{Thickness: 0.5}),
	))
	rows = append(rows, row.New(7).Add(
		col.New(7),
		text.NewCol(3, "Gesamtbetrag", props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Right}),
		text.NewCol(2, formatEUR(breakdown.GrossTotal), props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Right}),
	))

	// Tax mode notes
	switch taxMode {
	case models.TaxModeReverseCharge:
		rows = append(rows, row.New(8).Add(
			text.NewCol(12, "Steuerschuldnerschaft des Leistungsempfaengers (Abschnitt 13b UStG)", props.Text{
				Size:  7,
				Style: fontstyle.Italic,
				Top:   2,
			}),
		))
	case models.TaxModeKleinunternehmer:
		rows = append(rows, row.New(8).Add(
			text.NewCol(12, "Kein Ausweis von Umsatzsteuer, da Kleinunternehmer gemaess Abschnitt 19 UStG", props.Text{
				Size:  7,
				Style: fontstyle.Italic,
				Top:   2,
			}),
		))
	}

	return rows
}

// buildNotesSection adds payment terms and notes below the totals.
func buildNotesSection(paymentTerms, notes string) []core.Row {
	var rows []core.Row

	if paymentTerms != "" {
		rows = append(rows, row.New(3))
		rows = append(rows, row.New(6).Add(
			text.NewCol(12, fmt.Sprintf("Zahlungsbedingungen: %s", paymentTerms), props.Text{Size: 8}),
		))
	}

	if notes != "" {
		rows = append(rows, row.New(3))
		rows = append(rows, row.New(10).Add(
			text.NewCol(12, notes, props.Text{Size: 8}),
		))
	}

	return rows
}

// buildFooter creates the Pflichtangaben footer with 3-column layout.
// This appears on every page per UStG section 14.
func buildFooter(settings models.CompanySettings) core.Row {
	footerColor := &props.Color{Red: 120, Green: 120, Blue: 120}
	return row.New(15).Add(
		col.New(4).Add(
			text.New(settings.Handelsregister, props.Text{
				Size:  6,
				Color: footerColor,
			}),
			text.New(fmt.Sprintf("%s, %s %s", settings.Street, settings.PLZ, settings.City), props.Text{
				Top:   3,
				Size:  6,
				Color: footerColor,
			}),
		),
		col.New(4).Add(
			text.New(fmt.Sprintf("Steuernr.: %s", settings.Steuernummer), props.Text{
				Size:  6,
				Color: footerColor,
				Align: align.Center,
			}),
			text.New(fmt.Sprintf("USt-IdNr.: %s", settings.UStIDNr), props.Text{
				Top:   3,
				Size:  6,
				Color: footerColor,
				Align: align.Center,
			}),
		),
		col.New(4).Add(
			text.New(fmt.Sprintf("Bank: %s", settings.BankName), props.Text{
				Size:  6,
				Color: footerColor,
				Align: align.Right,
			}),
			text.New(fmt.Sprintf("IBAN: %s", settings.IBAN), props.Text{
				Top:   3,
				Size:  6,
				Color: footerColor,
				Align: align.Right,
			}),
			text.New(fmt.Sprintf("BIC: %s", settings.BIC), props.Text{
				Top:   6,
				Size:  6,
				Color: footerColor,
				Align: align.Right,
			}),
		),
	)
}

// buildDunningBody creates the body text for a dunning letter at the given level.
func buildDunningBody(level int, invoiceNumber string, invoiceGrossTotal, fee, interest decimal.Decimal) []core.Row {
	var bodyText string
	var title string

	switch level {
	case 1:
		title = "Zahlungserinnerung"
		bodyText = fmt.Sprintf(
			"Sehr geehrte Damen und Herren,\n\n"+
				"wir moechten Sie hoeflich daran erinnern, dass die Rechnung %s "+
				"ueber %s EUR noch nicht beglichen wurde. "+
				"Bitte ueberweisen Sie den offenen Betrag in den naechsten Tagen.",
			invoiceNumber, formatEUR(invoiceGrossTotal),
		)
	case 2:
		title = "1. Mahnung"
		bodyText = fmt.Sprintf(
			"Sehr geehrte Damen und Herren,\n\n"+
				"trotz unserer Erinnerung konnten wir bislang keinen Zahlungseingang "+
				"fuer die Rechnung %s feststellen. Wir bitten Sie dringend, "+
				"den ausstehenden Betrag umgehend zu begleichen.",
			invoiceNumber,
		)
	case 3:
		title = "2. Mahnung / Letzte Mahnung"
		bodyText = fmt.Sprintf(
			"Sehr geehrte Damen und Herren,\n\n"+
				"trotz mehrfacher Erinnerung und Mahnung ist die Rechnung %s "+
				"weiterhin unbeglichen. Sollte keine Zahlung innerhalb von 7 Tagen "+
				"bei uns eingehen, sehen wir uns gezwungen, ein Inkassounternehmen "+
				"mit der Beitreibung der Forderung zu beauftragen.",
			invoiceNumber,
		)
	}

	var rows []core.Row

	// Title
	rows = append(rows, row.New(12).Add(
		text.NewCol(12, title, props.Text{
			Size:  14,
			Style: fontstyle.Bold,
		}),
	))

	// Body text
	rows = append(rows, row.New(30).Add(
		text.NewCol(12, bodyText, props.Text{
			Size: 9,
		}),
	))

	// Fee + interest breakdown
	rows = append(rows, row.New(3))
	rows = append(rows, row.New(6).Add(
		col.New(7),
		text.NewCol(3, "Rechnungsbetrag", props.Text{Size: 8, Align: align.Right}),
		text.NewCol(2, formatEUR(invoiceGrossTotal), props.Text{Size: 8, Align: align.Right}),
	))

	if fee.GreaterThan(decimal.Zero) {
		rows = append(rows, row.New(5).Add(
			col.New(7),
			text.NewCol(3, "Mahngebuehr", props.Text{Size: 8, Align: align.Right}),
			text.NewCol(2, formatEUR(fee), props.Text{Size: 8, Align: align.Right}),
		))
	}

	if interest.GreaterThan(decimal.Zero) {
		rows = append(rows, row.New(5).Add(
			col.New(7),
			text.NewCol(3, "Verzugszinsen", props.Text{Size: 8, Align: align.Right}),
			text.NewCol(2, formatEUR(interest), props.Text{Size: 8, Align: align.Right}),
		))
	}

	// Total demand
	totalDemand := invoiceGrossTotal.Add(fee).Add(interest)
	rows = append(rows, row.New(1).Add(
		col.New(7),
		line.NewCol(5, props.Line{Thickness: 0.5}),
	))
	rows = append(rows, row.New(7).Add(
		col.New(7),
		text.NewCol(3, "Gesamtforderung", props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Right}),
		text.NewCol(2, formatEUR(totalDemand), props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Right}),
	))

	return rows
}

// buildTextCol is a convenience wrapper for text.NewCol with default styling.
func buildTextCol(gridSize int, content string, size float64) core.Col {
	return text.NewCol(gridSize, content, props.Text{Size: size})
}

// formatEUR formats a decimal as EUR currency string.
func formatEUR(d decimal.Decimal) string {
	return fmt.Sprintf("%s EUR", d.StringFixed(2))
}
