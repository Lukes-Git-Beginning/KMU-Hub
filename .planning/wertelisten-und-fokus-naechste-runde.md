# Wertelisten + Fokus-Kopplung — Stand 2026-08-09 (abgearbeitet)

> Dariens Wortlaut am Session-Ende 05.08.: *„als nächstes müssen wir uns an die
> Wertelisten setzen, die gehen noch nicht zu 100 Prozent. Und wenn ich im
> Modul-Editor im Modul auf Statistik klicke, wechselt das Menü links und rechts
> nicht."*
>
> **09.08.:** Darien wollte die konkreten Werteliste-Symptome noch nachschauen —
> bis dahin habe ich die Prüfliste W1–W11 selbst durchgespielt (echte Electron-QA,
> nicht Browser). Dabei sind **vier echte Defekte** aufgefallen und behoben.
> Was Darien konkret meinte, kann trotzdem noch etwas anderes sein → siehe
> „Was noch offen ist".

---

## F · Fokus-Kopplung — erledigt (Commit a927b37f)

Das Modul meldet jetzt zurück, wo der Nutzer steht; Leiste und rechtes Panel
folgen. Klick auf den Statistik-Reiter im Modul → Leiste steht auf „Statistik",
rechts der Statistik-Katalog. Ticket öffnen → „Zusatzfelder".

**Zwei Regeln, die das stabil halten:**

1. Nur eindeutige Orte melden. Die blanke Liste ist gleichzeitig Heimat von
   Begriffen, Wertelisten, Bereichen und Spalten → sie meldet `null` und lässt
   die Leiste in Ruhe. Nur ein Ort mit eigenem Reiter/Detail setzt die Auswahl.
2. `useEditorFocusEffect` hängt jetzt **nur** am `focusNonce`, nicht mehr an der
   Section. Sonst hätte die Rückmeldung den Fokus-Handler erneut ausgelöst — beim
   Öffnen eines Tickets wäre die Vorschau auf das *erste* Ticket zurückgesprungen.

**Rollout in weitere Module:** eine Zeile, analog zu `useEditorFocusEffect`:

```tsx
useEditorContextReport(tab === 'statistik' ? 'statistik' : detailOffen ? 'felder' : null)
```

QA: `desktop/scripts/qa-editor-fokus-m.mjs` — 12/12, echtes Electron-Fenster.

---

## W · Wertelisten — vier Defekte gefunden und behoben

| # | Fall | Ergebnis |
|---|---|---|
| W1 | Neue Liste anlegen | ✅ — **war kaputt:** nach dem Übernehmen fiel die Liste aus dem Panel (gehörte zu keinem Modul). Listen tragen jetzt ihr Modul (Commit 3f3c5767) |
| W2 | Option umbenennen | ✅ Liste, Chips, Filter, Statistik |
| W3 | Option-Farbe ändern | ✅ bis in den Chip der Tabelle |
| W4 | Option hinzufügen | ✅ sofort in allen gebundenen Auswahlfeldern |
| W5 | Option deaktivieren | ✅ raus aus der Auswahl, Datensätze behalten den Wert |
| W6 | Option löschen (in Benutzung) | ✅ Vorschau — **aber der Deploy zog nicht mit** (siehe unten, Commit 98377fbe) |
| W7 | Liste an ein Feld binden | ✅ Labels/Farben/Reihenfolge kommen aus der Liste |
| W8 | Liste umbenennen | ✅ Listenname und Spaltenüberschrift bleiben getrennt (R1) |
| W9 | Reihenfolge der Optionen | ✅ — **gab es gar nicht:** `order` wurde sortiert, war aber nicht änderbar. Jetzt Zieh-Griffe wie bei den Spalten (Commit 2e05e9ed) |
| W10 | Übernehmen | ✅ — **war kaputt:** siehe Hydration unten |
| W11 | Zurückrollen | ✅ inkl. der Umzüge |

