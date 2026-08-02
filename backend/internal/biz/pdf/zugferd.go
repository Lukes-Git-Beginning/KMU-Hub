package pdf

import (
	"bytes"
	"fmt"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"

	"github.com/kmuhub/kmuhub/internal/biz/einvoice"
	"github.com/kmuhub/kmuhub/internal/models"
)

// ZUGFeRD profiles supported (Factur-X / EN 16931)
const (
	ZUGFeRDProfileMinimum = "MINIMUM"
	ZUGFeRDProfileBasicWL = "BASIC_WL"
	ZUGFeRDProfileEN16931 = "EN16931"
)

// FacturXAttachmentName is the file name every Factur-X / ZUGFeRD 2.x reader
// looks for. The specification fixes it, and receiving software matches on it
// literally: an attachment called factur-x_RE-2026-0001_20260801.xml travels
// inside the PDF but is invisible to the recipient's e-invoice pipeline, so the
// invoice arrives as if it carried no structured data at all.
const FacturXAttachmentName = "factur-x.xml"

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
	// a payment due date and a fully populated issuer block. Both are missing
	// master data rather than a defect, so they wrap ErrGenerateFailed and reach
	// the user as a fixable message instead of a 500.
	if invoice.DueDate.IsZero() {
		return nil, fmt.Errorf("%w: invoice %s is missing the due date (BT-9)", einvoice.ErrGenerateFailed, invoice.InvoiceNumber)
	}
	if err := ValidateCompanySettingsForPDF(settings); err != nil {
		return nil, fmt.Errorf("%w: %s", einvoice.ErrGenerateFailed, err.Error())
	}

	// BT-10 (Leitweg-ID) is not stored on the invoice and only applies to
	// public-sector buyers, who receive XRechnung rather than ZUGFeRD-in-PDF.
	xmlBytes, err := einvoice.GenerateCII(invoice, settings, "")
	if err != nil {
		return nil, fmt.Errorf("zugferd: %w", err)
	}
	return xmlBytes, nil
}

// EmbedZUGFeRDXML embeds the Factur-X XML into an existing PDF and declares it
// as the machine-readable alternative of the printed invoice.
//
// Attaching the file is not enough: a reader locates the invoice data through
// the catalog's /AF array and the /AFRelationship of the file specification. An
// attachment without those entries is an ordinary enclosure that no e-invoice
// pipeline reads. That declaration, the fixed file name and the text/xml media
// type are what this function adds on top of pdfcpu's attachment support.
//
// Every failure is returned. The earlier version logged and handed back the
// original PDF, which left the caller unable to tell an e-invoice from a plain
// one — the recipient found out weeks later, when their accounting software
// found no data to import.
//
// lean: the result is a regular PDF 1.7 file, not PDF/A-3b. Conformance would
// need embedded fonts (maroto draws with the non-embedded standard 14),
// an sRGB output intent and XMP metadata carrying the pdfaid and Factur-X
// extension schemas — none of which maroto/v2 or pdfcpu v0.6 can write, so it
// would cost a new dependency plus an ICC profile asset. Claiming conformance
// in XMP without it would be worse than not claiming it: a validating receiver
// rejects a file that lies about its own standard, while it accepts this one.
// Upgrade when a customer's receiving software demands PDF/A-3b (public-sector
// portals do; ordinary B2B receivers read the attachment as it is).
func EmbedZUGFeRDXML(pdfBytes []byte, xmlBytes []byte, invoiceNumber string) ([]byte, error) {
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed

	ctx, _, _, _, err := api.ReadValidateAndOptimize(bytes.NewReader(pdfBytes), conf, time.Now())
	if err != nil {
		return nil, fmt.Errorf("zugferd embed: read invoice PDF %s: %w", invoiceNumber, err)
	}

	modTime := time.Now()
	attachment := model.Attachment{
		Reader:  bytes.NewReader(xmlBytes),
		ID:      FacturXAttachmentName,
		Desc:    fmt.Sprintf("Factur-X/ZUGFeRD invoice %s", invoiceNumber),
		ModTime: &modTime,
	}
	if err := ctx.AddAttachment(attachment, false); err != nil {
		return nil, fmt.Errorf("zugferd embed: attach %s to invoice PDF %s: %w", FacturXAttachmentName, invoiceNumber, err)
	}

	if err := declareFacturXAssociatedFile(ctx); err != nil {
		return nil, fmt.Errorf("zugferd embed: declare %s on invoice PDF %s: %w", FacturXAttachmentName, invoiceNumber, err)
	}

	var out bytes.Buffer
	if err := api.WriteContext(ctx, &out); err != nil {
		return nil, fmt.Errorf("zugferd embed: write invoice PDF %s: %w", invoiceNumber, err)
	}
	if out.Len() == 0 {
		return nil, fmt.Errorf("zugferd embed: writing invoice PDF %s produced no output", invoiceNumber)
	}
	return out.Bytes(), nil
}

// declareFacturXAssociatedFile marks the embedded XML as the machine-readable
// alternative of the document (ISO 32000-2 / PDF/A-3 associated files):
// /AFRelationship /Alternative on the file specification, the file specification
// in the catalog's /AF array, and text/xml as the media type of the stream.
func declareFacturXAssociatedFile(ctx *model.Context) error {
	_, fileSpecRef, err := ctx.SearchEmbeddedFilesNameTreeNodeByContent(FacturXAttachmentName)
	if err != nil {
		return err
	}
	if fileSpecRef == nil {
		return fmt.Errorf("attachment %s not found after adding it", FacturXAttachmentName)
	}

	fileSpec, err := ctx.DereferenceDict(fileSpecRef)
	if err != nil {
		return err
	}
	if fileSpec == nil {
		return fmt.Errorf("file specification for %s is not a dictionary", FacturXAttachmentName)
	}
	fileSpec["AFRelationship"] = types.Name("Alternative")

	// /text#2Fxml is text/xml written as a PDF name token — the solidus has to be
	// escaped, and pdfcpu passes name bytes through unchanged.
	if efDict := fileSpec.DictEntry("EF"); efDict != nil {
		sd, _, err := ctx.DereferenceStreamDict(efDict["F"])
		if err != nil {
			return err
		}
		if sd != nil {
			sd.Dict["Subtype"] = types.Name("text#2Fxml")
		}
	}

	root, err := ctx.Catalog()
	if err != nil {
		return err
	}
	// Append rather than replace: a PDF handed in by a caller may already
	// associate files of its own, and dropping them would lose data.
	var associated types.Array
	if existing, found := root.Find("AF"); found {
		if arr, err := ctx.DereferenceArray(existing); err == nil {
			associated = arr
		}
	}
	root["AF"] = append(associated, fileSpecRef)
	return nil
}
