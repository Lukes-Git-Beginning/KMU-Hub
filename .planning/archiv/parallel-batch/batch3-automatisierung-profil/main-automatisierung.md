# Main-Terminal — automatisierung Tiefe-Pass (A-1 … A-5)

> **Main-Terminal, Hauptklon `…/KMU Hub`, Port 5173, Branch `main` (direct-to-main).** Scope „Tiefe-Pass, FE-mock-first" (README). Zusatzrolle: beide Batches planen, `parallel/profil` am Ende mergen, kombinierte QA-Liste.

## Ausgangslage (Ist-Abgleich 2026-06-19)
automatisierung-FE ist **sehr vollständig** (`AutomatisierungPage.tsx` 302 Z. + Wizard/Editor/TemplateGallery/ExecutionLogViewer/Trigger-/Condition-/ActionBuilder + React-Flow-Nodes). **Aber im Demo tot:** der MSW-Mock (`mocks/handlers/automation.ts`) ist strukturell **inkompatibel** mit den Client-Typen (`automation-client.ts`) → Liste/Stats/Templates/Log zeigen überall EmptyState/0. Das ist der Kern-Fix. Daten via TanStack Query; UI-State in `stores/automatisierung.ts` (kein persist). i18n unter `automatisierung.*` (de.json 587–742) + `api.automation.*` (333–342).

## Workflow pro Phase
bauen → i18n ×4 → MSW-Daten → Compile-Gate (`npm run build`, echter Exit, nie `| tail`) → Playwright-QA :5173 + Bilder ansehen → commit+push `main` (vorher `git pull`) → `qa-automatisierung.md`.

---

### A-1 — MSW↔Client-Vertrag reparieren  `[FUNDAMENT/KERN — zuerst]`
**Ist:** Mock gibt `{workflows}` (`automation.ts:208`), Felder `status`/`trigger`/`execution_count`; Client erwartet `{automations}`, `is_active`/`trigger_type`/`last_triggered_at`. Stats-Mock `active_workflows`/`paused_workflows` (287–295) vs. Client `active`/`total`/`total_executions`/`success_rate`. Trigger-Katalog ohne `module`/`name`/`fields`/`config_params`. Action-Katalog ohne `params`/`output_fields`. Templates `category: sales/marketing/support` (188–193) ohne `complexity` vs. Client `vertrieb/personal/finanzen/kommunikation`. Executions filtern auf `workflow_id`; Log-Tab übergibt leere id (`AutomatisierungPage.tsx:292`, `ExecutionLogViewer.tsx:214`). Registrierung in `mocks/handlers/index.ts` prüfen.
**Soll:** Mock-Daten + Handler-Responses exakt an die `automation-client.ts`-Typen angleichen (List→`{automations}`, korrekte Felder, Stats, Trigger-/Action-Katalog mit `module`/`name`/`fields`/`params`/`output_fields`, Templates mit gültigen Kategorien + `complexity`). Globales Executions-Listing für den Log-Tab (siehe A-4). Falls nicht registriert → `index.ts` einmalig ergänzen.
**Verify:** `/automatisierung` zeigt 8 Automationen in der Tabelle, StatsBar echte Zahlen, Templates-Tab Karten mit Modul-Icons + Komplexität, Log-Tab Ausführungen.

### A-2 — Zeilen-Klick → `shared/DetailModal`  `[PATTERN]`
**Ist:** Zeilen-Klick → `handleEdit` → öffnet den **Wizard** (`AutomatisierungPage.tsx:235–239`). Keine Read-only-Detailansicht. Projektstandard: Zeilen-Klick = zentriertes Modal mit allen Infos + Funktionen.
**Soll:** Read-only `shared/DetailModal` mit allen Infos: Status, Trigger, Bedingungen, geordnete Aktionen, Scope, letzte Ausführungen (Mini-Liste). Aktionsleiste: **Bearbeiten** (→ Wizard), **Aktiv-Toggle**, **Duplizieren** (A-3), **Löschen** (A-3). Ganze Zeile `role=button`+Tastatur; innerer Toggle `stopPropagation`. Sticky Header/Close.
**Verify:** Zeile klicken → zentriertes Modal, alle Sektionen, Close sticky, Bearbeiten öffnet Wizard, Escape schließt.

