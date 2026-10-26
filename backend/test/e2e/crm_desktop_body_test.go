//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestContactAndDealDesktopBodyFlow sends the bodies the desktop really builds for contacts,
// companies and deals, not the hand-written minimal bodies of crm_test.go: every untouched
// profile field as "", `custom_fields: []`, optional fields left out. A validate tag or an
// omitempty oneof that rejects "" shows up here as a 400 where the unit test only decodes.
//
// Roles: member A works the contact book (crm:contact:create|edit, company create/edit); the
// admin owns the pipeline. A member writes the deals of their team (crm:deal:create|edit at scope
// team, decision 2026-10-25): their own here, the admin's deal stays read-only and deleting stays
// with manager/admin. The reporting-line half of the team sits in the gateway router test
// TestCRMDealTeamWrite_MemberAgainstRealSeed.
func TestContactAndDealDesktopBodyFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	adminToken, _ := registerAndLoginWithRole(t, base, "admin")
	memberToken, memberID := registerAndLoginWithRole(t, base, "member")

	suffix := time.Now().UnixNano()
	email := fmt.Sprintf("e2e-fe-contact-%d@example.com", suffix)
	lastName := fmt.Sprintf("Feldmann%d", suffix)

	// str reads an optional proto string: protojson drops "" and unset alike, so "cleared" and
	// "absent" are the same wire state.
	str := func(m map[string]interface{}, key string) string {
		s, _ := m[key].(string)
		return s
	}

	var contactID, companyID string

	t.Run("contact: create with the form body", func(t *testing.T) {
		// kontakte/adapters.ts coreContactFields + uiFormToWriteRequest (non-demo branch):
		// email/phone/notes are undefined when empty (left out), every profile field is a string
		// ("" when untouched), custom_fields is []. KontaktePage.tsx:192 posts it unchanged.
		resp, body := postJSON(t, base+"/api/v1/contacts", map[string]interface{}{
			"first_name":      "Franka",
			"last_name":       lastName,
			"email":           email,
			"phone":           "+49 170 1234567",
			"position":        "Einkaufsleiterin",
			"salutation":      "Frau",
			"title":           "Dr.",
			"mobile":          "+49 151 2345678",
			"department":      "Einkauf",
			"address_street":  "Hauptstrasse 1",
			"address_zip":     "80331",
			"address_city":    "Muenchen",
			"address_country": "Deutschland",
			"website":         "https://example.com",
			"linkedin":        "",
			"xing":            "",
			"category":        "customer",
			"status":          "active",
			"custom_fields":   []interface{}{},
		}, memberToken)
		requireStatus(t, resp, body, http.StatusCreated)

		var res struct {
			Contact map[string]interface{} `json:"contact"`
		}
		decodeBody(t, body, &res)
		contactID = str(res.Contact, "id")
		if contactID == "" {
			t.Fatalf("expected a contact id, got %s", string(body))
		}
		for key, want := range map[string]string{
			"first_name": "Franka", "last_name": lastName, "email": email, "position": "Einkaufsleiterin",
			"salutation": "Frau", "title": "Dr.", "department": "Einkauf", "address_city": "Muenchen",
			"category": "customer", "status": "active",
		} {
			if got := str(res.Contact, key); got != want {
				t.Errorf("created contact %s = %q, want %q (%s)", key, got, want, string(body))
			}
		}
		if str(res.Contact, "linkedin") != "" {
			t.Errorf("an empty linkedin must stay empty, got %q", str(res.Contact, "linkedin"))
		}
	})

	t.Run("contact: create with only the two name fields", func(t *testing.T) {
		// The minimal form: everything else "" or left out (coreContactFields on an empty form).
		resp, body := postJSON(t, base+"/api/v1/contacts", map[string]interface{}{
			"first_name": "Nur", "last_name": lastName + "Minimal", "position": "", "salutation": "",
			"title": "", "mobile": "", "department": "", "address_street": "", "address_zip": "",
			"address_city": "", "address_country": "", "website": "", "linkedin": "", "xing": "",
			"category": "", "status": "", "custom_fields": []interface{}{},
		}, memberToken)
		requireStatus(t, resp, body, http.StatusCreated)
	})

	t.Run("contact: update with the form body, emptied fields are cleared", func(t *testing.T) {
		// KontaktePage.tsx:188 — the edit dialog resends all fields; the user cleared mobile,
		// department and website and changed the name. email/phone are left out when the form
		// holds them empty, here they stay filled.
		resp, body := putJSON(t, base+"/api/v1/contacts/"+contactID, map[string]interface{}{
			"first_name":      "Franziska",
			"last_name":       lastName,
			"email":           email,
			"phone":           "+49 170 1234567",
			"position":        "Leiterin Einkauf",
			"salutation":      "Frau",
			"title":           "",
			"mobile":          "",
			"department":      "",
			"address_street":  "Hauptstrasse 1",
			"address_zip":     "80331",
			"address_city":    "Muenchen",
			"address_country": "Deutschland",
			"website":         "",
			"linkedin":        "",
			"xing":            "",
			"category":        "partner",
			"status":          "prospect",
			"custom_fields":   []interface{}{},
		}, memberToken)
		requireStatus(t, resp, body, http.StatusOK)

		resp, body = getJSON(t, base+"/api/v1/contacts/"+contactID, memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		var res struct {
			Contact map[string]interface{} `json:"contact"`
		}
		decodeBody(t, body, &res)
		for key, want := range map[string]string{
			"first_name": "Franziska", "position": "Leiterin Einkauf", "address_city": "Muenchen",
			"category": "partner", "status": "prospect", "email": email,
		} {
			if got := str(res.Contact, key); got != want {
				t.Errorf("updated contact %s = %q, want %q (%s)", key, got, want, string(body))
			}
		}
		for _, key := range []string{"title", "mobile", "department", "website"} {
			if got := str(res.Contact, key); got != "" {
				t.Errorf("contact %s must be cleared by \"\", still %q (%s)", key, got, string(body))
			}
		}
	})

	t.Run("contact: a value the form can not produce stays a 400", func(t *testing.T) {
		resp, body := putJSON(t, base+"/api/v1/contacts/"+contactID, map[string]interface{}{
			"category": "vip",
		}, memberToken)
		requireStatus(t, resp, body, http.StatusBadRequest)
	})

	t.Run("company: create and update with the form body", func(t *testing.T) {
		// api/hooks/useCompanies.ts toCompanyPayload (non-demo): name, domain (from the website
		// input), industry, address, notes, tag_ids undefined, custom_fields []. Fields that are
		// empty are undefined and therefore left out.
		resp, body := postJSON(t, base+"/api/v1/companies", map[string]interface{}{
			"name":          fmt.Sprintf("Feldmann Handels GmbH %d", suffix),
			"domain":        "feldmann-handel.example",
			"industry":      "Handel",
			"address":       "Hauptstrasse 1, 80331 Muenchen, Deutschland",
			"custom_fields": []interface{}{},
		}, memberToken)
		requireStatus(t, resp, body, http.StatusCreated)
		var res struct {
			Company map[string]interface{} `json:"company"`
		}
		decodeBody(t, body, &res)
		companyID = str(res.Company, "id")
		if companyID == "" {
			t.Fatalf("expected a company id, got %s", string(body))
		}
		if str(res.Company, "domain") != "feldmann-handel.example" || str(res.Company, "industry") != "Handel" {
			t.Errorf("created company lost domain or industry: %s", string(body))
		}

		resp, body = putJSON(t, base+"/api/v1/companies/"+companyID, map[string]interface{}{
			"name":          fmt.Sprintf("Feldmann Handels AG %d", suffix),
			"domain":        "feldmann-handel.example",
			"industry":      "Grosshandel",
			"address":       "Hauptstrasse 1, 80331 Muenchen, Deutschland",
			"notes":         "Rahmenvertrag 2026",
			"custom_fields": []interface{}{},
		}, memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		resp, body = getJSON(t, base+"/api/v1/companies/"+companyID, memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		decodeBody(t, body, &res)
		if str(res.Company, "industry") != "Grosshandel" || str(res.Company, "notes") != "Rahmenvertrag 2026" {
			t.Errorf("updated company did not take industry or notes: %s", string(body))
		}
	})

	t.Run("contact: link the company, it comes back with the name", func(t *testing.T) {
		// useUpdateContact accepts company_id (api/hooks/useContacts.ts:105); the kontakte form
		// keeps the company as free text, so this is the only way a link reaches the backend.
		resp, body := putJSON(t, base+"/api/v1/contacts/"+contactID, map[string]interface{}{
			"company_id":    companyID,
			"custom_fields": []interface{}{},
		}, memberToken)
		requireStatus(t, resp, body, http.StatusOK)

		resp, body = getJSON(t, base+"/api/v1/contacts/"+contactID, memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		var res struct {
			Contact map[string]interface{} `json:"contact"`
		}
		decodeBody(t, body, &res)
		if str(res.Contact, "company_id") != companyID || !strings.HasPrefix(str(res.Contact, "company_name"), "Feldmann Handels AG") {
			t.Errorf("expected the contact linked to the company, got %s", string(body))
		}
		// A link must not have wiped the profile fields it did not mention.
		if str(res.Contact, "position") != "Leiterin Einkauf" || str(res.Contact, "category") != "partner" {
			t.Errorf("linking the company changed other fields: %s", string(body))
		}

		resp, body = getJSON(t, base+"/api/v1/companies/"+companyID+"/contacts", memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		if !strings.Contains(string(body), contactID) {
			t.Errorf("company contacts do not list the contact: %s", string(body))
		}
	})

	var stageA, stageB, dealID string

	t.Run("deal: admin creates the stages and a deal with the form body", func(t *testing.T) {
		for i, name := range []string{"Anfrage", "Angebot"} {
			resp, body := postJSON(t, base+"/api/v1/pipeline-stages", map[string]interface{}{
				"name":        fmt.Sprintf("%s %d", name, suffix),
				"color":       "#3B82F6",
				"probability": float64(10 + 40*i),
			}, adminToken)
			requireStatus(t, resp, body, http.StatusCreated)
			var res struct {
				Stage map[string]interface{} `json:"stage"`
			}
			decodeBody(t, body, &res)
			if i == 0 {
				stageA = str(res.Stage, "id")
			} else {
				stageB = str(res.Stage, "id")
			}
		}
		if stageA == "" || stageB == "" {
			t.Fatal("expected two pipeline stage ids")
		}

		// DealsListPage.tsx:165 via useDeals.ts toDealPayload (non-demo): name, value, currency,
		// stage_id, expected_close_date as a date-input value (YYYY-MM-DD), notes undefined when
		// empty, custom_fields replaced by []. The form carries contact and company as names only,
		// so the plain create has no contact_id/company_id.
		resp, body := postJSON(t, base+"/api/v1/deals", map[string]interface{}{
			"name":                fmt.Sprintf("Rahmenvertrag %d", suffix),
			"value":               12500.0,
			"currency":            "EUR",
			"stage_id":            stageA,
			"expected_close_date": "2027-03-31",
			"custom_fields":       []interface{}{},
		}, adminToken)
		requireStatus(t, resp, body, http.StatusCreated)
		var res struct {
			Deal map[string]interface{} `json:"deal"`
		}
		decodeBody(t, body, &res)
		dealID = str(res.Deal, "id")
		if dealID == "" || str(res.Deal, "stage_id") != stageA || res.Deal["value"] != 12500.0 {
			t.Fatalf("expected the deal in stage A with its value, got %s", string(body))
		}
		if !strings.HasPrefix(str(res.Deal, "expected_close_date"), "2027-03-31") {
			t.Errorf("expected_close_date = %q, want 2027-03-31", str(res.Deal, "expected_close_date"))
		}
	})

	t.Run("deal: update with contact and company, then move the stage", func(t *testing.T) {
		// DealDetailPage.tsx:116 body (name, value, currency, expected_close_date, notes,
		// custom_fields []) plus the ids useUpdateDeal accepts.
		resp, body := putJSON(t, base+"/api/v1/deals/"+dealID, map[string]interface{}{
			"name":                fmt.Sprintf("Rahmenvertrag %d", suffix),
			"value":               15000.0,
			"currency":            "EUR",
			"contact_id":          contactID,
			"company_id":          companyID,
			"expected_close_date": "2027-04-30",
			"notes":               "Mit Kontakt und Firma verknuepft",
			"custom_fields":       []interface{}{},
		}, adminToken)
		requireStatus(t, resp, body, http.StatusOK)

		resp, body = getJSON(t, base+"/api/v1/deals/"+dealID, adminToken)
		requireStatus(t, resp, body, http.StatusOK)
		var res struct {
			Deal map[string]interface{} `json:"deal"`
		}
		decodeBody(t, body, &res)
		if str(res.Deal, "contact_id") != contactID || str(res.Deal, "company_id") != companyID {
			t.Errorf("deal lost the contact or company link: %s", string(body))
		}
		if res.Deal["value"] != 15000.0 || str(res.Deal, "notes") != "Mit Kontakt und Firma verknuepft" {
			t.Errorf("deal update did not take value or notes: %s", string(body))
		}
		if str(res.Deal, "stage_id") != stageA {
			t.Errorf("an update without stage_id must keep the stage, got %q", str(res.Deal, "stage_id"))
		}

		// useMoveDealToStage (useDeals.ts:165): POST /deals/{id}/stage {stage_id}.
		resp, body = postJSON(t, base+"/api/v1/deals/"+dealID+"/stage", map[string]interface{}{
			"stage_id": stageB,
		}, adminToken)
		requireStatus(t, resp, body, http.StatusOK)
		decodeBody(t, body, &res)
		if str(res.Deal, "stage_id") != stageB {
			t.Errorf("expected the deal in stage B after the move, got %s", string(body))
		}
		resp, body = getJSON(t, base+"/api/v1/deals/"+dealID, adminToken)
		requireStatus(t, resp, body, http.StatusOK)
		decodeBody(t, body, &res)
		if str(res.Deal, "stage_id") != stageB {
			t.Errorf("the move did not persist: %s", string(body))
		}
	})

	t.Run("member: works their own deal, the admin's deal and every delete stay 403", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/deals", memberToken)
		requireStatus(t, resp, body, http.StatusOK)

		resp, body = postJSON(t, base+"/api/v1/deals", map[string]interface{}{
			"name": "Member deal", "value": 1.0, "currency": "EUR", "stage_id": stageA, "custom_fields": []interface{}{},
		}, memberToken)
		requireStatus(t, resp, body, http.StatusCreated)
		var own struct {
			Deal map[string]interface{} `json:"deal"`
		}
		decodeBody(t, body, &own)
		ownID := str(own.Deal, "id")
		if ownID == "" {
			t.Fatalf("expected the member's deal id, got %s", string(body))
		}

		resp, body = putJSON(t, base+"/api/v1/deals/"+ownID, map[string]interface{}{
			"name": "Member deal gepflegt", "value": 2.0, "currency": "EUR", "custom_fields": []interface{}{},
		}, memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		resp, body = postJSON(t, base+"/api/v1/deals/"+ownID+"/stage", map[string]interface{}{
			"stage_id": stageB,
		}, memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		decodeBody(t, body, &own)
		if str(own.Deal, "stage_id") != stageB || str(own.Deal, "name") != "Member deal gepflegt" {
			t.Errorf("member's own deal did not take the update and the move: %s", string(body))
		}
		resp, body = deleteJSON(t, base+"/api/v1/deals/"+ownID, memberToken)
		requireStatus(t, resp, body, http.StatusForbidden)

		resp, body = putJSON(t, base+"/api/v1/deals/"+dealID, map[string]interface{}{
			"name": "Member rename", "custom_fields": []interface{}{},
		}, memberToken)
		requireStatus(t, resp, body, http.StatusForbidden)

		resp, body = postJSON(t, base+"/api/v1/deals/"+dealID+"/stage", map[string]interface{}{
			"stage_id": stageA,
		}, memberToken)
		requireStatus(t, resp, body, http.StatusForbidden)

		// Delete of contact and company is a manager/admin right as well (matrix rows
		// DELETE /contacts/{id}, DELETE /companies/{id}).
		resp, body = deleteJSON(t, base+"/api/v1/contacts/"+contactID, memberToken)
		requireStatus(t, resp, body, http.StatusForbidden)
		resp, body = deleteJSON(t, base+"/api/v1/companies/"+companyID, memberToken)
		requireStatus(t, resp, body, http.StatusForbidden)

		// The 403s changed nothing.
		resp, body = getJSON(t, base+"/api/v1/deals/"+dealID, adminToken)
		requireStatus(t, resp, body, http.StatusOK)
		var res struct {
			Deal map[string]interface{} `json:"deal"`
		}
		decodeBody(t, body, &res)
		if str(res.Deal, "stage_id") != stageB || !strings.HasPrefix(str(res.Deal, "name"), "Rahmenvertrag") {
			t.Errorf("a refused member write changed the deal: %s", string(body))
		}
	})

	t.Run("contact: the creator is the member", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/contacts/"+contactID, memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		var res struct {
			Contact map[string]interface{} `json:"contact"`
		}
		decodeBody(t, body, &res)
		if str(res.Contact, "created_by") != memberID {
			t.Errorf("created_by = %q, want the member %s", str(res.Contact, "created_by"), memberID)
		}
	})
}
