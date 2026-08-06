# kontakte — Mock-Exit DONE (Referenz-Modul)

> **Stand 2026-06-23.** kontakte/crm ist das erste vollständig echt geschaltete Modul (READ + voller CRUD durch die echte UI gegen das lokale Docker-Backend). Dient als **Referenz-Pattern** für alle weiteren Mock-Exit-Module.

## Was läuft echt (Live-verifiziert, Screenshots in `desktop/.qa-screenshots/crud-*.png`)
- **Liste**: echte Seed-Kontakte mit Name, Firma, **Position** (Job-Rolle), E-Mail, Telefon, Datum. Keine Raw-Keys.
- **Detail/360°**: Stammdaten, Notizen, Tags, **Deals** (real, crm), E-Mail-Verlauf, Aufgaben, Dateien, Timeline — alles echt geladen.
- **CRUD**: Create durch echte UI → Backend (`201`) → erscheint in Liste + Toast. Update `PUT 200`, Delete `200`. API-bewiesen + UI-bewiesen.
- Login `demo@local.test` / `Demo1234!` (jetzt **admin**, sonst 403 bei Mutationen).

## Das Referenz-Pattern (für jedes weitere OpenAPI-getippte Modul)
1. **Casing-Helper `api/casing.ts`** — `dual(obj, 'camelName')` liest beide Casings (Gateway snake_case ↔ OpenAPI camelCase, X-3). Nur OpenAPI-getippte Module brauchen das; handgetippte snake_case-Clients matchen schon.
2. **Mode-Branch im Write-Adapter** — `DEMO_MODE` aus `mocks/demo-mode-flag.ts` (Leaf, zieht keinen Handler-Graph): Mock behält vollen Feldumfang, echtes Backend bekommt nur backend-konforme Felder.
3. **Wire-Shape gegen echtes Backend prüfen** (nicht nur Mock): Methode (PUT vs PATCH!), Feldnamen (position vs title!), Pflicht-`custom_fields`-Form, Idempotency-Key bei Mutationen, RBAC-Rolle.

## Mock-verdeckte Bugs, die der Echt-Schaltung auffielen (alle gefixt FE-seitig)
| Bug | Mock | Echtes Backend | Fix |
|---|---|---|---|
| Update-Methode | akzeptiert PATCH | nur **PUT** (405 bei PATCH) | Hook → `authenticatedRequest` PUT; Mock-Handler → `http.put` |
| Job-Position | `title`/`_jobTitle` | `position`-Feld | Adapter: `position ↔ jobTitle` (READ+WRITE) |
| custom_fields | Objekt OK | **Array `[{field_id,value}]`** mit echten UUIDs (400 bei Objekt) | Real-Branch sendet `[]`; Extra-Felder = Backend-Lücke |
| RBAC | egal | `member` hat kein `contacts:write` (403) | Demo-User → admin (Seed idempotent) |

---

# 🔧 BACKEND-HANDOVER (Luke)

Drei Punkte, die nur backendseitig sauber lösbar sind:

### 1. Contact-Schema zu dünn (höchster Hebel)
Das Contact-Schema kennt nur Kernfelder (first_name, last_name, email, phone, **position**, notes, company_id, tags). Die UI hat **9 Extra-Felder** ohne Backend-Pendant: `mobile, address, jobTitle(≈position), department, website, category, status, socialMedia, projects`. Bisher per FE-Hack durch `custom_fields` geschleust — gegen das echte Backend unmöglich (custom_fields verlangt echte Custom-Field-UUIDs).
**Optionen:** (a) `extras jsonb`-Spalte am contact für frei-form UI-Felder, ODER (b) First-Class-Spalten für die häufigen (mobile/department/address). Bis dahin persistieren diese Felder gegen echtes Backend **nicht** (bewusste Mock/Real-Divergenz).

### 2. OpenAPI-Spec-Drift contacts (X-3, konsolidieren)
Die generierte Spec weicht vom Gateway ab:
- `PATCH /contacts/{id}` in der Spec, aber Gateway-Route ist **`PUT`**.
- `CreateContactRequest.title` in der Spec, Gateway-Feld ist **`position`** (kein `title`-json-Tag).
- `custom_fields: {object}` in der Spec, Gateway erwartet **`[{field_id,value}]`**.
- `ContactInfo` ist camelCase getippt, Wire ist snake_case.
→ Spec auf Gateway-Realität fixen + Typen regenerieren (das ist die globale X-3-Konsolidierung, Option B aus der Casing-Entscheidung).

### 3. Timeline-Endpoint hängt
`GET /api/v1/crm/contacts/{id}/timeline` (CHRONIK im 360°) bleibt gegen das lokale Backend im Lade-Spinner. Prüfen ob Route/Service vorhanden + antwortet.

---

# 📋 camelCase-Risiko-Set — nächste Mock-Exit-Module (X-3)

Nur diese Module lesen über OpenAPI-Typen (`components['schemas']`/`apiClient`) camelCase → brauchen `dual()`-Adapter beim Echt-Schalten. **Rest matcht schon** (handgetippte snake_case-Clients).

| Modul | Entität | Datei(en) | camelCase-Leser | Status |
|---|---|---|---|---|
| crm/companies | CompanyInfo | `modules/crm/companies/*` | `createdAt`, `contactCount`, `domain→website` | ✅ echt (`adapters.ts` backendCompanyToUI) |
| crm/deals | DealInfo | `useDeals.ts` (Pipeline + 360°) | `contactName`, `expectedCloseDate`, `stageId/stageName` | ✅ echt (backendDealToUI im Hook) |
| crm/pipeline-stages | PipelineStageInfo | `usePipelineStages.ts`, AuswertungenPage | `sortOrder`, `isWon`, `isLost`, `totalValue`, `dealCount` | ✅ echt (backendStageToUI im Hook) |
| crm/tags | TagInfo | `useContactTags.ts` | `entityType` (Hook-intern) | ✅ echt (dual()) |
| work/CustomFields | CustomFieldInfo | `modules/work/components/CustomFieldsSection.tsx` | `isRequired`, `fieldType`, `entityType`, `sortOrder` | ⬜ Sub-Terminal Lane B (work) |

**Alle crm-Module echt-geschaltet (23.06.):** kontakte, companies, deals, pipeline-stages, tags. Muster je Modul: Read-Normalizer (`dual()` + ggf. Feldname-Drift) im Hook/Adapter, Write mode-branched (`custom_fields:[]` real), Update PATCH→PUT (Hook + Mock), live per Playwright verifiziert.

**„Matcht schon" (snake_case, kein Casing-Fix):** chat (messages/channels/mentions), notifications, work (tasks/comments/activities/files/entity-links), email, timeline, activities, contact-tags (normalisiert).

**Wichtig:** Auch bei „matcht schon"-Modulen NICHT casing-blind sein — Methode/Wire-Shape/Idempotency/RBAC pro Modul gegen echtes Backend prüfen (siehe kontakte-Lehren oben).
