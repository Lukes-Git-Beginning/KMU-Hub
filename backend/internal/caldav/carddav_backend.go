package caldav

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/carddav"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/gateway"
	"github.com/kmuhub/kmuhub/internal/sysctx"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

// Compile-time interface compliance check.
var _ carddav.Backend = (*CardDAVBackend)(nil)

// CardDAVBackend implements the carddav.Backend interface from go-webdav,
// backed by KMU Hub's CRM gRPC service.
type CardDAVBackend struct {
	registry    *gateway.ServiceRegistry
	syncService *SyncTokenService
	pool        *pgxpool.Pool
}

// NewCardDAVBackend creates a new CardDAV backend adapter.
func NewCardDAVBackend(registry *gateway.ServiceRegistry, syncService *SyncTokenService, pool *pgxpool.Pool) *CardDAVBackend {
	return &CardDAVBackend{
		registry:    registry,
		syncService: syncService,
		pool:        pool,
	}
}

// crmClient returns a CRMServiceClient from the service registry.
func (b *CardDAVBackend) crmClient() (crmv1.CRMServiceClient, error) {
	conn, err := b.registry.GetConnection("crm")
	if err != nil {
		return nil, fmt.Errorf("get CRM gRPC connection: %w", err)
	}
	return crmv1.NewCRMServiceClient(conn), nil
}

// CurrentUserPrincipal returns the principal URL for the authenticated user.
func (b *CardDAVBackend) CurrentUserPrincipal(ctx context.Context) (string, error) {
	userID := UserFromCtx(ctx)
	if userID == uuid.Nil {
		return "", webdav.NewHTTPError(http.StatusUnauthorized, fmt.Errorf("not authenticated"))
	}
	return fmt.Sprintf("/carddav/principals/%s/", userID.String()), nil
}

// AddressBookHomeSetPath returns the address book home set path for the authenticated user.
func (b *CardDAVBackend) AddressBookHomeSetPath(ctx context.Context) (string, error) {
	userID := UserFromCtx(ctx)
	if userID == uuid.Nil {
		return "", webdav.NewHTTPError(http.StatusUnauthorized, fmt.Errorf("not authenticated"))
	}
	return fmt.Sprintf("/carddav/principals/%s/addressbooks/", userID.String()), nil
}

// ListAddressBooks returns the address books for the authenticated user.
// Two address books per user: personal contacts and company contacts.
func (b *CardDAVBackend) ListAddressBooks(ctx context.Context) ([]carddav.AddressBook, error) {
	userID := UserFromCtx(ctx)
	return []carddav.AddressBook{
		{
			Path:        fmt.Sprintf("/carddav/principals/%s/addressbooks/personal/", userID.String()),
			Name:        "Meine Kontakte",
			Description: "Persoenliche Kontakte",
		},
		{
			Path:        fmt.Sprintf("/carddav/principals/%s/addressbooks/company/", userID.String()),
			Name:        "Firmenkontakte",
			Description: "Geteilte Firmenkontakte",
		},
	}, nil
}

// GetAddressBook returns a specific address book by path.
func (b *CardDAVBackend) GetAddressBook(ctx context.Context, path string) (*carddav.AddressBook, error) {
	userID := UserFromCtx(ctx)
	bookType := addressBookTypeFromPath(path)

	switch bookType {
	case "personal":
		return &carddav.AddressBook{
			Path:        fmt.Sprintf("/carddav/principals/%s/addressbooks/personal/", userID.String()),
			Name:        "Meine Kontakte",
			Description: "Persoenliche Kontakte",
		}, nil
	case "company":
		return &carddav.AddressBook{
			Path:        fmt.Sprintf("/carddav/principals/%s/addressbooks/company/", userID.String()),
			Name:        "Firmenkontakte",
			Description: "Geteilte Firmenkontakte",
		}, nil
	default:
		return nil, webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("address book %q not found", bookType))
	}
}

// CreateAddressBook creates a new address book. Not supported -- address books
// are fixed (personal + company).
func (b *CardDAVBackend) CreateAddressBook(ctx context.Context, addressBook *carddav.AddressBook) error {
	return webdav.NewHTTPError(http.StatusForbidden, fmt.Errorf("address book creation via CardDAV is not supported"))
}

// DeleteAddressBook deletes an address book. Not supported -- address books are fixed.
func (b *CardDAVBackend) DeleteAddressBook(ctx context.Context, path string) error {
	return webdav.NewHTTPError(http.StatusForbidden, fmt.Errorf("address book deletion via CardDAV is not supported"))
}

