package repository_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/plugin/repository"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestIndustryTemplate_CreateGetListBySlug covers the global (non-tenant)
// catalogue repository: Create's ON CONFLICT(slug) upsert, both lookup
// paths (including the NotFound-shaped nil result GetBySlug must return for
// an unknown slug), and List's optional industry filter. Migration 000058
// ships real rows ("handwerk", "beratung", "handel", ...) so this test uses
// an industry value no seed row carries to keep the List assertion exact.
func TestIndustryTemplate_CreateGetListBySlug(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	repo := repository.NewIndustryTemplateRepository(pool)
	ctx := testutil.WithSystemCtx(context.Background())

	const slug = "repository-gaps-test-template"
	const industry = "repository-gaps-test-industry"
	now := time.Now()
	tmpl := &models.IndustryTemplate{
		ID: uuid.New(), Slug: slug, Name: "Repository Gaps Test Template",
		Description: "seeded by c-cov-plugin-repository-gaps", Industry: industry, Icon: "building",
		CustomFields: json.RawMessage(`[]`), ValidationRules: json.RawMessage(`[]`),
		WorkflowRules: json.RawMessage(`[]`), DefaultSettings: json.RawMessage(`{}`),
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.Create(ctx, tmpl); err != nil {
		t.Fatalf("create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "industry_templates", tmpl.ID)

	byID, err := repo.GetByID(ctx, tmpl.ID)
	if err != nil || byID == nil || byID.Slug != slug {
		t.Fatalf("get by id: got %+v, err %v", byID, err)
	}

	bySlug, err := repo.GetBySlug(ctx, slug)
	if err != nil || bySlug == nil || bySlug.ID != tmpl.ID {
		t.Fatalf("get by slug: got %+v, err %v", bySlug, err)
	}

	if unknown, err := repo.GetBySlug(ctx, "does-not-exist-slug"); err != nil || unknown != nil {
		t.Fatalf("get by unknown slug: got %+v, err %v", unknown, err)
	}
	if unknown, err := repo.GetByID(ctx, uuid.New()); err != nil || unknown != nil {
		t.Fatalf("get by unknown id: got %+v, err %v", unknown, err)
	}

	// ON CONFLICT (slug) DO UPDATE — same slug with a new name upserts
	// rather than erroring or duplicating.
	tmpl.Name = "Repository Gaps Test Template Renamed"
	tmpl.Icon = "wrench"
	if err := repo.Create(ctx, tmpl); err != nil {
		t.Fatalf("upsert on conflict: %v", err)
	}
	renamed, err := repo.GetByID(ctx, tmpl.ID)
	if err != nil || renamed == nil || renamed.Name != "Repository Gaps Test Template Renamed" || renamed.Icon != "wrench" {
		t.Fatalf("expected upserted fields, got %+v (err %v)", renamed, err)
	}

	filtered, err := repo.List(ctx, industry)
	if err != nil || len(filtered) != 1 || filtered[0].ID != tmpl.ID {
		t.Fatalf("list filtered by industry: got %+v, err %v", filtered, err)
	}

	if noMatch, err := repo.List(ctx, "no-such-industry"); err != nil || len(noMatch) != 0 {
		t.Fatalf("list with unmatched industry filter: got %d, err %v", len(noMatch), err)
	}

	all, err := repo.List(ctx, "")
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	found := false
	for _, r := range all {
		if r.ID == tmpl.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("list without filter did not include the seeded template among %d rows", len(all))
	}
}
