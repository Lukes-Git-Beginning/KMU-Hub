-- =============================================================================
-- demo-seed.sql — Cosmi CRM Demo-Daten (DACH/Deutsch)
-- =============================================================================
-- Zweck:     Kern-Tabellen mit sichtbaren Demo-Daten füllen, sodass
--            kontakte/CRM, work und finanzen im Frontend echte Daten zeigen.
-- Tenant:    00000000-0000-0000-0000-000000000001 (Bootstrap-Tenant)
-- Idempotent: Alle INSERTs mit ON CONFLICT DO NOTHING + festen UUIDs.
--             Mehrfach-Ausführung ist sicher.
-- Ausführen: als kmuhub-Superuser (umgeht RLS), z. B.:
--
--   docker exec -i docker-postgres-1 \
--     psql -U kmuhub -d kmuhub -f - < backend/seeds/demo/demo-seed.sql
--
-- ODER via SET-Workaround falls als App-User:
--   SET LOCAL app.current_tenant_id = '00000000-0000-0000-0000-000000000001';
-- =============================================================================

-- Sicherstellen, dass wir im System-Kontext laufen (RLS wird von kmuhub
-- als Superuser/BYPASSRLS-Rolle ohnehin ignoriert)
SET search_path TO public;

-- =============================================================================
-- 1. COMPANIES (8 DACH-Unternehmen)
-- =============================================================================
-- UUIDs: 22222222-0000-0000-0000-0000000000xx

INSERT INTO companies (
    id, tenant_id, name, domain, industry, employee_count,
    address, city, country, notes, created_by
) VALUES
(
    '22222222-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000001',
    'Müller & Söhne GmbH',
    'mueller-soehne.de',
    'Handwerk',
    42,
    'Hauptstraße 12',
    'Stuttgart',
    'DE',
    'Sanitär- und Heizungsbau, Stammkunde seit 2019',
    '11111111-1111-1111-1111-111111111111'
),
(
    '22222222-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000001',
    'Bäcker Konditorei Huber KG',
    'konditorei-huber.at',
    'Lebensmittel',
    18,
    'Bahnhofstraße 7',
    'Innsbruck',
    'AT',
    'Österreichischer Familienbetrieb, 3. Generation',
    '11111111-1111-1111-1111-111111111111'
),
(
    '22222222-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000001',
    'Zimmerei Schreinerei Vogel AG',
    'vogel-holzbau.ch',
    'Handwerk',
    67,
    'Industrieweg 3',
    'Zürich',
    'CH',
    'Holzbau + Innenausbau, Projekt Neubau Winterthur läuft',
    '11111111-1111-1111-1111-111111111111'
),
(
    '22222222-0000-0000-0000-000000000004',
    '00000000-0000-0000-0000-000000000001',
    'Steuerberatung Richter & Partner',
    'richter-steuerberatung.de',
    'Dienstleistungen',
    12,
    'Königsallee 55',
    'Düsseldorf',
    'DE',
    'Steuerberatung für KMUs, Referenzkunde für Finanzen-Modul',
    '11111111-1111-1111-1111-111111111111'
),
(
    '22222222-0000-0000-0000-000000000005',
    '00000000-0000-0000-0000-000000000001',
    'Elektro Brandner GmbH',
    'elektro-brandner.de',
    'Elektrotechnik',
    28,
    'Gewerbepark 8',
    'München',
    'DE',
    'Elektroinstallationen Gewerbe & Industrie',
    '11111111-1111-1111-1111-111111111111'
),
(
    '22222222-0000-0000-0000-000000000006',
    '00000000-0000-0000-0000-000000000001',
    'Logistik Weidemann e.K.',
    'weidemann-logistik.de',
    'Logistik',
    55,
    'Hafenstraße 21',
    'Hamburg',
    'DE',
    'Regionalspedition Norddeutschland',
    '11111111-1111-1111-1111-111111111111'
),
(
    '22222222-0000-0000-0000-000000000007',
    '00000000-0000-0000-0000-000000000001',
    'IT-Service Böhmer UG',
    'boehmer-it.de',
    'IT-Dienstleistungen',
    8,
    'Technikpark 4',
    'Nürnberg',
    'DE',
    'Managed Services + Cloud Migration für KMUs',
    '11111111-1111-1111-1111-111111111111'
),
(
    '22222222-0000-0000-0000-000000000008',
    '00000000-0000-0000-0000-000000000001',
    'Architekturbüro Schäfer & Kraus',
    'schaeferundkraus.de',
    'Architektur',
    15,
    'Schloßplatz 2',
    'Frankfurt am Main',
    'DE',
    'Hochbau und Sanierung, aktuell Bürokomplex-Auftrag',
    '11111111-1111-1111-1111-111111111111'
)
ON CONFLICT (id) DO NOTHING;

