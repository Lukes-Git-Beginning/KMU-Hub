//go:build e2e

package e2e

import (
	"net/http"
	"testing"
)

// hrGetEmployee reads one employee profile the way hr-client.ts hrEmployeeApi.get does
// (`{employee}`, snake_case) and returns the inner object.
func hrGetEmployee(t *testing.T, base, token, userID string) map[string]interface{} {
	t.Helper()
	resp, body := getJSON(t, base+"/api/v1/hr/employees/"+userID, token)
	requireStatus(t, resp, body, http.StatusOK)
	var wrapped struct {
		Employee map[string]interface{} `json:"employee"`
	}
	decodeBody(t, body, &wrapped)
	if wrapped.Employee == nil {
		t.Fatalf("expected a wrapped {employee}, got %s", string(body))
	}
	return wrapped.Employee
}

// hrRequireFields compares string fields of a profile. protojson omits empty strings, so an
// expected "" means "empty or absent".
func hrRequireFields(t *testing.T, emp map[string]interface{}, want map[string]string) {
	t.Helper()
	for key, exp := range want {
		got, _ := emp[key].(string)
		if got != exp {
			t.Errorf("employee.%s = %q, want %q (profile: %v)", key, got, exp, emp)
		}
	}
}

// hrRequirePartTime checks the contract type. The gateway writes the proto enum as its number
// (encoding/json on the pb struct, CONTRACT_PART_TIME = 2), hr-client.ts normaliseContractType
// reads both the number and the name.
func hrRequirePartTime(t *testing.T, emp map[string]interface{}) {
	t.Helper()
	if got := emp["contract_type"]; got != float64(2) && got != "CONTRACT_PART_TIME" {
		t.Errorf("employee.contract_type = %v, want CONTRACT_PART_TIME (2)", got)
	}
}