// GetAddressObject returns a single contact as a vCard by path.
func (b *CardDAVBackend) GetAddressObject(ctx context.Context, path string, req *carddav.AddressDataRequest) (*carddav.AddressObject, error) {
	userID := UserFromCtx(ctx)
	contactID, err := contactIDFromPath(path)
	if err != nil {
		return nil, webdav.NewHTTPError(http.StatusBadRequest, err)
	}

	client, err := b.crmClient()
	if err != nil {
		return nil, grpcToWebDAVError(err)
	}

	resp, err := client.GetContact(ctx, &crmv1.GetContactRequest{
		Id: contactID.String(),
	})
	if err != nil {
		return nil, grpcToWebDAVError(err)
	}

	contact := resp.Contact
	card := contactInfoToVCard(contact)
	bookType := addressBookTypeFromPath(path)
	modTime := parseContactUpdatedAt(contact.UpdatedAt)

	return &carddav.AddressObject{
		Path:    fmt.Sprintf("/carddav/principals/%s/addressbooks/%s/%s.vcf", userID.String(), bookType, contact.Id),
		ModTime: modTime,
		ETag:    GenerateETag(uuid.MustParse(contact.Id), modTime),
		Card:    card,
	}, nil
}

// ListAddressObjects returns all contacts in an address book.
func (b *CardDAVBackend) ListAddressObjects(ctx context.Context, path string, req *carddav.AddressDataRequest) ([]carddav.AddressObject, error) {
	userID := UserFromCtx(ctx)
	bookType := addressBookTypeFromPath(path)

	client, err := b.crmClient()
	if err != nil {
		return nil, grpcToWebDAVError(err)
	}

	listReq := &crmv1.ListContactsRequest{
		PageSize: 1000, // Reasonable limit for CalDAV sync
		UserId:   userID.String(),
	}

	switch bookType {
	case "personal":
		listReq.VisibilityFilter = "personal"
	case "company":
		listReq.VisibilityFilter = "shared"
	}

	resp, err := client.ListContacts(ctx, listReq)
	if err != nil {
		return nil, grpcToWebDAVError(err)
	}

	objects := make([]carddav.AddressObject, 0, len(resp.Contacts))
	for _, contact := range resp.Contacts {
		card := contactInfoToVCard(contact)
		modTime := parseContactUpdatedAt(contact.UpdatedAt)

		objects = append(objects, carddav.AddressObject{
			Path:    fmt.Sprintf("/carddav/principals/%s/addressbooks/%s/%s.vcf", userID.String(), bookType, contact.Id),
			ModTime: modTime,
			ETag:    GenerateETag(uuid.MustParse(contact.Id), modTime),
			Card:    card,
		})
	}

	return objects, nil
}

// QueryAddressObjects returns contacts matching a query. Currently returns
// all contacts in the address book (filtering done by the go-webdav layer).
func (b *CardDAVBackend) QueryAddressObjects(ctx context.Context, path string, query *carddav.AddressBookQuery) ([]carddav.AddressObject, error) {
	// For now, return all contacts and let go-webdav handle filtering
	return b.ListAddressObjects(ctx, path, nil)
}