-- =============================================================================
-- 2. CONTACTS (12 Kontakte, verknüpft zu companies)
-- =============================================================================
-- UUIDs: 33333333-0000-0000-0000-0000000000xx
-- Hinweis: tenant_id hat DEFAULT '000...000' (alte Migration), daher
--          explizit auf '000...001' setzen. visibility DEFAULT 'shared' → ok.

INSERT INTO contacts (
    id, tenant_id, first_name, last_name, email, phone,
    company_id, position, notes, created_by, visibility, client_segment
) VALUES
(
    '33333333-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000001',
    'Klaus', 'Müller',
    'k.mueller@mueller-soehne.de',
    '+49 711 3456789',
    '22222222-0000-0000-0000-000000000001',
    'Geschäftsführer',
    'Hauptansprechpartner, bevorzugt Anrufe vor 10 Uhr',
    '11111111-1111-1111-1111-111111111111',
    'shared',
    'A'
),
(
    '33333333-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000001',
    'Sabine', 'Müller',
    's.mueller@mueller-soehne.de',
    '+49 711 3456790',
    '22222222-0000-0000-0000-000000000001',
    'Buchhaltung',
    'Zuständig für Rechnungen und Zahlungsfreigaben',
    '11111111-1111-1111-1111-111111111111',
    'shared',
    'A'
),
(
    '33333333-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000001',
    'Thomas', 'Huber',
    'thomas@konditorei-huber.at',
    '+43 512 887766',
    '22222222-0000-0000-0000-000000000002',
    'Inhaber',
    'Interessiert an Lager- und Bestellverwaltung',
    '11111111-1111-1111-1111-111111111111',
    'shared',
    'B'
),
(
    '33333333-0000-0000-0000-000000000004',
    '00000000-0000-0000-0000-000000000001',
    'Markus', 'Vogel',
    'm.vogel@vogel-holzbau.ch',
    '+41 44 2345678',
    '22222222-0000-0000-0000-000000000003',
    'Projektleiter',
    'CH-Kunde, Kommunikation auf Englisch möglich',
    '11111111-1111-1111-1111-111111111111',
    'shared',
    'A'
),
(
    '33333333-0000-0000-0000-000000000005',
    '00000000-0000-0000-0000-000000000001',
    'Andrea', 'Richter',
    'a.richter@richter-steuerberatung.de',
    '+49 211 445566',
    '22222222-0000-0000-0000-000000000004',
    'Geschäftsführerin',
    'Referenzkundin, bereit für Testimonial',
    '11111111-1111-1111-1111-111111111111',
    'shared',
    'A'
),
(
    '33333333-0000-0000-0000-000000000006',
    '00000000-0000-0000-0000-000000000001',
    'Stefan', 'Brandner',
    's.brandner@elektro-brandner.de',
    '+49 89 6677889',
    '22222222-0000-0000-0000-000000000005',
    'Inhaber & Meister',
    'Interesse an Dialer-Modul für Angebots-Nachverfolgung',
    '11111111-1111-1111-1111-111111111111',
    'shared',
    'B'
),
(
    '33333333-0000-0000-0000-000000000007',
    '00000000-0000-0000-0000-000000000001',
    'Petra', 'Weidemann',
    'p.weidemann@weidemann-logistik.de',
    '+49 40 9988776',
    '22222222-0000-0000-0000-000000000006',
    'Disponentin',
    'Hauptkontakt für Tagesgeschäft',
    '11111111-1111-1111-1111-111111111111',
    'shared',
    'B'
),
(
    '33333333-0000-0000-0000-000000000008',
    '00000000-0000-0000-0000-000000000001',
    'Gerhard', 'Weidemann',
    'g.weidemann@weidemann-logistik.de',
    '+49 40 9988770',
    '22222222-0000-0000-0000-000000000006',
    'Geschäftsführer',
    'Entscheider, nur strategische Themen',
    '11111111-1111-1111-1111-111111111111',
    'shared',
    'A'
),
(
    '33333333-0000-0000-0000-000000000009',
    '00000000-0000-0000-0000-000000000001',
    'Felix', 'Böhmer',
    'f.boehmer@boehmer-it.de',
    '+49 911 334455',
    '22222222-0000-0000-0000-000000000007',
    'Gründer & CTO',
    'Tech-affin, möglicher Partner für Plugin-Entwicklung',
    '11111111-1111-1111-1111-111111111111',
    'shared',
    'A'
),
(
    '33333333-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000001',
    'Claudia', 'Schäfer',
    'c.schaefer@schaeferundkraus.de',
    '+49 69 8877665',
    '22222222-0000-0000-0000-000000000008',
    'Partnerin',
    'Zuständig für Projektvergabe und Beauftragungen',
    '11111111-1111-1111-1111-111111111111',
    'shared',
    'A'
),
(
    '33333333-0000-0000-0000-000000000011',
    '00000000-0000-0000-0000-000000000001',
    'Michael', 'Kraus',
    'm.kraus@schaeferundkraus.de',
    '+49 69 8877666',
    '22222222-0000-0000-0000-000000000008',
    'Partner',
    'Technischer Ansprechpartner für Bauprojekte',
    '11111111-1111-1111-1111-111111111111',
    'shared',
    'B'
),
(
    '33333333-0000-0000-0000-000000000012',
    '00000000-0000-0000-0000-000000000001',
    'Julia', 'Hartmann',
    'j.hartmann@gmail.com',
    '+49 176 22334455',
    NULL,
    'Freie Unternehmensberaterin',
    'Netzwerk-Kontakt, empfiehlt uns weiter',
    '11111111-1111-1111-1111-111111111111',
    'shared',
    'C'
)
ON CONFLICT (id) DO NOTHING;

