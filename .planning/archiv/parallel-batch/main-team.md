# Main-Terminal — team Tiefe-Pass + P1 Schulungen (TM-1 … TM-5)

> **Main-Terminal, Hauptklon `…/KMU Hub`, Dev-Port 5173, Branch `main`.** Ich baue **nur team**. helpdesk gehört dem Sub (`parallel/helpdesk`). Scope (Darien): Tiefe-Pass review-reif + P1 Schulungen-Backend-Swap. P2 (Personalakte↔Dok, Organigramm editierbar) ist NICHT in diesem Batch.

## Ausgangslage (Ist-Abgleich 2026-06-17)
team ist **solide** (10 Tabs, role-gefiltert, Settings-Panel ✅ mit personal + 2× tenant). Detail nutzt eigenen `MemberProfileDialog` (zentriert, ganze Karte/Zeile klickbar, Close sticht — funktional konform, KEIN Umbau auf `shared/DetailModal` in diesem Batch). Datenschicht: `lib/hr-hooks.ts` (TanStack) + MSW `mocks/handlers/team.ts` (+ Duplikate in `hr.ts`). Schulungen/Onboarding noch Zustand-Store.

## Workflow pro Punkt
bauen → i18n ×4 (`{var}`) → MSW/Store-Daten falls nötig → Compile-Gate (`npm run build > /tmp/build.log 2>&1; echo "EXIT $?"` + `grep -i error`, **nie `| tail`**) → Playwright-QA gegen **:5173** + Bilder ansehen → commit + push auf `main` → Eintrag in `qa-team.md`.

---

### TM-1 — Abwesenheiten-Bug (Widget bleibt leer)  `[BUG]`
**Ist (dreifacher Mismatch):** `mocks/handlers/team.ts` L164–166 liefert `{ absences: [...snake_case...] }`; `lib/hr-client.ts` L404 erwartet `{ entries: AbsenceEntry[] }`; `useAbsenceCalendar` (`hr-hooks.ts` L499) macht `select: data => data.entries` → `undefined`. Zusätzlich Feld-Mismatch (`user_id` vs `employeeId` etc.) und Duplikat-Handler in `hr.ts` L495–497 (`{ absences: [] }`).
**Soll:** team.ts-Handler liefert `{ entries: [...] }` mit **camelCase `AbsenceEntry`-Shape** (`hr-types.ts` L205–218: `employeeId`, `employeeName`, `leaveTypeName`, `leaveTypeKey`, `department`, `startDate`, `endDate`). `department` aus der Employee-Map ziehen. Danach den Duplikat-Handler `/hr/absences/calendar` aus `hr.ts` entfernen.
**Verify:** Abwesenheiten-Tab → Kalender zeigt Einträge mit Namen/Typ/Abteilung; mehrere Mitarbeiter sichtbar.

### TM-2 — SelfServiceView verkabeln (komplett Fake)
**Ist:** `SelfServiceView.tsx` liest hardcoded `CURRENT_USER`/`LEAVE_BALANCES`/`SALARY_STATEMENTS`; alle 4 Schnellaktionen (L228–231), „Änderung beantragen" (L191), „Neuer Antrag" (L257) = `toast.info('Mock')`; Gehaltszettel-Download (L318) = `toast.success` ohne Blob.
**Soll:** Auf echte Hooks: `useSelfProfile`, `useEmployeeLeaveBalance`. Antrags-Aktionen verkabeln (Urlaub/Krank/HomeOffice/Überstunden → echter Leave-Request via vorhandener Mutation, erscheint dann auch im Anträge-Tab). Gehaltszettel-Download als **echter Blob** (Demo-PDF via `URL.createObjectURL`) wie in anderen Modulen.
**Verify:** SelfService zeigt echte Profildaten; Antrag stellen → taucht im Anträge-/Requests-Tab auf; Download lädt eine Datei.

