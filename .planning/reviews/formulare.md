# Review — Modul „formulare" (review-reif machen)

> Branch `parallel/formulare`, Port 5174. Sub-Terminal parallel zum berichte-Builder.
> Auftrag: `.planning/parallel-batch/sub-formulare.md`.

## Schritt 0 — Ist-Analyse (verifiziert)

- **`FormularePage.tsx` (2457 Z.)** war bereits umfangreich: Tabs (Meine Formulare / Eingänge / Vorlagen), Karten-Grid, Editor mit Live-Vorschau, Feld-Config-Dialog (inkl. bedingter Logik + Seitenumbruch), öffentliche Vorschau, Submission-Slide-over, Share/New/Delete-Dialoge.
- **KEIN MSW-Handler** für `formulare` vorhanden → Demo war **tot**: `useFormulare`-Hooks riefen `/api/v1/formulare/*`, kein Handler bediente sie → leere Listen, jeder Request fiel auf echtes `fetch` zurück (failedApiRequest). **Bestätigt** der vermutete Haupt-Befund (F-1).
- **Client-Vertrag vollständig vorhanden**: `formulare-client.ts` + `useFormulare.ts` + `formulare-types.ts` (camelCase) decken Schemas-CRUD+duplicate, Submissions list/get/create/status, Export (Blob), Webhooks, Deliveries, Stats ab.
- **i18n**: 157 `formulare.*`-Keys, Parität DE/EN/FR/IT vorhanden. Flat dot-notation, `{var}`-Interpolation (ICU). Kein `{{ }}` gefunden.
- **Latenter Crash entdeckt**: Page-Maps (`FIELD_TYPE_ICONS` etc.) deckten nur 8 Feldtypen ab — **`email` fehlte**, obwohl die API `email` als Feldtyp führt. Jedes E-Mail-Feld (in echten Formularen Standard) → `Icon = undefined` → React „Element type is invalid"-Crash → ModuleErrorBoundary. Vorher unsichtbar, weil ohne Handler nie Felder rendern.
- **`SubmissionsPanel`**: rief `onLoaded()` (Parent-`setState`) in einem `useMemo` während des Renderns → React-Warnung „Cannot update a component while rendering a different component". Vorher nie getriggert (keine Submissions luden).

## Phasen

### F-1 — MSW-Vertrag + Demo lebendig ✅
- `mocks/handlers/formulare.ts` neu (stateful, in `handlers/index.ts` registriert). Bedient **alle** Client-Endpoints: Schemas list/get/create/update/delete/duplicate, Submissions list/get/create/status, Export (echter CSV-Blob mit `Content-Disposition` + UTF-8-BOM), Webhooks, Deliveries, Stats.
- Seed: **6 Formulare** (Kundenfeedback, Kontaktanfrage, Bewerbung, Eventanmeldung, Newsletter, Support-Entwurf) + **2 Vorlagen**, verschiedene Feldtypen, je realistische **Submissions** mit Status (new/read/archived). `submissionCount` bleibt mit der Liste synchron.
- **Fix `email`-Feldtyp**: zu Store-Union + `FIELD_TYPE_LABEL_KEYS`/`ICONS`/`OPTION_KEYS` + Editor-/Public-Vorschau + Feld-Config ergänzt. i18n `formulare.fieldType.email` ×4. Behebt den Render-Crash.
- **Fix `SubmissionsPanel`**: `useMemo` → `useEffect` für den `onLoaded`-Callback.
- **Seed-Kohärenz**: `submittedBy` = „Name"-Antwort (Absender-Spalte matcht Detail).
- **QA** (`scripts/qa-formulare-f1.mjs`, :5174): 6 Formular-Karten, 6 Eingangs-Gruppen, 13 sichtbare Submission-Zeilen, Detail öffnet, 2 Vorlagen mit Feld-Chips. **0 pageErrors, 0 console errors, 0 failedApi**. Screenshots gesichtet (forms/eingänge/detail/vorlagen) — sauber, keine Raw-Keys/`{{}}`.
- Gescopter Typecheck `tsconfig.formularecheck.json` grün.

