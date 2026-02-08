package project

import "errors"

var (
	ErrNotFound            = errors.New("project not found")
	ErrNameRequired        = errors.New("project name is required")
	ErrNameTooShort        = errors.New("project name must be at least 3 characters")
	ErrNameTooLong         = errors.New("project name must be at most 255 characters")
	ErrKeyRequired         = errors.New("project key is required")
	ErrKeyTooLong          = errors.New("project key must be at most 10 characters")
	ErrKeyTooShort         = errors.New("project key must be at least 2 characters")
	ErrKeyInvalidChars     = errors.New("project key must contain only uppercase letters and digits")
	ErrKeyAlreadyExists    = errors.New("project key already exists")
	ErrProjectArchived     = errors.New("project is archived")
	ErrNotProjectMember    = errors.New("user is not a member of this project")
	ErrNotProjectOwner     = errors.New("user is not an owner of this project")
	ErrCannotRemoveOnlyOwner = errors.New("cannot remove the only owner of a project")
	ErrInvalidRole         = errors.New("invalid project role")
	ErrInvalidViewType     = errors.New("view type must be 'list' or 'kanban'")
	ErrTemplateNotFound    = errors.New("template project not found")
)