### Die zwei schwerwiegenden Funde (beide Commit 98377fbe)

**1. Anpassungen wurden erst „lebendig", wenn jemand die Anpassungen-Seite öffnete.**
Die Hydration hing am Lesen der Entwurfs-Liste — und die liest nur diese eine
Seite. App starten, direkt ins Helpdesk: Original-Wertelisten. Ein Rollout, der
seit Wochen live war, sah aus, als hätte es ihn nie gegeben — und kam zurück,
sobald man einmal auf Anpassungen war. Jetzt beim Start, in jedem Fenster.

**2. „Bestehende Einträge werden geändert auf: X" galt nur in der Vorschau.**
Die Umzugs-Tabelle lebte im Editor. Nach dem Übernehmen fielen die Datensätze
auf die entfernte Option zurück. Die Tabelle ist jetzt Teil der Mandanten-Ebene:
sie wird deployt, rollt mit dem Snapshot zurück, und Ketten werden aufgelöst
(Mittel→Hoch, dann Hoch→Dringend ⇒ Mittel zeigt auf Dringend, nie auf etwas
Entferntes).

QA: `qa-editor-wertelisten-n.mjs` 16/16 (jede Änderung bis ins Modul),
`qa-editor-wertelisten-o.mjs` 7/7 (übernehmen → echtes Modul → zurückrollen),
`qa-editor-wertelisten-p.mjs` 3/3 (eigene Liste überlebt den Deploy).

### Nebenbei mitgenommen

- Optionszeilen haben Namen bekommen (sechs identische „Sichtbarkeit umschalten"-
  Buttons und sechs namenlose Eingabefelder pro Liste), Farbfelder sagen ihre Farbe.
- `fix(helpdesk)` 71f13dd1: Ticket-Nummern waren eine Quersumme der ID — fünf von
  vierzehn Demo-Tickets teilten sich eine Nummer mit einem anderen. Und eine
  Nummer bricht nicht mehr dreizeilig um.

---

## Was noch offen ist

1. **Dariens eigentliche Symptome.** Er wollte nachschauen, wo es hakte. Die vier
   Funde oben decken die Prüfliste ab — sie müssen nicht dasselbe sein.
2. **Panel zeigt bereits deployte Umzüge nicht.** Öffnet man den Editor erneut,
   steht die entfernte Option nur als „ausgeblendet" da, ohne „→ Ziel" und ohne
   „Wiederherstellen". Dafür bräuchte der Entwurf ein „Umzug aufheben".
3. **Vorschau schneidet die Liste rechts ab.** Im Editor-Canvas sind „Zugewiesen
   an", SLA und „Erstellt am" nicht sichtbar — gerade beim Konfigurieren von
   Spalten unpraktisch.
4. **Editor-Dokumentation** als Rollout-Vorlage (offen seit #32): Spalten, Fokus
   (beide Richtungen!), Wertelisten gehören hinein.

## QA-Regeln, die diese Runde bestätigt/erweitert hat

- **Alles Fenster-/IPC-Nahe mit echtem Electron testen** (`_electron.launch`) —
  Browser-Suiten sehen das zweite Fenster nicht.
- **Nach Code-Änderungen den Dev-Server neu starten, bevor die Electron-QA läuft.**
  Zweimal in dieser Runde ist der Editor mit „useDraftConfig must be used within a
  DraftConfigProvider" gestorben — reiner HMR-Zustand, kein echter Fehler, kostet
  aber jedes Mal einen Debug-Umweg.
- **Die Suiten einzeln fahren.** Drei Electron-Starts hintereinander in einem
  Befehl haben O reproduzierbar wacklig gemacht (einzeln 7/7, im Block 4/7).
- **`innerText` sieht keine Eingabefelder und liefert CSS-Großschreibung** — zwei
  falsche FAILs kamen daher; Werte über `evaluateAll(el => el.value)` prüfen.
