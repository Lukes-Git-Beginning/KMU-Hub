# Ein Modul in den Anpassungs-Editor bringen

> Die Vorlage für den Rollout. Der Editor kann seit den Sessions #32–#36 alles, was
> ein Modul braucht — offen ist nur noch, ihn auf die übrigen Module zu zeigen.
> Diese Datei beschreibt, wie das geht, und welche Fallen dabei Zeit gekostet haben.
>
> Zwei Referenz-Module, beide fertig instrumentiert:
> **Helpdesk** (`modules/helpdesk/HelpdeskPage.tsx`) — alle sieben Dimensionen an
> einem Modul mit zustandsgeführten Tabs. Jeder Schritt unten nennt die Stelle, an
> der er dort steht.
> **Kontakte** (`modules/kontakte/KontakteLayout.tsx`) — wie ein Modul mit **eigener
> Navigation** dazukommt, ohne dass Deep-Links brechen (Abschnitt 6).
>
> Wer ein Modul neu aufnimmt, fängt bei **Abschnitt 0** an: dort steht, welche
> Funktionen es bekommen muss, in welcher Reihenfolge, und wann es fertig ist.

## Worum es geht

Der Editor öffnet ein Modul in einer Sandbox und lässt den Kunden daran sieben Dinge
verändern: **Begriffe** (Beschriftungen umbenennen), **Wertelisten** (Status,
Prioritäten …), **Felder** (Zusatzfelder), **Bereiche** (Tabs an/aus), **Statistik**
(Kacheln an/aus), **Spalten** (Listen-Layout) und **Kanäle** (Ticket-Herkunft). Was
er verändert, landet in einem Entwurf; „Übernehmen" schreibt ihn in die Tenant-Ebene.

Ein Modul wird dafür **instrumentiert**: statt fester Beschriftungen und harter Enums
liest es seine Texte, Chips, Felder und Spalten aus den Hooks in
`components/customization/EditorSurface.tsx`. Alle Hooks sind außerhalb der Sandbox
No-Ops — instrumentierter Code läuft in der Live-App unverändert.

---

## 0 · Was jedes Modul bekommen muss

Damit der Editor bei allen 32 Modulen gleich funktioniert und nicht jedes Modul seine
eigene Teilmenge kann, gilt: **jede Dimension, die auf ein Modul zutrifft, wird auch
gebaut.** Nicht jede trifft auf jedes Modul zu — hier die Regel, wann welche fällig ist.

| Dimension | Fällig, wenn das Modul … | Nicht fällig, wenn … |
|---|---|---|
| **Begriffe** | überhaupt Überschriften, Tab- oder Abschnittsnamen hat → **trifft immer zu** | — |
| **Wertelisten** | einen Status, eine Priorität, eine Art/Kategorie o. Ä. als festen Enum rendert | das Modul nur freie Texte und Zahlen zeigt |
| **Felder** | Datensätze mit Detail-Ansicht führt (Kunde, Beleg, Gerät, Vertrag …) | es reine Auswertung ohne eigene Datensätze ist |
| **Bereiche** | mehr als eine Ansicht unter einem Modul-Dach hat (Tabs, Sub-Navigation) | es genau eine Ansicht ist |
| **Statistik** | eine Kennzahlen-/Auswertungsansicht hat | keine Kennzahlen vorkommen |
| **Spalten** | eine Tabelle oder Liste mit mehreren Spalten hat | es Kacheln/Board/Kalender ohne Spalten ist |
| **Kanäle** | Datensätze von außen entgegennimmt (Formular, Portal, E-Mail) | alles intern angelegt wird |

**Immer und ohne Ausnahme dazu** — das ist kein „je nach Modul", sondern der
Grundstock, an dem sich anfühlt, ob der Editor ein Werkzeug ist oder ein Formular:

- `useEditorGuard` um jede verändernde Aktion,
- `useEditorFocusEffect` **und** `useEditorContextReport` (beide Richtungen),
- jeder `dkey` in `LABEL_WHITELIST`.

### Reihenfolge der Arbeit pro Modul

Immer dieselbe, damit zwei Leute an zwei Modulen dasselbe Ergebnis bauen:

1. **Zuschnitt** — Tabelle oben durchgehen, festhalten welche Dimensionen zutreffen.
   Bei „trifft nicht zu" kurz **begründen** (im Registry-Kommentar), sonst sieht es
   später aus wie vergessen.
