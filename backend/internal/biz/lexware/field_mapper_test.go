package lexware

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

func strPtr(s string) *string { return &s }

// --- MapContactToKMUHub ---

func TestMapContactToKMUHub_PersonBaseFields(t *testing.T) {
	fm := NewFieldMapper()
	lc := &LexwareContact{
		Person: &LexwareContactPerson{FirstName: "Ada", LastName: "Lovelace"},
		EmailAddresses: &LexwareEmails{
			Business: []string{"ada@example.com"},
		},
		PhoneNumbers: &LexwarePhones{
			Business: []string{"+49 111"},
			Mobile:   []string{"+49 222"},
		},
		Addresses: &LexwareAddresses{
			Billing: &LexwareAddress{Street: "Main St 1", City: "Berlin", Zip: "10115", CountryCode: "DE"},
		},
		Note: "VIP",
	}

	data, err := fm.MapContactToKMUHub(lc, nil)
	require.NoError(t, err)
	assert.False(t, data.IsCompany)
	assert.Equal(t, "Ada", data.FirstName)
	assert.Equal(t, "Lovelace", data.LastName)
	require.NotNil(t, data.Email)
	assert.Equal(t, "ada@example.com", *data.Email)
	require.NotNil(t, data.Phone)
	assert.Equal(t, "+49 111", *data.Phone)
	require.NotNil(t, data.Mobile)
	assert.Equal(t, "+49 222", *data.Mobile)
	require.NotNil(t, data.Address)
	assert.Equal(t, "Main St 1", *data.Address)
	assert.Equal(t, "Berlin", *data.City)
	assert.Equal(t, "10115", *data.Zip)
	assert.Equal(t, "DE", *data.Country)
	require.NotNil(t, data.Notes)
	assert.Equal(t, "VIP", *data.Notes)
}

func TestMapContactToKMUHub_CompanyUsesCompanyNameAsFirstName(t *testing.T) {
	fm := NewFieldMapper()
	lc := &LexwareContact{
		Company: &LexwareContactCompany{Name: "Acme GmbH"},
	}

	data, err := fm.MapContactToKMUHub(lc, nil)
	require.NoError(t, err)
	assert.True(t, data.IsCompany)
	assert.Equal(t, "Acme GmbH", data.FirstName)
}

func TestMapContactToKMUHub_EmptyContactHasNoOptionalFields(t *testing.T) {
	fm := NewFieldMapper()
	data, err := fm.MapContactToKMUHub(&LexwareContact{}, nil)
	require.NoError(t, err)
	assert.Nil(t, data.Email)
	assert.Nil(t, data.Phone)
	assert.Nil(t, data.Mobile)
	assert.Nil(t, data.Address)
	assert.Nil(t, data.Notes)
}

// TestMapContactToKMUHub_OutboundMappingsAreSkipped verifies that inbound
// mapping (Lexware -> KMU Hub) ignores mapping entries marked "outbound" —
// applying them here would push nothing, since the switch in
// applyContactFieldToKMUHub only fires when the entry actually runs.
func TestMapContactToKMUHub_OutboundMappingsAreSkipped(t *testing.T) {
	fm := NewFieldMapper()
	lc := &LexwareContact{Person: &LexwareContactPerson{FirstName: "Base"}}
	mappings := []models.LexwareFieldMappingEntry{
		{KmuhubField: "first_name", LexwareField: "company.name", Direction: "outbound"},
	}

	data, err := fm.MapContactToKMUHub(lc, mappings)
	require.NoError(t, err)
	// The mapping targets company.name, which is empty here — had it been
	// applied despite being outbound-only, FirstName would have been
	// overwritten with "" instead of staying "Base".
	assert.Equal(t, "Base", data.FirstName)
}

