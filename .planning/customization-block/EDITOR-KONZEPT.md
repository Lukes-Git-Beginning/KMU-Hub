# Modul-Editor — Konzept (SSOT für den Bau)

> **Status: LOCKED (Session #26, 2026-07-22).** Recherche-Gate durchlaufen (`IST-EDITOR.md` + `MARKT-EDITOR.md` + `DRAFT-DEPLOY.md`), mit Darien besprochen, 4 Entscheidungen gefallen. Dies ist die verbindliche Bau-Vorlage. Datenschicht-SSOT bleibt `KONZEPT.md`; hier steht die Editor-UX + Sandbox/Draft/Deploy-Architektur.

## §0 Gelockte Entscheidungen

**Aus der Vision (`EDITOR-VISION-BRIEFING.md §0`, bestätigt):**
- Editor-Modell = **edit-in-place im eigenen Editor-Fenster (Sandbox, isoliert vom Live)**. Modul-Auswahl liegt im Admin-Hub.
- Speichern = **Entwurf + geplantes/terminiertes Deployment** (Change-Management für Config).
- v1-Scope = **Fundament-Trio** pro Modul (Felder + Begriffe + Wertelisten). Layout & Sichtbarkeit = Stufe 2.

**4 Besprechungs-Entscheide (Darien, 2026-07-22):**
1. **Fenster-Form = In-App-Overlay** (95vw/94vh, im selben Fenster). Erbt Theme/Auth/i18n/Demo-Mode, kein IPC. **Auflage:** `EditorFrame` self-contained bauen (eigener Provider-Baum: Sandbox-QueryClient + MemoryRouter + DraftConfigProvider), damit ein späterer Umzug in ein echtes Electron-Fenster (`ipc/compose.ts`-Muster) nur ein Mount-Point-Wechsel ist. Option B bleibt dokumentierter Upgrade-Pfad, falls Overlay in der Praxis nicht reicht.
2. **Granularität v1 = nur Trio-Panel** (Klick auf Trio-Navigation → Eigenschaften-Panel), Live-Vorschau des Moduls daneben. **Kein** DOM-Anklicken/Drag in v1 (= Field-Registry = grüne Wiese = Stufe 2).
3. **Terminierung gleich in v1.** Deploy-Dialog „Jetzt übernehmen / Terminiert am… / Als Entwurf speichern". Rollout-Job v1 gemockt (MSW), echter Cron = Luke-Paket.
4. **Modul-Abdeckung v1 = 2 Pilot-Module** (Kontakte + Helpdesk), sauber + review-reif. Danach Galerie ausrollen (rollierende Fertigstellung, Nico reviewt parallel).

## §1 Architektur (aus `IST-EDITOR.md`, verifiziert)

**Kernbefund:** Datenschicht trägt den Editor 1:1. Der eigentliche Aufwand ist die fehlende Render-Abstraktion (Module rendern Felder hart im JSX, keine Field-Registry). **Deshalb v1 = Property-Panel mit Trio-Tabs statt DOM-Overlay.** Live-Vorschau des Moduls läuft trotzdem (ICU-Live-Fix trägt sie schon).

### Sandbox-Isolation (drei Maßnahmen)
1. **MemoryRouter** als Router-Kontext (Live nutzt `createHashRouter`, App.tsx:8/261 — verifiziert). Navigation-Hooks (`useNavigate`/`useSearchParams`) zeigen ins Leere = korrekt für Sandbox.
2. **Separater QueryClient** (`new QueryClient()` im Modal-Scope) — verhindert Cache-Kontamination des Live-Systems. Cleanup beim Unmount.
3. **Store-Writes unterdrücken** — Zustand-Stores sind global singleton; im Editor-Kontext No-Op-Handler (der Nutzer klickt in v1 ohnehin keine echten Aktionen wie „Anrufen").

### Draft-Schicht = 4. Overlay-Ebene
```
effektiv = default ⊕ vendor ⊕ tenant ⊕ draft   (draft gewinnt, nur im Editor-Kontext aktiv)
```
- `ConfigProvenance` (`api/customization-types.ts`) um `'draft'` erweitern.
- `resolveLabelOverrides(locale, base?, draftOverlay?)` + `resolveValueSet(id, base?, draftOverlay?)` in `mocks/data/customization.ts` — optionaler `draftOverlay`-Param (**sauberer als globale Variable**: keine Side-Effect-Kontamination des Live-Resolvers).
- `DraftConfigProvider` (React Context + Reducer): hält `draftLabels` / `draftValueSets` / `draftCustomFields`, exponiert `setDraftLabel/setDraftValueSet/setDraftField`, `commitDraft`, `scheduleDeploy`, `discardDraft`, `isDirty`.

### Live-Preview
- ICU-Live-Fix (`i18n/i18n.ts:37` `bindI18nStore:'added removed'`, v1.2-verifiziert QA 3/3) trägt Label-Vorschau ohne Zusatzcode.
- `setDraftLabel` → `applyLabelOverlay(locale, {…liveTenant, …draft})` → Modul-Vorschau rendert sofort neu.
- Custom-Fields-Vorschau isoliert über Sandbox-QueryClient.
- **Risiko R-1 (KRITISCH):** `applyLabelOverlay` schreibt ins globale i18n-Bundle → Draft-Labels auch im Live-Hintergrund sichtbar solange Editor offen. **Mitigation v1:** beim Schließen (Discard/Commit) Overlay via `fetchAndApply()` zurücksetzen; Sandbox-Banner kommuniziert den Zustand.

## §2 UX / Wireframe (aus `MARKT-EDITOR.md`, Cosmi-eigenständig)

```
┌──────────────────────────────────────────────────────────────────────┐
│ ◄ Kontakte bearbeiten          ⟲ ⟳   [Vorschau ⊙]      Entwurf · 3 ●  │ 48px Toolbar
├──────────────────────────────────────────────────────────────────────┤
│  ▞ ENTWURF — Änderungen sind noch nicht live.          [Verwerfen]     │ Amber-Banner
├───────────────┬────────────────────────────────────┬──────────────────┤
│  BEARBEITEN   │        MODUL-VORSCHAU (live)        │   EIGENSCHAFTEN  │
│  › Felder  3  │   [echtes Modul in Sandbox]         │  [Panel je Auswahl│
│  › Begriffe 1 │                                    │   Feld/Begriff/   │
│  › Werte-  0  │                                    │   Werteliste,     │
│    listen     │                                    │   Herkunft-Badge, │
│  260px        │   Canvas (70%)                     │   Zurücksetzen]   │
├───────────────┴────────────────────────────────────┴──────────────────┤
│  3 Änderungen               [Als Entwurf speichern]  [Übernehmen ▾]    │ 40px Footer
└──────────────────────────────────────────────────────────────────────┘
```

**Übertragbare Muster:** Drei-Panel (links Trio-Nav / Mitte Sandbox-Canvas / rechts Eigenschaften) · persistenter **Amber-Draft-Banner** (das was Odoo Studio fehlt = Differenzierer) · Progressive Disclosure im Eigenschaften-Panel (3-4 häufige Props, „Weitere Optionen ▾") · Herkunft-Badge (Standard/Zentria/Angepasst/Entwurf) · Preview-Toggle · Übernehmen-Dropdown mit Drei-Wege-Wahl.
**Anti-Muster:** Properties-Wall (Salesforce/ServiceNow), generischer Dashboard-Look, Odoo-Klon, Emojis. Fonts: Plus Jakarta Sans / Clash Display. Motion: fade+scale(0.97→1) 200ms Editor-Öffnen, Panel slide-in 150ms (nur transform/opacity).

## §3 Draft/Deploy-Modell (aus `DRAFT-DEPLOY.md`)

**Zustände:** `draft → scheduled → live → superseded`.
- draft→live (sofort · „Jetzt übernehmen") · draft→scheduled (Termin) · scheduled→live (Job an Tag X) · live→superseded (Rollback reaktiviert vorherige Version, Overlay-Operation, modul-granular).
- Drafts **sparse/intent-basiert** (nur Abweichungen, kein Full-Snapshot) — entspricht Overlay-Prinzip.
- **Marktlücke:** ServiceNow/Salesforce haben keine native Terminierung → Cosmis geplantes Deploy = Alleinstellung (nächster Verwandter LaunchDarkly, aber nur Flag-granular).

**Deploy-Dialog-UX:** Änderungs-Zusammenfassung in Klartext + betroffene User-Zahl + Toggle „Jetzt / Termin" + DatePicker (Default nächster Morgen 06:00) + optionaler In-App-Ankündigungs-Banner.
**Governance:** RBAC-Key `admin:customization:manage` (existiert) · Audit `customization.draft_committed` / `.deploy_scheduled` / `.rolled_back` · Rollback 1-Klick.

## §4 v1-Roadmap-Schnitt (Bau-Phasen)

| Phase | Inhalt | Modell |
|---|---|---|
| **E-1 Fundament** | Draft-Schicht im Resolver (`draftOverlay`-Param, `'draft'`-Provenance) · `DraftConfigProvider` · Sandbox-Kontext (QueryClient + MemoryRouter + Store-No-Op) · Audit-Events. **Selbst (Opus-nah, architektur-kritisch).** | Fundament |
| **E-2 EditorFrame** | Overlay-Rahmen (DetailModal-Basis, self-contained für späteren Fenster-Umzug) · Toolbar · Amber-Draft-Banner · Drei-Panel-Layout · Commit-Footer. Motion-Tokens. | Rahmen |
| **E-3 Trio-Panel** | Bestehende `CustomFieldsTab` + `BegriffeTab` als Panel-Inhalt **integrieren** (modul-gefiltert) · `ValueSetsTab` **neu** (v1.3-Reste fließen hier ein) · Live-Preview-Verdrahtung. | UI |
| **E-4 Modul-Galerie** | `AnpassungenHubPage` → Modul-Galerie-Kacheln (Pilot: Kontakte + Helpdesk) · Kachel-Klick öffnet EditorFrame · „was ist pro Modul anpassbar" = Vendor-Ebene (editierbares Manifest pro Modul). | Einstieg |
| **E-5 Deploy** | Übernehmen-Dropdown (Jetzt / Terminiert / Entwurf) · Deploy-Dialog mit DatePicker + betroffene-User + Ankündigung · Entwurf-Liste · Rollback · gemockter Scheduler. | Deploy |
| **Gates je Phase** | i18n ×4 (`{var}`, ICU-Plural) · scoped tsc · eslint src/ · Playwright-QA + **Bilder ansehen** · 1 Commit + Push. | — |

**Pilot-Reihenfolge:** Kontakte zuerst komplett (Referenz), dann Helpdesk. Danach Galerie auf weitere Module.

## §5 Wiederverwenden / Neu / Erweitern

| Baustein | Aktion |
|---|---|
| `mocks/data/customization.ts` Resolver | **ERWEITERN** (`draftOverlay`-Param) |
| `api/customization-types.ts` `ConfigProvenance` | **ERWEITERN** (`'draft'`) |
| `i18n/useLabelOverlay.ts` `applyLabelOverlay` · `i18n/i18n.ts` ICU-Fix | **WIEDERVERWENDEN** unverändert |
| `BegriffeTab` · `CustomFieldsTab` · `FieldEditorModal` | **INTEGRIEREN** in Editor-Panel |
| `AnpassungenHubPage` | **ERWEITERN** → Modul-Galerie |
| `components/shared/DetailModal` | **BASIS** für EditorFrame |
| `stores/permissions.ts` `startPreview` · `PermissionPreviewBanner` | **MUSTER** für Draft-Mode + Banner |
| `mocks/data/audit-events.ts` `writeAuditEvent` | **WIEDERVERWENDEN** |
| Modul-Komponenten (`KontaktePage`, `HelpdeskPage`) | **UNVERÄNDERT** (Editor rendert as-is) |
| `EditorFrame` · `DraftConfigProvider` · `ModulePreviewWrapper` · `ValueSetsTab` · Modul-Galerie · Deploy-Dialog | **NEU** |
| `ipc/compose.ts` / `ipc/employee-wizard.ts` | **MUSTER für v2** (echtes Fenster, falls Overlay nicht reicht) |

## §6 Luke-Paket (backend-gaps, nach FE-Bau eintragen)
- `tenant_customization_drafts`-Tabelle (sparse intent, `scheduled_at`, Status-Maschine draft/scheduled/live/superseded).
- Draft→Live-Promotion-Job (Cron, `FOR UPDATE SKIP LOCKED`, idempotent) an `scheduled_at`.
- Draft-Overlay als 4. Resolver-Schicht serverseitig spiegeln (nur Editor-Session).
- Rollback = vorherige Version reaktivieren (modul-granular).
- Audit `customization.draft_committed/.deploy_scheduled/.rolled_back` an die Deploy-Routen.
- Editierbares-Manifest pro Modul (Vendor-Ebene: was der Kunde anpassen darf).

## §7 Offene Risiken (aus IST-EDITOR §Risiken)
- **R-1** globale i18n-Kontamination (Mitigation: Restore beim Schließen). **R-2** Store-Seiteneffekte (No-Op im Sandbox-Kontext). **R-3** keine Field-Registry → Layout ist Stufe 2. **R-5** Draft-Persist v1 = MSW. **R-7** editierbares Manifest pro Modul = Vendor-Ebene, vor E-4 fixieren.
