package caldav

import (
	"testing"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

func baseContactInfo() *crmv1.ContactInfo {
	return &crmv1.ContactInfo{
		Id:          uuid.New().String(),
		FirstName:   "Anna",
		LastName:    "Mueller",
		Email:       "anna.mueller@example.com",
		Phone:       "+49 30 1234567",
		CompanyName: strPtr("Zentria GmbH"),
		Position:    "Einkaufsleiterin",
		Notes:       "VIP-Kunde",
		UpdatedAt:   "2026-06-01T12:00:00Z",
	}
}

func TestAddressBookTypeFromPath_Personal(t *testing.T) {
	got := addressBookTypeFromPath("/carddav/principals/u1/addressbooks/personal/")

	assert.Equal(t, "personal", got)
}

func TestAddressBookTypeFromPath_Company(t *testing.T) {
	got := addressBookTypeFromPath("/carddav/principals/u1/addressbooks/company/")

	assert.Equal(t, "company", got)
}

func TestAddressBookTypeFromPath_MissingSegment_DefaultsToPersonal(t *testing.T) {
	got := addressBookTypeFromPath("/carddav/principals/u1/")

	assert.Equal(t, "personal", got)
}

func TestAddressBookTypeFromPath_UnknownType_DefaultsToPersonal(t *testing.T) {
	got := addressBookTypeFromPath("/carddav/principals/u1/addressbooks/shared/")

	assert.Equal(t, "personal", got)
}

func TestContactIDFromPath_Valid(t *testing.T) {
	id := uuid.New()

	got, err := contactIDFromPath("/carddav/principals/u1/addressbooks/personal/" + id.String() + ".vcf")

	require.NoError(t, err)
	assert.Equal(t, id, got)
}

func TestContactIDFromPath_InvalidUUID(t *testing.T) {
	_, err := contactIDFromPath("/carddav/principals/u1/addressbooks/personal/not-a-uuid.vcf")

	require.Error(t, err)
}

func TestContactIDFromPath_EmptyPath(t *testing.T) {
	_, err := contactIDFromPath("///")

	require.Error(t, err)
}

func TestSyncCollectionIDForAddressBook_DeterministicPerUserAndType(t *testing.T) {
	userID := uuid.New()

	got1 := syncCollectionIDForAddressBook(userID, "personal")
	got2 := syncCollectionIDForAddressBook(userID, "personal")

	assert.Equal(t, got1, got2)
}

func TestSyncCollectionIDForAddressBook_DiffersByType(t *testing.T) {
	userID := uuid.New()

	personal := syncCollectionIDForAddressBook(userID, "personal")
	company := syncCollectionIDForAddressBook(userID, "company")

	assert.NotEqual(t, personal, company)
}

func TestSyncCollectionIDForAddressBook_DiffersByUser(t *testing.T) {
	bookType := "personal"

	a := syncCollectionIDForAddressBook(uuid.New(), bookType)
	b := syncCollectionIDForAddressBook(uuid.New(), bookType)

	assert.NotEqual(t, a, b)
}

func TestContactInfoToVCard_FullFields(t *testing.T) {
	contact := baseContactInfo()

	card := contactInfoToVCard(contact)
	decoded := vcardEncodeDecode(t, card)

	assert.Equal(t, "4.0", decoded.Value(vcard.FieldVersion))
	assert.Equal(t, contact.Id, decoded.Value(vcard.FieldUID))
	assert.Equal(t, "Anna Mueller", decoded.Value(vcard.FieldFormattedName))

	name := decoded.Name()
	require.NotNil(t, name)
	assert.Equal(t, "Mueller", name.FamilyName)
	assert.Equal(t, "Anna", name.GivenName)

	assert.Equal(t, "anna.mueller@example.com", decoded.Value(vcard.FieldEmail))
	assert.Equal(t, "+49 30 1234567", decoded.Value(vcard.FieldTelephone))
	assert.Equal(t, "Zentria GmbH", decoded.Value(vcard.FieldOrganization))
	assert.Equal(t, "Einkaufsleiterin", decoded.Value(vcard.FieldTitle))
	assert.Equal(t, "VIP-Kunde", decoded.Value(vcard.FieldNote))

	rev, err := decoded.Revision()
	require.NoError(t, err)
	assert.True(t, time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC).Equal(rev))
}

func TestContactInfoToVCard_MissingOptionalFields_NoCrash(t *testing.T) {
	contact := baseContactInfo()
	contact.Email = ""
	contact.Phone = ""
	contact.CompanyName = nil
	contact.Position = ""
	contact.Notes = ""

	card := contactInfoToVCard(contact)

	assert.Empty(t, card.Value(vcard.FieldEmail))
	assert.Empty(t, card.Value(vcard.FieldTelephone))
	assert.Empty(t, card.Value(vcard.FieldOrganization))
	assert.Empty(t, card.Value(vcard.FieldTitle))
	assert.Empty(t, card.Value(vcard.FieldNote))
}

func TestContactInfoToVCard_NoName_FallsBackToUnnamed(t *testing.T) {
	contact := baseContactInfo()
	contact.FirstName = ""
	contact.LastName = ""

	card := contactInfoToVCard(contact)

	assert.Equal(t, "Unnamed Contact", card.Value(vcard.FieldFormattedName))
}

func TestParseContactUpdatedAt_RFC3339(t *testing.T) {
	got := parseContactUpdatedAt("2026-06-01T12:00:00Z")

	assert.True(t, time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC).Equal(got))
}

func TestParseContactUpdatedAt_Unparseable_FallsBackToNow(t *testing.T) {
	before := time.Now()

	got := parseContactUpdatedAt("not-a-date")

	assert.WithinDuration(t, before, got, 5*time.Second)
}

func TestParseContactUpdatedAt_Empty_FallsBackToNow(t *testing.T) {
	before := time.Now()

	got := parseContactUpdatedAt("")

	assert.WithinDuration(t, before, got, 5*time.Second)
}
