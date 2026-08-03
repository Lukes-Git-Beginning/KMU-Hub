package trigger

import (
	"sync"

	"github.com/kmuhub/kmuhub/internal/notification/event"
)

// TriggerRegistry holds all registered trigger definitions.
type TriggerRegistry struct {
	mu       sync.RWMutex
	triggers map[string]*TriggerDefinition
}

// NewTriggerRegistry creates a new registry and registers all built-in triggers.
func NewTriggerRegistry() *TriggerRegistry {
	r := &TriggerRegistry{
		triggers: make(map[string]*TriggerDefinition),
	}
	r.registerBuiltins()
	return r
}

// Register adds a trigger definition to the registry.
func (r *TriggerRegistry) Register(def *TriggerDefinition) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.triggers[def.Type] = def
}

// Get returns a trigger definition by type.
func (r *TriggerRegistry) Get(triggerType string) (*TriggerDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	def, ok := r.triggers[triggerType]
	return def, ok
}

// All returns all registered trigger definitions.
func (r *TriggerRegistry) All() []*TriggerDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	defs := make([]*TriggerDefinition, 0, len(r.triggers))
	for _, def := range r.triggers {
		defs = append(defs, def)
	}
	return defs
}

// Count returns the number of registered triggers.
func (r *TriggerRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.triggers)
}

// stringOps returns common string comparison operators.
func stringOps() []string {
	return []string{"equals", "not_equals", "contains", "not_contains", "starts_with", "ends_with", "in", "not_in"}
}

// numericOps returns common numeric comparison operators.
func numericOps() []string {
	return []string{"equals", "not_equals", "gt", "gte", "lt", "lte"}
}

