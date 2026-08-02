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
	// Additive RBAC guards (RequirePermissionAny keeps the legacy coarse token
	// valid while granting the capability-catalog.ts fine keys, per the FE gate
	// calls that reach each route). Companies and activities carry their own
	// legacy resource but share the crm:contact:* fine keys, since the FE gates
	// firms/activities visibility and edits through the contact capability
	// (KontakteLayout.tsx nav, ActivitiesListPage.tsx canEdit/canDelete).
	contactRead := middleware.RequirePermissionAny([2]string{"contacts", "read"}, [2]string{"crm:contact", "read"})
	contactCreate := middleware.RequirePermissionAny([2]string{"contacts", "write"}, [2]string{"crm:contact", "create"})
	contactEdit := middleware.RequirePermissionAny([2]string{"contacts", "write"}, [2]string{"crm:contact", "edit"})
	contactDelete := middleware.RequirePermissionAny([2]string{"contacts", "delete"}, [2]string{"crm:contact", "delete"})
	contactImport := middleware.RequirePermissionAny([2]string{"contacts", "write"}, [2]string{"crm:import", "run"})
	contactExport := middleware.RequirePermissionAny([2]string{"contacts", "read"}, [2]string{"crm:contact", "export"})

	companyRead := middleware.RequirePermissionAny([2]string{"companies", "read"}, [2]string{"crm:contact", "read"})
	companyCreate := middleware.RequirePermissionAny([2]string{"companies", "write"}, [2]string{"crm:contact", "create"})
	companyEdit := middleware.RequirePermissionAny([2]string{"companies", "write"}, [2]string{"crm:contact", "edit"})
	companyDelete := middleware.RequirePermissionAny([2]string{"companies", "delete"}, [2]string{"crm:contact", "delete"})

	dealRead := middleware.RequirePermissionAny([2]string{"deals", "read"}, [2]string{"crm:deal", "read"})
	dealCreate := middleware.RequirePermissionAny([2]string{"deals", "write"}, [2]string{"crm:deal", "create"})
	dealEdit := middleware.RequirePermissionAny([2]string{"deals", "write"}, [2]string{"crm:deal", "edit"})
	dealDelete := middleware.RequirePermissionAny([2]string{"deals", "delete"}, [2]string{"crm:deal", "delete"})

	activityRead := middleware.RequirePermissionAny([2]string{"activities", "read"}, [2]string{"crm:contact", "read"})
	activityEdit := middleware.RequirePermissionAny([2]string{"activities", "write"}, [2]string{"crm:contact", "edit"})
	activityDelete := middleware.RequirePermissionAny([2]string{"activities", "delete"}, [2]string{"crm:contact", "delete"})

	// "auswertungen" tab (pipeline/conversion/activity reports) is gated by
	// crm:deal:read in KontakteLayout.tsx, not crm:contact:read.
	reportRead := middleware.RequirePermissionAny([2]string{"reports", "read"}, [2]string{"crm:deal", "read"})

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
		r.With(contactRead).Get("/", c.HandleListContacts)
		r.With(contactRead).Get("/{id}", c.HandleGetContact)
		r.With(contactCreate).Post("/", c.HandleCreateContact)
		r.With(contactEdit).Put("/{id}", c.HandleUpdateContact)
		r.With(contactDelete).Delete("/{id}", c.HandleDeleteContact)
		r.With(contactEdit).Post("/{id}/tags", c.HandleAddContactTags)
		r.With(contactEdit).Delete("/{id}/tags", c.HandleRemoveContactTags)
		r.With(middleware.RequirePermission("contacts", "write")).Put("/{id}/visibility", c.HandleUpdateContactVisibility)

		// File attachments (via the Document service's generic entity-link
		// mechanism, not a second CRM-specific file store).
		r.With(contactRead).Get("/{id}/files", c.HandleListContactFiles)
		r.With(contactEdit).Post("/{id}/files", c.HandleCreateContactFile)

		// Import/Export
		r.With(contactImport).Post("/import/csv", c.HandleImportContactsCSV)
		r.With(contactImport).Post("/import/vcard", c.HandleImportContactsVCard)
		r.With(contactImport).Post("/import/preview", c.HandlePreviewImportCSV)
		r.With(contactExport).Post("/export/csv", c.HandleExportContactsCSV)
		r.With(contactExport).Post("/export/vcard", c.HandleExportContactsVCard)

		// Extension features: duplicate detection, timeline, consent management
		if c.ext != nil {
			c.ext.registerContactExtRoutes(r)
		}
	})

	// Leads (contact lifecycle stage -- same rows, same permissions as contacts)
	r.Route("/api/v1/leads", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(contactRead).Get("/", c.HandleListLeads)
		r.With(contactCreate).Post("/", c.HandleCreateLead)
		r.With(contactEdit).Patch("/{id}", c.HandleUpdateLead)
		r.With(contactEdit).Post("/{id}/convert", c.HandleConvertLead)
	})

	// Companies
	r.Route("/api/v1/companies", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(companyRead).Get("/", c.HandleListCompanies)
		r.With(companyRead).Get("/{id}", c.HandleGetCompany)
		r.With(companyRead).Get("/{id}/contacts", c.HandleGetCompanyContacts)
		r.With(companyCreate).Post("/", c.HandleCreateCompany)
		r.With(companyEdit).Put("/{id}", c.HandleUpdateCompany)
		r.With(companyDelete).Delete("/{id}", c.HandleDeleteCompany)

		// Extension features: duplicate detection
		if c.ext != nil {
			c.ext.registerCompanyExtRoutes(r)
		}
	})

	// Pipeline Stages — crm:pipeline:manage has no FE gate call (PipelineStagesEditor.tsx
	// mounts unguarded); legacy-only, not tightened here.
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
		r.With(dealRead).Get("/", c.HandleListDeals)
		r.With(dealRead).Get("/{id}", c.HandleGetDeal)
		r.With(dealCreate).Post("/", c.HandleCreateDeal)
		r.With(dealEdit).Put("/{id}", c.HandleUpdateDeal)
		r.With(dealEdit).Post("/{id}/stage", c.HandleMoveDealToStage)
		r.With(middleware.RequirePermission("deals", "write")).Post("/{id}/tags", c.HandleAddDealTags)
		r.With(middleware.RequirePermission("deals", "write")).Delete("/{id}/tags", c.HandleRemoveDealTags)
		r.With(dealDelete).Delete("/{id}", c.HandleDeleteDeal)
	})

	// Activities
	r.Route("/api/v1/activities", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(activityRead).Get("/", c.HandleListActivities)
		r.With(activityRead).Get("/{id}", c.HandleGetActivity)
		r.With(activityEdit).Post("/", c.HandleCreateActivity)
		r.With(activityEdit).Put("/{id}", c.HandleUpdateActivity)
		r.With(activityEdit).Post("/{id}/complete", c.HandleCompleteActivity)
		r.With(middleware.RequirePermission("activities", "write")).Post("/{id}/tags", c.HandleAddActivityTags)
		r.With(middleware.RequirePermission("activities", "write")).Delete("/{id}/tags", c.HandleRemoveActivityTags)
		r.With(activityDelete).Delete("/{id}", c.HandleDeleteActivity)
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
		r.With(reportRead).Get("/pipeline", c.HandleGetPipelineReport)
		r.With(reportRead).Get("/conversion", c.HandleGetConversionReport)
		r.With(reportRead).Get("/activities", c.HandleGetActivityReport)
	})

	// GDPR deletion requests (top-level, admin only)
	if c.ext != nil {
		c.ext.registerGDPRRoutes(r, authMiddleware)
	}
}