// TestHREmployeeEditDesktopBodyFlow is the admin side of the team page, with the bodies the
// desktop sends: the create wizard (CreateEmployeeWizard.tsx -> toSnakeCaseEmployeeBody), the
// edit dialog (EditMemberDialog.tsx -> toSnakeCaseUpdateBody) and the deactivate button
// (TeamPage.tsx, `{status}` only). After every write the profile is read back, so a field that
// the gateway drops on the way fails here instead of in front of the customer.
func TestHREmployeeEditDesktopBodyFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	adminToken, _ := registerAndLoginWithRole(t, base, "admin")
	memberToken, memberID := registerAndLoginWithRole(t, base, "member")
	_, managerID := registerAndLoginWithRole(t, base, "manager")

	t.Run("create with the wizard body but no user_id is refused", func(t *testing.T) {
		// The wizard sends identity fields (first_name, email, roles, ...) and no user_id; the
		// route files a profile for an EXISTING user and requires it. Open product decision
		// `decide-hr-employee-create-without-user-id`: the dialog picks an existing user and
		// sends user_id (variant b). Until then the wizard body ends in 400.
		resp, body := postJSON(t, base+"/api/v1/hr/employees", map[string]interface{}{
			"first_name":         "Wizard",
			"last_name":          "Neu",
			"email":              "wizard-neu@example.com",
			"temporary_password": "SecurePass123!",
			"roles":              []string{"member"},
			"contract_type":      "full_time",
			"work_days_per_week": 5,
			"annual_leave_days":  30,
			"workload_percent":   100,
			"start_date":         "2024-03-01",
			"address_country":    "DE",
			"send_invite_email":  true,
		}, adminToken)
		requireStatus(t, resp, body, http.StatusBadRequest)
	})

	t.Run("create with user_id stores address and emergency contact", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/hr/employees", map[string]interface{}{
			"user_id":                 memberID,
			"first_name":              "E2E",
			"last_name":               "Test",
			"department":              "Vertrieb",
			"position_title":          "Sachbearbeiter",
			"contract_type":           "full_time",
			"work_days_per_week":      5,
			"annual_leave_days":       28,
			"manager_user_id":         managerID,
			"start_date":              "2024-03-01",
			"location":                "Berlin",
			"address_street":          "Hauptstr. 1",
			"address_city":            "Berlin",
			"address_postal_code":     "10115",
			"address_country":         "DE",
			"emergency_contact_name":  "Erika Muster",
			"emergency_contact_phone": "+49 30 1234567",
			"send_invite_email":       false,
		}, adminToken)
		requireStatus(t, resp, body, http.StatusCreated)
		var wrapped struct {
			Employee map[string]interface{} `json:"employee"`
		}
		decodeBody(t, body, &wrapped)
		if wrapped.Employee == nil {
			t.Fatalf("expected a wrapped {employee}, got %s", string(body))
		}

		emp := hrGetEmployee(t, base, adminToken, memberID)
		hrRequireFields(t, emp, map[string]string{
			"user_id":                memberID,
			"department":             "Vertrieb",
			"position_title":         "Sachbearbeiter",
			"manager_user_id":        managerID,
			"start_date":             "2024-03-01",
			"address_street":         "Hauptstr. 1",
			"address_city":           "Berlin",
			"address_postal_code":    "10115",
			"address_country":        "DE",
			"emergency_contact_name": "Erika Muster",
			"status":                 "active",
		})
		if phone, _ := emp["emergency_contact_phone"].(string); phone == "" {
			t.Errorf("emergency_contact_phone was dropped on create: %v", emp)
		}
	})

	t.Run("a second profile for the same user is a conflict", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/hr/employees", map[string]interface{}{
			"user_id":    memberID,
			"start_date": "2024-03-01",
		}, adminToken)
		if resp.StatusCode != http.StatusConflict && resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 409 (or 400), got %d: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("edit dialog body changes every field it sends", func(t *testing.T) {
		// EditMemberDialog.handleSave: empty inputs are sent as undefined (key absent), the
		// contract type always; no work_days_per_week, annual_leave_days, start_date or status.
		resp, body := putJSON(t, base+"/api/v1/hr/employees/"+memberID, map[string]interface{}{
			"department":              "Innendienst",
			"position_title":          "Teamleitung",
			"contract_type":           "part_time",
			"manager_user_id":         managerID,
			"address_street":          "Marienplatz 2",
			"address_city":            "München",
			"address_postal_code":     "80331",
			"address_country":         "DE",
			"emergency_contact_name":  "Max Muster",
			"emergency_contact_phone": "+49 89 7654321",
		}, adminToken)
		requireStatus(t, resp, body, http.StatusOK)

		emp := hrGetEmployee(t, base, adminToken, memberID)
		hrRequireFields(t, emp, map[string]string{
			"department":             "Innendienst",
			"position_title":         "Teamleitung",
			"address_street":         "Marienplatz 2",
			"address_city":           "München",
			"address_postal_code":    "80331",
			"address_country":        "DE",
			"emergency_contact_name": "Max Muster",
			// Not in the dialog body: must survive the write untouched.
			"start_date": "2024-03-01",
			"status":     "active",
		})
		hrRequirePartTime(t, emp)
		if days := emp["annual_leave_days"]; days != float64(28) {
			t.Errorf("annual_leave_days = %v, want 28 (not part of the body)", days)
		}
		if phone, _ := emp["emergency_contact_phone"].(string); phone == "" || phone == "+49 30 1234567" {
			t.Errorf("emergency_contact_phone not updated: %q", phone)
		}
	})

	t.Run("a partial body leaves the other fields alone", func(t *testing.T) {
		resp, body := putJSON(t, base+"/api/v1/hr/employees/"+memberID, map[string]interface{}{
			"address_city": "Köln",
		}, adminToken)
		requireStatus(t, resp, body, http.StatusOK)

		emp := hrGetEmployee(t, base, adminToken, memberID)
		hrRequireFields(t, emp, map[string]string{
			"address_city":           "Köln",
			"address_street":         "Marienplatz 2",
			"address_postal_code":    "80331",
			"emergency_contact_name": "Max Muster",
			"department":             "Innendienst",
		})
	})

	t.Run("an empty string clears a field", func(t *testing.T) {
		resp, body := putJSON(t, base+"/api/v1/hr/employees/"+memberID, map[string]interface{}{
			"address_street":         "",
			"emergency_contact_name": "",
		}, adminToken)
		requireStatus(t, resp, body, http.StatusOK)

		emp := hrGetEmployee(t, base, adminToken, memberID)
		hrRequireFields(t, emp, map[string]string{
			"address_street":         "",
			"emergency_contact_name": "",
			"address_city":           "Köln",
			"address_postal_code":    "80331",
		})
	})

	t.Run("deactivate button body is refused with a hint and changes nothing", func(t *testing.T) {
		// TeamPage.tsx sends `{status: 'inactive'}` / `{status: 'active'}`; leaving means
		// offboarding (exit date and type are mandatory), so the route answers 400 instead of
		// a false 200.
		for _, status := range []string{"inactive", "active"} {
			resp, body := putJSON(t, base+"/api/v1/hr/employees/"+memberID, map[string]interface{}{
				"status": status,
			}, adminToken)
			requireStatus(t, resp, body, http.StatusBadRequest)
		}
		hrRequireFields(t, hrGetEmployee(t, base, adminToken, memberID), map[string]string{"status": "active"})
	})

	t.Run("invalid values are refused and nothing is written", func(t *testing.T) {
		for name, patch := range map[string]map[string]interface{}{
			"postal code":   {"address_postal_code": "ABC"},
			"country code":  {"address_country": "DEU"},
			"contract type": {"contract_type": "freelance"},
			"phone":         {"emergency_contact_phone": "nicht-eine-nummer"},
		} {
			resp, body := putJSON(t, base+"/api/v1/hr/employees/"+memberID, patch, adminToken)
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("%s: expected 400, got %d: %s", name, resp.StatusCode, string(body))
			}
		}
		hrRequireFields(t, hrGetEmployee(t, base, adminToken, memberID), map[string]string{
			"address_postal_code": "80331",
			"address_country":     "DE",
		})
		hrRequirePartTime(t, hrGetEmployee(t, base, adminToken, memberID))
	})

	t.Run("update for an unknown user is a 404", func(t *testing.T) {
		resp, body := putJSON(t, base+"/api/v1/hr/employees/00000000-0000-4000-8000-000000000000", map[string]interface{}{
			"department": "Nirgends",
		}, adminToken)
		requireStatus(t, resp, body, http.StatusNotFound)
	})

	t.Run("member cannot edit a profile and sees no address of a colleague", func(t *testing.T) {
		resp, body := putJSON(t, base+"/api/v1/hr/employees/"+managerID, map[string]interface{}{
			"department": "Gehackt",
		}, memberToken)
		requireStatus(t, resp, body, http.StatusForbidden)

		// A colleague's profile window carries the directory projection only.
		emp := hrGetEmployee(t, base, memberToken, memberID)
		for _, key := range []string{"address_city", "address_postal_code", "emergency_contact_phone", "annual_leave_days"} {
			if v, present := emp[key]; present && v != "" && v != float64(0) {
				t.Errorf("member sees %s = %v of a colleague's profile", key, v)
			}
		}
	})
}
