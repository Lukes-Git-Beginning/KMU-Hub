package gateway

import (
	"maps"
	"net/http"
	"regexp"

	"github.com/go-chi/chi/v5"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	settingsv1 "github.com/kmuhub/kmuhub/proto/settings/v1"
)

// CustomizationRoutes handles the label-override and value-set slices of the
// Customization v1.0 fundament (desktop/src/renderer/src/api/customization-types.ts).
//
// Labels deliberately reuse the generic tenant_settings storage (module_id
// "customization", one row per locale holding a sparse key→value JSON
// object) via the existing Settings gRPC client — no new table, no new
// proto RPC. Value-sets have their own tables (migration 000295) and proto
// RPCs (ListValueSets/GetValueSet/UpsertValueSet) behind the same client.
// Drafts/scheduling and module-area visibility are NOT part of this: their
// FE contract is still in flux (BACKLOG.yml, block E head comment).
// The vendor overlay layer stays unwritable server-side until R-5 (GDAP
// vendor access) is wired — the FE mock's activeConfigLayer() also always
// returns "tenant" for the same reason, for both labels and value-sets.
type CustomizationRoutes struct {
	registry *ServiceRegistry
}

// NewCustomizationRoutes creates a new CustomizationRoutes with the given service registry.
func NewCustomizationRoutes(registry *ServiceRegistry) *CustomizationRoutes {
	return &CustomizationRoutes{registry: registry}
}

// ServiceName returns "auth" because the Settings gRPC server backing this
// route is co-located with auth on port :50051 — same as SettingsRoutes.
func (cr *CustomizationRoutes) ServiceName() string { return "auth" }

// getSettingsClient lazily obtains a gRPC client for the Settings service.
func (cr *CustomizationRoutes) getSettingsClient() (settingsv1.SettingsServiceClient, error) {
	conn, err := cr.registry.GetConnection("auth")
	if err != nil {
		return nil, err
	}
	return settingsv1.NewSettingsServiceClient(conn), nil
}

// customizationModuleID is the tenant_settings module_id under which all
// customization label overrides are stored (one row per locale).
const customizationModuleID = "customization"

// labelWhitelist mirrors LABEL_WHITELIST in
// desktop/src/renderer/src/mocks/data/customization.ts — the curated i18n
// keys tenants may override. Keep in sync by hand; both lists are short and
// change rarely (module identity/nav keys are explicitly excluded).
var labelWhitelist = []string{
	"crm.contacts.title",
	"crm.companies.title",
	"crm.deals.title",
	"work.tasks.title",
	"work.projects.title",
	"kontakte.category.all",
	"kontakte.category.employee",
	"kontakte.category.customers",
	"kontakte.category.partner",
}

var labelWhitelistSet = func() map[string]bool {
	m := make(map[string]bool, len(labelWhitelist))
	for _, k := range labelWhitelist {
		m[k] = true
	}
	return m
}()

// localePattern accepts simple language codes ("de") and language-region
// codes ("de-DE") — enough to reject garbage keys without hard-coding the
// small set of locales the app ships today.
var localePattern = regexp.MustCompile(`^[a-z]{2}(-[A-Z]{2})?$`)

// RegisterRoutes registers the customization label-override and value-set
// HTTP routes.
//
// Routes summary:
//
//	GET /api/v1/customization/labels?locale=de       — resolved labels (any authenticated user)
//	PUT /api/v1/customization/labels                  — upsert tenant overrides (admin:customization:manage)
//	GET /api/v1/customization/value-sets[?base=1]     — all resolved value-sets (any authenticated user)
//	GET /api/v1/customization/value-sets/{id}[?base=1] — one resolved value-set (any authenticated user)
//	PUT /api/v1/customization/value-sets/{id}          — upsert tenant override (admin:customization:manage)
//	GET /api/v1/customization/fields[?entity=]         — unified custom fields (any authenticated user)
//	GET /api/v1/customization/fields/{id}              — one custom field (any authenticated user)
//	POST /api/v1/customization/fields                  — create (admin:customization:manage)
//	PUT /api/v1/customization/fields/{id}               — merge-patch update (admin:customization:manage)
//	DELETE /api/v1/customization/fields/{id}            — delete (admin:customization:manage)
func (cr *CustomizationRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/customization/labels", func(r chi.Router) {
		r.Use(authMiddleware)
		// Resolved labels drive UI text for every user, not just the ones who
		// may edit them — no RequirePermission guard on the read side.
		r.Get("/", cr.HandleGetLabels)
		r.With(middleware.RequirePermission("admin:customization", "manage")).Put("/", cr.HandlePutLabels)
	})

	r.Route("/api/v1/customization/value-sets", func(r chi.Router) {
		r.Use(authMiddleware)
		// Same read/write split as labels: resolved value-sets drive pickers
		// and status dots for every user, only the write is permission-gated.
		r.Get("/", cr.HandleListValueSets)
		r.Get("/{id}", cr.HandleGetValueSet)
		r.With(middleware.RequirePermission("admin:customization", "manage")).Put("/{id}", cr.HandlePutValueSet)
	})

	// Unified custom fields (route_customization_fields.go) — a dispatch
	// layer over the existing CRM and Work custom-field services, not a new
	// table. Same read/write split as labels and value-sets.
	r.Route("/api/v1/customization/fields", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Get("/", cr.HandleListCustomFields)
		r.Get("/{id}", cr.HandleGetCustomField)
		r.With(middleware.RequirePermission("admin:customization", "manage")).Post("/", cr.HandleCreateCustomField)
		r.With(middleware.RequirePermission("admin:customization", "manage")).Put("/{id}", cr.HandleUpdateCustomField)
		r.With(middleware.RequirePermission("admin:customization", "manage")).Delete("/{id}", cr.HandleDeleteCustomField)
	})
}

