# Prod-Seeds + Feature-Flags — Runbook (Hetzner)

Entsperrt den Hetzner-Review von helpdesk/wiki/berichte + Inbox **mit Daten**.
Reihenfolge einhalten: Code/Migrationen zuerst, dann Flags, dann Seeds.

## 0. Voraussetzung — Migrationen 236–238 müssen auf Prod sein

Die Seeds nutzen `tickets.description/category/ticket_number` (Migr. 236) und
`inbox_canned_responses` (Migr. 238). Diese kommen automatisch mit dem CD-Deploy
des Codes. Vor dem Seeden prüfen:

```bash
ssh -i ~/.ssh/hetzner_kmuhub deploy@178.104.38.195
sudo docker exec -i <pg-container> psql -U kmuhub -d kmuhub \
  -c "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1;"
# erwartet: 238 (oder höher)
```

## 1. Feature-Flags setzen (3 Zeilen) + Gateway neu starten

In `/opt/kmuhub/.env.production` ergänzen:

```
COSMI_MODULE_HELPDESK_ENABLED=true
COSMI_MODULE_WIKI_ENABLED=true
COSMI_MODULE_BERICHTE_ENABLED=true
```

Diese drei sind im Gateway-Block der `docker-compose.yml` bereits durchgereicht
(`${COSMI_MODULE_*_ENABLED:-false}`) — kein Compose-Edit nötig. Flags werden nur
beim Boot gelesen → Gateway neu starten:

```bash
cd /opt/kmuhub
sudo docker compose --env-file /opt/kmuhub/.env.production \
  -f deploy/docker/docker-compose.yml -f deploy/docker/docker-compose.prod.yml \
  up -d gateway
```

Check: `GET /api/v1/helpdesk/tickets` → 200 (nicht 404); Nav zeigt die 3 Module.

## 2. Reviewer-User ermitteln

`inbox_messages` ist **user-scoped** — der Seed muss auf die **eingeloggte
Reviewer-`user_id`** zeigen, sonst bleibt die Inbox leer.

```bash
sudo docker exec -i <pg-container> psql -U kmuhub -d kmuhub \
  -c "SELECT id, email FROM users ORDER BY created_at;"
```

UUID des Review-Accounts (z.B. Dariens Prod-Login) notieren → `<reviewer-uuid>`.

## 3. Seeds einspielen (idempotent, als Superuser → bypasst RLS)

Die UUID **ohne** Anführungszeichen übergeben — die Skripte quoten via `:'reviewer_id'` selbst.

```bash
sudo docker exec -i <pg-container> psql -U kmuhub -d kmuhub \
  -v reviewer_id=<reviewer-uuid> \
  -f - < backend/seeds/prod/helpdesk-seed.sql

sudo docker exec -i <pg-container> psql -U kmuhub -d kmuhub \
  -v reviewer_id=<reviewer-uuid> \
  -f - < backend/seeds/prod/inbox-seed.sql
```

`tenant_id` defaultet auf den Bootstrap-Tenant `00000000-…-0001`. Anderer Tenant:
zusätzlich `-v tenant_id=<tenant-uuid>` (ebenfalls ohne Quotes).

Beide Skripte geben am Ende Zählungen aus (tickets/open/messages bzw.
inbox_messages/unread/canned) — müssen > 0 sein. Mehrfaches Ausführen ist sicher
(idempotent: helpdesk via fixe UUIDs delete+reinsert, inbox/canned via ON CONFLICT).
