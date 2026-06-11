package bexio

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// TestContactSyncer_NilContactService_NoOutboundPanic verifies that syncOutbound
// does not panic when the ContactService is nil (G2 fix).
func TestContactSyncer_NilContactService_NoOutboundPanic(t *testing.T) {
	syncer := &ContactSyncer{
		contacts:    nil, // intentionally nil — simulates un-wired CRM gRPC
		fieldMapper: NewFieldMapper(),
		lookupCache: NewLookupCache(),
		// client and repo are nil — syncOutbound must return before using them
	}

	result := &SyncResult{}
	// Must not panic. Uses t.Run to catch any panic via testing's recover.
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("syncOutbound panicked with nil ContactService: %v", r)
			}
		}()
		syncer.syncOutbound(
			context.Background(),
			uuid.New(),
			uuid.New(),
			time.Now().Add(-time.Hour),
			[]models.BexioFieldMappingEntry{},
			result,
		)
	}()

	// The result must not have accumulated any failures (guard skips silently)
	if result.ItemsFailed != 0 {
		t.Errorf("expected 0 failed items after nil-guard skip, got %d", result.ItemsFailed)
	}
}

// TestContactSyncer_NilContactService_NoInboundPanic verifies that syncInbound
// does not panic when the ContactService is nil (G2 fix, inbound path).
func TestContactSyncer_NilContactService_NoInboundPanic(t *testing.T) {
	syncer := &ContactSyncer{
		contacts:    nil,
		fieldMapper: NewFieldMapper(),
		lookupCache: NewLookupCache(),
	}

	result := &SyncResult{}
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("syncInbound panicked with nil ContactService: %v", r)
			}
		}()
		syncer.syncInbound(
			context.Background(),
			uuid.New(),
			uuid.New(),
			nil,
			[]models.BexioFieldMappingEntry{},
			result,
		)
	}()

	if result.ItemsFailed != 0 {
		t.Errorf("expected 0 failed items after nil-guard skip, got %d", result.ItemsFailed)
	}
}