// registerBuiltins registers all 14 built-in trigger definitions.
func (r *TriggerRegistry) registerBuiltins() {
	// =========================================================================
	// CRM triggers (3)
	// =========================================================================
	r.Register(&TriggerDefinition{
		Type:        "crm.deal.stage_changed",
		Module:      "crm",
		Name:        "Deal-Phase geaendert",
		Description: "Wird ausgeloest, wenn sich die Phase eines Deals aendert",
		EventType:   event.EventCRMDealStageChanged,
		Fields: []TriggerField{
			{Key: "deal.value", Label: "Deal-Wert", Type: "number", Operators: numericOps()},
			{Key: "deal.stage_name", Label: "Neue Phase", Type: "string", Operators: stringOps()},
			{Key: "deal.stage_old", Label: "Alte Phase", Type: "string", Operators: stringOps()},
			{Key: "deal.owner_id", Label: "Deal-Besitzer", Type: "string", Operators: []string{"equals", "not_equals"}},
		},
	})

	r.Register(&TriggerDefinition{
		Type:        "crm.contact.created",
		Module:      "crm",
		Name:        "Kontakt erstellt",
		Description: "Wird ausgeloest, wenn ein neuer Kontakt erstellt wird",
		EventType:   "crm.contact.created",
		Fields: []TriggerField{
			{Key: "contact.email", Label: "E-Mail", Type: "string", Operators: stringOps()},
			{Key: "contact.name", Label: "Name", Type: "string", Operators: stringOps()},
			{Key: "contact.tags", Label: "Tags", Type: "string", Operators: []string{"contains", "not_contains"}},
		},
	})

	r.Register(&TriggerDefinition{
		Type:        "crm.deal.assigned",
		Module:      "crm",
		Name:        "Deal zugewiesen",
		Description: "Wird ausgeloest, wenn ein Deal einem Benutzer zugewiesen wird",
		EventType:   event.EventCRMDealAssigned,
		Fields: []TriggerField{
			{Key: "deal.value", Label: "Deal-Wert", Type: "number", Operators: numericOps()},
			{Key: "deal.owner_id", Label: "Neuer Besitzer", Type: "string", Operators: []string{"equals", "not_equals"}},
		},
	})

	// =========================================================================
	// Work triggers (3)
	// =========================================================================
	r.Register(&TriggerDefinition{
		Type:        "work.task.created",
		Module:      "work",
		Name:        "Aufgabe erstellt",
		Description: "Wird ausgeloest, wenn eine neue Aufgabe erstellt wird",
		EventType:   event.EventWorkTaskCreated,
		Fields: []TriggerField{
			{Key: "task.priority", Label: "Prioritaet", Type: "string", Operators: stringOps()},
			{Key: "task.project_id", Label: "Projekt-ID", Type: "string", Operators: []string{"equals", "not_equals"}},
		},
	})

	r.Register(&TriggerDefinition{
		Type:        "work.task.status_changed",
		Module:      "work",
		Name:        "Aufgaben-Status geaendert",
		Description: "Wird ausgeloest, wenn sich der Status einer Aufgabe aendert",
		EventType:   event.EventWorkTaskStatusChanged,
		Fields: []TriggerField{
			{Key: "task.status", Label: "Neuer Status", Type: "string", Operators: stringOps()},
			{Key: "task.priority", Label: "Prioritaet", Type: "string", Operators: stringOps()},
			{Key: "task.assignee_id", Label: "Zugewiesener", Type: "string", Operators: []string{"equals", "not_equals"}},
		},
	})

	r.Register(&TriggerDefinition{
		Type:        "work.task.completed",
		Module:      "work",
		Name:        "Aufgabe abgeschlossen",
		Description: "Wird ausgeloest, wenn eine Aufgabe als abgeschlossen markiert wird",
		EventType:   event.EventWorkTaskStatusChanged, // derived: status is_closed
		Fields: []TriggerField{
			{Key: "task.title", Label: "Titel", Type: "string", Operators: stringOps()},
			{Key: "task.project_id", Label: "Projekt-ID", Type: "string", Operators: []string{"equals", "not_equals"}},
		},
	})

	// =========================================================================
	// Email triggers (2)
	// =========================================================================
	r.Register(&TriggerDefinition{
		Type:        "email.message.received",
		Module:      "email",
		Name:        "E-Mail empfangen",
		Description: "Wird ausgeloest, wenn eine neue E-Mail empfangen wird",
		EventType:   event.EventEmailReceived,
		Fields: []TriggerField{
			{Key: "email.from", Label: "Absender", Type: "string", Operators: stringOps()},
			{Key: "email.subject", Label: "Betreff", Type: "string", Operators: stringOps()},
			{Key: "email.has_crm_contact", Label: "Hat CRM-Kontakt", Type: "boolean", Operators: []string{"equals"}},
		},
	})

	r.Register(&TriggerDefinition{
		Type:        "email.message.sent",
		Module:      "email",
		Name:        "E-Mail gesendet",
		Description: "Wird ausgeloest, wenn eine E-Mail gesendet wird",
		EventType:   event.EventEmailSent,
		Fields: []TriggerField{
			{Key: "email.to", Label: "Empfaenger", Type: "string", Operators: stringOps()},
			{Key: "email.subject", Label: "Betreff", Type: "string", Operators: stringOps()},
		},
	})

	// =========================================================================
	// Finance triggers (3)
	// =========================================================================
	r.Register(&TriggerDefinition{
		Type:        "biz.invoice.sent",
		Module:      "biz",
		Name:        "Rechnung versendet",
		Description: "Wird ausgeloest, wenn eine Rechnung versendet wird",
		EventType:   event.EventInvoiceSent,
		Fields: []TriggerField{
			{Key: "invoice.total", Label: "Rechnungsbetrag", Type: "number", Operators: numericOps()},
			{Key: "invoice.status", Label: "Status", Type: "string", Operators: stringOps()},
		},
	})

	r.Register(&TriggerDefinition{
		Type:        "biz.invoice.overdue",
		Module:      "biz",
		Name:        "Rechnung ueberfaellig",
		Description: "Wird ausgeloest, wenn eine Rechnung ueberfaellig wird (zeitbasiert, Pruefung alle 5 Minuten)",
		EventType:   event.EventInvoiceOverdue,
		TimeBased:   true,
		Fields: []TriggerField{
			{Key: "invoice.days_overdue", Label: "Tage ueberfaellig", Type: "number", Operators: numericOps()},
			{Key: "invoice.total", Label: "Rechnungsbetrag", Type: "number", Operators: numericOps()},
			{Key: "invoice.status", Label: "Status", Type: "string", Operators: stringOps()},
		},
	})

	r.Register(&TriggerDefinition{
		Type:        "biz.quote.created",
		Module:      "biz",
		Name:        "Angebot erstellt",
		Description: "Wird ausgeloest, wenn ein neues Angebot erstellt wird",
		EventType:   event.EventQuoteCreated,
		Fields: []TriggerField{
			{Key: "quote.total", Label: "Angebotsbetrag", Type: "number", Operators: numericOps()},
			{Key: "quote.status", Label: "Status", Type: "string", Operators: stringOps()},
		},
	})

	// =========================================================================
	// HR triggers (2)
	// =========================================================================
	r.Register(&TriggerDefinition{
		Type:        "hr.leave.approved",
		Module:      "hr",
		Name:        "Urlaub genehmigt",
		Description: "Wird ausgeloest, wenn ein Urlaubsantrag genehmigt wird",
		EventType:   event.EventLeaveApproved,
		Fields: []TriggerField{
			{Key: "leave.type", Label: "Urlaubsart", Type: "string", Operators: stringOps()},
			{Key: "leave.days", Label: "Anzahl Tage", Type: "number", Operators: numericOps()},
			{Key: "leave.employee_name", Label: "Mitarbeitername", Type: "string", Operators: stringOps()},
		},
	})

	r.Register(&TriggerDefinition{
		Type:        "hr.shift.ended",
		Module:      "hr",
		Name:        "Schichtende",
		Description: "Wird ausgeloest, wenn eine Schicht beendet wird",
		EventType:   event.EventShiftEnded,
		Fields: []TriggerField{
			{Key: "shift.duration_minutes", Label: "Dauer (Minuten)", Type: "number", Operators: numericOps()},
		},
	})

	// =========================================================================
	// Calendar trigger (1) -- synthetic, time-based
	// =========================================================================
	r.Register(&TriggerDefinition{
		Type:        "calendar.event.upcoming",
		Module:      "calendar",
		Name:        "Termin in Kuerze",
		Description: "Wird 15 Minuten vor einem Kalenderereignis ausgeloest (zeitbasiert)",
		EventType:   "calendar.event.upcoming",
		TimeBased:   true,
		Fields: []TriggerField{
			{Key: "event.title", Label: "Titel", Type: "string", Operators: stringOps()},
			{Key: "event.minutes_until", Label: "Minuten bis Start", Type: "number", Operators: numericOps()},
		},
	})

	// =========================================================================
	// Dialer triggers (3)
	// =========================================================================
	r.Register(&TriggerDefinition{
		Type:        "dialer.call.outcome_logged",
		Module:      "dialer",
		Name:        "Anrufergebnis protokolliert",
		Description: "Wird ausgeloest, wenn das Ergebnis eines Dialer-Anrufs protokolliert wird",
		EventType:   event.EventDialerCallOutcomeLogged,
		Fields: []TriggerField{
			{Key: "call.duration_seconds", Label: "Anrufdauer (Sek.)", Type: "number", Operators: numericOps()},
			{Key: "call.outcome", Label: "Ergebnis", Type: "string", Operators: stringOps()},
			{Key: "call.outcome_is_positive", Label: "Positives Ergebnis", Type: "boolean", Operators: []string{"equals"}},
			{Key: "call.contact_id", Label: "Kontakt-ID", Type: "string", Operators: []string{"equals", "not_equals"}},
		},
	})

	r.Register(&TriggerDefinition{
		Type:        "dialer.campaign.completed",
		Module:      "dialer",
		Name:        "Kampagne abgeschlossen",
		Description: "Wird ausgeloest, wenn alle Kontakte einer Kampagne abgearbeitet sind",
		EventType:   event.EventDialerCampaignCompleted,
		Fields: []TriggerField{
			{Key: "campaign.name", Label: "Kampagnenname", Type: "string", Operators: stringOps()},
			{Key: "campaign.total_contacts", Label: "Gesamtkontakte", Type: "number", Operators: numericOps()},
			{Key: "campaign.positive_rate", Label: "Erfolgsquote (%)", Type: "number", Operators: numericOps()},
		},
	})

	r.Register(&TriggerDefinition{
		Type:        "dialer.contact.callback_scheduled",
		Module:      "dialer",
		Name:        "Wiedervorlage geplant",
		Description: "Wird ausgeloest, wenn ein Kontakt zur Wiedervorlage markiert wird",
		EventType:   event.EventDialerCallbackScheduled,
		Fields: []TriggerField{
			{Key: "callback.contact_id", Label: "Kontakt-ID", Type: "string", Operators: []string{"equals", "not_equals"}},
			{Key: "callback.scheduled_at", Label: "Geplant fuer", Type: "date", Operators: numericOps()},
			{Key: "callback.agent_id", Label: "Agent-ID", Type: "string", Operators: []string{"equals", "not_equals"}},
		},
	})

	// =========================================================================
	// Automation trigger (1) -- generic inbound webhook
	// =========================================================================
	r.Register(&TriggerDefinition{
		Type:        "webhook.received",
		Module:      "automation",
		Name:        "Webhook empfangen",
		Description: "Wird ausgeloest, wenn ein externer Dienst den Webhook dieser Automation aufruft",
		EventType:   "automation.webhook.received",
		Fields: []TriggerField{
			{Key: "webhook.body", Label: "Payload (JSON)", Type: "string", Operators: stringOps()},
		},
	})
}
