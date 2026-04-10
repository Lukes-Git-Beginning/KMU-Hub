// Hand-written proto message types for the CreateInvoiceFromTimeEntries RPC.
// These are intentionally simplified (no protoimpl reflection machinery) matching
// the established pattern in this package (see datev_upload.pb.go, lexware.pb.go).
// Run `make proto-biz` after updating biz.proto to regenerate properly.

package bizv1

// CreateInvoiceFromTimeEntriesRequest carries all parameters needed to aggregate
// HR work-time entries and create a draft invoice from them.
type CreateInvoiceFromTimeEntriesRequest struct {
	TenantId        string `protobuf:"bytes,1,opt,name=tenant_id,json=tenantId,proto3" json:"tenant_id,omitempty"`
	EmployeeId      string `protobuf:"bytes,2,opt,name=employee_id,json=employeeId,proto3" json:"employee_id,omitempty"`
	CustomerName    string `protobuf:"bytes,3,opt,name=customer_name,json=customerName,proto3" json:"customer_name,omitempty"`
	CustomerAddress string `protobuf:"bytes,4,opt,name=customer_address,json=customerAddress,proto3" json:"customer_address,omitempty"`
	CustomerEmail   string `protobuf:"bytes,5,opt,name=customer_email,json=customerEmail,proto3" json:"customer_email,omitempty"`
	DateFrom        string `protobuf:"bytes,6,opt,name=date_from,json=dateFrom,proto3" json:"date_from,omitempty"`   // YYYY-MM-DD
	DateTo          string `protobuf:"bytes,7,opt,name=date_to,json=dateTo,proto3" json:"date_to,omitempty"`         // YYYY-MM-DD
	HourlyRate      string `protobuf:"bytes,8,opt,name=hourly_rate,json=hourlyRate,proto3" json:"hourly_rate,omitempty"` // Decimal as string
	Description     string `protobuf:"bytes,9,opt,name=description,proto3" json:"description,omitempty"`
	TaxMode         string `protobuf:"bytes,10,opt,name=tax_mode,json=taxMode,proto3" json:"tax_mode,omitempty"`
	CreatedBy       string `protobuf:"bytes,11,opt,name=created_by,json=createdBy,proto3" json:"created_by,omitempty"`
}

func (x *CreateInvoiceFromTimeEntriesRequest) GetTenantId() string {
	if x != nil {
		return x.TenantId
	}
	return ""
}

func (x *CreateInvoiceFromTimeEntriesRequest) GetEmployeeId() string {
	if x != nil {
		return x.EmployeeId
	}
	return ""
}

func (x *CreateInvoiceFromTimeEntriesRequest) GetCustomerName() string {
	if x != nil {
		return x.CustomerName
	}
	return ""
}

func (x *CreateInvoiceFromTimeEntriesRequest) GetCustomerAddress() string {
	if x != nil {
		return x.CustomerAddress
	}
	return ""
}

func (x *CreateInvoiceFromTimeEntriesRequest) GetCustomerEmail() string {
	if x != nil {
		return x.CustomerEmail
	}
	return ""
}

func (x *CreateInvoiceFromTimeEntriesRequest) GetDateFrom() string {
	if x != nil {
		return x.DateFrom
	}
	return ""
}

func (x *CreateInvoiceFromTimeEntriesRequest) GetDateTo() string {
	if x != nil {
		return x.DateTo
	}
	return ""
}

func (x *CreateInvoiceFromTimeEntriesRequest) GetHourlyRate() string {
	if x != nil {
		return x.HourlyRate
	}
	return ""
}

func (x *CreateInvoiceFromTimeEntriesRequest) GetDescription() string {
	if x != nil {
		return x.Description
	}
	return ""
}

func (x *CreateInvoiceFromTimeEntriesRequest) GetTaxMode() string {
	if x != nil {
		return x.TaxMode
	}
	return ""
}

func (x *CreateInvoiceFromTimeEntriesRequest) GetCreatedBy() string {
	if x != nil {
		return x.CreatedBy
	}
	return ""
}

// CreateInvoiceFromTimeEntriesResponse carries the newly created draft invoice.
// The invoice is serialized as JSON bytes to avoid a circular dependency on the
// Invoice proto message (which requires the full biz.pb.go descriptor machinery).
// The gateway proxy unmarshals these bytes directly into the HTTP response.
type CreateInvoiceFromTimeEntriesResponse struct {
	// InvoiceJson contains the JSON-encoded models.Invoice returned by the biz service.
	InvoiceJson []byte `protobuf:"bytes,1,opt,name=invoice_json,json=invoiceJson,proto3" json:"invoice_json,omitempty"`
}

func (x *CreateInvoiceFromTimeEntriesResponse) GetInvoiceJson() []byte {
	if x != nil {
		return x.InvoiceJson
	}
	return nil
}
