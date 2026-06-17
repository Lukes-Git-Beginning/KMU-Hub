# QA-Protokoll — dashboard (Main-Terminal)

> Pro fertiger Phase ein Eintrag: was gebaut, was Darien anschauen soll, Screenshots. `[PATTERN]` = zuerst ansehen (betrifft mehrere Phasen).

## D-1 — Persistenz scharf + Store-Crash-Fix ✅
**Gebaut:** MSW-Handler für `PUT/DELETE /dashboard/layout` + `GET/PUT /dashboard/defaults/{role}` (stateful, in-memory) ergänzt; Store-Crash in der Admin-Rollen-Seite gefixt (`s.layouts`→`personalLayouts`); `initFromServer()` beim Dashboard-Mount aufgerufen.
**Dateien:** `mocks/handlers/dashboard.ts`, `modules/settings/DashboardSettings.tsx`, `modules/dashboard/DashboardPage.tsx`.
**Was du anschauen sollst:**
1. **Admin-Rollen-Seite** `#/settings/dashboard` → Tab Administrator/Manager/Mitarbeiter, dann **„Aktuelles Layout als Standard"** klicken → muss **grünen Erfolgs-Toast** zeigen, **kein Weiß-Screen** (das war vorher ein harter Crash: `undefined.map`). Screenshot `4-after-copy.png`.
2. **Dashboard anpassen** (Button oben rechts) → Widget hinzufügen/entfernen → nach Reload bleibt der Stand (server-sync läuft jetzt durch statt 404).
**Verifiziert:** pageErrors 0, raw-keys 0, alle 5 Persistenz-Endpoints 200 + stateful echo. Screenshots: `.qa-screenshots/dashboard-d1/`.
