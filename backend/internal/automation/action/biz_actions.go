package action

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// ============================================================================
// CreateInvoiceDraftAction
// ============================================================================

// CreateInvoiceDraftAction creates an invoice draft via the Finance gRPC service.
type CreateInvoiceDraftAction struct {
	client bizv1.FinanceServiceClient
}

// NewCreateInvoiceDraftAction creates a new CreateInvoiceDraftAction.
func NewCreateInvoiceDraftAction(client bizv1.FinanceServiceClient) *CreateInvoiceDraftAction {
	return &CreateInvoiceDraftAction{client: client}
}

func (a *CreateInvoiceDraftAction) Type() string { return "biz.create_invoice_draft" }

type createInvoiceDraftConfig struct {
	TenantID  string `json:"tenant_id"`  // template
	CreatedBy string `json:"created_by"` // template: user ID
	Notes     string `json:"notes"`      // template (optional)
}

func (a *CreateInvoiceDraftAction) Execute(ctx context.Context, config json.RawMessage, env map[string]any) (ActionResult, error) {
	if a.client == nil {
		return ActionResult{Success: false, Error: "Finance service unavailable"}, fmt.Errorf("Finance gRPC client is nil")
	}

	var cfg createInvoiceDraftConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return ActionResult{Success: false, Error: "invalid config: " + err.Error()}, err
	}

	tenantID := resolveTemplate(cfg.TenantID, env)
	createdBy := resolveTemplate(cfg.CreatedBy, env)
	notes := resolveTemplate(cfg.Notes, env)

	req := &bizv1.CreateInvoiceRequest{
		TenantId:  tenantID,
		CreatedBy: createdBy,
		Notes:     notes,
	}

	resp, err := a.client.CreateInvoice(ctx, req)
	if err != nil {
		return ActionResult{Success: false, Error: err.Error()}, err
	}

	return ActionResult{
		Success: true,
		Output: map[string]any{
			"invoice_id": resp.GetInvoice().GetId(),
		},
	}, nil
}

// CreateInvoiceDraftDefinition returns the action definition for the catalog.
func CreateInvoiceDraftDefinition() *ActionDefinition {
	return &ActionDefinition{
		Type:        "biz.create_invoice_draft",
		Module:      "biz",
		Name:        "Rechnungsentwurf erstellen",
		Description: "Erstellt einen neuen Rechnungsentwurf im Finanzsystem",
		Params: []ConfigParam{
			{Key: "tenant_id", Label: "Mandant-ID", Type: "template", Required: true},
			{Key: "created_by", Label: "Erstellt von", Type: "template", Required: true},
			{Key: "notes", Label: "Notizen", Type: "template", Required: false},
		},
		OutputFields: []OutputField{
			{Key: "invoice_id", Label: "Rechnungs-ID", Type: "string"},
		},
	}
}

// ============================================================================
// CreateDunningAction
// ============================================================================

// CreateDunningAction creates a dunning record for an overdue invoice via the Finance gRPC service.
type CreateDunningAction struct {
	client bizv1.FinanceServiceClient
}

// NewCreateDunningAction creates a new CreateDunningAction.
func NewCreateDunningAction(client bizv1.FinanceServiceClient) *CreateDunningAction {
	return &CreateDunningAction{client: client}
}

func (a *CreateDunningAction) Type() string { return "biz.create_dunning" }

type createDunningConfig struct {
	TenantID  string `json:"tenant_id"`  // template
	InvoiceID string `json:"invoice_id"` // template
	Level     string `json:"level"`      // template: 1, 2, 3
	CreatedBy string `json:"created_by"` // template: user ID
}

func (a *CreateDunningAction) Execute(ctx context.Context, config json.RawMessage, env map[string]any) (ActionResult, error) {
	if a.client == nil {
		return ActionResult{Success: false, Error: "Finance service unavailable"}, fmt.Errorf("Finance gRPC client is nil")
	}

	var cfg createDunningConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return ActionResult{Success: false, Error: "invalid config: " + err.Error()}, err
	}

	tenantID := resolveTemplate(cfg.TenantID, env)
	invoiceID := resolveTemplate(cfg.InvoiceID, env)
	levelStr := resolveTemplate(cfg.Level, env)
	createdBy := resolveTemplate(cfg.CreatedBy, env)

	level := int32(1)
	if n, err := strconv.Atoi(levelStr); err == nil {
		level = int32(n)
	}

	req := &bizv1.CreateDunningRequest{
		TenantId:  tenantID,
		InvoiceId: invoiceID,
		Level:     level,
		CreatedBy: createdBy,
	}

	resp, err := a.client.CreateDunning(ctx, req)
	if err != nil {
		return ActionResult{Success: false, Error: err.Error()}, err
	}

	return ActionResult{
		Success: true,
		Output: map[string]any{
			"dunning_id": resp.GetDunning().GetId(),
		},
	}, nil
}

// CreateDunningDefinition returns the action definition for the catalog.
func CreateDunningDefinition() *ActionDefinition {
	return &ActionDefinition{
		Type:        "biz.create_dunning",
		Module:      "biz",
		Name:        "Mahnung erstellen",
		Description: "Erstellt eine Mahnung fuer eine ueberfaellige Rechnung",
		Params: []ConfigParam{
			{Key: "tenant_id", Label: "Mandant-ID", Type: "template", Required: true},
			{Key: "invoice_id", Label: "Rechnungs-ID", Type: "template", Required: true},
			{Key: "level", Label: "Mahnstufe", Type: "number", Required: true, Default: 1},
			{Key: "created_by", Label: "Erstellt von", Type: "template", Required: true},
		},
		OutputFields: []OutputField{
			{Key: "dunning_id", Label: "Mahnungs-ID", Type: "string"},
		},
	}
}
