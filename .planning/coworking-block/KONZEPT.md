# Co-Working-Space-Block — Konzept (durchgeplant 2026-07-20)

> **Auftrag (Darien, Idee 2026-07-20 nachts, durchgeplant 2026-07-20):** Virtueller Co-Working-Raum als Cosmi-STANDARD-Feature — Ambient-Presence mit Avataren + Voice, unabhängig vom Video/Meeting-Modul. Inspiration „On Together" (Steam), bewusst KEIN Klon. Referenzbild: Memory-Ordner `reference-virtual-coworking-on-together.png`.
> **Status: KONZEPT ENTSCHIEDEN (Besprechen-Gate durchlaufen, 4 Darien-Entscheide §3). Bau NICHT terminiert — Kandidat nach RBAC-Block (R-5→R-6) + Team-Review + O-0. Vor dem Bau: eigenes Terminal-Paket + Design-Recherche-Gate (Avatar-Stil/Raum-Szenen visuell).**

---

## 1. Markt-Kern (Web-Recherche 2026-07-20, Quellen im Report/Chat #22)

- **Vollbild-„Bürowelten" sind gescheitert:** Teamflow TOT · Around (Miro) eingestellt 03/2025 · Kosy aufgekauft · Kumospace Maintenance-Mode · Gather auf SMB zurückgeschraubt. Überlebt haben LEICHTE Ansätze (Tandem-Nische, Roam wächst mit 8-Min-Drop-ins, Discord-Voice als De-facto-Coworking, Slack Huddles).
- **Tragende UX-Muster:** Raum existiert IMMER (kein Scheduling) · Join ohne Redepflicht, Voice optional, Mute = Normalzustand · 3 Status reichen (Fokussiert/Ansprechbar/Pause) · dezente Join/Leave-Sounds · Anwesenheit sichtbar, Tätigkeit NICHT.
- **Body-Doubling ist belegt:** Focusmate +143 % Produktivität, +161 % bei Neurodivergenten, 85 % weniger einsam. Optionales „Ziel nennen beim Betreten" (Session-Commitment) steigert Wirkung.
- **On Together:** 94 % positiv (Consumer!), aber RAM-Lecks bis 32 GB → **Ressourcen-Budget = hartes Kriterium** (Ziel: idle <1 % CPU, <80 MB eigener RAM; nur transform/opacity animieren — deckt sich mit Motion-Hardrule).
- **DACH/BetrVG §87:** Anwesenheits-Tracking ist mitbestimmungspflichtig (echtes Vetorecht). BAG 2024: schon Echtzeit-Mithören ist mitbestimmungspflichtig. Voice-Aufzeichnung ohne Zustimmung strafbar (§201 StGB).
- **Fallen:** keine 2D-Welt bauen · kein Webcam-Ambient (Around-Tod) · kein Gaming-Look im B2B-Verkaufsgespräch · kein separates Abo (Presence als Teil des Pakets) · Phase 1 bewusst klein.
- **Cosmi-Differenzierer (hat KEIN Wettbewerber):** Inhalte aus dem Arbeitssystem in den Raum PINNEN — der Raum ist integrierte Fläche, nicht Deko daneben.

## 2. Technischer Ist-Stand (Explore 2026-07-20 — überraschend gute Anker)

- **Presence ist ECHT:** `features/presence/PresenceProvider.tsx` (WebSocket-Heartbeat 30 s, away nach 2 min Tab-Hidden) + `stores/presence.ts` (`presenceMap`, Levels online/away/dnd/in_call/offline, validiert in `route_video.go`), Bulk via `POST /video/presence/bulk`. TeamStatus-Widget konsumiert es bereits.
- **LiveKit fertig integriert:** BE `backend/internal/work/livekit/service.go` (Token+TURN), FE `livekit-client@2.17` + `features/video/*`. Fehlend: Audio-only-Token/Raumtyp + persistente (nicht call-gebundene) Räume.
- **Widget-Registry = Pin-Fundament:** 22 Widgets (`components/widgets/WidgetRegistry.tsx`, minimale WidgetProps) — beste Pin-Kandidaten: my-tasks, notification-summary, calendar-upcoming, team-status, team-chat. Fürs Panel braucht es eine abgespeckte Pin-Fläche (ohne react-grid-layout).
- **Overlay-Muster etabliert:** globale Overlays hängen in `AppShell.tsx` (FloatingCallBar, IncomingCallOverlay, SettingsOverlay …) → Phase-1-Panel dockt dort an.
- **Electron:** Popout-Muster existiert (`main/ipc/employee-wizard.ts`), aber `alwaysOnTop`/`frame:false`/`transparent` nirgends → Phase-2-Neuland.
- **Sounds:** `lib/notification-sound.ts` (Web-Audio, kein Asset) — Join/Leave-Töne dort ergänzbar.
- **Avatare:** Radix-Avatar + Initialen-Fallback; SVG-Illustrations-Suite (`components/shared/illustrations/`) definiert den Stil, aber KEIN Charakter-Set → neu.
- **Flag-Weg:** `featureflag/registry.go` + `GET /feature-flags`; als Modul zusätzlich Nav/Registry wie üblich.

