# Review — vertraege

> **Status:** review-reif (V-1…V-5 + Darien-Live-Fix-Runde F1–F5, gemergt, `f4a6844d`).
> **Lane:** FE/UX-Review (mock-first, FE auf Zustand-Store). Echte Backend-Anbindung / echte E-Signatur (Skribble) = Lukes Lane.
> **Screens:** `/#/vertraege` — Tabs Aktiv · Auslaufend · Archiv · Vorlagen · Einstellungen.

## Was gebaut wurde (Definition of Done)
- [x] **V-1 — Detail = zentriertes `DetailModal`** (war Slide-over): sticky Header (Titel + Typ · Nr + Status-Badge), sticky Footer (Bearbeiten / Unterschrift / Kündigen), Body scrollt intern. **Ganze Zeile klickbar** + tastaturbedienbar (Tab → Enter/Space); innere Buttons öffnen das Detail nicht (stopPropagation).
- [x] **V-2 — Dokument-Vorschau**: 4 aktive Verträge mit echter PDF-Vorschau im iframe (kein 404 / leerer Viewer). _(PDF rendert in Electron/headed; headless Chromium hat keinen PDF-Viewer.)_
- [x] **V-3 — Fristen-Notifications**: 3 „Vertrag läuft bald ab"-Hinweise in Bell + Center + Dashboard-Summary; „Auslaufend"-Tab gefüllt (3); kurzer Live-Toast beim Öffnen; **keine Duplikate** bei Reload. ICU-Plural-Bug im Notification-Center mitgefixt.
- [x] **V-4 — E-Signatur Demo-Flow**: „Zur Unterschrift senden" → Signer „Gesendet" + **ehrlicher Demo-Hinweis** (keine echte Mail); „Rücklauf simulieren" → Gesendet → Angesehen → Unterschrieben mit Audit-/Timeline-Events. Skribble = „Bald verfügbar".
- [x] **V-5 — Nummernkreis + echter Audit-User + Demo-Tiefe**: neue Vertragsnummer auto-vorbefüllt (V-2026-001 → 002), Live-Vorschau in Settings. Audit-Log trägt echten Auth-User (kein „Aktueller Benutzer"). Toter „Vertrag aus Vorlage"-Button gefixt (legt jetzt wirklich an).
- [x] **F1–F5 (Darien-Live-Fixes)**: E-Signatur aus dem reinen Detail entfernt (gehört in Verwaltung), ContractDialog → `DetailModal`, Detail schließt beim Bearbeiten, Dokumente-Sektion prominent, Notification-Karten aufklappbar (Öffnen/Anpinnen/Ignorieren).

## Worauf besonders achten
- Detail-Modal: beim Scrollen Header + Footer wirklich sticky? Status-Badge + Subtitle korrekt?
- „Auslaufend"-Tab: 3 Verträge, plausible Restlaufzeiten?
- Vertrag anlegen → Nummer vorbefüllt → erneut anlegen → hochgezählt?
- E-Signatur-Flow ehrlich gekennzeichnet (Demo-Rücklauf, keine echte Mail)?

## Out of scope (kein Mangel)
- Echte Vertrags-Backend-Anbindung, echte qualifizierte E-Signatur (Skribble/EES), echte Mail-Zustellung.

## Befunde
> Format pro Zeile: **Schweregrad** (P0 / P1 / P2) · **was** · **wo** (Datei/Screen) · **Repro**.

_(noch keine — von Nico zu füllen)_
