package contact

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"log/slog"

	"github.com/emersion/go-vcard"
	"github.com/google/uuid"
)

// ExportService handles exporting contacts to CSV and vCard formats.
type ExportService struct {
	contactProvider ContactProvider
	logger          *slog.Logger
}

// NewExportService creates a new ExportService.
func NewExportService(cp ContactProvider, logger *slog.Logger) *ExportService {
	if logger == nil {
		logger = slog.Default()
	}
	return &ExportService{
		contactProvider: cp,
		logger:          logger,
	}
}

// crmFieldToGermanHeader maps CRM field names to German column headers for CSV export.
var crmFieldToGermanHeader = map[string]string{
	"first_name": "Vorname",
	"last_name":  "Nachname",
	"email":      "E-Mail",
	"phone":      "Telefon",
	"company":    "Firma",
	"position":   "Position",
	"notes":      "Notizen",
}

// allExportFields is the ordered list of all exportable fields.
var allExportFields = []string{"first_name", "last_name", "email", "phone", "company", "position", "notes"}

// ExportCSV exports selected contacts to CSV with the specified fields.
// Returns CSV bytes with UTF-8 BOM for Excel compatibility.
func (s *ExportService) ExportCSV(ctx context.Context, contactIDs []uuid.UUID, fields []string) ([]byte, error) {
	if len(fields) == 0 {
		fields = allExportFields
	}

	contacts, err := s.contactProvider.ListByIDs(ctx, contactIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch contacts: %w", err)
	}

	var buf bytes.Buffer

	// UTF-8 BOM for Excel compatibility (same pattern as audit log CSV in Phase 9)
	buf.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(&buf)

	// Write German header row
	var headers []string
	for _, f := range fields {
		if h, ok := crmFieldToGermanHeader[f]; ok {
			headers = append(headers, h)
		} else {
			headers = append(headers, f)
		}
	}
	if err := writer.Write(headers); err != nil {
		return nil, err
	}

	// Write data rows
	for _, c := range contacts {
		var row []string
		for _, f := range fields {
			switch f {
			case "first_name":
				row = append(row, c.FirstName)
			case "last_name":
				row = append(row, c.LastName)
			case "email":
				if c.Email != nil {
					row = append(row, *c.Email)
				} else {
					row = append(row, "")
				}
			case "phone":
				if c.Phone != nil {
					row = append(row, *c.Phone)
				} else {
					row = append(row, "")
				}
			case "company":
				// Company name is not on the contact model directly,
				// so we leave empty for now (would need company join).
				row = append(row, "")
			case "position":
				if c.Position != nil {
					row = append(row, *c.Position)
				} else {
					row = append(row, "")
				}
			case "notes":
				if c.Notes != nil {
					row = append(row, *c.Notes)
				} else {
					row = append(row, "")
				}
			default:
				row = append(row, "")
			}
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	s.logger.Info("CSV export complete",
		"contacts", len(contacts),
		"fields", len(fields),
	)

	return buf.Bytes(), nil
}

// ExportAllCSV exports all contacts visible to the given user.
func (s *ExportService) ExportAllCSV(ctx context.Context, userID uuid.UUID, fields []string, isAdmin bool) ([]byte, error) {
	contacts, err := s.contactProvider.ListVisible(ctx, userID, isAdmin)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch contacts: %w", err)
	}

	// Collect IDs for re-use of ExportCSV logic
	var ids []uuid.UUID
	for _, c := range contacts {
		ids = append(ids, c.ID)
	}

	return s.ExportCSV(ctx, ids, fields)
}

// ExportVCard exports selected contacts as vCard 4.0.
func (s *ExportService) ExportVCard(ctx context.Context, contactIDs []uuid.UUID) ([]byte, error) {
	contacts, err := s.contactProvider.ListByIDs(ctx, contactIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch contacts: %w", err)
	}

	var buf bytes.Buffer
	enc := vcard.NewEncoder(&buf)

	for _, c := range contacts {
		card := make(vcard.Card)

		card.SetValue(vcard.FieldVersion, "4.0")

		// Full name
		fn := c.FirstName
		if c.LastName != "" {
			if fn != "" {
				fn += " "
			}
			fn += c.LastName
		}
		card.SetValue(vcard.FieldFormattedName, fn)

		// Structured name
		card.SetValue(vcard.FieldName, c.LastName+";"+c.FirstName+";;;")

		// Email
		if c.Email != nil && *c.Email != "" {
			card.SetValue(vcard.FieldEmail, *c.Email)
		}

		// Phone
		if c.Phone != nil && *c.Phone != "" {
			card.SetValue(vcard.FieldTelephone, *c.Phone)
		}

		// Position/Title
		if c.Position != nil && *c.Position != "" {
			card.SetValue(vcard.FieldTitle, *c.Position)
		}

		// Notes
		if c.Notes != nil && *c.Notes != "" {
			card.SetValue(vcard.FieldNote, *c.Notes)
		}

		if err := enc.Encode(card); err != nil {
			return nil, fmt.Errorf("failed to encode vCard: %w", err)
		}
	}

	s.logger.Info("vCard export complete", "contacts", len(contacts))

	return buf.Bytes(), nil
}
