package models

import (
	"time"

	"github.com/google/uuid"
)

// Project role constants
const (
	ProjectRoleOwner  = "owner"
	ProjectRoleMember = "member"
	ProjectRoleViewer = "viewer"
)

// ValidProjectRoles contains all valid project member roles
var ValidProjectRoles = map[string]bool{
	ProjectRoleOwner:  true,
	ProjectRoleMember: true,
	ProjectRoleViewer: true,
}

// Project represents a project in the work management system
type Project struct {
	ID               uuid.UUID  `json:"id"`
	Name             string     `json:"name"`
	Description      *string    `json:"description,omitempty"`
	ProjectKey       string     `json:"project_key"`
	NextTaskNumber   int        `json:"next_task_number"`
	IsTemplate       bool       `json:"is_template"`
	TemplateSourceID *uuid.UUID `json:"template_source_id,omitempty"`
	CreatedBy        uuid.UUID  `json:"created_by"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	ArchivedAt       *time.Time `json:"archived_at,omitempty"`
}

// ProjectWithDetails includes aggregated counts for list views
type ProjectWithDetails struct {
	Project
	MemberCount int    `json:"member_count"`
	TaskCount   int    `json:"task_count"`
	OwnerName   string `json:"owner_name"`
}

// ProjectMember represents a user's membership in a project
type ProjectMember struct {
	ProjectID uuid.UUID `json:"project_id"`
	UserID    uuid.UUID `json:"user_id"`
	Role      string    `json:"role"` // "owner", "member", "viewer"
	AddedAt   time.Time `json:"added_at"`
	FirstName string    `json:"first_name"` // denormalized from users table
	LastName  string    `json:"last_name"`  // denormalized from users table
}

// ProjectStatus represents a customizable status column in a project
type ProjectStatus struct {
	ID        uuid.UUID `json:"id"`
	ProjectID uuid.UUID `json:"project_id"`
	Name      string    `json:"name"`
	Color     *string   `json:"color,omitempty"`
	SortOrder int       `json:"sort_order"`
	IsDefault bool      `json:"is_default"`
	IsClosed  bool      `json:"is_closed"`
	CreatedAt time.Time `json:"created_at"`
}

// UserProjectPreference stores per-user view settings for a project
type UserProjectPreference struct {
	UserID       uuid.UUID `json:"user_id"`
	ProjectID    uuid.UUID `json:"project_id"`
	ViewType     string    `json:"view_type"`     // "list", "kanban"
	ListGroupBy  string    `json:"list_group_by"`
	ListSortBy   string    `json:"list_sort_by"`
	ListSortDesc bool      `json:"list_sort_desc"`
	UpdatedAt    time.Time `json:"updated_at"`
}
