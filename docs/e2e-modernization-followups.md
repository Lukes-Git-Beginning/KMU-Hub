# E2E-Modernisierung 2026-06-05 — Deferred Followups

Session-Kontext: E2E-/Smoke-Suite gegen aktuellen API-Stand modernisiert (CI seit 2026-03-08 rot, E2E lief nie gegen den RBAC-/Option-B-Stand). Dabei mehrere Production-Bugs gefunden und gefixt. Diese Liste sammelt die bewusst NICHT in dieser Session erledigten Punkte.

## Aus dieser Session neu

| # | Thema | Detail | Prio |
|---|-------|--------|------|
| F1 | Manager/Member-Mapping für Migration 000129 | 35 Modul-Permissions (documents, email, finance, formulare, helpdesk, hr, inbox, wiki, automations, search, settings, recording) sind admin-only geseedet. Produktentscheidung pro Modul nötig: was bekommen manager (read+write?) und member (read?) — HR/Finance sensibel. | Sprint 5 |
| F2 | Tenant-Scan-Sweep über alle Repos | Muster zweimal gefunden (work/task `GetByID`, dialer `GetSessionByID`): SELECT scannt `tenant_id` nicht, nachfolgendes UPDATE filtert aber auf `model.TenantID` → 0 Rows → Phantom-404. Systematischer Sweep: jede Repo-Methode, die ein Modell lädt das später in tenant-gefilterte UPDATEs geht. | Sprint 5 |
| F3 | gRPC-Proto-Cleanup dialer.proto | 7 `tenant_id`-Felder in Request-Messages sind jetzt tote Felder (Tenant kommt aus Metadata). Bei nächster Proto-Revision entfernen (Breaking → koordiniert). | Phase D |
| F4 | CompleteWrapUp-Handler nil-Body tolerieren | `POST /dialer/calls/{id}/complete` 400t bei leerem Body — Handler könnte `io.EOF` als leeren Request behandeln (API-Ergonomie). Test sendet derzeit `{}`. | Phase D |
| F5 | dialer_call_events Tenant via Subquery | INSERTs leiten tenant_id per Subquery vom Parent ab (`dialer_campaigns`/`dialer_call_sessions`/`users`). Funktional korrekt; falls RLS auf diesen Tabellen kommt, auf explizites Spalten-Wiring umstellen. | Sprint 5 (RLS-Welle) |

## Aus dem Startprompt übernommen (Vorgängersession)

- E2E-/Smoke-Services laufen in CI als Superuser (RLS bypassed) — später auf `kmuhub_app` umstellen
- forvar-Style-Cleanup (`tc := tc` seit Go 1.22 unnötig) in tenant_isolation-Tests
- Node-20-Actions-Deprecation (actions/checkout@v4 etc.) — ab 2026-06-16 Default Node 24
- SMOKE_ADMIN_TOKEN long-lived (Sprint 5), OnlyOffice JWT (known)
