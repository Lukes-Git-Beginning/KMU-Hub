package lexware

// LexwareContact represents a contact from the Lexware Office API.
type LexwareContact struct {
	ID             string                 `json:"id,omitempty"`
	Version        int                    `json:"version,omitempty"`
	Roles          LexwareContactRoles    `json:"roles"`
	Person         *LexwareContactPerson  `json:"person,omitempty"`
	Company        *LexwareContactCompany `json:"company,omitempty"`
	Addresses      *LexwareAddresses      `json:"addresses,omitempty"`
	EmailAddresses *LexwareEmails         `json:"emailAddresses,omitempty"`
	PhoneNumbers   *LexwarePhones         `json:"phoneNumbers,omitempty"`
	Note           string                 `json:"note,omitempty"`
	Archived       bool                   `json:"archived,omitempty"`
	UpdatedDate    string                 `json:"updatedDate,omitempty"`
	CreatedDate    string                 `json:"createdDate,omitempty"`
}

type LexwareContactRoles struct {
	Customer *LexwareRole `json:"customer,omitempty"`
	Vendor   *LexwareRole `json:"vendor,omitempty"`
}

type LexwareRole struct{}

type LexwareContactPerson struct {
	Salutation string `json:"salutation,omitempty"`
	FirstName  string `json:"firstName,omitempty"`
	LastName   string `json:"lastName,omitempty"`
}

type LexwareContactCompany struct {
	Name              string                 `json:"name,omitempty"`
	TaxNumber         string                 `json:"taxNumber,omitempty"`
	VatRegistrationID string                 `json:"vatRegistrationId,omitempty"`
	ContactPersons    []LexwareContactPerson `json:"contactPersons,omitempty"`
}

type LexwareAddresses struct {
	Billing  *LexwareAddress `json:"billing,omitempty"`
	Shipping *LexwareAddress `json:"shipping,omitempty"`
}

type LexwareAddress struct {
	Supplement  string `json:"supplement,omitempty"`
	Street      string `json:"street,omitempty"`
	Zip         string `json:"zip,omitempty"`
	City        string `json:"city,omitempty"`
	CountryCode string `json:"countryCode,omitempty"`
}

type LexwareEmails struct {
	Business []string `json:"business,omitempty"`
	Office   []string `json:"office,omitempty"`
	Private  []string `json:"private,omitempty"`
	Other    []string `json:"other,omitempty"`
}

type LexwarePhones struct {
	Business []string `json:"business,omitempty"`
	Office   []string `json:"office,omitempty"`
	Mobile   []string `json:"mobile,omitempty"`
	Private  []string `json:"private,omitempty"`
	Fax      []string `json:"fax,omitempty"`
	Other    []string `json:"other,omitempty"`
}

// IsCompany returns true if the contact has a company role.
func (c *LexwareContact) IsCompany() bool {
	return c.Company != nil && c.Company.Name != ""
}

// LexwareLineItem represents a single line item in an invoice or quotation.
type LexwareLineItem struct {
	ID             string           `json:"id,omitempty"`
	Type           string           `json:"type"`
	Name           string           `json:"name"`
	Description    string           `json:"description,omitempty"`
	Quantity       float64          `json:"quantity"`
	UnitName       string           `json:"unitName,omitempty"`
	UnitPrice      LexwareUnitPrice `json:"unitPrice"`
	LineItemAmount float64          `json:"lineItemAmount,omitempty"`
}

type LexwareUnitPrice struct {
	Currency          string  `json:"currency"`
	NetAmount         float64 `json:"netAmount"`
	GrossAmount       float64 `json:"grossAmount"`
	TaxRatePercentage float64 `json:"taxRatePercentage"`
}

