# IST-Analyse: Edit-in-place Modul-Editor — Machbarkeit
> Erstellt: 2026-07-22 | Basis: Echter Code-Befund (verifiziert via Grep/Read, kein Raten)
> Bezug: `EDITOR-VISION-BRIEFING.md §2` — sechs Analyse-Achsen, Deliverable = Machbarkeits-Urteil

---

## Kurzfazit

Edit-in-place im eigenen Editor-Fenster ist **realistisch, aber der Hauptaufwand liegt nicht in den Daten — er liegt in der fehlenden Render-Abstraktion der Module**. Die Datenschicht (Overlay-Resolver, ICU-Live-Fix, Draft-Erweiterung) ist sauber erweiterbar. Das größte strukturelle Problem: Module rendern ihre Felder, Labels und Layouts komplett **hart im JSX** — es gibt keine Field-Registry, keinen Renderpool, kein Layout-Protokoll. Das bedeutet, edit-in-place in Form von „klick ein Element an → Panel öffnet sich" setzt voraus, dass die anklickbaren Elemente zuerst annotiert werden. Das ist machbar und das Richtige, aber es ist der eigentliche Baustein dieser Phase, nicht die Datenschicht.

**Empfehlung:** Editor-Fenster als großes In-App-Overlay (kein zweites Electron-Fenster). Modul-Komponente läuft im Editor ohne Live-Router-Kontext, mit eigener React-Query-Instanz und `DraftConfigProvider`. Klick-Annotationen per `data-editable`-Attribute + Editor-Chrome-Overlay (kein `innerHTML`-Hacking). Draft-Schicht als vierte Provenance-Stufe über den bestehenden Resolver. Live-Preview via vorhandenen ICU-Fix.

---

## Achse 1: Modul-Architektur

**Urteil: GEHT MIT AUFWAND**

### Routing

`App.tsx:8` — `createHashRouter` (Hash-Routing für Electron `file://`). Module sind lazy-loaded als `React.LazyExoticComponent` unter `DeskEnvironment`. Routing über `react-router-dom v6`.

`KontakteLayout.tsx:8-9` — Module nutzen `NavLink` + `Outlet` für Sub-Navigation, `useTranslation`, `useCapabilitySet`. `KontaktePage.tsx` nutzt `useNavigate`, `useSearchParams` für Modals und Navigation nach `/mails`, `/chat`.

**Konsequenz für isoliertes Rendern:** Eine Modul-Komponente direkt außerhalb des Live-Routers zu rendern ist möglich — React Router v6 erlaubt das mit einem eigenen `StaticRouter` oder `MemoryRouter`. Die Navigation-Hooks (`useNavigate`) würden ins Leere zeigen, was für die Sandbox OK ist (kein echter Navigate-Intent). `useLocation`/`useSearchParams` brauchen den MemoryRouter als Kontext.

### Stores

Module hängen an **Zustand-Stores** (`useContactsStore`, `useWorkStore`, `useMeetingsStore`, `useNavigationStore` etc.). Zustand-Stores sind **global singleton** — sie existieren im JS-Modul-Scope, unabhängig vom React-Kontext. Eine isoliert gerenderte Modul-Instanz würde denselben globalen Zustand lesen wie die Live-App.

`KontaktePage.tsx:29-32` — 4 verschiedene Stores in einer Seite: `useContactsStore`, `useMeetingsStore`, `useNavigationStore` + `useCrmPrefsStore`. Der `startCall`-Handler in der Sandbox würde den Live-`useMeetingsStore` mutieren.

**Konsequenz:** Seiteneffekte wie `startCall`, `setIntent`, `setGroupManagerOpen` etc. könnten für den Editor-Kontext einfach no-ops werden (die Modul-Komponente kann in der Sandbox mit Props oder einem dünnen Mock-Store gerendert werden, der Write-Operationen unterdrückt). Alternativ: Editor-Sandbox-Store-Proxy der alle Writes feuert, aber keinen globalen Zustand mutiert.

### React Query

`App.tsx:107-116` — ein globaler `queryClient` mit IDB-Persister. Module nutzen `useContacts()`, `useTask()`, `useProjects()` etc. via diesen globalen Client.

**Konsequenz:** Die Sandbox braucht einen **eigenen QueryClient** (`new QueryClient()`), der nur die Sandbox-Abfragen hält. Das geht sauber: `<QueryClientProvider client={sandboxClient}>` als Wrapper um die Modul-Komponente. Der Sandbox-Client holt dieselben MSW-Daten wie der Live-Client, hat aber keinen Einfluss auf den Cache des Live-Systems.