## 3. Darien-Entscheide (2026-07-20, verbindlich)

1. **Format gestaffelt:** Phase A = andockbares Panel im Cosmi-Fenster (minimierbar zum Chip, AppShell-Overlay-Muster). Phase B = echtes rahmenloses Always-on-top-Mini-Fenster über allen Programmen (Electron: transparent/frameless/alwaysOnTop, eigenes `main/ipc/coworking.ts`).
2. **Eigenes MODUL + eigene Charaktere (minimal):** Co-Working ist ein eigenes Cosmi-Modul: dort **Räume erstellen** (Raum-Art + Funktionen wählbar), **Räume von Kollegen sehen + beitreten**, eigenen **Charakter personalisieren**. Personalisierung bewusst KLEIN (kein großer Charakter-Editor): ein paar Styles/Farben/Accessoires — **Kern-Idee: auswechselbare Köpfe, u.a. die PLANETEN DES SONNENSYSTEMS** (= Marken-Volltreffer zu „Cosmi", eigene Identity statt fremder Mascots). Warm + professionell, „weil Arbeitsprogramm".
3. **Privacy WÄHLBAR statt hart:** Anwesenheits-Auswertung ist eine **Tenant-Einstellung**. Empfehlung fürs Bau-Terminal: **Default AUS** (keine Logs, kein Chef-Dashboard, nur Echtzeit-Sichtbarkeit; Voice nie aufgezeichnet bleibt IMMER hart) + beim Aktivieren Hinweis auf Mitbestimmungspflicht (BetrVG §87) — so bleibt „betriebsratssicher by default" als Verkaufsargument UND Firmen mit Betriebsvereinbarung können mehr.
4. **Timing:** Kandidat NACH RBAC-Block (R-5 → R-6) + großem Team-Review + O-0 — als Sympathie-USP vor Launch 01.09 denkbar, Priorität = Dariens Call bei der nächsten Wellen-Planung.

## 4. Produktbild (Scope-Skizze fürs Bau-Paket)

- **Kern (Phase A, bewusst nur 3 Dinge + Modul-Rahmen):** ① Raum-Presence (Avatare erscheinen beim Joinen, Status Fokussiert/Ansprechbar/Pause, Join/Leave-Sound dezent) ② Voice optional (Audio-only-LiveKit, Mute default, „stiller Fokus" ohne Raum zu verlassen) ③ **Cosmi-Pins** (Aufgaben/Benachrichtigungen/nächster Termin als kompakte Karten im Raum — Widget-Registry-Ableger). Modul-Rahmen: Raum-Liste (eigene + beitretbare der Kollegen), Raum erstellen (Name, Szene, Funktionen an/aus), Charakter-Tab (Kopf/Farbe/Accessoire — Planeten-Set als Start).
- **Szene:** illustrierter gemeinsamer Arbeitstisch im Stil der bestehenden SVG-Suite; Charaktere = kleine Cosmi-Figuren mit wechselbaren Köpfen (Planeten etc.), Status-Ring. KEIN Pixel-Look, keine Minigames, kein Timer-Widget (Dariens Vorgabe; optionales „Ziel nennen beim Betreten" als leichtes Commitment-Feature prüfen).
- **Phase B:** dasselbe Panel als transparentes Always-on-top-Fenster (Sticker-Modus wie On Together, aber Cosmi-Stil); Multi-Monitor + Ressourcen-Budget beachten.
- **Später/offen:** Raum-Themes, Accessoire-Erweiterungen, Session-Statistiken NUR wenn Tenant-Privacy es erlaubt, Orbit/Selfhost-Betrieb (LiveKit self-hosted ✓ passt).
- **BE-Bedarf (Luke, wenn terminiert):** persistente Raum-Verwaltung (tenant-scoped, `coworking_rooms` + Mitgliedschaft), Audio-only-Join-Token (`POST /coworking/rooms/:id/join`), Presence-Dimension „im Raum X" (parallel zu presence-Levels), Charakter-Prefs (user-scoped), Flag `modules.coworking`, RLS wie üblich (tenant_id!).

## 5. Nächste Schritte

1. Bei der nächsten Wellen-Planung (nach RBAC/Review/O-0) priorisieren — dann eigenes Terminal-Paket schnüren (BRIEFING mit Datei-Karte).
2. **Design-Gate vor dem Bau:** Avatar-/Szenen-Stil visuell entwickeln (Planeten-Köpfe-Set, Tisch-Szene) und MIT Darien abstimmen (Reference-Screenshots = Pixel-Wahrheit-Regel gilt).
3. Pricing-/Abgrenzungsfrage zum video-Modul im Team klären (Standard-Feature = im Basis-Paket, kein eigener Meter — Markt-Lehre §1).
