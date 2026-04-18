package migrations_test

import (
	"os"
	"strings"
	"testing"
)

// TestMigration000075ConsentSetNull verifies that migration 000075 correctly
// changes consent_records.contact_id from ON DELETE CASCADE to ON DELETE SET NULL,
// including the required NOT NULL → nullable column change.
func TestMigration000075ConsentSetNull(t *testing.T) {
	up, err := os.ReadFile("000075_consent_records_contact_set_null.up.sql")
	if err != nil {
		t.Fatalf("up migration not found: %v", err)
	}
	upSQL := string(up)

	down, err := os.ReadFile("000075_consent_records_contact_set_null.down.sql")
	if err != nil {
		t.Fatalf("down migration not found: %v", err)
	}
	downSQL := string(down)

	t.Run("up drops existing FK constraint", func(t *testing.T) {
		if !strings.Contains(upSQL, "DROP CONSTRAINT consent_records_contact_id_fkey") {
			t.Error("up.sql must DROP CONSTRAINT consent_records_contact_id_fkey before re-adding")
		}
	})

	t.Run("up removes NOT NULL so SET NULL can work", func(t *testing.T) {
		if !strings.Contains(upSQL, "DROP NOT NULL") {
			t.Error("up.sql must ALTER COLUMN contact_id DROP NOT NULL (required for ON DELETE SET NULL)")
		}
	})

	t.Run("up adds FK with ON DELETE SET NULL", func(t *testing.T) {
		if !strings.Contains(upSQL, "ON DELETE SET NULL") {
			t.Error("up.sql must add FK constraint with ON DELETE SET NULL")
		}
		if strings.Contains(upSQL, "ON DELETE CASCADE") {
			t.Error("up.sql must not contain ON DELETE CASCADE")
		}
	})

	t.Run("down restores ON DELETE CASCADE", func(t *testing.T) {
		if !strings.Contains(downSQL, "ON DELETE CASCADE") {
			t.Error("down.sql must restore ON DELETE CASCADE")
		}
	})

	t.Run("down cleans NULL rows before restoring NOT NULL", func(t *testing.T) {
		if !strings.Contains(downSQL, "DELETE FROM consent_records WHERE contact_id IS NULL") {
			t.Error("down.sql must delete NULL contact_id rows before restoring NOT NULL constraint")
		}
	})

	t.Run("down restores NOT NULL constraint", func(t *testing.T) {
		if !strings.Contains(downSQL, "SET NOT NULL") {
			t.Error("down.sql must restore NOT NULL on contact_id")
		}
	})
}
