# Spalten-Panel — nächste Runde (Darien, 2026-08-05) — ✅ GEBAUT (`e377644c`)

> **Status 2026-08-05, Session #35:** beide Punkte gebaut, QA 16/16 + Regression 9/9.
> S1 als Variante **(a) Umbenennen** umgesetzt (Darien-Entscheid), S2 mit **Ziehen am
> Spaltenrand** (Darien-Entscheid) statt Presets. Details unten, Abweichungen und
> Funde stehen im Abschluss-Block am Dateiende.


> Dariens Wortlaut am Session-Ende: *„einmal die vorhandenen Spalten kann ich nicht
> bearbeiten sondern nur die neuen und zweitens muss man noch die Reihenfolge der
> Spalten anordnen können wie man es möchte und wieviel Platz die jeweilige Spalte
> einnehmen soll."*
>
> Stand: Sektion **Spalten** existiert (`SpaltenPanel.tsx`), Sichtbarkeit läuft über
> `moduleAreas` mit `col:`-Präfix, Zusatzfeld-Spalten sind voll bearbeitbar.

---

## S1 · Eingebaute Spalten bearbeitbar machen

**Ist:** `builtIns` (Ticket-Nr, Betreff, Kategorie, Priorität, Status, Zugewiesen an,
SLA, Erstellt am) haben nur einen Sichtbar/Aus-Schalter. Nur die aus Zusatzfeldern
erzeugten Spalten öffnen den Feld-Dialog.

**Warum es so gebaut wurde:** Eine eingebaute Spalte rendert Daten, die dem Modul
gehören (`ticket.subject`, `ticket.slaDueAt` …) — es gibt kein tenant-Feld dahinter,
das man umkonfigurieren könnte. Löschen würde die Liste inhaltlich brechen.

**Vor dem Bau mit Darien klären** (eine Frage, keine Blockade — Default unten):
Was heißt „bearbeiten" hier konkret?
- **(a) Umbenennen** — technisch schon möglich: die Überschriften sind `EditableText`,
  in der Vorschau per Klick änderbar. Im Panel fehlt nur der gleiche Zugriff.
  → *Default-Annahme: das ist gemeint.* Umsetzung: Namensfeld im Panel, schreibt
  denselben Label-Override wie der Klick in der Tabelle (eine Quelle, kein zweiter Name
  — genau der Fehler, den wir bei den Wertelisten schon korrigiert haben).
- **(b) Format/Inhalt** (Datum kurz/lang, SLA als Balken statt Text …) — deutlich
  größer, braucht pro Spalte eine Optionsliste. Nur bauen, wenn Darien das meint.
- **(c) Löschen** — bewusst weiter gesperrt lassen, mit sichtbarer Begründung im Panel
  statt eines fehlenden Buttons.

## S2 · Reihenfolge + Breite

**Ziel:** Spalten frei anordnen (Drag & Drop im Panel) und ihre Breite bestimmen.

**Kernproblem — der Draft-Wert ist heute ein Boolean.**
`moduleAreas` speichert `Record<string, boolean>` und wird von **drei** Dimensionen
geteilt: echte Bereiche (`areas`), Statistik-Widgets (`stat:`) und jetzt Spalten
(`col:`). Reihenfolge/Breite passen da nicht mehr rein.

**Vorgeschlagener Weg (rückwärtskompatibel):**
1. Wert-Typ auf `boolean | { visible?: boolean; order?: number; width?: number }`
   erweitern — `resolveModuleAreas` normalisiert beim Auflösen, Konsumenten fragen
   weiter `!== false` bzw. bekommen ein Objekt.
2. `setDraftModuleArea` um eine Variante für Teil-Updates ergänzen
   (`patchDraftModuleArea(moduleKey, key, {order})`), damit ein Breiten-Zug nicht die
   Sichtbarkeit überschreibt.
3. **Nicht anfassen:** `areas` und `stat:` bleiben boolean — die Erweiterung ist additiv,
   sonst brechen Bereiche/Statistik + deren QA (`qa-editor-helpdesk-statistik.mjs`).
