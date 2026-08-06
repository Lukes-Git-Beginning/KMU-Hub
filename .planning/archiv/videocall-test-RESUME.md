# RESUME — Videocall-Test (Luke, Nico, Darien)

**Stand 2026-06-22, ~00:40.** Komplette Video-Infra gebaut + deployt + verifiziert,
Test-Meeting ist LIVE. **Warten auf das Ergebnis des 2-Personen-Smoke.**

## ⏭️ NÄCHSTER SCHRITT (genau hier ansetzen)
Luke macht gerade den **2-Personen-Smoke** (er + Darien oder Nico): App → Meetings →
„Live jetzt" → **Beitreten**. Frag nach dem Ergebnis:
- **Bild + Ton da** → Nico/Darien dazu, 3er-Session + Feedback-Rubrik (`.planning/videocall-test-runbook.md`).
- **„Verbunden" aber kein Bild/Ton, oder hängt beim Verbinden** → **fast sicher Hetzner-Cloud-Firewall**
  blockt UDP 7882. → siehe „Cloud-Firewall-Fix" unten.

## Das LIVE Test-Meeting
- **Meeting-ID:** `fc996dcd-0beb-4975-9109-662a4fbdf2bb`, Status `in_progress`, Raum
  `meeting-fc996dcd-...`, Tenant `00000000-0000-0000-0000-000000000001`.
- **Teilnehmer (alle Tenant …0001):**
  - Luke (Organisator) `818d8531-907a-484a-9ea9-7827f661038f` — `lukeleonhoppe@gmail.com`
  - Darien `4ba037b7-5ec1-454d-b6ca-0a6f3dc9c66e` — `darien.stelter@googlemail.com`
  - Nico `2306bfa8-9e0e-4537-975e-f01c6049d45c` — `laramariehartig@gmail.com`
- Neues Meeting anlegen: `python .planning/create-test-meeting.py` (Login Luke → erstellt+startet;
  Idempotency-Key + Attendee-UUIDs sind im Skript; macht Meeting sofort `live`).

## Was deployt + verifiziert ist (alles auf `origin/main`, prod grün)
- **Backend `POST /api/v1/meetings/{id}/join`** — idempotent: Organisator startet implizit,
  jeder Attendee bekommt LiveKit-Token + TURN `ice_servers`. (`401` ohne Auth = Route da.)
- **Frontend „Beitreten" → echte LiveKit-`VideoCallView`** (in MeetingsPage + VideoPage),
  ice_servers durchgereicht. Im verteilten Windows-Build enthalten.
- **3 Infra-Fixes live:** TURN aktiv (`work`-Log `TURN configured host=turn.zentria.tech`),
  LiveKit annonciert öffentliche IP (`using external IPs ["178.104.38.195"]`, RTC udp_port
  **7882** + tcp_port 7881), `modules.video=true` (Meetings-Nav sichtbar).
- **Meetings-Liste-Fix:** `GET /api/v1/meetings` gibt jetzt ein nacktes Array zurück
  (`response.ProtoList`) — vorher `{meetings:[…]}`/`{}` → Frontend-Crash
  `(apiMeetings ?? []).map is not a function`.
- **Windows-Installer:** `desktop/dist/Cosmi Setup 0.1.0.exe` (145 MB, signiert, zeigt auf
  `app.zentria.tech`, Demo-Mode aus) — an Nico + Darien verteilt, installiert, registriert.

**Commits:** `4a86e60a` (join feature + erste Infra) · `d4eadb1c` (LiveKit last-wins fix) ·
`e3e7c12f` (Meetings-Liste als Array). CD läuft via `gh workflow run cd.yml --ref main`
(deploy-only Commits triggern KEIN CI → CD manuell dispatchen).

