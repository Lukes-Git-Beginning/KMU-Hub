package gateway

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

// CRMRoutes handles HTTP routes for the CRM backend service.
type CRMRoutes struct {
	registry *ServiceRegistry
	ext      *CRMExtRoutes // optional: direct-DB extension handlers
}

// NewCRMRoutes creates a new CRMRoutes with the given service registry.
// ext may be nil if extension features are not configured.
func NewCRMRoutes(registry *ServiceRegistry, ext *CRMExtRoutes) *CRMRoutes {
	return &CRMRoutes{registry: registry, ext: ext}
}

// ServiceName returns the backend service name.
func (c *CRMRoutes) ServiceName() string { return "crm" }

// getCRMClient lazily obtains a gRPC client for the CRM service.
func (c *CRMRoutes) getCRMClient() (crmv1.CRMServiceClient, error) {
	conn, err := c.registry.GetConnection("crm")
	if err != nil {
		return nil, err
	}
	return crmv1.NewCRMServiceClient(conn), nil
}

// RegisterRoutes registers all CRM HTTP routes.
func (c *CRMRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	// Custom Fields
	r.Route("/api/v1/custom-fields", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("custom_fields", "read")).Get("/", c.HandleListCustomFields)
		r.With(middleware.RequirePermission("custom_fields", "read")).Get("/{id}", c.HandleGetCustomField)
		r.With(middleware.RequirePermission("custom_fields", "write")).Post("/", c.HandleCreateCustomField)
		r.With(middleware.RequirePermission("custom_fields", "write")).Put("/{id}", c.HandleUpdateCustomField)
		r.With(middleware.RequirePermission("custom_fields", "delete")).Delete("/{id}", c.HandleDeleteCustomField)
	})

	// Tags
	r.Route("/api/v1/tags", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("tags", "read")).Get("/", c.HandleListTags)
		r.With(middleware.RequirePermission("tags", "read")).Get("/{id}", c.HandleGetTag)
		r.With(middleware.RequirePermission("tags", "write")).Post("/", c.HandleCreateTag)
		r.With(middleware.RequirePermission("tags", "write")).Put("/{id}", c.HandleUpdateTag)
		r.With(middleware.RequirePermission("tags", "delete")).Delete("/{id}", c.HandleDeleteTag)
	})

	// Contacts
	r.Route("/api/v1/contacts", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("contacts", "read")).Get("/", c.HandleListContacts)
		r.With(middleware.RequirePermission("contacts", "read")).Get("/{id}", c.HandleGetContact)
		r.With(middleware.RequirePermission("contacts", "write")).Post("/", c.HandleCreateContact)
		r.With(middleware.RequirePermission("contacts", "write")).Put("/{id}", c.HandleUpdateContact)
		r.With(middleware.RequirePermission("contacts", "delete")).Delete("/{id}", c.HandleDeleteContact)
		r.With(middleware.RequirePermission("contacts", "write")).Post("/{id}/tags", c.HandleAddContactTags)
		r.With(middleware.RequirePermission("contacts", "write")).Delete("/{id}/tags", c.HandleRemoveContactTags)
		r.With(middleware.RequirePermission("contacts", "write")).Put("/{id}/visibility", c.HandleUpdateContactVisibility)

		// Import/Export
		r.With(middleware.RequirePermission("contacts", "write")).Post("/import/csv", c.HandleImportContactsCSV)
		r.With(middleware.RequirePermission("contacts", "write")).Post("/import/vcard", c.HandleImportContactsVCard)
		r.With(middleware.RequirePermission("contacts", "write")).Post("/import/preview", c.HandlePreviewImportCSV)
		r.With(middleware.RequirePermission("contacts", "read")).Post("/export/csv", c.HandleExportContactsCSV)
		r.With(middleware.RequirePermission("contacts", "read")).Post("/export/vcard", c.HandleExportContactsVCard)

		// Extension features: duplicate detection, timeline, consent management
		if c.ext != nil {
			c.ext.registerContactExtRoutes(r)
		}
	})

	// Companies
	r.Route("/api/v1/companies", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("companies", "read")).Get("/", c.HandleListCompanies)
		r.With(middleware.RequirePermission("companies", "read")).Get("/{id}", c.HandleGetCompany)
		r.With(middleware.RequirePermission("companies", "read")).Get("/{id}/contacts", c.HandleGetCompanyContacts)
		r.With(middleware.RequirePermission("companies", "write")).Post("/", c.HandleCreateCompany)
		r.With(middleware.RequirePermission("companies", "write")).Put("/{id}", c.HandleUpdateCompany)
		r.With(middleware.RequirePermission("companies", "delete")).Delete("/{id}", c.HandleDeleteCompany)

		// Extension features: duplicate detection
		if c.ext != nil {
			c.ext.registerCompanyExtRoutes(r)
		}
	})

	// Pipeline Stages
	r.Route("/api/v1/pipeline-stages", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("pipeline_stages", "read")).Get("/", c.HandleListPipelineStages)
		r.With(middleware.RequirePermission("pipeline_stages", "read")).Get("/{id}", c.HandleGetPipelineStage)
		r.With(middleware.RequirePermission("pipeline_stages", "write")).Post("/", c.HandleCreatePipelineStage)
		r.With(middleware.RequirePermission("pipeline_stages", "write")).Put("/{id}", c.HandleUpdatePipelineStage)
		r.With(middleware.RequirePermission("pipeline_stages", "write")).Post("/reorder", c.HandleReorderPipelineStages)
		r.With(middleware.RequirePermission("pipeline_stages", "delete")).Delete("/{id}", c.HandleDeletePipelineStage)
	})

	// Deals
	r.Route("/api/v1/deals", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("deals", "read")).Get("/", c.HandleListDeals)
		r.With(middleware.RequirePermission("deals", "read")).Get("/{id}", c.HandleGetDeal)
		r.With(middleware.RequirePermission("deals", "write")).Post("/", c.HandleCreateDeal)
		r.With(middleware.RequirePermission("deals", "write")).Put("/{id}", c.HandleUpdateDeal)
		r.With(middleware.RequirePermission("deals", "write")).Post("/{id}/stage", c.HandleMoveDealToStage)
		r.With(middleware.RequirePermission("deals", "write")).Post("/{id}/tags", c.HandleAddDealTags)
		r.With(middleware.RequirePermission("deals", "write")).Delete("/{id}/tags", c.HandleRemoveDealTags)
		r.With(middleware.RequirePermission("deals", "delete")).Delete("/{id}", c.HandleDeleteDeal)
	})

	// Activities
	r.Route("/api/v1/activities", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("activities", "read")).Get("/", c.HandleListActivities)
		r.With(middleware.RequirePermission("activities", "read")).Get("/{id}", c.HandleGetActivity)
		r.With(middleware.RequirePermission("activities", "write")).Post("/", c.HandleCreateActivity)
		r.With(middleware.RequirePermission("activities", "write")).Put("/{id}", c.HandleUpdateActivity)
		r.With(middleware.RequirePermission("activities", "write")).Post("/{id}/complete", c.HandleCompleteActivity)
		r.With(middleware.RequirePermission("activities", "write")).Post("/{id}/tags", c.HandleAddActivityTags)
		r.With(middleware.RequirePermission("activities", "write")).Delete("/{id}/tags", c.HandleRemoveActivityTags)
		r.With(middleware.RequirePermission("activities", "delete")).Delete("/{id}", c.HandleDeleteActivity)
	})

	// Search
	r.Route("/api/v1/search", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Get("/", c.HandleSearch)
	})

	// Saved Filters
	r.Route("/api/v1/saved-filters", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("saved_filters", "read")).Get("/", c.HandleListSavedFilters)
		r.With(middleware.RequirePermission("saved_filters", "read")).Get("/{id}", c.HandleGetSavedFilter)
		r.With(middleware.RequirePermission("saved_filters", "write")).Post("/", c.HandleCreateSavedFilter)
		r.With(middleware.RequirePermission("saved_filters", "write")).Patch("/{id}", c.HandleUpdateSavedFilter)
		r.With(middleware.RequirePermission("saved_filters", "delete")).Delete("/{id}", c.HandleDeleteSavedFilter)
	})

	// Reports
	r.Route("/api/v1/reports", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("reports", "read")).Get("/pipeline", c.HandleGetPipelineReport)
		r.With(middleware.RequirePermission("reports", "read")).Get("/conversion", c.HandleGetConversionReport)
		r.With(middleware.RequirePermission("reports", "read")).Get("/activities", c.HandleGetActivityReport)
	})

	// GDPR deletion requests (top-level, admin only)
	if c.ext != nil {
		c.ext.registerGDPRRoutes(r, authMiddleware)
	}
}
