package einvoice

import (
	"bytes"
	"io"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// ExtractXMLFromPDF extracts an embedded e-invoice XML attachment from a PDF.
//
// It uses pdfcpu's ExtractAttachmentsRaw to enumerate all embedded file streams
// and returns the first attachment whose name contains known e-invoice identifiers
// (.xml extension, "factur-x", "zugferd", or "xrechnung").
//
// Returns ErrNoEmbeddedXML when no matching attachment is found.
func ExtractXMLFromPDF(pdfBytes []byte) ([]byte, error) {
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed

	attachments, err := api.ExtractAttachmentsRaw(bytes.NewReader(pdfBytes), "", nil, conf)
	if err != nil {
		return nil, ErrNoEmbeddedXML
	}

	for _, att := range attachments {
		// Check FileName first, then ID as fallback
		name := att.FileName
		if name == "" {
			name = att.ID
		}
		if !isEInvoiceAttachment(name) {
			continue
		}
		if att.Reader == nil {
			continue
		}
		data, readErr := io.ReadAll(att.Reader)
		if readErr != nil {
			continue
		}
		if len(data) > 0 {
			return data, nil
		}
	}

	return nil, ErrNoEmbeddedXML
}

// isEInvoiceAttachment returns true when the attachment filename matches the
// typical naming conventions for embedded e-invoice XML files.
func isEInvoiceAttachment(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".xml") ||
		strings.Contains(lower, "factur-x") ||
		strings.Contains(lower, "zugferd") ||
		strings.Contains(lower, "xrechnung")
}