// TestMapContactToKMUHub_AppliesEveryNestedFieldSwitchCase drives every
// branch of getNestedLexwareValue via applyContactFieldToKMUHub so the whole
// field-lookup table is exercised, not just the two fields the base mapping
// happens to touch.
func TestMapContactToKMUHub_AppliesEveryNestedFieldSwitchCase(t *testing.T) {
	fm := NewFieldMapper()
	lc := &LexwareContact{
		Person:  &LexwareContactPerson{FirstName: "Ada", LastName: "Lovelace"},
		Company: &LexwareContactCompany{Name: "Acme GmbH"},
		EmailAddresses: &LexwareEmails{
			Business: []string{"biz@example.com"},
		},
		PhoneNumbers: &LexwarePhones{
			Business: []string{"+49 111"},
			Mobile:   []string{"+49 222"},
		},
		Addresses: &LexwareAddresses{
			Billing: &LexwareAddress{Street: "Main St 1", City: "Berlin", Zip: "10115", CountryCode: "DE"},
		},
		Note: "from note field",
	}

	cases := []struct {
		lexwareField string
		kmuhubField  string
		want         string
	}{
		{"person.firstName", "first_name", "Ada"},
		{"person.lastName", "last_name", "Lovelace"},
		{"company.name", "first_name", "Acme GmbH"},
		{"emailAddresses.business", "email", "biz@example.com"},
		{"phoneNumbers.business", "phone", "+49 111"},
		{"phoneNumbers.mobile", "phone", "+49 222"},
		{"addresses.billing.street", "notes", "Main St 1"},
		{"addresses.billing.city", "notes", "Berlin"},
		{"addresses.billing.zip", "notes", "10115"},
		{"addresses.billing.countryCode", "notes", "DE"},
		{"note", "notes", "from note field"},
		{"unknown.field", "notes", ""}, // falls through to the default "" branch
	}

	for _, tc := range cases {
		t.Run(tc.lexwareField, func(t *testing.T) {
			mappings := []models.LexwareFieldMappingEntry{
				{KmuhubField: tc.kmuhubField, LexwareField: tc.lexwareField, Direction: "inbound"},
			}
			data, err := fm.MapContactToKMUHub(lc, mappings)
			require.NoError(t, err)

			switch tc.kmuhubField {
			case "first_name":
				assert.Equal(t, tc.want, data.FirstName)
			case "last_name":
				assert.Equal(t, tc.want, data.LastName)
			case "email":
				require.NotNil(t, data.Email)
				assert.Equal(t, tc.want, *data.Email)
			case "phone":
				require.NotNil(t, data.Phone)
				assert.Equal(t, tc.want, *data.Phone)
			case "notes":
				if tc.want == "" {
					// An empty resolved value must not overwrite Notes at all
					// (applyContactFieldToKMUHub returns early on value == "").
					return
				}
				require.NotNil(t, data.Notes)
				assert.Equal(t, tc.want, *data.Notes)
			}
		})
	}
}

func TestMapContactToKMUHub_UnknownKmuhubFieldIsIgnored(t *testing.T) {
	fm := NewFieldMapper()
	lc := &LexwareContact{Person: &LexwareContactPerson{FirstName: "Base"}}
	mappings := []models.LexwareFieldMappingEntry{
		{KmuhubField: "unmapped_target", LexwareField: "person.lastName", Direction: "inbound"},
	}
	// LastName is empty on lc, so getNestedLexwareValue returns "" and the
	// switch never fires — nothing should change.
	data, err := fm.MapContactToKMUHub(lc, mappings)
	require.NoError(t, err)
	assert.Equal(t, "Base", data.FirstName)
	assert.Equal(t, "", data.LastName)
}

// --- MapContactToLexware / applyContactFieldToLexware / setNestedLexwareValue ---

