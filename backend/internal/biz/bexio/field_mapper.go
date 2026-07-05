package bexio

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ContactSyncData holds mapped contact data ready for CRM service operations.
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

// FieldMapper transforms entities between KMU Hub and Bexio formats.
type FieldMapper struct{}

// NewFieldMapper creates a new field mapper.
func NewFieldMapper() *FieldMapper {
	return &FieldMapper{}
}

// --- Contact Mapping ---

// MapContactToBexio maps a KMU Hub contact to a Bexio contact.
func (fm *FieldMapper) MapContactToBexio(contact *models.Contact, companyName *string, mappings []models.BexioFieldMappingEntry, cache *LookupCache) (*BexioContact, error) {
	bc := &BexioContact{
		ContactTypeID: BexioContactTypePerson,
		CountryID:     1, // Default: Schweiz
	}

	if companyName != nil {
		bc.ContactTypeID = BexioContactTypeCompany
		bc.NameFirst = *companyName
	} else {
		bc.NameFirst = contact.FirstName
		last := contact.LastName
		bc.NameLast = &last
	}

	bc.Mail = contact.Email
	bc.PhoneFixed = contact.Phone
	bc.Remarks = contact.Notes

	// Apply custom field mappings
	for _, m := range mappings {
		if m.Direction == "inbound" {
			continue // Skip inbound-only mappings
		}
		fm.applyContactFieldToBexio(bc, contact, companyName, m, cache)
	}

	return bc, nil
}

func (fm *FieldMapper) applyContactFieldToBexio(bc *BexioContact, contact *models.Contact, companyName *string, m models.BexioFieldMappingEntry, cache *LookupCache) {
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

	switch m.BexioField {
	case "name_1":
		bc.NameFirst = value
	case "name_2":
		bc.NameLast = &value
	case "mail":
		bc.Mail = &value
	case "phone_fixed":
		bc.PhoneFixed = &value
	case "phone_mobile":
		bc.PhoneMobile = &value
	case "remarks":
		bc.Remarks = &value
	case "address":
		bc.Address = &value
	case "city":
		bc.City = &value
	case "postcode":
		bc.Postcode = &value
	case "country_id":
		if cache != nil {
			if id, ok := cache.GetCountryID(value); ok {
				bc.CountryID = id
			}
		}
	}
}

// MapContactToKMUHub maps a Bexio contact to KMU Hub format.
func (fm *FieldMapper) MapContactToKMUHub(bexioContact *BexioContact, mappings []models.BexioFieldMappingEntry, cache *LookupCache) (*ContactSyncData, error) {
	data := &ContactSyncData{
		IsCompany:  bexioContact.IsCompany(),
		FirstName:  bexioContact.NameFirst,
		Email:      bexioContact.Mail,
		Phone:      bexioContact.PhoneFixed,
		Mobile:     bexioContact.PhoneMobile,
		Address:    bexioContact.Address,
		City:       bexioContact.City,
		Zip:        bexioContact.Postcode,
		Notes:      bexioContact.Remarks,
	}

	if bexioContact.NameLast != nil {
		data.LastName = *bexioContact.NameLast
	}

	if bexioContact.Mail != nil {
		data.Email = bexioContact.Mail
	}

	// Apply custom field mappings
	for _, m := range mappings {
		if m.Direction == "outbound" {
			continue // Skip outbound-only mappings
		}
		fm.applyContactFieldToKMUHub(data, bexioContact, m, cache)
	}

	return data, nil
}

