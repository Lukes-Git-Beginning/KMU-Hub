package contact

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"log/slog"
	"slices"

	"github.com/emersion/go-vcard"
	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/csvutil"
	"github.com/kmuhub/kmuhub/internal/models"
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

	companyNames, err := s.resolveCompanyNames(ctx, contacts, fields)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve company names: %w", err)
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
			var val string
			switch f {
			case "first_name":
				val = c.FirstName
			case "last_name":
				val = c.LastName
			case "email":
				if c.Email != nil {
					val = *c.Email
				}
			case "phone":
				if c.Phone != nil {
					val = *c.Phone
				}
			case "company":
				if c.CompanyID != nil {
					val = companyNames[*c.CompanyID]
				}
			case "position":
				if c.Position != nil {
					val = *c.Position
				}
			case "notes":
				if c.Notes != nil {
					val = *c.Notes
				}
			}
			row = append(row, csvutil.NeutralizeFormulaCell(val))
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

	companyNames, err := s.resolveCompanyNames(ctx, contacts, []string{"company"})
	if err != nil {
		return nil, fmt.Errorf("failed to resolve company names: %w", err)
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

		// Organization
		if c.CompanyID != nil {
			if name := companyNames[*c.CompanyID]; name != "" {
				card.SetValue(vcard.FieldOrganization, name)
			}
		}

		if err := enc.Encode(card); err != nil {
			return nil, fmt.Errorf("failed to encode vCard: %w", err)
		}
	}

	s.logger.Info("vCard export complete", "contacts", len(contacts))

	return buf.Bytes(), nil
}

// resolveCompanyNames batches company_id -> name lookups for the given contacts in a
// single call instead of one per contact. Returns nil if "company" isn't requested.
func (s *ExportService) resolveCompanyNames(ctx context.Context, contacts []*models.Contact, fields []string) (map[uuid.UUID]string, error) {
	if !slices.Contains(fields, "company") {
		return nil, nil
	}

	seen := make(map[uuid.UUID]bool)
	var ids []uuid.UUID
	for _, c := range contacts {
		if c.CompanyID != nil && !seen[*c.CompanyID] {
			seen[*c.CompanyID] = true
			ids = append(ids, *c.CompanyID)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}

	return s.contactProvider.GetCompanyNames(ctx, ids)
}
