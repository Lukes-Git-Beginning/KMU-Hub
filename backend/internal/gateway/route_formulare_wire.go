package gateway

// Wire shape of the Formulare schema and submission routes as FormularePage reads them
// (formulare-types.ts): lists as `{items,total}`, single entities flat, camelCase, `fields` and
// `answers` as JSON instead of base64, status as a word. The gRPC service keeps its proto types;
// the conversion sits at this edge only.

import (
	"encoding/json"
	"strconv"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

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

// ============================================================================
// Webhooks, deliveries and share links (formulare-types.ts: FormWebhook, WebhookDelivery,
// FormShareLink). The FE reads their lists as bare arrays and single entities flat, camelCase.
// ============================================================================

func timePtr(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime()
	return &t
}

type formWebhookWire struct {
	ID              string     `json:"id"`
	FormSchemaID    string     `json:"formSchemaId"`
	TenantID        string     `json:"tenantId"`
	URL             string     `json:"url"`
	Secret          string     `json:"secret"`
	Events          []string   `json:"events"`
	Active          bool       `json:"active"`
	LastTriggeredAt *time.Time `json:"lastTriggeredAt"`
	LastStatus      *string    `json:"lastStatus"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

func toFormWebhookWire(w *formularev1.FormWebhook) formWebhookWire {
	events := w.GetEvents()
	if events == nil {
		events = []string{}
	}
	return formWebhookWire{
		ID:              w.GetId(),
		FormSchemaID:    w.GetFormSchemaId(),
		TenantID:        w.GetTenantId(),
		URL:             w.GetUrl(),
		Secret:          w.GetSecret(),
		Events:          events,
		Active:          w.GetActive(),
		LastTriggeredAt: timePtr(w.LastTriggeredAt),
		LastStatus:      nonEmptyPtr(w.GetLastStatus()),
		CreatedAt:       w.GetCreatedAt().AsTime(),
		UpdatedAt:       w.GetUpdatedAt().AsTime(),
	}
}

func toFormWebhookListWire(webhooks []*formularev1.FormWebhook) []formWebhookWire {
	out := make([]formWebhookWire, 0, len(webhooks))
	for _, w := range webhooks {
		out = append(out, toFormWebhookWire(w))
	}
	return out
}

type webhookDeliveryWire struct {
	ID               string          `json:"id"`
	WebhookID        string          `json:"webhookId"`
	SubmissionID     string          `json:"submissionId"`
	TenantID         string          `json:"tenantId"`
	Payload          json.RawMessage `json:"payload"`
	Status           string          `json:"status"`
	AttemptCount     int32           `json:"attemptCount"`
	MaxAttempts      int32           `json:"maxAttempts"`
	NextAttemptAt    *time.Time      `json:"nextAttemptAt"`
	LastError        *string         `json:"lastError"`
	LastResponseCode *int32          `json:"lastResponseCode"`
	CreatedAt        time.Time       `json:"createdAt"`
	DeliveredAt      *time.Time      `json:"deliveredAt"`
}

var deliveryStatusWords = map[formularev1.WebhookDeliveryStatus]string{
	formularev1.WebhookDeliveryStatus_WEBHOOK_DELIVERY_STATUS_PENDING:   "pending",
	formularev1.WebhookDeliveryStatus_WEBHOOK_DELIVERY_STATUS_DELIVERED: "delivered",
	formularev1.WebhookDeliveryStatus_WEBHOOK_DELIVERY_STATUS_FAILED:    "failed",
	formularev1.WebhookDeliveryStatus_WEBHOOK_DELIVERY_STATUS_DEAD:      "dead",
}

func webhookDeliveryStatusWord(s formularev1.WebhookDeliveryStatus) string {
	if word, ok := deliveryStatusWords[s]; ok {
		return word
	}
	return "pending"
}

// webhookDeliveryStatusFromQuery reads the `status` filter as the FE word or, as before, the
// proto number. Anything else is no filter.
func webhookDeliveryStatusFromQuery(raw string) (formularev1.WebhookDeliveryStatus, bool) {
	for status, word := range deliveryStatusWords {
		if raw == word {
			return status, true
		}
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return formularev1.WebhookDeliveryStatus(n), true
}

func toWebhookDeliveryWire(d *formularev1.WebhookDelivery) webhookDeliveryWire {
	wire := webhookDeliveryWire{
		ID:            d.GetId(),
		WebhookID:     d.GetWebhookId(),
		SubmissionID:  d.GetSubmissionId(),
		TenantID:      d.GetTenantId(),
		Payload:       jsonOrDefault(d.GetPayload(), "{}"),
		Status:        webhookDeliveryStatusWord(d.GetStatus()),
		AttemptCount:  d.GetAttemptCount(),
		MaxAttempts:   d.GetMaxAttempts(),
		NextAttemptAt: timePtr(d.NextAttemptAt),
		LastError:     nonEmptyPtr(d.GetLastError()),
		CreatedAt:     d.GetCreatedAt().AsTime(),
		DeliveredAt:   timePtr(d.DeliveredAt),
	}
	if code := d.GetLastResponseCode(); code != 0 {
		wire.LastResponseCode = &code
	}
	return wire
}

func toWebhookDeliveryListWire(deliveries []*formularev1.WebhookDelivery) []webhookDeliveryWire {
	out := make([]webhookDeliveryWire, 0, len(deliveries))
	for _, d := range deliveries {
		out = append(out, toWebhookDeliveryWire(d))
	}
	return out
}

// formShareLinkWire: the proto has no channel, url or view counter. They go out as zero values
// rather than invented ones: `channel` is the plain link, `url` is empty until the public
// fill-out page exists, `views` is 0.
type formShareLinkWire struct {
	ID             string     `json:"id"`
	FormSchemaID   string     `json:"formSchemaId"`
	TenantID       string     `json:"tenantId"`
	Token          string     `json:"token"`
	URL            string     `json:"url"`
	Channel        string     `json:"channel"`
	ExpiresAt      *time.Time `json:"expiresAt"`
	MaxSubmissions *int32     `json:"maxSubmissions"`
	Views          int32      `json:"views"`
	Submissions    int32      `json:"submissions"`
	Status         string     `json:"status"`
	CreatedBy      *string    `json:"createdBy"`
	CreatedAt      time.Time  `json:"createdAt"`
}

func toFormShareLinkWire(l *formularev1.FormShareLink) formShareLinkWire {
	wire := formShareLinkWire{
		ID:           l.GetId(),
		FormSchemaID: l.GetFormSchemaId(),
		TenantID:     l.GetTenantId(),
		Token:        l.GetToken(),
		Channel:      "link",
		ExpiresAt:    timePtr(l.ExpiresAt),
		Submissions:  l.GetSubmissionCount(),
		Status:       l.GetStatus(),
		CreatedBy:    nonEmptyPtr(l.GetCreatedBy()),
		CreatedAt:    l.GetCreatedAt().AsTime(),
	}
	if limit := l.GetMaxSubmissions(); limit > 0 {
		wire.MaxSubmissions = &limit
	}
	return wire
}

func toFormShareLinkListWire(links []*formularev1.FormShareLink) []formShareLinkWire {
	out := make([]formShareLinkWire, 0, len(links))
	for _, l := range links {
		out = append(out, toFormShareLinkWire(l))
	}
	return out
}
