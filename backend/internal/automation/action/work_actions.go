package action

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	workv1 "github.com/kmuhub/kmuhub/proto/work/v1"
)

// CreateTaskAction creates a new task in a project via gRPC.
type CreateTaskAction struct {
	client workv1.WorkServiceClient
}

// NewCreateTaskAction creates a new CreateTaskAction.
func NewCreateTaskAction(client workv1.WorkServiceClient) *CreateTaskAction {
	return &CreateTaskAction{client: client}
}

func (a *CreateTaskAction) Type() string { return "work.create_task" }

type createTaskConfig struct {
	ProjectID  string `json:"project_id"`  // template
	Title      string `json:"title"`       // template
	AssigneeID string `json:"assignee_id"` // template
	CreatedBy  string `json:"created_by"`  // template: user ID; defaults to the event actor
	Priority   string `json:"priority"`    // "low", "normal", "high", "urgent"
	DueDays    int    `json:"due_days"`    // days from now
}

func (a *CreateTaskAction) Execute(ctx context.Context, config json.RawMessage, env map[string]any) (ActionResult, error) {
	if a.client == nil {
		return ActionResult{Success: false, Error: "work service unavailable"}, fmt.Errorf("work gRPC client is nil")
	}

	var cfg createTaskConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return ActionResult{Success: false, Error: "invalid config: " + err.Error()}, err
	}

	projectID := resolveTemplate(cfg.ProjectID, env)
	title := resolveTemplate(cfg.Title, env)
	assigneeID := resolveTemplate(cfg.AssigneeID, env)
	priority := cfg.Priority
	if priority == "" {
		priority = "normal"
	}

	createdBy, err := resolveCreatedBy(cfg.CreatedBy, env)
	if err != nil {
		return ActionResult{Success: false, Error: err.Error()}, err
	}

	req := &workv1.CreateTaskRequest{
		ProjectId: projectID,
		Title:     title,
		Priority:  priority,
		CreatedBy: createdBy,
	}

	if assigneeID != "" {
		req.AssigneeId = &assigneeID
	}

	if cfg.DueDays > 0 {
		dueDate := time.Now().AddDate(0, 0, cfg.DueDays)
		req.DueDate = timestamppb.New(dueDate)
	}

	resp, err := a.client.CreateTask(ctx, req)
	if err != nil {
		return ActionResult{Success: false, Error: err.Error()}, err
	}

	return ActionResult{
		Success: true,
		Output: map[string]any{
			"task_id": resp.GetTask().GetId(),
		},
	}, nil
}

// resolveCreatedBy renders the task creator. The work service rejects a task
// without a creator (tasks.created_by is NOT NULL), so an empty or non-UUID
// value is reported as a configuration error here rather than as an opaque
// gRPC failure. Falls back to the user who triggered the event.
func resolveCreatedBy(tmpl string, env map[string]any) (string, error) {
	createdBy := strings.TrimSpace(resolveTemplate(tmpl, env))
	if createdBy == "" {
		createdBy = strings.TrimSpace(fmt.Sprint(env["actor_id"]))
	}
	if id, err := uuid.Parse(createdBy); err != nil || id == uuid.Nil {
		return "", fmt.Errorf("created_by must be a user ID (config created_by or event actor_id), got %q", createdBy)
	}
	return createdBy, nil
}

// CreateTaskDefinition returns the action definition for the catalog.
func CreateTaskDefinition() *ActionDefinition {
	return &ActionDefinition{
		Type:        "work.create_task",
		Module:      "work",
		Name:        "Aufgabe erstellen",
		Description: "Erstellt eine neue Aufgabe in einem Projekt",
		Params: []ConfigParam{
			{Key: "project_id", Label: "Projekt-ID", Type: "template", Required: true},
			{Key: "title", Label: "Titel", Type: "template", Required: true},
			{Key: "assignee_id", Label: "Zugewiesener", Type: "template", Required: false},
			{Key: "created_by", Label: "Erstellt von", Type: "template", Required: false, Default: "{{actor_id}}"},
			{Key: "priority", Label: "Prioritaet", Type: "select", Required: false, Default: "normal"},
			{Key: "due_days", Label: "Faellig in (Tagen)", Type: "number", Required: false, Default: 0},
		},
		OutputFields: []OutputField{
			{Key: "task_id", Label: "Aufgaben-ID", Type: "string"},
		},
	}
}
