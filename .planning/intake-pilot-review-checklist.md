# Intake-Pilot — Lokale Review-Checkliste (Darien, 2026-07-30)

> Cosmi läuft im Dev-Mode (`npm run dev`, demo). Alles unten ist **FE-mock-first** — echte
> Persistenz/öffentliche Route/CSAT-Mail kommt später von Luke. Feedback pro Punkt in die
> Notiz-Spalte; ich ziehe daraus danach neue Phasen (TaskCreate pro Item, sequentiell).
>
> Status-Legende: [ ] offen · [x] passt · [!] Feedback (siehe Notiz)

---

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
