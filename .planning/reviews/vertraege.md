# Review-Fäden — vertraege

> Eine Datei pro Modul. Der Bau-Strom trägt nach JEDER fertigen Phase einen Eintrag ein (Struktur siehe `_TEMPLATE.md`). Vorlage für Dariens Feinschliff-Review (Pfad nachklicken, worauf achten, Screenshots, offene Punkte).

**Modul:** `vertraege` · **Strom:** L · **Reviewer (zugeteilt):** offen

---

## Phase 1 — Modul-Einstellungen (VertraegeSettingsPanel)  ·  marathon/luke-fe  ·  Status: ⬜ ungereviewt

**Hinklicken (Pfad in der App):**
- Route(n): `/vertraege` → Sidebar unten „Modul-Einstellungen" → Eintrag „Verträge" (kontext-vorausgewählt)
- Schritte: Persönlich: Standard-Tab umstellen (z.B. Vorlagen) → Overlay schließen → `/vertraege` neu laden → Modul öffnet auf dem gewählten Tab. Tabellendichte „Kompakt" → Vertragsliste wird enger. Für alle: Vertragsart deaktivieren → „Vertrag anlegen" bietet sie nicht mehr an; Kündigungsfrist/Verlängerung ändern → Dialog vorbelegt.

**Worauf achten (Feinschliff):**
- [ ] Layout/Hierarchie bei voller Breite + schmal (760 geprüft, Screenshot)
- [ ] Keine Raw-i18n-Keys, keine Emojis, keine ASCII-Umlaute (QA: 0 Raw-Keys, alle 4 Sprachen befüllt)
- [ ] Interaktionen echt: Standard-Tab + Dichte + Erinnerungs-Vorauswahl greifen real; Vertragsarten-Toggle + Standardwerte steuern den Neu-Dialog real
- [ ] Erinnerungs-Resolve: persönliche Vorauswahl überschreibt Unternehmens-Standard (Checkbox „Standard des Unternehmens verwenden")
- [ ] Lock-Verhalten „Für alle" für Nicht-Modulleiter

**Screenshots:** `desktop/.qa-screenshots/vertraege-settings/` (panel-top, panel-tenant, panel-pref-set, module-default-tab, panel-760) — QA `desktop/scripts/qa-vertraege-settings.mjs`

**Bekannte offene Punkte / Backend-Bedarf:**
- Tenant-Settings laufen mock-first (`stores/vertraegeSettings.ts`, localStorage) — Persistenz via `tenant_settings` (module_id='vertraege') = Luke-Backend; Settings-Fundament-Endpoints existieren seit Migration 000138.
- Nummernkreis: nur Format + Vorschau; automatische Vergabe = Backend.
- Standard-Laufzeit (Monate): Setting vorhanden, greift noch nicht im Dialog (Enddatum-Autofill bewusst nicht in dieser Phase).
- **Offene Frage für Darien:** vertraege-Domain fehlt komplett in der OpenAPI-Spec (vorbestehend, betrifft nicht diese FE-Phase).
- Beobachtung (vorbestehend, global): Topbar rendert ein „⚙️"-Emoji (innerText-Dump bei 760) — kollidiert mit der No-Emoji-Regel, nicht Teil dieser Phase.

**Reviewer-Notizen (beim Feinschliff auszufüllen):**
- _<hier trägt der Reviewer Feedback ein → wird zu TaskCreate-Items>_

---

## Phase 4 — Audit-Log + Erinnerungs-Detail  ·  marathon/luke-fe  ·  Status: ⬜ ungereviewt

**Hinklicken (Pfad in der App):**
- Route: `/vertraege` → beliebigen Vertrag in der Tabelle anklicken → DetailPanel öffnet sich
- Sektion "ERINNERUNGEN": zeigt Erinnerungstermine (endDate - X Tage) mit Datum rechts; vergangene Termine sind durchgestrichen + gedimmt
- Sektion "ÄNDERUNGSHISTORIE (N)": chronologischer Feed, neueste zuerst, mit farbigen Icon-Badges je Aktionstyp

**Was gebaut:**
- `AuditLogFeed`-Komponente (inline, nicht importiert): ersetzt den einfachen Timeline-Feed durch einen Activity-Feed mit Lucide-Icons (FilePlus/FilePen/FileX/Pen/BellRing/History) und farbigen Icon-Kreisen; neueste Einträge fett; Leer-Zustand mit Rahmen + zentriertem History-Icon
- `ReminderSchedule`-Komponente (inline): berechnet endDate - X Tage für jeden reminderDays-Eintrag, zeigt Datum rechts, vergangene Termine gedimmt + line-through; Leer-Zustand (keine reminderDays) mit Bell-Icon + Text; Unbefristet-Fallback wenn kein endDate
- **Store-Codierung:** Neue Store-Mutationen schreiben stabile englische Action-Codes (`contract_created`, `contract_updated`, `contract_terminated`). Legacy Mock-Einträge enthalten vorübersetzte deutsche Freitexte — der Renderer erkennt bekannte Codes via `isActionCode()` und übersetzt via i18n; unbekannte Strings werden als Fallback direkt angezeigt
- `ContractHistoryActionCode`-Union-Type im Store + `meta?: string`-Feld für Zusatzpayload (z.B. Kündigungsgrund)
- `addContractFromTemplate` im Store: jetzt mit Code `contract_created` + `meta: template.name`

**Was bewusst nicht gebaut:**
- Kein Reload des `selectedContract` nach Bearbeiten (vorbestehendes Design: DetailPanel zeigt Snapshot — wird erst nach erneutem Anklicken des Vertrags aktualisiert). Das ist eine bekannte UX-Lücke, nicht Phase-4-Scope.
- `contract_signed`/`reminder_triggered` als Codes vorhanden, aber noch kein UI-Trigger (eSignatur-Dialog schreibt noch kein `contract_signed` — kann in einer Folge-Phase ergänzt werden)
- Terminierungsdialog schreibt `contract_terminated` mit `meta: reason` — der Feed zeigt aktuell nur den übersetzten Code, nicht den reason. Erweiterung möglich via `entry.meta`-Appendix im Label.

**Offene Fragen für Darien:**
- vertraege-Domain fehlt weiterhin komplett in der OpenAPI-Spec (vorbestehend, betrifft nicht diese FE-Phase)
- Wann API-Swap der Page auf `useVertraege`-Hooks (statt `useVertraegeStore`)? Dann müssen AuditLogFeed + ReminderSchedule die Props aus den API-Typen beziehen
- `selectedContract` Snapshot-Problem: soll nach Bearbeiten ein `setSelectedContract(updatedContract)` aus dem Store eingefügt werden? (kleiner Fix, braucht Entscheidung)

**Screenshots:** `desktop/.qa-screenshots/vertraege-audit/` (1-detail-with-history, 2-empty-states, 3-after-mutation) — QA `desktop/scripts/qa-vertraege-audit.mjs`

**QA-Ergebnis:** rawKeys: [], pageErrors: [], alle Assertions grün (3/3 Schritte)

**tsc-Gate:** 0 neue Fehler in eigenen Dateien (vorbestehende ~98 typed-i18n-Fehler in fremden Panels unverändert)

**Reviewer-Notizen (beim Feinschliff auszufüllen):**
- _<hier trägt der Reviewer Feedback ein → wird zu TaskCreate-Items>_

---

<!-- Phasen-Einträge hier anhängen — Struktur aus _TEMPLATE.md kopieren.
     Status-Legende: ⬜ ungereviewt · 🟡 Feedback offen · ✅ grün -->