-- =============================================================================
-- 3. DEALS (8 Deals über alle 6 Pipeline-Stages verteilt)
-- =============================================================================
-- Stage-UUIDs (aus Bootstrap-Migration 000008):
--   Lead:        3e90ed01-37fa-47a9-a269-51acab0033c3
--   Qualified:   d0de0095-b7c6-43b0-8026-53ed4dd95a0b
--   Proposal:    74e1a7ef-4fa2-4250-a698-16a6491c5e31
--   Negotiation: 4409c1f6-be1f-40ca-be76-058982b95ce3
--   Won:         bd589abe-eacc-4d0c-8336-d2f74a7ea109
--   Lost:        81f5d925-0c28-4f82-9783-eed17e2727e5
-- UUIDs: 44444444-0000-0000-0000-0000000000xx

INSERT INTO deals (
    id, tenant_id, name, value, currency, stage_id,
    contact_id, company_id, owner_id,
    expected_close_date, notes, created_by
) VALUES
(
    '44444444-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000001',
    'Jahresvertrag Heizungswartung 2027',
    18500.00, 'EUR',
    '4409c1f6-be1f-40ca-be76-058982b95ce3',  -- Negotiation
    '33333333-0000-0000-0000-000000000001',
    '22222222-0000-0000-0000-000000000001',
    '11111111-1111-1111-1111-111111111111',
    '2026-07-31',
    'Verhandlung läuft, Rabattfrage offen (ca. 5%)',
    '11111111-1111-1111-1111-111111111111'
),
(
    '44444444-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000001',
    'Cosmi-Einführung Holzbau Vogel',
    9800.00, 'EUR',
    '74e1a7ef-4fa2-4250-a698-16a6491c5e31',  -- Proposal
    '33333333-0000-0000-0000-000000000004',
    '22222222-0000-0000-0000-000000000003',
    '11111111-1111-1111-1111-111111111111',
    '2026-08-15',
    'Angebot verschickt, Rückmeldung bis 30.06 erbeten',
    '11111111-1111-1111-1111-111111111111'
),
(
    '44444444-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000001',
    'Steuerberatung Richter — Finanzen-Modul Pilot',
    4200.00, 'EUR',
    'bd589abe-eacc-4d0c-8336-d2f74a7ea109',  -- Won
    '33333333-0000-0000-0000-000000000005',
    '22222222-0000-0000-0000-000000000004',
    '11111111-1111-1111-1111-111111111111',
    '2026-06-01',
    'Abgeschlossen, Onboarding läuft seit Anfang Juni',
    '11111111-1111-1111-1111-111111111111'
),
(
    '44444444-0000-0000-0000-000000000004',
    '00000000-0000-0000-0000-000000000001',
    'Elektro Brandner — Dialer & CRM Paket',
    6750.00, 'EUR',
    'd0de0095-b7c6-43b0-8026-53ed4dd95a0b',  -- Qualified
    '33333333-0000-0000-0000-000000000006',
    '22222222-0000-0000-0000-000000000005',
    '11111111-1111-1111-1111-111111111111',
    '2026-09-30',
    'Qualifiziert, Demo-Termin vereinbart für KW 26',
    '11111111-1111-1111-1111-111111111111'
),
(
    '44444444-0000-0000-0000-000000000005',
    '00000000-0000-0000-0000-000000000001',
    'Weidemann Logistik — Vollpaket',
    22000.00, 'EUR',
    '74e1a7ef-4fa2-4250-a698-16a6491c5e31',  -- Proposal
    '33333333-0000-0000-0000-000000000008',
    '22222222-0000-0000-0000-000000000006',
    '11111111-1111-1111-1111-111111111111',
    '2026-10-15',
    'Großauftrag, Angebot inkl. 3-Jahres-Abo erstellt',
    '11111111-1111-1111-1111-111111111111'
),
(
    '44444444-0000-0000-0000-000000000006',
    '00000000-0000-0000-0000-000000000001',
    'Böhmer IT — Reseller-Partnerschaft',
    0.00, 'EUR',
    '3e90ed01-37fa-47a9-a269-51acab0033c3',  -- Lead
    '33333333-0000-0000-0000-000000000009',
    '22222222-0000-0000-0000-000000000007',
    '11111111-1111-1111-1111-111111111111',
    '2026-12-31',
    'Erstgespräch geführt, Interesse an White-Label-Deal',
    '11111111-1111-1111-1111-111111111111'
),
(
    '44444444-0000-0000-0000-000000000007',
    '00000000-0000-0000-0000-000000000001',
    'Architekturbüro Schäfer & Kraus — Projektmanagement',
    11300.00, 'EUR',
    '4409c1f6-be1f-40ca-be76-058982b95ce3',  -- Negotiation
    '33333333-0000-0000-0000-000000000010',
    '22222222-0000-0000-0000-000000000008',
    '11111111-1111-1111-1111-111111111111',
    '2026-08-31',
    'Letzter Einwand: DSGVO-Konformität (geklärt), finale Unterschrift ausstehend',
    '11111111-1111-1111-1111-111111111111'
),
(
    '44444444-0000-0000-0000-000000000008',
    '00000000-0000-0000-0000-000000000001',
    'Konditorei Huber — CRM Basis',
    2900.00, 'EUR',
    '81f5d925-0c28-4f82-9783-eed17e2727e5',  -- Lost
    '33333333-0000-0000-0000-000000000003',
    '22222222-0000-0000-0000-000000000002',
    '11111111-1111-1111-1111-111111111111',
    '2026-05-31',
    'Entschieden für Konkurrenzprodukt (Preis). Reaktivierung Q4 möglich.',
    '11111111-1111-1111-1111-111111111111'
)
ON CONFLICT (id) DO NOTHING;

