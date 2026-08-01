package gateway

// Lead inbox. A lead is a contact at an earlier lifecycle stage (migration
// 000259), so these routes talk to the CRM service like the contact routes
// next door -- there is no lead service and no lead table.

import (
	"net/http"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

// leadWire is the exact shape the desktop inbox reads (the Lead type in
// api/hooks/useLeads.ts). Hand-mapped because that type is camelCase while
// response.Proto would emit the snake_case proto field names.
type leadWire struct {
	ID                  string  `json:"id"`
	FirstName           string  `json:"firstName"`
	LastName            string  `json:"lastName"`
	Company             string  `json:"company"`
	Email               string  `json:"email"`
	Phone               string  `json:"phone"`
	Source              string  `json:"source"`
	Score               int32   `json:"score"`
	Temperature         string  `json:"temperature"`
	TemperatureOverride *string `json:"temperatureOverride,omitempty"`
	Status              string  `json:"status"`
	LifecycleStage      string  `json:"lifecycleStage"`
	Notes               string  `json:"notes"`
	CreatedAt           string  `json:"createdAt"`
}

func toLeadWire(l *crmv1.LeadInfo) leadWire {
	return leadWire{
		ID:                  l.GetId(),
		FirstName:           l.GetFirstName(),
		LastName:            l.GetLastName(),
		Company:             l.GetCompany(),
		Email:               l.GetEmail(),
		Phone:               l.GetPhone(),
		Source:              l.GetSource(),
		Score:               l.GetScore(),
		Temperature:         l.GetTemperature(),
		TemperatureOverride: l.TemperatureOverride,
		Status:              l.GetStatus(),
		LifecycleStage:      l.GetLifecycleStage(),
		Notes:               l.GetNotes(),
		CreatedAt:           l.GetCreatedAt(),
	}
}

// HandleListLeads serves the tenant's lead inbox.
// GET /api/v1/leads
func (c *CRMRoutes) HandleListLeads(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	page, pageSize := parsePagination(r, 1, 20)

	resp, err := client.ListLeads(r.Context(), &crmv1.ListLeadsRequest{
		Stage:    r.URL.Query().Get("stage"),
		Status:   r.URL.Query().Get("status"),
		Search:   r.URL.Query().Get("search"),
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	items := make([]leadWire, 0, len(resp.GetLeads()))
	for _, l := range resp.GetLeads() {
		items = append(items, toLeadWire(l))
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"items": items,
		"total": resp.GetTotal(),
	})
}

type createLeadRequest struct {
	FirstName string `json:"firstName" validate:"required"`
	LastName  string `json:"lastName" validate:"required"`
	Company   string `json:"company"`
	Email     string `json:"email" validate:"omitempty,email"`
	Phone     string `json:"phone" validate:"omitempty,phone_dach"`
	Notes     string `json:"notes"`
	Source    string `json:"source" validate:"required,oneof=manual csv dialer"`
}

// HandleCreateLead records a raw prospect. The score is computed server-side;
// a client-sent score would just be a client-chosen priority.
// POST /api/v1/leads
func (c *CRMRoutes) HandleCreateLead(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	req, ok := decodeAndValidate[createLeadRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.CreateLead(r.Context(), &crmv1.CreateLeadRequest{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Company:   req.Company,
		Email:     req.Email,
		Phone:     req.Phone,
		Notes:     req.Notes,
		Source:    req.Source,
		CreatedBy: middleware.GetUserID(r.Context()),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, map[string]any{"lead": toLeadWire(resp.GetLead())})
}

type updateLeadRequest struct {
	Status *string `json:"status,omitempty" validate:"omitempty,oneof=new contacted qualified disqualified"`
	// An empty string clears the manual override and hands the displayed
	// temperature back to the computed score.
	Temperature *string `json:"temperature,omitempty" validate:"omitempty,oneof=hot warm cold"`
}

// HandleUpdateLead moves a lead through the inbox or pins its temperature.
// PATCH /api/v1/leads/{id}
func (c *CRMRoutes) HandleUpdateLead(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	leadID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateLeadRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.UpdateLead(r.Context(), &crmv1.UpdateLeadRequest{
		Id:          leadID,
		Status:      req.Status,
		Temperature: req.Temperature,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"lead": toLeadWire(resp.GetLead())})
}

// HandleConvertLead qualifies a lead. The row already is the contact, so this
// is a stage change rather than a copy into a second table.
// POST /api/v1/leads/{id}/convert
func (c *CRMRoutes) HandleConvertLead(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	leadID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.ConvertLead(r.Context(), &crmv1.ConvertLeadRequest{Id: leadID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"lead": toLeadWire(resp.GetLead())})
}