## Cloud-Firewall-Fix (falls Smoke kein Medien-Flow zeigt)
Host-`ufw` ist **inaktiv** (kein Host-Blocking). Verdächtig: **Hetzner-Cloud-Firewall** (extern).
LiveKit-SFU lauscht/annonciert `178.104.38.195` mit **UDP 7882** (Mux) + **TCP 7881**.
→ In der **Hetzner-Cloud-Console** (Projekt → Firewalls, oder am Server `kmuhub-prod`) eine
Inbound-Regel **allow UDP 7882** (+ zur Sicherheit TCP 7881) ergänzen. Danach Smoke wiederholen.
Falls Luke einen **Hetzner-API-Token** hat: Firewall-Status direkt per API prüfen/setzen.
Diagnose im Call: DevTools → `chrome://webrtc-internals` → aktives Candidate-Pair? `relay`(TURN)/
`srflx`/`host` auf `178.104.38.195` = gut; nur lokale `172.x`/`192.168.x` + keine Verbindung = Port zu.

## Verify-Kommandos (Prod-Reads, read-only)
- Container/Health: `ssh -i ~/.ssh/hetzner_kmuhub deploy@178.104.38.195 'sudo -n docker ps --format "{{.Names}}\t{{.Status}}"'`
- TURN-Log: `... docker logs docker-work-1 2>&1 | grep -i "TURN "`
- LiveKit external IP: `... docker logs docker-livekit-1 2>&1 | grep -i "external"`
- DB: `... docker exec docker-postgres-1 psql -U kmuhub -d kmuhub -P pager=off -c "..."`
- ⚠️ Direkte Prod-Host-WRITES (compose-edit, docker restart) werden vom Classifier geblockt →
  Fixes IMMER über Repo + CD. DB-SELECTs gehen.

## Wichtige Learnings (für Memory/Knowledge)
1. **LiveKit v1.7.2 ist last-wins bei mehreren `--config`** (KEIN key-by-key Merge — Repo-Kommentar
   war falsch). Effektive Prod-Config = die LETZTE Datei = `livekit-secrets.yaml` (gerendert aus
   `livekit-secrets.yaml.tmpl` via `render-configs.sh`). Sie muss self-contained sein: keys +
   webhook + voller rtc-Block. Base-`livekit.yaml` wird in Prod ignoriert. Ein rtc-only Overlay als
   letzte Datei → „keys must be provided"-Crash-Loop (erst-Deploy-Fail dieser Session).
2. **Prod-Video-Stack war still fehlkonfiguriert:** TURN aus trotz gesetzter Secrets (Compose-
   Passthrough fehlte im `work`-Env), `use_external_ip:false` (interne IP annonciert), `modules.video`
   gar nicht gesetzt. Erst durch Aktivieren aufgedeckt.
3. **`GET /api/v1/meetings` (und potenziell weitere List-Endpoints) lieferten gewrappte Proto-
   Envelopes** statt nackter Arrays → Frontend `.map`-Crash. Fix: `response.ProtoList`. ANDERE
   List-Endpoints prüfen (recordings, action-items, active-calls) — gleiches Muster möglich.
4. **Idempotency Hard-Mode:** POST/PUT/PATCH/DELETE brauchen `Idempotency-Key`-Header (Login exempt).

## Offene Folge-Punkte (nach dem Test)
- Meeting-Erstellen-Formular ans Backend verdrahten (`useCreateMeeting`) + User-Picker.
- „Beitreten" auch für `scheduled` Meetings in der Meetings-Nav (Organisator startet aus Nav;
  aktuell zeigt scheduled nur „Details" → Meeting muss vorab gestartet werden).
- `/video`-Route hat keinen Nav-Eintrag (nur `/meetings` verlinkt).
- LiveKit-WARN: UDP-Receive-Buffer (`net.core.rmem_max` auf 5 MB erhöhen) — Performance.
- Andere List-Endpoints auf Wrapper-Bug prüfen (siehe Learning 3).
- Recording-Test (Runde 2): Egress/MinIO/Consent verifizieren.
