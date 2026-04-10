// Hand-written proto message types for the GenerateZUGFeRDInvoicePDF RPC.
// These follow the established pattern in this package (see biz_time.pb.go).
// Run `make proto-biz` after updating biz.proto to regenerate properly.

package bizv1

// GenerateZUGFeRDInvoicePDFRequest carries the parameters needed to generate
// a Factur-X/ZUGFeRD 2.1 compliant PDF for the specified invoice.
type GenerateZUGFeRDInvoicePDFRequest struct {
	Id       string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	TenantId string `protobuf:"bytes,2,opt,name=tenant_id,json=tenantId,proto3" json:"tenant_id,omitempty"`
}

func (x *GenerateZUGFeRDInvoicePDFRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *GenerateZUGFeRDInvoicePDFRequest) GetTenantId() string {
	if x != nil {
		return x.TenantId
	}
	return ""
}

// GenerateZUGFeRDInvoicePDFResponse carries the generated PDF bytes and filename.
// If ZUGFeRD XML embedding fails, the service returns the plain invoice PDF
// (graceful degradation) — the caller cannot distinguish the two cases from
// the response shape alone.
type GenerateZUGFeRDInvoicePDFResponse struct {
	PdfData  []byte `protobuf:"bytes,1,opt,name=pdf_data,json=pdfData,proto3" json:"pdf_data,omitempty"`
	Filename string `protobuf:"bytes,2,opt,name=filename,proto3" json:"filename,omitempty"`
}

func (x *GenerateZUGFeRDInvoicePDFResponse) GetPdfData() []byte {
	if x != nil {
		return x.PdfData
	}
	return nil
}

func (x *GenerateZUGFeRDInvoicePDFResponse) GetFilename() string {
	if x != nil {
		return x.Filename
	}
	return ""
}
