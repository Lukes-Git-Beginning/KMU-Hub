# Editor-Rollout — die übrigen Module (Paket D)

> Übergabe aus Session #37 (2026-08-10). Paket C ist abgearbeitet, die Vorlage steht.
> Darien: *„ne rollout machen wir erst morgen"*.

## Vor dem ersten Handgriff

**`docs/EDITOR-MODULE-ROLLOUT.md` lesen, Abschnitt 0 zuerst.** Dort steht, welche der
sieben Dimensionen für ein Modul fällig ist, in welcher Reihenfolge gearbeitet wird
und wann ein Modul als fertig gilt. Die Doku ist die Single Source of Truth — was hier
steht, ist nur der Fahrplan.

Nicht mit der Instrumentierung anfangen. Erst **Zuschnitt** (welche Dimensionen
treffen zu, Begründung für die, die nicht zutreffen), dann Registry, dann bauen.

## Reihenfolge

**Welle 1:** finanzen, inventar, einkauf, vertraege, produktion, vermietung,
formulare, work
**Welle 2:** kalender, zeiterfassung, rapporte, fuhrpark

**Erstes Modul: `finanzen`.** Es ist das dichteste der Welle-1-Module (Belege,
Status-Enums, Tabellen mit vielen Spalten, Auswertungen) — was dort trägt, trägt
überall. Gleichzeitig ist es das Modul mit der meisten Demo-Tiefe (P2.5 ist die
Referenz für funktionale Tiefe), also fällt am ehesten auf, wenn eine Dimension fehlt.

## Was pro Modul zu klären ist

Die zwei Fragen, die den Zuschnitt entscheiden — beide beantwortbar, ohne Code zu
schreiben:

1. **Hat das Modul eine eigene Bereichs-Leiste?**
   - Zustandsgeführte Tabs → wie Helpdesk, nichts weiter zu tun.
   - Router-geführte Navigation → **Kontakte-Muster** (Doku Abschnitt 6): Zustand
     führt, Routen bleiben Einstieg, Präfix-Match, Detail-Seiten im `<Outlet/>`.
   - Keine Leiste → registriere die **Seite selbst**, nicht das Layout (`work`).
2. **Welche Enums rendert es als Chips?** Das werden die Wertelisten. Ohne ein Feld,
   das auf eine Liste zeigt, hat eine neu erstellte Liste keinen Ort im Modul.

## Die zwei Prüfungen, die tatsächlich etwas fangen

Beide haben in #37 je einen Fehler gefunden, den grüne Prüfungen durchgelassen hätten:

- **Eine Umbenennung nach dem Übernehmen im ECHTEN Modul nachweisen**, nicht nur in
  der Vorschau. Die Vorschau lügt, wenn ein Key nicht in `LABEL_WHITELIST` steht.
  (Der Wächter-Test fängt das inzwischen vorher ab — aber die Suite gehört trotzdem
  dazu, denn er prüft nur die Liste, nicht den Weg.)
- **Die Screenshots ansehen.** „11/11 grün" bewies nicht, dass auf der
  Firmen-Detailseite der richtige Bereich markiert war.

## Offene Punkte, die während des Rollouts mitlaufen

- **`statWidgets` mit `locked: true`** — Kacheln für noch nicht gebaute Features. Beim
  Zuschnitt bewusst entscheiden, ob ein Modul solche bekommt (Katalog zeigt sie grau),
  statt sie wegzulassen und später zu vergessen.
- **`intake`** hat bisher nur Helpdesk. Beim Zuschnitt prüfen, ob ein Modul Datensätze
  von außen entgegennimmt (formulare ist der nächste Kandidat).
- **Kontakte-Spalten**: Kontakte hat noch keine `listColumns` im Registry-Eintrag. Die
  Bereiche sind seit #37 da, die Spalten-Dimension ist dort noch offen — beim
  Durchgang durch Welle 1 nachziehen.