// resolvedLabelJSON mirrors ResolvedLabel in customization-types.ts.
type resolvedLabelJSON struct {
	Value      string `json:"value"`
	Provenance string `json:"provenance"`
	Key        string `json:"key"`
}

// labelOverridesResponseJSON mirrors LabelOverridesResponse in customization-types.ts.
type labelOverridesResponseJSON struct {
	Locale string                       `json:"locale"`
	Labels map[string]resolvedLabelJSON `json:"labels"`
}

// resolveLabels builds the full whitelist-keyed response: a stored override
// wins with provenance "tenant"; a whitelist key with no stored override
// falls back to the code-bundled i18n default (empty value here — the FE
// bundle already holds the real text) with provenance "default".
func resolveLabels(overrides map[string]string) map[string]resolvedLabelJSON {
	out := make(map[string]resolvedLabelJSON, len(labelWhitelist))
	for _, key := range labelWhitelist {
		if v, ok := overrides[key]; ok && v != "" {
			out[key] = resolvedLabelJSON{Value: v, Provenance: "tenant", Key: key}
		} else {
			out[key] = resolvedLabelJSON{Value: "", Provenance: "default", Key: key}
		}
	}
	return out
}

// structValueToStringMap converts a structpb object Value back into a
// map[string]string, dropping any non-string entries defensively. Returns
// nil for a nil Value or a Value that is not a JSON object.
func structValueToStringMap(v *structpb.Value) map[string]string {
	if v == nil {
		return nil
	}
	raw, ok := v.AsInterface().(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]string, len(raw))
	for k, val := range raw {
		if s, ok := val.(string); ok {
			out[k] = s
		}
	}
	return out
}

// applyLabelOverrides merges incoming label overrides onto the existing
// stored map for one locale: unknown (non-whitelisted) keys are dropped
// (mirrors the MSW mock's "silently filtered"), an empty-string value
// clears that key back to the default, and everything else overwrites.
// Pure and side-effect-free so the merge policy is testable without a
// gRPC round trip.
func applyLabelOverrides(existing, incoming map[string]string) map[string]string {
	merged := make(map[string]string, len(existing)+len(incoming))
	maps.Copy(merged, existing)
	for key, value := range incoming {
		if !labelWhitelistSet[key] {
			continue
		}
		if value == "" {
			delete(merged, key)
			continue
		}
		merged[key] = value
	}
	return merged
}

// getTenantLabelOverrides fetches the tenant's raw override map for one
// locale (nil if none stored yet — callers treat that as "all defaults").
func (cr *CustomizationRoutes) getTenantLabelOverrides(r *http.Request, client settingsv1.SettingsServiceClient, tenantID, locale string) (map[string]string, error) {
	resp, err := client.GetTenantSettings(r.Context(), &settingsv1.GetTenantSettingsRequest{
		TenantId: tenantID,
		ModuleId: customizationModuleID,
	})
	if err != nil {
		return nil, err
	}
	for _, e := range resp.GetEntries() {
		if e.GetKey() == locale {
			return structValueToStringMap(e.GetValue()), nil
		}
	}
	return nil, nil
}

