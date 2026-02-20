package action

import (
	"context"
	"encoding/json"
	"fmt"

	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

// ============================================================================
// UpdateDealFieldAction
// ============================================================================

// UpdateDealFieldAction updates a field on a CRM deal via gRPC.
// Supports updating assignee (via UpdateDeal) and stage (via MoveDealToStage).
type UpdateDealFieldAction struct {
	client crmv1.CRMServiceClient
}

// NewUpdateDealFieldAction creates a new UpdateDealFieldAction.
func NewUpdateDealFieldAction(client crmv1.CRMServiceClient) *UpdateDealFieldAction {
	return &UpdateDealFieldAction{client: client}
}

func (a *UpdateDealFieldAction) Type() string { return "crm.update_deal_field" }

type updateDealFieldConfig struct {
	DealID string `json:"deal_id"` // template: "{{deal.id}}"
	Field  string `json:"field"`   // "stage", "assignee"
	Value  string `json:"value"`   // template
}

func (a *UpdateDealFieldAction) Execute(ctx context.Context, config json.RawMessage, env map[string]any) (ActionResult, error) {
	if a.client == nil {
		return ActionResult{Success: false, Error: "CRM service unavailable"}, fmt.Errorf("CRM gRPC client is nil")
	}

	var cfg updateDealFieldConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return ActionResult{Success: false, Error: "invalid config: " + err.Error()}, err
	}

	dealID := resolveTemplate(cfg.DealID, env)
	value := resolveTemplate(cfg.Value, env)

	switch cfg.Field {
	case "stage":
		// MoveDealToStage uses stage_id
		resp, err := a.client.MoveDealToStage(ctx, &crmv1.MoveDealToStageRequest{
			DealId:  dealID,
			StageId: value,
		})
		if err != nil {
			return ActionResult{Success: false, Error: err.Error()}, err
		}
		return ActionResult{
			Success: true,
			Output: map[string]any{
				"deal_id":       resp.GetDeal().GetId(),
				"updated_field": "stage",
			},
		}, nil

	case "assignee":
		resp, err := a.client.UpdateDeal(ctx, &crmv1.UpdateDealRequest{
			Id:      dealID,
			OwnerId: &value,
		})
		if err != nil {
			return ActionResult{Success: false, Error: err.Error()}, err
		}
		return ActionResult{
			Success: true,
			Output: map[string]any{
				"deal_id":       resp.GetDeal().GetId(),
				"updated_field": "assignee",
			},
		}, nil

	default:
		return ActionResult{Success: false, Error: "unsupported field: " + cfg.Field}, nil
	}
}

// UpdateDealFieldDefinition returns the action definition for the catalog.
func UpdateDealFieldDefinition() *ActionDefinition {
	return &ActionDefinition{
		Type:        "crm.update_deal_field",
		Module:      "crm",
		Name:        "Deal-Feld aktualisieren",
		Description: "Aktualisiert ein Feld eines CRM-Deals (Phase, Zustaendiger)",
		Params: []ConfigParam{
			{Key: "deal_id", Label: "Deal-ID", Type: "template", Required: true},
			{Key: "field", Label: "Feld", Type: "select", Required: true},
			{Key: "value", Label: "Neuer Wert", Type: "template", Required: true},
		},
		OutputFields: []OutputField{
			{Key: "deal_id", Label: "Deal-ID", Type: "string"},
			{Key: "updated_field", Label: "Aktualisiertes Feld", Type: "string"},
		},
	}
}

// ============================================================================
// CreateContactAction
// ============================================================================

// CreateContactAction creates a new CRM contact via gRPC.
type CreateContactAction struct {
	client crmv1.CRMServiceClient
}

// NewCreateContactAction creates a new CreateContactAction.
func NewCreateContactAction(client crmv1.CRMServiceClient) *CreateContactAction {
	return &CreateContactAction{client: client}
}

func (a *CreateContactAction) Type() string { return "crm.create_contact" }

type createContactConfig struct {
	FirstName string `json:"first_name"` // template
	LastName  string `json:"last_name"`  // template
	Email     string `json:"email"`      // template
}

func (a *CreateContactAction) Execute(ctx context.Context, config json.RawMessage, env map[string]any) (ActionResult, error) {
	if a.client == nil {
		return ActionResult{Success: false, Error: "CRM service unavailable"}, fmt.Errorf("CRM gRPC client is nil")
	}

	var cfg createContactConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return ActionResult{Success: false, Error: "invalid config: " + err.Error()}, err
	}

	firstName := resolveTemplate(cfg.FirstName, env)
	lastName := resolveTemplate(cfg.LastName, env)
	email := resolveTemplate(cfg.Email, env)

	resp, err := a.client.CreateContact(ctx, &crmv1.CreateContactRequest{
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
	})
	if err != nil {
		return ActionResult{Success: false, Error: err.Error()}, err
	}

	return ActionResult{
		Success: true,
		Output: map[string]any{
			"contact_id": resp.GetContact().GetId(),
		},
	}, nil
}

// CreateContactDefinition returns the action definition for the catalog.
func CreateContactDefinition() *ActionDefinition {
	return &ActionDefinition{
		Type:        "crm.create_contact",
		Module:      "crm",
		Name:        "Kontakt erstellen",
		Description: "Erstellt einen neuen Kontakt im CRM",
		Params: []ConfigParam{
			{Key: "first_name", Label: "Vorname", Type: "template", Required: true},
			{Key: "last_name", Label: "Nachname", Type: "template", Required: true},
			{Key: "email", Label: "E-Mail", Type: "template", Required: false},
		},
		OutputFields: []OutputField{
			{Key: "contact_id", Label: "Kontakt-ID", Type: "string"},
		},
	}
}
