package contact

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"github.com/emersion/go-vcard"
	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ContactProvider abstracts the CRM contact service for import/export use.
type ContactProvider interface {
	GetByEmail(ctx context.Context, email string) (*models.Contact, error)
	CreateForImport(ctx context.Context, c *models.Contact) error
	UpdateForImport(ctx context.Context, c *models.Contact) error
	ListByIDs(ctx context.Context, ids []uuid.UUID) ([]*models.Contact, error)
	ListVisible(ctx context.Context, userID uuid.UUID, isAdmin bool) ([]*models.Contact, error)

	// FindOrCreateCompany resolves a company name to the company-relation ID, creating a
	// bare company (name only) if none exists yet. createdBy is the importing user.
	FindOrCreateCompany(ctx context.Context, name string, createdBy uuid.UUID) (uuid.UUID, error)
	// GetCompanyNames resolves company IDs to names in a single batch, for export.
	GetCompanyNames(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error)
}

// ImportService handles importing contacts from CSV and vCard files.
type ImportService struct {
	contactProvider ContactProvider
	logger          *slog.Logger
}

// NewImportService creates a new ImportService.
func NewImportService(cp ContactProvider, logger *slog.Logger) *ImportService {
	if logger == nil {
		logger = slog.Default()
	}
	return &ImportService{
		contactProvider: cp,
		logger:          logger,
	}
}

// ImportResult holds the results of an import operation.
type ImportResult struct {
	ImportedCount int           `json:"imported_count"`
	MergedCount   int           `json:"merged_count"`
	SkippedCount  int           `json:"skipped_count"`
	Errors        []ImportError `json:"errors,omitempty"`
}

// ImportError describes an error encountered during import of a specific row.
type ImportError struct {
	Row    int    `json:"row"`
	Reason string `json:"reason"`
}

// ImportPreview is returned from PreviewCSV for the field-mapping wizard UI.
type ImportPreview struct {
	Columns         []string            `json:"columns"`
	SampleRows      [][]string          `json:"sample_rows"`
	DetectedMapping map[string]string   `json:"detected_mapping"`
}

// knownMappings maps common CSV column header names to CRM field names.
// Supports German (DACH) and English column names.
var knownMappings = map[string]string{
	"email": "email", "e-mail": "email", "mail": "email",
	"vorname": "first_name", "first name": "first_name", "firstname": "first_name", "first_name": "first_name",
	"nachname": "last_name", "last name": "last_name", "lastname": "last_name", "name": "last_name", "last_name": "last_name",
	"firma": "company", "company": "company", "unternehmen": "company", "organisation": "company",
	"telefon": "phone", "phone": "phone", "tel": "phone", "mobil": "phone",
	"position": "position", "titel": "position", "title": "position", "job title": "position",
	"notizen": "notes", "notes": "notes", "bemerkung": "notes",
}

// PreviewCSV reads the first rows of a CSV file and auto-detects field mapping.
func (s *ImportService) PreviewCSV(ctx context.Context, reader io.Reader) (*ImportPreview, error) {
	csvReader, err := newAutoCSVReader(reader)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidCSV, err)
	}

	// Read header row
	header, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("%w: cannot read header", ErrInvalidCSV)
	}

	// Auto-detect field mapping
	detectedMapping := make(map[string]string)
	for _, col := range header {
		lower := strings.ToLower(strings.TrimSpace(col))
		if field, ok := knownMappings[lower]; ok {
			detectedMapping[col] = field
		}
	}

	// Read up to 5 sample rows
	var sampleRows [][]string
	for i := 0; i < 5; i++ {
		row, readErr := csvReader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			break
		}
		sampleRows = append(sampleRows, row)
	}

	return &ImportPreview{
		Columns:         header,
		SampleRows:      sampleRows,
		DetectedMapping: detectedMapping,
	}, nil
}

