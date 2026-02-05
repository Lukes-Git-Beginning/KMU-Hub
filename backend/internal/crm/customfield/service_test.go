package customfield

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// mockRepository implements Repository for testing
type mockRepository struct {
	fields        map[uuid.UUID]*models.CustomFieldDefinition
	fieldsByKey   map[string]*models.CustomFieldDefinition // key = entityType:fieldName
	shouldFailGet bool
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		fields:      make(map[uuid.UUID]*models.CustomFieldDefinition),
		fieldsByKey: make(map[string]*models.CustomFieldDefinition),
	}
}

func (m *mockRepository) Create(_ context.Context, field *models.CustomFieldDefinition) error {
	m.fields[field.ID] = field
	key := string(field.EntityType) + ":" + field.FieldName
	m.fieldsByKey[key] = field
	return nil
}

func (m *mockRepository) GetByID(_ context.Context, id uuid.UUID) (*models.CustomFieldDefinition, error) {
	if m.shouldFailGet {
		return nil, ErrFieldNotFound
	}
	f, ok := m.fields[id]
	if !ok {
		return nil, ErrFieldNotFound
	}
	return f, nil
}

func (m *mockRepository) GetByEntityAndName(_ context.Context, entityType models.EntityType, fieldName string) (*models.CustomFieldDefinition, error) {
	key := string(entityType) + ":" + fieldName
	f, ok := m.fieldsByKey[key]
	if !ok {
		return nil, ErrFieldNotFound
	}
	return f, nil
}

func (m *mockRepository) List(_ context.Context, entityType *models.EntityType, offset, limit int) ([]*models.CustomFieldDefinition, int, error) {
	var result []*models.CustomFieldDefinition
	for _, f := range m.fields {
		if entityType == nil || f.EntityType == *entityType {
			result = append(result, f)
		}
	}

	total := len(result)
	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return result[offset:end], total, nil
}

func (m *mockRepository) Update(_ context.Context, field *models.CustomFieldDefinition) error {
	m.fields[field.ID] = field
	key := string(field.EntityType) + ":" + field.FieldName
	m.fieldsByKey[key] = field
	return nil
}

func (m *mockRepository) Delete(_ context.Context, id uuid.UUID) error {
	f, ok := m.fields[id]
	if !ok {
		return ErrFieldNotFound
	}
	key := string(f.EntityType) + ":" + f.FieldName
	delete(m.fieldsByKey, key)
	delete(m.fields, id)
	return nil
}

func newTestService() (*Service, *mockRepository) {
	repo := newMockRepository()
	svc := NewService(repo)
	return svc, repo
}

