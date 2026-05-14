# Tenant-Admin Handbook

> Status: Skelett. Inhaltliche Befuellung in Sprint 5/6.

Operationen, die ein Tenant-Admin selbst durchfuehren kann.

## User-Management

### User anlegen

*TODO Sprint 5: Schritt-fuer-Schritt mit Screenshots.*

- Manuell ueber Settings → Team
- Bulk-Import via CSV (Sprint 6)
- Einladungs-Flow per E-Mail

### Rollen + Berechtigungen

Verfuegbare Rollen:

- `admin` — Tenant-Admin, alle Rechte inkl. Billing + Modul-Konfiguration
- `hr_admin` — Zugriff auf HR-Daten aller Mitarbeiter (siehe [hr_document_access](../../backend/migrations/000127_rls_welle4_hr_role_based.up.sql))
- `manager` — Erweiterte Sicht auf Teammitglieder, Genehmigungen
- `member` — Standard-Mitarbeiter

*TODO Sprint 5: Role-Matrix-Tabelle pro Modul.*

### MFA-Pflicht erzwingen

*TODO Sprint 5: Settings → Security → MFA-Policy.*

## Modul-Aktivierung

Feature-Flags pro Modul. Aktive Module siehe Settings → Module.

| Modul | Default | Bemerkung |
|---|---|---|
| CRM | aktiv | Basis-Modul, deaktivieren entfernt Kontakte/Companies |
| Helpdesk | aktiv | Ticketing + SLA |
| Wiki | aktiv | Knowledge-Base |
| Dialer | aktiv | Phase-1 (Internal), PSTN-Bridging ab Q3 |
| Berichte | aktiv | Module-Reports |
| Formulare | aktiv | Custom-Forms + Workflows |
| HR | inaktiv | Aktivierung ueber Sales-Team |
| Plugins | inaktiv | WASM-Plugin-System (Phase D nach Launch) |
| Finance | inaktiv | finance_line_items-Normalisierung folgt in Sprint 4 |

## Integrationen

*TODO Sprint 5: Bexio, Lexware, DATEV-API-Setup.*

- Bexio: OAuth-Flow, Sync-Intervalle, Konfliktloesung
- Lexware: API-Key, Webhook-URL eintragen
- DATEV-API: OAuth + Mandant-Auswahl

## Backup + Audit-Log

- Audit-Log: Settings → Audit (gefiltert auf eigenen Tenant via RLS)
- Backup: zentrale Strategie, kein Tenant-Self-Service (siehe [Backup-Strategie](../operations/BACKUP.md))
- Export: GDPR-Export-Endpoint (`security/gdpr`) — Self-Service Sprint 6

## Eskalation bei Problemen

Siehe [Runbook](../operations/RUNBOOK.md). Pilot-Kunden nutzen direkten Slack-Draht.
