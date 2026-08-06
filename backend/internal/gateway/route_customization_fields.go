package gateway

import (
	"net/http"
	"regexp"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
	workv1 "github.com/kmuhub/kmuhub/proto/work/v1"
)

// ============================================================================
// Unified custom fields (desktop/src/renderer/src/mocks/data/custom-fields.ts,
// consumed via api/hooks/useCustomFields.ts against /api/v1/customization/fields)
//
// There is NO third table here. Two real, tenant-scoped tables already exist
// and already have full CRUD gRPC services behind their own REST surfaces:
//   - custom_field_definitions (migration 000005, entity_type in
//     contact/company/deal/activity) via CRMServiceClient, REST at
//     /api/v1/custom-fields
//   - work_custom_field_definitions (migration 000146, task only) via
//     WorkServiceClient, REST at /api/v1/work/custom-fields
// This file is a thin dispatch layer in front of both: it does not touch
// either table, it translates the FE's unified 6-entity/9-type shape to
// whichever backend owns the request. helpdesk_ticket has no backend table
// at all (additive FE-mock-only entity per the file header comment in
// custom-fields.ts) — reads return it empty, writes are rejected with a
// clear 400, never silently dropped.
//
// Column gaps that are NOT bridged, on purpose, because bridging them would
// mean accepting a write and throwing it away — the exact bug class this
// loop run (BACKLOG.yml block B) exists to close elsewhere:
//   - valueSetId, validation{min,max,pattern}: no column on either table.
//     A create/update that sets either is rejected with 400, never ignored.
//   - visible: no column on either table. Always read back as true; a write
//     that explicitly sets it to false is rejected with 400.
//   - required, defaultValue: no column on work_custom_field_definitions
//     (CRM has both). A work_task write with required=true or a non-empty
//     defaultValue is rejected with 400.
//   - inUse: no cheap way to compute "does any record use this field" over
//     either table today; always reported as false. Read-only field, so this
//     is a known simplification, not a silent write loss — see BACKLOG.yml.
// ============================================================================

// crmEntityByFEEntity maps the FE's crm_* entity names to the CRM backend's
// entity_type enum (contact/company/deal/activity — no "crm_" prefix there).
var crmEntityByFEEntity = map[string]string{
	"crm_contact":  "contact",
	"crm_company":  "company",
	"crm_deal":     "deal",
	"crm_activity": "activity",
}

var feEntityByCRMEntity = map[string]string{
	"contact":  "crm_contact",
	"company":  "crm_company",
	"deal":     "crm_deal",
	"activity": "crm_activity",
}

var validCustomFieldEntities = map[string]bool{
	"work_task":       true,
	"crm_contact":     true,
	"crm_company":     true,
	"crm_deal":        true,
	"crm_activity":    true,
	"helpdesk_ticket": true,
}

// feToCRMFieldType translates the FE's 9-value type enum to the CRM
// backend's older 6-value one (models.FieldType, migration 000005 predates
// url/email/phone and spells multi-select without the underscore). Returns
// false when the FE type has no CRM equivalent — the caller must reject the
// request rather than silently coercing it.
func feToCRMFieldType(t string) (string, bool) {
	switch t {
	case "multi_select":
		return "multiselect", true
	case "text", "number", "date", "boolean", "select":
		return t, true
	default:
		return "", false
	}
}

func crmToFEFieldType(t string) string {
	if t == "multiselect" {
		return "multi_select"
	}
	return t
}

var (
	slugNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)
	slugTrim     = regexp.MustCompile(`^_+|_+$`)
	umlautRepl   = strings.NewReplacer("ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss")
)

