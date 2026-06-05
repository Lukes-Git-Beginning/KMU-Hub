# LiveKit / COSMI_ENV Production-Befunde — RESOLVED (2026-06-05)

> **Status: Alle Befunde behoben und in Production verifiziert** (Session 2026-06-05
> Abend, Commits `68158907`, `7d492bb6`, `5f16f0d9`, `78043a63`, `ce2a5e5d`).
> Dieses Dokument bleibt als Incident-Historie erhalten — die Befund-Klasse
> (Compose/Overlay-Gap, RLS-Read-Gap) ist lehrreich fuer kuenftige Service-Onboardings.

## Urspruengliche Befunde (entdeckt bei der Smoke-Gruen-Session)

| # | Befund | Fix |
|---|---|---|
| F-A | `COSMI_ENV` nirgends gesetzt → Prod-Secrets-Assertion (R1-P0.3) lief nie | `COSMI_ENV: ${COSMI_ENV:-development}` auf allen 24 Services + `COSMI_ENV=production` in `.env.production` (`68158907` + Server-Edit) |
| F-B | work/dialer signierten LiveKit-Tokens mit hartkodiertem `devkey` | `${LIVEKIT_API_KEY:-devkey}`-Interpolation; Prod zieht echten Key aus `.env.production` (`68158907`) |
| F-C | Gateway ohne LiveKit-API-Pair → Webhook-Validierung im Skip-Modus | Gateway bekommt das Pair in der Basis-Compose; Log-Warnung verschwunden (`68158907`) |

## In der Fix-Session zusaetzlich entdeckt

| # | Befund | Fix |
|---|---|---|
| F-D | **Alle 24 Services liefen in Prod mit den Dev-Secrets der Basis-Compose** (JWT_SECRET `docker-dev-secret-…`, VAULT_MASTER_SECRET, WOPI_JWT_SECRET, MinIO `minioadmin`) — das Prod-Overlay deckte nur ONLYOFFICE/Grafana/LiveKit-Server ab | Kompletter `${VAR:-dev-default}`-Sweep der Basis-Compose; Prod gewinnt via `--env-file` (`68158907`). JWT-Rotation invalidierte Alt-Sessions (akzeptiert, keine echten User) |
| F-E | Assertion-Blocklist blockte nur die Go-Defaults (`wopi-dev-secret-change-me`, `kmuhub_dev`), nicht die abweichenden Compose-Dev-Werte; JWT_SECRET wurde gar nicht geprueft | Zweigeteilte Deny-Lists: `composeDevSecrets` (immer verboten) + `configDefaultSecrets` (nur bei deklariertem Requirement) + JWT-Mindestlaenge 32 (`68158907` + `78043a63`) |
| F-F | `egress.yaml` hardkodierte devkey+minioadmin ohne Overlay-Mechanismus → Recording-Uploads tot | `egress.yaml.tmpl` + render-configs.sh rendert `egress-secrets.yaml`; Prod-Overlay mountet es ueber `/etc/egress.yaml` (`68158907`) |
| F-G | `LIVEKIT_WS_URL=wss://<ip>:7443` war ein nie funktionaler Template-Wert (Port nirgends gemappt) UND work nutzte dieselbe URL fuer die Room-/Egress-Server-API (twirp) → CreateRoom connection refused | URL-Split: neues `LIVEKIT_INTERNAL_URL=ws://livekit:7880` fuer Server-API (`LiveKitServerAPIURL()`-Fallback), `LIVEKIT_WS_URL=wss://app.zentria.tech` public. Caddy proxyt `/rtc*` → livekit:7880 (Signaling mit TLS; twirp bleibt intern; Media via WebRTC-Ports+TURN) (`7d492bb6`) |
| F-H | `JoinCallResponse.ws_url`/`StartMeetingResponse.{token,ws_url}` waren IMMER leer — der Kommentar "populated by gateway" beschrieb nie existenten Code; der Desktop-Client speist exakt diese Felder in die LiveKit-Connection → **Video-Join war auch im Meeting-Flow nie funktionsfaehig** | VideoGRPCServer bekommt tokenGen (RoomManager) + publicWSURL; StartMeeting stellt den Organizer-Token aus (`7d492bb6`) |
| F-I | `call_sessions`-SELECTs lasen `tenant_id` nicht zurueck → JoinCall erbte `uuid.Nil` in den `call_participants`-INSERT → RLS-Verletzung (42501), Join = 500. Klassischer Read-Side-Gap des Option-B-Retrofits | `tenant_id` in `GetCallSession`, `GetCallSessionByRoomName`, `ListActiveCallsForUser` (`5f16f0d9`) |
| F-J | Assertion-Scharfschaltung riss die schlanken Modul-Services (formulare, vertraege, …): sie erhalten kein WOPI/MinIO/Vault-Env, fielen auf Go-Defaults zurueck, die unconditional Checks verweigerten den Start. R1-P0.3 war nie mit den Welle-2-Services kompatibel | Variadic `config.Load(ctx, ...Requirement)`: `RequireVault` (auth, email, biz), `RequireMinIO` (chat, work, email, document, gateway), `RequireWOPI` (document, gateway). Compose-Dev-Werte bleiben fuer ALLE verboten (`78043a63`) |
| F-K | dialer hatte das LiveKit-Pair, aber als einziger LiveKit-Service kein `LIVEKIT_WEBHOOK_SECRET` → Trio-Check liess ihn als letzten Service im Restart-Loop | Compose-Zeile ergaenzt (`ce2a5e5d`). Followup: dialer nutzt LiveKit im Code gar nicht direkt — Env-Block evtl. komplett entfernen |

