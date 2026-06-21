# Batch-NEXT · Paket A (HAUPT-Terminal) — mails → review-reif

> **10-Phasen-Autonom-Batch** ([[feedback_ten_phase_autonomous_batch]]). Dieses Paket fährt das **Haupt-Terminal** (`KMU Hub`, Port 5173).
> **Disjunkt zu Paket B** (Block-Engine, Sub-Terminal). Berührt NUR `modules/mails`, `mocks/handlers/email.ts`, neu `modules/mails/settings`, i18n-Keys `mails.*`.

## Ziel
Das **mails**-Modul (E-Mail-Client) von „gebaut" auf **review-reif → Nico** bringen: Demo-Tiefe-Standard durchsetzen + Feature-Lücken schließen + Settings-Panel ergänzen.

## Ist-Stand (verifiziert 2026-06-22)
- `modules/mails/` ~3455 Zeilen: `MailsPage`, `ComposeInline/Modal/WindowPage`, `compose-shared`, `EmailTemplateDialog`, `ImportWizard`, `ExportDialog`, `tabs/MailServerSettingsTab`.
- MSW-Handler vorhanden: `mocks/handlers/email.ts`.
- **Fehlt:** `modules/mails/settings/` (kein Modul-Settings-Panel — [[feedback_module_settings_per_module]]).
- Tracker-Eintrag „⬜ Neubau" ist **veraltet** — es ist ein Tiefe-Pass.

## Recherche-Auftrag (VOR dem Bauen — Gate)
Das Terminal prüft morgen früh und meldet gebündelt zurück:
1. Wie tief ist mails wirklich? (3-Panel? Thread-Ansicht? Compose voll? Welche tabs/Views existieren, was ist Stub/toter Button?)
2. Welche der 10 Phasen unten sind schon (teil-)erledigt → streichen/zusammenfassen.
3. Demo-Tiefe-Audit: Slide-over vs. `shared/DetailModal`? Ganze Zeile klickbar? Downloads/Exporte echt oder Toast-Stub?
4. `email.ts`-Handler-Umfang: reicht der MSW-Vertrag für Unified Inbox / Labels / Regeln, oder ausbauen?

## Gate-Fragen an Darien (gebündelt stellen, dann bauen)
- **Demo-Daten-Tiefe:** Wie realistisch sollen die Demo-Mails sein (Threads, mehrere Konten, Anhänge)?
- **Unified Inbox:** ein zusammengeführter Eingang über alle Konten — ja, und als Default-Ansicht?
- **CRM-Integration-Umfang:** nur Thread↔Kontakt-Verknüpfung, oder auch „Deal aus Mail" + Aktivität protokollieren?
- **Settings-Scope:** was gehört personal (Signatur, Standard-Konto) vs. tenant (Server, Compliance)? ([[feedback_settings_scope_hr_data]])

## Phasen (vorläufig — beim Gate verfeinern)
- [ ] **MA-1** Demo-Tiefe-Standard: Slide-over→`shared/DetailModal`, ganze Zeile klickbar (`role=button`+stopPropagation), tote Buttons/Stubs fixen, Downloads/Exporte wirken ([[feedback_module_depth_standard]] [[feedback_detail_modal_standard]])
- [ ] **MA-2** Lese-Tiefe: Thread-/Konversations-Ansicht, Zitate ein-/ausklappen, Inline-Bilder, Anhänge-Vorschau + Download
- [ ] **MA-3** Konten-Verwaltung + Unified Inbox (mehrere Konten via MSW, Konto-Switch, „Alle Eingänge")
- [ ] **MA-4** Volltext-Suche + Filter (gelesen/ungelesen/markiert/Label) + Sortierung (`shared/SortMenu`, Feld+Richtung — [[feedback_recurring_ux_patterns]])
- [ ] **MA-5** Vorlagen vervollständigen: `EmailTemplateDialog`→CRUD, Variablen-Platzhalter, Einfügen in Compose
- [ ] **MA-6** Regeln/Filter + Labels (stateful MSW: wenn-dann-Regeln, Label-CRUD + Zuweisung)
- [ ] **MA-7** CRM-Integration: Thread↔Kontakt-Verknüpfung, „Deal aus Mail", Aktivität protokollieren
- [ ] **MA-8** Bulk-Aktionen (Mehrfachauswahl: verschieben/löschen/markieren/Spam) + Tastatur-Shortcuts + leere Zustände
- [ ] **MA-9** Settings-Panel `modules/mails/settings` (`ModuleSettingsShell`, personal+tenant: Signatur, Standard-Konto, Benachrichtigungen, Server)
- [ ] **MA-10** i18n ×4 Vollständigkeit (`mails.*`, `{var}` nie `{{var}}`, ICU-Plural) + Demo-Tiefe-Schlusscheck + Playwright-QA + Screenshots WIRKLICH ansehen

## Disjunktheit / Konflikt-Vermeidung
- **NUR** `modules/mails/**`, `mocks/handlers/email.ts`, neu `modules/mails/settings/**` anfassen.
- **Finger weg von** `components/shared/document/**`, `modules/wiki/**`, `modules/berichte/**` (= Paket B).
- i18n-Dateien (`de/en/fr.json`): nur `mails.*`-Keys ergänzen → rebased additiv mit Paket B.

## Build-+-Verify-Standard (pro Phase, [[feedback_two_terminal_nico]])
bauen → i18n ×4 → gescopter tsc (foreground, echter Exit, NIE `| tail`) → `eslint src/ --quiet` ([[feedback_lint_before_push]]) → Playwright-Screenshot-QA + PNGs ansehen → ein Commit (explizite Pfade) → push (`pull --rebase` über parallele Pushes). Latenz: vorbestehenden `useTasks.ts`-tsc-Fehler ignorieren.