### Fazit Achse 1

Isolation ist machbar mit drei Maßnahmen:
1. `MemoryRouter` als Router-Kontext für die Sandbox
2. Separater `QueryClient` für die Sandbox (verhindert Cache-Kontamination)
3. Store-Writes: im Editor-Kontext unterdrücken (Store-Proxy oder no-op-Callbacks)

---

## Achse 2: Feld-/Render-Abstraktion

**Urteil: GRÜNE WIESE — der größte Aufwand dieser Phase**

### Befund: Felder sind hart im JSX

`KontaktePage.tsx` (886 Zeilen) — Felder, Labels, Status-Chips sind komplett inline im JSX. Es gibt keine Abstraktion „dieses Element ist editierbar" oder „dieses Element gehört zu Feld X". Ein Kontakt-Name wird als direktes `{contact.firstName} {contact.lastName}` gerendert — kein Wrapper, kein `data-field`-Attribut.

`ContactDetailPanel.tsx` — Detail-Panel ohne Field-Registry. Felder hardcoded im JSX.

`CustomFieldsSection.tsx:21-202` — Diese Komponente ist die nächste bestehende Abstraktion: rendert Custom Fields generisch per `definitions.map()`. Sie arbeitet mit einem `CustomFieldInfo[]`-Array und gibt je nach `fieldType` unterschiedliche Inputs zurück. **Aber:** sie ist auf Custom Fields beschränkt, nicht auf alle Felder des Moduls. Kern-Felder (Name, E-Mail, Firma) sind nach wie vor hartkodiert.

`modules/formulare/FormularePage.tsx:4547` — `renderField(field)` als lokale Funktion für Formular-Felder. Modul-intern, nicht wiederverwendbar.

### Keine Field-Registry

Eine app-weite `FieldRegistry` (Typ: `Record<ModuleId, FieldDefinition[]>`) existiert **nicht**. Es gibt:
- `report-sources/registry.ts` — 11 Quellen mit `FieldDefinition[]` für Berichte-Builder (modul-agnostisch, aber nur für Reports)
- `block-registry.ts` — für die Dokument-Block-Engine

Diese sind **nicht** die gesuchte Edit-in-place-Abstraktion. Sie beweisen aber, dass das Codebase-Muster „Registry + generischer Renderer" bereits existiert und grundsätzlich bekannt ist.

### Was das für Edit-in-place bedeutet

Ohne Abstraktion gibt es zwei Ansätze für „Elemente anklicken → Panel":

**Option A (Annotation ohne Abstraktion):** `data-editable`-Attribute auf ausgewählten Elementen in den Modul-Komponenten. Der Editor-Chrome scannt den DOM nach diesen Attributen und rendert eine Click-Overlay-Schicht. Kein Umbau der Modul-Komponenten nötig, aber fragil und schwer wartbar.

**Option B (Abstrahierte Field-Registry, empfohlen):** Eine `ModuleFieldRegistry` definiert pro Modul, welche Felder anpassbar sind (Kern-Felder + Custom Fields). Der Editor rendert das Modul mit einer `EditableFieldWrapper`-Komponente über diese Felder. Das Modul selbst bleibt unverändert; die Editor-Sandbox-Schicht überlagert anpassbare Elemente.

Für **v1 (Trio: Custom Fields + Labels + Value-Sets)** reicht eine abgespeckte Variante: Der Editor öffnet das Modul, zeigt aber primär die bestehenden Tab-UIs (CustomFieldsTab, BegriffeTab, künftiger ValueSetsTab) in einem modul-zentrierten Rahmen — nicht echtes DOM-Overlay. Das ist der pragmatische Einstieg. Echtes „Element anklicken = Property-Panel" kommt in v2 mit der Field-Registry.

---

## Achse 3: Sandbox-Isolation + Draft-Schicht

**Urteil: GEHT — sauber erweiterbar, Resolver-Architektur passt perfekt**

### Bestehender Resolver

`mocks/data/customization.ts` — `resolveLabelOverrides(locale, base?)` + `resolveValueSet(id, base?)`. Die Resolver implementieren bereits `default → vendor → tenant` mit `base=true`-Schalter (zeigt reine Code-Default-Baseline, R-6-Muster).

