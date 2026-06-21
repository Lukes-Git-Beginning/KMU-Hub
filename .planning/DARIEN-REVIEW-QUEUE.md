# Darien-Review-Queue — meine offenen Reviews (vor Nico)

> Module, die **review-reif** sind und auf **meinen** (Dariens) eigenen Review warten, BEVOR sie an Nico gehen.
> Ablauf je Modul: phasenweise durchgehen → Findings ins Paket eintragen → ich (Claude) arbeite sie als FIX-Phasen ab → **dann** an Nico.
> Reviewen immer am Dev-Server: `cd desktop && npm run dev` (Port 5173).

| # | Modul | Stand | Review-Paket | Klick-Pfad | Status |
|---|---|---|---|---|---|
| 1 | **berichte** | Report-Authoring R-0…R-6 komplett (11 Datenquellen, PDF, Scheduling) | `.planning/berichte-REVIEW-paket.md` | Modul **Berichte** (`/#/berichte`) → Tab *Berichte* → öffnen / „Neuer Bericht" | ⬜ wartet auf Darien |
| 2 | **wiki** | **Phase B (PB-1…PB-5)** — auf gemeinsamer Block-Engine (wie berichte) | `.planning/wiki-REVIEW-paket.md` | Modul **Wissen/Wiki** (`/#/wiki`) → Artikel öffnen → *Bearbeiten* | ⬜ wartet auf Darien |

## Hinweise
- Beide laufen auf **derselben Block-Engine** (`components/shared/document`) — beim Review nebeneinander stellen, gleiches Look-and-Feel prüfen.
- Findings trägst du in das jeweilige `*-REVIEW-paket.md` (Spalte „Anpassungen" / „Findings-Sammelstelle") ein.
- Nach Abarbeitung der Findings: Haken hier auf ✅, Modul an Nico (Review-Pipeline im `MASTER-TRACKER.md`).
- Neue review-reife Module hier ergänzen, sobald sie auf deinen Review warten.