// slugifyFieldKey mirrors toKey() in mocks/data/custom-fields.ts. It is the
// only place a "key" exists for work_task fields (work_custom_field_definitions
// has no separate key column, only `name`) — derived, not stored, so renaming
// a work_task field's label also changes its key. Documented gap, not fixed
// here: fixing it means a schema change and is out of scope for a route unit.
func slugifyFieldKey(label string) string {
	s := strings.ToLower(label)
	s = umlautRepl.Replace(s)
	s = slugNonAlnum.ReplaceAllString(s, "_")
	s = slugTrim.ReplaceAllString(s, "")
	if len(s) > 48 {
		s = s[:48]
	}
	return s
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// customFieldJSON mirrors CustomFieldDefinition in custom-fields.ts.
// valueSetId, validation and inUse are intentionally absent — see the file
// header comment; a caller reading through the FE's optional-field types
// sees them as simply unset, not as null.
type customFieldJSON struct {
	ID           string   `json:"id"`
	Entity       string   `json:"entity"`
	Key          string   `json:"key"`
	Label        string   `json:"label"`
	Type         string   `json:"type"`
	Required     bool     `json:"required"`
	Options      []string `json:"options"`
	DefaultValue *string  `json:"defaultValue,omitempty"`
	Visible      bool     `json:"visible"`
	Order        int32    `json:"order"`
	InUse        bool     `json:"inUse"`
}

type fieldsListResponseJSON struct {
	Fields []customFieldJSON `json:"fields"`
}

type fieldResponseJSON struct {
	Field customFieldJSON `json:"field"`
}

func crmFieldToJSON(f *crmv1.CustomFieldInfo) customFieldJSON {
	return customFieldJSON{
		ID:           f.GetId(),
		Entity:       feEntityByCRMEntity[f.GetEntityType()],
		Key:          f.GetFieldName(),
		Label:        f.GetFieldLabel(),
		Type:         crmToFEFieldType(f.GetFieldType()),
		Required:     f.GetIsRequired(),
		Options:      nonNilStrings(f.GetOptions()),
		DefaultValue: f.DefaultValue,
		Visible:      true,
		Order:        f.GetSortOrder(),
		InUse:        false,
	}
}

func workDefToJSON(d *workv1.CustomFieldDefinitionProto) customFieldJSON {
	label := d.GetName()
	return customFieldJSON{
		ID:       d.GetId(),
		Entity:   "work_task",
		Key:      slugifyFieldKey(label),
		Label:    label,
		Type:     d.GetFieldType(),
		Required: false,
		Options:  nonNilStrings(d.GetOptions()),
		Visible:  true,
		Order:    d.GetPosition(),
		InUse:    false,
	}
}

func maxFieldOrder(fields []customFieldJSON) int32 {
	max := int32(-1)
	for _, f := range fields {
		if f.Order > max {
			max = f.Order
		}
	}
	return max
}

// getCRMClient and getWorkClient mirror CRMRoutes.getCRMClient /
// WorkRoutes.getWorkClient — this route dispatches to both existing services,
// it does not own either.
func (cr *CustomizationRoutes) getCRMClient() (crmv1.CRMServiceClient, error) {
	conn, err := cr.registry.GetConnection("crm")
	if err != nil {
		return nil, err
	}
	return crmv1.NewCRMServiceClient(conn), nil
}

func (cr *CustomizationRoutes) getWorkClient() (workv1.WorkServiceClient, error) {
	conn, err := cr.registry.GetConnection("work")
	if err != nil {
		return nil, err
	}
	return workv1.NewWorkServiceClient(conn), nil
}

func (cr *CustomizationRoutes) listCRMCustomFields(w http.ResponseWriter, r *http.Request, crmEntityType string) ([]customFieldJSON, bool) {
	client, err := cr.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, "crm")
		return nil, false
	}
	resp, err := client.ListCustomFields(r.Context(), &crmv1.ListCustomFieldsRequest{
		EntityType: crmEntityType,
		Page:       1,
		PageSize:   500, // generous cap, not paginated on the wire — mirrors the flat array the FE mock returns
	})
	if err != nil {
		respondGRPCError(w, err)
		return nil, false
	}
	out := make([]customFieldJSON, 0, len(resp.GetCustomFields()))
	for _, f := range resp.GetCustomFields() {
		out = append(out, crmFieldToJSON(f))
	}
	return out, true
}

func (cr *CustomizationRoutes) listWorkCustomFields(w http.ResponseWriter, r *http.Request) ([]customFieldJSON, bool) {
	client, err := cr.getWorkClient()
	if err != nil {
		respondServiceUnavailable(w, "work")
		return nil, false
	}
	resp, err := client.ListCustomFieldDefinitions(r.Context(), &workv1.ListCustomFieldDefinitionsRequest{})
	if err != nil {
		respondGRPCError(w, err)
		return nil, false
	}
	out := make([]customFieldJSON, 0, len(resp.GetDefinitions()))
	for _, d := range resp.GetDefinitions() {
		out = append(out, workDefToJSON(d))
	}
	return out, true
}

