package lexware

import (
	"encoding/json"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

type ContactSyncData struct {
	FirstName string
	LastName  string
	Email     *string
	Phone     *string
	Mobile    *string
	Address   *string
	City      *string
	Zip       *string
	Country   *string
	Notes     *string
	IsCompany bool
}

type FieldMapper struct{}

func NewFieldMapper() *FieldMapper {
	return &FieldMapper{}
}

func (fm *FieldMapper) MapContactToKMUHub(lc *LexwareContact, mappings []models.LexwareFieldMappingEntry) (*ContactSyncData, error) {
	data := &ContactSyncData{
		IsCompany: lc.IsCompany(),
	}

	if lc.Person != nil {
		data.FirstName = lc.Person.FirstName
		data.LastName = lc.Person.LastName
	}
	if lc.Company != nil && lc.IsCompany() {
		data.FirstName = lc.Company.Name
	}

	if lc.EmailAddresses != nil && len(lc.EmailAddresses.Business) > 0 {
		data.Email = &lc.EmailAddresses.Business[0]
	}
	if lc.PhoneNumbers != nil && len(lc.PhoneNumbers.Business) > 0 {
		data.Phone = &lc.PhoneNumbers.Business[0]
	}
	if lc.PhoneNumbers != nil && len(lc.PhoneNumbers.Mobile) > 0 {
		data.Mobile = &lc.PhoneNumbers.Mobile[0]
	}

	if lc.Addresses != nil && lc.Addresses.Billing != nil {
		data.Address = &lc.Addresses.Billing.Street
		data.City = &lc.Addresses.Billing.City
		data.Zip = &lc.Addresses.Billing.Zip
		data.Country = &lc.Addresses.Billing.CountryCode
	}

	if lc.Note != "" {
		data.Notes = &lc.Note
	}

	for _, m := range mappings {
		if m.Direction == "outbound" {
			continue
		}
		fm.applyContactFieldToKMUHub(data, lc, m)
	}

	return data, nil
}

func (fm *FieldMapper) applyContactFieldToKMUHub(data *ContactSyncData, lc *LexwareContact, m models.LexwareFieldMappingEntry) {
	value := fm.getNestedLexwareValue(lc, m.LexwareField)
	if value == "" {
		return
	}

	switch m.KmuhubField {
	case "first_name":
		data.FirstName = value
	case "last_name":
		data.LastName = value
	case "email":
		data.Email = &value
	case "phone":
		data.Phone = &value
	case "notes":
		data.Notes = &value
	}
}

func (fm *FieldMapper) getNestedLexwareValue(lc *LexwareContact, field string) string {
	switch field {
	case "person.firstName":
		if lc.Person != nil {
			return lc.Person.FirstName
		}
	case "person.lastName":
		if lc.Person != nil {
			return lc.Person.LastName
		}
	case "company.name":
		if lc.Company != nil {
			return lc.Company.Name
		}
	case "emailAddresses.business":
		if lc.EmailAddresses != nil && len(lc.EmailAddresses.Business) > 0 {
			return lc.EmailAddresses.Business[0]
		}
	case "phoneNumbers.business":
		if lc.PhoneNumbers != nil && len(lc.PhoneNumbers.Business) > 0 {
			return lc.PhoneNumbers.Business[0]
		}
	case "phoneNumbers.mobile":
		if lc.PhoneNumbers != nil && len(lc.PhoneNumbers.Mobile) > 0 {
			return lc.PhoneNumbers.Mobile[0]
		}
	case "addresses.billing.street":
		if lc.Addresses != nil && lc.Addresses.Billing != nil {
			return lc.Addresses.Billing.Street
		}
	case "addresses.billing.city":
		if lc.Addresses != nil && lc.Addresses.Billing != nil {
			return lc.Addresses.Billing.City
		}
	case "addresses.billing.zip":
		if lc.Addresses != nil && lc.Addresses.Billing != nil {
			return lc.Addresses.Billing.Zip
		}
	case "addresses.billing.countryCode":
		if lc.Addresses != nil && lc.Addresses.Billing != nil {
			return lc.Addresses.Billing.CountryCode
		}
	case "note":
		return lc.Note
	}
	return ""
}

func (fm *FieldMapper) MapContactToLexware(contact *models.Contact, companyName *string, mappings []models.LexwareFieldMappingEntry) (*LexwareContact, error) {
	lc := &LexwareContact{
		Roles: LexwareContactRoles{
			Customer: &LexwareRole{},
		},
	}

	if companyName != nil {
		lc.Company = &LexwareContactCompany{Name: *companyName}
	} else {
		lc.Person = &LexwareContactPerson{
			FirstName: contact.FirstName,
			LastName:  contact.LastName,
		}
	}

	if contact.Email != nil {
		lc.EmailAddresses = &LexwareEmails{
			Business: []string{*contact.Email},
		}
	}

	if contact.Phone != nil {
		lc.PhoneNumbers = &LexwarePhones{
			Business: []string{*contact.Phone},
		}
	}

	if contact.Notes != nil {
		lc.Note = *contact.Notes
	}

	for _, m := range mappings {
		if m.Direction == "inbound" {
			continue
		}
		fm.applyContactFieldToLexware(lc, contact, companyName, m)
	}

	return lc, nil
}

func (fm *FieldMapper) applyContactFieldToLexware(lc *LexwareContact, contact *models.Contact, companyName *string, m models.LexwareFieldMappingEntry) {
	var value string
	switch m.KmuhubField {
	case "first_name":
		value = contact.FirstName
	case "last_name":
		value = contact.LastName
	case "email":
		if contact.Email != nil {
			value = *contact.Email
		}
	case "phone":
		if contact.Phone != nil {
			value = *contact.Phone
		}
	case "notes":
		if contact.Notes != nil {
			value = *contact.Notes
		}
	case "company_name":
		if companyName != nil {
			value = *companyName
		}
	default:
		return
	}

	fm.setNestedLexwareValue(lc, m.LexwareField, value)
}

func (fm *FieldMapper) setNestedLexwareValue(lc *LexwareContact, field, value string) {
	switch field {
	case "person.firstName":
		if lc.Person == nil {
			lc.Person = &LexwareContactPerson{}
		}
		lc.Person.FirstName = value
	case "person.lastName":
		if lc.Person == nil {
			lc.Person = &LexwareContactPerson{}
		}
		lc.Person.LastName = value
	case "company.name":
		if lc.Company == nil {
			lc.Company = &LexwareContactCompany{}
		}
		lc.Company.Name = value
	case "emailAddresses.business":
		if lc.EmailAddresses == nil {
			lc.EmailAddresses = &LexwareEmails{}
		}
		lc.EmailAddresses.Business = []string{value}
	case "phoneNumbers.business":
		if lc.PhoneNumbers == nil {
			lc.PhoneNumbers = &LexwarePhones{}
		}
		lc.PhoneNumbers.Business = []string{value}
	case "addresses.billing.street":
		if lc.Addresses == nil {
			lc.Addresses = &LexwareAddresses{}
		}
		if lc.Addresses.Billing == nil {
			lc.Addresses.Billing = &LexwareAddress{}
		}
		lc.Addresses.Billing.Street = value
	case "addresses.billing.city":
		if lc.Addresses == nil {
			lc.Addresses = &LexwareAddresses{}
		}
		if lc.Addresses.Billing == nil {
			lc.Addresses.Billing = &LexwareAddress{}
		}
		lc.Addresses.Billing.City = value
	case "addresses.billing.zip":
		if lc.Addresses == nil {
			lc.Addresses = &LexwareAddresses{}
		}
		if lc.Addresses.Billing == nil {
			lc.Addresses.Billing = &LexwareAddress{}
		}
		lc.Addresses.Billing.Zip = value
	case "addresses.billing.countryCode":
		if lc.Addresses == nil {
			lc.Addresses = &LexwareAddresses{}
		}
		if lc.Addresses.Billing == nil {
			lc.Addresses.Billing = &LexwareAddress{}
		}
		lc.Addresses.Billing.CountryCode = value
	case "note":
		lc.Note = value
	}
}

// MapInvoiceToLexware maps a KMU Hub invoice to a Lexware invoice.
func (fm *FieldMapper) MapInvoiceToLexware(invoice *models.Invoice, contactLexwareID string, _ []models.LexwareFieldMappingEntry) (*LexwareInvoice, error) {
	li := &LexwareInvoice{
		VoucherDate: invoice.InvoiceDate.Format("2006-01-02"),
		Address: LexwareDocAddress{
			ContactID: contactLexwareID,
			Name:      invoice.CustomerName,
		},
		TaxType: LexwareTaxTypeNet,
	}

	if !invoice.DueDate.IsZero() {
		li.DueDate = invoice.DueDate.Format("2006-01-02")
	}

	positions, err := fm.mapLineItemsToLexware(invoice.LineItems)
	if err != nil {
		return nil, fmt.Errorf("map line items: %w", err)
	}
	li.LineItems = positions

	return li, nil
}

// MapQuoteToLexware maps a KMU Hub quote to a Lexware quotation.
func (fm *FieldMapper) MapQuoteToLexware(quote *models.Quote, contactLexwareID string, _ []models.LexwareFieldMappingEntry) (*LexwareQuotation, error) {
	lq := &LexwareQuotation{
		VoucherDate: quote.CreatedAt.Format("2006-01-02"),
		Address: LexwareDocAddress{
			ContactID: contactLexwareID,
			Name:      quote.CustomerName,
		},
		TaxType: LexwareTaxTypeNet,
	}

	if quote.ValidUntil != nil {
		lq.ExpirationDate = quote.ValidUntil.Format("2006-01-02")
	}

	positions, err := fm.mapLineItemsToLexware(quote.LineItems)
	if err != nil {
		return nil, fmt.Errorf("map line items: %w", err)
	}
	lq.LineItems = positions

	return lq, nil
}

func (fm *FieldMapper) mapLineItemsToLexware(lineItemsJSON json.RawMessage) ([]LexwareLineItem, error) {
	if len(lineItemsJSON) == 0 {
		return nil, nil
	}

	var items []struct {
		Description string          `json:"description"`
		Quantity    decimal.Decimal `json:"quantity"`
		UnitPrice   decimal.Decimal `json:"unit_price"`
		TaxRate     decimal.Decimal `json:"tax_rate"`
	}

	if err := json.Unmarshal(lineItemsJSON, &items); err != nil {
		return nil, fmt.Errorf("unmarshal line items: %w", err)
	}

	lineItems := make([]LexwareLineItem, 0, len(items))
	for _, item := range items {
		netAmount, _ := item.UnitPrice.Float64()
		qty, _ := item.Quantity.Float64()
		taxRate, _ := item.TaxRate.Float64()

		lineItems = append(lineItems, LexwareLineItem{
			Type:     "custom",
			Name:     item.Description,
			Quantity: qty,
			UnitPrice: LexwareUnitPrice{
				Currency:          "EUR",
				NetAmount:         netAmount,
				TaxRatePercentage: taxRate,
			},
		})
	}

	return lineItems, nil
}

// Default mappings

func DefaultContactMappings() []models.LexwareFieldMappingEntry {
	return []models.LexwareFieldMappingEntry{
		{KmuhubField: "first_name", LexwareField: "person.firstName", Direction: "both", Required: true},
		{KmuhubField: "last_name", LexwareField: "person.lastName", Direction: "both", Required: false},
		{KmuhubField: "email", LexwareField: "emailAddresses.business", Direction: "both", Required: false},
		{KmuhubField: "phone", LexwareField: "phoneNumbers.business", Direction: "both", Required: false},
		{KmuhubField: "notes", LexwareField: "note", Direction: "both", Required: false},
	}
}

func DefaultInvoiceMappings() []models.LexwareFieldMappingEntry {
	return []models.LexwareFieldMappingEntry{
		{KmuhubField: "invoice_number", LexwareField: "voucherNumber", Direction: "outbound", Required: true},
		{KmuhubField: "invoice_date", LexwareField: "voucherDate", Direction: "outbound", Required: true},
		{KmuhubField: "due_date", LexwareField: "dueDate", Direction: "outbound", Required: false},
	}
}

func DefaultQuoteMappings() []models.LexwareFieldMappingEntry {
	return []models.LexwareFieldMappingEntry{
		{KmuhubField: "quote_number", LexwareField: "voucherNumber", Direction: "outbound", Required: true},
		{KmuhubField: "valid_until", LexwareField: "expirationDate", Direction: "outbound", Required: false},
	}
}

func ValidateFieldMappings(entityType string, mappings []models.LexwareFieldMappingEntry) error {
	validDirections := map[string]bool{"inbound": true, "outbound": true, "both": true}
	lexwareTargets := make(map[string]bool)

	for _, m := range mappings {
		if m.KmuhubField == "" || m.LexwareField == "" {
			return fmt.Errorf("%w: empty field names not allowed", ErrInvalidFieldMapping)
		}
		if !validDirections[m.Direction] {
			return fmt.Errorf("%w: invalid direction %q", ErrInvalidFieldMapping, m.Direction)
		}
		if lexwareTargets[m.LexwareField] {
			return fmt.Errorf("%w: duplicate lexware target field %q", ErrInvalidFieldMapping, m.LexwareField)
		}
		lexwareTargets[m.LexwareField] = true
	}

	switch entityType {
	case "contact":
		if !lexwareTargets["person.firstName"] && !lexwareTargets["company.name"] {
			return fmt.Errorf("%w: contact mapping must include person.firstName or company.name", ErrInvalidFieldMapping)
		}
	case "invoice":
		if !lexwareTargets["voucherNumber"] {
			return fmt.Errorf("%w: invoice mapping must include voucherNumber", ErrInvalidFieldMapping)
		}
	}

	return nil
}