func createTestField(repo *mockRepository, entityType models.EntityType, fieldName string, fieldType models.FieldType) *models.CustomFieldDefinition {
	field := &models.CustomFieldDefinition{
		ID:         uuid.New(),
		EntityType: entityType,
		FieldName:  fieldName,
		FieldLabel: fieldName + " Label",
		FieldType:  fieldType,
		IsRequired: false,
		SortOrder:  0,
		CreatedBy:  uuid.New(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	repo.fields[field.ID] = field
	key := string(entityType) + ":" + fieldName
	repo.fieldsByKey[key] = field
	return field
}

func TestService_Create(t *testing.T) {
	tests := []struct {
		name    string
		input   CreateInput
		setup   func(*mockRepository)
		wantErr error
	}{
		{
			name: "success_text_field",
			input: CreateInput{
				EntityType: models.EntityTypeContact,
				FieldName:  "website",
				FieldLabel: "Website URL",
				FieldType:  models.FieldTypeText,
				CreatedBy:  uuid.New(),
			},
			setup: func(r *mockRepository) {},
		},
		{
			name: "success_select_field_with_options",
			input: CreateInput{
				EntityType: models.EntityTypeContact,
				FieldName:  "priority",
				FieldLabel: "Priority Level",
				FieldType:  models.FieldTypeSelect,
				Options:    []string{"low", "medium", "high"},
				CreatedBy:  uuid.New(),
			},
			setup: func(r *mockRepository) {},
		},
		{
			name: "success_multiselect_field",
			input: CreateInput{
				EntityType: models.EntityTypeDeal,
				FieldName:  "tags",
				FieldLabel: "Tags",
				FieldType:  models.FieldTypeMultiSelect,
				Options:    []string{"hot", "cold", "warm"},
				CreatedBy:  uuid.New(),
			},
			setup: func(r *mockRepository) {},
		},
		{
			name: "success_required_field",
			input: CreateInput{
				EntityType: models.EntityTypeCompany,
				FieldName:  "industry",
				FieldLabel: "Industry",
				FieldType:  models.FieldTypeText,
				IsRequired: true,
				CreatedBy:  uuid.New(),
			},
			setup: func(r *mockRepository) {},
		},
		{
			name: "invalid_entity_type",
			input: CreateInput{
				EntityType: "invalid",
				FieldName:  "test",
				FieldLabel: "Test",
				FieldType:  models.FieldTypeText,
				CreatedBy:  uuid.New(),
			},
			setup:   func(r *mockRepository) {},
			wantErr: ErrInvalidEntityType,
		},
		{
			name: "invalid_field_type",
			input: CreateInput{
				EntityType: models.EntityTypeContact,
				FieldName:  "test",
				FieldLabel: "Test",
				FieldType:  "invalid",
				CreatedBy:  uuid.New(),
			},
			setup:   func(r *mockRepository) {},
			wantErr: ErrInvalidFieldType,
		},
		{
			name: "missing_field_name",
			input: CreateInput{
				EntityType: models.EntityTypeContact,
				FieldName:  "",
				FieldLabel: "Test",
				FieldType:  models.FieldTypeText,
				CreatedBy:  uuid.New(),
			},
			setup:   func(r *mockRepository) {},
			wantErr: ErrFieldNameRequired,
		},
		{
			name: "missing_field_label",
			input: CreateInput{
				EntityType: models.EntityTypeContact,
				FieldName:  "test",
				FieldLabel: "",
				FieldType:  models.FieldTypeText,
				CreatedBy:  uuid.New(),
			},
			setup:   func(r *mockRepository) {},
			wantErr: ErrFieldLabelRequired,
		},
		{
			name: "select_without_options",
			input: CreateInput{
				EntityType: models.EntityTypeContact,
				FieldName:  "priority",
				FieldLabel: "Priority",
				FieldType:  models.FieldTypeSelect,
				Options:    nil,
				CreatedBy:  uuid.New(),
			},
			setup:   func(r *mockRepository) {},
			wantErr: ErrOptionsRequired,
		},
		{
			name: "multiselect_without_options",
			input: CreateInput{
				EntityType: models.EntityTypeContact,
				FieldName:  "tags",
				FieldLabel: "Tags",
				FieldType:  models.FieldTypeMultiSelect,
				Options:    []string{},
				CreatedBy:  uuid.New(),
			},
			setup:   func(r *mockRepository) {},
			wantErr: ErrOptionsRequired,
		},
		{
			name: "duplicate_field",
			input: CreateInput{
				EntityType: models.EntityTypeContact,
				FieldName:  "existing_field",
				FieldLabel: "Existing",
				FieldType:  models.FieldTypeText,
				CreatedBy:  uuid.New(),
			},
			setup: func(r *mockRepository) {
				createTestField(r, models.EntityTypeContact, "existing_field", models.FieldTypeText)
			},
			wantErr: ErrFieldExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newTestService()
			tt.setup(repo)

			field, err := svc.Create(context.Background(), tt.input)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, field)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, field)
				assert.Equal(t, tt.input.EntityType, field.EntityType)
				assert.Equal(t, tt.input.FieldName, field.FieldName)
				assert.Equal(t, tt.input.FieldLabel, field.FieldLabel)
				assert.Equal(t, tt.input.FieldType, field.FieldType)
				assert.Equal(t, tt.input.IsRequired, field.IsRequired)
				assert.NotEqual(t, uuid.Nil, field.ID)
			}
		})
	}
}