// HandleListCustomFields serves GET /api/v1/customization/fields[?entity=].
func (cr *CustomizationRoutes) HandleListCustomFields(w http.ResponseWriter, r *http.Request) {
	entity := r.URL.Query().Get("entity")
	if entity != "" && !validCustomFieldEntities[entity] {
		response.Error(w, http.StatusBadRequest, "unknown entity")
		return
	}

	fields := make([]customFieldJSON, 0)

	if entity == "" || entity == "work_task" {
		workFields, ok := cr.listWorkCustomFields(w, r)
		if !ok {
			return
		}
		fields = append(fields, workFields...)
	}

	if entity == "" || crmEntityByFEEntity[entity] != "" {
		crmFields, ok := cr.listCRMCustomFields(w, r, crmEntityByFEEntity[entity])
		if !ok {
			return
		}
		fields = append(fields, crmFields...)
	}

	// entity == "helpdesk_ticket" or entity == "" both fall through to here
	// with nothing added for it — no backend table, see file header comment.

	response.JSON(w, http.StatusOK, fieldsListResponseJSON{Fields: fields})
}

// customFieldOwner identifies which backend a resolved field id belongs to.
type customFieldOwner int

const (
	ownerNone customFieldOwner = iota
	ownerCRM
	ownerWork
)

// resolveCustomFieldOwner tries CRM first, then Work — both return a real
// gRPC NotFound on a miss (verified against mapCRMError / mapWorkError), so
// falling through on NotFound and surfacing any other error immediately is
// safe: it never masks a real failure as "try the other backend".
func (cr *CustomizationRoutes) resolveCustomFieldOwner(w http.ResponseWriter, r *http.Request, id string) (customFieldOwner, *crmv1.CustomFieldInfo, *workv1.CustomFieldDefinitionProto, bool) {
	crmClient, err := cr.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, "crm")
		return ownerNone, nil, nil, false
	}
	crmResp, crmErr := crmClient.GetCustomField(r.Context(), &crmv1.GetCustomFieldRequest{Id: id})
	if crmErr == nil {
		return ownerCRM, crmResp.GetCustomField(), nil, true
	}
	if st, ok := status.FromError(crmErr); !ok || st.Code() != codes.NotFound {
		respondGRPCError(w, crmErr)
		return ownerNone, nil, nil, false
	}

	workClient, err := cr.getWorkClient()
	if err != nil {
		respondServiceUnavailable(w, "work")
		return ownerNone, nil, nil, false
	}
	workResp, workErr := workClient.GetCustomFieldDefinition(r.Context(), &workv1.GetCustomFieldDefinitionRequest{Id: id})
	if workErr != nil {
		respondGRPCError(w, workErr) // NotFound here is the real "does not exist anywhere" 404
		return ownerNone, nil, nil, false
	}
	return ownerWork, nil, workResp.GetDefinition(), true
}

// HandleGetCustomField serves GET /api/v1/customization/fields/{id}.
func (cr *CustomizationRoutes) HandleGetCustomField(w http.ResponseWriter, r *http.Request) {
	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}
	owner, crmInfo, workDef, ok := cr.resolveCustomFieldOwner(w, r, id)
	if !ok {
		return
	}
	switch owner {
	case ownerCRM:
		response.JSON(w, http.StatusOK, fieldResponseJSON{Field: crmFieldToJSON(crmInfo)})
	case ownerWork:
		response.JSON(w, http.StatusOK, fieldResponseJSON{Field: workDefToJSON(workDef)})
	}
}

// customFieldValidationRequest exists only so an incoming validation object
// can be detected and rejected — see rejectUnsupportedCustomFieldFields.
type customFieldValidationRequest struct {
	Min     *float64 `json:"min,omitempty"`
	Max     *float64 `json:"max,omitempty"`
	Pattern *string  `json:"pattern,omitempty"`
}

// createCustomFieldUnifiedRequest mirrors CreateCustomFieldInput in
// custom-fields.ts.
type createCustomFieldUnifiedRequest struct {
	Entity       string                        `json:"entity" validate:"required,oneof=work_task crm_contact crm_company crm_deal crm_activity helpdesk_ticket"`
	Label        string                        `json:"label" validate:"required"`
	Type         string                        `json:"type" validate:"required"`
	Required     bool                          `json:"required"`
	Options      []string                      `json:"options"`
	ValueSetID   *string                       `json:"valueSetId,omitempty"`
	Validation   *customFieldValidationRequest `json:"validation,omitempty"`
	DefaultValue *string                       `json:"defaultValue,omitempty"`
	Visible      *bool                         `json:"visible,omitempty"`
}

