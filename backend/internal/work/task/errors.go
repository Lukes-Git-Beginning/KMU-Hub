package task

import "errors"

var (
	ErrNotFound              = errors.New("task not found")
	ErrTitleRequired         = errors.New("task title is required")
	ErrTitleTooLong          = errors.New("task title must be at most 500 characters")
	ErrInvalidPriority       = errors.New("invalid task priority")
	ErrMaxDepthExceeded      = errors.New("maximum subtask nesting depth (5) exceeded")
	ErrCircularDependency    = errors.New("dependency would create a circular reference")
	ErrSelfDependency        = errors.New("a task cannot depend on itself")
	ErrDuplicateDependency   = errors.New("this dependency already exists")
	ErrInvalidDependencyType = errors.New("invalid dependency type")
	ErrNotProjectMember      = errors.New("user is not a member of this project")
	ErrViewerCannotEdit      = errors.New("viewers cannot create or edit tasks")
	ErrCannotMoveToProject   = errors.New("cannot move task to project")
	ErrTaskNumberConflict    = errors.New("task number conflict")
	ErrParentInDiffProject   = errors.New("parent task must be in the same project")
	ErrInvalidDateRange      = errors.New("start_date must not be after due_date")
)
