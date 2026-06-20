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

**Offen / Depth-Notiz:** Header-Stat „neue Eingänge / Eingänge diese Woche" zeigt beim ersten Laden `0`, weil Submissions lazy erst beim Öffnen einer Gruppe geladen werden (nach Besuch des Eingänge-Tabs korrekt). → in F-2 eager laden.

## Definition of Done
- [x] Demo-Mode lebendig (0 failedApiRequests/pageErrors, Listen + Details + Submissions gefüllt) — F-1
- [ ] Formular-Zeile + Submission-Zeile → DetailModal mit allen Infos/Aktionen — F-2
- [ ] DnD-Formular-Builder, alle Feldtypen, Live-Vorschau, Reihenfolge persistent — F-3
- [ ] DSGVO-Einwilligungsfeld + Consent im Submission-Detail + echter Export-Download — F-4
- [ ] Moduleinstellungen (personal + tenant) via ModuleSettingsShell — F-5
- [ ] i18n ×4 vollständig, 0 Raw-Keys, 0 `{{}}` (per Screenshot verifiziert) — laufend
- [x] `reviews/formulare.md` mit DoD-Häkchen + Befunden, je Phase ein Commit/Push