// LexwareInvoice represents an invoice from the Lexware Office API.
type LexwareInvoice struct {
	ID            string             `json:"id,omitempty"`
	Version       int                `json:"version,omitempty"`
	VoucherNumber string             `json:"voucherNumber,omitempty"`
	VoucherDate   string             `json:"voucherDate,omitempty"`
	DueDate       string             `json:"dueDate,omitempty"`
	Address       LexwareDocAddress  `json:"address"`
	LineItems     []LexwareLineItem  `json:"lineItems"`
	TotalPrice    *LexwareTotalPrice `json:"totalPrice,omitempty"`
	TaxType       string             `json:"taxType"`
	VoucherStatus string             `json:"voucherStatus,omitempty"`
	Introduction  string             `json:"introduction,omitempty"`
	Remark        string             `json:"remark,omitempty"`
	UpdatedDate   string             `json:"updatedDate,omitempty"`
	CreatedDate   string             `json:"createdDate,omitempty"`
}

type LexwareDocAddress struct {
	ContactID   string `json:"contactId,omitempty"`
	Name        string `json:"name,omitempty"`
	Street      string `json:"street,omitempty"`
	Zip         string `json:"zip,omitempty"`
	City        string `json:"city,omitempty"`
	CountryCode string `json:"countryCode,omitempty"`
}

type LexwareTotalPrice struct {
	Currency         string  `json:"currency"`
	TotalNetAmount   float64 `json:"totalNetAmount"`
	TotalGrossAmount float64 `json:"totalGrossAmount"`
	TotalTaxAmount   float64 `json:"totalTaxAmount"`
}

// LexwareQuotation represents a quotation from the Lexware Office API.
type LexwareQuotation struct {
	ID             string             `json:"id,omitempty"`
	Version        int                `json:"version,omitempty"`
	VoucherNumber  string             `json:"voucherNumber,omitempty"`
	VoucherDate    string             `json:"voucherDate,omitempty"`
	ExpirationDate string             `json:"expirationDate,omitempty"`
	Address        LexwareDocAddress  `json:"address"`
	LineItems      []LexwareLineItem  `json:"lineItems"`
	TotalPrice     *LexwareTotalPrice `json:"totalPrice,omitempty"`
	TaxType        string             `json:"taxType"`
	VoucherStatus  string             `json:"voucherStatus,omitempty"`
	Introduction   string             `json:"introduction,omitempty"`
	Remark         string             `json:"remark,omitempty"`
	UpdatedDate    string             `json:"updatedDate,omitempty"`
	CreatedDate    string             `json:"createdDate,omitempty"`
}

// LexwareListResponse wraps paginated list responses from the Lexware API.
type LexwareListResponse[T any] struct {
	Content          []T  `json:"content"`
	First            bool `json:"first"`
	Last             bool `json:"last"`
	TotalPages       int  `json:"totalPages"`
	TotalElements    int  `json:"totalElements"`
	NumberOfElements int  `json:"numberOfElements"`
	Size             int  `json:"size"`
	Number           int  `json:"number"`
}

// LexwareErrorResponse represents an error response from the Lexware API.
type LexwareErrorResponse struct {
	Timestamp string `json:"timestamp"`
	Status    int    `json:"status"`
	Error     string `json:"error"`
	Message   string `json:"message"`
	Path      string `json:"path"`
}

// Lexware tax type constants.
const (
	LexwareTaxTypeNet                      = "net"
	LexwareTaxTypeGross                    = "gross"
	LexwareTaxTypeIntraCommunitySupply     = "intraCommunitySupply"
	LexwareTaxTypeExternalService          = "externalService"
	LexwareTaxTypeThirdPartyCountryService  = "thirdPartyCountryService"
	LexwareTaxTypeThirdPartyCountryDelivery = "thirdPartyCountryDelivery"
)

// Lexware voucher status constants.
const (
	LexwareVoucherStatusDraft    = "draft"
	LexwareVoucherStatusOpen     = "open"
	LexwareVoucherStatusPaid     = "paid"
	LexwareVoucherStatusVoided   = "voided"
	LexwareVoucherStatusAccepted = "accepted"
	LexwareVoucherStatusRejected = "rejected"
)

// LexwareWebhookEvent represents an incoming webhook event.
type LexwareWebhookEvent struct {
	EventType  string `json:"eventType"`
	ResourceID string `json:"resourceId"`
	OrgID      string `json:"organizationId"`
}
