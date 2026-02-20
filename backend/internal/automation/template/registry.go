package template

import (
	"context"
	"log/slog"

	"github.com/kmuhub/kmuhub/internal/automation/workflow"
	"github.com/kmuhub/kmuhub/internal/models"
)

// TemplateRegistry provides access to pre-built automation templates
// and can seed them into the database.
type TemplateRegistry struct {
	templates map[string]*models.AutomationTemplate
}

// NewTemplateRegistry creates a new template registry loaded with
// all pre-built templates from templates.go.
func NewTemplateRegistry() *TemplateRegistry {
	r := &TemplateRegistry{
		templates: make(map[string]*models.AutomationTemplate),
	}

	for _, tmpl := range Templates {
		r.templates[tmpl.ID] = tmpl
	}

	return r
}

// Get returns a template by ID.
func (r *TemplateRegistry) Get(id string) (*models.AutomationTemplate, bool) {
	tmpl, ok := r.templates[id]
	return tmpl, ok
}

// All returns all registered templates.
func (r *TemplateRegistry) All() []*models.AutomationTemplate {
	result := make([]*models.AutomationTemplate, 0, len(r.templates))
	for _, tmpl := range r.templates {
		result = append(result, tmpl)
	}
	return result
}

// Count returns the number of registered templates.
func (r *TemplateRegistry) Count() int {
	return len(r.templates)
}

// SeedToDatabase upserts all pre-built templates into the database.
// This is called at startup to ensure templates are available.
func (r *TemplateRegistry) SeedToDatabase(ctx context.Context, repo workflow.TemplateRepository) error {
	for _, tmpl := range r.templates {
		if err := repo.UpsertTemplate(ctx, tmpl); err != nil {
			slog.Error("failed to seed template",
				"template_id", tmpl.ID,
				"template_name", tmpl.Name,
				"error", err,
			)
			return err
		}
	}

	slog.Info("automation templates seeded", "count", len(r.templates))
	return nil
}
