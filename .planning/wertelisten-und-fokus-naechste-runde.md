# Nächste Runde (Darien, 2026-08-06): Wertelisten + Fokus-Kopplung

> Dariens Wortlaut am Session-Ende 05.08.: *„als nächstes müssen wir uns an die
> Wertelisten setzen, die gehen noch nicht zu 100 Prozent. Und wenn ich im
> Modul-Editor im Modul auf Statistik klicke, wechselt das Menü links und rechts
> nicht."*

---

## W · Wertelisten „gehen nicht zu 100 %"

**Was zuerst passieren muss:** Darien fragen, welche Stellen konkret hakten. Die
Aussage ist bewusst grob — nicht raten, sondern die Symptome einsammeln und dann
gezielt reparieren. Bis dahin unten die Prüfliste, mit der ich das Modul ohnehin
systematisch durchgehe.

**Prüfliste (jede Zeile = ein Durchstich bis ins Modul UND bis nach „Übernehmen"):**

| # | Fall | Erwartung |
|---|---|---|
| W1 | Neue Werteliste anlegen | erscheint im Panel, im Spalten-Block „ohne Spalte", im Statistik-Katalog |
| W2 | Option umbenennen | wirkt in Datensatz, Liste, Chips, Statistik-Aufschlüsselung |
| W3 | Option-Farbe ändern | Chip-Farbe folgt überall (Liste + Detail + Statistik) |
| W4 | Option hinzufügen | sofort in allen gebundenen Auswahlfeldern wählbar |
| W5 | Option deaktivieren | verschwindet aus der Auswahl, bestehende Datensätze behalten den Wert |
| W6 | Option löschen (in Benutzung) | Umzugs-Dialog (`valueSetMigrations`), Vorschau remappt live, Deploy zieht die Datensätze mit |
| W7 | Liste an ein Feld binden | `valueSetId` + `useFieldOptions` — Labels/Farben aus der Liste |
| W8 | Liste umbenennen | Listenname ≠ Spaltenüberschrift (zwei Speicher, in R1 geklärt) — beides muss stimmen |
| W9 | Reihenfolge der Optionen | Sortierung wirkt in Auswahl + Statistik |
| W10 | Übernehmen | alles oben überlebt den Deploy (die Whitelist-Falle der Spalten prüfen: gilt sie auch für Value-Sets?) |
| W11 | Zurückrollen | Liste kehrt auf den Vorzustand zurück, Datensätze bleiben heil |

**Bekannte Verdachtsmomente aus dem Code (vor der Reparatur verifizieren):**

- **Vordefinierte vs. selbst angelegte Listen.** Die Kachel-/Panel-Logik kennt
  `EditorModuleDef.valueSetIds` als „gehört zum Modul". Eine im Editor NEU angelegte
  Liste hat keine Modul-Zuordnung — sie wird nur über ein gebundenes Feld sichtbar
  (so zählt sie seit R7 auch in den Kachel-Zahlen). Löst der Kunde die Bindung,
  ist die Liste faktisch heimatlos. **Braucht wahrscheinlich ein `moduleKey` am
  Value-Set.**
- **Migrations-Pfad W6** ist der komplexeste Teil und bisher am wenigsten
  hands-on geprüft (`valueSetMigrations` + Deploy-Anwendung).
- **Deploy-Filter:** bei Labels war `LABEL_WHITELIST` die stille Falle (Spalten-
  Runde). Für Value-Sets denselben Pfad einmal explizit durchspielen.

---

## F · Fokus-Kopplung ist einseitig

**Symptom:** Im Editor auf den **Statistik-Reiter im Modul** klicken → linke Leiste
und rechtes Panel bleiben stehen, wo sie waren.

**Warum:** `useEditorFocusEffect` (EditorSurface) koppelt nur **Leiste → Vorschau**:
die Leiste setzt `focusSection`, das Modul reagiert (`statistik: () => setTab('statistik')`).
Die Gegenrichtung existiert nicht — das Modul meldet seinen Kontext nirgends.

**Vorgeschlagener Weg:**

1. `EditorSurfaceValue` um `reportContext(section: EditorFocusSection | null)`
   erweitern (no-op außerhalb der Sandbox, wie `setLabel`/`setAreaLayout`).
2. Modul meldet beim Tab-Wechsel: `useEditorContextReport(tab === 'statistik' ? 'statistik' : null)`
   — ein kleiner Hook, der nur bei Änderung meldet.
3. `EditorWorkspace` setzt daraufhin `activeSection` — **ohne** `focusNonce` zu
   erhöhen, sonst schiebt die Leiste die Vorschau zurück und es entsteht eine
   Schleife (Leiste → Vorschau → Leiste → …). Guard: nur setzen, wenn `!==` aktuelle
   Section.
4. Rollout-Muster: dieselben drei Zeilen wie `useEditorFocusEffect` in jedem Modul —
   gehört mit in die Editor-Dokumentation.

**QA:** in der Electron-Suite (`qa-editor-electron-l.mjs`) oder einer neuen:
Modul-Tab „Statistik" klicken → Leiste zeigt „Statistik" aktiv, rechtes Panel zeigt
den Statistik-Katalog. Und die Gegenprobe: Leiste klicken → Vorschau folgt weiterhin
(keine Regression an R1).

---

## Reihenfolge morgen

1. Symptome der Wertelisten von Darien einsammeln → W-Prüfliste abarbeiten.
2. F (Fokus-Kopplung) — kleiner, klar umrissen, kann dazwischen laufen.
3. Danach steht weiterhin die **Editor-Dokumentation** als Rollout-Vorlage an
   (offen seit #32); Spalten-, Fokus- und Wertelisten-Erkenntnisse gehören hinein.
