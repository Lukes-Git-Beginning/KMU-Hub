-- Seed industry templates (Phase 20)

-- Handwerk (Bau/Handwerk)
INSERT INTO industry_templates (id, slug, name, description, industry, icon, custom_fields, validation_rules, workflow_rules, default_settings)
VALUES (
    'a1b2c3d4-1111-4000-a000-000000000001',
    'handwerk',
    'Handwerk & Bau',
    'Branchenvorlage für Bau- und Handwerksbetriebe. Enthält Felder für Baustellen, Gewerke und Aufträge sowie automatische Baubeginn-Erinnerungen.',
    'Handwerk',
    'hammer',
    '[
        {"name": "baustelle", "label": "Baustelle", "type": "text"},
        {"name": "gewerk", "label": "Gewerk", "type": "select", "options": ["Rohbau", "Elektro", "Sanitaer", "Heizung", "Trockenbau", "Maler", "Dachdecker", "Sonstige"]},
        {"name": "auftrags_nr", "label": "Auftrags-Nr.", "type": "text"},
        {"name": "baubeginn", "label": "Baubeginn", "type": "date"},
        {"name": "bauende", "label": "Bauende (geplant)", "type": "date"},
        {"name": "bauherr", "label": "Bauherr", "type": "text"},
        {"name": "bauvolumen", "label": "Bauvolumen (EUR)", "type": "number"}
    ]'::jsonb,
    '[
        {
            "name": "Auftrags-Nr. Format",
            "entity_type": "contact",
            "field_name": "auftrags_nr",
            "rule_type": "regex",
            "rule_config": {"pattern": "^[A-Z]{2,4}-\\d{4,8}$", "required": false},
            "error_message": "Auftrags-Nr. muss dem Format AB-12345 entsprechen",
            "priority": 0
        },
        {
            "name": "Bauvolumen positiv",
            "entity_type": "contact",
            "field_name": "bauvolumen",
            "rule_type": "range",
            "rule_config": {"min": 0, "required": false},
            "error_message": "Bauvolumen muss positiv sein",
            "priority": 1
        }
    ]'::jsonb,
    '[
        {
            "name": "Baubeginn-Erinnerung",
            "trigger_event": "after_create_contact",
            "conditions": [
                {"field": "baubeginn", "operator": "exists"}
            ],
            "actions": [
                {
                    "type": "create_task",
                    "config": {
                        "title": "Baubeginn vorbereiten: {{baustelle}}",
                        "description": "Baustelle {{baustelle}}, Gewerk: {{gewerk}}, Auftrags-Nr: {{auftrags_nr}}",
                        "priority": "high"
                    }
                }
            ],
            "priority": 0
        }
    ]'::jsonb,
    '{}'::jsonb
) ON CONFLICT (slug) DO UPDATE SET
    name = EXCLUDED.name, description = EXCLUDED.description,
    custom_fields = EXCLUDED.custom_fields, validation_rules = EXCLUDED.validation_rules,
    workflow_rules = EXCLUDED.workflow_rules, updated_at = NOW();

