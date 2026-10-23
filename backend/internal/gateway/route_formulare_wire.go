package gateway

// Wire shape of the Formulare schema and submission routes as FormularePage reads them
// (formulare-types.ts): lists as `{items,total}`, single entities flat, camelCase, `fields` and
// `answers` as JSON instead of base64, status as a word. The gRPC service keeps its proto types;
// the conversion sits at this edge only.

import (
	"encoding/json"
	"time"

	formularev1 "github.com/kmuhub/kmuhub/proto/formulare/v1"
)

type formSchemaWire struct {
	ID              string          `json:"id"`
	TenantID        string          `json:"tenantId"`
	Title           string          `json:"title"`
	Description     string          `json:"description"`
	Fields          json.RawMessage `json:"fields"`
	Status          string          `json:"status"`
	IsTemplate      bool            `json:"isTemplate"`
	IsPublic        bool            `json:"isPublic"`
	PageCount       int32           `json:"pageCount"`
	IntakeTargetID  *string         `json:"intakeTargetId,omitempty"`
	SubmissionCount int32           `json:"submissionCount"`
	CreatedBy       *string         `json:"createdBy"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
	DeletedAt       *time.Time      `json:"deletedAt"`
}

// formSchemaStatusWord maps the proto status to the FE word. The FE also knows `closed`; the
// proto has no such value, so a schema is never reported as closed.
func formSchemaStatusWord(s formularev1.FormSchemaStatus) string {
	switch s {
	case formularev1.FormSchemaStatus_FORM_SCHEMA_STATUS_ACTIVE:
		return "active"
	case formularev1.FormSchemaStatus_FORM_SCHEMA_STATUS_ARCHIVED:
		return "archived"
	default:
		return "draft"
	}
}

func formSchemaStatusFromWord(word string) formularev1.FormSchemaStatus {
	switch word {
	case "active":
		return formularev1.FormSchemaStatus_FORM_SCHEMA_STATUS_ACTIVE
	case "archived":
		return formularev1.FormSchemaStatus_FORM_SCHEMA_STATUS_ARCHIVED
	default:
		return formularev1.FormSchemaStatus_FORM_SCHEMA_STATUS_DRAFT
	}
}

// jsonOrDefault never lets an empty blob become a JSON `null`: the editor iterates `fields`
// and reads keys of `answers`.
func jsonOrDefault(b []byte, def string) json.RawMessage {
	if len(b) == 0 {
		return json.RawMessage(def)
	}
	return json.RawMessage(b)
}

func nonEmptyPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func toFormSchemaWire(s *formularev1.FormSchema) formSchemaWire {
	wire := formSchemaWire{
		ID:              s.GetId(),
		TenantID:        s.GetTenantId(),
		Title:           s.GetTitle(),
		Description:     s.GetDescription(),
		Fields:          jsonOrDefault(s.GetFields(), "[]"),
		Status:          formSchemaStatusWord(s.GetStatus()),
		IsTemplate:      s.GetIsTemplate(),
		IsPublic:        s.GetIsPublic(),
		PageCount:       s.GetPageCount(),
		IntakeTargetID:  s.IntakeTargetId,
		SubmissionCount: s.GetSubmissionCount(),
		CreatedBy:       nonEmptyPtr(s.GetCreatedBy()),
		CreatedAt:       s.GetCreatedAt().AsTime(),
		UpdatedAt:       s.GetUpdatedAt().AsTime(),
	}
	if s.DeletedAt != nil {
		deleted := s.GetDeletedAt().AsTime()
		wire.DeletedAt = &deleted
	}
	return wire
}

type formSchemaListWire struct {
	Items []formSchemaWire `json:"items"`
	Total int32            `json:"total"`
}

func toFormSchemaListWire(resp *formularev1.ListFormSchemasResponse) formSchemaListWire {
	items := make([]formSchemaWire, 0, len(resp.GetSchemas()))
	for _, s := range resp.GetSchemas() {
		items = append(items, toFormSchemaWire(s))
	}
	return formSchemaListWire{Items: items, Total: resp.GetTotal()}
}

type formSubmissionWire struct {
	ID           string          `json:"id"`
	FormSchemaID *string         `json:"formSchemaId"`
	TenantID     string          `json:"tenantId"`
	Answers      json.RawMessage `json:"answers"`
	Status       string          `json:"status"`
	SubmittedBy  *string         `json:"submittedBy"`
	IPAddress    *string         `json:"ipAddress"`
	SubmittedAt  time.Time       `json:"submittedAt"`
}

func formSubmissionStatusWord(s formularev1.FormSubmissionStatus) string {
	switch s {
	case formularev1.FormSubmissionStatus_FORM_SUBMISSION_STATUS_READ:
		return "read"
	case formularev1.FormSubmissionStatus_FORM_SUBMISSION_STATUS_ARCHIVED:
		return "archived"
	default:
		return "new"
	}
}

func toFormSubmissionWire(s *formularev1.FormSubmission) formSubmissionWire {
	return formSubmissionWire{
		ID:           s.GetId(),
		FormSchemaID: nonEmptyPtr(s.GetFormSchemaId()),
		TenantID:     s.GetTenantId(),
		Answers:      jsonOrDefault(s.GetAnswers(), "{}"),
		Status:       formSubmissionStatusWord(s.GetStatus()),
		SubmittedBy:  nonEmptyPtr(s.GetSubmittedBy()),
		IPAddress:    nonEmptyPtr(s.GetIpAddress()),
		SubmittedAt:  s.GetSubmittedAt().AsTime(),
	}
}

type formSubmissionListWire struct {
	Items []formSubmissionWire `json:"items"`
	Total int32                `json:"total"`
}

func toFormSubmissionListWire(resp *formularev1.ListSubmissionsResponse) formSubmissionListWire {
	items := make([]formSubmissionWire, 0, len(resp.GetSubmissions()))
	for _, s := range resp.GetSubmissions() {
		items = append(items, toFormSubmissionWire(s))
	}
	return formSubmissionListWire{Items: items, Total: resp.GetTotal()}
}

// jsonBlob turns a request value for a `bytes` proto field into the bytes the service stores.
// The FE sends `fields`/`answers` as JSON; a JSON string is still read as the base64 the old
// body shape used.
func jsonBlob(raw json.RawMessage) []byte {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	if raw[0] == '"' {
		var decoded []byte
		if err := json.Unmarshal(raw, &decoded); err == nil {
			return decoded
		}
	}
	return raw
}

// firstSet picks the camelCase value the FE sends or, failing that, the snake_case one.
func firstSet[T any](camel, snake *T) *T {
	if camel != nil {
		return camel
	}
	return snake
}