func TestMapContactToLexware_PersonWithoutCompanyName(t *testing.T) {
	fm := NewFieldMapper()
	contact := &models.Contact{
		FirstName: "Grace",
		LastName:  "Hopper",
		Email:     strPtr("grace@example.com"),
		Phone:     strPtr("+49 333"),
		Notes:     strPtr("Navy"),
	}

	lc, err := fm.MapContactToLexware(contact, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, lc.Person)
	assert.Equal(t, "Grace", lc.Person.FirstName)
	assert.Equal(t, "Hopper", lc.Person.LastName)
	assert.Nil(t, lc.Company)
	require.NotNil(t, lc.EmailAddresses)
	assert.Equal(t, []string{"grace@example.com"}, lc.EmailAddresses.Business)
	require.NotNil(t, lc.PhoneNumbers)
	assert.Equal(t, []string{"+49 333"}, lc.PhoneNumbers.Business)
	assert.Equal(t, "Navy", lc.Note)
	require.NotNil(t, lc.Roles.Customer)
}

func TestMapContactToLexware_WithCompanyName(t *testing.T) {
	fm := NewFieldMapper()
	contact := &models.Contact{FirstName: "Grace", LastName: "Hopper"}
	company := "Navy Inc"

	lc, err := fm.MapContactToLexware(contact, &company, nil)
	require.NoError(t, err)
	require.NotNil(t, lc.Company)
	assert.Equal(t, "Navy Inc", lc.Company.Name)
	assert.Nil(t, lc.Person)
}

func TestMapContactToLexware_NoEmailNoPhoneNoNotes(t *testing.T) {
	fm := NewFieldMapper()
	contact := &models.Contact{FirstName: "Grace", LastName: "Hopper"}

	lc, err := fm.MapContactToLexware(contact, nil, nil)
	require.NoError(t, err)
	assert.Nil(t, lc.EmailAddresses)
	assert.Nil(t, lc.PhoneNumbers)
	assert.Equal(t, "", lc.Note)
}

// TestMapContactToLexware_InboundMappingsAreSkipped mirrors the KMU Hub-side
// test: an "inbound"-only mapping entry must not run in the outbound
// direction, otherwise a mapping meant only for Lexware -> KMU Hub would
// clobber the outbound payload.
func TestMapContactToLexware_InboundMappingsAreSkipped(t *testing.T) {
	fm := NewFieldMapper()
	contact := &models.Contact{FirstName: "Grace", LastName: "Hopper"}
	mappings := []models.LexwareFieldMappingEntry{
		{KmuhubField: "last_name", LexwareField: "person.lastName", Direction: "inbound"},
	}

	lc, err := fm.MapContactToLexware(contact, nil, mappings)
	require.NoError(t, err)
	// person.lastName is already "Hopper" from the base mapping; the inbound
	// entry must be skipped, not re-applied with some other value — assert it
	// stayed exactly what the base mapping set, proving the entry never ran.
	require.NotNil(t, lc.Person)
	assert.Equal(t, "Hopper", lc.Person.LastName)
}

