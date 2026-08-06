# Videocall-Test Runbook (Luke, Nico, Darien)

Stand 2026-06-21. Production-Stack ist deployt + verifiziert. Dieser Runbook führt
den 3-Personen-Videocall-Test durch.

## Was bereits live + verifiziert ist

- **Backend `POST /meetings/{id}/join`** (idempotent, Organisator startet implizit,
  Attendees bekommen Token + TURN `ice_servers`) — Route auf Prod (`401` ohne Auth = vorhanden).
- **Frontend „Beitreten"** mountet die echte LiveKit-`VideoCallView` (statt Mock-Raum).
- **TURN aktiv:** `work`-Log `TURN configured host=turn.zentria.tech`.
- **LiveKit annonciert öffentliche IP:** `using external IPs ["178.104.38.195"]`,
  RTC UDP-Mux **7882** + TCP 7881 (= die von Docker published Ports).
- **`modules.video` an** → „Meetings"-Nav sichtbar.

**Eine offene empirische Unbekannte:** Hetzner-**Cloud**-Firewall für UDP 7882 (extern,
host-`ufw` ist inaktiv). Wird im 2-Personen-Smoke geklärt (webrtc-internals → `relay`/`srflx`).

---

## Schritt 1 — Accounts (gleicher Tenant)

Nico + Darien brauchen je einen Account. Zwei Wege:
- **Self-Register:** App öffnen → Registrieren (`RegisterPage`) → E-Mail + Passwort + Name.
  Bekommen automatisch Rolle `member` (reicht: `meetings:write` ist für alle Rollen geseedet).
- **Invite durch Luke:** `POST /api/v1/invitations` (braucht SMTP in Prod).

Luke hat bereits einen Account (Organisator).

> ⚠️ Alle drei müssen im **selben Tenant** sein (Self-Register über dieselbe Instanz = selber
> Tenant, da aktuell Single-Tenant). Danach mir die **3 E-Mails** nennen.

## Schritt 2 — Test-Meeting anlegen **und starten** (ich übernehme)

Wichtig: Der „Beitreten"-Button erscheint in der Meetings-Nav nur bei Status **live**
(`in_progress`). Ein nur `scheduled` Meeting zeigt bloß „Details". Daher wird das Test-Meeting
angelegt **und** sofort gestartet, damit alle drei „Beitreten" sehen.

Sobald die 3 Accounts existieren, erledige ich (mit Lukes Login als Organisator):
1. User-UUIDs + tenant aus der DB lesen.
2. `POST /api/v1/meetings` mit `attendee_user_ids: [Nico, Darien]` (Luke = Organisator, wird
   automatisch Attendee). Felder: `title`, `scheduled_start`, `scheduled_end` (RFC3339).
3. `POST /api/v1/meetings/{id}/start` → Status `in_progress`, Raum provisioniert.

→ Das Meeting taucht dann bei allen drei in **Meetings → „Live jetzt"** mit „Beitreten" auf.

## Schritt 3 — 2-Personen-Smoke ZUERST (Luke + 1)

Bevor alle drei zusammenkommen — Medien + TURN bestätigen:
1. Luke + eine zweite Person (Nico **oder** Darien), idealerweise in **verschiedenen Netzen**
   (einer Heimnetz, einer Mobilfunk-Hotspot), öffnen die App → Meetings → „Beitreten".
2. Erwartung: echte `VideoCallView` öffnet, beide sehen + hören sich.
3. **TURN/Connectivity prüfen:** im Call DevTools öffnen → `chrome://webrtc-internals` (Electron:
   über das Menü/DevTools) → ICE-Kandidaten ansehen. Erwartung: aktive Candidate-Pair mit
   `relay` (TURN) **oder** `srflx`/`host` auf `178.104.38.195`. Wenn **nur** `host` mit
   `172.x`/lokalen IPs und keine Verbindung → Cloud-Firewall UDP 7882 ist zu (siehe unten).
4. Mute/Kamera/Screenshare durchklicken.

**Wenn der Smoke scheitert (kein Bild/Ton trotz „connected"):** sehr wahrscheinlich Hetzner-
Cloud-Firewall. Dann: in der Hetzner-Console für den Server `kmuhub-prod` eine Firewall-Regel
**UDP 7882 inbound allow** (und zur Sicherheit TCP 7881) ergänzen. Danach Smoke wiederholen.

## Schritt 4 — 3-Personen-Session

Alle drei: App → Meetings → „Beitreten" auf dem Live-Meeting. Dann das eigentliche Feedback.

### Bewertungs-Rubrik (jeder füllt aus)
Bild (Schärfe/Framerate/Freeze) · Ton (Klarheit/Echo/Delay/Sync zum Bild) ·
Verbindungsaufbau-Dauer · Stabilität bei 3 Teilnehmern · Screenshare-Lesbarkeit ·
Gallery↔Speaker-Layout · gefühlte Latenz · **welche Optionen fehlen einem Kunden wirklich** ·
je Person: Netz-Setup (Heimnetz/Büro/Mobil) für NAT-Diagnose.

### Bekannte Lücken — bitte NICHT als „Bug" melden (sind bekannt/geplant)
- Im echten Call fehlen noch: In-Call Device-Picker (Mic/Kamera-Wechsel), Teilnehmer-Liste,
  In-Call-Chat, Hintergrund-Blur (Button da, ohne Funktion), „Hand heben" (nur lokal).
- FloatingCallBar (Mini-Leiste beim Tab-Wechsel): Mute/Kamera-Buttons sind No-op.
- Recording wurde in dieser Runde bewusst ausgeklammert (kommt in Runde 2).
- Meeting-Erstellen-Formular schreibt aktuell nur lokal (deshalb legt das Backend-Meeting ich an).

---

## Offene Folge-Punkte (nach dem Test)
- Meeting-Erstellen-Formular ans Backend verdrahten (`useCreateMeeting`) + User-Picker, damit
  Meetings UI-nativ entstehen.
- „Beitreten" auch für `scheduled` Meetings in der Meetings-Nav (Organisator startet aus der Nav).
- UDP-Receive-Buffer auf dem Server erhöhen (`net.core.rmem_max`) — LiveKit-WARN, Performance.
- Optional: `/video`-Route einen Nav-Eintrag geben (heute nur `/meetings` verlinkt).