4. `HelpdeskPage.visibleColumns` sortiert nach `order` (fehlend → Reihenfolge aus
   `columnDefs`), Breite als `style={{width}}` am `<th>` + `table-layout: fixed`,
   sonst ignoriert der Browser die Breiten bei langen Zellinhalten.
5. Drag & Drop: prüfen, was im Repo schon benutzt wird (Formulare-Feldliste hat eine
   Sortierung — `FormularePage` `moveField`), lieber das vorhandene Muster
   wiederverwenden als eine neue Bibliothek ziehen.

**QA-Erweiterung:** `qa-editor-spalten-h.mjs` um zwei Fälle ergänzen —
(1) Reihenfolge ändern → `thead`-Reihenfolge folgt, (2) Breite setzen → `<th>` trägt sie.

---

---

## Abschluss (Session #35, Commit `e377644c`)

**Gebaut wie geplant:** `ModuleAreaSetting = boolean | {visible,order,width}`,
`resolveModuleAreas` normalisiert weiter auf boolean (areas/`stat:` unberührt), neu
`resolveModuleAreaLayout`; `patchDraftModuleArea` + `setDraftModuleAreaOrder`;
Panel-Liste sortierbar über dnd-kit (dasselbe Muster wie die Formulare-Feldliste).

**Drei Funde, die im Plan nicht standen:**

1. **Umbenennen überlebte den Deploy nicht.** `LABEL_WHITELIST` ist der Deploy-Filter,
   und `helpdesk.table.*` stand nicht drin — der Name lebte nur im Sandbox-Bundle.
   Die acht Spalten-Keys sind jetzt eingetragen. **Merkposten für den Rollout:** jedes
   Modul, das Spalten bekommt, muss seine `listColumns[].labelKey` dort nachtragen.
2. **`status` hing an `common.status`.** Umbenennen hätte jedes Status-Label der App
   mitgezogen → eigener Key `helpdesk.table.status`. Gleiche Prüfung bei jedem Modul:
   eine Spaltenüberschrift darf keinen geteilten `common.*`-Key benutzen.
3. **Layout-Einträge durften nicht als „sichtbar" gelten.** Das Sortieren schreibt für
   JEDE Spalte ein Objekt; ein Objekt ohne `visible` wurde zu `true` normalisiert und
   hat die Opt-in-Spalten (die `=== true` fragen) von selbst eingeschaltet — im ersten
   QA-Lauf sofort sichtbar (8 → 11 Spalten). Jetzt bleiben Layout-only-Einträge aus der
   Sichtbarkeits-Map heraus.

**Zwei UX-Entscheidungen beim Bauen:**

- **Breiten einfrieren beim ersten Zug.** `table-layout: fixed` verteilt alle Spalten
  ohne Breite gleichmäßig — die Liste wäre beim ersten Pixel zusammengeklappt. Der
  erste Zug misst deshalb die Ist-Breiten und schreibt sie mit (ein Dispatch, ein
  Undo-Schritt zusammen mit dem Zug).
- **Zähler.** Reihenfolge und Breiten zählen als je EINE Änderung, nicht als eine pro
  Spalte — ein Zug stand sonst als „11 Änderungen" im Footer.

**QA:** `qa-editor-spalten-i.mjs` 16/16 inkl. Deploy-Durchstich (Name, Reihenfolge,
Breite und die ausgeblendete Spalte im echten Modul), `qa-editor-spalten-h.mjs` 9/9,
Statistik-Suite grün. **Lehre fürs nächste QA-Skript:** der Sichtbarkeits-Schalter ist
`role="switch"`, nicht `button` — `getByRole('button')` läuft dort in den Timeout.

---

## Nicht vergessen
- Beides ist Draft-Ebene: wirkt sofort in der Sandbox, im echten Modul erst nach
  „Übernehmen" — dieselbe Regel, die Darien für die Statistik gesetzt hat.
- Nach dem Bau: `.planning/RESUME-NEXT.md` Top-Block fortschreiben.
- Danach steht weiterhin die **Editor-Dokumentation** als Rollout-Vorlage für die
  anderen Module an (seit Session #32 offen) — die Spalten-Dimension gehört dann mit
  hinein, inklusive `useEditorFocusEffect` und `useEntityFieldDraft`.