```
effektiv = code_default ⊕ vendor_overlay ⊕ tenant_overlay
                                              ↑ gewinnt per key
```

### Draft als vierte Schicht

Der Resolver ist eine reine JavaScript-Funktion (`resolveLabelOverrides`), die auf in-memory Maps arbeitet (`vendorLabels`, `tenantLabels`). Eine Draft-Schicht einzuziehen bedeutet: eine dritte Map (`draftLabels`) hinzufügen und sie in der Resolution-Kette als oberste Schicht einbinden.

**Erweiterung (minimal, nicht-breaking):**

```typescript
// Neue vierte Schicht
let draftLabels: LocaleLabelMap = {}
let draftValueSets: Record<string, ValueSet> = {}
let draftActive = false

export function enableDraftMode(): void {
  draftLabels = {}
  draftValueSets = {}
  draftActive = true
}
export function disableDraftMode(): void {
  draftActive = false
  draftLabels = {}
  draftValueSets = {}
}
export function commitDraft(): void {
  // Draft-Inhalt in tenant-Layer übernehmen
  for (const [locale, map] of Object.entries(draftLabels)) {
    tenantLabels[locale] ??= {}
    Object.assign(tenantLabels[locale], map)
  }
  disableDraftMode()
}

// In resolveLabelOverrides: nach tenantValue check:
if (draftActive) {
  const draftValue = draftLabels[locale]?.[key]
  if (draftValue !== undefined) {
    result[key] = { key, value: draftValue, provenance: 'draft' as ConfigProvenance }
    continue
  }
}
```

**Provenance** muss um `'draft'` erweitert werden (`ConfigProvenance` in `customization-types.ts:40`). Draft-Badges (gelb/amber) im Editor analog zu tenant-Badges (grün).

### Draft nur im Editor-Kontext

Der Draft-Modus darf nicht global aktiv sein — sonst sieht der Live-User Draft-Overrides. Lösung: `DraftConfigContext` (React Context), der beim Editor-Fenster-Mount `enableDraftMode()` ruft und beim Unmount `disableDraftMode()`. Die Sandbox-Instanz (eigener QueryClient, eigener MemoryRouter) enthält diesen Context als Wrapper.

**Alternativ** (sicherer, keine globale Variable): Die Resolver-Funktion bekommt einen optionalen `draftOverlay`-Parameter, der nur im Editor-Kontext befüllt ist. Keine globale Side-Effect-Variable. Das ist die sauberere Architektur.

### Value-Set-Draft

Analog: `resolveValueSet(id, base?, draftOverlay?)`. Der `draftOverlay` ist ein `Record<string, ValueSet>`, der im Editor lokal gehalten wird und nicht in den globalen Store schreibt, bis Commit.

---

## Achse 4: Wiederverwendung

**Urteil: GEHT — das meiste bleibt, wenig wird ersetzt**

### Was direkt wiederverwendbar ist

