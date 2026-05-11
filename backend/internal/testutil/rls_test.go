package testutil

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/sysctx"
)

func TestWithTenantCtx_StampsTenantIDInContext(t *testing.T) {
	ctx := WithTenantCtx(context.Background(), TenantA)
	got, err := middleware.GetTenantID(ctx)
	if err != nil {
		t.Fatalf("GetTenantID: %v", err)
	}
	if got != TenantA {
		t.Fatalf("expected tenant %s, got %s", TenantA, got)
	}
}

func TestWithSystemCtx_FlagsAsSystem(t *testing.T) {
	ctx := WithSystemCtx(context.Background())
	if !sysctx.Is(ctx) {
		t.Fatal("WithSystemCtx must produce a system-flagged context")
	}
}

func TestTenantConstants_AreDistinctAndNotZero(t *testing.T) {
	if TenantA == TenantB {
		t.Fatal("TenantA and TenantB must differ")
	}
	if TenantA == uuid.Nil || TenantB == uuid.Nil {
		t.Fatal("TenantA/B must not be zero UUID")
	}
}
