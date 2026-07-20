# R-4 — HR-Datenkategorien-Tiefe (Terminal-Briefing)

> **Stand 2026-07-20.** Recherche-Gate durchlaufen (Personio-Detail + BambooHR/HiBob/Factorial via 2 Web-Agents, Cosmi-Ist via Explore-Agent), alle Produktfragen mit Darien entschieden. **Baufreigabe erteilt — dieses Dokument ist das vollständige Arbeitspaket.**
> Kontext: KONZEPT.md §4 R-4 · R-3 komplett (Batch 1–5) · Konventionen aus R3-RECHERCHE §3 gelten weiter (Default VERSTECKEN · „Wert sichtbar, Edit gesperrt" · Header-Chip · scope-own still · NoAccess-Seite).

---

## §0 Darien-Entscheide (2026-07-20, verbindlich)

1. **Self-Service mit Genehmigung („Vorschlagen"):** Mitarbeiter ändern eigene Stammdaten nicht direkt — Änderung wird VORGESCHLAGEN, HR genehmigt (Diff-Karte Alt links / Neu rechts, BambooHR-Pattern; Feld sperrt bis Entscheidung: „Änderung ausstehend"). **Pro Rolle abgestuft:** member/manager dürfen vorschlagen, **Aushilfe/extern NICHT** (deren Änderungen laufen über Vorgesetzte), hr_admin/admin bearbeiten direkt ohne Schleife.
2. **Personen-Sichtbarkeit (Mitarbeiterliste/Organigramm, nur Grunddaten Name/Position/Abteilung/Kontakt):** Jeder sieht seine Umgebung = **2 Ebenen über + 2 Ebenen unter sich**. Ganze Firma nur mit Extra-Recht (neuer Fein-Schalter; admin/hr_admin/it_admin haben ihn default — „HR und IT haben Organigramm + Übersicht über alle Accounts").
3. **Akten-Zugriff Scope „team":** die **GANZE Vorgesetzten-Linie nach unten** (alle Ebenen, nicht nur 2) — **nie nach oben**. Vom eigenen Chef sieht man nur die Person, nie Vertrag/Gehalt. Eigene Daten sind im team-Scope **eingeschlossen** (keine Personio-Falle „eigene separat aktivieren").
4. **Granularität: 5 Schubladen** — Persönlich / Job / Gehalt & Lohn / Dokumente / Abwesenheiten, je Stufe (keine/ansehen/bearbeiten [+vorschlagen für eigene]) × Bereich (eigene/Team/alle). KEINE Feld-Ebene in 1.0 (BambooHR-Checkbox-Dschungel vermeiden); „besonders geschützte Einzelfelder" später nachrüstbar.
5. **Austritt (Offboarding): voller Dialog + Kaskade** — aber **KEIN One-Click**: Einstieg NICHT als Button vorne in der Liste, sondern **nur im Bearbeiten-Bereich des Mitarbeiterprofils**, mit eigenem Bestätigungs-Dialog der die Konsequenzen auflistet. Kaskade: HR-Status inactive + Admin-Account deaktiviert + Seat frei + Rollen entzogen + **Warnung mit Neuzuweisung wenn die Person Vorgesetzte/Genehmigerin ist** (schließt Personios bekannte Approver-Verwaisungs-Lücke). Checklisten-/Template-System bewusst NICHT hier (gehört zum Onboarding-System-Deliverable).
6. **Neuer Baustein (Session-Fund):** Es gibt aktuell KEINEN erreichbaren Weg, Mitarbeiterdaten zu bearbeiten — `EditMemberDialog` ist tot verdrahtet (TeamPage.tsx:197 `_setEditMember` nie aufgerufen), `MemberDetailPanel` wird als Ganzes nirgends gerendert. **R-4 baut Bearbeiten direkt im Profil (MemberProfileContent), pro Schublade gated.**

---

## §1 Markt-Kern (Kurzreferenz, Quellen im Chat-Verlauf Session R-4-Gate)

- **Personio:** Sektionen × (View/**Propose**/Edit) × Scope (Own/Reporting-Line/Custom/All). Propose triggert Approval-Task in HR-Inbox (Hover-Approve, Bulk). Reporting-Line = direkte + indirekte; Own NICHT enthalten (häufige Fehlerquelle — wir machen es besser, §0.3). Dokument-Kategorien haben EIGENE view/edit-Rechte unabhängig vom Profil-Zugriff.
- **Personios größte Schwäche = Cosmi-USP bestätigt:** Admin ist binär, „IT-Admin ohne Gehaltsblick" unmöglich, Community fordert seit Jahren „Admin Light". Cosmi hat it_admin ohne salary ✓ — in R-4 NICHT aufweichen.
- **BambooHR Self-Service-Referenz:** Feld-Lock nach Einreichung + Hinweistext, HR-Inbox-Karte mit **Alt-Wert links | Neu-Wert rechts**, Status Pending/Approved/Denied, Antragsteller kann canceln.
- **Personio Offboarding:** Termination-Dialog (Termination Date, Last Day of Work, Type, Reason, Regretted, Backfill) → Status Inactive, Zugang am Austrittstag weg, inactive zählt nicht gegen Seats. Lücke: Genehmiger-Verwaisung (manuell aufzuräumen) — unsere Kaskade löst das (§0.5).
- **Vorgesetzten-Sicht Markt:** nicht berechtigte Sektionen werden KOMPLETT ausgeblendet (kein ausgegrautes „kein Zugriff") — deckt sich mit unserer Default-VERSTECKEN-Konvention.

## §2 Ist-Stand + Lücken (Explore-verifiziert 2026-07-20)

**Vorhanden:** 20 team-Keys im Katalog (`capability-catalog.ts:360–381`) inkl. `data_personal/data_job/salary/documents` je view+edit — aber alle Datenkategorie-Keys **nicht scopeable**. ROLE_DEFS (`mocks/data/rbac.ts`): admin+hr_admin alle 20, it_admin NUR employee:read, manager operativ (absence:approve, corrections, training, onboarding; employee:read scope 'team'), member/readonly employee:read+absence:read all. `EmployeeProfile.managerUserId` existiert (hr-types.ts:231ff; mock-db `managerId`) → Grundlage für Reporting-Line ✓. `HRDocumentCategory.visibility: 'hr_only'|'manager'|'employee'` existiert im Typ. Gates aus R-3 Batch 2 in MemberProfileContent (data_personal/data_job/documents/payroll + isSelf-Bypass), EmployeePayrollData (salary view/edit), PersonnelDocuments, PayrollPrepPanel, HRApprovalDialog.

**Lücken (= R-4-Arbeitsliste):**
- **L1** Scope `team` ≈ `all` (useCapability.ts:64–73, dokumentierte Lücke) — kein Reporting-Line-Resolver; Schubladen-Keys nicht scopeable.
- **L2** `MemberDetailPanel` zeigt Kontakt/Beschäftigung UNGEGATED — aber Panel ist eh tot (nur `DocumentsSection`/`EmployeeModuleLeadSection` daraus wiederverwendet) → toten Code aufräumen statt gaten.
- **L3** Dokument-Kategorie-`visibility` wird nirgends ausgewertet.
- **L4** `SelfServiceView` ungegated: Gehaltsabrechnungen HARDCODED (alle sehen dieselben Demo-Zahlen, `SALARY_STATEMENTS` Zeile 65), „Änderung beantragen" = toast.info-Stub.
- **L5** Kein Genehmigungs-Workflow für eigene Datenänderungen (`UpdateSelfProfileInput` = nur Adresse+Notfallkontakt, direkt).
- **L6** Offboarding = nur `status:'inactive'` (TeamPage.tsx:269) — keine Kaskade (Admin-Account bleibt aktiv, Seat belegt, Rollen bleiben, Genehmiger verwaisen), kein Dialog.
- **L7** `CreateEmployeeWizard` Schritt 1: wer `employee:create` hat, kann BELIEBIGE Rollen vergeben (inkl. admin) — Eskalations-Loch; Rollen-Schritt braucht `team:role:assign`/`admin:role:assign` + Eskalations-Guard (R-2-Muster: nur Rollen vergeben, deren Keys man selbst hat).
- **L8** manager hat keine data_personal/data_job-Grants — sieht Profildaten nur durchs ungegatete tote Panel.
- **L9** Seed-Drift: `handlers/team.ts` (18 MA aus mock-db) vs. `handlers/hr.ts` (6 eigene Seeds) registrieren teils dieselben Routen (`GET /hr/employees`) — Gewinner hängt von Registrier-Reihenfolge ab. Mindestens dokumentieren, besser harmonisieren (team.ts als SSOT).
- **L10** mock-db `MockEmployee.salary` wird nicht ins Wire gemappt (gut), aber `payrollMasterData`-Store startet leer — für Demo-Tiefe seeden.
- **L11** Keine durchgängigen `managerId`-Ketten in den Seeds (für 2-Ebenen-QA braucht es ≥3, besser 4 Hierarchie-Ebenen).

## §3 Zielmodell (Umsetzungs-Spezifikation)

### 3.1 Katalog-Änderungen (`capability-catalog.ts` — Drei-Stellen-Regel: Katalog + ROLE_DEFS + i18n!)

- **Scopeable machen:** `team:data_personal:view/edit`, `team:data_job:view/edit`, `team:salary:view/edit`, `team:documents:view/edit` (+ `team:absence:read` bleibt scopeable). Scope-Semantik: own = nur eigene Akte · team = eigene + GANZE Linie runter · all = alle.
- **NEU `team:self:propose`** (fine, nicht scopeable): „Änderungen an eigenen Daten vorschlagen" — schaltet den Self-Service-Antrags-Flow frei (§0.1). EIN Key für alle Schubladen (kuratierte KMU-Version statt 5×propose).
- **NEU `team:directory:full`** (fine, nicht scopeable): „Alle Mitarbeitenden sehen (Liste/Organigramm)" — ohne diesen Key gilt die 2-rauf/2-runter-Umgebungssicht (§0.2).
- **NEU `team:employee:offboard`** (fine): Austritts-Flow ausführen (getrennt von `employee:deactivate`, das als schnelles Sperren bestehen bleibt).
- `absence`-Doppelrolle beachten: der ABWESENHEITS-KALENDER (wer ist wann weg — Planungs-Board) bleibt an `team:absence:read` (member: all, das ist gewollt/üblich). Die AKTEN-Schublade „Abwesenheiten" (Salden, Historie, Anträge einer Person im Profil) folgt dem Scope von `absence:read`.

### 3.2 Rollen-Default-Matrix (ROLE_DEFS-Nachzug)

| Schublade | admin | hr_admin | it_admin | manager | member | readonly | extern |
|---|---|---|---|---|---|---|---|
| Persönlich | edit all | edit all | — | view **team** | view **own** | — | — |
| Job | edit all | edit all | — | view **team** | view **own** | — | — |
| Gehalt & Lohn | edit all | edit all | — | — | view **own** | — | — |
| Dokumente | edit all | edit all | — | view **team** | view **own** | — | — |
| Abwesenheiten (Akte) | edit all | edit all | — | read **team** (+approve) | read **own** | — | — |
| `self:propose` | (edit direkt) | (edit direkt) | — | ✓ | ✓ | — | — |
| `directory:full` | ✓ | ✓ | ✓ | — | — | — | — |
| `employee:offboard` | ✓ | ✓ | — | — | — | — | — |

- member salary:view **own** = sieht EIGENES Gehalt + eigene Abrechnungen (Personio-Standard „All Employees sehen eigene Daten") — löst L4-Hardcode ab.
- member documents:view **own** = eigene Dokumente, zusätzlich gefiltert nach Kategorie-`visibility === 'employee'` (L3). manager team-Scope + `visibility in ('manager','employee')`. hr_admin/admin sehen alle inkl. `hr_only`.
- it_admin bleibt OHNE alle Schubladen (USP!) — bekommt nur `directory:full` (Account-Übersicht) — und `employee:read` hat er schon.
- extern/Aushilfe: kein self:propose (§0.1), keine Schubladen; employee:read hat extern eh nicht.
- readonly (Steuerberater Elena): bewusst KEINE Schubladen, kein directory:full (Finance-Zugang reicht; bei Bedarf per Custom-Rolle).

### 3.3 Reporting-Line-Resolver (NEU, nur für team/HR — andere Module behalten team≈all, dokumentiert)

- Neue Utility `modules/team/reporting-line.ts`: baut aus der Employee-Liste (`managerUserId`) die Kette. `getDescendantUserIds(viewerUserId)` (ganze Linie runter, zyklus-sicher), `getVisibleDirectoryUserIds(viewerUserId)` (2 rauf via manager-Kette + 2 runter + self).
- Neuer Hook `useHrScopedCapability(key, targetUserId)`: own → self-Vergleich · team → self ∪ Descendants · all → true. NICHT das generische `useScopedCapability` umbauen (dort bleibt team≈all für die anderen Module — Kommentar aktualisieren, dass team/HR jetzt einen echten Resolver hat).
- Seeds (L11): `managerId`-Ketten in mock-db konsistent ziehen, 4 Ebenen für QA: Stefan (GF) → Bereichsleiter (z.B. sarah, thomas) → Teamleiter → Mitarbeiter/Aushilfe. Roster-Ids aus shared-ids.ts verwenden (NIE Namen hardcoden, memory reference_current_user_source).

### 3.4 Flächen-Umbau

1. **TeamPage-Liste + OrgChart:** ohne `directory:full` auf Umgebungssicht filtern (2↑/2↓, `getVisibleDirectoryUserIds`); Hinweis-Zeile „Eingeschränkte Ansicht" (RestrictedModeBadge-Muster). OrgChart zeigt nur den sichtbaren Ausschnitt.
2. **MemberProfileContent:** Schubladen-Gates von `useHasCapability` auf `useHrScopedCapability(key, employee.userId)` umstellen (isSelf-Bypass wird dadurch systematisch: own-Scope). Abwesenheits-Sektion (Salden/Historie) an `absence:read`-Scope. Sektionen ohne Recht KOMPLETT ausblenden (Markt-Muster §1).
3. **Bearbeiten im Profil (L-neu, §0.6):** pro Schublade Edit-Stift nur mit `*:edit`-Scope-Treffer; wiederverwendet `EditMemberDialog`-Formularlogik → als Sektions-Edit in MemberProfileContent verdrahten. Toten Code aufräumen: `MemberDetailPanel` (bis auf exportierte Sektionen) + tote TeamPage-Edit-States.
4. **Self-Service-Antrags-Flow (L4/L5):** `SelfServiceView` auf echte Daten (eigenes Profil via `/hr/employees/me`, eigene Abrechnungen nur mit salary:view own — MSW-stateful statt Hardcode). Mit `team:self:propose`: Formular „Änderung vorschlagen" (Feld alt→neu) → `POST /hr/change-requests` → Feld-Lock „Änderung ausstehend" (Badge). OHNE propose-Key: read-only + Hinweis „Änderungen über Vorgesetzte" .
5. **HR-Genehmigungs-Inbox:** neuer Bereich im Team-HR-Tab (neben HRApprovalDialog-Abwesenheiten): Karten-Liste offener Änderungsanträge, **Alt links | Neu rechts**, Genehmigen/Ablehnen (Ablehnen mit Grund), gated `data_personal:edit` (bzw. Schublade des Feldes). MSW: `CHANGE_REQUESTS`-Registry stateful (approve mutiert Employee, reject mit Grund zurück an Antragsteller).
6. **Austritts-Flow (L6, §0.5):** Einstieg im Profil-Bearbeiten (ItemActions im Profil-Header, gated `employee:offboard`) → `OffboardEmployeeDialog`: letzter Arbeitstag, Austrittsdatum, Austrittsart (Kündigung MA/AG, Fristablauf, Rente, Aufhebung), Grund (optional), Nachbesetzung-Toggle → **Zusammenfassungs-/Bestätigungs-Schritt** mit Konsequenzliste („Login wird am X gesperrt · Platz wird frei · Rollen werden entzogen") + **Abhängigkeits-Check**: ist die Person managerUserId von jemandem oder Genehmiger (absence-approver)? → Pflicht-Select „Übernimmt: …" vor Bestätigung. Kaskade in MSW: HR-Status inactive + AdminUser deactivated + USER_ROLE_ASSIGNMENTS leeren + Seat-Zähler − 1 + manager-Reassign.
7. **CreateEmployeeWizard (L7):** Rollen-Schritt gated (`team:role:assign || admin:role:assign`, sonst Default-Rolle member + Hinweis); Eskalations-Guard aus R-2 wiederverwenden (nur Rollen anbieten, deren Keys der Anleger selbst hat — admin-Preset nicht für hr_admin ohne Vollrechte).
8. **Rollen-Editor (RoleEditorPage):** Schubladen-Zeilen bekommen durch scopeable=true automatisch Scope-Selects ✓ (R-2-Mechanik); neue Keys erscheinen via Katalog. Prüfen: Scope-Select-Labels („Eigene/Team/Alle") — „Team" umbenennen in „Team (unterstellte Personen)" via i18n, damit die Linie-runter-Semantik klar ist.

### 3.5 Contract/Luke (backend-gaps §RBAC R-4-Block ergänzen)

`GET /hr/employees` muss serverseitig nach directory-Sichtbarkeit + Schubladen-Scope filtern (FE-Filter = nur UX) · `POST /hr/change-requests` + approve/reject (Registry-Shape aus MSW = Referenz) · Offboard-Endpoint mit Kaskade (Account+Rollen+Seat transaktional) + Abhängigkeits-Report (wer verwaist) · Scope-Dimension 'team' = Reporting-Line-Resolver serverseitig (managerUserId-Kette, zyklus-sicher) · Dokument-visibility serverseitig erzwingen · `self:propose`/`directory:full`/`employee:offboard` als neue Seeds.

---

## §4 Phasenplan (Bau-Reihenfolge, 1 Commit am Ende; Auto-Deploy scharf — nur Grünes pushen)

- **P1 Fundament:** Katalog-Änderungen + neue Keys + ROLE_DEFS-Matrix (§3.2) + reporting-line.ts + useHrScopedCapability + Seed-Ketten (L11) + i18n-Keys ×4 (Script `scripts/i18n-rbac-r4.mjs`, ICU `{var}`!).
- **P2 Sichtbarkeit + Profil:** Directory-Filter Liste/OrgChart + MemberProfileContent-Scope-Umbau + Abwesenheits-Schublade + Dokument-visibility (L3) + toten Code aufräumen (L2).
- **P3 Self-Service + Inbox:** SelfServiceView echt (L4) + Change-Request-Flow + HR-Inbox-Karten (L5) — MSW stateful.
- **P4 Bearbeiten + Austritt:** Sektions-Edit im Profil (§0.6) + OffboardEmployeeDialog + Kaskade + Wizard-Rollen-Gate (L7).
- **P5 Gates + Handover:** scoped tsc (`tsconfig.rbaccheck.json` erweitern) · `eslint src/ --quiet` · Key-Check ×4 · **QA `scripts/qa-rbac-enforcement-r4.mjs`** (Rollen: admin-Regression · hr_admin voll · it_admin ohne Schubladen aber directory:full · manager team-Kette (sieht Untergebenen-Akte, NICHT Chef-Akte, kein Gehalt) · member own (eigene Abrechnung sichtbar, fremde Akte zu, propose-Flow) · extern nichts · Umgebungssicht-Filter bei member in tiefer Kette) + **alle Bilder ansehen** · backend-gaps R-4-Block · RESUME-NEXT #22 · Memory.

**QA-Learnings aus R-3 übernehmen:** same-hash `page.goto` remountet nicht → via `/#/dashboard` bouncen · Rollen-Wechsel via Editor-Preview + ProfileSwitcher · exakte de.json-Labels + `getByRole` · nur 1 Dev-Server (PowerShell-Kill, kein `pkill`).

## §5 Stolpersteine

- **Drei-Stellen-Regel** für jeden neuen Key (Katalog + ROLE_DEFS + i18n ×4) — sonst Raw-Keys im Editor.
- **`useScopedCapability` NICHT global umbauen** — team≈all bleibt für alle Nicht-HR-Module (sonst brechen R-3-Gates, die 'team' pragmatisch nutzen, z.B. crm:deal:delete manager team).
- **absence-Kalender vs. -Akte** trennen (§3.1) — sonst verliert das Planungs-Board für member seine Funktion (Regression!).
- **Employee-Wizard-Popout** (Electron IPC `main/ipc/employee-wizard.ts`) beim Rollen-Gate mitprüfen.
- **Seed-Drift L9:** vor P3 klären, welcher Handler `GET /hr/employees` wirklich bedient (mocks/handlers/index.ts-Reihenfolge) — Change-Request-Flow muss auf die SSOT-Seeds schreiben.
- Loading = deny (`ready`-Guard) an neuen Flächen; Preview-Mode muss auch den Directory-Filter treffen (Editor-Preview „Als Rolle anzeigen").
- mock-db-Index-Falle (Session #15: EMPLOYEES vs IDS um 1 verschoben) — bei Ketten-Seeds NAME_TO_USER_ID/shared-ids nutzen, nie Index-Annahmen.
