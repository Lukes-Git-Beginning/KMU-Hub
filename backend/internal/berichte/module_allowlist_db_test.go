package berichte

// The Report Builder saves the module of its data source. These tests pin the
// module allowlist to the desktop's ReportModule list end to end: service
// validation plus the CHECK constraints from 000354, against the real DB.

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// desktopReportModules mirrors ReportModule in
// desktop/src/renderer/src/api/berichte-types.ts.
var desktopReportModules = []string{
	"finanzen", "crm", "helpdesk", "inventar", "produktion", "work", "kommunikation",
	"hr", "zeiterfassung", "vertraege", "einkauf", "fuhrpark", "rapporte", "cross",
}

func TestService_EveryDesktopModuleIsStored(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Berichte Module Allowlist Tenant")
	ctx := testutil.WithTenantCtx(context.Background(), tenant)
	svc := NewService(NewPostgresRepository(pool), nil, Options{})
	builder := json.RawMessage(`{"kind":"builder","sourceId":"work","viz":"bar","dimensions":[],"measures":[],"filters":[]}`)

	for _, module := range desktopReportModules {
		t.Run(module, func(t *testing.T) {
			def, err := svc.CreateDefinition(ctx, CreateDefinitionInput{
				TenantID: tenant, Name: "Builder " + module, Module: module, QueryConfig: builder,
			})
			if err != nil {
				t.Fatalf("create definition: %v", err)
			}
			t.Cleanup(func() { testutil.CleanupRow(t, pool, "report_definitions", def.ID) })
			got, err := svc.GetDefinition(ctx, tenant, def.ID)
			if err != nil || got.Module != module {
				t.Fatalf("stored definition module = %v (err %v), want %q", got, err, module)
			}

			doc, err := svc.CreateDocument(ctx, CreateDocumentInput{TenantID: tenant, Title: "Doc " + module, Module: module})
			if err != nil {
				t.Fatalf("create document: %v", err)
			}
			t.Cleanup(func() { testutil.CleanupRow(t, pool, "report_documents", doc.ID) })
			if doc.Module != module {
				t.Fatalf("document module = %q, want %q", doc.Module, module)
			}
		})
	}

	t.Run("an existing definition moves to a builder module", func(t *testing.T) {
		def, err := svc.CreateDefinition(ctx, CreateDefinitionInput{TenantID: tenant, Name: "Move", Module: "crm", QueryConfig: builder})
		if err != nil {
			t.Fatalf("create definition: %v", err)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "report_definitions", def.ID) })
		module := "fuhrpark"
		updated, err := svc.UpdateDefinition(ctx, UpdateDefinitionInput{TenantID: tenant, DefinitionID: def.ID, Module: &module})
		if err != nil || updated.Module != module {
			t.Fatalf("update definition module = %v (err %v), want %q", updated, err, module)
		}
	})

	t.Run("an unknown module stays refused", func(t *testing.T) {
		_, err := svc.CreateDefinition(ctx, CreateDefinitionInput{TenantID: tenant, Name: "Bogus", Module: "bogus", QueryConfig: builder})
		if !errors.Is(err, ErrInvalidModule) {
			t.Fatalf("definition with module bogus: err = %v, want ErrInvalidModule", err)
		}
		_, err = svc.CreateDocument(ctx, CreateDocumentInput{TenantID: tenant, Title: "Bogus", Module: "bogus"})
		if !errors.Is(err, ErrInvalidModule) {
			t.Fatalf("document with module bogus: err = %v, want ErrInvalidModule", err)
		}
	})
}

// The builder modules are not KPI modules: the executor computes dashboard KPIs
// for the six executor modules only, everything else is dropped.
func TestService_GetDashboardKPIs_DropsBuilderOnlyModules(t *testing.T) {
	exec := &mockExecutor{}
	svc, _, _ := newServiceForTest(exec)
	if _, err := svc.GetDashboardKPIs(context.Background(), testTenant, []string{"hr", "crm", "fuhrpark"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exec.kpiModules) != 1 || exec.kpiModules[0] != "crm" {
		t.Fatalf("KPI modules = %v, want [crm]", exec.kpiModules)
	}
}
