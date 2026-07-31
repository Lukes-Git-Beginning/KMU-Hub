# Intake-Pilot — Review-Feedback & Bau-Plan (Darien lokal, 2026-07-30)

> Live-Review der Session #32-Pilot. Feedback wird zu Bau-Phasen.
> Status: [ ] geplant · [~] im Bau · [x] fertig+QA

## ✅ BAU-STATUS (2026-07-30) — 6 Commits, verifiziert, NICHT gepusht
- **Block A KOMPLETT** (`94028e79`, `0b557440`): pro-Kanal-Vorlagen (`intakeForms {agent,selfservice,external}`),
  2 echte bearbeitbare Ticket-Formulare, Kanäle-Panel pro Kanal + „Neue Vorlage", Deep-Link
  „Formular bearbeiten →" (`/formulare?edit=<id>`), Agent-Dialog = Vorlage-Felder + Profi-Werkzeuge
  (extrahierte `IntakeFieldInputs`), Selfservice bindet Kanal-Formular.
- **Block C KOMPLETT** (`0b557440`, `668d0b05`): Feld-Trennung (Modul-Zusatzfelder = intern, raus aus
  dem Neu-Dialog, im Detail; Einreicher-Felder aus der Vorlage), Editor-Hinweis „Zusatzfelder = intern".
- **B1 KOMPLETT** (`e0c32f5b`): Statistik-Aufschlüsselungen generisch + dynamisch — jedes Select-Zusatzfeld
  wird automatisch zu „Nach <Feld>" (SLA-Stufe, Kontaktkanal…), Namen aus den Wertelisten.
- **B3 KOMPLETT** (`dfe86019`): CSAT-Umfrage-Frage konfigurierbar (Store + Settings-Feld + Widget zeigt sie).
- **B2 REST (bewusst offen):** dynamische Feld-Breakdowns im Editor-Statistik-Katalog togglebar machen.
  Braucht den Custom-Field-Hook im Panel-Kontext (außerhalb Sandbox) — nicht-trivial. Breakdowns sind
  aktuell immer sichtbar (Dariens Kern-Wunsch erfüllt). Sauber nachziehen (ggf. mit der Editor-Doku).
- **2 Bugs unterwegs gefixt** (beide von Playwright-QA gefangen): JSON-Syntaxfehler in de.json (nicht-
  escaptes `"`), verwaister `setNtSubject` in `handleOpenNewTicket`.
- **NÄCHSTES:** Darien reviewt A+C+B1+B3 lokal → dann Editor-Dokumentation (Rollout-Vorlage).

---

## BLOCK A — Pro-Kanal-Ticket-Vorlagen (aus Punkt 4)

**Befund (2 Bugs + 1 Konzept-Lücke):**
1. Das gebundene „Support-Ticket (Helpdesk)" ist eine **Vorlage** (`isTemplate: true`, `tmpl-ticket-intake`), kein bearbeitbares Formular → man kommt beim Bearbeiten nicht weiter.
2. „Formular bearbeiten →" navigiert nur auf `/formulare` (Übersicht), öffnet **nicht** das konkrete Formular im Editor (Editor ist state-basiert `openEditor(schema)`, cross-window zum eigenen Editor-Fenster).
3. Nur **ein** globales `intakeFormId` für alle Kanäle → keine Differenzierung Agent ≠ Selfservice ≠ Extern.

**Darien-Entscheide (2026-07-30):**
- Jeder Kanal kriegt eine **zuweisbare Vorlage**; man erstellt Vorlagen und weist sie 1, 2 oder allen 3 Kanälen zu (Agent kann eine andere haben als Selfservice).
- Agent-Erstellpfad = **Profi-Werkzeuge behalten + Vorlage-Felder dazu**: der „Neues Ticket"-Dialog behält Kundensuche / Priorität / SLA-Stufe / interne Zuweisung UND rendert zusätzlich die Felder der gebundenen Agent-Vorlage.

**Phasen:**
- [ ] **A1** Datenmodell: `intakeFormId: string` → `intakeForms: { agent, selfservice, external }` (pro Kanal ein FormId, darf identisch sein). Store + Migration der Default-Bindung.
- [ ] **A2** Vorlage→Formular sauber: das gebundene Ding muss ein echtes, bearbeitbares Formular sein (Ticket-Intake-Vorlagen bearbeitbar machen ODER beim Binden eine Formular-Instanz aus der Vorlage erzeugen). Entscheidung im Bau.
- [ ] **A3** Kanäle-Panel: pro Kanal ein Formular-Wähler + „bearbeiten" + „neue Vorlage erstellen". Dropdown listet Ticket-Vorlagen (`intakeTargetId==='helpdesk_ticket'`).
- [ ] **A4** „Formular bearbeiten →" öffnet das **konkrete** gebundene Formular direkt im Editor (cross-window Deep-Link mit FormId).
- [ ] **A5** Vorlagen-Verwaltung: neue Ticket-Vorlagen anlegen/duplizieren, die man Kanälen zuweist.
- [ ] **A6** Agent-Dialog: Profi-Werkzeuge (Kontakt/SLA/Zuweisung) bleiben + Vorlage-Felder werden dazu gerendert (nach der gebundenen Agent-Vorlage).
- [ ] **A7** Selfservice/Extern: `IntakeFormFill` rendert das jeweilige Kanal-Formular (statt des globalen).