// HandleGetLabels returns the resolved label map for a locale (default "de").
func (cr *CustomizationRoutes) HandleGetLabels(w http.ResponseWriter, r *http.Request) {
	client, err := cr.getSettingsClient()
	if err != nil {
		respondServiceUnavailable(w, cr.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	locale := r.URL.Query().Get("locale")
	if locale == "" {
		locale = "de"
	}
	if !localePattern.MatchString(locale) {
		response.Error(w, http.StatusBadRequest, "invalid locale")
		return
	}

	// base=1 returns the code-default baseline (empty overlay) for every
	// whitelist key — used by the editor's "Cosmi-Standard" comparison view.
	// Skips the settings lookup entirely: an empty override map already
	// resolves every key to provenance "default".
	var overrides map[string]string
	if r.URL.Query().Get("base") != "1" {
		overrides, err = cr.getTenantLabelOverrides(r, client, tenantID.String(), locale)
		if err != nil {
			respondGRPCError(w, err)
			return
		}
	}

	response.JSON(w, http.StatusOK, labelOverridesResponseJSON{
		Locale: locale,
		Labels: resolveLabels(overrides),
	})
}

// updateLabelOverridesRequest is the HTTP body for PUT /customization/labels.
// Mirrors UpdateLabelOverridesInput in customization-types.ts.
type updateLabelOverridesRequest struct {
	Locale    string            `json:"locale" validate:"required"`
	Layer     string            `json:"layer" validate:"omitempty,oneof=tenant vendor"`
	Overrides map[string]string `json:"overrides" validate:"required"`
}

// HandlePutLabels upserts label overrides for a locale on the tenant layer.
// Unknown (non-whitelisted) keys are silently dropped — same behaviour as
// the MSW mock. An empty-string value clears that key's override.
func (cr *CustomizationRoutes) HandlePutLabels(w http.ResponseWriter, r *http.Request) {
	client, err := cr.getSettingsClient()
	if err != nil {
		respondServiceUnavailable(w, cr.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	callerID := middleware.GetUserID(r.Context())
	if callerID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	req, ok := decodeAndValidate[updateLabelOverridesRequest](w, r)
	if !ok {
		return
	}
	if !localePattern.MatchString(req.Locale) {
		response.Error(w, http.StatusBadRequest, "invalid locale")
		return
	}
	// The vendor overlay layer has no writer yet (R-5 GDAP vendor access is
	// not wired) — reject explicitly rather than silently writing a vendor
	// request into the tenant layer.
	if req.Layer == "vendor" {
		response.Error(w, http.StatusBadRequest, "layer 'vendor' is not writable yet")
		return
	}

	existing, err := cr.getTenantLabelOverrides(r, client, tenantID.String(), req.Locale)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	merged := applyLabelOverrides(existing, req.Overrides)

	mergedAny := make(map[string]any, len(merged))
	for k, v := range merged {
		mergedAny[k] = v
	}
	mergedValue, err := structpb.NewValue(mergedAny)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to encode overrides")
		return
	}

	_, err = client.PutTenantSettings(r.Context(), &settingsv1.PutTenantSettingsRequest{
		TenantId:  tenantID.String(),
		ModuleId:  customizationModuleID,
		UpdatedBy: callerID,
		Entries: []*settingsv1.SettingEntry{
			{Key: req.Locale, Value: mergedValue},
		},
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, labelOverridesResponseJSON{
		Locale: req.Locale,
		Labels: resolveLabels(merged),
	})
}

// ============================================================================
// Value-sets (migration 000295, settingsv1.ListValueSets/GetValueSet/UpsertValueSet)
// ============================================================================

// valueSetOptionJSON mirrors ResolvedValueSetOption in customization-types.ts.
// Field names are camelCase and `id` (not `key`/`option_key`) on purpose: the
// MSW mock is the binding wire contract for this slice and it deviates from
// the snake_case/{items,total} house rule (BACKLOG.yml customization-value-sets-routes).
type valueSetOptionJSON struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Color      string `json:"color,omitempty"`
	Order      int32  `json:"order"`
	Active     bool   `json:"active"`
	Provenance string `json:"provenance"`
}

// valueSetJSON mirrors ResolvedValueSet in customization-types.ts.
type valueSetJSON struct {
	ID         string               `json:"id"`
	Name       string               `json:"name"`
	Options    []valueSetOptionJSON `json:"options"`
	Provenance string               `json:"provenance"`
}

// valueSetsResponseJSON mirrors ValueSetsResponse (GET .../value-sets).
type valueSetsResponseJSON struct {
	ValueSets []valueSetJSON `json:"valueSets"`
}

// valueSetResponseJSON mirrors ValueSetResponse (GET/PUT .../value-sets/{id}).
type valueSetResponseJSON struct {
	ValueSet valueSetJSON `json:"valueSet"`
}

// valueSetToJSON converts the resolved proto message to the FE wire shape.
// Options is always a non-nil slice so an empty list serialises as `[]`.
func valueSetToJSON(vs *settingsv1.ValueSet) valueSetJSON {
	options := make([]valueSetOptionJSON, 0, len(vs.GetOptions()))
	for _, o := range vs.GetOptions() {
		options = append(options, valueSetOptionJSON{
			ID:         o.GetKey(),
			Label:      o.GetLabel(),
			Color:      o.GetColor(),
			Order:      o.GetOrder(),
			Active:     o.GetActive(),
			Provenance: o.GetProvenance(),
		})
	}
	return valueSetJSON{
		ID:         vs.GetKey(),
		Name:       vs.GetName(),
		Options:    options,
		Provenance: vs.GetProvenance(),
	}
}

// HandleListValueSets returns every value-set resolved for the caller's
// tenant. ?base=1 returns only the shipped baselines (the editor's "reset
// to" view) — tenant-only lists have no baseline and are omitted from that
// view, which is expected, not a bug.
func (cr *CustomizationRoutes) HandleListValueSets(w http.ResponseWriter, r *http.Request) {
	client, err := cr.getSettingsClient()
	if err != nil {
		respondServiceUnavailable(w, cr.ServiceName())
		return
	}
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	resp, err := client.ListValueSets(r.Context(), &settingsv1.ListValueSetsRequest{
		TenantId: tenantID.String(),
		Base:     r.URL.Query().Get("base") == "1",
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	valueSets := make([]valueSetJSON, 0, len(resp.GetValueSets()))
	for _, vs := range resp.GetValueSets() {
		valueSets = append(valueSets, valueSetToJSON(vs))
	}
	response.JSON(w, http.StatusOK, valueSetsResponseJSON{ValueSets: valueSets})
}

// HandleGetValueSet returns one resolved value-set by key. 404 when neither
// the shipped registry nor the tenant knows the key.
func (cr *CustomizationRoutes) HandleGetValueSet(w http.ResponseWriter, r *http.Request) {
	client, err := cr.getSettingsClient()
	if err != nil {
		respondServiceUnavailable(w, cr.ServiceName())
		return
	}
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	vs, err := client.GetValueSet(r.Context(), &settingsv1.GetValueSetRequest{
		TenantId: tenantID.String(),
		Key:      chi.URLParam(r, "id"),
		Base:     r.URL.Query().Get("base") == "1",
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, valueSetResponseJSON{ValueSet: valueSetToJSON(vs)})
}

// upsertValueSetOptionRequest is one option in the PUT body. Mirrors
// ValueSetOption in customization-types.ts — `id`, not `key`.
type upsertValueSetOptionRequest struct {
	ID     string `json:"id" validate:"required"`
	Label  string `json:"label" validate:"required"`
	Color  string `json:"color"`
	Order  int32  `json:"order"`
	Active bool   `json:"active"`
}

// upsertValueSetRequest is the HTTP body for PUT /customization/value-sets/{id}.
// Mirrors UpsertValueSetInput in customization-types.ts.
type upsertValueSetRequest struct {
	Layer   string                        `json:"layer" validate:"omitempty,oneof=tenant vendor"`
	Name    string                        `json:"name"`
	Options []upsertValueSetOptionRequest `json:"options" validate:"required,min=1,dive"`
}

// HandlePutValueSet replaces the tenant's override for one value-set: system
// keys (e.g. "ticket_priority") are overridden, unknown keys create a
// tenant-owned list — both are the same write server-side (Service.UpsertValueSet).
func (cr *CustomizationRoutes) HandlePutValueSet(w http.ResponseWriter, r *http.Request) {
	client, err := cr.getSettingsClient()
	if err != nil {
		respondServiceUnavailable(w, cr.ServiceName())
		return
	}
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	callerID := middleware.GetUserID(r.Context())
	if callerID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	req, ok := decodeAndValidate[upsertValueSetRequest](w, r)
	if !ok {
		return
	}
	// The vendor overlay layer has no writer yet (R-5 GDAP vendor access is
	// not wired) — same rule as the label overrides PUT.
	if req.Layer == "vendor" {
		response.Error(w, http.StatusBadRequest, "layer 'vendor' is not writable yet")
		return
	}

	// The mock defaults an omitted name to the set id (resolveValueSet in
	// mocks/data/customization.ts); the service rejects an empty name outright.
	name := req.Name
	key := chi.URLParam(r, "id")
	if name == "" {
		name = key
	}

	options := make([]*settingsv1.ValueSetOption, 0, len(req.Options))
	for _, o := range req.Options {
		options = append(options, &settingsv1.ValueSetOption{
			Key:    o.ID,
			Label:  o.Label,
			Color:  o.Color,
			Order:  o.Order,
			Active: o.Active,
		})
	}

	vs, err := client.UpsertValueSet(r.Context(), &settingsv1.UpsertValueSetRequest{
		TenantId:  tenantID.String(),
		UpdatedBy: callerID,
		ValueSet: &settingsv1.ValueSet{
			Key:     key,
			Name:    name,
			Options: options,
		},
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, valueSetResponseJSON{ValueSet: valueSetToJSON(vs)})
}