-- =============================================================================
-- 4. PROJECTS (3 Projekte)
-- =============================================================================
-- UUIDs: 55555555-0000-0000-0000-0000000000xx
-- ACHTUNG: project_key UNIQUE für nicht-archivierte Projekte

INSERT INTO projects (
    id, name, description, project_key, next_task_number,
    is_template, created_by
) VALUES
(
    '55555555-0000-0000-0000-000000000001',
    'Cosmi Launch 2026',
    'Alle internen Aufgaben rund um den Produktlaunch im September 2026. Koordination Marketing, Technik und Sales.',
    'LAUNCH',
    1,
    false,
    '11111111-1111-1111-1111-111111111111'
),
(
    '55555555-0000-0000-0000-000000000002',
    'Onboarding Richter & Partner',
    'Kundenprojekt: Einführung Cosmi Finanzen-Modul bei Steuerberatung Richter & Partner in Düsseldorf.',
    'OB-RICH',
    1,
    false,
    '11111111-1111-1111-1111-111111111111'
),
(
    '55555555-0000-0000-0000-000000000003',
    'Website Relaunch zentria.tech',
    'Neugestaltung und Content-Refresh der Unternehmenswebsite. Ziel: Launch parallel zum Produkt-Launch.',
    'WEB',
    1,
    false,
    '11111111-1111-1111-1111-111111111111'
)
ON CONFLICT (id) DO NOTHING;