| Baustein | Datei | Status | Nutzung im Editor |
|---|---|---|---|
| `resolveConfig` / `resolveLabelOverrides` / `resolveValueSet` | `mocks/data/customization.ts` | Fertig, funktioniert | Draft-Schicht als 4. Layer ergänzen |
| ICU-Live-Fix | `i18n/i18n.ts:37` — `bindI18nStore: 'added removed'` | Fertig, verifiziert | Live-Preview im Editor-Fenster funktioniert bereits |
| `applyLabelOverlay` + `captureDefaults` | `i18n/useLabelOverlay.ts` | Fertig | Editor ruft `applyLabelOverlay(locale, draftLabels)` für Live-Vorschau |
| `CustomFieldsTab` + `FieldEditorModal` | `modules/admin/anpassungen/` | Fertig (v1.1) | In Editor-Sidebar als Panel-Inhalt direkt einbinden |
| `BegriffeTab` | `modules/admin/anpassungen/BegriffeTab.tsx` | Fertig (v1.2) | In Editor-Sidebar als Panel-Inhalt direkt einbinden |
| `DetailModal` | `components/shared/DetailModal.tsx` | Fertig | Editor-Fenster = DetailModal (maxWidth='max-w-7xl' o. fullscreen) |
| `startPreview` / `endPreview` | `stores/permissions.ts:54-55` | Fertig (R-5) | Muster für Draft-on/off-Toggle; Editor-Sandbox-Context analog |
| `PermissionPreviewBanner` | `components/layout/PermissionPreviewBanner.tsx` | Fertig | Vorlage für Draft-Banner („Du bearbeitest — nicht live") |
| RBAC Two-Pane-Muster | `UserOverrideEditorPage.tsx`, `RoleEditorPage.tsx` | Fertig (R-6) | Editor-Chrome-Layout: linkes Panel = Modul-Vorschau, rechts = Property-Panel |
| `StagedCommitFooter` (implizit in R-6) | RBAC-Editoren | Fertig | Dirty-State + Übernehmen/Verwerfen-Footer im Editor |
| Audit `writeAuditEvent` | `mocks/data/audit-events.ts` | Fertig | Draft-Commit feuert `customization.draft_committed` |
| `ConfigProvenance` + `ConfigLayer` Typen | `api/customization-types.ts` | Fertig | `'draft'` als neue Provenance ergänzen |
| Electron-Fenster-Muster | `ipc/compose.ts`, `ipc/employee-wizard.ts` | Fertig | Vorlage für echtes zweites Fenster (falls gewünscht) |

### Was integriert (nicht ersetzt) wird

- `CustomFieldsTab` + `BegriffeTab`: **bleiben als Komponenten**, werden aber in den Editor-Sidebar eingebunden statt als Admin-Tab. Der standalone „Anpassungen"-Hub kann auf die Editor-Einstiege verlinken.
- `AnpassungenHubPage.tsx`: bleibt als Modul-Galerie/Einstieg im Admin-Hub. Klick auf Modul → öffnet Editor-Fenster.

### Was neu gebaut wird

- `EditorFrame` (neues Komponente): großes Overlay/Modal (DetailModal-Basis), enthält MemoryRouter + Sandbox-QueryClient + DraftConfigProvider + Modul-Vorschau + Property-Panel + Editor-Chrome (Toolbar, Sandbox-Banner, Commit-Footer)
- `DraftConfigProvider` (React Context + Reducer): hält den lokalen Draft-Zustand, exponiert `setDraftLabel`, `setDraftValueSet`, `setDraftField`, `commitDraft`, `discardDraft`
- `ModulePreviewWrapper`: rendert die Modul-Komponente mit Editor-Annotationen (v1: ohne DOM-Overlay, nur mit Marker für anklickbare Bereiche)
- Neue `ConfigProvenance: 'draft'` in `customization-types.ts`
- `editor:open-window` IPC-Handler (optional, falls echtes zweites Fenster gewählt)

### Was ersetzt wird

Nichts wird aktiv ersetzt. Der bestehende `AnpassungenHubPage`-Tab bleibt; er wird zum Galerie-Einstieg. Die bisherigen Sub-Tabs (Felder/Begriffe) bleiben nutzbar, werden aber sekundär.

---

## Achse 5: Fenster-Mechanik

**Urteil: GEHT — zwei Optionen, In-App-Overlay ist empfohlen**

### Was existiert

`DetailModal.tsx` — das Standard-Cosmi-Fenster-Muster. Zentriertes Dialog (`max-h-[90vh]`), Gradient-Stripe, Header mit Close + optionalem Back. **Basis für den Editor-Rahmen.**

```typescript
// DetailModal.tsx:52
className={cn('flex max-h-[90vh] flex-col gap-0 overflow-hidden rounded-2xl p-0', maxWidth, className)}
```

Zwei existierende sekundäre Electron-Fenster-Muster:
- `ipc/compose.ts:4-25` — E-Mail-Verfassen-Fenster (680×620px, eigenes Hash-Route `#/compose-window`)
- `ipc/employee-wizard.ts` — Mitarbeiter-Wizard (960×720px, Hash-Route `#/employee-wizard-window`)

Beide laden dieselbe App (gleicher Vite-Build) mit anderem Hash-Route-Einstieg. Die App bootet komplett neu im neuen Fenster — eigene Provider, eigene Query-Client-Instanz, eigene Store-Instanz.

### Option A: In-App-Overlay (empfohlen für v1)

**Was:** `DetailModal` mit `maxWidth="max-w-[95vw] max-h-[95vh]"` + `className="m-0"` als vollbild-nahe Overlay. Der Modul-Inhalt wird innerhalb des Overlays gerendert, eingebettet im bestehenden Provider-Baum.

**Vorteile:**
- Kein IPC-Aufwand, kein neues Electron-Fenster
- Cosmi-Theme, Auth-State, i18n, Toaster — alles geerbt
- Schließen via ESC oder Close-Button — Standard-Dialog-Verhalten
- MSW (Demo-Mode) läuft im selben Window-Kontext
- Sandbox-QueryClient als lokale Variable im Modal-Scope

**Nachteile:**
- Nicht in einem eigenen OS-Fenster (kein eigenes Minimize/Maximize)
- Modal und Live-App teilen denselben JS-Thread (kein echtes Isolierungs-Konzept)

**Eigener QueryClient im Modal:** `const [sandboxClient] = useState(() => new QueryClient())` — wird beim Modal-Mount erzeugt, beim Unmount verworfen. `<QueryClientProvider client={sandboxClient}>` als Wrapper.

**Eigener Router-Kontext:** `<MemoryRouter initialEntries={[modulePath]}>` innerhalb des Modals.

### Option B: Echtes zweites Electron-Fenster

**Was:** Neues IPC-Handle `editor:open-window` in `src/main/ipc/` analog zu `compose.ts`. Öffnet neues `BrowserWindow` mit Route `#/editor-window?module=kontakte`. Rendert den Editor vollständig isoliert.

**Vorteile:**
- Echte OS-Fenster-Isolation (eigener JS-Context, eigenes MSW, eigene Stores)
- Resizable, minimierbar, eigenes Title-Bar
- Entspricht Dariens Wortlaut „extra Fenster für den Editor"

**Nachteile:**
- Auth-State muss übergeben werden (localStorage-basiert, wird geerbt, aber IDB-Cache nicht sofort verfügbar)
- MSW-Handler laufen erneut (sauber, kein Problem, aber doppelter Init-Overhead)
- IPC für Commit → Live-App (Commit im Editor-Fenster muss den globalen customization-Store im Main-Fenster updaten — braucht `ipcRenderer.send` + `ipcMain`-Handler + Main-Window-Event-Bus). Nicht trivial.
- Größeres Setup

**Empfehlung für v1:** Option A (In-App-Overlay). Fühlt sich als „eigenständiges Fenster" an (95vw/95vh, eigener Rahmen, Sandbox-Banner), ohne IPC-Komplexität. Option B als spätere Stufe möglich.

---

## Achse 6: Live-Preview

**Urteil: GEHT — Mechanismus existiert und ist verifiziert**

### Vorhandener ICU-Live-Fix

`i18n/i18n.ts:37`:
```typescript
.use(new ICU({ bindI18nStore: 'added removed', bindI18n: 'languageChanged' }))
```
Dieser Fix (v1.2, Session #25, verifiziert QA `qa-customization-v12-final.mjs 3/3 PASS`) sorgt dafür, dass `addResourceBundle` den ICU-Cache leert und alle `t()`-Konsumenten re-rendern. **Dieser Mechanismus ist die Basis der Live-Preview.**

### Label-Overrides im Editor

`i18n/useLabelOverlay.ts:78-98` — `applyLabelOverlay(locale, labels)`:
1. Baut `overrides: Record<string, string>` aus der ResolvedLabelMap
2. Ruft `i18n.addResourceBundle(locale, 'translation', overrides, true, true)`
3. Ruft `i18n.changeLanguage(locale)` für Re-Render

Im Editor: jedes Mal wenn ein Draft-Label gesetzt wird, ruft der `DraftConfigProvider` `applyLabelOverlay` mit dem aktuellen Draft auf. Der Modul-Vorschau-Bereich rendert sofort mit dem neuen Label. Das funktioniert bereits ohne zusätzlichen Code — der ICU-Fix macht es live.

**Einschränkung:** `applyLabelOverlay` schreibt in das **globale** i18n-Bundle — auch die Live-App im Hintergrund würde während des Editor-Sessions das Draft-Label sehen. Lösung: beim Editor-Close (`discard` oder `commit`) einmal den echten Live-Overlay neu anlegen (via `fetchAndApply`). Alternativ: Draft-Labels nur in einer lokalen i18n-Instanz halten (sauberer, aber mehr Aufwand: `i18n.cloneInstance()` gibt es in i18next nicht out-of-the-box).

**Pragmatische v1-Lösung:** Draft-Labels auch in das globale Bundle schreiben, aber beim Editor-Schließen ohne Commit den Overlay zurücksetzen (`clearAllLabelOverrides('draft')` + `fetchAndApply`). Der Nutzer sieht Draft-Labels auch im Live-Hintergrund während der Editor offen ist — das ist ein akzeptabler Trade-off für v1, der durch den Sandbox-Banner kommuniziert wird.

### Custom Fields im Editor

Custom Fields (Definitionen) ändern sich in der Sandbox via Draft-CustomFields-Store. Da die Modul-Komponente ihre Custom Fields via React Query holt (`useTaskCustomFields`, `useCustomFields`), sind sie in der Sandbox-QueryClient-Instanz isoliert: Draft-Field-Definitionen werden vom Sandbox-QueryClient gecacht, der Live-QueryClient sieht sie nicht.

### Value-Sets im Editor

Analog zu Labels: `resolveValueSet(id, draftOverlay)`. Der Draft-Overlay-Parameter enthält die lokalen Änderungen. Modul-Komponenten, die Value-Sets konsumieren, müssen dazu entweder über den `resolveValueSet`-Resolver gehen (was neue Komponenten tun) oder via React Query (Sandbox-Client übernimmt).

---

## Empfohlener technischer Ansatz: Sandbox + Draft-Schicht + Live-Preview

### Architektur-Überblick

```
Admin-Hub → Modul-Galerie-Kachel [„Kontakte bearbeiten"] 
  → EditorModal (DetailModal maxWidth='max-w-[96vw] h-[94vh]')
      ├─ DraftConfigProvider (React Context)
      │    ├─ draftLabels: LocaleLabelMap
      │    ├─ draftValueSets: Record<string, ValueSet>  
      │    ├─ draftCustomFields: CustomFieldDefinition[]
      │    ├─ setDraftLabel(locale, key, value)   → applyLabelOverlay() für Live-Preview
      │    ├─ commitDraft()                        → writeAuditEvent + MSW-Persist
      │    └─ discardDraft()                       → fetchAndApply() für Restore
      ├─ QueryClientProvider (sandboxClient = new QueryClient())
      ├─ MemoryRouter (initialEntries=['/kontakte'])
      │    └─ <KontaktePage />  ← unveränderte Modul-Komponente
      └─ EditorChrome (links: Modul-Vorschau, rechts: Property-Panel)
           ├─ Sandbox-Banner: „Entwurf — nicht live"
           ├─ Modul-Vorschau (70% Breite)
           │    └─ [data-editable]-Annotationen auf anklickbaren Elementen
           ├─ Property-Panel (30% Breite)
           │    ├─ [v1] CustomFieldsTab (eingebettet, aus bestehendem Code)
           │    ├─ [v1] BegriffeTab (eingebettet, modul-gefiltert)
           │    └─ [v1] ValueSetsTab (neu, v1.3)
           └─ Footer: [Übernehmen] [Als Entwurf speichern] [Verwerfen]
```

### Draft-Schicht im Resolver (empfohlene Variante ohne globale Variable)

```typescript
// customization-types.ts — ConfigProvenance erweitern:
export type ConfigProvenance = 'default' | 'vendor' | 'tenant' | 'draft'

// customization.ts — resolveLabelOverrides mit optionalem draftOverlay:
export function resolveLabelOverrides(
  locale: string,
  base = false,
  draftOverlay?: LocaleLabelMap,  // NEU — nur im Editor-Kontext befüllt
): ResolvedLabelMap {
  // ... bestehende Logik ...
  // Nach tenantValue-Check:
  const draftValue = draftOverlay?.[locale]?.[key]
  if (draftValue !== undefined) {
    result[key] = { key, value: draftValue, provenance: 'draft' }
    continue
  }
}
```

### DraftConfigProvider (neu)

```typescript
interface DraftConfigContextValue {
  draftLabels: LocaleLabelMap
  draftValueSets: Record<string, ValueSet>
  isDirty: boolean
  setDraftLabel: (locale: string, key: string, value: string) => void
  setDraftValueSet: (id: string, set: Omit<ValueSet, 'layer'>) => void
  commitDraft: () => Promise<void>  // → MSW + Audit + applyLabelOverlay (live-persist)
  discardDraft: () => void           // → fetchAndApply() restore
}
```

Der Provider ruft bei jedem `setDraftLabel` sofort `applyLabelOverlay(locale, { ...liveTenantLabels, ...draftLabels[locale] })` auf — das ist der Live-Preview-Mechanismus.

### Sandbox-QueryClient

```typescript
// Im EditorModal:
const [sandboxClient] = useState(
  () => new QueryClient({ defaultOptions: { queries: { staleTime: STALE_TIME } } })
)
// Cleanup beim Unmount:
useEffect(() => () => sandboxClient.clear(), [sandboxClient])
```

---

## Wiederverwenden vs. Neu vs. Ersetzen

| Komponente / Datei | Aktion | Begründung |
|---|---|---|
| `mocks/data/customization.ts` — `resolveLabelOverrides`, `resolveValueSet` | **ERWEITERN** (optional `draftOverlay`-Param) | Resolver-Architektur ist exakt richtig, nur vierte Schicht ergänzen |
| `api/customization-types.ts` — `ConfigProvenance` | **ERWEITERN** (`'draft'` hinzufügen) | Ein Typen-Eintrag, alles andere passt |
| `i18n/useLabelOverlay.ts` — `applyLabelOverlay` | **WIEDERVERWENDEN** unverändert | ICU-Live-Fix ist die Live-Preview-Engine |
| `i18n/i18n.ts` — ICU-Fix | **WIEDERVERWENDEN** unverändert | Fundament, bereits verifiziert |
| `modules/admin/anpassungen/BegriffeTab.tsx` | **INTEGRIEREN** (in Editor-Sidebar einbetten) | Zu Property-Panel-Inhalt machen; nicht als Admin-Tab-Standalone ersetzen |
| `modules/admin/anpassungen/CustomFieldsTab.tsx` + `FieldEditorModal.tsx` | **INTEGRIEREN** (in Editor-Sidebar einbetten) | Analog BegriffeTab |
| `modules/admin/anpassungen/AnpassungenHubPage.tsx` | **ERWEITERN** (wird Modul-Galerie) | Aus Tab-Inhalt wird Modul-Galerie-Kacheln → öffnen Editor-Fenster |
| `components/shared/DetailModal.tsx` | **WIEDERVERWENDEN** als Basis | maxWidth + height via Props auf Fullscreen-nah strecken |
| `stores/permissions.ts` — `startPreview` / `endPreview` | **MUSTER ÜBERNEHMEN** | Draft-Mode analog zum Permission-Preview-Modus |
| `components/layout/PermissionPreviewBanner.tsx` | **MUSTER ÜBERNEHMEN** | Draft-Banner visuell analog gebaut |
| `mocks/data/audit-events.ts` — `writeAuditEvent` | **WIEDERVERWENDEN** | Draft-Commit schreibt `customization.draft_committed` |
| `ipc/compose.ts` + `ipc/employee-wizard.ts` | **MUSTER FÜR SPÄTERE STUFE** (v2: echtes Fenster) | Pattern dokumentiert; für v1 nicht nötig |
| Modul-Komponenten (`KontaktePage`, `HelpdeskPage`, etc.) | **UNVERÄNDERT WIEDERVERWENDEN** | Kein Umbau nötig; Editor-Chrome rendert sie als-is |

---

## Risiken & offene technische Fragen

### R-1: Globale i18n-Kontamination durch Draft-Labels (KRITISCH)

`applyLabelOverlay` schreibt ins globale i18n-Bundle. Draft-Labels sind damit auch im Live-Hintergrund sichtbar, solange der Editor offen ist. **Mitigation v1:** Beim Editor-Schließen (Discard oder Commit) den Overlay-Stand wiederherstellen. Beim Commit: ohnehin Persist + Reload. Beim Discard: `fetchAndApply()` für sauberen Reset.

### R-2: Zustand-Store-Seiteneffekte in der Sandbox (MITTEL)

Module wie `KontaktePage` rufen `startCall()`, `navigate('/mails')` etc. Diese würden in der Sandbox den Live-Zustand verändern. **Mitigation:** Im Sandbox-Kontext No-Op-Handler via Props oder einen Sandbox-Context übergeben, der alle navigations-seitigen Stores auf No-Op stellt. Für v1 des Editors (der nur Felder/Begriffe/Wertelisten bearbeitet, keine echten Aktionen ausführt) ist das Risiko gering — der Nutzer klickt nicht auf „Anrufen" im Editor.

### R-3: Keine Field-Registry — „Element anklicken = Property-Panel" ist grüne Wiese (MITTEL-GROSS)

Edit-in-place im DOM-Sinne (Hover-Outline auf anklickbare Elemente) setzt Annotations-Arbeit voraus. Für v1 vermeiden: Property-Panel zeigt stattdessen die drei Trio-Tabs (Felder/Begriffe/Wertelisten). DOM-Hover-Overlay kommt in v2 mit Field-Registry.

### R-4: Modul-Komponenten haben Navigation-Dependencies (KLEIN)

`useNavigate`, `useLocation`, `useSearchParams` brauchen Router-Kontext. Im MemoryRouter sind sie vorhanden, navigieren aber nur im Sandbox-Router. Kein echter Seiten-Wechsel beim Klick auf „Zur Firma" im Editor. Das ist korrektes Verhalten für eine Sandbox.

### R-5: Draft-Persist und Entwurf-Speichern (OFFEN)

Wie wird ein gespeicherter Entwurf persistiert, der noch nicht committed ist? Optionen:
a) MSW-only (reicht für v1 Demo)
b) Eigene `tenant_drafts`-Tabelle im BE (Luke-Paket)
c) `tenant_settings` mit Key `customization.draft` (JSONB, quick-and-dirty)

