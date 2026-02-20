package template

import (
	"encoding/json"
	"time"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Templates contains all 12 pre-built automation templates.
// These are organized by category: Vertrieb, Finanzen, Personal, Kommunikation.
var Templates = []*models.AutomationTemplate{
	// =========================================================================
	// Vertrieb (Sales) -- 4 templates
	// =========================================================================

	// 1. Deal gewonnen -> Rechnung erstellen
	{
		ID:          "deal-won-invoice",
		Name:        "Deal gewonnen: Rechnung erstellen",
		Description: "Erstellt automatisch einen Rechnungsentwurf, wenn ein Deal als gewonnen markiert wird.",
		Category:    "vertrieb",
		Complexity:  models.ComplexityMedium,
		TriggerType: "crm.deal.stage_changed",
		TriggerConfig: mustJSON(map[string]any{
			"event_type": "crm.deal.stage_changed",
		}),
		Conditions: mustJSON(models.ConditionConfig{
			Mode: models.ConditionModeSimple,
			Simple: &models.Condition{
				Field:    "deal.stage_name",
				Operator: "equals",
				Value:    "Gewonnen",
			},
		}),
		Actions: mustJSON([]models.ActionConfig{
			{
				Type: "biz.create_invoice_draft",
				Config: mustJSON(map[string]any{
					"tenant_id":  "{{tenant_id}}",
					"created_by": "{{actor_id}}",
					"notes":      "Automatisch erstellt aus Deal: {{deal.name}}",
				}),
				OnError: models.OnErrorAbort,
			},
			{
				Type: "notification.send",
				Config: mustJSON(map[string]any{
					"target_user_id": "{{deal.owner_id}}",
					"title":          "Rechnung erstellt",
					"body":           "Ein Rechnungsentwurf wurde fuer Deal '{{deal.name}}' erstellt.",
				}),
				OnError: models.OnErrorContinue,
			},
		}),
		CreatedAt: time.Now(),
	},

	// 2. Deal gewonnen -> Follow-up Aufgabe
	{
		ID:          "deal-won-task",
		Name:        "Deal gewonnen: Follow-up Aufgabe",
		Description: "Erstellt eine Follow-up Aufgabe, wenn ein Deal als gewonnen markiert wird.",
		Category:    "vertrieb",
		Complexity:  models.ComplexitySimple,
		TriggerType: "crm.deal.stage_changed",
		TriggerConfig: mustJSON(map[string]any{
			"event_type": "crm.deal.stage_changed",
		}),
		Conditions: mustJSON(models.ConditionConfig{
			Mode: models.ConditionModeSimple,
			Simple: &models.Condition{
				Field:    "deal.stage_name",
				Operator: "equals",
				Value:    "Gewonnen",
			},
		}),
		Actions: mustJSON([]models.ActionConfig{
			{
				Type: "work.create_task",
				Config: mustJSON(map[string]any{
					"project_id":  "{{project_id}}",
					"title":       "Follow-up: {{deal.name}}",
					"assignee_id": "{{deal.owner_id}}",
					"priority":    "high",
					"due_days":    3,
				}),
				OnError: models.OnErrorAbort,
			},
		}),
		CreatedAt: time.Now(),
	},

	// 3. Neuer Kontakt -> Benachrichtigung
	{
		ID:          "new-contact-notification",
		Name:        "Neuer Kontakt: Benachrichtigung",
		Description: "Sendet eine Benachrichtigung, wenn ein neuer Kontakt erstellt wird.",
		Category:    "vertrieb",
		Complexity:  models.ComplexitySimple,
		TriggerType: "crm.contact.created",
		TriggerConfig: mustJSON(map[string]any{
			"event_type": "crm.contact.created",
		}),
		Actions: mustJSON([]models.ActionConfig{
			{
				Type: "notification.send",
				Config: mustJSON(map[string]any{
					"target_user_id": "{{actor_id}}",
					"title":          "Neuer Kontakt erstellt",
					"body":           "Kontakt '{{contact.name}}' wurde erstellt.",
				}),
				OnError: models.OnErrorContinue,
			},
		}),
		CreatedAt: time.Now(),
	},

	// 4. Deal-Phase geaendert -> E-Mail senden
	{
		ID:          "deal-stage-email",
		Name:        "Deal-Phase geaendert: E-Mail senden",
		Description: "Sendet eine E-Mail an den Deal-Besitzer, wenn sich die Phase aendert.",
		Category:    "vertrieb",
		Complexity:  models.ComplexityMedium,
		TriggerType: "crm.deal.stage_changed",
		TriggerConfig: mustJSON(map[string]any{
			"event_type": "crm.deal.stage_changed",
		}),
		Actions: mustJSON([]models.ActionConfig{
			{
				Type: "email.send",
				Config: mustJSON(map[string]any{
					"account_id": "{{email_account_id}}",
					"to":         "{{deal.owner_email}}",
					"subject":    "Deal-Phase geaendert: {{deal.name}}",
					"body":       "Der Deal '{{deal.name}}' wurde von '{{deal.stage_old}}' nach '{{deal.stage_name}}' verschoben.",
				}),
				OnError: models.OnErrorContinue,
			},
		}),
		CreatedAt: time.Now(),
	},

	// =========================================================================
	// Finanzen (Finance) -- 3 templates
	// =========================================================================

	// 5. Ueberfaellige Rechnung -> Mahnung (MUST-HAVE)
	{
		ID:          "invoice-overdue-dunning",
		Name:        "Ueberfaellige Rechnung: Mahnung erstellen",
		Description: "Erstellt automatisch eine Mahnung, wenn eine Rechnung ueberfaellig wird.",
		Category:    "finanzen",
		Complexity:  models.ComplexityMedium,
		TriggerType: "biz.invoice.overdue",
		TriggerConfig: mustJSON(map[string]any{
			"event_type": "biz.invoice.overdue",
		}),
		Conditions: mustJSON(models.ConditionConfig{
			Mode: models.ConditionModeSimple,
			Simple: &models.Condition{
				Field:    "invoice.days_overdue",
				Operator: "gte",
				Value:    "14",
			},
		}),
		Actions: mustJSON([]models.ActionConfig{
			{
				Type: "biz.create_dunning",
				Config: mustJSON(map[string]any{
					"tenant_id":  "{{tenant_id}}",
					"invoice_id": "{{invoice.id}}",
					"level":      "1",
					"created_by": "system",
				}),
				OnError: models.OnErrorAbort,
			},
			{
				Type: "notification.send",
				Config: mustJSON(map[string]any{
					"target_user_id": "{{actor_id}}",
					"title":          "Mahnung erstellt",
					"body":           "Eine Mahnung fuer Rechnung {{invoice.number}} wurde erstellt ({{invoice.days_overdue}} Tage ueberfaellig).",
				}),
				OnError: models.OnErrorContinue,
			},
		}),
		CreatedAt: time.Now(),
	},

	// 6. Zahlung eingegangen -> Benachrichtigung
	{
		ID:          "payment-received-notification",
		Name:        "Zahlung eingegangen: Benachrichtigung",
		Description: "Sendet eine Benachrichtigung, wenn eine Zahlung eingegangen ist.",
		Category:    "finanzen",
		Complexity:  models.ComplexitySimple,
		TriggerType: "biz.invoice.sent",
		TriggerConfig: mustJSON(map[string]any{
			"event_type": "biz.payment.received",
		}),
		Actions: mustJSON([]models.ActionConfig{
			{
				Type: "notification.send",
				Config: mustJSON(map[string]any{
					"target_user_id": "{{actor_id}}",
					"title":          "Zahlung eingegangen",
					"body":           "Eine Zahlung fuer Rechnung {{invoice.number}} wurde verbucht.",
				}),
				OnError: models.OnErrorContinue,
			},
		}),
		CreatedAt: time.Now(),
	},

	// 7. Angebot erstellt -> Follow-up Aufgabe nach 7 Tagen
	{
		ID:          "quote-followup",
		Name:        "Angebot erstellt: Follow-up nach 7 Tagen",
		Description: "Erstellt eine Follow-up Aufgabe 7 Tage nach Angebotserstellung.",
		Category:    "finanzen",
		Complexity:  models.ComplexityMedium,
		TriggerType: "biz.quote.created",
		TriggerConfig: mustJSON(map[string]any{
			"event_type": "biz.quote.created",
		}),
		Actions: mustJSON([]models.ActionConfig{
			{
				Type: "work.create_task",
				Config: mustJSON(map[string]any{
					"project_id":  "{{project_id}}",
					"title":       "Follow-up: Angebot {{quote.number}}",
					"assignee_id": "{{actor_id}}",
					"priority":    "medium",
					"due_days":    7,
				}),
				OnError: models.OnErrorAbort,
			},
		}),
		CreatedAt: time.Now(),
	},

	// =========================================================================
	// Personal (HR) -- 3 templates
	// =========================================================================

	// 8. Urlaub genehmigt -> Kalender + Benachrichtigung (MUST-HAVE)
	{
		ID:          "leave-approved-calendar",
		Name:        "Urlaub genehmigt: Kalender-Eintrag",
		Description: "Erstellt automatisch einen Kalender-Eintrag und benachrichtigt das Team, wenn Urlaub genehmigt wird.",
		Category:    "personal",
		Complexity:  models.ComplexitySimple,
		TriggerType: "hr.leave.approved",
		TriggerConfig: mustJSON(map[string]any{
			"event_type": "hr.leave.approved",
		}),
		Actions: mustJSON([]models.ActionConfig{
			{
				Type: "calendar.create_event",
				Config: mustJSON(map[string]any{
					"calendar_id": "{{calendar_id}}",
					"title":       "Urlaub: {{leave.employee_name}}",
					"is_all_day":  true,
					"start_time":  "{{leave.start_date}}",
					"end_time":    "{{leave.end_date}}",
					"created_by":  "{{actor_id}}",
				}),
				OnError: models.OnErrorAbort,
			},
			{
				Type: "notification.send",
				Config: mustJSON(map[string]any{
					"target_user_id": "{{actor_id}}",
					"title":          "Urlaub genehmigt",
					"body":           "Der Urlaub von {{leave.employee_name}} ({{leave.days}} Tage, {{leave.type}}) wurde genehmigt.",
				}),
				OnError: models.OnErrorContinue,
			},
		}),
		CreatedAt: time.Now(),
	},

	// 9. Schichtende > 9h -> Warnung an HR
	{
		ID:          "shift-overtime-alert",
		Name:        "Ueberstunden-Warnung: Schicht > 9 Stunden",
		Description: "Sendet eine Warnung an HR, wenn eine Schicht laenger als 9 Stunden dauert.",
		Category:    "personal",
		Complexity:  models.ComplexityMedium,
		TriggerType: "hr.shift.ended",
		TriggerConfig: mustJSON(map[string]any{
			"event_type": "hr.shift.ended",
		}),
		Conditions: mustJSON(models.ConditionConfig{
			Mode: models.ConditionModeSimple,
			Simple: &models.Condition{
				Field:    "shift.duration_minutes",
				Operator: "gt",
				Value:    "540",
			},
		}),
		Actions: mustJSON([]models.ActionConfig{
			{
				Type: "notification.send",
				Config: mustJSON(map[string]any{
					"target_user_id": "{{hr_manager_id}}",
					"title":          "Ueberstunden-Warnung",
					"body":           "Mitarbeiter {{shift.employee_id}} hat eine Schicht von {{shift.duration_minutes}} Minuten beendet.",
				}),
				OnError: models.OnErrorContinue,
			},
		}),
		CreatedAt: time.Now(),
	},

	// 10. Urlaubsantrag -> Manager benachrichtigen
	{
		ID:          "leave-request-notification",
		Name:        "Urlaubsantrag: Manager benachrichtigen",
		Description: "Benachrichtigt den Manager, wenn ein neuer Urlaubsantrag gestellt wird.",
		Category:    "personal",
		Complexity:  models.ComplexitySimple,
		TriggerType: "hr.leave.approved",
		TriggerConfig: mustJSON(map[string]any{
			"event_type": "hr.leave.requested",
		}),
		Actions: mustJSON([]models.ActionConfig{
			{
				Type: "notification.send",
				Config: mustJSON(map[string]any{
					"target_user_id": "{{manager_id}}",
					"title":          "Neuer Urlaubsantrag",
					"body":           "{{leave.employee_name}} hat {{leave.days}} Tage {{leave.type}} beantragt.",
				}),
				OnError: models.OnErrorContinue,
			},
		}),
		CreatedAt: time.Now(),
	},

	// =========================================================================
	// Kommunikation (Communication) -- 2 templates
	// =========================================================================

	// 11. E-Mail von CRM-Kontakt -> Aufgabe erstellen
	{
		ID:          "email-crm-contact-task",
		Name:        "E-Mail von CRM-Kontakt: Aufgabe erstellen",
		Description: "Erstellt eine Aufgabe, wenn eine E-Mail von einem bekannten CRM-Kontakt eingeht.",
		Category:    "kommunikation",
		Complexity:  models.ComplexityMedium,
		TriggerType: "email.message.received",
		TriggerConfig: mustJSON(map[string]any{
			"event_type": "email.message.received",
		}),
		Conditions: mustJSON(models.ConditionConfig{
			Mode: models.ConditionModeSimple,
			Simple: &models.Condition{
				Field:    "email.has_crm_contact",
				Operator: "equals",
				Value:    "true",
			},
		}),
		Actions: mustJSON([]models.ActionConfig{
			{
				Type: "work.create_task",
				Config: mustJSON(map[string]any{
					"project_id":  "{{project_id}}",
					"title":       "E-Mail beantworten: {{email.subject}}",
					"assignee_id": "{{actor_id}}",
					"priority":    "medium",
					"due_days":    1,
				}),
				OnError: models.OnErrorAbort,
			},
		}),
		CreatedAt: time.Now(),
	},

	// 12. Aufgabe abgeschlossen -> Team benachrichtigen
	{
		ID:          "task-completed-notification",
		Name:        "Aufgabe abgeschlossen: Team benachrichtigen",
		Description: "Benachrichtigt das Team, wenn eine Aufgabe als abgeschlossen markiert wird.",
		Category:    "kommunikation",
		Complexity:  models.ComplexitySimple,
		TriggerType: "work.task.completed",
		TriggerConfig: mustJSON(map[string]any{
			"event_type": "work.task.status_changed",
		}),
		Actions: mustJSON([]models.ActionConfig{
			{
				Type: "notification.send",
				Config: mustJSON(map[string]any{
					"target_user_id": "{{task.assignee_id}}",
					"title":          "Aufgabe abgeschlossen",
					"body":           "Die Aufgabe '{{task.title}}' wurde abgeschlossen.",
				}),
				OnError: models.OnErrorContinue,
			},
		}),
		CreatedAt: time.Now(),
	},
}

// mustJSON marshals v to json.RawMessage, panicking on error.
// Only used for compile-time template definitions.
func mustJSON(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic("mustJSON: " + err.Error())
	}
	return data
}
