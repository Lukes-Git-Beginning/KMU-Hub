# LiveKit / COSMI_ENV Production-Befunde (2026-06-05)

Entdeckt bei der Smoke-Grün-Session (Sprint-4-Vorzug). Alle drei Befunde gehören
zu EINEM Fix-Komplex und sollten in einer eigenen, fokussierten Session mit
beobachtetem Deploy behoben werden — naive Teilfixes können Production crashen.

## F-A: Production-Secrets-Assertion ist inaktiv (KRITISCH)

`backend/internal/config/config.go:154` — das Feld heißt `COSMI_ENV`
(default `development`). Weder `/opt/kmuhub/.env.production` noch irgendein
Container setzt `COSMI_ENV=production`. Folge: `AssertProductionSecrets`
(R1-P0-Fix) läuft in Production NIE scharf. Verifiziert via
`docker exec docker-{gateway,work,dialer}-1 printenv COSMI_ENV` → not set.

## F-B: work + dialer signieren LiveKit-Tokens mit devkey (KRITISCH)

`deploy/docker/docker-compose.yml` hartkodiert `LIVEKIT_API_KEY: devkey` /
`LIVEKIT_API_SECRET: devsecret` für `work` (Zeile ~257) und `dialer`
(Zeile ~461). Das Prod-Overlay überschreibt das nicht. Der LiveKit-Server
kennt aber nur den echten Key aus `livekit-secrets.yaml` (gerendert aus
`.env.production`, Key-ID `APIkeyd9ab…`). Folge: **Video-Calls, Meetings-Join
und Dialer-Calls können in Production nicht funktionieren** — LiveKit lehnt
devkey-signierte Tokens ab. Fiel nie auf, weil Pilot-0 noch nicht live ist.

## F-C: Gateway ohne LiveKit-Keys → Webhook-Validierung im Skip-Modus

Die am 2026-06-05 deployte Webhook-Signatur-Validierung (S4.2/R2-P1.1,
Commit f5788d8d) degradiert graceful: Gateway-Log
`livekit webhook signature validation disabled`. Der Gateway bekommt nur
`LIVEKIT_WEBHOOK_SECRET` (Basis-Compose, devsecret), nicht das API-Paar.

## Fix-Plan (eigene Session)

1. **Sweep:** Basis-Compose nach ALLEN hartkodierten Dev-Secrets durchsuchen
   (devkey/devsecret, `minioadmin`, `docker-dev-wopi-secret…`, …) und je
   Service klären, welcher Production-Wert wo herkommt.
2. **Prod-Overlay** (`deploy/docker/docker-compose.prod.yml`): per
   `${VAR}`-Interpolation aus `.env.production` durchreichen (Muster:
   `ONLYOFFICE_JWT_SECRET` Zeile 57): `LIVEKIT_API_KEY`/`LIVEKIT_API_SECRET`
   für gateway, work, dialer; `LIVEKIT_WEBHOOK_SECRET` für gateway
   (non-empty in .env.production vorhanden); fehlende Production-Werte
   (MinIO!, WOPI) in `.env.production` ergänzen.
3. **`COSMI_ENV=production`** erst aktivieren, NACHDEM Schritt 2 deployed und
   verifiziert ist — sonst crasht die Assertion alle Services beim Start
   (sie blockt Dev-Defaults UND leere Pflichtwerte; beachte auch die
   TURN-Symmetrie-Prüfung COTURN_HOST/TURN_SECRET).
4. **Verifikation:** Gateway-Log ohne "validation disabled"-Warn; Token-Probe
   gegen LiveKit; Video-Call-Smoke; Assertion-Log beim Start.

Achtung Deploy-Reihenfolge: Schritt 2 und 3 als getrennte Commits/Deploys,
jeweils mit Container-Watch. `--skip-smoke` ist seit 914a12dd entfernt — ein
Crash triggert den deploy.sh-Auto-Rollback (Code, nicht DB; hier ohne
Migrations unkritisch).