// ImportCSV imports contacts from a CSV file using the provided field mapping.
func (s *ImportService) ImportCSV(
	ctx context.Context,
	reader io.Reader,
	fieldMapping map[string]string,
	visibility string,
	ownerID uuid.UUID,
	mergeByEmail bool,
) (*ImportResult, error) {
	csvReader, err := newAutoCSVReader(reader)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidCSV, err)
	}

	// Read header row
	header, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("%w: cannot read header", ErrInvalidCSV)
	}

	// Build column index from header
	colIndex := make(map[string]int)
	for i, col := range header {
		colIndex[col] = i
	}

	result := &ImportResult{}
	rowNum := 1 // 1-based (header is row 0)
	companyCache := make(map[string]uuid.UUID)

	for {
		row, readErr := csvReader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			result.Errors = append(result.Errors, ImportError{Row: rowNum, Reason: readErr.Error()})
			result.SkippedCount++
			rowNum++
			continue
		}
		rowNum++

		// Extract fields using mapping
		fields := extractFieldsFromRow(row, header, fieldMapping, colIndex)

		importErr := s.importSingleContact(ctx, fields, visibility, ownerID, mergeByEmail, result, companyCache)
		if importErr != nil {
			result.Errors = append(result.Errors, ImportError{Row: rowNum, Reason: importErr.Error()})
			result.SkippedCount++
		}
	}

	s.logger.Info("CSV import complete",
		"imported", result.ImportedCount,
		"merged", result.MergedCount,
		"skipped", result.SkippedCount,
	)

	return result, nil
}

// ImportVCard imports contacts from a vCard file.
func (s *ImportService) ImportVCard(
	ctx context.Context,
	reader io.Reader,
	visibility string,
	ownerID uuid.UUID,
	mergeByEmail bool,
) (*ImportResult, error) {
	dec := vcard.NewDecoder(reader)
	result := &ImportResult{}
	rowNum := 0
	companyCache := make(map[string]uuid.UUID)

	for {
		card, err := dec.Decode()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Errors = append(result.Errors, ImportError{Row: rowNum, Reason: err.Error()})
			result.SkippedCount++
			rowNum++
			continue
		}
		rowNum++

		fields := extractFieldsFromVCard(card)

		importErr := s.importSingleContact(ctx, fields, visibility, ownerID, mergeByEmail, result, companyCache)
		if importErr != nil {
			result.Errors = append(result.Errors, ImportError{Row: rowNum, Reason: importErr.Error()})
			result.SkippedCount++
		}
	}

	s.logger.Info("vCard import complete",
		"imported", result.ImportedCount,
		"merged", result.MergedCount,
		"skipped", result.SkippedCount,
	)

	return result, nil
}

// importSingleContact creates or merges a single contact.
func (s *ImportService) importSingleContact(
	ctx context.Context,
	fields map[string]string,
	visibility string,
	ownerID uuid.UUID,
	mergeByEmail bool,
	result *ImportResult,
	companyCache map[string]uuid.UUID,
) error {
	emailVal := strings.TrimSpace(fields["email"])

	// Validate email if present
	if emailVal != "" {
		if _, err := mail.ParseAddress(emailVal); err != nil {
			return fmt.Errorf("invalid email: %s", emailVal)
		}
	}

	firstName := strings.TrimSpace(fields["first_name"])
	lastName := strings.TrimSpace(fields["last_name"])
	if firstName == "" && lastName == "" {
		return fmt.Errorf("both first_name and last_name are empty")
	}

	// Try merge by email if enabled
	if mergeByEmail && emailVal != "" {
		existing, err := s.contactProvider.GetByEmail(ctx, emailVal)
		if err == nil && existing != nil {
			// Merge: only update empty fields on existing contact
			changed := false
			if existing.FirstName == "" && firstName != "" {
				existing.FirstName = firstName
				changed = true
			}
			if existing.LastName == "" && lastName != "" {
				existing.LastName = lastName
				changed = true
			}
			phone := strings.TrimSpace(fields["phone"])
			if existing.Phone == nil && phone != "" {
				existing.Phone = &phone
				changed = true
			}
			position := strings.TrimSpace(fields["position"])
			if existing.Position == nil && position != "" {
				existing.Position = &position
				changed = true
			}
			notes := strings.TrimSpace(fields["notes"])
			if existing.Notes == nil && notes != "" {
				existing.Notes = &notes
				changed = true
			}
			if existing.CompanyID == nil {
				companyID, companyErr := s.resolveCompany(ctx, fields["company"], ownerID, companyCache)
				if companyErr != nil {
					return companyErr
				}
				if companyID != nil {
					existing.CompanyID = companyID
					changed = true
				}
			}

			if changed {
				existing.UpdatedAt = time.Now()
				if updateErr := s.contactProvider.UpdateForImport(ctx, existing); updateErr != nil {
					return updateErr
				}
			}
			result.MergedCount++
			return nil
		}
	}

	// Create new contact
	contact := &models.Contact{
		ID:         uuid.New(),
		FirstName:  firstName,
		LastName:   lastName,
		Visibility: visibility,
		OwnerID:    &ownerID,
		CreatedBy:  ownerID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if emailVal != "" {
		contact.Email = &emailVal
	}
	phone := strings.TrimSpace(fields["phone"])
	if phone != "" {
		contact.Phone = &phone
	}
	position := strings.TrimSpace(fields["position"])
	if position != "" {
		contact.Position = &position
	}
	notes := strings.TrimSpace(fields["notes"])
	if notes != "" {
		contact.Notes = &notes
	}
	companyID, companyErr := s.resolveCompany(ctx, fields["company"], ownerID, companyCache)
	if companyErr != nil {
		return companyErr
	}
	contact.CompanyID = companyID

	if createErr := s.contactProvider.CreateForImport(ctx, contact); createErr != nil {
		return createErr
	}
	result.ImportedCount++
	return nil
}

// resolveCompany maps a company name to the company-relation ID, reusing companyCache
// within one import run so a file with many rows for the same company issues exactly one
// find-or-create call instead of one per row. Returns nil, nil if name is blank.
func (s *ImportService) resolveCompany(ctx context.Context, name string, ownerID uuid.UUID, cache map[string]uuid.UUID) (*uuid.UUID, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil
	}

	key := strings.ToLower(name)
	if id, ok := cache[key]; ok {
		return &id, nil
	}

	id, err := s.contactProvider.FindOrCreateCompany(ctx, name, ownerID)
	if err != nil {
		return nil, fmt.Errorf("resolve company %q: %w", name, err)
	}
	cache[key] = id
	return &id, nil
}

