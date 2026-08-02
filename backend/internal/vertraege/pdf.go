package vertraege

import (
	"fmt"
	"strings"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/consts/pagesize"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// renderContractPDF renders a contract and its parties as a paginated A4 PDF.
// Rows use auto height (text.NewAutoRow) throughout so long notes and party
// lists wrap and page-break correctly instead of clipping on a single page.
func renderContractPDF(c *Contract) ([]byte, error) {
	cfg := config.NewBuilder().
		WithPageSize(pagesize.A4).
		WithLeftMargin(15).
		WithTopMargin(15).
		WithRightMargin(15).
		WithPageNumber().
		Build()

	m := maroto.New(cfg)

	m.AddRows(row.New(10).Add(
		col.New(8).Add(text.New("Vertrag", props.Text{Size: 16, Style: fontstyle.Bold})),
		col.New(4).Add(text.New(c.ContractNumber, props.Text{Size: 9, Align: align.Right})),
	))
	m.AddRows(text.NewAutoRow(c.Title, props.Text{Size: 11, Style: fontstyle.Bold}))
	m.AddRows(text.NewAutoRow(fmt.Sprintf("Status: %s", contractStatusLabel(c.Status)), props.Text{Size: 9}))
	m.AddRows(text.NewAutoRow(fmt.Sprintf("Art: %s", contractTypeLabel(c.ContractType)), props.Text{Size: 9}))
	m.AddRows(text.NewAutoRow(fmt.Sprintf("Beginn: %s", c.StartsOn.Format("02.01.2006")), props.Text{Size: 9}))
	m.AddRows(text.NewAutoRow(fmt.Sprintf("Ende: %s", contractEndLabel(c)), props.Text{Size: 9}))

	if strings.TrimSpace(c.Notes) != "" {
		m.AddRows(row.New(3))
		m.AddRows(text.NewAutoRow("Notizen", props.Text{Size: 10, Style: fontstyle.Bold}))
		for _, paragraph := range notesParagraphs(c.Notes) {
			m.AddRows(text.NewAutoRow(paragraph, props.Text{Size: 9}))
		}
	}

	if len(c.Parties) > 0 {
		m.AddRows(row.New(3))
		m.AddRows(text.NewAutoRow("Vertragsparteien", props.Text{Size: 10, Style: fontstyle.Bold}))
		for _, p := range c.Parties {
			m.AddRows(text.NewAutoRow(partyLine(p), props.Text{Size: 8}))
		}
	}

	m.AddRows(row.New(6))
	m.AddRows(text.NewAutoRow(contractSignatureLine(c), props.Text{Size: 8, Style: fontstyle.Italic}))

	doc, err := m.Generate()
	if err != nil {
		return nil, fmt.Errorf("generate contract pdf: %w", err)
	}
	return doc.GetBytes(), nil
}

func contractStatusLabel(status ContractStatus) string {
	switch status {
	case ContractStatusDraft:
		return "Entwurf"
	case ContractStatusActive:
		return "Aktiv"
	case ContractStatusExpired:
		return "Abgelaufen"
	case ContractStatusTerminated:
		return "Gekuendigt"
	default:
		return string(status)
	}
}

func contractTypeLabel(t ContractType) string {
	switch t {
	case ContractTypeRental:
		return "Miete"
	case ContractTypeService:
		return "Dienstleistung"
	case ContractTypeEmployment:
		return "Arbeitsvertrag"
	case ContractTypeNDA:
		return "Geheimhaltung"
	case ContractTypeOther:
		return "Sonstiges"
	default:
		return string(t)
	}
}

func contractEndLabel(c *Contract) string {
	if c.EndsOn == nil {
		return "unbefristet"
	}
	return c.EndsOn.Format("02.01.2006")
}

// notesParagraphs splits notes on blank lines so each paragraph becomes its
// own auto-height row. This keeps maroto's per-row page-break working across
// long contract texts instead of relying on one giant row that could
// overflow past a single page.
// lean: no reflow across a single overlong paragraph; split notes into
// shorter paragraphs upstream if that ever becomes a real document.
func notesParagraphs(notes string) []string {
	raw := strings.Split(strings.ReplaceAll(notes, "\r\n", "\n"), "\n")
	var paragraphs []string
	for _, p := range raw {
		if p = strings.TrimSpace(p); p != "" {
			paragraphs = append(paragraphs, p)
		}
	}
	return paragraphs
}

func partyLine(p *ContractParty) string {
	name := partyName(p)
	line := fmt.Sprintf("%s (%s)", name, p.RoleInContract)
	if p.SignedOn != nil {
		line += fmt.Sprintf(" - unterschrieben am %s", p.SignedOn.Format("02.01.2006"))
	}
	return line
}

func partyName(p *ContractParty) string {
	if p.ExternalName != nil && strings.TrimSpace(*p.ExternalName) != "" {
		return *p.ExternalName
	}
	switch p.PartyType {
	case PartyTypeContact:
		if p.ContactID != nil {
			return fmt.Sprintf("Kontakt %s", p.ContactID.String()[:8])
		}
	case PartyTypeCompany:
		if p.CompanyID != nil {
			return fmt.Sprintf("Firma %s", p.CompanyID.String()[:8])
		}
	}
	return string(p.PartyType)
}

// contractSignatureLine describes the customer-signature state. Embedding the
// stored signature image is left for later — no maroto image embedding
// exists anywhere in this codebase yet (see internal/rapporte/pdf.go).
// lean: text-only signature line; embed the stored image once a customer asks for it in the PDF.
func contractSignatureLine(c *Contract) string {
	if c.SignedBy == nil || strings.TrimSpace(*c.SignedBy) == "" {
		return "Unterschrift: ausstehend"
	}
	signedAt := ""
	if c.SignedAt != nil {
		signedAt = " am " + c.SignedAt.Format("02.01.2006 15:04")
	}
	return fmt.Sprintf("Unterschrieben von %s%s", *c.SignedBy, signedAt)
}
