package dialer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"

	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

// contactFilterCriteria maps the JSON blob stored in saved_filters.filter_json
// to the fields supported by ListContactsRequest. Unknown fields are ignored
// for forward-compatibility.
type contactFilterCriteria struct {
	CompanyID string   `json:"company_id,omitempty"`
	TagIDs    []string `json:"tag_ids,omitempty"`
	Search    string   `json:"search,omitempty"`
}

// CRMBridge abstracts CRM service interactions for the dialer.
// Phase 2: HubSpotBridge, SalesforceBridge, etc. implement this interface.
type CRMBridge interface {
	// CreateCallActivity logs a completed call as a CRM activity.
	CreateCallActivity(ctx context.Context, input CallActivityInput) error

	// GetContactDetails retrieves display information for a contact.
	GetContactDetails(ctx context.Context, contactID uuid.UUID) (*ContactDetails, error)

	// ResolveFilterContacts expands a saved CRM filter into a flat contact list
	// suitable for campaign import.
	ResolveFilterContacts(ctx context.Context, filterID uuid.UUID) ([]ContactImport, error)
}

// CallActivityInput carries the data needed to create a call activity in the CRM.
type CallActivityInput struct {
	ContactID       uuid.UUID
	AgentID         uuid.UUID
	Subject         string
	Description     string
	DurationSeconds int
	OutcomeLabel    string
	TenantID        uuid.UUID
}

// ContactDetails holds the fields the dialer displays for an active call.
type ContactDetails struct {
	ID      uuid.UUID
	Name    string
	Phone   string
	Company string
	Email   string
}

// ContactImport is a lightweight contact record used when building campaign
// lead lists from a saved CRM filter.
type ContactImport struct {
	ContactID uuid.UUID
	Name      string
	Phone     string
	Company   string
}

// GRPCCRMBridge implements CRMBridge by calling the CRM microservice over gRPC.
type GRPCCRMBridge struct {
	client crmv1.CRMServiceClient
}

// NewGRPCCRMBridge creates a GRPCCRMBridge from an existing gRPC connection.
func NewGRPCCRMBridge(conn *grpc.ClientConn) *GRPCCRMBridge {
	return &GRPCCRMBridge{
		client: crmv1.NewCRMServiceClient(conn),
	}
}

// CreateCallActivity logs the completed call as a CRM activity of type "call".
// The subject is formatted as "Anruf — {outcomeLabel} ({duration})" to give
// sales reps an at-a-glance summary in the activity feed.
func (b *GRPCCRMBridge) CreateCallActivity(ctx context.Context, input CallActivityInput) error {
	subject := fmt.Sprintf("Anruf — %s (%s)", input.OutcomeLabel, FormatDuration(input.DurationSeconds))

	// due_date must be a non-empty string; use today's date in RFC3339 date
	// format so the activity appears in the current-day activity view.
	dueDate := time.Now().UTC().Format("2006-01-02")

	req := &crmv1.CreateActivityRequest{
		ActivityType: "call",
		Subject:      subject,
		Description:  input.Description,
		CreatedBy:    input.AgentID.String(),
		DueDate:      dueDate,
	}

	// contact_id is optional in the proto; only set it when we have a real ID.
	if input.ContactID != uuid.Nil {
		contactIDStr := input.ContactID.String()
		req.ContactId = &contactIDStr
	}

	if _, err := b.client.CreateActivity(ctx, req); err != nil {
		return fmt.Errorf("crm bridge create call activity: %w", err)
	}
	return nil
}

// GetContactDetails fetches contact information from the CRM for display
// during an active dialer call.
func (b *GRPCCRMBridge) GetContactDetails(ctx context.Context, contactID uuid.UUID) (*ContactDetails, error) {
	resp, err := b.client.GetContact(ctx, &crmv1.GetContactRequest{
		Id: contactID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("crm bridge get contact: %w", err)
	}
	if resp.Contact == nil {
		return nil, fmt.Errorf("crm bridge get contact: empty response")
	}

	c := resp.Contact
	name := strings.TrimSpace(c.FirstName + " " + c.LastName)

	company := ""
	if c.CompanyName != nil {
		company = *c.CompanyName
	}

	id, err := uuid.Parse(c.Id)
	if err != nil {
		return nil, fmt.Errorf("crm bridge parse contact id: %w", err)
	}

	return &ContactDetails{
		ID:      id,
		Name:    name,
		Phone:   c.Phone,
		Company: company,
		Email:   c.Email,
	}, nil
}

// ResolveFilterContacts fetches all contacts matched by a saved CRM filter.
// It pages through the result set automatically, collecting every contact into
// a flat slice for campaign lead-list import.
func (b *GRPCCRMBridge) ResolveFilterContacts(ctx context.Context, filterID uuid.UUID) ([]ContactImport, error) {
	// First resolve the saved filter to get its JSON criteria.
	filterResp, err := b.client.GetSavedFilter(ctx, &crmv1.GetSavedFilterRequest{
		Id: filterID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("crm bridge get saved filter: %w", err)
	}
	if filterResp.Filter == nil {
		return nil, fmt.Errorf("crm bridge get saved filter: empty response")
	}

	// Parse filter criteria from the saved filter's JSON blob.
	var criteria contactFilterCriteria
	if filterJSON := filterResp.Filter.FilterJson; filterJSON != "" {
		if err := json.Unmarshal([]byte(filterJSON), &criteria); err != nil {
			slog.WarnContext(ctx, "dialer: unmarshal saved filter criteria failed, importing all contacts",
				"filter_id", filterID,
				"error", err,
			)
		}
	}

	const pageSize = 100
	var (
		page    int32 = 1
		results []ContactImport
	)

	for {
		req := &crmv1.ListContactsRequest{
			Page:     page,
			PageSize: pageSize,
			Search:   criteria.Search,
			TagIds:   criteria.TagIDs,
		}
		if criteria.CompanyID != "" {
			req.CompanyId = &criteria.CompanyID
		}

		resp, err := b.client.ListContacts(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("crm bridge list contacts (page %d): %w", page, err)
		}

		for _, c := range resp.Contacts {
			id, err := uuid.Parse(c.Id)
			if err != nil {
				continue // Skip contacts with malformed IDs.
			}
			name := strings.TrimSpace(c.FirstName + " " + c.LastName)
			company := ""
			if c.CompanyName != nil {
				company = *c.CompanyName
			}
			results = append(results, ContactImport{
				ContactID: id,
				Name:      name,
				Phone:     c.Phone,
				Company:   company,
			})
		}

		// Stop when we've fetched the last page.
		if int32(len(resp.Contacts)) < pageSize || int32(len(results)) >= resp.Total {
			break
		}
		page++
	}

	return results, nil
}