## Verifikation (Production, 2026-06-05)

- Smoke 24/24 nach jedem der Deploys (CD mit Auto-Rollback aktiv)
- Gateway-Log ohne `livekit webhook signature validation disabled`
- work/dialer/gateway `printenv LIVEKIT_API_KEY` ohne devkey-Treffer
- Egress-Container mit gerenderter Config (0 devkey/minioadmin)
- **LiveKit-Token-Probe end-to-end: Login → CreateCall 201 → JoinCall 200 (Token + wss-URL) → `GET /rtc/validate?access_token=…` → 200** — Video-Calls in Production erstmals funktionsfaehig
- `COSMI_ENV=production` scharf: alle 24 Services starten mit aktiver Assertion,
  Smoke 24/24 PASS und Token-Probe `/rtc/validate` = 200 unter dem finalen Stand

## Lessons

1. **Prod-Overlay-Coverage ≠ Basis-Compose-Secrets.** Jeder neue Service braucht
   einen bewussten Blick: welcher Env-Eintrag ist Secret, wo kommt der Prod-Wert
   her? Seit dem Sweep gilt: Basis-Compose nutzt `${VAR:-dev-default}`, Prod
   gewinnt automatisch via `--env-file` — neue Secrets DIESEM Muster folgen.
2. **Assertions muessen die realen Dev-Werte blocken**, nicht nur die eigenen
   Defaults — und nur das pruefen, was der Service konsumiert (Requirements-API).
3. **RLS-Read-Gap:** Wer `tenant_id` in INSERTs vererbt, muss sie in JEDEM
   speisenden SELECT zuruecklesen (bekannte Lesson, dritter Fund dieser Klasse).
4. **Template-Werte testen:** `wss://…:7443` stand seit Sprint 0 in
   PRODUCTION_TEMPLATE und war nie erreichbar — niemand hat je ein Token validiert.
   Verifikation gehoert in den Smoke (Followup: LiveKit-Smoke-Test).
5. **CI-Paths-Filter-Falle:** deploy-only-Commits (`deploy/**`) triggern kein CI,
   und CD haengt per `workflow_run` an CI → fuer reine Deploy-Aenderungen
   `gh workflow run CD --ref main` dispatchen.

## Offene Followups (non-blocking)

- Smoke-Test 25: LiveKit-Token-Probe (`/rtc/validate`) in `deploy/scripts/smoke.sh`
  aufnehmen (Sprint 4/5)
- `DOCUMENT_JWT_SECRET` ist eine wirkungslose Altlast in `/opt/kmuhub/.env.production`
  (Go-Code kennt nur `WOPI_JWT_SECRET`) — bei naechstem Server-Touch entfernen
- Calendar-Event-Join (`GenerateJoinToken`) liefert in `ws_url` einen Meeting-LINK
  (`…/room/{name}`) statt der Signaling-URL — pruefen, ob der Event-Flow im Client
  ueberhaupt verdrahtet ist (Sprint 4/5)
- Ansible `env.production.j2` fehlen `MIGRATION_DATABASE_URL`/`KMUHUB_APP_DB_PASSWORD`
  (Role-Split aus Migration 121) — vor erstem Pilot-Provisioning nachziehen
