# Pilot-0 Onboarding

> Status: Skelett. Inhaltliche Befuellung in Sprint 5/6 vor Launch 2026-09-01.

Dokumentation und Materialien fuer den ersten produktiven Pilot-Kunden (ZFA, Start 2026-09-01).

## Inhalt

- [Setup-Guide](SETUP-GUIDE.md) — Erstmaliges Einrichten eines Pilot-Tenants
- [Admin-Handbook](ADMIN-HANDBOOK.md) — Tenant-Admin-Operationen (User, Rollen, Module aktivieren)
- *Datenimport-Guide* — Bexio/Lexware/CSV-Import (Sprint 5)
- *User-Quickstart* — Endbenutzer-Onboarding (Sprint 6)
- *Pilot-Vertrag-Template* — Kunde-Onboarding-Paket (Sprint 6, mit Legal)

## Begleitende Operations-Doku

Externe Operations-Themen leben in [`docs/operations/`](../operations/):

- [Runbook](../operations/RUNBOOK.md) — On-Call, Eskalation, Incident-Response
- [Backup-Strategie](../operations/BACKUP.md) — Frequenz, Recovery, RPO/RTO

## Pilot-0-Spezifika

- **Kunde:** ZFA (Zahnarztpraxis aus Luke's Netzwerk)
- **Tenant-Strategie:** Dedizierter Pilot-Tenant in der Multi-Tenant-Production-Instanz
- **Module aktiv ab Tag 1:** CRM, Helpdesk, Wiki, Dialer, Berichte, Formulare (laut Modul-Scope-Matrix)
- **Module deaktiviert via Feature-Flag:** Plugin-WASM, finance_line_items-Normalisierung (kommt in Sprint 5)
- **Support-Kanal:** Direkter Slack-/Telefon-Draht zu Luke (waehrend Pilot-Phase)
- **Pilot-Dauer:** Juli–Oktober 2026 (vier Monate Bestandskunde + Iteration), danach Handwerk-Segment ab November
