//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestCRMContactFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	token, _ := registerAndLogin(t, base)

	var contactID string

	// Create contact
	t.Run("create_contact", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/contacts", map[string]interface{}{
			"first_name": "Max",
			"last_name":  "Mustermann",
			"email":      "max@example.com",
			"phone":      "+49 170 1234567",
			"type":       "person",
		}, token)
		requireStatus(t, resp, body, http.StatusCreated)

		var result struct {
			Contact map[string]interface{} `json:"contact"`
		}
		decodeBody(t, body, &result)

		id, ok := result.Contact["id"].(string)
		if !ok || id == "" {
			t.Fatal("expected contact id")
		}
		contactID = id

		if result.Contact["first_name"] != "Max" {
			t.Fatalf("expected first_name Max, got %v", result.Contact["first_name"])
		}
	})

	// Get contact
	t.Run("get_contact", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/contacts/"+contactID, token)
		requireStatus(t, resp, body, http.StatusOK)

		var result struct {
			Contact map[string]interface{} `json:"contact"`
		}
		decodeBody(t, body, &result)

		if result.Contact["id"] != contactID {
			t.Fatalf("expected contact id %s, got %v", contactID, result.Contact["id"])
		}
	})

	// Update contact
	t.Run("update_contact", func(t *testing.T) {
		resp, body := putJSON(t, base+"/api/v1/contacts/"+contactID, map[string]interface{}{
			"first_name": "Maximilian",
			"last_name":  "Mustermann",
			"email":      "max@example.com",
			"phone":      "+49 170 1234567",
			"type":       "person",
		}, token)
		requireStatus(t, resp, body, http.StatusOK)

		var result struct {
			Contact map[string]interface{} `json:"contact"`
		}
		decodeBody(t, body, &result)

		if result.Contact["first_name"] != "Maximilian" {
			t.Fatalf("expected updated first_name Maximilian, got %v", result.Contact["first_name"])
		}
	})

	// List contacts
	t.Run("list_contacts", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/contacts", token)
		requireStatus(t, resp, body, http.StatusOK)

		var result struct {
			Contacts []map[string]interface{} `json:"contacts"`
		}
		decodeBody(t, body, &result)

		found := false
		for _, c := range result.Contacts {
			if c["id"] == contactID {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("created contact not found in list")
		}
	})

	// Delete contact
	t.Run("delete_contact", func(t *testing.T) {
		resp, body := deleteJSON(t, base+"/api/v1/contacts/"+contactID, token)
		requireStatus(t, resp, body, http.StatusOK)

		// Verify deleted
		resp, body = getJSON(t, base+"/api/v1/contacts/"+contactID, token)
		if resp.StatusCode == http.StatusOK {
			t.Fatal("expected contact to be deleted")
		}
	})

	// No auth
	t.Run("no_auth", func(t *testing.T) {
		resp, _ := getJSON(t, base+"/api/v1/contacts", "")
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
	})
}

func TestCRMCompanyFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	token, _ := registerAndLogin(t, base)

	var companyID string

	t.Run("create_company", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/companies", map[string]interface{}{
			"name":    "Musterfirma GmbH",
			"domain":  "musterfirma.de",
			"address": "Musterstrasse 1, 12345 Musterstadt",
		}, token)
		requireStatus(t, resp, body, http.StatusCreated)

		var result struct {
			Company map[string]interface{} `json:"company"`
		}
		decodeBody(t, body, &result)

		id, ok := result.Company["id"].(string)
		if !ok || id == "" {
			t.Fatal("expected company id")
		}
		companyID = id
	})

	t.Run("get_company", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/companies/"+companyID, token)
		requireStatus(t, resp, body, http.StatusOK)
	})

	t.Run("list_companies", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/companies", token)
		requireStatus(t, resp, body, http.StatusOK)

		var result struct {
			Companies []map[string]interface{} `json:"companies"`
		}
		decodeBody(t, body, &result)

		if len(result.Companies) == 0 {
			t.Fatal("expected at least one company")
		}
	})

	t.Run("update_company", func(t *testing.T) {
		resp, body := putJSON(t, base+"/api/v1/companies/"+companyID, map[string]interface{}{
			"name":   "Musterfirma AG",
			"domain": "musterfirma.de",
		}, token)
		requireStatus(t, resp, body, http.StatusOK)
	})

	t.Run("delete_company", func(t *testing.T) {
		resp, body := deleteJSON(t, base+"/api/v1/companies/"+companyID, token)
		requireStatus(t, resp, body, http.StatusOK)
	})
}

func TestCRMDealFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	token, _ := registerAndLogin(t, base)

	// First create a pipeline stage
	var stageID string
	t.Run("create_pipeline_stage", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/pipeline-stages", map[string]interface{}{
			"name":      "Prospect",
			"position":  1,
			"color":     "#3B82F6",
			"is_closed": false,
		}, token)
		requireStatus(t, resp, body, http.StatusCreated)

		var result struct {
			PipelineStage map[string]interface{} `json:"pipeline_stage"`
		}
		decodeBody(t, body, &result)

		id, ok := result.PipelineStage["id"].(string)
		if !ok || id == "" {
			t.Fatal("expected pipeline stage id")
		}
		stageID = id
	})

	// Create a contact for the deal
	var contactID string
	t.Run("create_contact_for_deal", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/contacts", map[string]interface{}{
			"first_name": "Deal",
			"last_name":  "Contact",
			"email":      "deal@example.com",
			"type":       "person",
		}, token)
		requireStatus(t, resp, body, http.StatusCreated)

		var result struct {
			Contact map[string]interface{} `json:"contact"`
		}
		decodeBody(t, body, &result)
		contactID = result.Contact["id"].(string)
	})

	var dealID string
	t.Run("create_deal", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/deals", map[string]interface{}{
			"title":      "Big Deal",
			"value":      50000.0,
			"currency":   "EUR",
			"stage_id":   stageID,
			"contact_id": contactID,
		}, token)
		requireStatus(t, resp, body, http.StatusCreated)

		var result struct {
			Deal map[string]interface{} `json:"deal"`
		}
		decodeBody(t, body, &result)

		id, ok := result.Deal["id"].(string)
		if !ok || id == "" {
			t.Fatal("expected deal id")
		}
		dealID = id
	})

	t.Run("get_deal", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/deals/"+dealID, token)
		requireStatus(t, resp, body, http.StatusOK)

		var result struct {
			Deal map[string]interface{} `json:"deal"`
		}
		decodeBody(t, body, &result)

		if result.Deal["title"] != "Big Deal" {
			t.Fatalf("expected title Big Deal, got %v", result.Deal["title"])
		}
	})

	t.Run("list_deals", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/deals", token)
		requireStatus(t, resp, body, http.StatusOK)

		// Response should contain deals array
		var raw map[string]json.RawMessage
		decodeBody(t, body, &raw)
		if _, ok := raw["deals"]; !ok {
			t.Fatal("expected deals key in response")
		}
	})

	t.Run("update_deal", func(t *testing.T) {
		resp, body := putJSON(t, base+"/api/v1/deals/"+dealID, map[string]interface{}{
			"title":    "Big Deal Updated",
			"value":    75000.0,
			"currency": "EUR",
			"stage_id": stageID,
		}, token)
		requireStatus(t, resp, body, http.StatusOK)
	})

	t.Run("delete_deal", func(t *testing.T) {
		resp, body := deleteJSON(t, base+"/api/v1/deals/"+dealID, token)
		requireStatus(t, resp, body, http.StatusOK)
	})
}