-- =============================================================================
-- 5. PROJECT MEMBERS (Demo-User als Owner in allen Projekten)
-- =============================================================================

INSERT INTO project_members (project_id, user_id, role, added_at, tenant_id) VALUES
    ('55555555-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', 'owner', NOW(), '00000000-0000-0000-0000-000000000001'),
    ('55555555-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', 'owner', NOW(), '00000000-0000-0000-0000-000000000001'),
    ('55555555-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111', 'owner', NOW(), '00000000-0000-0000-0000-000000000001')
ON CONFLICT (project_id, user_id) DO NOTHING;

-- =============================================================================
-- 6. PROJECT STATUSES (pro Projekt: Todo, In Arbeit, Review, Erledigt)
-- =============================================================================
-- UUIDs: 66666666-0000-0000-0000-xxxx0000000y
--   Projekt 1 (LAUNCH): 66666666-0000-0000-0000-000100000001..4
--   Projekt 2 (OB-RICH): 66666666-0000-0000-0000-000200000001..4
--   Projekt 3 (WEB):    66666666-0000-0000-0000-000300000001..4

INSERT INTO project_statuses (
    id, project_id, tenant_id, name, color, sort_order, is_default, is_closed
) VALUES
-- LAUNCH
('66666666-0000-0000-0000-000100000001', '55555555-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'Offen',     '#6b7280', 0, true,  false),
('66666666-0000-0000-0000-000100000002', '55555555-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'In Arbeit', '#3b82f6', 1, false, false),
('66666666-0000-0000-0000-000100000003', '55555555-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'Review',    '#f59e0b', 2, false, false),
('66666666-0000-0000-0000-000100000004', '55555555-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'Erledigt',  '#22c55e', 3, false, true),
-- OB-RICH
('66666666-0000-0000-0000-000200000001', '55555555-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'Offen',     '#6b7280', 0, true,  false),
('66666666-0000-0000-0000-000200000002', '55555555-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'In Arbeit', '#3b82f6', 1, false, false),
('66666666-0000-0000-0000-000200000003', '55555555-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'Review',    '#f59e0b', 2, false, false),
('66666666-0000-0000-0000-000200000004', '55555555-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'Erledigt',  '#22c55e', 3, false, true),
-- WEB
('66666666-0000-0000-0000-000300000001', '55555555-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'Offen',     '#6b7280', 0, true,  false),
('66666666-0000-0000-0000-000300000002', '55555555-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'In Arbeit', '#3b82f6', 1, false, false),
('66666666-0000-0000-0000-000300000003', '55555555-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'Review',    '#f59e0b', 2, false, false),
('66666666-0000-0000-0000-000300000004', '55555555-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'Erledigt',  '#22c55e', 3, false, true)
ON CONFLICT (id) DO NOTHING;

-- =============================================================================
-- 7. TASKS (10 Tasks über die 3 Projekte verteilt)
-- =============================================================================
-- UUIDs: 77777777-0000-0000-0000-0000000000xx
-- WICHTIG: task_number ist UNIQUE per project_id (nicht global)
-- tasks hat KEIN tenant_id — Isolation läuft über project_id → project → user

INSERT INTO tasks (
    id, project_id, task_number, title, description,
    status_id, priority, assignee_id, due_date, created_by
) VALUES
-- LAUNCH-Projekt (Projekt 1)
(
    '77777777-0000-0000-0000-000000000001',
    '55555555-0000-0000-0000-000000000001',
    1,
    'Landing-Page-Texte finalisieren',
    'Hero-Headline, drei Feature-Blöcke und Pricing-Abschnitt auf deutsch reviewen. Umlaut-Prüfung nicht vergessen.',
    '66666666-0000-0000-0000-000100000002',  -- In Arbeit
    'high',
    '11111111-1111-1111-1111-111111111111',
    '2026-07-10',
    '11111111-1111-1111-1111-111111111111'
),
(
    '77777777-0000-0000-0000-000000000002',
    '55555555-0000-0000-0000-000000000001',
    2,
    'Pressekit erstellen (PDF + Logos)',
    'Pressemitteilung zum Launch, Produktscreenshots, Zentria-Logos in SVG und PNG.',
    '66666666-0000-0000-0000-000100000001',  -- Offen
    'medium',
    '11111111-1111-1111-1111-111111111111',
    '2026-07-25',
    '11111111-1111-1111-1111-111111111111'
),
(
    '77777777-0000-0000-0000-000000000003',
    '55555555-0000-0000-0000-000000000001',
    3,
    'Beta-Tester-Feedback auswerten',
    '12 Beta-Nutzer haben Feedback gegeben. Kritische Punkte priorisieren und als Tickets anlegen.',
    '66666666-0000-0000-0000-000100000003',  -- Review
    'urgent',
    '11111111-1111-1111-1111-111111111111',
    '2026-06-30',
    '11111111-1111-1111-1111-111111111111'
),
(
    '77777777-0000-0000-0000-000000000004',
    '55555555-0000-0000-0000-000000000001',
    4,
    'Launch-Event-Einladungen versenden',
    'LinkedIn-Post + E-Mail-Kampagne an Warteliste (ca. 230 Kontakte).',
    '66666666-0000-0000-0000-000100000004',  -- Erledigt
    'high',
    '11111111-1111-1111-1111-111111111111',
    '2026-06-15',
    '11111111-1111-1111-1111-111111111111'
),
-- OB-RICH-Projekt (Projekt 2)
(
    '77777777-0000-0000-0000-000000000005',
    '55555555-0000-0000-0000-000000000002',
    1,
    'Stammdaten Richter & Partner importieren',
    'CSV-Export aus Datev importieren. Kontakte, Mandanten, offene Rechnungen als Seed.',
    '66666666-0000-0000-0000-000200000004',  -- Erledigt
    'urgent',
    '11111111-1111-1111-1111-111111111111',
    '2026-06-05',
    '11111111-1111-1111-1111-111111111111'
),
(
    '77777777-0000-0000-0000-000000000006',
    '55555555-0000-0000-0000-000000000002',
    2,
    'Schulung Finanzen-Modul (Andrea Richter)',
    '2-stündige Zoom-Schulung: Angebote, Rechnungen, Mahnwesen. Screencast-Aufzeichnung für Doku.',
    '66666666-0000-0000-0000-000200000002',  -- In Arbeit
    'high',
    '11111111-1111-1111-1111-111111111111',
    '2026-07-03',
    '11111111-1111-1111-1111-111111111111'
),
(
    '77777777-0000-0000-0000-000000000007',
    '55555555-0000-0000-0000-000000000002',
    3,
    'DATEV-Schnittstelle konfigurieren',
    'Buchungsexport via DATEV-Format aktivieren. Testimport mit 10 Buchungen prüfen.',
    '66666666-0000-0000-0000-000200000001',  -- Offen
    'medium',
    '11111111-1111-1111-1111-111111111111',
    '2026-07-20',
    '11111111-1111-1111-1111-111111111111'
),
-- WEB-Projekt (Projekt 3)
(
    '77777777-0000-0000-0000-000000000008',
    '55555555-0000-0000-0000-000000000003',
    1,
    'Astro-Seitenstruktur festlegen',
    'Sitemap: Home, Funktionen (6 Seiten), Preise, Blog, Kontakt. Navigation und Footer.',
    '66666666-0000-0000-0000-000300000004',  -- Erledigt
    'high',
    '11111111-1111-1111-1111-111111111111',
    '2026-06-10',
    '11111111-1111-1111-1111-111111111111'
),
(
    '77777777-0000-0000-0000-000000000009',
    '55555555-0000-0000-0000-000000000003',
    2,
    'Hero-Sektion animieren (Motion)',
    'Einlauf-Animation für Headline + CTA. Nur transform/opacity, keine Layout-Animations.',
    '66666666-0000-0000-0000-000300000002',  -- In Arbeit
    'medium',
    '11111111-1111-1111-1111-111111111111',
    '2026-07-15',
    '11111111-1111-1111-1111-111111111111'
),
(
    '77777777-0000-0000-0000-000000000010',
    '55555555-0000-0000-0000-000000000003',
    3,
    'SEO-Metadaten und OG-Tags pflegen',
    'Alle Seiten: title, description, og:image (1200x630). Keywords auf DACH-KMU ausrichten.',
    '66666666-0000-0000-0000-000300000001',  -- Offen
    'low',
    '11111111-1111-1111-1111-111111111111',
    '2026-08-01',
    '11111111-1111-1111-1111-111111111111'
)
ON CONFLICT (id) DO NOTHING;

-- =============================================================================
-- 8. FINANCE — company_settings (Pflicht-Grundkonfiguration für Finanzen-Modul)
-- =============================================================================

INSERT INTO company_settings (
    id, tenant_id, name, street, plz, city, country,
    steuernummer, ust_id_nr, handelsregister,
    bank_name, iban, bic, logo_url, accent_color,
    is_kleinunternehmer, default_payment_terms_days, default_quote_validity_days
) VALUES (
    'aaaaaaaa-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000001',
    'Zentria GmbH',
    'Musterstraße 1',
    '80331',
    'München',
    'DE',
    '123/456/78901',
    'DE123456789',
    'HRB 123456 AG München',
    'Deutsche Bank AG',
    'DE89370400440532013000',
    'DEUTDEDB',
    '',
    '#0ea5e9',
    false,
    30,
    30
) ON CONFLICT (tenant_id) DO NOTHING;

-- =============================================================================
-- 9. FINANCE — Nummernkreise 2026
-- =============================================================================

INSERT INTO finance_number_sequences (
    id, tenant_id, document_type, prefix, current_number, fiscal_year
) VALUES
(
    'bbbbbbbb-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000001',
    'invoice', 'RE', 3, 2026
),
(
    'bbbbbbbb-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000001',
    'quote', 'AN', 2, 2026
),
(
    'bbbbbbbb-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000001',
    'credit_note', 'GS', 0, 2026
)
ON CONFLICT (tenant_id, document_type, fiscal_year) DO NOTHING;

-- =============================================================================
-- 10. FINANCE — Rechnungen (3 Demo-Rechnungen)
-- =============================================================================
-- contact_id REFERENCES contacts — ohne FK auf tenant_id, daher kompatibel

-- ENTFERNT: finance_invoices nutzt eine separate finance_line_items-Tabelle,
-- nicht eine line_items-JSON-Spalte. Separat fixen (nicht Teil des ersten
-- Mock-Exit-Ziels kontakte/work).

-- =============================================================================
-- 11. FINANCE — Angebot (1 Demo-Angebot)
-- =============================================================================

-- ENTFERNT: finance_quotes — gleicher line_items-Mismatch wie finance_invoices.

-- =============================================================================
-- FERTIG — Zusammenfassung:
--   companies:                8 Zeilen
--   contacts:                12 Zeilen
--   deals:                    8 Zeilen (über alle 6 Stages)
--   projects:                 3 Zeilen
--   project_members:          3 Zeilen
--   project_statuses:        12 Zeilen (4 pro Projekt)
--   tasks:                   10 Zeilen
--   company_settings:         1 Zeile
--   finance_number_sequences: 3 Zeilen
--   finance_invoices:         3 Zeilen
--   finance_quotes:           1 Zeile
-- =============================================================================

-- =============================================================================
-- 12. DEMO-USER → ADMIN (idempotent)
-- =============================================================================
-- Der Demo-User wird via POST /auth/register angelegt und bekommt nur die
-- member-Rolle (contacts:read, kein write) → Create/Update/Delete = 403.
-- Damit die Demo-Umgebung volle Modul-Funktionalität zeigt, wird demo@local.test
-- auf admin gehoben (206 Permissions, system-global, kein tenant_id auf user_roles).
-- Legt den User NICHT an (das macht register), hebt nur die Rolle wenn vorhanden.
DO $$
DECLARE
  v_user_id  uuid;
  v_admin_id uuid;
  v_member_id uuid;
BEGIN
  SELECT id INTO v_user_id  FROM users WHERE email = 'demo@local.test';
  SELECT id INTO v_admin_id FROM roles WHERE name = 'admin';
  SELECT id INTO v_member_id FROM roles WHERE name = 'member';
  IF v_user_id IS NOT NULL AND v_admin_id IS NOT NULL THEN
    INSERT INTO user_roles (user_id, role_id)
      VALUES (v_user_id, v_admin_id)
      ON CONFLICT DO NOTHING;
    IF v_member_id IS NOT NULL THEN
      DELETE FROM user_roles WHERE user_id = v_user_id AND role_id = v_member_id;
    END IF;
  END IF;
END $$;
