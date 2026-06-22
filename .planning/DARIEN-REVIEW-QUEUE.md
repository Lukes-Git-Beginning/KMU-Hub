# Darien-Review-Queue — meine offenen Reviews (vor Nico)

> Module, die **review-reif** sind und auf **meinen** (Dariens) eigenen Review warten, BEVOR sie an Nico gehen.
> Ablauf je Modul: phasenweise durchgehen → Findings ins Paket eintragen → ich (Claude) arbeite sie als FIX-Phasen ab → **dann** an Nico.
> Reviewen immer am Dev-Server: `cd desktop && npm run dev` (Port 5173).

| # | Modul | Stand | Review-Paket | Klick-Pfad | Status |
|---|---|---|---|---|---|
| 1 | **berichte** | Report-Authoring R-0…R-6 (11 Datenquellen, PDF, Scheduling) | `.planning/berichte-REVIEW-paket.md` | **Berichte** (`/#/berichte`) → Tab *Berichte* → „Neuer Bericht" | ⬜ wartet auf Darien |
| 2 | **wiki** | Phase B (PB-1…PB-5) — gemeinsame Block-Engine | `.planning/wiki-REVIEW-paket.md` | **Wissen/Wiki** (`/#/wiki`) → Artikel → *Bearbeiten* | ⬜ wartet auf Darien |
| 3 | **mails** | MA-1…MA-10 — Stateful-MSW, Threads, Multi-Account, Vorlagen, Labels/Regeln, CRM, Settings | `.planning/mails-REVIEW-paket.md` | **E-Mail** (`/#/mails`) | ⬜ wartet auf Darien |
| 4 | **kommunikation/chat** | KO-1…KO-10 — Stateful chat-store, Gruppen-DMs, Threads, Lesezeichen, Datei-DL, Suche-Jump, Unified-Inbox, Slash-Commands, Kanal-Edit | `.planning/kommunikation-REVIEW-paket.md` | **Kommunikation** (`/#/kommunikation`) → *Team* + *Posteingang* | ⬜ wartet auf Darien |
| 5 | **formulare** | FO-1…FO-10 (Sub) — DnD, DSGVO, Mail-Benachrichtigung, Feldtypen, bedingte Logik, Einreichungen, Analytics, Settings | ⚠ Review-Paket fehlt (Sub hat keins erstellt) — Stand aus FO-Commits `98325b43`…`e72709e1` | **Formulare** (`/#/formulare`) | ⬜ wartet auf Darien |

## Hinweise
- **berichte + wiki** laufen auf **derselben Block-Engine** (`components/shared/document`) — beim Review nebeneinander, gleiches Look-and-Feel prüfen.
- **mails + kommunikation** teilen das Stateful-MSW-Store-Muster (`email-store`/`chat-store`) und das `ModuleSettingsShell`-Settings-Muster — kohärent prüfen.
- Findings je Modul ins jeweilige `*-REVIEW-paket.md` (Findings-Sammelstelle). formulare: Findings notieren, Review-Paket wird bei Bedarf nachgezogen.
- Nach Abarbeitung der Findings: Haken hier auf ✅, Modul an Nico (Pipeline im `MASTER-TRACKER.md`).
- **Reihenfolge-Tipp:** ältester zuerst (berichte/wiki), dann die frischen (mails → kommunikation → formulare). Oder nach Wichtigkeit für den nächsten Pilot.