2. **Registry-Eintrag** anlegen (Abschnitt 1) — er ist die Absichtserklärung.
3. **Instrumentieren** in der Reihenfolge aus Abschnitt 2: Begriffe → Wertelisten →
   Bereiche → Felder → Spalten → Statistik → Kanäle. Begriffe zuerst, weil der Rest
   auf denselben Beschriftungen aufsetzt.
4. **Whitelist** nachtragen und `vitest run …/label-whitelist.test.ts` fahren.
5. **Fokus beidseitig** verdrahten (der Schritt, der am ehesten liegen bleibt).
6. **QA-Suite** nach Vorlage schreiben und die Screenshots ansehen (Abschnitt 4).
7. Ein Commit, ein Push.

### Wann ist ein Modul fertig?

Abnahme-Checkliste — alles davon, sonst gilt es als angefangen, nicht als fertig:

- [ ] Registry-Eintrag vollständig, „trifft nicht zu"-Fälle begründet
- [ ] jede zutreffende Dimension instrumentiert
- [ ] `label-whitelist.test.ts` grün
- [ ] verändernde Aktionen laufen durch `useEditorGuard`, Tab-Wechsel **nicht**
- [ ] beide Fokus-Richtungen verdrahtet, die blanke Liste meldet `null`
- [ ] eigene QA-Suite, **echtes Electron**, alle Prüfungen grün
- [ ] eine Umbenennung und eine Bereichs-Abschaltung sind **nach dem Übernehmen im
      echten Modul** nachgewiesen — nicht nur in der Vorschau
- [ ] Screenshots angesehen: keine rohen Schlüssel, kein zerlegtes Layout, keine
      leeren Zustände

---

## 1 · Registry-Eintrag

`modules/admin/anpassungen/editor/editorModules.ts`, Array `EDITOR_MODULES`:

| Feld | Bedeutung |
|---|---|
| `key` | stabiler Modul-Schlüssel, taucht in Routen und Entwürfen auf |
| `titleKey` | Anzeigename, **immer** ein `rbac.module.*`-Key. Modulnamen sind unveränderlich (Darien 2026-07-22) — nie in die Whitelist |
| `previewPath` | Route, an der die Sandbox startet |
| `icon` | Lucide-Name für die Kachel in der Galerie |
| `labelKeys` | umbenennbare **Inhalts**-Überschriften (Objekt-/Datensatz-Nomen), nie der Modulname |
| `valueSetIds` | Wertelisten, die dieses Modul im Editor anbietet |
| `fieldEntities` | Entitäten für Zusatzfelder |
| `areas` | ein-/ausschaltbare Tabs oder Abschnitte. `key` ist der modul-eigene Tab-Schlüssel |
| `statWidgets` | Kacheln/Diagramme der Statistik-Ansicht. `locked: true` = Feature noch nicht gebaut → in der Auswahl grau, im Modul verborgen |
| `listColumns` | eingebaute Spalten der Liste. `valueSetId` setzen, wenn die Spalte bereits eine Werteliste rendert — sonst bietet das Spalten-Panel sie unter „Wertelisten ohne Spalte" an, obwohl sie längst eine hat |
| `intake` | nur für Module mit konfigurierbaren Erfassungs-Kanälen (bislang Helpdesk) |
| `Component` | die Modul-Seite, `lazy()`-geladen |

---

## 2 · Instrumentierung im Modul

Alle Hooks aus `@/components/customization/EditorSurface`.

### Überschriften umbenennbar machen

```tsx
<h3 className="…"><EditableText as="span" dkey="helpdesk.stats.byStatus" /></h3>
```

Steckt die Beschriftung in einem Bedienelement (Tab, Karte, Schalter), dann
`interactive` setzen — sonst frisst der Editor den Klick, mit dem der Nutzer den Tab
wechseln will:

```tsx
<button onClick={…}>
  <EditableText as="span" dkey={item.dkey} interactive />
</button>
```

Frei stehend = **einfacher Klick** benennt um. `interactive` = **Doppelklick**.

> **Pflichtschritt:** jeden `dkey` zusätzlich in `LABEL_WHITELIST`
> (`mocks/data/customization.ts`) eintragen. Siehe Falle 1 — der Test in
> `mocks/data/__tests__/label-whitelist.test.ts` erzwingt es.

### Chips aus Wertelisten