### A-3 — Löschen + Duplizieren verkabeln
**Ist:** `useDeleteAutomation` existiert (`useAutomation.ts:150–162`), wird nirgends genutzt. Duplizieren fehlt komplett.
**Soll:** Löschen via `shared/ConfirmDialog` aus Detail-Modal + Zeilen-Aktion. Duplizieren: bestehende Automation als Vorlage laden → Kopie „… (Kopie)" anlegen.
**Verify:** Löschen entfernt (mit Bestätigung), Duplizieren legt Kopie an, beides sofort in der Liste.

### A-4 — Log-Tab echt + Dry-Run für neue + Editor verkabeln
**Ist:** Log-Tab mountet `ExecutionLogViewer` mit leerer `automationId` → ungültiger Pfad → immer leer. Dry-Run im Review-Step `disabled` bis gespeichert (`AutomationWizard.tsx:183`). `AutomationEditor` ist toter Code (nie gerendert); „Zum Editor wechseln" (`AutomationWizard.tsx:409`) setzt nur Store-`editorMode`, Page liest ihn nicht.
**Soll:** (a) Log-Tab zeigt **alle** Ausführungen (globales Listing im Mock + Viewer behandelt fehlende id = alle). (b) Dry-Run auch für ungespeicherte Drafts (Simulation aus Draft). (c) Editor verkabeln: `AutomatisierungPage` liest `editorMode` aus Store + rendert `AutomationEditor` (React-Flow-Canvas) im Dialog, wenn `'editor'`.
**Verify:** Log-Tab gefüllt; Dry-Run wirkt vor dem Speichern; „Zum Editor wechseln" zeigt den Flow-Canvas.

### A-5 — Settings-Panel + i18n/Demo-Tiefe-Schlusscheck
**Ist:** automatisierung fehlt in `module-settings-registry.tsx` (kein Panel). `TemplatePreviewDialog.tsx:190` hat Hardcode `<h4>Trigger</h4>`.
**Soll:** `AutomatisierungSettingsPanel` (ModuleSettingsShell, `personal` = Standard-Tab/Ansicht; `tenant` = Ausführungs-Retention, Fehler-Benachrichtigungs-Default) + Eintrag in `module-settings-registry.tsx`. Hardcode fixen + Schlusscheck (tote Buttons / Toast-only / 0 Raw-Keys / 0 `{{var}}`).
**Verify:** Modul-Einstellungen → automatisierung-Eintrag (personal+tenant), EN sauber, keine Raw-Keys.

---

## Nach A-5: Merge + Abschluss
1. `git pull` (Sub hat ggf. nichts auf main gepusht — egal, sicherheitshalber).
2. `parallel/profil` mergen: `git checkout main && git pull && git merge --no-ff parallel/profil -m "merge: profil review-reif (Sub)"`. i18n-Konflikt: **beide** Key-Blöcke behalten (automatisierung.* + profil.*), dann `npm run build` (echter Exit).
3. Kombinierte QA-Liste `qa-combined.md` (automatisierung + profil) für Darien/Nico.
4. Pipeline-Haken im `MASTER-TRACKER.md` (#7 automatisierung, #8 profil → review-reif).

## Definition of Done (automatisierung review-reif)
A-1…A-5 verifiziert (Screenshots angesehen), Demo lebendig (keine EmptyStates mehr durch Mock-Mismatch), 0 Raw-Keys/Doppelklammern/Console-Errors, je ein Commit auf `main`, `qa-automatisierung.md` gepflegt. **Out of scope:** echte Engine, echtes Backend (Luke).