// PutAddressObject creates or updates a contact from a vCard.
func (b *CardDAVBackend) PutAddressObject(ctx context.Context, path string, card vcard.Card, opts *carddav.PutAddressObjectOptions) (*carddav.AddressObject, error) {
	userID := UserFromCtx(ctx)
	bookType := addressBookTypeFromPath(path)

	// ACL check: company address book requires admin/manager role
	if bookType == "company" {
		if err := b.checkCompanyContactPermission(ctx, userID); err != nil {
			return nil, err
		}
	}

	contactID, err := contactIDFromPath(path)
	if err != nil {
		return nil, webdav.NewHTTPError(http.StatusBadRequest, err)
	}

	input := VCardToContactInput(card)

	client, err := b.crmClient()
	if err != nil {
		return nil, grpcToWebDAVError(err)
	}

	// Try to get existing contact to decide create vs update
	var resultContactID string
	existingResp, err := client.GetContact(ctx, &crmv1.GetContactRequest{
		Id: contactID.String(),
	})
	if err == nil && existingResp.Contact != nil {
		// Update existing contact
		resultContactID = existingResp.Contact.Id
		updateReq := &crmv1.UpdateContactRequest{
			Id: resultContactID,
		}
		if input.FirstName != "" {
			updateReq.FirstName = &input.FirstName
		}
		if input.LastName != "" {
			updateReq.LastName = &input.LastName
		}
		if input.Email != nil {
			updateReq.Email = input.Email
		}
		if input.Phone != nil {
			updateReq.Phone = input.Phone
		}
		if input.Position != nil {
			updateReq.Position = input.Position
		}
		if input.Notes != nil {
			updateReq.Notes = input.Notes
		}

		_, updateErr := client.UpdateContact(ctx, updateReq)
		if updateErr != nil {
			return nil, grpcToWebDAVError(updateErr)
		}
	} else {
		// Create new contact
		visibility := "personal"
		if bookType == "company" {
			visibility = "shared"
		}

		firstName := input.FirstName
		if firstName == "" {
			firstName = "Unnamed"
		}
		lastName := input.LastName

		createReq := &crmv1.CreateContactRequest{
			FirstName: firstName,
			LastName:  lastName,
			CreatedBy: userID.String(),
		}
		if input.Email != nil {
			createReq.Email = *input.Email
		}
		if input.Phone != nil {
			createReq.Phone = *input.Phone
		}
		if input.Position != nil {
			createReq.Position = *input.Position
		}
		if input.Notes != nil {
			createReq.Notes = *input.Notes
		}

		createResp, createErr := client.CreateContact(ctx, createReq)
		if createErr != nil {
			return nil, grpcToWebDAVError(createErr)
		}
		resultContactID = createResp.Contact.Id

		// Set visibility for the created contact
		_, visErr := client.UpdateContactVisibility(ctx, &crmv1.UpdateContactVisibilityRequest{
			ContactId:  resultContactID,
			Visibility: visibility,
			UserId:     userID.String(),
			IsAdmin:    bookType == "company",
		})
		if visErr != nil {
			slog.Warn("failed to set contact visibility via CardDAV",
				"contact_id", resultContactID,
				"visibility", visibility,
				"error", visErr,
			)
		}
	}

	// Increment sync token
	syncCollectionID := syncCollectionIDForAddressBook(userID, bookType)
	if syncErr := b.syncService.IncrementAndLog(ctx, "addressbook", syncCollectionID, path, "modified"); syncErr != nil {
		slog.Warn("failed to increment sync token after CardDAV PUT",
			"book_type", bookType,
			"error", syncErr,
		)
	}

	now := time.Now()
	return &carddav.AddressObject{
		Path:    fmt.Sprintf("/carddav/principals/%s/addressbooks/%s/%s.vcf", userID.String(), bookType, resultContactID),
		ModTime: now,
		ETag:    GenerateETag(uuid.MustParse(resultContactID), now),
		Card:    card,
	}, nil
}

// DeleteAddressObject deletes a contact.
func (b *CardDAVBackend) DeleteAddressObject(ctx context.Context, path string) error {
	userID := UserFromCtx(ctx)
	bookType := addressBookTypeFromPath(path)

	// ACL check for company contacts
	if bookType == "company" {
		if err := b.checkCompanyContactPermission(ctx, userID); err != nil {
			return err
		}
	}

	// For personal contacts, verify ownership
	if bookType == "personal" {
		if err := b.checkPersonalContactOwnership(ctx, userID, path); err != nil {
			return err
		}
	}

	contactID, err := contactIDFromPath(path)
	if err != nil {
		return webdav.NewHTTPError(http.StatusBadRequest, err)
	}

	client, err := b.crmClient()
	if err != nil {
		return grpcToWebDAVError(err)
	}

	_, err = client.DeleteContact(ctx, &crmv1.DeleteContactRequest{
		Id: contactID.String(),
	})
	if err != nil {
		return grpcToWebDAVError(err)
	}

	// Increment sync token
	syncCollectionID := syncCollectionIDForAddressBook(userID, bookType)
	if syncErr := b.syncService.IncrementAndLog(ctx, "addressbook", syncCollectionID, path, "deleted"); syncErr != nil {
		slog.Warn("failed to increment sync token after CardDAV DELETE",
			"book_type", bookType,
			"error", syncErr,
		)
	}

	return nil
}

// checkCompanyContactPermission verifies the user has admin or manager role
// for modifying company (shared) contacts.
func (b *CardDAVBackend) checkCompanyContactPermission(ctx context.Context, userID uuid.UUID) error {
	// CardDAV's BasicAuth middleware injects userID but no tenant. After RLS
	// activation, a direct read on `users` would return zero rows.
	// App-passwords are single-tenant in practice (Welle-3 follow-up: inject
	// tenant_id from app_specific_passwords for proper multi-tenant CardDAV).
	ctx = sysctx.With(ctx)

	var role string
	err := b.pool.QueryRow(ctx,
		`SELECT role FROM users WHERE id = $1`,
		userID,
	).Scan(&role)
	if err != nil {
		return webdav.NewHTTPError(http.StatusForbidden, fmt.Errorf("unable to verify user role"))
	}

	if role != "admin" && role != "manager" {
		return webdav.NewHTTPError(http.StatusForbidden, fmt.Errorf("insufficient permissions to modify company contacts"))
	}
	return nil
}

