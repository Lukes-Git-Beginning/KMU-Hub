# Review — profil

> **Status:** review-reif (P-1…P-5, Sub-Terminal, gemergt, `6f44d65a`).
> **Lane:** FE/UX-Review (mock-first). Echtes Avatar-/Dokument-Storage, echte DND-Backend-Anbindung = Lukes Lane.
> **Screens:** `/#/profil` — Tabs Profil · Dokumente · Abwesenheiten · Zeiterfassung.

## Was gebaut wurde (Definition of Done)
- [x] **P-1 — current-user = Stefan Vogel** (durchgängig: Header, Form, Sidebar, Topbar — kein „Darien Morales" mehr). „Mitglied seit" reaktiviert (März 2021, lokalisiert). **Abwesenheiten-Tab-Crash gefixt** (camelCase ↔ snake_case Datenvertrag) → zeigt Stefans 3 eigene Anträge, Resturlaub 17/30 konsistent.
- [x] **P-2 — Dokumente-Tab an MSW**: 7 Demo-Docs (Arbeitsvertrag / Gehaltsabrechnungen / Zertifikate / Bescheinigung) mit Kategorie/Größe/Datum/Uploader. Upload-Dialog (stateful), Vorschau = `DetailModal` (Metadaten + Demo-Badge + sticky Download), Download = echter Blob. **Ganze Doc-Zeile klickbar.**
- [x] **P-3 — Avatar + DND**: Avatar-Upload über MSW (überlebt Reload). DND-Schalter im Demo umschaltbar (lokaler Fallback statt „Backend nicht erreichbar"-Disabled), Status-Text spiegelt den Zustand.
- [x] **P-4 — Dead-Code-Cleanup**: verwaisten `tabs/zeiterfassung/`-Ordner (11 Dateien, 0 Importe) + 151 tote i18n-Keys (0 Live-Referenz) entfernt — echtes Zeiterfassung-Modul **unverändert funktional**, Build grün.
- [x] **P-5 — Profil-Karte + Schlusscheck**: Profil-Karte (Overlay, Ping→Chat) verifiziert — Fremd-User „Nachricht senden" navigiert in den Chat, eigener User „Mein Profil öffnen". Keine toten Buttons / Toast-Stubs, 0 Raw-Keys über alle 4 Tabs.

## Worauf besonders achten
- Identität durchgängig Stefan Vogel (Header/Form/Sidebar/Topbar)?
- Dokumente: Liste gefüllt, Upload → erscheint oben + überlebt Session, Vorschau-Modal sticky Download?
- Avatar wechseln → überlebt Reload? DND-Schalter wirklich umschaltbar?
- Zeiterfassung-Tab nach dem Cleanup vollständig (Heute/Woche/Auswertungen/Team/Korrekturen), keine Raw-Keys?
- Profil-Karte irgendwo aufrufen (Team-Dashboard/Chat) → „Nachricht senden" landet im Chat?

## Out of scope (kein Mangel)
- Echtes Avatar-/Dokument-Storage-Backend, echte DND-Backend-Anbindung.
- Bekannte app-globale Demo-Lücke: `GET /feature-flags` 1× `ERR_CONNECTION_REFUSED` beim Start (Feature-Flag-System ist per ADR OFF) — tritt überall auf, **nicht** profil-spezifisch.

## Befunde
> Format pro Zeile: **Schweregrad** (P0 / P1 / P2) · **was** · **wo** (Datei/Screen) · **Repro**.

_(noch keine — von Nico zu füllen)_
