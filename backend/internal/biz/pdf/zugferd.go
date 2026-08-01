package pdf

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"

	"github.com/kmuhub/kmuhub/internal/biz/einvoice"
	"github.com/kmuhub/kmuhub/internal/models"
)

// ZUGFeRD profiles supported (Factur-X / EN 16931)
const (
	ZUGFeRDProfileMinimum = "MINIMUM"
	ZUGFeRDProfileBasicWL = "BASIC_WL"
	ZUGFeRDProfileEN16931 = "EN16931"
)

// GenerateZUGFeRDXML generates a Factur-X / ZUGFeRD 2.1 XML (Cross-Industry
// Invoice, EN16931 profile) for embedding into an invoice PDF.
//
// The document itself is rendered by internal/biz/einvoice, which owns the amount
// and tax-category logic shared with the XRechnung writer. Keeping a second
// renderer here would let the two outbound formats report different numbers for
// the same invoice — the reason the previous string-template writer used the raw
// quantity × unit price as the tax basis while writing the stored line total as
// the line amount, and category S even for a 0 % line.
func GenerateZUGFeRDXML(invoice models.Invoice, settings models.CompanySettings) ([]byte, error) {
	// The PDF path carries §14 UStG requirements the bare XML writer does not:
	// a payment due date and a fully populated issuer block.
	if invoice.DueDate.IsZero() {
		return nil, fmt.Errorf("zugferd: invoice %s is missing the due date (BT-9)", invoice.InvoiceNumber)
	}
	if err := ValidateCompanySettingsForPDF(settings); err != nil {
		return nil, fmt.Errorf("zugferd: %w", err)
	}

	// BT-10 (Leitweg-ID) is not stored on the invoice and only applies to
	// public-sector buyers, who receive XRechnung rather than ZUGFeRD-in-PDF.
	xmlBytes, err := einvoice.GenerateCII(invoice, settings, "")
	if err != nil {
		return nil, fmt.Errorf("zugferd: %w", err)
	}
	return xmlBytes, nil
}

// EmbedZUGFeRDXML embeds the Factur-X XML into an existing PDF as /EmbeddedFiles.
// The resulting PDF/A-3b compliant file has the XML attachment visible to e-invoice validators.
func EmbedZUGFeRDXML(pdfBytes []byte, xmlBytes []byte, invoiceNumber string) ([]byte, error) {
	fileName := fmt.Sprintf("factur-x_%s_%s.xml", invoiceNumber, time.Now().Format("20060102"))

	// Write XML to temp file (pdfcpu AddAttachments requires file paths)
	tmpDir, err := os.MkdirTemp("", "zugferd-*")
	if err != nil {
		slog.Warn("zugferd embed skipped: mkdir temp failed, returning PDF without XML attachment",
			"invoice_number", invoiceNumber, "error", err)
		return pdfBytes, nil
	}
	defer os.RemoveAll(tmpDir)

	xmlPath := filepath.Join(tmpDir, fileName)
	if err := os.WriteFile(xmlPath, xmlBytes, 0o600); err != nil {
		slog.Warn("zugferd embed skipped: write temp XML failed, returning PDF without XML attachment",
			"invoice_number", invoiceNumber, "error", err)
		return pdfBytes, nil
	}

	pdfReader := bytes.NewReader(pdfBytes)
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed

	var outBuf bytes.Buffer
	if err := api.AddAttachments(io.ReadSeeker(pdfReader), &outBuf, []string{xmlPath}, false, conf); err != nil {
		slog.Warn("zugferd embed skipped: AddAttachments failed, returning PDF without XML attachment",
			"invoice_number", invoiceNumber, "error", err)
		return pdfBytes, nil
	}

	if outBuf.Len() == 0 {
		slog.Warn("zugferd embed skipped: AddAttachments produced empty output, returning original PDF",
			"invoice_number", invoiceNumber)
		return pdfBytes, nil
	}

	return outBuf.Bytes(), nil
}
