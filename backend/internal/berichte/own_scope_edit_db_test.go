package berichte

// berichte:reports:edit sits at scope own for member (000256): the gateway
// passes the caller as OwnerID, and an edit of a definition or document
// someone else created answers not found. These tests pin that against the
// real DB with real users as creators.

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestService_OwnScopeEdit(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close) // registered first, so it runs after the row cleanups

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Berichte Own Scope Edit Tenant")
	ctx := testutil.WithTenantCtx(context.Background(), tenant)
	svc := NewService(NewPostgresRepository(pool), nil, Options{})

	seedUser := func(email string) uuid.UUID {
		id := testutil.SeedRow(t, pool, "users", map[string]any{
			"id": uuid.New(), "tenant_id": tenant, "email": email, "password_hash": "x",
			"first_name": "Own", "last_name": "Scope", "is_active": true,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
		return id
	}
	author := seedUser("berichte-own-author-" + tenant.String() + "@test.local")
	stranger := seedUser("berichte-own-stranger-" + tenant.String() + "@test.local")

	def, err := svc.CreateDefinition(ctx, CreateDefinitionInput{
		TenantID: tenant, Name: "Author's report", Module: "crm", CreatedBy: &author,
	})
	if err != nil {
		t.Fatalf("create definition: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "report_definitions", def.ID) })

	doc, err := svc.CreateDocument(ctx, CreateDocumentInput{TenantID: tenant, Title: "Author's document", CreatedBy: &author})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "report_documents", doc.ID) })

	orphan, err := svc.CreateDocument(ctx, CreateDocumentInput{TenantID: tenant, Title: "Nobody's document"})
	if err != nil {
		t.Fatalf("create orphan document: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "report_documents", orphan.ID) })

	str := func(s string) *string { return &s }

	t.Run("a stranger at own scope gets not found and changes nothing", func(t *testing.T) {
		if _, err := svc.UpdateDefinition(ctx, UpdateDefinitionInput{
			TenantID: tenant, DefinitionID: def.ID, Name: str("hijacked"), OwnerID: &stranger,
		}); !errors.Is(err, ErrDefinitionNotFound) {
			t.Fatalf("definition update err = %v, want ErrDefinitionNotFound", err)
		}
		if _, err := svc.UpdateDocument(ctx, UpdateDocumentInput{
			TenantID: tenant, DocumentID: doc.ID, Title: str("hijacked"), OwnerID: &stranger,
		}); !errors.Is(err, ErrDocumentNotFound) {
			t.Fatalf("document update err = %v, want ErrDocumentNotFound", err)
		}
		gotDef, err := svc.GetDefinition(ctx, tenant, def.ID)
		if err != nil || gotDef.Name != "Author's report" {
			t.Fatalf("definition after refused edit = %v (err %v)", gotDef, err)
		}
		gotDoc, err := svc.GetDocument(ctx, tenant, doc.ID)
		if err != nil || gotDoc.Title != "Author's document" {
			t.Fatalf("document after refused edit = %v (err %v)", gotDoc, err)
		}
	})

	t.Run("a row without creator is foreign to an own-scoped editor", func(t *testing.T) {
		if _, err := svc.UpdateDocument(ctx, UpdateDocumentInput{
			TenantID: tenant, DocumentID: orphan.ID, Title: str("claimed"), OwnerID: &author,
		}); !errors.Is(err, ErrDocumentNotFound) {
			t.Fatalf("orphan update err = %v, want ErrDocumentNotFound", err)
		}
	})

	t.Run("the author edits their own definition and document", func(t *testing.T) {
		gotDef, err := svc.UpdateDefinition(ctx, UpdateDefinitionInput{
			TenantID: tenant, DefinitionID: def.ID, Name: str("Author's report v2"), OwnerID: &author,
		})
		if err != nil || gotDef.Name != "Author's report v2" {
			t.Fatalf("author definition update = %v (err %v)", gotDef, err)
		}
		gotDoc, err := svc.UpdateDocument(ctx, UpdateDocumentInput{
			TenantID: tenant, DocumentID: doc.ID, Title: str("Author's document v2"), OwnerID: &author,
		})
		if err != nil || gotDoc.Title != "Author's document v2" {
			t.Fatalf("author document update = %v (err %v)", gotDoc, err)
		}
	})

	t.Run("without owner filter the edit stays tenant-wide", func(t *testing.T) {
		gotDef, err := svc.UpdateDefinition(ctx, UpdateDefinitionInput{
			TenantID: tenant, DefinitionID: def.ID, Name: str("Manager's fix"),
		})
		if err != nil || gotDef.Name != "Manager's fix" {
			t.Fatalf("unfiltered definition update = %v (err %v)", gotDef, err)
		}
		gotDoc, err := svc.UpdateDocument(ctx, UpdateDocumentInput{
			TenantID: tenant, DocumentID: orphan.ID, Status: str("final"),
		})
		if err != nil || gotDoc.Status != "final" {
			t.Fatalf("unfiltered document update = %v (err %v)", gotDoc, err)
		}
	})
}
