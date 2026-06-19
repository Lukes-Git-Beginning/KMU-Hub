# Review — automatisierung

> **Status:** review-reif (A-1…A-5, Main-Terminal, gemergt, `8274f821`→`29b7d5cd`).
> **Lane:** FE/UX-Review (mock-first). Echte Automatisierungs-Engine / echtes Ausführungs-Backend = Lukes Lane.
> **Screens:** `/#/automatisierung` — Liste · Vorlagen · Protokoll · Editor · Einstellungen.
> **Kontext:** Das Modul war im Demo **komplett tot** (MSW lieferte `{workflows}` mit falschen Feldern → überall EmptyState). Jetzt lebendig.

## Was gebaut wurde (Definition of Done)
- [x] **A-1 — MSW↔Client-Vertrag repariert**: Liste zeigt **8 Automationen**, Stats (5 aktiv / 8 gesamt / 75 %), Vorlagen-Galerie, Trigger-/Action-Katalog. Aktiv-Toggle persistiert. Fehlende Endpoints ergänzt (enable/disable/createFromTemplate/test-condition/dry-run/getExecution); **stateful** Mock.
- [x] **A-2 — Detail = `DetailModal`**: Zeilen-Klick → zentriertes Modal (Auslöser / Bedingungen / Aktionen / Details / letzte Läufe + Aktionsleiste), sticky Close. **Ganze Zeile klickbar**. „Bearbeiten" öffnet Wizard.
- [x] **A-3 — Löschen + Duplizieren**: Löschen mit Bestätigung (`ConfirmDialog`), Duplizieren als „(Kopie)" — beide sofort in der Liste.
- [x] **A-4 — Protokoll + Editor**: Log-Tab global (`/executions`, zeigt alle Läufe). Dry-Run für ungespeicherte Drafts. Visueller `AutomationEditor` (React Flow) verkabelt (war toter Code) — „Zum Editor wechseln" zeigt Canvas. TriggerSelector-Dup-Key-Fix (mehrere `event`-Trigger).
- [x] **A-5 — Settings-Panel** (`ModuleSettingsShell`, Eintrag „Automatisierung"): **persönlich** (Standard-Ansicht → Page-Tab) + **tenant** (Protokoll-Retention + Fehler-Benachrichtigung). i18n-Schlusscheck (Hardcode „Trigger" + 2 tote Keys raus).

## Worauf besonders achten
- Liste/Stats/Vorlagen wirklich gefüllt (kein EmptyState mehr)?
- Aktiv-Toggle, Löschen, Duplizieren → wirken sofort + plausibel?
- Detail-Modal: alle Sektionen + sticky Close; ganze Zeile klickbar?
- „Zum Editor wechseln" → Canvas sichtbar; Dry-Run gibt Rückmeldung?
- Settings personal/tenant korrekt; EN-Umschalten ohne Raw-Keys.

## Out of scope (kein Mangel)
- Echte Automatisierungs-Engine, echtes Ausführungs-Backend (Läufe sind Demo-Daten).

## Befunde
> Format pro Zeile: **Schweregrad** (P0 / P1 / P2) · **was** · **wo** (Datei/Screen) · **Repro**.

_(noch keine — von Nico zu füllen)_