func (fm *FieldMapper) applyContactFieldToKMUHub(data *ContactSyncData, bc *BexioContact, m models.BexioFieldMappingEntry, cache *LookupCache) {
	var value string

	switch m.BexioField {
	case "name_1":
		value = bc.NameFirst
	case "name_2":
		if bc.NameLast != nil {
			value = *bc.NameLast
		}
	case "mail":
		if bc.Mail != nil {
			value = *bc.Mail
		}
	case "phone_fixed":
		if bc.PhoneFixed != nil {
			value = *bc.PhoneFixed
		}
	case "phone_mobile":
		if bc.PhoneMobile != nil {
			value = *bc.PhoneMobile
		}
	case "remarks":
		if bc.Remarks != nil {
			value = *bc.Remarks
		}
	case "address":
		if bc.Address != nil {
			value = *bc.Address
		}
	case "city":
		if bc.City != nil {
			value = *bc.City
		}
	case "postcode":
		if bc.Postcode != nil {
			value = *bc.Postcode
		}
	default:
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

// --- Invoice Mapping ---

// MapInvoiceToBexio maps a KMU Hub invoice to a Bexio invoice. cache resolves the
// document currency to the tenant's Bexio currency ID; pass nil to fall back to CHF.
func (fm *FieldMapper) MapInvoiceToBexio(invoice *models.Invoice, contactBexioID int, _ []models.BexioFieldMappingEntry, cache *LookupCache) (*BexioInvoice, error) {
	bi := &BexioInvoice{
		ContactID:   contactBexioID,
		UserID:      1, // Default user in Bexio, resolved by API
		CurrencyID:  resolveBexioCurrencyID(cache, invoice.Currency),
		MwstType:    BexioMwstTypeExclusive,
		MwstIsNet:   true,
		IsValidFrom: invoice.InvoiceDate.Format("2006-01-02"),
	}

	if !invoice.DueDate.IsZero() {
		due := invoice.DueDate.Format("2006-01-02")
		bi.IsValidTo = &due
	}

	title := fmt.Sprintf("Rechnung %s", invoice.InvoiceNumber)
	bi.Title = &title

	// Map line items from JSON
	positions, err := fm.mapLineItemsToPositions(invoice.LineItems)
	if err != nil {
		return nil, fmt.Errorf("map line items: %w", err)
	}
	bi.Positions = positions

	return bi, nil
}

// ImportedInvoiceInput holds a Bexio invoice mapped for the read-only import path in
// the finance service (source='bexio'). It carries only what a Bexio-origin invoice
// provides; the finance service assigns no RE-number and never runs it through
// Send()/NextNumberInTx, so the GoBD gap-free sequence stays untouched.
type ImportedInvoiceInput struct {
	ExternalID      string    // Bexio kb_invoice.id
	ExternalNumber  string    // Bexio document_nr (shown as invoice_number)
	Status          string    // mapped Cosmi status (draft/sent/paid/cancelled)
	ContactID       uuid.UUID // resolved Cosmi contact; uuid.Nil when unresolved
	CustomerName    string    // filled by the puller from the resolved contact
	CustomerEmail   *string   // filled by the puller from the resolved contact
	CustomerAddress *string   // filled by the puller from the resolved contact
	InvoiceDate     time.Time
	DueDate         *time.Time
	Currency        string
	Subtotal        decimal.Decimal
	TotalTax        decimal.Decimal
	GrossTotal      decimal.Decimal
	Lines           []ImportedInvoiceLine
	BexioUpdatedAt  *time.Time // for last-write-wins in bexio_entity_mappings
}

// ImportedInvoiceLine is a single Bexio position mapped to the Cosmi line shape.
type ImportedInvoiceLine struct {
	Position    int
	Description string
	Quantity    decimal.Decimal
	UnitPrice   decimal.Decimal
	TaxRate     decimal.Decimal
	LineTotal   decimal.Decimal
}

// MapInvoiceToKMUHub maps a Bexio invoice to the read-only import shape (reverse of
// MapInvoiceToBexio). Contact resolution (Bexio contact_id -> Cosmi contact UUID)
// happens in the puller via the contact entity-mapping; pass uuid.Nil when unresolved.
func (fm *FieldMapper) MapInvoiceToKMUHub(bi *BexioInvoice, contactID uuid.UUID, _ []models.BexioFieldMappingEntry, cache *LookupCache) (*ImportedInvoiceInput, error) {
	in := &ImportedInvoiceInput{
		ExternalID:     strconv.Itoa(bi.ID),
		Status:         mapBexioInvoiceStatus(bi.KBItemStatusID),
		ContactID:      contactID,
		Currency:       resolveKMUHubCurrency(cache, bi.CurrencyID),
		Subtotal:       parseBexioDecimal(bi.TotalNet),
		TotalTax:       parseBexioDecimal(bi.TotalTaxes),
		GrossTotal:     firstNonZeroDecimal(parseBexioDecimal(bi.TotalGross), parseBexioDecimal(bi.Total)),
		BexioUpdatedAt: parseBexioTime(bi.UpdatedAt),
	}
	if bi.DocumentNr != nil {
		in.ExternalNumber = *bi.DocumentNr
	}
	if t := parseBexioTime(bi.IsValidFrom); t != nil {
		in.InvoiceDate = *t
	}
	if bi.IsValidTo != nil {
		in.DueDate = parseBexioTime(*bi.IsValidTo)
	}
	in.Lines = mapBexioPositionsToLines(bi.Positions)
	return in, nil
}

// mapBexioInvoiceStatus maps a Bexio KBItemStatusID to a Cosmi invoice status.
// Imported invoices are read-only regardless of status (source='bexio' guard); the
// mapping only drives display.
func mapBexioInvoiceStatus(bexioStatusID int) string {
	switch bexioStatusID {
	case BexioStatusPaid:
		return "paid"
	case BexioStatusSent, BexioStatusPartial:
		return "sent"
	case BexioStatusDeclined:
		return "cancelled"
	case BexioStatusDraft, BexioStatusPending:
		return "draft"
	default:
		return "sent"
	}
}

// mapBexioPositionsToLines reverses mapLineItemsToPositions for the import path.
// lean: tax_rate is not resolved — a Bexio position's tax_id is a tenant-specific tax
// record, not a rate; defaults to 0. Add a Bexio tax lookup (like LookupCache) when
// imported invoices must carry real VAT rates for reporting.
func mapBexioPositionsToLines(positions []BexioInvoicePosition) []ImportedInvoiceLine {
	lines := make([]ImportedInvoiceLine, 0, len(positions))
	for i, p := range positions {
		qty := parseBexioDecimalStr(p.Amount)
		unit := parseBexioDecimalStr(p.UnitPrice)
		total := qty.Mul(unit)
		if p.PositionTotal != nil {
			if t, err := decimal.NewFromString(*p.PositionTotal); err == nil {
				total = t
			}
		}
		lines = append(lines, ImportedInvoiceLine{
			Position:    i + 1,
			Description: p.Text,
			Quantity:    qty,
			UnitPrice:   unit,
			TaxRate:     decimal.Zero,
			LineTotal:   total,
		})
	}
	return lines
}

// parseBexioDecimal parses an optional Bexio money string; nil/empty/invalid -> zero.
func parseBexioDecimal(s *string) decimal.Decimal {
	if s == nil {
		return decimal.Zero
	}
	return parseBexioDecimalStr(*s)
}

func parseBexioDecimalStr(s string) decimal.Decimal {
	if s == "" {
		return decimal.Zero
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

func firstNonZeroDecimal(a, b decimal.Decimal) decimal.Decimal {
	if !a.IsZero() {
		return a
	}
	return b
}

// resolveKMUHubCurrency reverse-resolves a Bexio currency ID to a currency code.
// lean: LookupCache only indexes code->ID, so a proper reverse lookup is not yet
// available; falls back to the default currency. Add a reverse index in LookupCache
// when imported Bexio invoices in foreign currencies must show their true currency.
func resolveKMUHubCurrency(_ *LookupCache, _ int) string {
	return models.DefaultCurrency
}

// --- Quote Mapping ---

// MapQuoteToBexio maps a KMU Hub quote to a Bexio quote (KB offer). cache resolves
// the document currency to the tenant's Bexio currency ID; pass nil to fall back to CHF.
func (fm *FieldMapper) MapQuoteToBexio(quote *models.Quote, contactBexioID int, _ []models.BexioFieldMappingEntry, cache *LookupCache) (*BexioQuote, error) {
	bq := &BexioQuote{
		ContactID:   contactBexioID,
		UserID:      1,
		CurrencyID:  resolveBexioCurrencyID(cache, quote.Currency),
		MwstType:    BexioMwstTypeExclusive,
		MwstIsNet:   true,
		IsValidFrom: quote.CreatedAt.Format("2006-01-02"),
	}

	if quote.ValidUntil != nil {
		until := quote.ValidUntil.Format("2006-01-02")
		bq.IsValidTo = &until
	}

	title := fmt.Sprintf("Offerte %s", quote.QuoteNumber)
	bq.Title = &title

	positions, err := fm.mapLineItemsToPositions(quote.LineItems)
	if err != nil {
		return nil, fmt.Errorf("map line items: %w", err)
	}
	bq.Positions = positions

	return bq, nil
}

// resolveBexioCurrencyID maps a document currency code to the tenant's Bexio
// currency ID via the lookup cache. Bexio currency IDs are tenant-specific, so a
// populated cache is required; when the cache is nil/unpopulated or the code is
// unknown it falls back to 1 (CHF), preserving the historical default (B6).
func resolveBexioCurrencyID(cache *LookupCache, code string) int {
	if code == "" {
		code = models.DefaultCurrency
	}
	if cache != nil {
		if id, ok := cache.GetCurrencyID(code); ok {
			return id
		}
	}
	return 1 // CHF fallback
}

// mapLineItemsToPositions converts KMU Hub line items JSON to Bexio positions.
func (fm *FieldMapper) mapLineItemsToPositions(lineItemsJSON json.RawMessage) ([]BexioInvoicePosition, error) {
	if len(lineItemsJSON) == 0 {
		return nil, nil
	}

	var items []struct {
		Description string          `json:"description"`
		Quantity    decimal.Decimal `json:"quantity"`
		UnitPrice   decimal.Decimal `json:"unit_price"`
		TaxRate     decimal.Decimal `json:"tax_rate"`
		Discount    decimal.Decimal `json:"discount"`
	}

	if err := json.Unmarshal(lineItemsJSON, &items); err != nil {
		return nil, fmt.Errorf("unmarshal line items: %w", err)
	}

	positions := make([]BexioInvoicePosition, 0, len(items))
	for _, item := range items {
		positions = append(positions, BexioInvoicePosition{
			Type:          "KbPositionCustom",
			Amount:        item.Quantity.String(),
			UnitPrice:     item.UnitPrice.String(),
			Text:          item.Description,
			DiscountInPct: item.Discount.String(),
		})
	}

	return positions, nil
}

// --- Default Mappings ---

// DefaultContactMappings returns the default field mappings for contacts.
func DefaultContactMappings() []models.BexioFieldMappingEntry {
	return []models.BexioFieldMappingEntry{
		{KmuhubField: "first_name", BexioField: "name_1", Direction: "both", Required: true},
		{KmuhubField: "last_name", BexioField: "name_2", Direction: "both", Required: false},
		{KmuhubField: "email", BexioField: "mail", Direction: "both", Required: false},
		{KmuhubField: "phone", BexioField: "phone_fixed", Direction: "both", Required: false},
		{KmuhubField: "notes", BexioField: "remarks", Direction: "both", Required: false},
	}
}

// DefaultInvoiceMappings returns the default field mappings for invoices.
func DefaultInvoiceMappings() []models.BexioFieldMappingEntry {
	return []models.BexioFieldMappingEntry{
		{KmuhubField: "invoice_number", BexioField: "document_nr", Direction: "outbound", Required: true},
		{KmuhubField: "invoice_date", BexioField: "is_valid_from", Direction: "outbound", Required: true},
		{KmuhubField: "due_date", BexioField: "is_valid_to", Direction: "outbound", Required: false},
	}
}

// DefaultQuoteMappings returns the default field mappings for quotes.
func DefaultQuoteMappings() []models.BexioFieldMappingEntry {
	return []models.BexioFieldMappingEntry{
		{KmuhubField: "quote_number", BexioField: "document_nr", Direction: "outbound", Required: true},
		{KmuhubField: "valid_until", BexioField: "is_valid_to", Direction: "outbound", Required: false},
	}
}

// --- Validation ---

// ValidateFieldMappings validates that field mappings are correct for an entity type.
func ValidateFieldMappings(entityType string, mappings []models.BexioFieldMappingEntry) error {
	validDirections := map[string]bool{"inbound": true, "outbound": true, "both": true}
	bexioTargets := make(map[string]bool)

	for _, m := range mappings {
		if m.KmuhubField == "" || m.BexioField == "" {
			return fmt.Errorf("%w: empty field names not allowed", ErrInvalidFieldMapping)
		}
		if !validDirections[m.Direction] {
			return fmt.Errorf("%w: invalid direction %q", ErrInvalidFieldMapping, m.Direction)
		}
		if bexioTargets[m.BexioField] {
			return fmt.Errorf("%w: duplicate bexio target field %q", ErrInvalidFieldMapping, m.BexioField)
		}
		bexioTargets[m.BexioField] = true
	}

	// Check required defaults are present
	switch entityType {
	case "contact":
		if !bexioTargets["name_1"] {
			return fmt.Errorf("%w: contact mapping must include name_1", ErrInvalidFieldMapping)
		}
	case "invoice":
		if !bexioTargets["document_nr"] {
			return fmt.Errorf("%w: invoice mapping must include document_nr", ErrInvalidFieldMapping)
		}
	}

	return nil
}