### R-6: Terminiertes Deploy (geplant für v1, §0 Entscheidung 2) (OFFEN)

Geplantes Deploy (Commit an Tag X) = Scheduler-Job, der den Draft-Layer in den Tenant-Layer promotet. Das braucht:
- `scheduled_at`-Timestamp am Draft-Objekt
- Backend-Cron oder Job-Queue (Luke-Track)
- FE: Datepicker im Commit-Dialog
Für v1 MVP: sofortiges Commit (`commitDraft()`) + „Als Entwurf speichern" ohne Termin. Terminierung als v1.x-Schritt.

### R-7: Modul-Galerie: welche Module wann editierbar (KONZEPTIONELL OFFEN)

Welche Module erscheinen in der Editor-Galerie? Alle 34 Module oder erst Pilot-Set? Welche Felder sind pro Modul anpassbar (v1: nur Custom Fields + Labels + Value-Sets = Trio)? Das muss vor dem Bau entschieden werden (Darien-Frage §5.4 aus dem Briefing).

---

## Referenz-Dateien (alle verifiziert)

| Datei | Relevanz |
|---|---|
| `desktop/src/renderer/src/App.tsx` | Router (`createHashRouter`), QueryClient-Setup, Provider-Baum |
| `desktop/src/renderer/src/main.tsx` | App-Root (StrictMode, kein eigener Provider) |
| `desktop/src/renderer/src/modules/kontakte/KontaktePage.tsx` | Typisches Modul: Store-Dependencies, Navigation, JSX-Felder |
| `desktop/src/renderer/src/modules/kontakte/KontakteLayout.tsx` | NavLink + Outlet, useCapabilitySet |
| `desktop/src/renderer/src/modules/helpdesk/HelpdeskPage.tsx` | Anderes Modul: useHelpdeskPrefsStore, useCapabilitySet, DetailModal |
| `desktop/src/renderer/src/modules/work/components/CustomFieldsSection.tsx` | Nächste bestehende Feld-Abstraktion (Custom Fields only) |
| `desktop/src/renderer/src/modules/admin/anpassungen/AnpassungenHubPage.tsx` | Bestehender Hub-Einstieg (Sub-Tab-Controller) |
| `desktop/src/renderer/src/modules/admin/anpassungen/BegriffeTab.tsx` | Wiederverwendbarer Label-Editor |
| `desktop/src/renderer/src/modules/admin/anpassungen/CustomFieldsTab.tsx` | Wiederverwendbarer Felder-Editor |
| `desktop/src/renderer/src/components/shared/DetailModal.tsx` | Fenster-Muster-Basis |
| `desktop/src/renderer/src/api/customization-types.ts` | Typen: ConfigProvenance, ConfigLayer, ValueSet etc. |
| `desktop/src/renderer/src/mocks/data/customization.ts` | Resolver + CRUD + Draft-Erweiterungspunkt |
| `desktop/src/renderer/src/i18n/i18n.ts` | ICU-Live-Fix (`bindI18nStore: 'added removed'`) |
| `desktop/src/renderer/src/i18n/useLabelOverlay.ts` | `applyLabelOverlay` + `captureDefaults` |
| `desktop/src/renderer/src/stores/permissions.ts` | `startPreview`/`endPreview` als Muster |
| `desktop/src/main/ipc/compose.ts` | Sekundäres-Fenster-Muster (für v2-Option B) |
| `desktop/src/main/ipc/employee-wizard.ts` | Sekundäres-Fenster-Muster (für v2-Option B) |
