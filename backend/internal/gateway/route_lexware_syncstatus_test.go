package gateway

import (
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// The sync dashboard reads {contact_sync, invoice_sync, quote_sync} with
// {status, items_synced, items_failed, last_sync_at} (LexwareSyncDashboard.tsx);
// the flat fields stay for existing readers.
func TestLexwareSyncStatusBody_PerEntityCards(t *testing.T) {
	synced := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	card := func(t *testing.T, body map[string]interface{}, key string) map[string]interface{} {
		t.Helper()
		c, ok := body[key].(map[string]interface{})
		if !ok {
			t.Fatalf("%s missing or not an object: %#v", key, body[key])
		}
		return c
	}

	t.Run("nothing synced yet is idle", func(t *testing.T) {
		body := lexwareSyncStatusBody(&bizv1.GetLexwareSyncStatusResponse{})
		for _, k := range []string{"contact_sync", "invoice_sync", "quote_sync"} {
			c := card(t, body, k)
			if c["status"] != "idle" || c["items_synced"] != int32(0) || c["items_failed"] != 0 {
				t.Errorf("%s = %#v, want idle/0/0", k, c)
			}
			if _, has := c["last_sync_at"]; has {
				t.Errorf("%s carries last_sync_at without a sync", k)
			}
		}
	})

	t.Run("mapped counts show up as synced items", func(t *testing.T) {
		body := lexwareSyncStatusBody(&bizv1.GetLexwareSyncStatusResponse{
			TotalContactsMapped: 12, TotalInvoicesMapped: 3, TotalQuotesMapped: 1,
			LastContactSyncAt: timestamppb.New(synced),
		})
		c := card(t, body, "contact_sync")
		if c["status"] != "completed" || c["items_synced"] != int32(12) || c["last_sync_at"] != synced {
			t.Errorf("contact_sync = %#v", c)
		}
		if i := card(t, body, "invoice_sync"); i["status"] != "completed" || i["items_synced"] != int32(3) {
			t.Errorf("invoice_sync = %#v", i)
		}
		if q := card(t, body, "quote_sync"); q["items_synced"] != int32(1) {
			t.Errorf("quote_sync = %#v", q)
		}
		if body["total_contacts_mapped"] != int32(12) {
			t.Errorf("flat total_contacts_mapped lost: %#v", body["total_contacts_mapped"])
		}
	})

	t.Run("an error newer than the last sync marks contacts failed", func(t *testing.T) {
		body := lexwareSyncStatusBody(&bizv1.GetLexwareSyncStatusResponse{
			LastContactSyncAt: timestamppb.New(synced),
			LastSyncError:     "401 from Lexware",
			LastSyncErrorAt:   timestamppb.New(synced.Add(time.Hour)),
		})
		if c := card(t, body, "contact_sync"); c["status"] != "failed" {
			t.Errorf("contact_sync = %#v, want failed", c)
		}
		if body["last_sync_error"] != "401 from Lexware" {
			t.Errorf("flat last_sync_error lost: %#v", body["last_sync_error"])
		}
	})

	t.Run("an error older than the last sync is superseded", func(t *testing.T) {
		body := lexwareSyncStatusBody(&bizv1.GetLexwareSyncStatusResponse{
			LastContactSyncAt: timestamppb.New(synced),
			LastSyncError:     "timeout",
			LastSyncErrorAt:   timestamppb.New(synced.Add(-time.Hour)),
		})
		if c := card(t, body, "contact_sync"); c["status"] != "completed" {
			t.Errorf("contact_sync = %#v, want completed", c)
		}
	})
}
