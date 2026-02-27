package action

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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
	Priority   string `json:"priority"`    // "low", "medium", "high", "critical"
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
		priority = "medium"
	}

	req := &workv1.CreateTaskRequest{
		ProjectId: projectID,
		Title:     title,
		Priority:  priority,
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
			{Key: "priority", Label: "Prioritaet", Type: "select", Required: false, Default: "medium"},
			{Key: "due_days", Label: "Faellig in (Tagen)", Type: "number", Required: false, Default: 0},
		},
		OutputFields: []OutputField{
			{Key: "task_id", Label: "Aufgaben-ID", Type: "string"},
		},
	}
}
