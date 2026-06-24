package gateway

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/middleware"
	workv1 "github.com/kmuhub/kmuhub/proto/work/v1"
)

// WorkRoutes handles HTTP routes for the Work backend service.
type WorkRoutes struct {
	registry *ServiceRegistry
}

// NewWorkRoutes creates a new WorkRoutes with the given service registry.
func NewWorkRoutes(registry *ServiceRegistry) *WorkRoutes {
	return &WorkRoutes{registry: registry}
}

// ServiceName returns the backend service name.
func (w *WorkRoutes) ServiceName() string { return "work" }

// getWorkClient lazily obtains a gRPC client for the Work service.
func (w *WorkRoutes) getWorkClient() (workv1.WorkServiceClient, error) {
	conn, err := w.registry.GetConnection("work")
	if err != nil {
		return nil, err
	}
	return workv1.NewWorkServiceClient(conn), nil
}

// RegisterRoutes registers all Work HTTP routes.
func (w *WorkRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	// Projects
	r.Route("/api/v1/projects", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("projects", "read")).Get("/", w.HandleListProjects)
		r.With(middleware.RequirePermission("projects", "read")).Get("/{id}", w.HandleGetProject)
		r.With(middleware.RequirePermission("projects", "write")).Post("/", w.HandleCreateProject)
		r.With(middleware.RequirePermission("projects", "write")).Put("/{id}", w.HandleUpdateProject)
		r.With(middleware.RequirePermission("projects", "write")).Post("/{id}/archive", w.HandleArchiveProject)
		r.With(middleware.RequirePermission("projects", "delete")).Delete("/{id}", w.HandleDeleteProject)

		// Members
		r.With(middleware.RequirePermission("projects", "read")).Get("/{id}/members", w.HandleListProjectMembers)
		r.With(middleware.RequirePermission("projects", "write")).Post("/{id}/members", w.HandleAddProjectMember)
		r.With(middleware.RequirePermission("projects", "write")).Put("/{id}/members/{userId}", w.HandleUpdateProjectMemberRole)
		r.With(middleware.RequirePermission("projects", "write")).Delete("/{id}/members/{userId}", w.HandleRemoveProjectMember)

		// Templates
		r.With(middleware.RequirePermission("projects", "write")).Post("/{id}/template", w.HandleSaveProjectAsTemplate)
		r.With(middleware.RequirePermission("projects", "write")).Post("/from-template", w.HandleCreateProjectFromTemplate)

		// Statuses
		r.With(middleware.RequirePermission("projects", "read")).Get("/{id}/statuses", w.HandleListProjectStatuses)
		r.With(middleware.RequirePermission("projects", "write")).Post("/{id}/statuses", w.HandleCreateProjectStatus)
		r.With(middleware.RequirePermission("projects", "write")).Post("/{id}/statuses/reorder", w.HandleReorderProjectStatuses)

		// Preferences
		r.With(middleware.RequirePermission("projects", "read")).Get("/{id}/preferences", w.HandleGetUserProjectPreference)
		r.With(middleware.RequirePermission("projects", "write")).Put("/{id}/preferences", w.HandleSetUserProjectPreference)
	})

	// Project Statuses (top-level for update/delete by status ID)
	r.Route("/api/v1/project-statuses", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("projects", "write")).Put("/{id}", w.HandleUpdateProjectStatus)
		r.With(middleware.RequirePermission("projects", "delete")).Delete("/{id}", w.HandleDeleteProjectStatus)
	})

	// Tasks
	r.Route("/api/v1/tasks", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("tasks", "read")).Get("/", w.HandleListTasks)
		r.With(middleware.RequirePermission("tasks", "read")).Get("/{id}", w.HandleGetTask)
		r.With(middleware.RequirePermission("tasks", "write")).Post("/", w.HandleCreateTask)
		r.With(middleware.RequirePermission("tasks", "write")).Put("/{id}", w.HandleUpdateTask)
		r.With(middleware.RequirePermission("tasks", "delete")).Delete("/{id}", w.HandleDeleteTask)
		r.With(middleware.RequirePermission("tasks", "write")).Post("/{id}/move", w.HandleMoveTask)
		r.With(middleware.RequirePermission("tasks", "read")).Get("/{id}/subtasks", w.HandleListSubtasks)

		// Dependencies
		r.With(middleware.RequirePermission("tasks", "read")).Get("/{id}/dependencies", w.HandleListTaskDependencies)
		r.With(middleware.RequirePermission("tasks", "write")).Post("/{id}/dependencies", w.HandleCreateTaskDependency)

		// Comments
		r.With(middleware.RequirePermission("tasks", "read")).Get("/{id}/comments", w.HandleListTaskComments)
		r.With(middleware.RequirePermission("tasks", "write")).Post("/{id}/comments", w.HandleCreateTaskComment)

		// Entity links
		r.With(middleware.RequirePermission("tasks", "read")).Get("/{id}/links", w.HandleListTaskEntityLinks)
		r.With(middleware.RequirePermission("tasks", "write")).Post("/{id}/links", w.HandleLinkEntityToTask)

		// Activity log
		r.With(middleware.RequirePermission("tasks", "read")).Get("/{id}/activities", w.HandleListTaskActivities)

		// Files
		r.With(middleware.RequirePermission("tasks", "read")).Get("/{id}/files", w.HandleListTaskFiles)
		r.With(middleware.RequirePermission("tasks", "write")).Post("/{id}/files", w.HandleAttachFileToTask)

		// Custom fields
		r.With(middleware.RequirePermission("tasks", "read")).Get("/{id}/custom-fields", w.HandleGetTaskCustomFieldValues)
		r.With(middleware.RequirePermission("tasks", "write")).Put("/{id}/custom-fields", w.HandleSetTaskCustomFieldValues)

		// Time entries
		r.With(middleware.RequirePermission("tasks", "read")).Get("/{id}/time-entries", w.HandleListTimeEntries)
		r.With(middleware.RequirePermission("tasks", "write")).Post("/{id}/time-entries", w.HandleAddManualTimeEntry)
		r.With(middleware.RequirePermission("tasks", "read")).Get("/{id}/time-summary", w.HandleGetTaskTimeSummary)

		// Timer
		r.With(middleware.RequirePermission("tasks", "write")).Post("/{id}/timer/start", w.HandleStartTimer)

		// Labels
		r.With(middleware.RequirePermission("work_labels", "write")).Put("/{id}/labels", w.HandleSetTaskLabels)
	})

	// Timer (user-level)
	r.Route("/api/v1/timer", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("tasks", "write")).Post("/stop", w.HandleStopTimer)
		r.With(middleware.RequirePermission("tasks", "read")).Get("/active", w.HandleGetActiveTimer)
	})

	// Time entries (top-level update/delete)
	r.Route("/api/v1/time-entries", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("tasks", "write")).Put("/{id}", w.HandleUpdateTimeEntry)
		r.With(middleware.RequirePermission("tasks", "delete")).Delete("/{id}", w.HandleDeleteTimeEntry)
	})

	// Task dependencies (top-level delete)
	r.Route("/api/v1/task-dependencies", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("tasks", "delete")).Delete("/{id}", w.HandleDeleteTaskDependency)
	})

	// Task comments (top-level update/delete)
	r.Route("/api/v1/task-comments", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("tasks", "write")).Put("/{id}", w.HandleUpdateTaskComment)
		r.With(middleware.RequirePermission("tasks", "delete")).Delete("/{id}", w.HandleDeleteTaskComment)
	})

	// Entity links (top-level delete)
	r.Route("/api/v1/task-links", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("tasks", "delete")).Delete("/{id}", w.HandleUnlinkEntityFromTask)
	})

	// Task files (top-level delete)
	r.Route("/api/v1/task-files", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("tasks", "delete")).Delete("/{id}", w.HandleRemoveTaskFile)
	})

	// Entity tasks (list tasks linked to a CRM entity)
	r.Route("/api/v1/entity-tasks", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("tasks", "read")).Get("/", w.HandleListEntityTasks)
	})

	// Work search
	r.Route("/api/v1/work/search", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("tasks", "read")).Get("/", w.HandleSearchTasks)
	})

	// Work labels
	r.Route("/api/v1/work/labels", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("work_labels", "read")).Get("/", w.HandleListLabels)
		r.With(middleware.RequirePermission("work_labels", "write")).Post("/", w.HandleCreateLabel)
		r.With(middleware.RequirePermission("work_labels", "read")).Get("/{id}", w.HandleGetLabel)
		r.With(middleware.RequirePermission("work_labels", "write")).Put("/{id}", w.HandleUpdateLabel)
		r.With(middleware.RequirePermission("work_labels", "delete")).Delete("/{id}", w.HandleDeleteLabel)
	})

	// Work custom field definitions
	r.Route("/api/v1/work/custom-fields", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("work_custom_fields", "read")).Get("/", w.HandleListCustomFieldDefinitions)
		r.With(middleware.RequirePermission("work_custom_fields", "write")).Post("/", w.HandleCreateCustomFieldDefinition)
		r.With(middleware.RequirePermission("work_custom_fields", "read")).Get("/{id}", w.HandleGetCustomFieldDefinition)
		r.With(middleware.RequirePermission("work_custom_fields", "write")).Put("/{id}", w.HandleUpdateCustomFieldDefinition)
		r.With(middleware.RequirePermission("work_custom_fields", "delete")).Delete("/{id}", w.HandleDeleteCustomFieldDefinition)
	})
}

// ============================================================================
// Helpers
// ============================================================================

// parseTimestamp parses a date string into a protobuf Timestamp.
// Supports RFC3339 and date-only formats.
func parseTimestamp(s string) (*timestamppb.Timestamp, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return timestamppb.New(t), nil
		}
	}
	return nil, fmt.Errorf("invalid timestamp format: %s", s)
}