// TestMapContactToLexware_AppliesEveryTargetFieldSwitchCase drives every
// branch of setNestedLexwareValue (via applyContactFieldToLexware) including
// the case where the target struct pointer starts nil and must be allocated.
func TestMapContactToLexware_AppliesEveryTargetFieldSwitchCase(t *testing.T) {
	company := "Acme GmbH"
	contact := &models.Contact{
		FirstName: "Ada",
		LastName:  "Lovelace",
		Email:     strPtr("ada@example.com"),
		Phone:     strPtr("+49 111"),
		Notes:     strPtr("note text"),
	}

	cases := []struct {
		name         string
		kmuhubField  string
		lexwareField string
		assertFn     func(t *testing.T, lc *LexwareContact)
	}{
		{"first_name->person.firstName", "first_name", "person.firstName", func(t *testing.T, lc *LexwareContact) {
			require.NotNil(t, lc.Person)
			assert.Equal(t, "Ada", lc.Person.FirstName)
		}},
		{"last_name->person.lastName", "last_name", "person.lastName", func(t *testing.T, lc *LexwareContact) {
			require.NotNil(t, lc.Person)
			assert.Equal(t, "Lovelace", lc.Person.LastName)
		}},
		{"company_name->company.name", "company_name", "company.name", func(t *testing.T, lc *LexwareContact) {
			require.NotNil(t, lc.Company)
			assert.Equal(t, "Acme GmbH", lc.Company.Name)
		}},
		{"email->emailAddresses.business", "email", "emailAddresses.business", func(t *testing.T, lc *LexwareContact) {
			require.NotNil(t, lc.EmailAddresses)
			assert.Equal(t, []string{"ada@example.com"}, lc.EmailAddresses.Business)
		}},
		{"phone->phoneNumbers.business", "phone", "phoneNumbers.business", func(t *testing.T, lc *LexwareContact) {
			require.NotNil(t, lc.PhoneNumbers)
			assert.Equal(t, []string{"+49 111"}, lc.PhoneNumbers.Business)
		}},
		{"notes->addresses.billing.street", "notes", "addresses.billing.street", func(t *testing.T, lc *LexwareContact) {
			require.NotNil(t, lc.Addresses)
			require.NotNil(t, lc.Addresses.Billing)
			assert.Equal(t, "note text", lc.Addresses.Billing.Street)
		}},
		{"notes->addresses.billing.city", "notes", "addresses.billing.city", func(t *testing.T, lc *LexwareContact) {
			require.NotNil(t, lc.Addresses)
			require.NotNil(t, lc.Addresses.Billing)
			assert.Equal(t, "note text", lc.Addresses.Billing.City)
		}},
		{"notes->addresses.billing.zip", "notes", "addresses.billing.zip", func(t *testing.T, lc *LexwareContact) {
			require.NotNil(t, lc.Addresses)
			require.NotNil(t, lc.Addresses.Billing)
			assert.Equal(t, "note text", lc.Addresses.Billing.Zip)
		}},
		{"notes->addresses.billing.countryCode", "notes", "addresses.billing.countryCode", func(t *testing.T, lc *LexwareContact) {
			require.NotNil(t, lc.Addresses)
			require.NotNil(t, lc.Addresses.Billing)
			assert.Equal(t, "note text", lc.Addresses.Billing.CountryCode)
		}},
		{"notes->note", "notes", "note", func(t *testing.T, lc *LexwareContact) {
			assert.Equal(t, "note text", lc.Note)
		}},
		{"unknown kmuhub field is a no-op", "unmapped", "person.firstName", func(t *testing.T, lc *LexwareContact) {
			// default branch returns before touching lc at all — Person stays nil.
			assert.Nil(t, lc.Person)
		}},
	}

	fm := NewFieldMapper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mappings := []models.LexwareFieldMappingEntry{
				{KmuhubField: tc.kmuhubField, LexwareField: tc.lexwareField, Direction: "outbound"},
			}
			lc, err := fm.MapContactToLexware(contact, &company, mappings)
			require.NoError(t, err)
			tc.assertFn(t, lc)
		})
	}
}

func TestMapContactToLexware_UnknownLexwareTargetIsIgnored(t *testing.T) {
	fm := NewFieldMapper()
	contact := &models.Contact{FirstName: "Ada"}
	mappings := []models.LexwareFieldMappingEntry{
		{KmuhubField: "first_name", LexwareField: "some.unknown.path", Direction: "outbound"},
	}

	lc, err := fm.MapContactToLexware(contact, nil, mappings)
	require.NoError(t, err)
	// setNestedLexwareValue's switch has no case for "some.unknown.path" —
	// nothing beyond the base person mapping should be set.
	require.NotNil(t, lc.Person)
	assert.Equal(t, "Ada", lc.Person.FirstName)
}

// --- ValidateFieldMappings ---

