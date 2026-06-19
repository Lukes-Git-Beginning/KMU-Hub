# Review — team

> **Status:** review-reif (Tiefe-Pass TM-1…TM-5, Main-Terminal, gemergt, `8a49415c`).
> **Lane:** FE/UX-Review (mock-first). Echtes HR-Backend = Lukes Lane, **nicht** Teil dieses Reviews.
> **Screens:** `/#/team` — Tabs Mitarbeiter · Abwesenheiten · Personalakte · Organigramm · Schulungen · Self-Service.

## Was gebaut wurde (Definition of Done)
- [x] **TM-1 — Abwesenheiten-Kalender** zeigt echte Demo-Daten: Lena (Krankheit Di–Do), Markus (Homeoffice heute), Sophie (Sonderurlaub Mo); Felix/Julia korrekt rausgefiltert (außerhalb Fenster). Datums-/Abteilungs-Filter wirken. Datenvertrag-Bug (`{entries}` + camelCase) behoben.
- [x] **TM-2 — Self-Service** verkabelt: Profil = **Stefan Vogel** (= Sidebar/Topbar), Salden 17/30 Urlaub + 4/5 Sonderurlaub, 3 echte Anträge. **Neuen Antrag anlegen wirkt** (erscheint sofort, fließt in HR-Anfragen-Tab) + Blob-Download.
- [x] **TM-3 — Personalakte (Dokumente)** an MSW: echte Mitarbeiter (Felix Krause / Kevin Baumann / Laura Neumann), KPIs (12 Dok / 9 MA / 1 bald fällig / 1 abgelaufen), Vorschau-Dialog mit „Demo-Vorschau"-Badge, **Download wirkt** (lädt Datei, kein Toast-Stub), Upload verkabelt (File-Input + POST).
- [x] **TM-4 — Organigramm**: E-Mail-/Anruf-Buttons wirken (öffnen Mail / starten Anruf statt Toast). 8 i18n-Keys von `{{}}` auf `{}` korrigiert, `team.page.title` ergänzt.
- [x] **TM-5 — Mitarbeiter deaktivieren** komplett wirksam: Menü → „Deaktivieren" → Bestätigung → Mitarbeiter ausgegraut + „Inaktiv"-Badge (in Karte **und** Zeile). Schulungen-Tab rendert (6 Schulungen / 12 Teilnahmen, Anlegen/Erfassen).
- [x] **Umlaut-Cleanup**: „Geschaeftsführung" → „Geschäftsführung" (war im Abteilungs-Chip + Kalender sichtbar).

## Worauf besonders achten
- Abwesenheiten über mehrere Filter/Abteilungen durchklicken — stimmen die gezeigten Personen mit dem Zeitfenster?
- Self-Service: einen neuen Antrag anlegen → taucht er sofort auf und auch im HR-Anfragen-Tab?
- Deaktivieren: ganze Zeile vs. Menü; bleibt der „Inaktiv"-Zustand nach Reload sichtbar?
- EN-Umschalten über alle 6 Tabs (Raw-Keys / `{{ }}`).

## Out of scope (kein Mangel)
- Schulungen laufen über einen funktionalen Zustand-Store (swap-ready), **nicht** über MSW — der MSW-Swap ist Lukes Backend-Vorbereitung.
- Echte DATEV-Lohnschnittstelle, echtes Personalakte-Storage-Backend.

## Befunde
> Format pro Zeile: **Schweregrad** (P0 blockt / P1 sollte / P2 nice) · **was** · **wo** (Datei/Screen) · **Repro**.

_(noch keine — von Nico zu füllen)_
