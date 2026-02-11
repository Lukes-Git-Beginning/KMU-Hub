package gdpr

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// DataExportHandler defines the interface for per-module GDPR data export.
// Each module registers a handler to export its user-owned data.
type DataExportHandler interface {
	// ModuleName returns the unique module identifier (e.g., "auth", "crm", "chat").
	ModuleName() string

	// ExportUserData exports all data belonging to the given user from this module.
	// Returns the data as JSON bytes. Returns nil, nil if the module has no data for this user.
	ExportUserData(ctx context.Context, userID uuid.UUID) ([]byte, error)
}

// ExecuteExport iterates over all registered export handlers, collects each module's
// user data as JSON, and packages everything into a ZIP archive.
// Each module's data becomes a separate JSON file in the archive.
func ExecuteExport(ctx context.Context, userID uuid.UUID, handlers []DataExportHandler) ([]byte, error) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	// Add metadata file with export information
	metadata := map[string]interface{}{
		"export_type":    "gdpr_data_export",
		"user_id":        userID.String(),
		"generated_at":   time.Now().UTC().Format(time.RFC3339),
		"modules_count":  len(handlers),
		"format_version": "1.0",
	}

	metaJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("gdpr: failed to marshal export metadata: %w", err)
	}

	metaFile, err := zipWriter.Create("_metadata.json")
	if err != nil {
		return nil, fmt.Errorf("gdpr: failed to create metadata entry in zip: %w", err)
	}
	if _, err := metaFile.Write(metaJSON); err != nil {
		return nil, fmt.Errorf("gdpr: failed to write metadata to zip: %w", err)
	}

	// Collect data from each module handler
	for _, handler := range handlers {
		moduleName := handler.ModuleName()

		data, exportErr := handler.ExportUserData(ctx, userID)
		if exportErr != nil {
			slog.Error("gdpr export: module export failed",
				"module", moduleName,
				"user_id", userID,
				"error", exportErr,
			)
			// Write error marker file for this module
			errFile, createErr := zipWriter.Create(fmt.Sprintf("%s/_error.txt", moduleName))
			if createErr != nil {
				continue
			}
			_, _ = errFile.Write([]byte(fmt.Sprintf("Export failed: %s", exportErr.Error())))
			continue
		}

		if data == nil || len(data) == 0 {
			slog.Info("gdpr export: no data for module",
				"module", moduleName,
				"user_id", userID,
			)
			continue
		}

		fileName := fmt.Sprintf("%s/data.json", moduleName)
		file, createErr := zipWriter.Create(fileName)
		if createErr != nil {
			return nil, fmt.Errorf("gdpr: failed to create zip entry for %s: %w", moduleName, createErr)
		}
		if _, writeErr := file.Write(data); writeErr != nil {
			return nil, fmt.Errorf("gdpr: failed to write data for %s: %w", moduleName, writeErr)
		}
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("gdpr: failed to finalize zip archive: %w", err)
	}

	return buf.Bytes(), nil
}

// ---------------------------------------------------------------------------
// Per-module export handlers
// ---------------------------------------------------------------------------

// AuthExportHandler exports authentication-related user data (sessions, 2FA status, login history).
type AuthExportHandler struct{}

func (h *AuthExportHandler) ModuleName() string { return "auth" }

func (h *AuthExportHandler) ExportUserData(ctx context.Context, userID uuid.UUID) ([]byte, error) {
	// Exports user profile, session history, 2FA enrollment status.
	// Actual DB queries will be wired when gRPC integration is connected.
	data := map[string]interface{}{
		"module":      "auth",
		"description": "Authentication and session data",
		"user_id":     userID.String(),
		"note":        "Full export will include: profile, sessions, login history, 2FA status",
	}
	return json.MarshalIndent(data, "", "  ")
}

// CRMExportHandler exports CRM-related user data (contacts, deals, activities created by user).
type CRMExportHandler struct{}

func (h *CRMExportHandler) ModuleName() string { return "crm" }

func (h *CRMExportHandler) ExportUserData(ctx context.Context, userID uuid.UUID) ([]byte, error) {
	// Stub: will be connected to CRM service for contacts, deals, activities, notes
	data := map[string]interface{}{
		"module":      "crm",
		"description": "CRM contacts, deals, activities, and notes",
		"user_id":     userID.String(),
		"note":        "Stub handler - will export contacts, deals, activities owned by user",
	}
	return json.MarshalIndent(data, "", "  ")
}

// ChatExportHandler exports chat messages and channel memberships.
type ChatExportHandler struct{}

func (h *ChatExportHandler) ModuleName() string { return "chat" }

func (h *ChatExportHandler) ExportUserData(ctx context.Context, userID uuid.UUID) ([]byte, error) {
	// Stub: will be connected to Chat service for messages, channels, DMs
	data := map[string]interface{}{
		"module":      "chat",
		"description": "Chat messages, channels, and direct messages",
		"user_id":     userID.String(),
		"note":        "Stub handler - will export messages, channel memberships, file references",
	}
	return json.MarshalIndent(data, "", "  ")
}

// WorkExportHandler exports work/project data (tasks, time entries, comments).
type WorkExportHandler struct{}

func (h *WorkExportHandler) ModuleName() string { return "work" }

func (h *WorkExportHandler) ExportUserData(ctx context.Context, userID uuid.UUID) ([]byte, error) {
	// Stub: will be connected to Work service for tasks, projects, time entries
	data := map[string]interface{}{
		"module":      "work",
		"description": "Projects, tasks, time entries, and comments",
		"user_id":     userID.String(),
		"note":        "Stub handler - will export tasks, time entries, comments owned by user",
	}
	return json.MarshalIndent(data, "", "  ")
}

// CalendarExportHandler exports calendar events, bookings, and preferences.
type CalendarExportHandler struct{}

func (h *CalendarExportHandler) ModuleName() string { return "calendar" }

func (h *CalendarExportHandler) ExportUserData(ctx context.Context, userID uuid.UUID) ([]byte, error) {
	// Stub: will be connected to Calendar service for events, calendars, bookings
	data := map[string]interface{}{
		"module":      "calendar",
		"description": "Calendars, events, bookings, and preferences",
		"user_id":     userID.String(),
		"note":        "Stub handler - will export calendars, events, resource bookings",
	}
	return json.MarshalIndent(data, "", "  ")
}

// SessionExportHandler exports session history and device metadata.
type SessionExportHandler struct{}

func (h *SessionExportHandler) ModuleName() string { return "sessions" }

func (h *SessionExportHandler) ExportUserData(ctx context.Context, userID uuid.UUID) ([]byte, error) {
	// Exports session device history, IP addresses, last active times.
	// Real implementation queries user_sessions table.
	data := map[string]interface{}{
		"module":      "sessions",
		"description": "Session history with device and location metadata",
		"user_id":     userID.String(),
		"note":        "Full export will include: device names, IP addresses, login timestamps",
	}
	return json.MarshalIndent(data, "", "  ")
}

// NotificationExportHandler exports notification preferences and history.
type NotificationExportHandler struct{}

func (h *NotificationExportHandler) ModuleName() string { return "notifications" }

func (h *NotificationExportHandler) ExportUserData(ctx context.Context, userID uuid.UUID) ([]byte, error) {
	// Stub: will be connected to Notification service for preferences and history
	data := map[string]interface{}{
		"module":      "notifications",
		"description": "Notification preferences and delivery history",
		"user_id":     userID.String(),
		"note":        "Stub handler - will export notification preferences and history",
	}
	return json.MarshalIndent(data, "", "  ")
}
