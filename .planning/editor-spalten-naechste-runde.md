# Spalten-Panel — nächste Runde (Darien, 2026-08-05)

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

## Nicht vergessen
- Beides ist Draft-Ebene: wirkt sofort in der Sandbox, im echten Modul erst nach
  „Übernehmen" — dieselbe Regel, die Darien für die Statistik gesetzt hat.
- Nach dem Bau: `.planning/RESUME-NEXT.md` Top-Block fortschreiben.
- Danach steht weiterhin die **Editor-Dokumentation** als Rollout-Vorlage für die
  anderen Module an (seit Session #32 offen) — die Spalten-Dimension gehört dann mit
  hinein, inklusive `useEditorFocusEffect` und `useEntityFieldDraft`.
