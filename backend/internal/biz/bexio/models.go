package bexio

// BexioContact represents a contact from the Bexio API.
// contact_type_id: 1 = company, 2 = person
type BexioContact struct {
	ID            int     `json:"id,omitempty"`
	ContactTypeID int     `json:"contact_type_id"`
	NameFirst     string  `json:"name_1"`
	NameLast      *string `json:"name_2,omitempty"`
	SalutationID  int     `json:"salutation_id"` // 0 = none (treat as null)
	TitleID       *int    `json:"title_id,omitempty"`
	Birthday      *string `json:"birthday,omitempty"`
	Address       *string `json:"address,omitempty"`
	Postcode      *string `json:"postcode,omitempty"`
	City          *string `json:"city,omitempty"`
	CountryID     int     `json:"country_id"`
	Mail          *string `json:"mail,omitempty"`
	MailSecond    *string `json:"mail_second,omitempty"`
	PhoneFixed    *string `json:"phone_fixed,omitempty"`
	PhoneMobile   *string `json:"phone_mobile,omitempty"`
	Fax           *string `json:"fax,omitempty"`
	URL           *string `json:"url,omitempty"`
	SkypeID       *string `json:"skype_name,omitempty"`
	Remarks       *string `json:"remarks,omitempty"`
	LanguageID    *int    `json:"language_id,omitempty"`
	IsLead        bool    `json:"is_lead"`
	OwnerID       *int    `json:"owner_id,omitempty"`
	UpdatedAt     string  `json:"updated_at,omitempty"`
}

// IsCompany returns true if the contact is a company (contact_type_id 1).
func (c *BexioContact) IsCompany() bool {
	return c.ContactTypeID == 1
}

// BexioInvoicePosition represents a single line item in a Bexio invoice.
type BexioInvoicePosition struct {
	Type            string  `json:"type"` // KbPositionCustom, KbPositionArticle, etc.
	Amount          string  `json:"amount"`
	UnitID          *int    `json:"unit_id,omitempty"`
	AccountID       *int    `json:"account_id,omitempty"`
	TaxID           *int    `json:"tax_id,omitempty"`
	Text            string  `json:"text"`
	UnitPrice       string  `json:"unit_price"`
	DiscountInPct   string  `json:"discount_in_percent"`
	PositionTotal   *string `json:"position_total,omitempty"`
	InternalPos     *int    `json:"internal_pos,omitempty"`
	ArticleID       *int    `json:"article_id,omitempty"`
}

// BexioInvoice represents an invoice from the Bexio API (kb_invoice).
type BexioInvoice struct {
	ID              int                    `json:"id,omitempty"`
	DocumentNr      *string                `json:"document_nr,omitempty"`
	Title           *string                `json:"title,omitempty"`
	ContactID       int                    `json:"contact_id"`
	ContactSubID    *int                   `json:"contact_sub_id,omitempty"`
	UserID          int                    `json:"user_id"`
	LogopayerID     *int                   `json:"logopaper_id,omitempty"`
	LanguageID      *int                   `json:"language_id,omitempty"`
	BankAccountID   *int                   `json:"bank_account_id,omitempty"`
	CurrencyID      int                    `json:"currency_id"`
	PaymentTypeID   *int                   `json:"payment_type_id,omitempty"`
	Header          *string                `json:"header,omitempty"`
	Footer          *string                `json:"footer,omitempty"`
	MwstType        int                    `json:"mwst_type"` // 0=incl, 1=excl, 2=exempt
	MwstIsNet       bool                   `json:"mwst_is_net"`
	IsValidFrom     string                 `json:"is_valid_from"`
	IsValidTo       *string                `json:"is_valid_to,omitempty"`
	TotalGross      *string                `json:"total_gross,omitempty"`
	TotalNet        *string                `json:"total_net,omitempty"`
	TotalTaxes      *string                `json:"total_taxes,omitempty"`
	Total           *string                `json:"total,omitempty"`
	TotalRoundingDiff *float64             `json:"total_rounding_difference,omitempty"`
	KBItemStatusID  int                    `json:"kb_item_status_id"`
	Positions       []BexioInvoicePosition `json:"positions,omitempty"`
	UpdatedAt       string                 `json:"updated_at,omitempty"`
}

// BexioQuote represents a quote (KB offer) from the Bexio API.
type BexioQuote struct {
	ID              int                    `json:"id,omitempty"`
	DocumentNr      *string                `json:"document_nr,omitempty"`
	Title           *string                `json:"title,omitempty"`
	ContactID       int                    `json:"contact_id"`
	ContactSubID    *int                   `json:"contact_sub_id,omitempty"`
	UserID          int                    `json:"user_id"`
	LogopayerID     *int                   `json:"logopaper_id,omitempty"`
	LanguageID      *int                   `json:"language_id,omitempty"`
	BankAccountID   *int                   `json:"bank_account_id,omitempty"`
	CurrencyID      int                    `json:"currency_id"`
	Header          *string                `json:"header,omitempty"`
	Footer          *string                `json:"footer,omitempty"`
	MwstType        int                    `json:"mwst_type"`
	MwstIsNet       bool                   `json:"mwst_is_net"`
	IsValidFrom     string                 `json:"is_valid_from"`
	IsValidTo       *string                `json:"is_valid_to,omitempty"`
	TotalGross      *string                `json:"total_gross,omitempty"`
	TotalNet        *string                `json:"total_net,omitempty"`
	TotalTaxes      *string                `json:"total_taxes,omitempty"`
	Total           *string                `json:"total,omitempty"`
	KBItemStatusID  int                    `json:"kb_item_status_id"`
	Positions       []BexioInvoicePosition `json:"positions,omitempty"`
	UpdatedAt       string                 `json:"updated_at,omitempty"`
}

// BexioPaymentStatus represents payment information for an invoice.
type BexioPaymentStatus struct {
	ID        int     `json:"id"`
	Date      string  `json:"date"`
	Value     string  `json:"value"`
	BankAccountID *int `json:"bank_account_id,omitempty"`
	Title     *string `json:"title,omitempty"`
	IsBankAccountNew bool `json:"is_bank_account_new"`
}

// BexioLookupEntry represents a cached lookup value (country, currency, salutation).
type BexioLookupEntry struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Bexio contact type constants.
const (
	BexioContactTypeCompany = 1
	BexioContactTypePerson  = 2
)

// Bexio MwSt type constants.
const (
	BexioMwstTypeInclusive = 0
	BexioMwstTypeExclusive = 1
	BexioMwstTypeExempt    = 2
)

// Bexio KB item status constants.
const (
	BexioStatusDraft    = 1
	BexioStatusPending  = 2
	BexioStatusSent     = 6
	BexioStatusDeclined = 19
	BexioStatusAccepted = 32
	BexioStatusPaid     = 9
	BexioStatusPartial  = 10
)

// BexioListResponse wraps a paginated list response from the Bexio API.
type BexioListResponse[T any] struct {
	Items  []T `json:"items"`
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

// BexioErrorResponse represents an error response from the Bexio API.
type BexioErrorResponse struct {
	ErrorCode int    `json:"error_code"`
	Message   string `json:"message"`
}

// BexioTokenResponse represents an OAuth2 token response.
type BexioTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}