### F-2 — Zeilen → DetailModal (Tiefe) ✅
- **Formular-Karte → `DetailModal`** (zentriert): 360°-Ansicht mit Meta-Grid (Sichtbarkeit, Eingänge, Erstellt von/am, Zuletzt geändert, Seiten), Feld-Liste (Icon + Pflicht-Badge + Typ) und Aktions-Footer: **Duplizieren · Teilen · Archivieren/Aktivieren · Löschen · Vorschau · Bearbeiten**. Karte ist `role="button"` + Tastatur (Enter/Space), innere `ItemActions` per `stopPropagation` isoliert.
- **Standalone-Vorschau**: „Vorschau" öffnet ein zweites DetailModal (mit Zurück-Pfeil) das das Formular wie für externe Nutzer rendert (`renderFieldPreview`, alle Feldtypen inkl. E-Mail) — entkoppelt vom Editor-Draft.
- **Submission-Zeile → `DetailModal`** statt Slide-over (`DetailPanel` → `DetailModal`, `width` → `maxWidth`). „Kein Slide-over" erfüllt.
- **Eager-Stats-Fix** (Depth-Notiz aus F-1 erledigt): Submissions werden jetzt per `useQueries` für alle aktiven Formulare vorab geladen (geteilter Query-Key dedupliziert mit den Gruppen-Panels). Header + Stat-Cards sind ab dem ersten Laden korrekt („9 neue Eingänge", „16 diese Woche"). Lazy-`onLoaded`/`setState`-Muster komplett entfernt.
- **i18n** `formulare.detail.*` ×4 (9 Keys: subtitle, visibility, private, submissions, createdBy/At, updatedAt, pages, noFields).
- **QA** (`scripts/qa-formulare-f2.mjs`): Form-Modal (5 Feld-Zeilen, 6 Aktionen), Vorschau (4 Inputs), Submission-Modal zentriert. **0 pageErrors, 0 failedApi**, keine Raw-Keys. Screenshots gesichtet.

### F-3 — DnD-Formular-Builder (dnd-kit) ✅
- Die Editor-Feldliste ist jetzt eine **echte Drag-and-drop-Liste** (`DndContext` + `SortableContext` + `verticalListSortingStrategy`). Neue `SortableFieldItem`-Komponente (Modul-Ebene, sonst würde `useSortable` bei jedem Render remounten): Drag-Listeners auf einem **dedizierten Grip-Handle**, innere Edit/Löschen-Buttons mit `data-field-control` + `onPointerDown`-Stop (kein Drag-vs-Klick-Konflikt, wie work-Kanban). `PointerSensor` (5px-Aktivierung) + `KeyboardSensor` (a11y).
- `onDragEnd` → `arrayMove` über `draft.fields` → `reorderFields` (Store, vorher ungenutzt). Live-Vorschau spiegelt die neue Reihenfolge sofort; Speichern persistiert via MSW-PATCH.
- Alle Feldtypen in der Palette (inkl. neu `email`), Feld-Eigenschaften (Label/Pflicht/Platzhalter/Optionen/bedingte Logik) unverändert nutzbar. Seitenumbrüche sind ebenfalls sortierbar.
- i18n `formulare.editor.feldVerschieben` ×4 (Drag-Handle-aria-Label).
- **QA** (`scripts/qa-formulare-f3.mjs`): Handle „Name" von Pos 0 → 2 gezogen (`changed: true`), nach Speichern + Neuöffnen Reihenfolge erhalten (`matchesAfter: true`), Palette-Add (Datum) ergibt 6 Felder. **0 pageErrors.** Screenshots gesichtet (Handles + Live-Preview-Sync).