// checkPersonalContactOwnership verifies the user owns a personal contact.
func (b *CardDAVBackend) checkPersonalContactOwnership(ctx context.Context, userID uuid.UUID, path string) error {
	contactID, err := contactIDFromPath(path)
	if err != nil {
		return webdav.NewHTTPError(http.StatusBadRequest, err)
	}

	// Same rationale as checkCompanyContactPermission — BasicAuth has no tenant.
	ctx = sysctx.With(ctx)

	var ownerID *uuid.UUID
	err = b.pool.QueryRow(ctx,
		`SELECT owner_id FROM contacts WHERE id = $1`,
		contactID,
	).Scan(&ownerID)
	if err != nil {
		return webdav.NewHTTPError(http.StatusNotFound, fmt.Errorf("contact not found"))
	}

	if ownerID == nil || *ownerID != userID {
		return webdav.NewHTTPError(http.StatusForbidden, fmt.Errorf("not the owner of this personal contact"))
	}
	return nil
}

// Path parsing helpers

// addressBookTypeFromPath extracts the address book type from a CardDAV path.
// Returns "personal" or "company".
func addressBookTypeFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, part := range parts {
		if part == "addressbooks" && i+1 < len(parts) {
			bookType := parts[i+1]
			if bookType == "personal" || bookType == "company" {
				return bookType
			}
		}
	}
	return "personal" // Default to personal
}

// contactIDFromPath extracts the contact UUID from a CardDAV path.
// Expected path: .../addressbooks/{type}/{contactID}.vcf
func contactIDFromPath(path string) (uuid.UUID, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return uuid.Nil, fmt.Errorf("contact ID not found in path: %s", path)
	}
	last := parts[len(parts)-1]
	// Strip .vcf extension
	uid := strings.TrimSuffix(last, ".vcf")
	return uuid.Parse(uid)
}

// syncCollectionIDForAddressBook generates a deterministic UUID for sync token
// tracking per user + address book type combination.
func syncCollectionIDForAddressBook(userID uuid.UUID, bookType string) uuid.UUID {
	// Use a namespace UUID to generate a deterministic ID
	// This ensures the same user + bookType always gets the same sync collection ID
	return uuid.NewSHA1(userID, []byte("addressbook:"+bookType))
}

// contactInfoToVCard converts a CRM ContactInfo proto message to a vCard Card.
func contactInfoToVCard(contact *crmv1.ContactInfo) vcard.Card {
	card := make(vcard.Card)

	// VERSION:4.0
	card.SetValue(vcard.FieldVersion, "4.0")

	// UID
	card.SetValue(vcard.FieldUID, contact.Id)

	// FN (Formatted Name)
	fn := strings.TrimSpace(contact.FirstName + " " + contact.LastName)
	if fn == "" {
		fn = "Unnamed Contact"
	}
	card.SetValue(vcard.FieldFormattedName, fn)

	// N (Name)
	card.SetName(&vcard.Name{
		FamilyName: contact.LastName,
		GivenName:  contact.FirstName,
	})

	// EMAIL
	if contact.Email != "" {
		card.SetValue(vcard.FieldEmail, contact.Email)
	}

	// TEL
	if contact.Phone != "" {
		card.SetValue(vcard.FieldTelephone, contact.Phone)
	}

	// ORG (CompanyName)
	if contact.CompanyName != nil && *contact.CompanyName != "" {
		card.SetValue(vcard.FieldOrganization, *contact.CompanyName)
	}

	// TITLE (Position)
	if contact.Position != "" {
		card.SetValue(vcard.FieldTitle, contact.Position)
	}

	// NOTE
	if contact.Notes != "" {
		card.SetValue(vcard.FieldNote, contact.Notes)
	}

	// REV
	modTime := parseContactUpdatedAt(contact.UpdatedAt)
	card.SetRevision(modTime)

	return card
}

// parseContactUpdatedAt parses the UpdatedAt string from ContactInfo proto.
func parseContactUpdatedAt(updatedAt string) time.Time {
	t, err := time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		// Try other common formats
		t, err = time.Parse("2006-01-02T15:04:05Z07:00", updatedAt)
		if err != nil {
			return time.Now()
		}
	}
	return t
}