func TestService_GetByID(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mockRepository) uuid.UUID
		wantErr error
	}{
		{
			name: "found",
			setup: func(r *mockRepository) uuid.UUID {
				field := createTestField(r, models.EntityTypeContact, "website", models.FieldTypeText)
				return field.ID
			},
		},
		{
			name: "not_found",
			setup: func(r *mockRepository) uuid.UUID {
				return uuid.New()
			},
			wantErr: ErrFieldNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newTestService()
			id := tt.setup(repo)

			field, err := svc.GetByID(context.Background(), id)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, field)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, field)
				assert.Equal(t, id, field.ID)
			}
		})
	}
}

func TestService_List(t *testing.T) {
	svc, repo := newTestService()

	// Create test fields
	createTestField(repo, models.EntityTypeContact, "field1", models.FieldTypeText)
	createTestField(repo, models.EntityTypeContact, "field2", models.FieldTypeNumber)
	createTestField(repo, models.EntityTypeCompany, "field3", models.FieldTypeText)
	createTestField(repo, models.EntityTypeDeal, "field4", models.FieldTypeDate)

	t.Run("list_all", func(t *testing.T) {
		fields, total, err := svc.List(context.Background(), ListInput{
			Page:     1,
			PageSize: 10,
		})
		require.NoError(t, err)
		assert.Equal(t, 4, total)
		assert.Len(t, fields, 4)
	})

	t.Run("filter_by_entity_type", func(t *testing.T) {
		entityType := models.EntityTypeContact
		fields, total, err := svc.List(context.Background(), ListInput{
			EntityType: &entityType,
			Page:       1,
			PageSize:   10,
		})
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, fields, 2)
		for _, f := range fields {
			assert.Equal(t, models.EntityTypeContact, f.EntityType)
		}
	})

	t.Run("pagination", func(t *testing.T) {
		fields, total, err := svc.List(context.Background(), ListInput{
			Page:     1,
			PageSize: 2,
		})
		require.NoError(t, err)
		assert.Equal(t, 4, total)
		assert.Len(t, fields, 2)

		fields2, total2, err := svc.List(context.Background(), ListInput{
			Page:     2,
			PageSize: 2,
		})
		require.NoError(t, err)
		assert.Equal(t, 4, total2)
		assert.Len(t, fields2, 2)
	})

	t.Run("default_page_values", func(t *testing.T) {
		fields, total, err := svc.List(context.Background(), ListInput{
			Page:     0, // Should default to 1
			PageSize: 0, // Should default to 20
		})
		require.NoError(t, err)
		assert.Equal(t, 4, total)
		assert.Len(t, fields, 4)
	})

	t.Run("page_size_capped", func(t *testing.T) {
		_, _, err := svc.List(context.Background(), ListInput{
			Page:     1,
			PageSize: 500, // Should be capped at 100
		})
		require.NoError(t, err)
	})
}

