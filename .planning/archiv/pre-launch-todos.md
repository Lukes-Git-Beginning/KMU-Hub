# Pre-Launch To-Dos — App-Shell / Auth / Branding

> Separate Liste von der Modul-Feature-Parität. Betrifft die App-Hülle (Login, Session, Branding), nicht einzelne Module.
> Erfasst 2026-06-01. Launch-Ziel: 01.09.2026.

## Login & Session

| # | To-Do | Beschreibung | FE/BE | Notiz |
|---|---|---|---|---|
| 1 | **Anmelde-Screen überarbeiten** | Login-Screen redesignen (Premium-SaaS-Ästhetik, Cosmi-Identität, kein generisches Dashboard-Look) | FE | Design-Pass mit `frontend-design`/`impeccable`. Reference-Screenshots von Darien willkommen. |
| 2 | **"Daten speichern" für schnelles Einloggen** | Remember-me / sichere Zugangsdaten-Speicherung für schnellen Re-Login | FE | ✅ **Großteils schon da**: `main/ipc/auth.ts` nutzt Electron `safeStorage` (OS-Keychain) für verschlüsselte Token-Persistenz. Offen: explizite „Angemeldet bleiben"-Checkbox + UX-Politur. Kein Neubau. |
| 3 | **App-Sperre bei Inaktivität** | Programm nach ~20 Min Inaktivität sperren (Lock-Screen-Overlay), ohne auszuloggen; Entsperren via Passwort/PIN | FE (+evtl. BE) | ❌ Komplett offen (kein Idle/Auto-Lock im Code). Idle-Timer + Lock-Overlay; Session bleibt aktiv, nur UI gesperrt. Timeout konfigurierbar (Setting). Für Finanzberatung (DSGVO) relevant. Re-Auth-Flow mit Luke/Security abstimmen. |
| 4 | **Einblende-Animation Login → Programm** | Übergangs-Animation vom Login-Screen ins eigentliche Programm | FE | Nur `transform`/`opacity` (GPU, Motion-Hardrule). Tokens aus `lib/motion.ts`. |
| 6 | **"Passwort vergessen"-Flow** | Reset-Link im Login-Screen + Mail-Token-Flow | FE + BE | ❌ Fehlt komplett (Befund Welle 2). LoginPage hat keinen Reset-Link, BE keinen Endpoint. Vor Launch nötig. Backend → Luke (in `backend-gaps.md`). |

## Branding

| # | To-Do | Beschreibung | Notiz |
|---|---|---|---|
| 5 | **Cosmi-Logo fertigstellen** | Finales Cosmi-Logo fertig machen | Custom-SVG. In App-Shell, Login, Favicon/App-Icon, Loading-States einsetzen. |

## Modul-Einstellungen (Feature-Konzept, erfasst 2026-06-06)

> Größeres Feature — braucht **gründliche Analyse + Markt-Recherche + die richtige Tiefe** (Workflow [[feedback_market_driven_workflow]]). Hier nur das Konzept notiert, NICHT gebaut. Mit Login-Screen & Co. auf dieselbe Pre-Launch-Liste.

**Idee:** Ein eigener Einstieg zu **modulspezifischen Einstellungen**, getrennt von den globalen App-Einstellungen.

**Kern-Anforderungen (von Darien):**
1. **Eigener Einstieg statt globale Settings:** Der „Einstellungen"-Knopf links unten (bzw. ein Knopf) führt NICHT zu den normalen globalen Einstellungen, sondern zu den **Modul-Einstellungen**. (⚠ Zu klären: ersetzt es den bestehenden Settings-Knopf oder wird es ein zusätzlicher/zweiter Knopf? Vermutlich separat, da globale Settings weiter gebraucht werden.)
2. **Als Fenster/Modal in Cosmi** — öffnet sich als Overlay-Fenster, KEIN Routing zu einer anderen Programm-Seite. (Konsistent mit dem Kontakt-Detail-Modal-Ansatz.)
3. **Kontext-sensitiv:** Das Programm erkennt das aktive Modul. Bin ich in „Buchhaltung" und klicke auf Modul-Einstellungen → direkt die Buchhaltungs-Modul-Einstellungen offen. Gleiches Verhalten in jedem Modul.
4. **Anpinnbar:** Oben links (wo man Module anpinnen kann) lassen sich auch die Modul-Einstellungen anpinnen. Position immer gleich — toggle-bar nur die Sichtbarkeit des Buttons (wegmachbar, wenn man ihn nicht will).
5. **Inhalt pro Modul — zwei Ebenen:**
   - **Komfort/persönlich:** generelle Ansichts-/Komfort-Entscheidungen pro Nutzer (z.B. Standard-Ansicht Liste vs. Raster, Default-Sortierung etc.).
   - **Bereichs-/Admin-Ebene:** für Leute die den Bereich leiten — Überblicke + Einstellungen, die sich auf **alle Nutzer** des Moduls auswirken (modulweite Konfiguration).
6. **Berechtigungen (RBAC):** Rollenbasiert steuern, wer welche Bereiche der Modul-Einstellungen sehen/ändern darf (persönliche Komfort-Settings für alle; modulweite/Admin-Settings nur für Bereichs-Leiter/Rollen mit Recht).

**Für die spätere Tiefen-Analyse (offen):**
- Markt-Recherche: wie lösen vergleichbare Tools „Workspace-/Modul-/Space-Settings" + Rollen (Notion, Linear, monday, HubSpot Objekt-Settings)?
- Architektur: zentrale Modul-Settings-Registry (pro Modul deklarierte Settings-Schemas) → eine generische Settings-Fenster-Shell, die je aktivem Modul das passende Schema rendert. Persönlich vs. modulweit trennen (User-Scope vs. Tenant-Scope, passt zu Option-B-Multi-Tenancy).
- Welche konkreten Settings pro Modul? (je Modul beim jeweiligen Bau mitdenken/sammeln.)
- BE-Bedarf: Settings-Persistenz (User-Scope + Tenant/Modul-Scope), RBAC-Checks → Luke (`backend-gaps.md`).
- Pin-Mechanik: an die bestehende Modul-Anpin-Logik (oben links) andocken.

## Status (Stand 2026-06-02, nach Ist-Abgleich)
- #1 Anmelde-Screen: offen (Redesign)
- #2 Remember-me: **großteils erledigt** (safeStorage), nur Checkbox/UX offen
- #3 App-Sperre: offen (Neubau)
- #4 Login→App-Animation: offen
- #5 Cosmi-Logo: offen
- #6 Passwort-vergessen: offen (FE+BE — Luke-Backend in `backend-gaps.md`)