-- Beratung (Dienstleistung)
INSERT INTO industry_templates (id, slug, name, description, industry, icon, custom_fields, validation_rules, workflow_rules, default_settings)
VALUES (
    'a1b2c3d4-2222-4000-a000-000000000002',
    'beratung',
    'Beratung & Dienstleistung',
    'Branchenvorlage für Beratungsunternehmen. Enthält Felder für Mandate, Stundensätze und Budgets sowie automatische Budget-Warnungen.',
    'Beratung',
    'briefcase',
    '[
        {"name": "mandat_nr", "label": "Mandat-Nr.", "type": "text"},
        {"name": "stundensatz", "label": "Stundensatz (EUR)", "type": "number"},
        {"name": "budget", "label": "Budget (EUR)", "type": "number"},
        {"name": "budget_verbraucht", "label": "Budget verbraucht (EUR)", "type": "number"},
        {"name": "projekttyp", "label": "Projekttyp", "type": "select", "options": ["Strategieberatung", "IT-Beratung", "Unternehmensberatung", "Rechtsberatung", "Steuerberatung", "Sonstige"]},
        {"name": "abrechnungsart", "label": "Abrechnungsart", "type": "select", "options": ["Stundenbasis", "Festpreis", "Retainer", "Erfolgshonorar"]}
    ]'::jsonb,
    '[
        {
            "name": "Stundensatz positiv",
            "entity_type": "contact",
            "field_name": "stundensatz",
            "rule_type": "range",
            "rule_config": {"min": 0, "max": 10000, "required": false},
            "error_message": "Stundensatz muss zwischen 0 und 10.000 liegen",
            "priority": 0
        },
        {
            "name": "Budget positiv",
            "entity_type": "contact",
            "field_name": "budget",
            "rule_type": "range",
            "rule_config": {"min": 0, "required": false},
            "error_message": "Budget muss positiv sein",
            "priority": 1
        }
    ]'::jsonb,
    '[
        {
            "name": "Budget-Warnung bei 80%",
            "trigger_event": "after_update_contact",
            "conditions": [
                {"field": "budget", "operator": "gt", "value": 0},
                {"field": "budget_verbraucht", "operator": "exists"}
            ],
            "actions": [
                {
                    "type": "send_notification",
                    "config": {
                        "message": "Budget-Warnung: Mandat {{mandat_nr}} hat 80% des Budgets erreicht",
                        "channel": "admin"
                    }
                }
            ],
            "priority": 0
        }
    ]'::jsonb,
    '{}'::jsonb
) ON CONFLICT (slug) DO UPDATE SET
    name = EXCLUDED.name, description = EXCLUDED.description,
    custom_fields = EXCLUDED.custom_fields, validation_rules = EXCLUDED.validation_rules,
    workflow_rules = EXCLUDED.workflow_rules, updated_at = NOW();

-- Handel (Gross-/Einzelhandel)
INSERT INTO industry_templates (id, slug, name, description, industry, icon, custom_fields, validation_rules, workflow_rules, default_settings)
VALUES (
    'a1b2c3d4-3333-4000-a000-000000000003',
    'handel',
    'Handel & Vertrieb',
    'Branchenvorlage für Handelsunternehmen. Enthält Felder für Liefertermine, Versand und Kreditlimits sowie automatische Liefertermin-Erinnerungen.',
    'Handel',
    'shopping-cart',
    '[
        {"name": "liefertermin", "label": "Liefertermin", "type": "date"},
        {"name": "versandart", "label": "Versandart", "type": "select", "options": ["Standard", "Express", "Spedition", "Abholung", "Paketdienst"]},
        {"name": "tracking_nr", "label": "Tracking-Nr.", "type": "text"},
        {"name": "kundennummer", "label": "Kundennummer", "type": "text"},
        {"name": "kreditlimit", "label": "Kreditlimit (EUR)", "type": "number"},
        {"name": "offene_posten", "label": "Offene Posten (EUR)", "type": "number"},
        {"name": "zahlungsziel_tage", "label": "Zahlungsziel (Tage)", "type": "number"}
    ]'::jsonb,
    '[
        {
            "name": "Kreditlimit positiv",
            "entity_type": "contact",
            "field_name": "kreditlimit",
            "rule_type": "range",
            "rule_config": {"min": 0, "required": false},
            "error_message": "Kreditlimit muss positiv sein",
            "priority": 0
        },
        {
            "name": "Zahlungsziel gueltig",
            "entity_type": "contact",
            "field_name": "zahlungsziel_tage",
            "rule_type": "range",
            "rule_config": {"min": 0, "max": 365, "required": false},
            "error_message": "Zahlungsziel muss zwischen 0 und 365 Tagen liegen",
            "priority": 1
        }
    ]'::jsonb,
    '[
        {
            "name": "Liefertermin-Erinnerung",
            "trigger_event": "after_create_contact",
            "conditions": [
                {"field": "liefertermin", "operator": "exists"}
            ],
            "actions": [
                {
                    "type": "create_task",
                    "config": {
                        "title": "Lieferung vorbereiten: {{kundennummer}}",
                        "description": "Versandart: {{versandart}}, Liefertermin: {{liefertermin}}",
                        "priority": "medium"
                    }
                }
            ],
            "priority": 0
        }
    ]'::jsonb,
    '{}'::jsonb
) ON CONFLICT (slug) DO UPDATE SET
    name = EXCLUDED.name, description = EXCLUDED.description,
    custom_fields = EXCLUDED.custom_fields, validation_rules = EXCLUDED.validation_rules,
    workflow_rules = EXCLUDED.workflow_rules, updated_at = NOW();
