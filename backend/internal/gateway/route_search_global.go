package gateway

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
	documentv1 "github.com/kmuhub/kmuhub/proto/document/v1"
)

// globalSearchTimeout is the maximum time to wait for all search backends.
const globalSearchTimeout = 500 * time.Millisecond

// GlobalSearchRoutes handles the global search endpoint that fans out
// to CRM, Chat, Documents, Work, and Email services in parallel.
type GlobalSearchRoutes struct {
	registry *ServiceRegistry
}

// NewGlobalSearchRoutes creates a new GlobalSearchRoutes.
func NewGlobalSearchRoutes(registry *ServiceRegistry) *GlobalSearchRoutes {
	return &GlobalSearchRoutes{registry: registry}
}

// ServiceName returns the service name.
func (g *GlobalSearchRoutes) ServiceName() string { return "global-search" }

// RegisterRoutes registers the global search endpoint.
func (g *GlobalSearchRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/search/global", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("search", "read")).Get("/", g.HandleGlobalSearch)
	})
}

// globalSearchResult holds results from one search module.
type globalSearchResult struct {
	Module  string      `json:"module"`
	Results interface{} `json:"results"`
	Total   int32       `json:"total"`
	Error   string      `json:"error,omitempty"`
}

// crmSearchReadKeys maps each CRM search entity type to the read keys of the
// list route that shows the same rows (contactRead, companyRead, dealRead,
// activityRead in route_crm.go). A search hit is a row of that list, so a caller
// who may not open the list must not find its rows through a search either.
var crmSearchReadKeys = []struct {
	entityType string
	keys       [][2]string
}{
	{"contact", [][2]string{{"contacts", "read"}, {"crm:contact", "read"}}},
	{"company", [][2]string{{"companies", "read"}, {"crm:contact", "read"}}},
	{"deal", [][2]string{{"deals", "read"}, {"crm:deal", "read"}}},
	{"activity", [][2]string{{"activities", "read"}, {"crm:contact", "read"}}},
}

// Read keys of the other global-search sources, matching their own list routes
// (docRead in route_document.go, boxRead in route_email.go).
var (
	documentSearchReadKeys = [][2]string{{"documents", "read"}, {"documents:file", "read"}}
	emailSearchReadKeys    = [][2]string{{"email", "read"}, {"email:mailbox", "read"}}
)

func holdsAnyOf(ctx context.Context, keys [][2]string) bool {
	return middleware.HoldsAnyPermission(ctx, keys[0], keys[1:]...)
}

// readableCRMSearchTypes narrows the requested CRM entity types (none = every
// type) to those the caller may list. A type CRM does not know passes through
// for the CRM service to reject. An empty result means nothing is readable:
// never forward it, because an empty list asks CRM for every type.
func readableCRMSearchTypes(ctx context.Context, requested []string) []string {
	if len(requested) == 0 {
		for _, t := range crmSearchReadKeys {
			requested = append(requested, t.entityType)
		}
	}
	readable := make([]string, 0, len(requested))
	for _, et := range requested {
		allowed := true
		for _, t := range crmSearchReadKeys {
			if t.entityType == et {
				allowed = holdsAnyOf(ctx, t.keys)
				break
			}
		}
		if allowed {
			readable = append(readable, et)
		}
	}
	return readable
}

// HandleGlobalSearch fans out a search query to CRM, Documents, and Email
// in parallel with a 500ms timeout. Each module's failure is isolated and
// reported individually. search:read only opens the route: a source the caller
// may not read in its own module is left out of the response, and CRM is asked
// only for the entity types the caller may list.
func (g *GlobalSearchRoutes) HandleGlobalSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		response.Error(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}

	limit := 5
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 20 {
			limit = parsed
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), globalSearchTimeout)
	defer cancel()

	var mu sync.Mutex
	results := make([]globalSearchResult, 0, 5)

	var wg sync.WaitGroup

	fanOut := func(search func() globalSearchResult) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := search()
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}()
	}

	if crmTypes := readableCRMSearchTypes(ctx, nil); len(crmTypes) > 0 {
		fanOut(func() globalSearchResult { return g.searchCRM(ctx, query, int32(limit), crmTypes) })
	}
	if holdsAnyOf(ctx, documentSearchReadKeys) {
		fanOut(func() globalSearchResult { return g.searchDocuments(ctx, query, int32(limit)) })
	}
	if holdsAnyOf(ctx, emailSearchReadKeys) {
		fanOut(func() globalSearchResult { return g.searchEmail(ctx, query, int32(limit)) })
	}

	wg.Wait()

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"query":   query,
		"modules": results,
	})
}

func (g *GlobalSearchRoutes) searchCRM(ctx context.Context, query string, limit int32, entityTypes []string) globalSearchResult {
	conn, err := g.registry.GetConnection("crm")
	if err != nil {
		return globalSearchResult{Module: "crm", Error: "service unavailable"}
	}

	client := crmv1.NewCRMServiceClient(conn)
	resp, err := client.Search(ctx, &crmv1.SearchRequest{
		Query:       query,
		EntityTypes: entityTypes,
		Limit:       limit,
	})
	if err != nil {
		return globalSearchResult{Module: "crm", Error: "search failed"}
	}

	return globalSearchResult{
		Module:  "crm",
		Results: resp.Results,
		Total:   int32(len(resp.Results)),
	}
}

func (g *GlobalSearchRoutes) searchDocuments(ctx context.Context, query string, limit int32) globalSearchResult {
	conn, err := g.registry.GetConnection("document")
	if err != nil {
		return globalSearchResult{Module: "documents", Error: "service unavailable"}
	}

	client := documentv1.NewDocumentServiceClient(conn)
	resp, err := client.SearchFiles(ctx, &documentv1.SearchFilesRequest{
		Query:   query,
		Page:    1,
		PerPage: limit,
	})
	if err != nil {
		return globalSearchResult{Module: "documents", Error: "search failed"}
	}

	return globalSearchResult{
		Module:  "documents",
		Results: resp.Results,
		Total:   resp.Total,
	}
}

func (g *GlobalSearchRoutes) searchEmail(_ context.Context, _ string, _ int32) globalSearchResult {
	// Email search RPC does not exist yet. Return empty results gracefully.
	// This will be implemented when email search is added (candidate for Phase 14 Unified Inbox).
	return globalSearchResult{
		Module:  "email",
		Results: []interface{}{},
		Total:   0,
	}
}
