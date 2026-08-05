# Intake-Pilot — Lokale Review-Checkliste (Darien, 2026-07-30)

> Cosmi läuft im Dev-Mode (`npm run dev`, demo). Alles unten ist **FE-mock-first** — echte
> Persistenz/öffentliche Route/CSAT-Mail kommt später von Luke. Feedback pro Punkt in die
> Notiz-Spalte; ich ziehe daraus danach neue Phasen (TaskCreate pro Item, sequentiell).
>
> Status-Legende: [ ] offen · [x] passt · [!] Feedback (siehe Notiz)

---

# TEIL 1 — Intake v2 (dein Feedback Punkt 4/5/6, frisch gebaut)

## A · Pro-Kanal-Vorlagen (Punkt 4)
**Pfad:** Helpdesk → Editor öffnen → Sektion **Kanäle**.

- [ ] Jeder der 3 Kanäle hat einen **eigenen** Vorlagen-Wähler (nicht mehr ein globales Formular für alle)
- [ ] Man kann 1, 2 oder alle 3 Kanäle mit unterschiedlichen Vorlagen belegen
- [ ] „Neue Vorlage" legt eine Kopie an, die man sofort zuweisen kann
- [ ] „Formular bearbeiten →" öffnet **direkt dieses Formular** im Editor (nicht nur die Liste)
- [ ] Die zugewiesenen Formulare sind **wirklich bearbeitbar** (nicht mehr die read-only Vorlage von vorher)
- Notiz:

## B · Agent-Dialog aus der Vorlage (Punkt 4)
**Pfad:** Helpdesk → „Neues Ticket".

- [ ] Der Dialog zeigt die Felder der **gebundenen Agent-Vorlage** (ändert man die Vorlage, ändert sich der Dialog)
- [ ] Die Profi-Werkzeuge sind **erhalten geblieben**: Sektion „Interne Zuordnung (nur Agent)" mit Kontakt / Priorität / Zuweisung
- [ ] Absenden erzeugt ein Ticket, in dem sowohl Vorlage-Felder als auch die interne Zuordnung korrekt stehen
- Notiz:

## C · Feld-Trennung intern vs. Einreicher (Punkt 6)
**Pfad:** Editor → **Zusatzfelder** · und Helpdesk → „Neues Ticket" + Ticket-Detail.

- [ ] Modul-Zusatzfelder sind **raus** aus dem Neu-Dialog (die sind intern)
- [ ] Im **Ticket-Detail** sind sie weiterhin da und editierbar
- [ ] Editor-Panel „Zusatzfelder" erklärt sichtbar: *diese Felder sind intern, nur für Modul-Mitarbeiter*
- [ ] Was der Einreicher ausfüllt, kommt aus der **Kanal-Vorlage** — die Trennung ist beim Durchklicken verständlich
- Notiz:

## D · Dynamische Statistik + CSAT-Frage (Punkt 5)
**Pfad:** Helpdesk → Statistik-Tab · und Moduleinstellungen.

- [ ] Statistik zeigt für **jedes Auswahl-Zusatzfeld** automatisch eine Aufschlüsselung („Nach SLA-Stufe", „Nach Kontaktkanal" …)
- [ ] Neue Werteliste im Editor anlegen → erscheint **ohne Zutun** als neue Aufschlüsselung
- [ ] Die Namen in der Aufschlüsselung kommen aus der Werteliste (umbenennen wirkt durch)
- [ ] Moduleinstellungen → CSAT: **„Umfrage Frage"** ist frei änderbar
- [ ] Geänderte Frage steht im CSAT-Widget über den Sternen
- Notiz:

> **Bewusst offen (B2-Rest):** die Feld-Aufschlüsselungen sind aktuell **immer** sichtbar. Sie im
> Editor-Statistik-Katalog einzeln an-/abschaltbar zu machen braucht noch einen Umbau am Panel-Kontext.
> Sag Bescheid, ob dir „immer sichtbar" reicht oder ob das nachgezogen werden soll.

---

# TEIL 2 — Pilot-Grundlage (Session #32, falls du nochmal drüber willst)

