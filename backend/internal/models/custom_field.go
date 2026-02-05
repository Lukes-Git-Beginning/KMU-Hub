package models

import (
	"time"

	"github.com/google/uuid"
)

// FieldType represents the type of a custom field
type FieldType string

const (
	FieldTypeText        FieldType = "text"
	FieldTypeNumber      FieldType = "number"
	FieldTypeDate        FieldType = "date"
	FieldTypeBoolean     FieldType = "boolean"
	FieldTypeSelect      FieldType = "select"
	FieldTypeMultiSelect FieldType = "multiselect"
)

// ValidFieldTypes contains all valid field types
var ValidFieldTypes = map[FieldType]bool{
	FieldTypeText:        true,
	FieldTypeNumber:      true,
	FieldTypeDate:        true,
	FieldTypeBoolean:     true,
	FieldTypeSelect:      true,
	FieldTypeMultiSelect: true,
}

// EntityType represents the type of CRM entity a custom field belongs to
type EntityType string

const (
	EntityTypeContact  EntityType = "contact"
	EntityTypeCompany  EntityType = "company"
	EntityTypeDeal     EntityType = "deal"
	EntityTypeActivity EntityType = "activity"
)

// ValidEntityTypes contains all valid entity types
var ValidEntityTypes = map[EntityType]bool{
	EntityTypeContact:  true,
	EntityTypeCompany:  true,
	EntityTypeDeal:     true,
	EntityTypeActivity: true,
}

// CustomFieldDefinition represents the schema definition for a custom field
type CustomFieldDefinition struct {
	ID           uuid.UUID  `json:"id"`
	EntityType   EntityType `json:"entity_type"`
	FieldName    string     `json:"field_name"`
	FieldLabel   string     `json:"field_label"`
	FieldType    FieldType  `json:"field_type"`
	Options      []string   `json:"options,omitempty"`       // For select/multiselect
	DefaultValue *string    `json:"default_value,omitempty"` // Optional default
	IsRequired   bool       `json:"is_required"`
	SortOrder    int        `json:"sort_order"`
	CreatedBy    uuid.UUID  `json:"created_by"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// CustomFieldValue represents the actual value of a custom field for an entity
type CustomFieldValue struct {
	FieldID uuid.UUID `json:"field_id"`
	Value   any       `json:"value"` // Type depends on field_type
}

// IsValid checks if the field type is valid
func (ft FieldType) IsValid() bool {
	return ValidFieldTypes[ft]
}

// IsValid checks if the entity type is valid
func (et EntityType) IsValid() bool {
	return ValidEntityTypes[et]
}

// RequiresOptions returns true if the field type requires options
func (ft FieldType) RequiresOptions() bool {
	return ft == FieldTypeSelect || ft == FieldTypeMultiSelect
}