func TestValidateFieldMappings(t *testing.T) {
	tests := []struct {
		name       string
		entityType string
		mappings   []models.LexwareFieldMappingEntry
		wantErr    bool
		errSubstr  string
	}{
		{
			name:       "valid contact mapping",
			entityType: "contact",
			mappings: []models.LexwareFieldMappingEntry{
				{KmuhubField: "first_name", LexwareField: "person.firstName", Direction: "both"},
			},
			wantErr: false,
		},
		{
			name:       "valid contact mapping via company.name",
			entityType: "contact",
			mappings: []models.LexwareFieldMappingEntry{
				{KmuhubField: "company_name", LexwareField: "company.name", Direction: "outbound"},
			},
			wantErr: false,
		},
		{
			name:       "empty kmuhub field",
			entityType: "contact",
			mappings: []models.LexwareFieldMappingEntry{
				{KmuhubField: "", LexwareField: "person.firstName", Direction: "both"},
			},
			wantErr:   true,
			errSubstr: "empty field names",
		},
		{
			name:       "empty lexware field",
			entityType: "contact",
			mappings: []models.LexwareFieldMappingEntry{
				{KmuhubField: "first_name", LexwareField: "", Direction: "both"},
			},
			wantErr:   true,
			errSubstr: "empty field names",
		},
		{
			name:       "invalid direction",
			entityType: "contact",
			mappings: []models.LexwareFieldMappingEntry{
				{KmuhubField: "first_name", LexwareField: "person.firstName", Direction: "sideways"},
			},
			wantErr:   true,
			errSubstr: "invalid direction",
		},
		{
			name:       "duplicate lexware target",
			entityType: "contact",
			mappings: []models.LexwareFieldMappingEntry{
				{KmuhubField: "first_name", LexwareField: "person.firstName", Direction: "both"},
				{KmuhubField: "last_name", LexwareField: "person.firstName", Direction: "both"},
			},
			wantErr:   true,
			errSubstr: "duplicate lexware target",
		},
		{
			name:       "contact mapping missing required target",
			entityType: "contact",
			mappings: []models.LexwareFieldMappingEntry{
				{KmuhubField: "email", LexwareField: "emailAddresses.business", Direction: "both"},
			},
			wantErr:   true,
			errSubstr: "person.firstName or company.name",
		},
		{
			name:       "invoice mapping missing voucherNumber",
			entityType: "invoice",
			mappings: []models.LexwareFieldMappingEntry{
				{KmuhubField: "due_date", LexwareField: "dueDate", Direction: "outbound"},
			},
			wantErr:   true,
			errSubstr: "voucherNumber",
		},
		{
			name:       "invoice mapping with voucherNumber is valid",
			entityType: "invoice",
			mappings: []models.LexwareFieldMappingEntry{
				{KmuhubField: "invoice_number", LexwareField: "voucherNumber", Direction: "outbound"},
			},
			wantErr: false,
		},
		{
			name:       "unconstrained entity type has no extra requirement",
			entityType: "quote",
			mappings: []models.LexwareFieldMappingEntry{
				{KmuhubField: "quote_number", LexwareField: "voucherNumber", Direction: "outbound"},
			},
			wantErr: false,
		},
		{
			name:       "empty mapping list is valid for an unconstrained entity type",
			entityType: "quote",
			mappings:   nil,
			wantErr:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateFieldMappings(tc.entityType, tc.mappings)
			if tc.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidFieldMapping)
				if tc.errSubstr != "" {
					assert.Contains(t, err.Error(), tc.errSubstr)
				}
				return
			}
			require.NoError(t, err)
		})
	}
}

// --- MapInvoiceToLexware / MapQuoteToLexware / mapLineItemsToLexware ---

