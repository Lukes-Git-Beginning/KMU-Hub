# Pilot-0 Onboarding

> ⚠️ **HINFÄLLIG** seit 2026-08-12 (`.planning/launch-lagebild-2026-08-12.md`).
>
> **Es gibt keinen Pilot-0.** ZFA ist kalt und nicht mehr Fokus, die Vertriebs-Pipeline ist leer,
> der unten genannte Zeitraum „Juli–Oktober 2026" ist bereits verstrichen. Wer nach diesem Dokument
> plant, plant für einen Kunden, der nicht kommt.
>
> Der Inhalt bleibt als **generische Vorlage** stehen — Setup-Guide und Admin-Handbook sind
> kundenunabhängig brauchbar. Die ZFA-Spezifika unten sind Beispieldaten, keine Zusagen.
> Vor dem ersten echten Kunden gehört hierher außerdem, was Lagebild §10 nennt: eine haltbare
> Support-Zusage statt „direkter Draht zu Luke", drei Incident-SOPs und ein zweiter Zugriffsweg
> auf SSH-Key und Vault-Passwort.

Dokumentation und Materialien fuer einen ersten produktiven Pilot-Kunden.

## Inhalt

- [Setup-Guide](SETUP-GUIDE.md) — Erstmaliges Einrichten eines Pilot-Tenants
- [Admin-Handbook](ADMIN-HANDBOOK.md) — Tenant-Admin-Operationen (User, Rollen, Module aktivieren)
- *Datenimport-Guide* — Bexio/Lexware/CSV-Import (offen)
- *User-Quickstart* — Endbenutzer-Onboarding (offen)
- *Pilot-Vertrag-Template* — Kunde-Onboarding-Paket (offen, mit Legal — Etappe 4)

## Begleitende Operations-Doku

Externe Operations-Themen leben in [`docs/operations/`](../operations/):

- [Runbook](../operations/RUNBOOK.md) — On-Call, Eskalation, Incident-Response
- [Backup-Strategie](../operations/BACKUP.md) — Frequenz, Recovery, RPO/RTO

## Beispiel-Zuschnitt eines Piloten (⚠ Platzhalter, kein zugesagter Kunde)

- **Kunde:** ~~ZFA (Zahnarztpraxis aus Luke's Netzwerk)~~ — **kalt seit 2026-08-12, kein Pilot in Aussicht**
- **Tenant-Strategie:** Dedizierter Pilot-Tenant in der Multi-Tenant-Production-Instanz
- **Module aktiv ab Tag 1:** CRM, Helpdesk, Wiki, Dialer, Berichte, Formulare (laut Modul-Scope-Matrix)
- **Module deaktiviert via Feature-Flag:** Plugin-WASM, finance_line_items-Normalisierung
- **Support-Kanal:** ⚠ „Direkter Slack-/Telefon-Draht zu Luke" ist keine haltbare Zusage bei 12–16
  verfügbaren Tagen im Monat (Lagebild §10). Vor dem ersten Kunden ersetzen durch etwas, das gehalten
  werden kann — z. B. „Antwort binnen eines Werktags".
- **Pilot-Dauer:** ~~Juli–Oktober 2026~~ — Zeitraum verstrichen, keine Nachfolgeplanung
