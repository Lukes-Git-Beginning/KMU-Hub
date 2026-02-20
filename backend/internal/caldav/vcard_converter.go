package caldav

import (
	"strings"

	"github.com/emersion/go-vcard"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ContactInput represents parsed vCard data for creating/updating CRM contacts.
// This is an intermediate type -- the caller maps it to gRPC create/update requests.
type ContactInput struct {
	FirstName string
	LastName  string
	Email     *string
	Phone     *string
	Position  *string
	Notes     *string
}

// ContactToVCard converts a CRM Contact model to a vCard 4.0 Card.
func ContactToVCard(contact *models.Contact) vcard.Card {
	card := make(vcard.Card)

	// VERSION:4.0
	card.SetValue(vcard.FieldVersion, "4.0")

	// UID: contact UUID
	card.SetValue(vcard.FieldUID, contact.ID.String())

	// FN (Formatted Name)
	fn := strings.TrimSpace(contact.FirstName + " " + contact.LastName)
	if fn == "" {
		fn = "Unnamed Contact"
	}
	card.SetValue(vcard.FieldFormattedName, fn)

	// N (Name): Family;Given;;;
	card.SetName(&vcard.Name{
		FamilyName: contact.LastName,
		GivenName:  contact.FirstName,
	})

	// EMAIL
	if contact.Email != nil && *contact.Email != "" {
		card.SetValue(vcard.FieldEmail, *contact.Email)
	}

	// TEL
	if contact.Phone != nil && *contact.Phone != "" {
		card.SetValue(vcard.FieldTelephone, *contact.Phone)
	}

	// TITLE (Position)
	if contact.Position != nil && *contact.Position != "" {
		card.SetValue(vcard.FieldTitle, *contact.Position)
	}

	// NOTE
	if contact.Notes != nil && *contact.Notes != "" {
		card.SetValue(vcard.FieldNote, *contact.Notes)
	}

	// REV (Revision timestamp)
	card.SetRevision(contact.UpdatedAt)

	return card
}

// ContactToVCardWithCompany converts a CRM Contact with relations to a vCard 4.0 Card.
// This includes the ORG field from CompanyName if available.
func ContactToVCardWithCompany(contact *models.Contact, companyName *string) vcard.Card {
	card := ContactToVCard(contact)

	// ORG (from company name if available)
	if companyName != nil && *companyName != "" {
		card.SetValue(vcard.FieldOrganization, *companyName)
	}

	return card
}

// VCardToContactInput parses a vCard Card into a ContactInput.
func VCardToContactInput(card vcard.Card) *ContactInput {
	input := &ContactInput{}

	// Parse name from N field first, fall back to FN
	if name := card.Name(); name != nil {
		input.FirstName = name.GivenName
		input.LastName = name.FamilyName
	}

	// If N was empty, try to parse from FN
	if input.FirstName == "" && input.LastName == "" {
		fn := card.PreferredValue(vcard.FieldFormattedName)
		if fn != "" {
			parts := strings.SplitN(fn, " ", 2)
			input.FirstName = parts[0]
			if len(parts) > 1 {
				input.LastName = parts[1]
			}
		}
	}

	// EMAIL (first value)
	if email := card.PreferredValue(vcard.FieldEmail); email != "" {
		input.Email = &email
	}

	// TEL (first value)
	if tel := card.PreferredValue(vcard.FieldTelephone); tel != "" {
		input.Phone = &tel
	}

	// TITLE -> Position
	if title := card.PreferredValue(vcard.FieldTitle); title != "" {
		input.Position = &title
	}

	// NOTE
	if note := card.PreferredValue(vcard.FieldNote); note != "" {
		input.Notes = &note
	}

	return input
}
