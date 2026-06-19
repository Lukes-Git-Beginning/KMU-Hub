# Parallel-Batch 3 — kombinierte Review-Liste (automatisierung + profil)

> **Stand 2026-06-19.** Beide Module review-reif, `parallel/profil` gemergt (Merge `6f44d65a`), Build grün, je Build- + Playwright-Screenshot-verifiziert. Für Nicos Review. Beide = FE-mock-first; echtes Backend (Automatisierungs-Engine, Avatar-/Dokument-Storage) ist Lukes Lane.

## Modul A — automatisierung (Main, `main`, A-1…A-5)
Commits `8274f821` → `29b7d5cd`. Kern: das Modul war im Demo **komplett tot** (MSW-Mock lieferte `{workflows}` mit falschen Feldern → Liste/Stats/Vorlagen/Log überall EmptyState). Jetzt lebendig.

| # | Was | Review-Fokus |
|---|---|---|
| A-1 | MSW↔Client-Vertrag repariert (`mocks/handlers/automation.ts` an `automation-types.ts` angeglichen: `{automations}`, `is_active`/`trigger_type`/`scope`, Stats als Fraktion, Trigger-/Action-Katalog mit `module`/`name`/`params`, Templates valide Kategorien+Komplexität; fehlende Endpoints enable/disable/createFromTemplate/test-condition/dry-run/getExecution; **stateful** Mock) | Liste zeigt 8 Automationen, Stats 5/8/75 %, Vorlagen-Galerie, Aktiv-Toggle persistiert |
| A-2 | Zeilen-Klick → `shared/DetailModal` (Auslöser/Bedingungen/Aktionen/Details/letzte Läufe + Aktionsleiste); ganze Zeile klickbar (`role=button`) | Detail zentriert, sticky Close, „Bearbeiten" öffnet Wizard |
| A-3 | Löschen (`ConfirmDialog`) + Duplizieren („(Kopie)") | Beide sofort in der Liste; Löschen mit Bestätigung |
| A-4 | Log-Tab global (`/executions` + `useAllExecutions`) · Dry-Run für ungespeicherte Drafts · `AutomationEditor` (React Flow) verkabelt (war toter Code) · TriggerSelector Dup-Key-Fix (mehrere `event`-Trigger) | Protokoll zeigt alle Läufe · „Zum Editor wechseln" zeigt Canvas |
| A-5 | Settings-Panel (`ModuleSettingsShell`: persönlich Standard-Ansicht→Page-Tab, tenant Protokoll-Retention + Fehler-Benachrichtigung) · i18n-Schlusscheck (Hardcode „Trigger" + 2 tote Keys raus) | Modul-Einstellungen → „Automatisierung"-Eintrag, personal/tenant korrekt |

**Out of scope:** echte Automatisierungs-Engine, echtes Ausführungs-Backend (Luke).

## Modul B — profil (Sub, `parallel/profil`, P-1…P-5)
Commits `00a06eaa` → `3461e805`. Kern: Dokumente-Tab war leer + Stubs, Profil-Defaults hardcodeten „Darien Morales", verwaister Toter-Code-Ordner.

| # | Was | Review-Fokus |
|---|---|---|
| P-1 | current-user als **Stefan Vogel** geseedet (`stores/settings.ts`) + „Mitglied seit" + Abwesenheiten-Daten-Vertrag gefixt | Profil zeigt durchgängig Stefan Vogel (= Sidebar/Topbar) |
| P-2 | Dokumente-Tab an MSW verdrahtet (`hr.ts`: List/Upload/Preview/Download, 7 Demo-Docs + Kategorien) | Dokumente gefüllt, Upload/Preview/Download wirken |
| P-3 | Avatar-Upload über MSW + DND-Demo-Fallback | Avatar wechselbar, DND im Demo umschaltbar |
| P-4 | Verwaisten `tabs/zeiterfassung/`-Ordner (11 Dateien) + 151 tote i18n-Keys entfernt | Zeiterfassung-Tab unverändert funktional, Build grün |
| P-5 | Profil-Karte (Ping→Chat) verifiziert + Demo-Tiefe-Schlusscheck | Keine toten Buttons, 0 Raw-Keys |

**Out of scope:** echtes Avatar-/Dokument-Storage-Backend, echte DND-Backend-Anbindung (Luke).

## Verifikation (post-merge)
- `npm run build` grün (echter Exit 0). 4 i18n-JSON valide, keine Konfliktmarker, automatisierung-Keys in allen 4 Sprachen intakt.
- automatisierung: Screenshots `desktop/.qa-screenshots/automatisierung/` (1-automations/2-templates/3-log/4-detail/6-editor/7-settings).
- profil: Screenshots `desktop/.qa-screenshots/profil/` (1-profil/2-dokumente) + Sub-Protokoll `qa-profil.md`.
- 0 Raw-Keys · 0 Doppelklammern · 0 Console-Errors in beiden Modulen.

## Offen für Nico-Review
Beide Module gegen die Demo-Tiefe-/UX-Standards prüfen (DetailModal, ganze Zeile klickbar, sticky Close, keine leeren Screens). Bei grün → an die Backend-Lane (Luke) für die echten Endpoints.
