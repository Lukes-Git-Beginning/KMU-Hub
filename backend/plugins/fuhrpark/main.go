// Package main implements the Fuhrpark (fleet management) WASM plugin.
// Build: GOOS=wasip1 GOARCH=wasm go build -o fuhrpark.wasm ./plugins/fuhrpark/
package main

import (
	"encoding/json"
	"fmt"
)

// Hook request/response types (matching SDK types)
type hookRequest struct {
	HookType   string          `json:"hook_type"`
	Module     string          `json:"module"`
	EntityType string          `json:"entity_type"`
	EntityID   string          `json:"entity_id"`
	EntityData json.RawMessage `json:"entity_data"`
}

type hookResponse struct {
	Modified     bool            `json:"modified"`
	ModifiedData json.RawMessage `json:"modified_data,omitempty"`
	Error        string          `json:"error,omitempty"`
}

// handle_hook is the entry point called by the host runtime
//
//export handle_hook
func handle_hook() int32 {
	// In a real WASM execution, the host would write the request
	// to shared memory and the plugin would read it.
	// This is a reference implementation showing the logic flow.
	return 0
}

// handleContactCreated processes after_create hooks on contacts
func handleContactCreated(req *hookRequest) *hookResponse {
	var contactData map[string]any
	if err := json.Unmarshal(req.EntityData, &contactData); err != nil {
		return &hookResponse{Error: fmt.Sprintf("failed to parse contact data: %v", err)}
	}

	// Check if this is a vehicle contact (has license_plate custom field)
	customFields, ok := contactData["custom_fields"].(map[string]any)
	if !ok {
		return &hookResponse{Modified: false}
	}

	licensePlate, hasPlate := customFields["license_plate"]
	if !hasPlate || licensePlate == "" {
		return &hookResponse{Modified: false}
	}

	// Store initial fuel cost data in KV store
	logInfo(fmt.Sprintf("Vehicle contact created: %s", licensePlate))

	return &hookResponse{Modified: false}
}

// handleContactUpdated processes after_update hooks on contacts
func handleContactUpdated(req *hookRequest) *hookResponse {
	var contactData map[string]any
	if err := json.Unmarshal(req.EntityData, &contactData); err != nil {
		return &hookResponse{Error: fmt.Sprintf("failed to parse contact data: %v", err)}
	}

	customFields, ok := contactData["custom_fields"].(map[string]any)
	if !ok {
		return &hookResponse{Modified: false}
	}

	// Check mileage for maintenance reminder
	if mileage, hasMileage := customFields["mileage"]; hasMileage {
		if km, numOK := mileage.(float64); numOK {
			checkMaintenanceDue(req.EntityID, km, customFields)
		}
	}

	return &hookResponse{Modified: false}
}

// checkMaintenanceDue checks if maintenance is due based on mileage
func checkMaintenanceDue(entityID string, mileage float64, customFields map[string]any) {
	// Read last maintenance mileage from KV store
	lastMaintenance := 0.0

	// Read settings for interval
	interval := 15000.0 // Default 15,000 km

	if mileage-lastMaintenance >= interval {
		licensePlate := ""
		if lp, ok := customFields["license_plate"].(string); ok {
			licensePlate = lp
		}
		logInfo(fmt.Sprintf("Maintenance due for %s (mileage: %.0f km, last: %.0f km)", licensePlate, mileage, lastMaintenance))
	}
}

// calculateFuelCosts calculates fuel costs for a vehicle using KV store data
func calculateFuelCosts(installationID, vehicleID string) (float64, error) {
	// Read fuel entries from KV store
	// Key pattern: fuel:{vehicleID}:{date}
	// Value: {"liters": 45.5, "price_per_liter": 1.85, "total": 84.18, "mileage": 52300}

	// This is a reference — in a real plugin, we'd use the SDK's KVList with prefix
	return 0.0, nil
}

// logInfo logs an info message via the host API
func logInfo(msg string) {
	// In WASM context, this calls the imported log_info function
	_ = msg
}

func main() {}