```tsx
const priorityValueSet = useModuleValueSet('ticket_priority')
const prioBy = new Map((priorityValueSet?.options ?? []).map((o) => [o.id, o]))
// Label mit Rückfall auf das i18n-Enum, falls eine Option fehlt:
const priorityLabels = { low: prioBy.get('low')?.label ?? t('helpdesk.priority.low'), … }
```

Gerendert wird mit `VsChip` aus `@/components/shared` — die eine Stelle, die
Chip-Form und Farb-Rückfall für alle Module entscheidet:

```tsx
<VsChip label={priorityLabels[t.priority]} color={priorityColorOf(t.priority)}
        fallbackClass={priorityColors[t.priority]} />
```

_Helpdesk: Zeilen 219–247._

### Tabs und Abschnitte schaltbar machen

```tsx
const areaEnabled = useModuleAreas('helpdesk')
const visibleTabs = ALL_TABS.filter((key) => areaEnabled[key] !== false)
```

`!== false` statt `=== true`: ein fehlender Schlüssel bedeutet **an**. Sonst wäre ein
frisch instrumentiertes Modul beim ersten Öffnen leer. Dazu gehört ein Effekt, der
den aktiven Tab umsetzt, wenn der Nutzer genau den ausschaltet.

_Helpdesk: Zeilen 324–341._

### Zusatzfelder

```tsx
const fieldDefs = useModuleCustomFields('helpdesk_ticket')   // sichtbar + sortiert
const fieldOptions = useFieldOptions(fieldDefs)              // key → Auswahlmöglichkeiten
```

`useFieldOptions` löst beide Fälle auf: an eine Werteliste gebundene Felder holen
Beschriftung, Farbe und Reihenfolge von dort, ungebundene nutzen ihre freien Optionen.
Das ist auch der Weg, auf dem eine **neu erstellte** Werteliste im Modul ankommt —
ohne ein Feld, das auf sie zeigt, hat eine Liste keinen Ort.

### Listenspalten

Drei Handgriffe, nichts davon modulspezifisch:

```tsx
const columnLayout = useModuleColumnLayout('helpdesk')
const visibleColumns = orderColumns(
  columnDefs.filter((c) => c.optIn ? areaEnabled[`col:${c.key}`] === true
                                   : areaEnabled[`col:${c.key}`] !== false),
  columnLayout.layout,
)
// am <th>:
<th style={columnWidthStyle(columnLayout.layout, col.key)} …>
```

Eingebaute Spalten sind standardmäßig sichtbar (`!== false`), aus Zusatzfeldern
abgeleitete standardmäßig aus (`=== true`). Ziehbare Kanten und die Prozent-Anzeige
gehören dazu — Muster in Helpdesk Zeilen 574–640 und 850–890, inklusive
`freezeWidths` (siehe Falle 4).

### Entfernte Optionen

```tsx
const statusMigration = useValueSetMigration('ticket_status')
const effectiveStatus = statusMigration[ticket.status] ?? ticket.status
```

Der Entfernen-Dialog verspricht „Bestehende Einträge werden geändert auf: X". Ohne
diese Zuordnung hält das Versprechen nur in der Vorschau.

### Verändernde Aktionen sperren

```tsx
const guard = useEditorGuard()
<button onClick={guard(() => deleteTicket(id))}>…</button>
```

Alles einpacken, was etwas verändert oder aus dem Modul heraus navigiert (anlegen,
löschen, senden, zuweisen, eskalieren). **Nicht** einpacken, was nur den Zustand
umschaltet (Tab wechseln, ein Detail öffnen) — die Vorschau muss begehbar bleiben.

### Fokus in beide Richtungen

Das ist die Instrumentierung, die am ehesten vergessen wird, und die den Editor
zusammenhängend wirken lässt.

**Leiste → Vorschau:** ein Klick auf „Statistik" links soll die Vorschau auf den
Statistik-Tab stellen, nicht den Nutzer suchen lassen.

```tsx
useEditorFocusEffect({
  statistik: () => { setSelectedTicketId(null); setTab('statistik') },
  felder:    () => { setTab('tickets'); setSelectedTicketId(baseTickets[0]?.id ?? null) },
  spalten:   () => { setSelectedTicketId(null); setTab('tickets') },
  …
})
```

**Vorschau → Leiste:** wandert der Nutzer im Modul, ziehen Leiste und
Eigenschaften-Panel mit.

```tsx
useEditorContextReport(tab === 'statistik' ? 'statistik' : selectedTicketId ? 'felder' : null)
```

