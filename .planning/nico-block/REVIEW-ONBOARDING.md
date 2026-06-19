# Nico-Review-Onboarding

> **Für Darien:** Schick Nico den Block unter „═══ KOPIERFERTIGER TEXT ═══" (alles ab dort bis zum Ende). Sie fügt ihn in ihrem VS-Code-Claude im eigenen Repo-Klon ein. Der Text ist self-contained — ihr Claude liest sich den Rest (Checklisten in `.planning/reviews/<modul>.md`, `CLAUDE.md`) selbst.
>
> **Was Nico reviewt:** 6 fertige, review-reife Module — **team, dashboard, vertraege, helpdesk, automatisierung, profil**. Alle FE-mock-first. Sie prüft FE/UX gegen die Cosmi-Standards, **nicht** Lukes Backend. Tempo ~2–3 Tage/Modul, ein Modul nach dem anderen. Befunde landen in `.planning/reviews/<modul>.md` + kurze Meldung an dich.

---

═══ KOPIERFERTIGER TEXT (ab hier an Nico) ═══

# Dein Auftrag: 6 Cosmi-Module reviewen (FE/UX)

Hi! Du bist **FE/UX-Reviewer** für das Cosmi-CRM. Sechs Module sind fertig gebaut und „review-reif" — du klickst sie durch, prüfst sie gegen unsere Qualitäts-Standards und schreibst Befunde auf. Du **baust nichts** und **fixt nichts** — du findest und dokumentierst. Qualität vor Tempo, es gibt kein Zeitlimit.

## Wichtig vorweg: Was du NICHT als Mangel wertest
Die Module laufen im **Demo-Modus ohne echtes Backend** (ein Mock-Server namens MSW liefert die Daten). Das ist **so gewollt**. Also:
- **Fehlende echte Backend-Anbindung ist KEIN Befund.** Avatar-Upload, der nach echtem Server-Reload weg wäre; Verträge, die nicht in einer echten DB landen; Automatisierungen, die nicht echt „feuern" — alles bewusst gemockt. Nicht melden.
- Was du **schon** wertest: tote Buttons, leere/kaputte Screens, Layout-Brüche, fehlende Übersetzungen, alles unter „Wogegen du prüfst".
- Jede Modul-Checkliste in `.planning/reviews/<modul>.md` hat einen Block **„Out of scope (kein Mangel)"** — lies den zuerst, dann weißt du, was beim jeweiligen Modul bewusst gemockt ist.

## Einmal-Setup (ca. 20–30 Min)
1. **Repo klonen** (eigener Klon, Branch `main`), dann immer zuerst aktuell ziehen:
   ```bash
   git checkout main && git pull
   ```
2. **Claude Code** im Repo-Root starten (`claude`).
3. **App starten** (im Ordner `desktop/`):
   ```bash
   npm install
   npm run dev      # Vite auf http://localhost:5173 — Fenster offen lassen
   ```
   Im Browser `http://localhost:5173` öffnen → App im Demo-Modus.
