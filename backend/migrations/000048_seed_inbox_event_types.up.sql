-- Seed event types for Email, Document, Finance (Biz), HR, and Inbox modules.
-- Follows the same pattern as migration 000020 which seeded chat/crm/system event types.
INSERT INTO event_types (module_id, event_key, display_name, description, default_priority, default_in_app, default_desktop_push, default_sound) VALUES
    -- Email events
    ('email', 'email.message.received', 'E-Mail empfangen', 'Eine neue E-Mail ist eingegangen', 'normal', true, true, 'default'),
    ('email', 'email.message.sent', 'E-Mail gesendet', 'Eine E-Mail wurde erfolgreich versendet', 'low', true, false, ''),
    ('email', 'email.message.deleted', 'E-Mail geloescht', 'Eine E-Mail wurde geloescht', 'low', false, false, ''),

    -- Document events
    ('document', 'document.file.uploaded', 'Dokument hochgeladen', 'Ein neues Dokument wurde hochgeladen', 'normal', true, false, 'default'),
    ('document', 'document.file.shared', 'Dokument geteilt', 'Ein Dokument wurde mit Ihnen geteilt', 'normal', true, true, 'default'),
    ('document', 'document.file.versioned', 'Neue Dokumentversion', 'Eine neue Version eines Dokuments wurde erstellt', 'low', true, false, ''),

    -- Finance (Biz) events
    ('biz', 'biz.invoice.created', 'Rechnung erstellt', 'Eine neue Rechnung wurde erstellt', 'normal', true, false, 'default'),
    ('biz', 'biz.invoice.sent', 'Rechnung versendet', 'Eine Rechnung wurde an den Kunden versendet', 'normal', true, true, 'default'),
    ('biz', 'biz.invoice.overdue', 'Rechnung ueberfaellig', 'Eine Rechnung ist ueberfaellig', 'urgent', true, true, 'urgent'),
    ('biz', 'biz.payment.received', 'Zahlung eingegangen', 'Eine Zahlung wurde empfangen', 'normal', true, true, 'default'),
    ('biz', 'biz.quote.created', 'Angebot erstellt', 'Ein neues Angebot wurde erstellt', 'normal', true, false, 'default'),
    ('biz', 'biz.dunning.created', 'Mahnung erstellt', 'Eine Mahnung wurde erstellt', 'urgent', true, true, 'urgent'),

    -- HR events
    ('hr', 'hr.leave.requested', 'Urlaubsantrag eingereicht', 'Ein neuer Urlaubsantrag wurde eingereicht', 'normal', true, true, 'default'),
    ('hr', 'hr.leave.approved', 'Urlaubsantrag genehmigt', 'Ein Urlaubsantrag wurde genehmigt', 'normal', true, true, 'default'),
    ('hr', 'hr.leave.rejected', 'Urlaubsantrag abgelehnt', 'Ein Urlaubsantrag wurde abgelehnt', 'normal', true, true, 'default'),
    ('hr', 'hr.shift.started', 'Schicht gestartet', 'Ein Mitarbeiter hat eine Schicht begonnen', 'low', true, false, ''),
    ('hr', 'hr.shift.ended', 'Schicht beendet', 'Ein Mitarbeiter hat eine Schicht beendet', 'low', true, false, ''),

    -- Inbox events
    ('inbox', 'inbox.item.created', 'Neuer Inbox-Eintrag', 'Ein neuer Eintrag wurde der Inbox hinzugefuegt', 'normal', true, false, 'default'),
    ('inbox', 'inbox.item.assigned', 'Inbox-Eintrag zugewiesen', 'Ein Inbox-Eintrag wurde Ihnen zugewiesen', 'normal', true, true, 'default');
