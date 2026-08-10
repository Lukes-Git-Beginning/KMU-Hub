package caldav

import (
	"bytes"
	"testing"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// vcardEncodeDecode runs a Card through the real wire format (vCard text)
// and back, like an external address book client would, instead of
// comparing in-memory vcard.Card values directly.
func vcardEncodeDecode(t *testing.T, card vcard.Card) vcard.Card {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, vcard.NewEncoder(&buf).Encode(card))
	decoded, err := vcard.NewDecoder(&buf).Decode()
	require.NoError(t, err)
	return decoded
}

func strPtr(s string) *string { return &s }

func baseContact() *models.Contact {
	return &models.Contact{
		ID:        uuid.New(),
		TenantID:  uuid.New(),
		CreatedBy: uuid.New(),
		FirstName: "Anna",
		LastName:  "Mueller",
		Email:     strPtr("anna.mueller@example.com"),
		Phone:     strPtr("+49 30 1234567"),
		Position:  strPtr("Einkaufsleiterin"),
		Notes:     strPtr("VIP-Kunde, bevorzugt Anruf statt E-Mail"),
		UpdatedAt: time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
	}
}

func TestContactToVCard_FullFields(t *testing.T) {
	contact := baseContact()

	card := ContactToVCard(contact)

	assert.Equal(t, "4.0", card.Value(vcard.FieldVersion))
	assert.Equal(t, contact.ID.String(), card.Value(vcard.FieldUID))
	assert.Equal(t, "Anna Mueller", card.Value(vcard.FieldFormattedName))

	name := card.Name()
	require.NotNil(t, name)
	assert.Equal(t, "Mueller", name.FamilyName)
	assert.Equal(t, "Anna", name.GivenName)

	assert.Equal(t, "anna.mueller@example.com", card.Value(vcard.FieldEmail))
	assert.Equal(t, "+49 30 1234567", card.Value(vcard.FieldTelephone))
	assert.Equal(t, "Einkaufsleiterin", card.Value(vcard.FieldTitle))
	assert.Equal(t, "VIP-Kunde, bevorzugt Anruf statt E-Mail", card.Value(vcard.FieldNote))

	rev, err := card.Revision()
	require.NoError(t, err)
	assert.True(t, contact.UpdatedAt.Equal(rev), "revision must survive: got %v want %v", rev, contact.UpdatedAt)
}

func TestContactToVCard_MissingOptionalFields(t *testing.T) {
	contact := baseContact()
	contact.Email = nil
	contact.Phone = nil
	contact.Position = nil
	contact.Notes = nil

	card := ContactToVCard(contact)

	assert.Empty(t, card.Value(vcard.FieldEmail))
	assert.Empty(t, card.Value(vcard.FieldTelephone))
	assert.Empty(t, card.Value(vcard.FieldTitle))
	assert.Empty(t, card.Value(vcard.FieldNote))
	// FN/N must still be present regardless of optional fields
	assert.Equal(t, "Anna Mueller", card.Value(vcard.FieldFormattedName))
}

func TestContactToVCard_EmptyStringFieldsAreOmitted(t *testing.T) {
	contact := baseContact()
	contact.Email = strPtr("")
	contact.Phone = strPtr("")
	contact.Position = strPtr("")
	contact.Notes = strPtr("")

	card := ContactToVCard(contact)

	assert.Empty(t, card.Value(vcard.FieldEmail))
	assert.Empty(t, card.Value(vcard.FieldTelephone))
	assert.Empty(t, card.Value(vcard.FieldTitle))
	assert.Empty(t, card.Value(vcard.FieldNote))
}

func TestContactToVCard_NoName_FallsBackToUnnamed(t *testing.T) {
	contact := baseContact()
	contact.FirstName = ""
	contact.LastName = ""

	card := ContactToVCard(contact)

	assert.Equal(t, "Unnamed Contact", card.Value(vcard.FieldFormattedName))
}

func TestContactToVCardWithCompany_SetsOrg(t *testing.T) {
	contact := baseContact()
	company := "Zentria GmbH"

	card := ContactToVCardWithCompany(contact, &company)

	assert.Equal(t, "Zentria GmbH", card.Value(vcard.FieldOrganization))
	// base fields must still be present
	assert.Equal(t, "Anna Mueller", card.Value(vcard.FieldFormattedName))
}

func TestContactToVCardWithCompany_NilOrEmptyCompany_NoOrg(t *testing.T) {
	contact := baseContact()

	cardNil := ContactToVCardWithCompany(contact, nil)
	assert.Empty(t, cardNil.Value(vcard.FieldOrganization))

	empty := ""
	cardEmpty := ContactToVCardWithCompany(contact, &empty)
	assert.Empty(t, cardEmpty.Value(vcard.FieldOrganization))
}

func TestVCardRoundTrip_FullContact(t *testing.T) {
	contact := baseContact()

	card := ContactToVCard(contact)
	decoded := vcardEncodeDecode(t, card)
	input := VCardToContactInput(decoded)

	assert.Equal(t, contact.FirstName, input.FirstName)
	assert.Equal(t, contact.LastName, input.LastName)
	require.NotNil(t, input.Email)
	assert.Equal(t, *contact.Email, *input.Email)
	require.NotNil(t, input.Phone)
	assert.Equal(t, *contact.Phone, *input.Phone)
	require.NotNil(t, input.Position)
	assert.Equal(t, *contact.Position, *input.Position)
	require.NotNil(t, input.Notes)
	assert.Equal(t, *contact.Notes, *input.Notes)
}