## 1 · Herkunfts-Reiter im Helpdesk (P4)
**Pfad:** Sidebar → Helpdesk → oben die Reiter **Alle / Agent / Selfservice / Extern**
(erscheinen nur, wenn ≥2 Kanäle aktiv sind).

- [ ] Reiter filtern die Ticketliste korrekt nach Herkunft
- [ ] „zusammenführen"-Option vorhanden und verständlich
- [ ] Seed-Tickets sind sinnvoll über die Kanäle verteilt (nicht alle in einem)
- Notiz:

## 2 · Editor „Kanäle"-Panel (P6)
**Pfad:** Helpdesk → Editor öffnen (eigenes Fenster) → Sektion **Kanäle**.

- [ ] 3 Kanal-Toggles (Agent / Selfservice / Extern) schalten sichtbar
- [ ] Formular-Wähler bindet ein Formular an den Intake
- [ ] „Formular bearbeiten →" führt in den Formular-Editor
- [ ] Extern-Toggle zeigt den Hinweis „Route noch nicht live" (P5 = Luke)
- [ ] Sandbox/Vorschau reagiert live auf die Toggles
- Notiz:

## 3 · Selfservice-Einstieg (P4)
**Pfad:** Settings → **Über Cosmi** → IT-Support → **Support-Ticket erstellen**.

- [ ] Gebundenes Formular wird gerendert
- [ ] Requester = dein eingeloggtes Profil (Name/E-Mail automatisch, keine Eingabe nötig)
- [ ] Absenden → „Vielen Dank / Dein Ticket …"
- [ ] Neues Ticket taucht im Helpdesk unter Reiter **Selfservice** auf
- Notiz:

## 4 · Formular → Ticket, shared Engine (P2)
**Pfad:** Formulare-Modul → Formular „Support-Ticket (Helpdesk)" im Editor öffnen.

- [ ] Panel **„Bei Einreichung"**: Ziel-Wähler + Feld-Zuordnungs-Checkliste (6 Rollen grün)
- [ ] Feld-Config-Dialog hat Dropdown **„Feld-Rolle im Ziel"** (Betreff/Beschreibung/Priorität/Kategorie/Kontakt)
- [ ] Unmarkierte Felder = Zusatzfelder (landen in custom_fields)
- [ ] Interaktive Vorschau **dispatcht wirklich** → „Helpdesk-Ticket erstellt · Referenz …"
- [ ] Das so erzeugte Ticket steht im Helpdesk mit korrekt gemappten Feldern
- Notiz:

## 5 · CSAT / Kundenzufriedenheit (P3)
**Pfad A:** Helpdesk → Statistik-Tab. **Pfad B:** Helpdesk → Moduleinstellungen → CSAT-Sektion.

- [ ] Statistik zeigt CSAT-Kachel + Verteilungs-Chart (Ø ~4.0/5 aus 3 Seeds)
- [ ] Settings-Sektion „Kundenzufriedenheit (CSAT)": An/Aus + Verzögerung (Sofort … 3 Tage)
- [ ] Toggle **aus** → Kachel + Chart verschwinden sofort
- [ ] CSAT-Widget am geschlossenen Ticket (Sterne abgebbar)
- Notiz:

## 6 · Ticket-Durchreichung & Zusatzfelder (P1)
**Pfad:** Helpdesk → „Neues Ticket" + ein bestehendes Ticket öffnen.

- [ ] Neu-Dialog: Beschreibung + Kontakt + Kategorie + Zusatzfelder landen am erstellten Ticket
- [ ] Zusatzfelder im Detail sind **echte Controls** (Dropdown/Checkbox/Input), nicht „—"
- [ ] Feld-Edit → weg-navigieren → zurück → Wert bleibt erhalten (persistiert am Wire)
- Notiz:

---

## Gesamteindruck / Querschnitt
- [ ] i18n: keine Raw-Keys, keine `{{doppelten}}` Klammern
- [ ] Keine Emojis in der UI
- [ ] Zurück-/Close-Buttons überall sichtbar (sticky), Detail als zentriertes Modal
- [ ] Leere Zustände sehen gut aus (nicht „leerer Screen")
- Freitext-Feedback:
