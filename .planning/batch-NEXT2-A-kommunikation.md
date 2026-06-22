# Batch-NEXT2 · Paket A (HAUPT-Terminal) — kommunikation/chat → review-reif

> **10-Phasen-Autonom-Batch** ([[feedback_ten_phase_autonomous_batch]]). Fährt das **Haupt-Terminal** (`KMU Hub`, Port 5173).
> **Disjunkt zu Paket B** (formulare, Sub-Terminal). Berührt NUR `modules/kommunikation/**`, `mocks/handlers/chat.ts`, `mocks/data/chat*.ts`, neu `modules/kommunikation/settings/**`, i18n-Keys `kommunikation.*` / `chat.*`.

## Ziel
Das **kommunikation/chat**-Modul (Team-Messaging, 3-Panel da) von „gebaut" auf **review-reif → Nico** bringen: Demo-Tiefe-Standard + DMs + Suche + Edit/Löschen/Lesezeichen + Unified Inbox + Settings-Panel.

## Ist-Stand (verifiziert 2026-06-22, grob)
- `modules/kommunikation/` ~28 Dateien / ~4853 Zeilen, 3-Panel-Chat (Kanal-Liste │ Nachrichten │ Detail/Thread).
- MSW-Handler: `mocks/handlers/chat.ts`.
- **Fehlt:** `modules/kommunikation/settings/` (kein Modul-Settings-Panel — [[feedback_module_settings_per_module]]).
- Tracker P2 (DMs+Suche) / P3 (Unified+Edit/Löschen/Lesezeichen) sind die offenen FE-Stränge; P4 LiveKit + P5 Bots/Webhooks sind 🔒 (Luke) → NICHT in diesem Paket.

## Recherche-Auftrag (VOR dem Bauen — Gate)
1. Wie tief ist chat wirklich? (Kanäle? Threads/Antworten? Reaktionen? send/edit/delete — wirken sie, oder Toast-Stub? MSW stateful oder statisch?)
2. Welche der 10 Phasen unten sind schon (teil-)erledigt → streichen/zusammenfassen.
3. Demo-Tiefe-Audit: ganze Zeile/Nachricht klickbar? tote Buttons? Downloads echt? `chat.ts`-Vertragsumfang?
4. Gibt es schon DMs/Unread-Tracking/Suche im Ansatz, oder Neubau?

## Gate-Fragen an Darien (gebündelt stellen, dann bauen)
- **DM-Umfang:** nur 1:1, oder auch Gruppen-DMs?
- **Unified Inbox:** ein zusammengeführter „Alle ungelesenen"-Eingang über Kanäle + DMs — als Default-Fokus?
- **Threads:** Antworten als Inline-Thread (wie Slack) oder flach mit Zitat?
- **Settings-Scope:** personal (Status, Benachrichtigungen, Lese-Bestätigungen) vs. tenant (Kanal-Erstellungs-Policy, Nachrichten-Aufbewahrung)?

## Phasen (vorläufig — beim Gate verfeinern)
- [ ] **KO-1** Demo-Tiefe-Standard: stateful MSW (send/edit/delete/read persistieren), ganze Nachricht/Zeile klickbar, tote Buttons/Stubs fixen, Downloads wirken ([[feedback_module_depth_standard]] [[feedback_detail_modal_standard]])
- [ ] **KO-2** Direktnachrichten (1:1, ggf. Gruppen): DM-Liste, „Neue Nachricht"-Personen-Picker, DM↔Kanal-Umschalten
- [ ] **KO-3** Volltextsuche über Nachrichten + Unread-Tracking (pro Kanal/DM) + Jump-to-Result
- [ ] **KO-4** Nachricht bearbeiten / löschen / Lesezeichen (Bookmarks) — stateful, mit „bearbeitet"-Markierung
- [ ] **KO-5** Unified Inbox: „Alle ungelesenen" über Kanäle + DMs, fokussiertes Abarbeiten
- [ ] **KO-6** Reaktionen (Emoji) + Threads/Antworten (Inline-Thread oder Zitat je Gate)
- [ ] **KO-7** @Mentions + Mention-Benachrichtigung + Kanal-Verwaltung (erstellen/umbenennen/verlassen, privat/öffentlich)
- [ ] **KO-8** Dateien/Anhänge im Chat (Upload-Mock + Vorschau + echter Download) + Link-Vorschau (`shared/LinkPreviewPopover` wiederverwenden)
- [ ] **KO-9** Settings-Panel `modules/kommunikation/settings` (`ModuleSettingsShell`, personal: Status/Benachrichtigungen/Lese-Bestätigung; tenant: Kanal-Policy/Aufbewahrung)
- [ ] **KO-10** i18n ×4 Vollständigkeit (`kommunikation.*`/`chat.*`, `{var}`, ICU-Plural) + Demo-Tiefe-Schlusscheck + Playwright-QA + Screenshots WIRKLICH ansehen

## Disjunktheit / Konflikt-Vermeidung
- **NUR** `modules/kommunikation/**`, `mocks/handlers/chat.ts`, `mocks/data/chat*.ts`, neu `modules/kommunikation/settings/**`.
- **Finger weg von** `modules/formulare/**`, `mocks/handlers/formulare.ts` (= Paket B).
- i18n (`de/en/fr/it.json`): nur `kommunikation.*`/`chat.*`-Keys → rebased additiv mit Paket B.

## Build-+-Verify-Standard (pro Phase)
bauen → i18n ×4 → gescopter tsc (foreground, echter Exit, NIE `| tail`) → `eslint src/ --quiet` ([[feedback_lint_before_push]]) → Playwright-Screenshot-QA + PNGs ansehen → ein Commit (explizite Pfade, `mocks/data/` braucht ggf. `git add -f`) → push (`pull --rebase`). Scoped tsconfig anlegen (`tsconfig.kommunikationcheck.json`). Latenz: vorbestehende latente tsc-Fehler ignorieren.