> **Regel für die Rückmeldung: nur eindeutige Orte melden.** Die blanke Liste ist
> gleichzeitig Heimat von Begriffen, Wertelisten, Bereichen **und** Spalten — dort
> `null` melden. Meldet ein Modul dort etwas, schaukeln sich die beiden Richtungen
> gegenseitig auf und reißen die Vorschau vom Fleck.

_Helpdesk: Zeilen 356–378._

---

## 3 · Die Fallen

Alle in den Sessions #34–#36 bezahlt.

**1 · `LABEL_WHITELIST` ist der Deploy-Filter.** `applyDraftToTenant` schreibt nur
Beschriftungen, deren Key dort steht. Ein `EditableText` ohne Eintrag lässt sich im
Editor umbenennen, die Vorschau zeigt den neuen Namen — und beim Übernehmen ist er
still weg. Kein Fehler, keine Meldung. Genau so lief die Spalten-Runde auf, und im
August 2026 fanden sich 20 weitere Keys in derselben Lage.
→ Der Test `mocks/data/__tests__/label-whitelist.test.ts` prüft jeden instrumentierten
`dkey` und jeden umbenennbaren Registry-Key gegen die Liste. Neue dynamische Stelle
(`dkey={ausdruck}`) dort in `KNOWN_DYNAMIC_SOURCES` eintragen.

**2 · `isDirty` heißt „weicht von Live ab", nicht „ungespeichert".** Wer es als
zweites liest, baut Speichern-Logik, die im falschen Moment greift.

**3 · Zwei Fenster sind zwei JS-Heaps.** Der Editor läuft als eigenes
`BrowserWindow`. Alles Fensterübergreifende gehört in geteilten Speicher, und jede
Mutation liest vorher frisch (`beforeMutation`) — sonst überschreibt das eine Fenster
blind, was das andere gerade geschrieben hat.

**4 · Prozent-Spaltenbreiten brauchen einen Container mit fester Breite.** Der
zurückgenommene Versuch „Vorschau einpassen" (`02d8c3da`) hat die Tabelle
auseinandergezogen. Verwandt: der Wechsel auf `table-layout: fixed` gibt jeder Spalte
ohne eigene Breite den gleichen Anteil — deshalb friert `freezeWidths` vor dem ersten
Ziehen den Ist-Zustand ein, sonst springt die ganze Liste beim ersten Pixel.

**5 · Prozent-Höhen brauchen einen Elternteil mit echter Höhe.** Sonst bleibt das
Balkendiagramm unsichtbar (`8c05d9a3`).

---

## 4 · QA

**Regel, teuer gelernt:** alles Fenster- oder IPC-Nahe mit einer **echten
Electron-Suite** prüfen. Browser-Suiten sehen das zweite Editor-Fenster nicht und
haben in #35 drei Fehler durchgelassen.

Vorlagen in `desktop/scripts/`:

| Suite | prüft |
|---|---|
| `qa-editor-begriffe-s.mjs` | Umbenennen → Übernehmen → Name steht im **echten** Modul |
| `qa-editor-wertelisten-n/o/p.mjs` | Wertelisten bis ins Modul, Übernehmen/Zurückrollen, eigene Liste |
| `qa-editor-entwurf-r.mjs` | Entwürfe, Namen, Fensterverhalten |
| `qa-editor-electron-l.mjs` | Spalten: Breite, Reihenfolge, Sichtbarkeit über Fenstergrenzen |
| `qa-editor-fokus-m.mjs` | beide Fokus-Richtungen |

Ablauf: `npm run dev` (genau **ein** Dev-Server), dann die Suiten **einzeln** fahren.
Nach Code-Änderungen den Dev-Server neu starten. Weitere Stolpersteine: `innerText`
sieht keine Eingabefelder und liefert CSS-Großschreibung — Beschriftungen im
Bearbeiten-Zustand über den Feld-Wert prüfen, nicht über den Fließtext.

**Und: die Screenshots wirklich ansehen.** „8/8 grün" beweist, dass die Prüfungen
liefen, nicht dass die Seite gut aussieht.

> Eine Suite, die einen Fehler fangen soll, sollte einmal **gegen den kaputten Stand**
> gefahren werden. Bei `qa-editor-begriffe-s.mjs` war genau das der Beweis: mit
> fehlenden Whitelist-Keys bleiben die Vorschau-Prüfungen grün und nur die
> Modul-Prüfungen fallen um — dieselbe Täuschung, die den Fehler so teuer gemacht hat.

