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
	// Additive guards: every route keeps its legacy coarse key AND accepts the
	// matching capability-catalogue key. Extend, never swap (see
	// middleware.RequirePermissionAny doc comment).
	//
	// Member-add/role-change, from-template and the preference write have no
	// FE caller at all (grepped, zero hits) — no additive target, left as-is.
	// Kanban-column delete keeps its stricter "projects:delete" legacy key but
	// gets work:project:edit as its catalogue alt: the whole statuses tab in
	// ProjectSettingsDialog only renders once projectCan.edit is true, same as
	// create/reorder/update, so the FE boundary for all four is identical even
	// though the BE historically required more for delete. work:board:export
	// has no backend route at all — key stays unused, no route invented.
	var (
		projRead         = middleware.RequirePermissionAny([2]string{"projects", "read"}, [2]string{"work:project", "read"})
		projCreate       = middleware.RequirePermissionAny([2]string{"projects", "write"}, [2]string{"work:project", "create"})
		projEdit         = middleware.RequirePermissionAny([2]string{"projects", "write"}, [2]string{"work:project", "edit"})
		projDelete       = middleware.RequirePermissionAny([2]string{"projects", "delete"}, [2]string{"work:project", "delete"})
		projStatusDelete = middleware.RequirePermissionAny([2]string{"projects", "delete"}, [2]string{"work:project", "edit"})
		projMemberManage = middleware.RequirePermissionAny([2]string{"projects", "write"}, [2]string{"work:project", "manage_members"})
	)

	// Projects
	r.Route("/api/v1/projects", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(projRead).Get("/", w.HandleListProjects)
		r.With(projRead).Get("/{id}", w.HandleGetProject)
		r.With(projCreate).Post("/", w.HandleCreateProject)
		r.With(projEdit).Put("/{id}", w.HandleUpdateProject)
		r.With(projEdit).Post("/{id}/archive", w.HandleArchiveProject)
		r.With(projDelete).Delete("/{id}", w.HandleDeleteProject)

		// Members. Add/role-change have no FE caller (grepped, zero hits) —
		// stay on the coarse key only, no catalogue key invented.
		r.With(projRead).Get("/{id}/members", w.HandleListProjectMembers)
		r.With(middleware.RequirePermission("projects", "write")).Post("/{id}/members", w.HandleAddProjectMember)
		r.With(middleware.RequirePermission("projects", "write")).Put("/{id}/members/{userId}", w.HandleUpdateProjectMemberRole)
		r.With(projMemberManage).Delete("/{id}/members/{userId}", w.HandleRemoveProjectMember)

		// Templates. Save-as-template lives inside the edit-gated settings
		// dialog; create-from-template has no FE caller at all.
		r.With(projEdit).Post("/{id}/template", w.HandleSaveProjectAsTemplate)
		r.With(middleware.RequirePermission("projects", "write")).Post("/from-template", w.HandleCreateProjectFromTemplate)

		// Statuses
		r.With(projRead).Get("/{id}/statuses", w.HandleListProjectStatuses)
		r.With(projEdit).Post("/{id}/statuses", w.HandleCreateProjectStatus)
		r.With(projEdit).Post("/{id}/statuses/reorder", w.HandleReorderProjectStatuses)

		// Preferences. Personal per-user view state, no FE capability gate at
		// either call site — stays on the coarse key only.
		r.With(projRead).Get("/{id}/preferences", w.HandleGetUserProjectPreference)
		r.With(middleware.RequirePermission("projects", "write")).Put("/{id}/preferences", w.HandleSetUserProjectPreference)

		// Time entries roll-up ("Stunden abrechnen" dialog).
		r.With(projRead).Get("/{id}/time-entries", w.HandleListProjectTimeEntries)

		// Team utilization roll-up (Auslastung view).
		r.With(projRead).Get("/{id}/team-utilization", w.HandleListProjectTeamUtilization)
	})

	// Project Statuses (top-level for update/delete by status ID)
	r.Route("/api/v1/project-statuses", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(projEdit).Put("/{id}", w.HandleUpdateProjectStatus)
		r.With(projStatusDelete).Delete("/{id}", w.HandleDeleteProjectStatus)
	})

	// Additive guards for tasks: every sub-resource action in TaskDetailPage
	// (dependencies, entity links, custom fields, files, labels) threads the
	// same `can.edit` (work:task:edit) prop into its section component —
	// verified against TaskDetailPage.tsx, not assumed. Comment CREATE is
	// gated separately by work:task:comment (canComment); comment
	// update/delete and time-entry update/delete are `isOwn`/`isOwner`
	// client-side checks with no capability call at all, so they keep their
	// coarse key only. MoveTask (kanban drag) has no FE gate call either.
	var (
		taskRead    = middleware.RequirePermissionAny([2]string{"tasks", "read"}, [2]string{"work:task", "read"})
		taskCreate  = middleware.RequirePermissionAny([2]string{"tasks", "write"}, [2]string{"work:task", "create"})
		taskEdit    = middleware.RequirePermissionAny([2]string{"tasks", "write"}, [2]string{"work:task", "edit"})
		taskDelete  = middleware.RequirePermissionAny([2]string{"tasks", "delete"}, [2]string{"work:task", "delete"})
		taskComment = middleware.RequirePermissionAny([2]string{"tasks", "write"}, [2]string{"work:task", "comment"})
		taskEditDel = middleware.RequirePermissionAny([2]string{"tasks", "delete"}, [2]string{"work:task", "edit"})
		timeLog     = middleware.RequirePermissionAny([2]string{"tasks", "write"}, [2]string{"work:time", "log"})
		labelEdit   = middleware.RequirePermissionAny([2]string{"work_labels", "write"}, [2]string{"work:task", "edit"})
	)

	// Tasks
	r.Route("/api/v1/tasks", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(taskRead).Get("/", w.HandleListTasks)
		r.With(taskRead).Get("/{id}", w.HandleGetTask)
		r.With(taskCreate).Post("/", w.HandleCreateTask)
		r.With(taskEdit).Put("/{id}", w.HandleUpdateTask)
		r.With(taskDelete).Delete("/{id}", w.HandleDeleteTask)
		r.With(middleware.RequirePermission("tasks", "write")).Post("/{id}/move", w.HandleMoveTask)
		r.With(taskRead).Get("/{id}/subtasks", w.HandleListSubtasks)

		// Dependencies
		r.With(taskRead).Get("/{id}/dependencies", w.HandleListTaskDependencies)
		r.With(taskEdit).Post("/{id}/dependencies", w.HandleCreateTaskDependency)

		// Comments
		r.With(taskRead).Get("/{id}/comments", w.HandleListTaskComments)
		r.With(taskComment).Post("/{id}/comments", w.HandleCreateTaskComment)

		// Entity links
		r.With(taskRead).Get("/{id}/links", w.HandleListTaskEntityLinks)
		r.With(taskEdit).Post("/{id}/links", w.HandleLinkEntityToTask)

		// Activity log
		r.With(taskRead).Get("/{id}/activities", w.HandleListTaskActivities)

		// Files
		r.With(taskRead).Get("/{id}/files", w.HandleListTaskFiles)
		r.With(taskEdit).Post("/{id}/files", w.HandleAttachFileToTask)

		// Custom fields
		r.With(taskRead).Get("/{id}/custom-fields", w.HandleGetTaskCustomFieldValues)
		r.With(taskEdit).Put("/{id}/custom-fields", w.HandleSetTaskCustomFieldValues)

		// Time entries
		r.With(taskRead).Get("/{id}/time-entries", w.HandleListTimeEntries)
		r.With(timeLog).Post("/{id}/time-entries", w.HandleAddManualTimeEntry)
		r.With(taskRead).Get("/{id}/time-summary", w.HandleGetTaskTimeSummary)

		// Timer
		r.With(timeLog).Post("/{id}/timer/start", w.HandleStartTimer)

		// Labels
		r.With(labelEdit).Put("/{id}/labels", w.HandleSetTaskLabels)
	})

	// Timer (user-level)
	r.Route("/api/v1/timer", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(timeLog).Post("/stop", w.HandleStopTimer)
		r.With(taskRead).Get("/active", w.HandleGetActiveTimer)
	})

	// Time entries (top-level update/delete) — isOwner-only in the FE, no
	// capability gate call, stays on the coarse key.
	r.Route("/api/v1/time-entries", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("tasks", "write")).Put("/{id}", w.HandleUpdateTimeEntry)
		r.With(middleware.RequirePermission("tasks", "delete")).Delete("/{id}", w.HandleDeleteTimeEntry)
	})

	// Task dependencies (top-level delete)
	r.Route("/api/v1/task-dependencies", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(taskEditDel).Delete("/{id}", w.HandleDeleteTaskDependency)
	})

	// Task comments (top-level update/delete) — isOwn-only in the FE, no
	// capability gate call, stays on the coarse key.
	r.Route("/api/v1/task-comments", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("tasks", "write")).Put("/{id}", w.HandleUpdateTaskComment)
		r.With(middleware.RequirePermission("tasks", "delete")).Delete("/{id}", w.HandleDeleteTaskComment)
	})

	// Entity links (top-level delete)
	r.Route("/api/v1/task-links", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(taskEditDel).Delete("/{id}", w.HandleUnlinkEntityFromTask)
	})

	// Task files (top-level delete)
	r.Route("/api/v1/task-files", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(taskEditDel).Delete("/{id}", w.HandleRemoveTaskFile)
	})

	// Entity tasks (list tasks linked to a CRM entity)
	r.Route("/api/v1/entity-tasks", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(taskRead).Get("/", w.HandleListEntityTasks)
	})

	// Work search
	r.Route("/api/v1/work/search", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(taskRead).Get("/", w.HandleSearchTasks)
	})

	// Work labels (definition CRUD) and custom-field definitions have no
	// catalogue key — the catalogue only curates task/project/time/board
	// subjects, label/custom-field *definition* management lives elsewhere.
	// Stay on the coarse key only, no key invented.

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
