# Operations Runbook

> Status: Skelett. Inhaltliche Befuellung in Sprint 5/6, bevor Pilot-0 live geht.

On-Call-Operationen fuer Production-Incidents.

## Eskalations-Pfad

1. **Severity 1** (Production komplett down, Datenverlust drohend) — sofortiges Paging an Luke (Discord-DM + Telefon).
2. **Severity 2** (Modul-Ausfall, Login broken, Pilot-Kunde betroffen) — Discord-Channel `#cosmi-prod-alerts`, Reaktion in 30 Min.
3. **Severity 3** (Performance-Degradation, Single-User-Bug) — Tracking in Issue-Tracker, Bearbeitung im naechsten Sprint.

## Alert-Kanaele

- **Discord `#cosmi-prod-alerts`** — Alertmanager-Webhook (`ALERT_WEBHOOK_URL` in `.env.production`)
- **GitHub Actions Failures** — Repo-Notifications
- **Sentry** — *TODO Sprint 5: Integration aktivieren oder verwerfen*

## Haeufige Incidents

### Container unhealthy

*TODO Sprint 5: SOP fuer einzelne Services (auth, crm, ...).*

Snippet:

```bash
ssh deploy@prod
docker compose ps --filter health=unhealthy
docker compose logs --tail=200 <service>
docker compose restart <service>
```

### Migration-Drift

*TODO Sprint 5: SOP wenn DB-Migration-Head und Code-Migration-Head divergieren.*

### Auto-Smoke False-Positive

`--skip-smoke` ist Default, manuelle Smoke-Verify in Phase D.

### RLS-Policy 42501 Spike

Indikator: Service-Logs voller "row violates policy" oder "permission denied for table". Ursache meist:

- Migration aktiv, aber Service-Build nicht aktualisiert (INSERT ohne tenant_id)
- sysctx-Wrap fehlt auf einem Pre-JWT-Pfad

*TODO Sprint 5: Diagnostik-Queries dokumentieren.*

### OnlyOffice unhealthy (JWT-Secret fehlt)

Indikator: `docker compose ps` zeigt OnlyOffice-Container als `unhealthy`, Dokumente lassen sich nicht oeffnen.

Ursache: `ONLYOFFICE_JWT_SECRET` ist leer in `/opt/kmuhub/.env.production`. Bestandsproblem seit Sprint 1.

SOP (einmalige User-Action):

```bash
ssh deploy@prod
SECRET=$(openssl rand -hex 32)
# /opt/kmuhub/.env.production editieren:
#   ONLYOFFICE_JWT_SECRET=<SECRET>
docker compose restart onlyoffice
docker compose logs --tail=50 onlyoffice  # erwartet: "JWT secret loaded"
```

Hinweis: `ONLYOFFICE_JWT_SECRET` konfiguriert ausschliesslich den DocumentServer-Container (docker-compose.prod.yml mappt es auf dessen `JWT_SECRET`). Der Go-Code (gateway + document) nutzt fuer das WOPI-Protokoll ein eigenes Secret: `WOPI_JWT_SECRET`. Eine Variable `DOCUMENT_JWT_SECRET` existiert nicht — falls sie noch in `/opt/kmuhub/.env.production` steht, ist sie eine wirkungslose Altlast.

### TURN-Server unerreichbar

Pilot-0 erfordert TURN fuer Calls. coturn laeuft auf eigener Hetzner-VM (`turn.zentria.tech:3478`).

*TODO Sprint 5: SOP fuer coturn-Restart + LiveKit-Failover.*

## Deploy-Prozeduren

Siehe `deploy/scripts/deploy.sh`. Pflicht-Reihenfolge:

1. Backup-Erstellung (`backup.sh`) vor jedem Production-Deploy
2. `sudo GIT_SSH_COMMAND='ssh -i /home/deploy/.ssh/github_deploy' bash deploy/scripts/deploy.sh --force`
3. Phase D manuelle Smoke-Verify

`--skip-smoke` ist Default solange Bestands-Smoke-Issues offen (POST /contacts 403, OnlyOffice JWT, SMOKE_ADMIN_TOKEN TTL, Idempotency 401).

## Rollback

```bash
sudo bash deploy/scripts/rollback.sh
```

ACHTUNG: `rollback()` rollt nur Code, NICHT DB-Migrationen. Bei DB-Drift muss manuell `migrate -path migrations -database $MIGRATION_DATABASE_URL down N` ausgefuehrt werden, wobei N die Anzahl der seit dem letzten guten Stand applied Migrations ist.

## Kontakte

- **Luke** — Lead Dev, Discord-DM `@luke` / Telefon (siehe Pilot-Vertrag)
- **Hetzner Support** — Cloud-Console > Support, Reaktion innerhalb 4h (Business-SLA)
- **Domain-Registrar (zentria.tech)** — *TODO Sprint 5: Registrar-Login dokumentieren*