---

## 5 · Reihenfolge des Rollouts

Aus `.planning/customization-block/MODUL-AUDIT.md`:

1. finanzen, inventar, einkauf, vertraege, produktion, vermietung, formulare, work
2. kalender, zeiterfassung, rapporte, fuhrpark

**kontakte** ist seit 2026-08-10 erledigt (siehe unten) und taugt jetzt als zweite
Vorlage — Helpdesk zeigt zustandsgeführte Tabs, Kontakte zeigt, wie ein Modul mit
eigenen Routen dazu kommt.

**Zu `work` — geprüft, keine Sonderbehandlung nötig.** Es sieht auf den ersten Blick
aus wie Kontakte (verschachtelte `<Routes>` unter `work/*`), hat aber gar keine
Bereichs-Leiste: „Navigation is handled by the main sidebar/topnav — no sub-nav
needed", und die inneren Routen sind Detail-Ansichten (`projects/:id/*`), keine Tabs.
Für den Editor wird deshalb **nicht** `WorkLayout` registriert — sonst stünde die
Vorschau leer, weil `<Routes>` im Editor-Fenster nichts trifft — sondern direkt die
Seite, die man beurteilen soll (`projects/ProjectsListPage`). `areas: []` ist dort
inhaltlich richtig, kein Mangel.

**Regel daraus für jedes Modul:** registriert wird die Komponente, die ohne passende
URL etwas Sinnvolles zeigt. Hat ein Modul eine eigene Bereichs-Leiste, muss diese
zustandsgeführt sein (Kontakte-Muster); hat es keine, nimmt man die Seite selbst.

## 6 · Module mit eigener Navigation (das Kontakte-Muster)

Erledigt am 2026-08-10 — hier steht, weil dasselbe bei jedem Modul mit eigener
Bereichs-Leiste wiederkommt.

**Das Problem:** Kontakte navigierte seine sechs Bereiche über `NavLink` + `<Outlet/>`,
also über den Router. Die Sandbox hat aber keine passende URL — der Inhalt bliebe leer.
Einen eigenen Router darf sie auch nicht aufmachen: `#/editor-window` ist selbst eine
Route des globalen `createHashRouter` (`App.tsx`), ein `MemoryRouter` darin wäre
verschachtelt, und react-router 7.17 verbietet das. (Der Kommentarkopf in
`ModuleSandbox.tsx` verspricht hier mehr, als stimmt — er nennt den MemoryRouter „the
path for the future real-window mode". Das eigene Fenster gibt es inzwischen, es hilft
aber nicht. Der Weg kostet einen eigenen React-Root, nicht nur einen Wrapper.)

**Die Lösung, die jetzt steht** — nachbauen, wenn das nächste Modul eine eigene
Bereichs-Leiste hat:

1. Der aktive Bereich wird **Zustand** im Layout, die Bereichs-Seiten lädt das Layout
   selbst per `lazy()` + `<Suspense>`.
2. Die Routen bleiben als **Einstieg** stehen, ohne eigenes `element` — beide
   Richtungen verdrahten:
   - **URL → Zustand** per Effekt auf `location.pathname` (Deep-Link, Zurück-Button).
   - **Zustand → URL** beim Klick (`navigate`), damit Links teilbar bleiben.
   - Im Editor entfällt nur der zweite Teil: `if (!editing) navigate(path)`.
3. **Detail-Seiten bleiben echte Routen** und rendern weiter über `<Outlet/>`. Das
   Layout entscheidet per `matchPath`, ob es den Outlet oder den Bereich zeigt.
4. Der Pfad-Vergleich muss **Präfix** sein, längster gewinnt — sonst leuchtet auf
   jeder Detail-Seite der erste Bereich statt dem, aus dem sie kommt. Genau das taten
   die `NavLink`s mit `end: false`, und genau das ging beim ersten Versuch verloren.
5. Registry zeigt danach auf das **Layout**, nicht auf die Listenseite, und `areas`
   listet die Bereiche.

Muster: `modules/kontakte/KontakteLayout.tsx`. Abnahme: `scripts/qa-kontakte-bereiche-t.mjs`
prüft beide Hälften — Live-Routing (Klick, URL, Deep-Link, Zurück, Detail) und Editor
(Leiste sichtbar, Bereich abschalten, Ergebnis im echten Modul).