// extractFieldsFromRow maps a CSV row to CRM fields using the column mapping.
func extractFieldsFromRow(row, header []string, fieldMapping map[string]string, colIndex map[string]int) map[string]string {
	fields := make(map[string]string)
	for csvCol, crmField := range fieldMapping {
		idx, ok := colIndex[csvCol]
		if !ok || idx >= len(row) {
			continue
		}
		fields[crmField] = row[idx]
	}
	return fields
}

// extractFieldsFromVCard extracts CRM fields from a vCard.
func extractFieldsFromVCard(card vcard.Card) map[string]string {
	fields := make(map[string]string)

	// Full name -> split into first/last
	if fn := card.PreferredValue(vcard.FieldFormattedName); fn != "" {
		parts := strings.SplitN(fn, " ", 2)
		if len(parts) == 2 {
			fields["first_name"] = parts[0]
			fields["last_name"] = parts[1]
		} else {
			fields["last_name"] = fn
		}
	}

	// Structured name takes priority if present
	if names := card.Name(); names != nil {
		if names.GivenName != "" {
			fields["first_name"] = names.GivenName
		}
		if names.FamilyName != "" {
			fields["last_name"] = names.FamilyName
		}
	}

	if email := card.PreferredValue(vcard.FieldEmail); email != "" {
		fields["email"] = email
	}

	if tel := card.PreferredValue(vcard.FieldTelephone); tel != "" {
		fields["phone"] = tel
	}

	if org := card.PreferredValue(vcard.FieldOrganization); org != "" {
		fields["company"] = org
	}

	if title := card.PreferredValue(vcard.FieldTitle); title != "" {
		fields["position"] = title
	}

	if note := card.PreferredValue(vcard.FieldNote); note != "" {
		fields["notes"] = note
	}

	return fields
}

// newAutoCSVReader detects the delimiter (comma, semicolon, tab) and returns a csv.Reader.
func newAutoCSVReader(reader io.Reader) (*csv.Reader, error) {
	br := bufio.NewReader(reader)

	// Peek first line to detect delimiter
	firstLine, err := br.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, err
	}
	if firstLine == "" {
		return nil, fmt.Errorf("empty file")
	}

	// Count delimiter candidates in the first line
	commas := strings.Count(firstLine, ",")
	semicolons := strings.Count(firstLine, ";")
	tabs := strings.Count(firstLine, "\t")

	delimiter := ','
	if semicolons > commas && semicolons > tabs {
		delimiter = ';'
	} else if tabs > commas && tabs > semicolons {
		delimiter = '\t'
	}

	// Create a new reader combining first line + rest
	combined := io.MultiReader(strings.NewReader(firstLine), br)
	csvReader := csv.NewReader(combined)
	csvReader.Comma = delimiter
	csvReader.LazyQuotes = true
	csvReader.TrimLeadingSpace = true
	csvReader.FieldsPerRecord = -1 // Allow variable field count

	return csvReader, nil
}