// TestVCardRoundTrip_CompanyOnlyName covers a contact with no surname (only
// a first-name field holding a company-style name, e.g. imported from a
// legacy system) -- the N field's GivenName must roundtrip even though
// FamilyName is empty.
func TestVCardRoundTrip_CompanyOnlyName(t *testing.T) {
	contact := baseContact()
	contact.FirstName = "Zentria AG"
	contact.LastName = ""

	card := ContactToVCard(contact)
	decoded := vcardEncodeDecode(t, card)
	input := VCardToContactInput(decoded)

	assert.Equal(t, "Zentria AG", input.FirstName)
	assert.Empty(t, input.LastName)
}

func TestVCardToContactInput_FallbackToFN_MultiWord(t *testing.T) {
	card := make(vcard.Card)
	card.SetValue(vcard.FieldFormattedName, "John Middle Doe")
	// deliberately no N field -- forces the FN-splitting fallback path

	input := VCardToContactInput(card)

	assert.Equal(t, "John", input.FirstName)
	assert.Equal(t, "Middle Doe", input.LastName)
}

func TestVCardToContactInput_FallbackToFN_SingleWord(t *testing.T) {
	card := make(vcard.Card)
	card.SetValue(vcard.FieldFormattedName, "Cher")

	input := VCardToContactInput(card)

	assert.Equal(t, "Cher", input.FirstName)
	assert.Empty(t, input.LastName)
}

func TestVCardToContactInput_NoNameNoFN_EmptyResult(t *testing.T) {
	card := make(vcard.Card)
	card.SetValue(vcard.FieldEmail, "nobody@example.com")

	input := VCardToContactInput(card)

	assert.Empty(t, input.FirstName)
	assert.Empty(t, input.LastName)
	require.NotNil(t, input.Email)
	assert.Equal(t, "nobody@example.com", *input.Email)
}

func TestVCardToContactInput_NField_TakesPrecedenceOverFN(t *testing.T) {
	card := make(vcard.Card)
	card.SetName(&vcard.Name{FamilyName: "Mueller", GivenName: "Anna"})
	card.SetValue(vcard.FieldFormattedName, "Completely Different Name")

	input := VCardToContactInput(card)

	assert.Equal(t, "Anna", input.FirstName)
	assert.Equal(t, "Mueller", input.LastName)
}

// TestVCardMultipleEmailsAndPhones_LibraryKeepsAllButConverterPicksPreferred
// documents the current (and intentional) behaviour: an incoming vCard can
// carry several EMAIL/TEL entries with TYPE labels and a PREF marker -- the
// underlying go-vcard library preserves every one of them through a real
// wire-format roundtrip, but ContactInput only has room for a single
// Email/Phone (matching models.Contact), so VCardToContactInput
// deterministically extracts just the preferred entry.
func TestVCardMultipleEmailsAndPhones_LibraryKeepsAllButConverterPicksPreferred(t *testing.T) {
	card := make(vcard.Card)
	card.SetValue(vcard.FieldVersion, "4.0")
	card.SetValue(vcard.FieldFormattedName, "Anna Mueller")

	workEmail := &vcard.Field{Value: "anna@work.example.com", Params: vcard.Params{
		vcard.ParamType: []string{vcard.TypeWork},
	}}
	homeEmail := &vcard.Field{Value: "anna@home.example.com", Params: vcard.Params{
		vcard.ParamType:      []string{vcard.TypeHome},
		vcard.ParamPreferred: []string{"1"},
	}}
	card.Add(vcard.FieldEmail, workEmail)
	card.Add(vcard.FieldEmail, homeEmail)

	mobileTel := &vcard.Field{Value: "+49 170 1112222", Params: vcard.Params{
		vcard.ParamType:      []string{vcard.TypeCell},
		vcard.ParamPreferred: []string{"1"},
	}}
	officeTel := &vcard.Field{Value: "+49 30 9998888", Params: vcard.Params{
		vcard.ParamType: []string{vcard.TypeWork},
	}}
	card.Add(vcard.FieldTelephone, mobileTel)
	card.Add(vcard.FieldTelephone, officeTel)

	decoded := vcardEncodeDecode(t, card)

	// Library level: all entries survive the wire roundtrip.
	emails := decoded.Values(vcard.FieldEmail)
	assert.ElementsMatch(t, []string{"anna@work.example.com", "anna@home.example.com"}, emails)
	tels := decoded.Values(vcard.FieldTelephone)
	assert.ElementsMatch(t, []string{"+49 170 1112222", "+49 30 9998888"}, tels)

	// Converter level: only the preferred (PREF=1) entry is extracted.
	input := VCardToContactInput(decoded)
	require.NotNil(t, input.Email)
	assert.Equal(t, "anna@home.example.com", *input.Email)
	require.NotNil(t, input.Phone)
	assert.Equal(t, "+49 170 1112222", *input.Phone)
}