---

## BLOCK B — Statistik ↔ Wertelisten + CSAT-Umfrage (aus Punkt 5)

**Befund:** Status/Priorität-Aufschlüsselungen ziehen Label+Farbe **schon** aus den Wertelisten
(`useModuleValueSet`, HelpdeskPage:235-240) → Umbenennen wirkt live. **Aber:** die Aufschlüsselungen
sind **fest verdrahtet** (nur byStatus + byPriority). Eine **neue** Werteliste bekommt keine
Aufschlüsselung; eine entfernte bleibt als totes Widget.

**Darien-Wunsch (2026-07-30):**
- Statistik-Aufschlüsselungen sollen **dynamisch aus den vorhandenen Wertelisten** kommen: neue Werteliste → erscheint automatisch in der Statistik; keine Werteliste (bzw. entfernt) → taucht nicht auf. Namen immer aus der Werteliste gezogen (falls „Status" beim Kunden anders heißt, stimmt die Statistik mit).
- CSAT/Umfrage-Inhalt (was der Kunde bei der Rückfrage bekommt) muss später **änderbar** sein.

**Phasen:**
- [ ] **B1** Statistik-Aufschlüsselungen generisch: für jede am `helpdesk_ticket` gruppierbare Werteliste automatisch ein „Verteilung nach <Werteliste>"-Widget (byStatus/byPriority verallgemeinern, Namen/Farben aus dem Value-Set, neue/entfernte Sets spiegeln sich). *(Klärung offen: alle Wertelisten oder nur die mit Ticket-Feld — siehe unten.)*
- [ ] **B2** Statistik-Katalog im Editor (`StatistikPanel`) zeigt die Wertelisten-Widgets dynamisch + togglebar.
- [ ] **B3** CSAT-Umfrage-Inhalt konfigurierbar (Fragetext / Skala-Labels / was der Kunde sieht) — in Helpdesk-Settings oder Editor.

**Offene Klärung B1:** „jede Werteliste bekommt eine Aufschlüsselung" — gemeint sind vermutlich die
Wertelisten, nach denen ein Ticket auch ein Feld trägt (Status, Priorität, künftig Kategorie/Typ), weil
man nur danach gruppieren kann. Wertelisten ohne Ticket-Feld hätten keine Datengrundlage. → bestätigen.

---

## BLOCK C — Feld-Trennung: intern vs. Einreicher (aus Punkt 6)

**Darien-Prinzip (2026-07-30):** Es gibt zwei Feld-Welten am Ticket, die heute **vermischt** sind
(`useModuleCustomFields('helpdesk_ticket')` erscheint sowohl im Neu-Dialog als auch im Detail):

1. **Einreicher-Felder** — was jemand beim Erstellen ausfüllt (Betreff/Beschreibung/Gerät…). Gehören in
   den **Formular-/Ticket-Editor** = die Kanal-Vorlage (Block A). Selfservice/Extern sehen genau diese.
2. **Modul-interne Felder** — existieren nur im Modul, Einreicher sehen sie NICHT (Status, Aktionen,
   interne Vermerke, Eskalationsgrund). Nur Agenten im Modul pflegen sie. Gehören in die
   **„Zusatzfelder"-Sektion des Modul-Editors** (`useModuleCustomFields`).

**Konsequenz / Phasen (verzahnt mit Block A):**
- [ ] **C1** Modul-Zusatzfelder als **intern** kennzeichnen: erscheinen im **Agent-Dialog** (Agenten sind intern) + **Ticket-Detail**, aber NICHT im Selfservice-/Extern-Formular (`IntakeFormFill`).
- [ ] **C2** Einreicher-Felder kommen ausschließlich aus der Kanal-Vorlage (Formular-Editor), nicht mehr aus den Modul-Zusatzfeldern.
- [ ] **C3** Editor-Wording/Hint schärfen: „Zusatzfelder" = interne Felder, die nur Modul-Mitarbeiter sehen; Einreicher-Felder bearbeitet man im Ticket-Formular (Kanäle-Panel → „Formular bearbeiten").
- [ ] **C4** Feld-Taxonomie am Ticket dokumentieren: (a) Wertelisten-Felder Status/Priorität [intern], (b) Modul-Zusatzfelder [intern], (c) Intake-Formular-Felder [Einreicher, pro Kanal].

---

## Review-Stand
- [x] Punkt 4 (Kanal-Vorlagen) → Block A
- [x] Punkt 5 (Statistik ↔ Wertelisten + CSAT) → Block B
- [x] Punkt 6 (Feld-Trennung) → Block C
- [ ] Querschnitt (i18n/Emojis/sticky/leere Zustände) — offen
