package customfield

import "time"

// ValidFieldTypes lists all supported custom field type values.
var ValidFieldTypes = map[string]struct{}{
	"text":         {},
	"number":       {},
	"date":         {},
	"boolean":      {},
	"select":       {},
	"multi_select": {},
	"url":          {},
	"email":        {},
	"phone":        {},
}

// Definition represents a tenant-scoped custom field schema for tasks.
type Definition struct {
	ID        string
	TenantID  string
	Name      string
	FieldType string
	Options   []string
	Position  int
	CreatedAt time.Time
	UpdatedAt time.Time
}