### F-4 — DSGVO-Einwilligungsfeld + Submission-Tiefe + Export ✅
- **Neuer Feldtyp `consent`** (Frontend/Demo-Pseudo-Typ, in Store + API ergänzt mit `consentText` + `privacyUrl`). In der Palette als „Einwilligung" (ShieldCheck). Beim Hinzufügen automatisch Pflicht + Default-DSGVO-Text + Datenschutz-Link. Feld-Config zeigt für consent eigene Eingaben (Zweckbindungstext + Link), Pflicht-Toggle ausgeblendet (immer Pflicht).
- **Rendering**: Consent rendert in Editor-Live-Vorschau, standalone Vorschau und öffentlicher Editor-Vorschau als Checkbox mit Zweckbindungstext + verlinkter Datenschutzerklärung.
- **Submission-Tiefe**: Detail zeigt Meta-Grid (Absender + IP-Adresse) und einen prominenten **Consent-Bestätigungsblock** („Eingewilligt am {date}" mit ShieldCheck + erfasstem Zweckbindungstext); consent wird aus der normalen Antwortliste ausgeblendet (keine Dopplung).
- **Echter Export**: `useExportSubmissions` verdrahtet (vorher ungenutzter Stub). Wiederverwendbare `ExportMenu` (CSV/XLSX) im **Eingänge-Gruppen-Header** (pro Formular) und im **Form-Detail-Modal**. MSW liefert echten CSV-Blob (UTF-8-BOM, `Content-Disposition`) → echter Datei-Download. Der irreführende globale Toast-Stub wurde entfernt.
- **Seed**: alle „Datenschutz-Einwilligung"-Felder auf `consent` upgegradet.
- i18n ×4: `fieldType.consent`, `consent.*` (8), `submission.ipAdresse`, `export.downloadStarted`/`perFormHint` — 12 Keys.
- **QA** (`scripts/qa-formulare-f4.mjs`): consent im Form-Detail, Submission-Consent-Block + IP, **echter Download** (`kundenfeedback_eingaenge.csv`, `downloaded: true`), Consent-Feld via Palette hinzugefügt. **0 pageErrors, 0 failedApi.** Screenshots gesichtet.

### F-5 — Moduleinstellungen + i18n + QA ✅
- **Neuer Prefs-Store** `stores/formularePrefs.ts` (zustand+persist): personal (`defaultTab`, `defaultExportFormat`) + tenant (`defaultConsentText`, `defaultPrivacyUrl`, `notifyOnSubmission`, `retentionDays`).
- **`FormulareSettingsPanel`** via `ModuleSettingsShell` (personal-Section „Ansicht" + tenant-Section „Datenschutz & Eingänge"), in `module-settings-registry.tsx` registriert (`id: formulare`, Gruppe `module`, `navMatch ['/formulare']`, Icon `FileInput`).
- **Prefs greifen echt**: `defaultTab` → Initial-Tab der Page (QA: öffnet auf „Vorlagen"); `defaultExportFormat` → ExportMenu listet das Standard-Format zuerst + „Standard"-Marker (QA: XLSX zuerst); `defaultConsentText`/`defaultPrivacyUrl` → Vorbelegung neuer Consent-Felder (Konstanten in den Store verschoben).
- i18n ×4: `moduleSettings.entries.formulare`, `formulare.settings.*` (23), `formulare.export.standard` — 24 Keys. **Parität: alle 4 Sprachen 200 `formulare.*`-Keys.**
- **QA** (`scripts/qa-formulare-f5.mjs`): Settings-Overlay rendert das Panel (personal + tenant, keine Raw-Keys), defaultTab + defaultExportFormat greifen, Screenshots bei 1440/1280/1024. **0 pageErrors.** Screenshots gesichtet (Overlay + Reflow).

## Ergebnis
Alle 5 Phasen grün, je Phase ein Commit + Push auf `parallel/formulare`. i18n DE/EN/FR/IT vollständig + paritätisch, gescopter Typecheck grün, Demo lebendig (0 failedApi/pageErrors). Modul ist review-reif.

## Definition of Done
- [x] Demo-Mode lebendig (0 failedApiRequests/pageErrors, Listen + Details + Submissions gefüllt) — F-1
- [x] Formular-Zeile + Submission-Zeile → DetailModal mit allen Infos/Aktionen — F-2
- [x] DnD-Formular-Builder, alle Feldtypen, Live-Vorschau, Reihenfolge persistent — F-3
- [x] DSGVO-Einwilligungsfeld + Consent im Submission-Detail + echter Export-Download — F-4
- [x] Moduleinstellungen (personal + tenant) via ModuleSettingsShell — F-5
- [x] i18n ×4 vollständig, 0 Raw-Keys, 0 `{{}}` (per Screenshot verifiziert) — laufend
- [x] `reviews/formulare.md` mit DoD-Häkchen + Befunden, je Phase ein Commit/Push