// updateCustomFieldUnifiedRequest mirrors UpdateCustomFieldInput
// (Partial<Omit<CreateCustomFieldInput,'entity'>>) in custom-fields.ts.
type updateCustomFieldUnifiedRequest struct {
	Label        *string                       `json:"label,omitempty"`
	Type         *string                       `json:"type,omitempty"`
	Required     *bool                         `json:"required,omitempty"`
	Options      []string                      `json:"options,omitempty"`
	ValueSetID   *string                       `json:"valueSetId,omitempty"`
	Validation   *customFieldValidationRequest `json:"validation,omitempty"`
	DefaultValue *string                       `json:"defaultValue,omitempty"`
	Visible      *bool                         `json:"visible,omitempty"`
}

// rejectUnsupportedCustomFieldFields refuses (400, not silent drop) a write
// that touches valueSetId/validation/visible=false — none of which either
// backing table can persist today.
func rejectUnsupportedCustomFieldFields(w http.ResponseWriter, valueSetID *string, validation *customFieldValidationRequest, visible *bool) bool {
	if valueSetID != nil {
		response.Error(w, http.StatusBadRequest, "valueSetId is not supported yet (no backend column) — see BACKLOG.yml customization-fields-route")
		return false
	}
	if validation != nil {
		response.Error(w, http.StatusBadRequest, "validation is not supported yet (no backend column) — see BACKLOG.yml customization-fields-route")
		return false
	}
	if visible != nil && !*visible {
		response.Error(w, http.StatusBadRequest, "visible=false is not supported yet (no backend column, fields are always visible)")
		return false
	}
	return true
}

// HandleCreateCustomField serves POST /api/v1/customization/fields.
func (cr *CustomizationRoutes) HandleCreateCustomField(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeAndValidate[createCustomFieldUnifiedRequest](w, r)
	if !ok {
		return
	}
	if !rejectUnsupportedCustomFieldFields(w, req.ValueSetID, req.Validation, req.Visible) {
		return
	}

	if req.Entity == "work_task" {
		if req.Required {
			response.Error(w, http.StatusBadRequest, "required is not supported for work_task custom fields yet (no backend column)")
			return
		}
		if req.DefaultValue != nil && *req.DefaultValue != "" {
			response.Error(w, http.StatusBadRequest, "defaultValue is not supported for work_task custom fields yet (no backend column)")
			return
		}
		cr.createWorkCustomField(w, r, req)
		return
	}

	if crmEntityByFEEntity[req.Entity] != "" {
		cr.createCRMCustomField(w, r, req)
		return
	}

	// helpdesk_ticket: no backend table. Reject explicitly rather than
	// pretending to accept it — see file header comment.
	response.Error(w, http.StatusBadRequest, "custom fields for helpdesk_ticket are not backed by the server yet — see BACKLOG.yml customization-fields-route")
}

func (cr *CustomizationRoutes) createWorkCustomField(w http.ResponseWriter, r *http.Request, req createCustomFieldUnifiedRequest) {
	existing, ok := cr.listWorkCustomFields(w, r)
	if !ok {
		return
	}
	client, err := cr.getWorkClient()
	if err != nil {
		respondServiceUnavailable(w, "work")
		return
	}
	resp, err := client.CreateCustomFieldDefinition(r.Context(), &workv1.CreateCustomFieldDefinitionRequest{
		Name:      req.Label,
		FieldType: req.Type,
		Options:   req.Options,
		Position:  maxFieldOrder(existing) + 1,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, fieldResponseJSON{Field: workDefToJSON(resp.GetDefinition())})
}

func (cr *CustomizationRoutes) createCRMCustomField(w http.ResponseWriter, r *http.Request, req createCustomFieldUnifiedRequest) {
	crmType, ok := feToCRMFieldType(req.Type)
	if !ok {
		response.Error(w, http.StatusBadRequest, "field type '"+req.Type+"' is only supported for work_task, not for CRM entities")
		return
	}
	crmEntityType := crmEntityByFEEntity[req.Entity]

	existing, ok := cr.listCRMCustomFields(w, r, crmEntityType)
	if !ok {
		return
	}
	client, err := cr.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, "crm")
		return
	}
	grpcReq := &crmv1.CreateCustomFieldRequest{
		EntityType: crmEntityType,
		FieldName:  slugifyFieldKey(req.Label),
		FieldLabel: req.Label,
		FieldType:  crmType,
		Options:    req.Options,
		IsRequired: req.Required,
		SortOrder:  maxFieldOrder(existing) + 1,
		CreatedBy:  middleware.GetUserID(r.Context()),
	}
	if req.DefaultValue != nil {
		grpcReq.DefaultValue = req.DefaultValue
	}

	resp, err := client.CreateCustomField(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, fieldResponseJSON{Field: crmFieldToJSON(resp.GetCustomField())})
}