4. **Einmal lesen** zur Orientierung: `CLAUDE.md` (Repo-Root, Abschnitt „UI/UX") + die Checkliste des Moduls, das du gerade reviewst (`.planning/reviews/<modul>.md`).

**Claude-Code-Skills, die dir beim Review helfen** (du musst sie nicht selbst aufrufen — sag deinem Claude in Worten was du willst, es wählt sie):
- **`/run`** — startet/zeigt die App, navigiert zu einem Screen.
- **`/critique`** und **`/audit`** — UX-/Design-Review eines Screens (Hierarchie, Abstände, Konsistenz, Anti-Patterns).
- **`/verify`** — prüft, ob eine Funktion wirklich tut, was sie soll (nicht nur „lädt").

## Wogegen du prüfst (die Cosmi-Demo-Tiefe-Standards)
Für **jedes** Modul, durch **alle** Tabs/Ansichten, **in voller Fensterbreite** (1440px) **und** schmaler (~1024px):

**Detail-Ansichten & Interaktion**
- [ ] Klick auf eine Listen-/Tabellenzeile öffnet eine **echte Detail-Ansicht** (kein toter Klick).
- [ ] Das Detail ist ein **zentriertes Modal-Fenster** (nicht von rechts reingeschoben) mit allen Infos + Funktionen drin.
- [ ] Der **Schließen-Button ist immer sichtbar** (sticky) — verschwindet nicht beim Scrollen, überlappt nichts.
- [ ] Die **ganze Zeile** ist klickbar (nicht nur der Titel oder das Drei-Punkte-Menü). Innere Buttons (Bearbeiten, Menü) öffnen NICHT das Detail.
- [ ] Tastatur: Zeile per Tab fokussierbar, Enter/Space öffnet, Escape schließt das Modal.

**Tote Stellen & leere Zustände**
- [ ] **Keine toten Buttons** — alles, was klickbar aussieht, tut sichtbar etwas (kein „Toast und sonst nichts", kein Button ohne Wirkung).
- [ ] **Keine leeren/kaputten Screens** — auch leere Listen haben einen sinnvollen Leer-Zustand, nicht nur weiße Fläche.
- [ ] **Downloads/Exporte** laden wirklich eine Datei (kein bloßer Toast).

**Sprache / i18n** (kritisch — häufigste Fehlerquelle)
- [ ] **Keine „Raw-Keys"**: nirgends taucht ein Schlüssel wie `team.page.title` oder `helpdesk.sla.left` als sichtbarer Text auf.
- [ ] **Keine doppelten Klammern** im Text (`{{name}}` statt eingesetztem Wert) — der Wert muss eingesetzt sein.
- [ ] **EN umschalten** und nochmal durchklicken: alle Texte englisch, keine deutschen Reste, keine Raw-Keys. (Sprache wechselt über den Sprach-Umschalter in der App.)
- [ ] Deutsche Texte: echte Umlaute („für", „löschen" — nie „fuer"/„loeschen"), **keine Emojis**.

**Projektweite UX-Muster**
- [ ] **Sortierung** bietet Feld **und** Richtung (auf/ab), nicht nur ein einzelnes Sortierkriterium.
- [ ] **Zurück-/Schließen-Weg** aus jeder Detail-/Unteransicht vorhanden.
- [ ] **Modul-Einstellungen**: das Modul hat einen Eintrag im Einstellungs-Fenster mit **persönlichem** und **tenant**-Bereich (sofern in der Modul-Checkliste vermerkt).
- [ ] Allgemeine Optik: keine „Karte-in-Karte", keine abgeschnittenen Texte, konsistente Abstände, keine lila AI-Gradienten.

## Befund-Format (so dokumentierst du)
Trag jeden Befund in `.planning/reviews/<modul>.md` unter „Befunde" ein, eine Zeile pro Fund:

> **Schweregrad** · **was** · **wo** · **Repro**

- **Schweregrad:** **P0** = blockt (kaputt/leer/toter Kern-Button) · **P1** = sollte vor Launch weg (Raw-Key, fehlende EN-Übersetzung, UX-Bruch) · **P2** = nice-to-have (Politur, Abstand, Wording).
- **was:** ein Satz, was falsch ist.
- **wo:** Screen/Tab + wenn möglich Datei (dein Claude findet die Datei für dich).
- **Repro:** die Klicks, um es zu sehen.

Beispiel:
`P1 · „Mitglied seit" zeigt Rohschlüssel statt Datum · /profil → Tab Profil → Account-Info · Profil öffnen, EN umschalten`

## Reihenfolge & Ablauf
Empfohlene Reihenfolge (erst ein kleines Modul zum Einarbeiten, dann die größeren — du kannst aber auch anders sortieren):

1. **profil** — überschaubar (4 Tabs), guter Einstieg in den Workflow.
2. **dashboard** — Widgets, Persistenz, Team-Umschalter.
3. **team** — HR, 6 Tabs (Abwesenheiten, Self-Service, Personalakte, Organigramm, Schulungen).
4. **helpdesk** — Tickets, viel Interaktion (Detail-Modal, Zuweisen/Eskalieren/Mergen, Sortierung).
5. **vertraege** — Verträge, E-Signatur-Demo-Flow, Fristen.
6. **automatisierung** — Automationen, Vorlagen, Protokoll, visueller Editor.

**Pro Modul:**
1. `git checkout main && git pull` (immer aktuell — wir pushen oft).
2. Öffne `.planning/reviews/<modul>.md` — dort steht, was gebaut wurde, worauf besonders zu achten ist, und was bewusst out-of-scope (kein Mangel) ist.
3. Klick das Modul **vollständig** durch — alle Tabs, beide Breiten, DE **und** EN, mit Daten + leeren Zuständen.
4. Trag Befunde in dieselbe Datei ein (Block „Befunde").
5. **Melde Darien**: „Modul X durch — N Befunde (P0/P1/P2-Verteilung), Liste in `.planning/reviews/X.md`." Dann nächstes Modul.

**Bei Unsicherheit** (ist das ein Bug oder Absicht? fehlt da Backend?): nicht raten — kurz Darien fragen. Lieber einmal fragen als einen Fehlbefund schreiben.

Viel Erfolg — gründlich schlägt schnell. Wenn ein Screen sich „halb fertig" anfühlt, vertrau dem Gefühl und schreib es auf.