func TestService_Update(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mockRepository) uuid.UUID
		input   UpdateInput
		wantErr error
	}{
		{
			name: "update_label",
			setup: func(r *mockRepository) uuid.UUID {
				field := createTestField(r, models.EntityTypeContact, "website", models.FieldTypeText)
				return field.ID
			},
			input: UpdateInput{
				FieldLabel: strPtr("Updated Label"),
			},
		},
		{
			name: "update_options",
			setup: func(r *mockRepository) uuid.UUID {
				field := createTestField(r, models.EntityTypeContact, "priority", models.FieldTypeSelect)
				field.Options = []string{"low", "high"}
				r.fields[field.ID] = field
				return field.ID
			},
			input: UpdateInput{
				Options: []string{"low", "medium", "high", "urgent"},
			},
		},
		{
			name: "update_required",
			setup: func(r *mockRepository) uuid.UUID {
				field := createTestField(r, models.EntityTypeContact, "website", models.FieldTypeText)
				return field.ID
			},
			input: UpdateInput{
				IsRequired: boolPtr(true),
			},
		},
		{
			name: "update_sort_order",
			setup: func(r *mockRepository) uuid.UUID {
				field := createTestField(r, models.EntityTypeContact, "website", models.FieldTypeText)
				return field.ID
			},
			input: UpdateInput{
				SortOrder: intPtr(5),
			},
		},
		{
			name: "field_not_found",
			setup: func(r *mockRepository) uuid.UUID {
				return uuid.New()
			},
			input: UpdateInput{
				FieldLabel: strPtr("Test"),
			},
			wantErr: ErrFieldNotFound,
		},
		{
			name: "empty_label",
			setup: func(r *mockRepository) uuid.UUID {
				field := createTestField(r, models.EntityTypeContact, "website", models.FieldTypeText)
				return field.ID
			},
			input: UpdateInput{
				FieldLabel: strPtr(""),
			},
			wantErr: ErrFieldLabelRequired,
		},
		{
			name: "remove_options_from_select",
			setup: func(r *mockRepository) uuid.UUID {
				field := createTestField(r, models.EntityTypeContact, "priority", models.FieldTypeSelect)
				field.Options = []string{"low", "high"}
				r.fields[field.ID] = field
				return field.ID
			},
			input: UpdateInput{
				Options: []string{},
			},
			wantErr: ErrOptionsRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newTestService()
			id := tt.setup(repo)

			field, err := svc.Update(context.Background(), id, tt.input)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, field)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, field)
				if tt.input.FieldLabel != nil {
					assert.Equal(t, *tt.input.FieldLabel, field.FieldLabel)
				}
				if tt.input.Options != nil {
					assert.Equal(t, tt.input.Options, field.Options)
				}
				if tt.input.IsRequired != nil {
					assert.Equal(t, *tt.input.IsRequired, field.IsRequired)
				}
				if tt.input.SortOrder != nil {
					assert.Equal(t, *tt.input.SortOrder, field.SortOrder)
				}
			}
		})
	}
}

func TestService_Delete(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mockRepository) uuid.UUID
		wantErr error
	}{
		{
			name: "success",
			setup: func(r *mockRepository) uuid.UUID {
				field := createTestField(r, models.EntityTypeContact, "website", models.FieldTypeText)
				return field.ID
			},
		},
		{
			name: "not_found",
			setup: func(r *mockRepository) uuid.UUID {
				return uuid.New()
			},
			wantErr: ErrFieldNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newTestService()
			id := tt.setup(repo)

			err := svc.Delete(context.Background(), id)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				// Verify deletion
				_, exists := repo.fields[id]
				assert.False(t, exists)
			}
		})
	}
}

func TestFieldType_Validation(t *testing.T) {
	tests := []struct {
		fieldType models.FieldType
		valid     bool
	}{
		{models.FieldTypeText, true},
		{models.FieldTypeNumber, true},
		{models.FieldTypeDate, true},
		{models.FieldTypeBoolean, true},
		{models.FieldTypeSelect, true},
		{models.FieldTypeMultiSelect, true},
		{models.FieldType("invalid"), false},
		{models.FieldType(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.fieldType), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.fieldType.IsValid())
		})
	}
}

func TestEntityType_Validation(t *testing.T) {
	tests := []struct {
		entityType models.EntityType
		valid      bool
	}{
		{models.EntityTypeContact, true},
		{models.EntityTypeCompany, true},
		{models.EntityTypeDeal, true},
		{models.EntityTypeActivity, true},
		{models.EntityType("invalid"), false},
		{models.EntityType(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.entityType), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.entityType.IsValid())
		})
	}
}

func TestFieldType_RequiresOptions(t *testing.T) {
	tests := []struct {
		fieldType       models.FieldType
		requiresOptions bool
	}{
		{models.FieldTypeText, false},
		{models.FieldTypeNumber, false},
		{models.FieldTypeDate, false},
		{models.FieldTypeBoolean, false},
		{models.FieldTypeSelect, true},
		{models.FieldTypeMultiSelect, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.fieldType), func(t *testing.T) {
			assert.Equal(t, tt.requiresOptions, tt.fieldType.RequiresOptions())
		})
	}
}

// Helper functions
func strPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func intPtr(i int) *int {
	return &i
}