// HandleUpdateCustomField serves PUT /api/v1/customization/fields/{id}.
// This is a merge-patch: only fields present in the body change, matching
// the FE's Partial<...> update type. CRM's own UpdateCustomField RPC is
// already partial-friendly; Work's replaces the whole row, so the current
// definition is fetched first and merged before writing it back.
func (cr *CustomizationRoutes) HandleUpdateCustomField(w http.ResponseWriter, r *http.Request) {
	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}
	req, ok := decodeAndValidate[updateCustomFieldUnifiedRequest](w, r)
	if !ok {
		return
	}
	if !rejectUnsupportedCustomFieldFields(w, req.ValueSetID, req.Validation, req.Visible) {
		return
	}

	owner, _, workDef, ok := cr.resolveCustomFieldOwner(w, r, id)
	if !ok {
		return
	}

	switch owner {
	case ownerCRM:
		if req.Type != nil {
			response.Error(w, http.StatusBadRequest, "type cannot be changed after creation for CRM-backed entities (crm_contact/crm_company/crm_deal/crm_activity)")
			return
		}
		client, err := cr.getCRMClient()
		if err != nil {
			respondServiceUnavailable(w, "crm")
			return
		}
		grpcReq := &crmv1.UpdateCustomFieldRequest{Id: id, Options: req.Options}
		if req.Label != nil {
			grpcReq.FieldLabel = req.Label
		}
		if req.DefaultValue != nil {
			grpcReq.DefaultValue = req.DefaultValue
		}
		if req.Required != nil {
			grpcReq.IsRequired = req.Required
		}
		resp, err := client.UpdateCustomField(r.Context(), grpcReq)
		if err != nil {
			respondGRPCError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, fieldResponseJSON{Field: crmFieldToJSON(resp.GetCustomField())})

	case ownerWork:
		if req.Required != nil && *req.Required {
			response.Error(w, http.StatusBadRequest, "required is not supported for work_task custom fields yet (no backend column)")
			return
		}
		if req.DefaultValue != nil && *req.DefaultValue != "" {
			response.Error(w, http.StatusBadRequest, "defaultValue is not supported for work_task custom fields yet (no backend column)")
			return
		}
		client, err := cr.getWorkClient()
		if err != nil {
			respondServiceUnavailable(w, "work")
			return
		}
		name := workDef.GetName()
		if req.Label != nil {
			name = *req.Label
		}
		fieldType := workDef.GetFieldType()
		if req.Type != nil {
			fieldType = *req.Type
		}
		options := workDef.GetOptions()
		if req.Options != nil {
			options = req.Options
		}
		resp, err := client.UpdateCustomFieldDefinition(r.Context(), &workv1.UpdateCustomFieldDefinitionRequest{
			Id:        id,
			Name:      name,
			FieldType: fieldType,
			Options:   options,
			Position:  workDef.GetPosition(),
		})
		if err != nil {
			respondGRPCError(w, err)
			return
		}
		response.JSON(w, http.StatusOK, fieldResponseJSON{Field: workDefToJSON(resp.GetDefinition())})
	}
}

// HandleDeleteCustomField serves DELETE /api/v1/customization/fields/{id}.
// CRM enforces ErrFieldInUse (409) when the field still has data on a
// record; work_custom_field_definitions has no such guard. That asymmetry
// is a real backend behaviour difference, not introduced here — left as-is.
func (cr *CustomizationRoutes) HandleDeleteCustomField(w http.ResponseWriter, r *http.Request) {
	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	crmClient, err := cr.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, "crm")
		return
	}
	_, crmErr := crmClient.DeleteCustomField(r.Context(), &crmv1.DeleteCustomFieldRequest{Id: id})
	if crmErr == nil {
		response.JSON(w, http.StatusOK, map[string]string{"status": "custom field deleted"})
		return
	}
	if st, ok := status.FromError(crmErr); !ok || st.Code() != codes.NotFound {
		respondGRPCError(w, crmErr)
		return
	}

	workClient, err := cr.getWorkClient()
	if err != nil {
		respondServiceUnavailable(w, "work")
		return
	}
	if _, err := workClient.DeleteCustomFieldDefinition(r.Context(), &workv1.DeleteCustomFieldDefinitionRequest{Id: id}); err != nil {
		respondGRPCError(w, err) // NotFound here is the real "does not exist anywhere" 404
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "custom field deleted"})
}
