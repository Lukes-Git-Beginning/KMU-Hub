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

// HandleGlobalSearch fans out a search query to CRM, Documents, and Email
// in parallel with a 500ms timeout. Each module's failure is isolated and
// reported individually.
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

	// CRM search
	wg.Add(1)
	go func() {
		defer wg.Done()
		result := g.searchCRM(ctx, query, int32(limit))
		mu.Lock()
		results = append(results, result)
		mu.Unlock()
	}()

	// Document search
	wg.Add(1)
	go func() {
		defer wg.Done()
		result := g.searchDocuments(ctx, query, int32(limit))
		mu.Lock()
		results = append(results, result)
		mu.Unlock()
	}()

	// Email search
	wg.Add(1)
	go func() {
		defer wg.Done()
		result := g.searchEmail(ctx, query, int32(limit))
		mu.Lock()
		results = append(results, result)
		mu.Unlock()
	}()

	wg.Wait()

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"query":   query,
		"modules": results,
	})
}

func (g *GlobalSearchRoutes) searchCRM(ctx context.Context, query string, limit int32) globalSearchResult {
	conn, err := g.registry.GetConnection("crm")
	if err != nil {
		return globalSearchResult{Module: "crm", Error: "service unavailable"}
	}

	client := crmv1.NewCRMServiceClient(conn)
	resp, err := client.Search(ctx, &crmv1.SearchRequest{
		Query: query,
		Limit: limit,
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
