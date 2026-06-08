# Luke-FE-Backlog — Lane vertraege · dashboard · profil

> Nach dem Pilot (`phase-01-vertraege-settings.md`). Reihenfolge = Modul für Modul, je Phase eine Bau-Einheit mit voller Verify-Schleife. Phasen-Detail jeweils in `module-phase-plans.md` (Module mit „→ Strom L" markiert). Spec wird bei Bedarf hier verfeinert.

## vertraege (FE-only auf Zustand-Store)
- **P1 — Backend-Anbindungs-UI + Audit-Log + Erinnerungen:** FE-Vorbereitung für echte contracts-CRUD (TanStack-Hooks-Gerüst, Audit-Log-Tab read-mock, Fristen-/Erinnerungs-Badges). Echte Endpoints baust du vormittags im Backend.
- **P2 — Dokumente echt:** Upload statt String, Versionen, PDF-Viewer (an Dokumente-Modul/MinIO andocken — koordiniert mit deinem Backend).
- **P3 — E-Signatur:** Eigen-Canvas (EES) als FE; Skribble (eIDAS) als Option markieren. Rechtsstufe = Darien-Entscheidung → im Review-Faden als offene Frage.
- **P4 — CRM/Finanzen-Verknüpfung:** Vertrag ↔ Kontakt/Deal/Rechnung, KI-Fristencheck (Stub).

## dashboard (vollständig; Layout-Persistenz Mock)
- **P1 — Widget-Konfiguration + Persistenz-UI** (Backend-Persistenz = dein BE).
- **P2 — DnD-Resize/Reorder** (dnd-kit, wie Work-Kanban-Muster).
- **P3 — KPI-Widgets modul-/lizenzabhängig.**
- **P4 — Modul-übergreifende Übersicht + Alerts (FE), Team-Dashboard.**

## profil (4 Tabs, gut)
- **P1 — Avatar-Upload-UI + Presence-Status** (Upload-Endpoint = dein BE; FE mock-first bis dahin).
- **P2 — Benachrichtigungs-Präferenzen im Profil.**
- **P3 — Shortcuts zu Appearance/Security + Account-Info.**
- **P4 — Profil-Karte (Overlay anywhere, Ping→Chat).**

## Regeln (Erinnerung)
- Nur diese drei Module. Branch `marathon/luke-fe`. Hot Files additiv. Pro Phase: bauen → i18n ×4 → scoped tsc → QA → Screenshots ansehen → Review-Faden → commit → push.
- Backend-Bedarf → `backend-handover-luke.md`. „Kompiliert" reicht nicht.