### TM-3 — Personalakte/Dokumente echt + Download/Preview
**Ist:** `PersonnelDocuments.tsx` liest hardcoded `MOCK_DOCUMENTS` (L89–101), ignoriert den vorhandenen `useEmployeeDocuments`-Hook (den `MemberProfileContent` schon nutzt). `handleDownload` (L141) + „Vorschau" (L261) = nur Toast; Upload (L145) nimmt keine Datei entgegen.
**Soll:** `PersonnelDocuments.tsx` auf `useEmployeeDocuments` umstellen (MSW-Handler `GET /hr/employees/:id/documents` in `team.ts` ergänzen falls fehlt). Download + Preview als echter Blob/iframe (Demo-PDF). Upload-Dialog mindestens funktional (Datei wählen → erscheint in Liste, Demo).
**Verify:** Personalakte-Tab zeigt API-Dokumente; Download lädt; Vorschau rendert PDF (headed testen).

### TM-4 — OrgChart-Actions + i18n-Doppelklammern + Missing-Key
- **OrgChart:** `OrgChart.tsx` E-Mail (L387) + Anruf (L394) = `toast.info` → auf `setIntent`/`startCall` umbauen (Muster wie `MemberProfileContent.tsx`, ~5 Zeilen).
- **i18n `{{var}}` → `{var}`** (8 Keys, ×4 Sprachen) im `team.moduleAssignment.*`-Namespace: `team.member.modules.lastUsed`, `…bulk.confirmBody`, `…bulk.label`, `…bulk.toast`, `…cellAria`, `…confirmRevoke.body`, `…toast.granted`, `…toast.revoked`. (Werden in `ModuleAssignmentTab.tsx` bei jeder Bulk-Aktion roh angezeigt.)
- **Missing-Key:** `team.page.title` fehlt (de+en, nur `defaultValue: 'Team'`) → in alle 4 Sprachen anlegen.
**Verify:** Modul-Zuweisung Bulk-Aktion → korrekt interpolierter Text (keine `{{ }}`); OrgChart-Detail E-Mail/Anruf öffnen Intent/Call; kein Raw-Key im Titel.

### TM-5 — Schulungen-Tab Zustand→MSW-Hook (P1) + handleDeactivate-Fix
- **Schulungen (Tracker-P1):** Tab liest heute `useTeamStore()` (`TeamPage.tsx` L143–146); kein MSW-Handler für `/hr/trainings`. Handler in `team.ts` ergänzen (Trainings + Teilnahmen seeden), `useTrainings`/`useTrainingParticipations`-Hooks in `hr-hooks.ts`, Tab-Render auf TanStack Query umstellen. „Schulung hinzufügen"/„Teilnahme erfassen"-Dialoge auf Mutations.
- **handleDeactivate:** `TeamPage.tsx` L231–234 sendet `updateEmployeeMutation.mutate({ id, data: {} })` (leer). Auf `data: { status: 'inactive' }` (oder dedizierten Deactivate-Handler) umstellen, damit der Mitarbeiter sichtbar deaktiviert wird.
**Verify:** Schulungen-Tab lädt aus MSW; Schulung anlegen + Teilnahme erfassen wirken; Mitarbeiter deaktivieren ändert sichtbar den Status.

---

## Out of scope (NICHT in diesem Batch)
- `MemberProfileDialog` → `shared/DetailModal` (funktional konform, optional späterer Pattern-Angleich).
- P2: Personalakte↔Dokumente-Verknüpfung tiefer, Organigramm editierbar (Position/Manager).
- `OnboardingChecklist` (komplett Mock, explizites TODO, kein Backend) — für Demo akzeptabel, nicht anfassen außer es stört im Review-Blick.

## Definition of Done (team review-reif)
Alle 5 Punkte verifiziert (Screenshots angesehen), 0 Raw-Keys / 0 `{{var}}` / 0 Console-Errors, jede Phase ein Commit+Push auf `main`, `qa-team.md` gepflegt.