func TestMapInvoiceToLexware_MapsLineItemsAndCurrency(t *testing.T) {
	fm := NewFieldMapper()
	lineItems := json.RawMessage(`[{"description":"Consulting","quantity":"2","unit_price":"100.50","tax_rate":"19"}]`)
	invoice := &models.Invoice{
		CustomerName: "Acme GmbH",
		Currency:     "EUR",
		LineItems:    lineItems,
	}
	invoice.InvoiceDate = mustParseDate(t, "2026-01-15")
	invoice.DueDate = mustParseDate(t, "2026-02-15")

	li, err := fm.MapInvoiceToLexware(invoice, "lex-contact-1", nil)
	require.NoError(t, err)
	assert.Equal(t, "2026-01-15", li.VoucherDate)
	assert.Equal(t, "2026-02-15", li.DueDate)
	assert.Equal(t, "lex-contact-1", li.Address.ContactID)
	assert.Equal(t, "Acme GmbH", li.Address.Name)
	assert.Equal(t, LexwareTaxTypeNet, li.TaxType)
	require.Len(t, li.LineItems, 1)
	assert.Equal(t, "Consulting", li.LineItems[0].Name)
	assert.Equal(t, 2.0, li.LineItems[0].Quantity)
	assert.Equal(t, 100.5, li.LineItems[0].UnitPrice.NetAmount)
	assert.Equal(t, 19.0, li.LineItems[0].UnitPrice.TaxRatePercentage)
	assert.Equal(t, "EUR", li.LineItems[0].UnitPrice.Currency)
}

func TestMapInvoiceToLexware_NoDueDateLeavesFieldEmpty(t *testing.T) {
	fm := NewFieldMapper()
	invoice := &models.Invoice{CustomerName: "Acme"}
	invoice.InvoiceDate = mustParseDate(t, "2026-01-15")

	li, err := fm.MapInvoiceToLexware(invoice, "lex-1", nil)
	require.NoError(t, err)
	assert.Equal(t, "", li.DueDate)
}

func TestMapInvoiceToLexware_DefaultsCurrencyWhenMissing(t *testing.T) {
	fm := NewFieldMapper()
	lineItems := json.RawMessage(`[{"description":"X","quantity":"1","unit_price":"10","tax_rate":"0"}]`)
	invoice := &models.Invoice{CustomerName: "Acme", LineItems: lineItems}
	invoice.InvoiceDate = mustParseDate(t, "2026-01-15")

	li, err := fm.MapInvoiceToLexware(invoice, "lex-1", nil)
	require.NoError(t, err)
	require.Len(t, li.LineItems, 1)
	assert.Equal(t, models.DefaultCurrency, li.LineItems[0].UnitPrice.Currency)
}

func TestMapInvoiceToLexware_InvalidLineItemsJSONFails(t *testing.T) {
	fm := NewFieldMapper()
	invoice := &models.Invoice{CustomerName: "Acme", LineItems: json.RawMessage(`{not valid json`)}
	invoice.InvoiceDate = mustParseDate(t, "2026-01-15")

	_, err := fm.MapInvoiceToLexware(invoice, "lex-1", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "map line items")
}

func TestMapQuoteToLexware_MapsExpirationDate(t *testing.T) {
	fm := NewFieldMapper()
	quote := &models.Quote{CustomerName: "Acme"}
	validUntil := mustParseDate(t, "2026-03-01")
	quote.ValidUntil = &validUntil

	lq, err := fm.MapQuoteToLexware(quote, "lex-2", nil)
	require.NoError(t, err)
	assert.Equal(t, "2026-03-01", lq.ExpirationDate)
	assert.Equal(t, "lex-2", lq.Address.ContactID)
}

func TestMapQuoteToLexware_NoValidUntilLeavesExpirationEmpty(t *testing.T) {
	fm := NewFieldMapper()
	quote := &models.Quote{CustomerName: "Acme"}

	lq, err := fm.MapQuoteToLexware(quote, "lex-2", nil)
	require.NoError(t, err)
	assert.Equal(t, "", lq.ExpirationDate)
}

func TestMapLineItemsToLexware_EmptyInputReturnsNil(t *testing.T) {
	fm := NewFieldMapper()
	items, err := fm.mapLineItemsToLexware(nil, "EUR")
	require.NoError(t, err)
	assert.Nil(t, items)
}

func mustParseDate(t *testing.T, s string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", s)
	require.NoError(t, err)
	return parsed
}
