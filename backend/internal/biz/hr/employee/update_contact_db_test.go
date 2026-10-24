package employee

// UpdateEmployee against a real database: the six address and emergency-contact columns
// were silently dropped on the HR update path (200, nothing stored), and a fake repository
// cannot prove the UPDATE statement writes them.

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func strPtr(s string) *string { return &s }

func readContact(t *testing.T, pool *pgxpool.Pool, tenantID, profileID uuid.UUID) [6]string {
	t.Helper()
	var c [6]string
	err := pool.QueryRow(testutil.WithTenantCtx(context.Background(), tenantID),
		`SELECT emergency_contact_name, emergency_contact_phone, address_street,
		        address_city, address_postal_code, address_country
		   FROM hr_employee_profiles WHERE id = $1 AND tenant_id = $2`, profileID, tenantID,
	).Scan(&c[0], &c[1], &c[2], &c[3], &c[4], &c[5])
	if err != nil {
		t.Fatalf("read contact columns: %v", err)
	}
	return c
}

func TestUpdateEmployee_PersistsAddressAndEmergencyContact_RealSQL(t *testing.T) {
	pool, tenantID, ctx := newOffboardFixture(t)
	svc := newOffboardSvc(pool)
	userID := seedOffboardUser(t, pool, tenantID, "contact-"+uuid.NewString()[:8]+"@example.com")
	profileID := seedOffboardProfile(t, pool, tenantID, userID, nil)

	// All six fields land.
	_, err := svc.UpdateEmployee(ctx, profileID, UpdateEmployeeInput{
		EmergencyContactName:  strPtr("Erika Muster"),
		EmergencyContactPhone: strPtr("+49 30 1234567"),
		AddressStreet:         strPtr("Hauptstr. 1"),
		AddressCity:           strPtr("Berlin"),
		AddressPostalCode:     strPtr("10115"),
		AddressCountry:        strPtr("AT"),
	}, "admin")
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	want := [6]string{"Erika Muster", "+49 30 1234567", "Hauptstr. 1", "Berlin", "10115", "AT"}
	if got := readContact(t, pool, tenantID, profileID); got != want {
		t.Fatalf("stored = %v, want %v", got, want)
	}

	// An update that names none of them keeps them (nil = no change).
	if _, err = svc.UpdateEmployee(ctx, profileID, UpdateEmployeeInput{Department: strPtr("Sales")}, "admin"); err != nil {
		t.Fatalf("update without contact fields: %v", err)
	}
	if got := readContact(t, pool, tenantID, profileID); got != want {
		t.Fatalf("after an unrelated update stored = %v, want %v", got, want)
	}

	// An empty string clears a text field, and only that one.
	if _, err = svc.UpdateEmployee(ctx, profileID, UpdateEmployeeInput{AddressStreet: strPtr("")}, "admin"); err != nil {
		t.Fatalf("clear street: %v", err)
	}
	want[2] = ""
	if got := readContact(t, pool, tenantID, profileID); got != want {
		t.Fatalf("after clearing the street stored = %v, want %v", got, want)
	}
}

func TestUpdateEmployee_RejectsAMalformedCountryCode_RealSQL(t *testing.T) {
	pool, tenantID, ctx := newOffboardFixture(t)
	svc := newOffboardSvc(pool)
	userID := seedOffboardUser(t, pool, tenantID, "country-"+uuid.NewString()[:8]+"@example.com")
	profileID := seedOffboardProfile(t, pool, tenantID, userID, nil)
	before := readContact(t, pool, tenantID, profileID)

	for _, country := range []string{"", "D", "DEU"} {
		_, err := svc.UpdateEmployee(ctx, profileID, UpdateEmployeeInput{
			AddressCountry: strPtr(country), AddressCity: strPtr("Wien"),
		}, "admin")
		if !errors.Is(err, ErrInvalidCountry) {
			t.Fatalf("country %q: err = %v, want ErrInvalidCountry", country, err)
		}
	}
	if got := readContact(t, pool, tenantID, profileID); got != before {
		t.Fatalf("a refused update changed the row: %v -> %v", before, got)
	}
}
