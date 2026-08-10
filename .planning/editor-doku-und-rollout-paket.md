# Editor: Dokumentation + Rollout-Vorbereitung (Paket C)

> ## ✅ ABGEARBEITET in Session #37 (2026-08-10)
>
> Alle vier Punkte erledigt und gepusst. Was daraus geworden ist:
>
> | | Ergebnis |
> |---|---|
> | **C1** `VsChip` | `components/shared/VsChip.tsx` (`7d6f2eb6`) |
> | **C2** LABEL_WHITELIST | Wächter-Test gebaut, fand **20 still verworfene Keys** (`7d6f2eb6`) |
> | **C3** Kontakte | Dariens Entscheidung: Tabs auf Zustand, gebaut (`139fde20`), QA 11/11 |
> | **C4** Dokumentation | **`docs/EDITOR-MODULE-ROLLOUT.md`** (`30f8114f` + `7879ccc2`) |
>
> **Weiter geht es mit `.planning/editor-rollout-paket.md`** (Paket D, erstes Modul
> `finanzen`). Die Inhalte dieser Datei sind in die Doku übergegangen — sie bleibt nur
> als Beleg stehen, wonach gearbeitet wurde.
>
> Der Text unten ist der **ursprüngliche Auftrag**, unverändert.

---

> Übergabe aus Session #36 (2026-08-09). A und B sind abgearbeitet, C ist der
> nächste Block — Darien: *„speicher alles ab, wir machen das in nem neuen Terminal
> weiter."*
>
> **Reihenfolge offen** — Darien wurde gefragt, ob zuerst die Dokumentation oder
> zuerst `VsChip` kommt, hat aber die Session beendet. Beides steht unten; die
> Dokumentation trägt die anderen drei Punkte als Arbeitspakete mit, also ist sie
> der naheliegende Einstieg.

---

## C1 · `VsChip` nach `components/shared/` heben

**Ist:** liegt lokal in `HelpdeskPage.tsx`. Jedes Modul, das Wertelisten-Chips
rendert, braucht ihn — beim Rollout würde er sonst achtmal kopiert.

**Auftrag:** nach `components/shared/` verschieben (Name beibehalten), Helpdesk
umstellen, Export in die shared-Index-Datei. Danach ist er die eine Stelle, die
Chip-Farbe/Fallback für alle Module entscheidet.

**Warum vor dem Rollout:** steht als ⚠ im Rollout-Plan (`.planning/customization-block/NEXT-TERMINAL-PLAN.md` §Rollout).

## C2 · `LABEL_WHITELIST` pro Modul pflegen

**Die Falle:** `applyDraftToTenant` schreibt **nur** Labels, deren Key in
`LABEL_WHITELIST` (`mocks/data/customization.ts`) steht. Alles andere wird beim
Übernehmen **still verworfen** — im Editor sieht man die Umbenennung, live ist sie
weg. Genau das war die stille Falle der Spalten-Runde.

**Auftrag:** Beim Instrumentieren eines Moduls jeden `EditableText`-Key zusätzlich
in die Whitelist eintragen — und in der Dokumentation als Pflichtschritt führen.
Perspektivisch: ein Test, der jeden im Code verwendeten `dkey` gegen die Whitelist
prüft und rot wird, wenn einer fehlt.

## C3 · Kontakte-Sonderweg

**Ist:** Kontakte hat router-basierte Sub-Tabs; der Editor blockiert Navigation
(`useBlocker`), deshalb `areas: []` im Registry-Eintrag — keine Bereiche, keine
Statistik-Sektion. Helpdesk konnte alles, weil seine Tabs State sind.

**Auftrag:** entscheiden, ob (a) die Kontakte-Tabs auf State umgestellt werden
(sauber, aber Eingriff ins Modul), oder (b) die Sandbox einen eigenen Router
bekommt (MemoryRouter geht im Overlay nicht, im eigenen Fenster aber schon —
siehe Kommentarkopf in `ModuleSandbox.tsx`).

## C4 · Editor-Dokumentation (der eigentliche Hebel)

Offen seit #32. Sie ist die Vorlage, mit der jedes weitere Modul in den Editor
kommt — auch von Nico bearbeitbar. Was hinein gehört:

**1. Registry-Eintrag** (`editorModules.ts`): key, titleKey, previewPath, icon,
labelKeys, valueSetIds, fieldEntities, areas, statWidgets, listColumns, intake.

**2. Instrumentierung im Modul** — die Schritte, jeweils mit Beispiel aus Helpdesk:
- statische Überschriften → `<EditableText dkey="…" />`
- Beschriftungen in Bedienelementen → `<EditableText … interactive />`
- Chips aus Wertelisten → `useModuleValueSet` + `VsChip`
- Tabs/Abschnitte → `areas` + `useModuleAreas`
- Zusatzfelder → `useModuleCustomFields` + `useFieldOptions`
- Listenspalten → `useModuleColumnLayout` + `orderColumns` + `columnWidthStyle`
- entfernte Optionen → `useValueSetMigration`
- mutierende Aktionen → `useEditorGuard`
- **Fokus in BEIDE Richtungen**: `useEditorFocusEffect` (Leiste → Vorschau) **und**
  `useEditorContextReport` (Vorschau → Leiste). Regel für die Rückmeldung: nur
  eindeutige Orte melden, die blanke Liste meldet `null`.

**3. Die Fallen, die uns Zeit gekostet haben** (alle in Sessions #34–#36 gelernt):
- LABEL_WHITELIST (C2) — still verworfene Umbenennungen.
- `isDirty` heißt „weicht von Live ab", **nicht** „ungespeichert".
- Zwei Fenster = zwei JS-Heaps: alles Fensterübergreifende über geteilten Speicher,
  jede Mutation liest vorher frisch (`beforeMutation`).
- Prozent-Spaltenbreiten + Container ohne feste Breite = explodierende Tabelle
  (der zurückgenommene „Vorschau einpassen"-Versuch).
- Prozent-Höhen brauchen einen Elternteil mit echter Höhe (Balkendiagramm).

**4. QA-Vorlage** — die Suiten aus #36 als Muster:
`qa-editor-fokus-m.mjs` (beide Fokus-Richtungen), `qa-editor-wertelisten-n/o/p.mjs`
(Wertelisten bis ins Modul, Deploy/Rollback, eigene Liste), `qa-editor-entwurf-r.mjs`
(Entwürfe/Namen/Fenster), `qa-editor-electron-l.mjs` (Spalten).
**Regeln:** echtes Electron für alles Fenster-/IPC-Nahe · Dev-Server nach
Code-Änderungen neu starten · Suiten einzeln fahren · `innerText` sieht keine
Eingabefelder und liefert CSS-Großschreibung.

**5. Rollout-Reihenfolge** (aus MODUL-AUDIT): finanzen, inventar, einkauf,
vertraege, produktion, vermietung, formulare, work → dann kalender,
zeiterfassung, rapporte, fuhrpark. Kontakte separat (C3).

---

## Was in Session #36 fertig wurde (Kontext fürs neue Terminal)

| Commit | Inhalt |
|---|---|
| `a927b37f` | Fokus-Kopplung beidseitig (Modul meldet seinen Ort zurück) |
| `71f13dd1` | Ticket-Nummern waren doppelt (Quersumme der ID) |
| `98377fbe` | Anpassungen wirkten erst nach Besuch der Anpassungen-Seite · Umzugs-Tabelle überlebt den Deploy |
| `2e05e9ed` | Wertelisten-Optionen sortierbar (gab es nie) |
| `3f3c5767` | Eigene Werteliste verschwand nach dem Übernehmen |
| `f9796377` | Entwürfe: richtiger Stand beim Öffnen + Namen + Umbenennen |
| `115ebd6d` | Benachrichtigungen: Ton standardmäßig aus + „alles stumm" |
| `8c05d9a3` | Balkendiagramm, breiteres Editor-Fenster, deployte Umzüge sichtbar |
| `02d8c3da` | Revert: „Vorschau einpassen" (zerlegte die Spaltenbreiten) |
| `bed82797` | Spalten-Panel einzeilig |

**Entscheidungen von Darien:** B1 Spaltenbreite bleibt wie sie ist (8–80 % Grenzen
genügen) · B2 kompaktes Panel (gebaut).

**Noch offen aus der A-Runde:** nichts.
