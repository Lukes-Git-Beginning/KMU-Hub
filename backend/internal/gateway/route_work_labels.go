package gateway

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/server/response"
	workv1 "github.com/kmuhub/kmuhub/proto/work/v1"
)

// ============================================================================
// Label Handlers
// ============================================================================

type createLabelRequest struct {
	Name  string `json:"name"  validate:"required"`
	Color string `json:"color" validate:"omitempty"`
}

type updateLabelRequest struct {
	Name  string `json:"name"  validate:"required"`
	Color string `json:"color" validate:"omitempty"`
}

func (w *WorkRoutes) HandleListLabels(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	resp, err := client.ListLabels(r.Context(), &workv1.ListLabelsRequest{})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.Proto(wr, http.StatusOK, resp)
}

func (w *WorkRoutes) HandleCreateLabel(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	req, ok := decodeAndValidate[createLabelRequest](wr, r)
	if !ok {
		return
	}

	grpcReq := &workv1.CreateLabelRequest{
		Name:  req.Name,
		Color: req.Color,
	}

	resp, err := client.CreateLabel(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.Proto(wr, http.StatusCreated, resp)
}

func (w *WorkRoutes) HandleGetLabel(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	labelID, ok := validateUUIDParam(wr, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetLabel(r.Context(), &workv1.GetLabelRequest{Id: labelID})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.Proto(wr, http.StatusOK, resp)
}

func (w *WorkRoutes) HandleUpdateLabel(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	labelID := chi.URLParam(r, "id")
	if labelID == "" {
		response.Error(wr, http.StatusBadRequest, "missing label id")
		return
	}

	req, ok := decodeAndValidate[updateLabelRequest](wr, r)
	if !ok {
		return
	}

	resp, err := client.UpdateLabel(r.Context(), &workv1.UpdateLabelRequest{
		Id:    labelID,
		Name:  req.Name,
		Color: req.Color,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.Proto(wr, http.StatusOK, resp)
}

func (w *WorkRoutes) HandleDeleteLabel(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	labelID := chi.URLParam(r, "id")
	if labelID == "" {
		response.Error(wr, http.StatusBadRequest, "missing label id")
		return
	}

	_, err = client.DeleteLabel(r.Context(), &workv1.DeleteLabelRequest{Id: labelID})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	wr.WriteHeader(http.StatusNoContent)
}

// HandleSetTaskLabels replaces all labels for a task.
// Called via PUT /api/v1/tasks/{id}/labels
func (w *WorkRoutes) HandleSetTaskLabels(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	taskID, ok := validateUUIDParam(wr, r, "id")
	if !ok {
		return
	}

	type setTaskLabelsReq struct {
		LabelIDs []string `json:"label_ids"`
	}
	req, ok := decodeAndValidate[setTaskLabelsReq](wr, r)
	if !ok {
		return
	}

	_, err = client.SetTaskLabels(r.Context(), &workv1.SetTaskLabelsRequest{
		TaskId:   taskID,
		LabelIds: req.LabelIDs,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	wr.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Custom Field Definition Handlers
// ============================================================================

type createCustomFieldDefinitionRequest struct {
	Name      string   `json:"name"       validate:"required"`
	FieldType string   `json:"field_type" validate:"required"`
	Options   []string `json:"options,omitempty"`
	Position  int32    `json:"position,omitempty"`
}

type updateCustomFieldDefinitionRequest struct {
	Name      string   `json:"name"       validate:"required"`
	FieldType string   `json:"field_type" validate:"required"`
	Options   []string `json:"options,omitempty"`
	Position  int32    `json:"position,omitempty"`
}

func (w *WorkRoutes) HandleListCustomFieldDefinitions(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	resp, err := client.ListCustomFieldDefinitions(r.Context(), &workv1.ListCustomFieldDefinitionsRequest{})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.Proto(wr, http.StatusOK, resp)
}

func (w *WorkRoutes) HandleCreateCustomFieldDefinition(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	req, ok := decodeAndValidate[createCustomFieldDefinitionRequest](wr, r)
	if !ok {
		return
	}

	resp, err := client.CreateCustomFieldDefinition(r.Context(), &workv1.CreateCustomFieldDefinitionRequest{
		Name:      req.Name,
		FieldType: req.FieldType,
		Options:   req.Options,
		Position:  req.Position,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.Proto(wr, http.StatusCreated, resp)
}

func (w *WorkRoutes) HandleGetCustomFieldDefinition(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	defID, ok := validateUUIDParam(wr, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetCustomFieldDefinition(r.Context(), &workv1.GetCustomFieldDefinitionRequest{Id: defID})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.Proto(wr, http.StatusOK, resp)
}

func (w *WorkRoutes) HandleUpdateCustomFieldDefinition(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	defID := chi.URLParam(r, "id")
	if defID == "" {
		response.Error(wr, http.StatusBadRequest, "missing definition id")
		return
	}

	req, ok := decodeAndValidate[updateCustomFieldDefinitionRequest](wr, r)
	if !ok {
		return
	}

	resp, err := client.UpdateCustomFieldDefinition(r.Context(), &workv1.UpdateCustomFieldDefinitionRequest{
		Id:        defID,
		Name:      req.Name,
		FieldType: req.FieldType,
		Options:   req.Options,
		Position:  req.Position,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.Proto(wr, http.StatusOK, resp)
}

func (w *WorkRoutes) HandleDeleteCustomFieldDefinition(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	defID := chi.URLParam(r, "id")
	if defID == "" {
		response.Error(wr, http.StatusBadRequest, "missing definition id")
		return
	}

	_, err = client.DeleteCustomFieldDefinition(r.Context(), &workv1.DeleteCustomFieldDefinitionRequest{Id: defID})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	wr.WriteHeader(http.StatusNoContent)
}

// labelIDsFromQuery parses comma-separated label IDs from a query parameter.
func labelIDsFromQuery(r *http.Request, param string) []string {
	raw := r.URL.Query().Get(param)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
