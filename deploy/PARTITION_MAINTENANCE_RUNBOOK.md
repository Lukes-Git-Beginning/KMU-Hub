# Partition Maintenance Runbook — R2-P1.10

Declarative monthly range-partitioning + pg_cron retention for the three
ephemeral log tables: `events`, `dialer_call_events`, `automation_executions`.

`audit_log` is **excluded** — statutory retention (§257/§147 AO) + hash chain.

---

## Prerequisites

- Production DB backup complete and verified
- At least 15 minutes of low-traffic window (migration holds ACCESS EXCLUSIVE
  on each table briefly for RENAME, CREATE, INSERT, DROP)
- SSH access to Hetzner prod: `ssh -i ~/.ssh/hetzner_kmuhub deploy@178.104.38.195`

---

## Step-by-Step Maintenance Window

### 1. Create DB backup

```bash
ssh -i ~/.ssh/hetzner_kmuhub deploy@178.104.38.195
sudo docker exec -t $(sudo docker ps -qf name=postgres) \
    pg_dump -U kmuhub kmuhub | gzip > /opt/kmuhub/backups/pre-partition-$(date +%Y%m%d-%H%M).sql.gz
```

Verify the backup is non-empty before proceeding.

### 2. Build the pg_cron-enabled Postgres image

The custom Dockerfile at `deploy/docker/postgres/Dockerfile` extends
`pgvector/pgvector:pg16` with `postgresql-16-cron`.

```bash
cd /opt/kmuhub
sudo docker compose \
    --env-file /opt/kmuhub/.env.production \
    -f deploy/docker/docker-compose.yml \
    -f deploy/docker/docker-compose.prod.yml \
    build postgres
```

### 3. Restart Postgres with pg_cron loaded

The prod overlay command now includes `-c shared_preload_libraries=pg_cron`
and `-c cron.database_name=kmuhub`.

```bash
sudo docker compose \
    --env-file /opt/kmuhub/.env.production \
    -f deploy/docker/docker-compose.yml \
    -f deploy/docker/docker-compose.prod.yml \
    up -d postgres
```

Wait for the healthcheck to go green (up to 30 s):

```bash
sudo docker compose \
    --env-file /opt/kmuhub/.env.production \
    -f deploy/docker/docker-compose.yml \
    -f deploy/docker/docker-compose.prod.yml \
    ps postgres
```

### 4. Verify shared_preload_libraries

```bash
sudo docker exec -it $(sudo docker ps -qf name=postgres) \
    psql -U kmuhub -d kmuhub -c "SHOW shared_preload_libraries;"
```

Expected output includes `pg_cron`.

```bash
sudo docker exec -it $(sudo docker ps -qf name=postgres) \
    psql -U kmuhub -d kmuhub -c "SHOW cron.database_name;"
```

Expected: `kmuhub`.

### 5. Apply migration 000218

Run via the migrate service (uses `MIGRATION_DATABASE_URL` = `kmuhub` superuser
for DDL + CREATE EXTENSION privileges):

```bash
sudo docker compose \
    --env-file /opt/kmuhub/.env.production \
    -f deploy/docker/docker-compose.yml \
    -f deploy/docker/docker-compose.prod.yml \
    run --rm migrate
```

Or run golang-migrate directly:

```bash
sudo docker exec -it $(sudo docker ps -qf name=migrate) true 2>/dev/null || \
    migrate -path /migrations -database "$MIGRATION_DATABASE_URL" up 1
```

### 6. Verify partition structure

```sql
-- Connect to DB
sudo docker exec -it $(sudo docker ps -qf name=postgres) psql -U kmuhub -d kmuhub

-- Check partitions exist for each table
SELECT relname FROM pg_class
WHERE relname LIKE 'events_%'
ORDER BY relname;

SELECT relname FROM pg_class
WHERE relname LIKE 'dialer_call_events_%'
ORDER BY relname;

SELECT relname FROM pg_class
WHERE relname LIKE 'automation_executions_%'
ORDER BY relname;

-- Verify pg_cron jobs are registered
SELECT jobname, schedule, command FROM cron.job ORDER BY jobname;

-- Verify RLS is active on the two tenant tables
SELECT tablename, rowsecurity, forcerowd
FROM pg_tables
JOIN pg_class ON pg_class.relname = pg_tables.tablename
WHERE tablename IN ('dialer_call_events', 'automation_executions');
```

Expected: each table has 16 monthly partitions + 1 DEFAULT partition.
pg_cron jobs: `partition-advance-events` (25th monthly) + `partition-retention-drop` (daily).

### 7. Run smoke test

```bash
sudo GIT_SSH_COMMAND='ssh -i /home/deploy/.ssh/github_deploy' \
    bash deploy/scripts/deploy.sh --skip-build
```

Or manually hit the health endpoint:

```bash
curl -s https://app.zentria.tech/health | jq .
```

All services should report healthy.

### 8. Merge branch to main

After verification, merge the feature branch into main (or push directly if
working on main). Trigger CD if not auto-triggered.

---

## Rollback Path

If migration 000218 needs to be reverted:

```bash
# Run the down migration
migrate -path /migrations -database "$MIGRATION_DATABASE_URL" down 1
```

The `.down.sql` restores all three tables as plain (non-partitioned) tables,
copies data back from the partitioned parent (existing rows in surviving
partitions), and removes the pg_cron jobs. Data in already-dropped partitions
is lost (expected — they were beyond the 90-day retention window).

After rollback, revert the docker-compose.yml and docker-compose.prod.yml
changes and rebuild Postgres with the original `pgvector/pgvector:pg16` image.

---

## Ongoing Partition Management

After migration 000218 is applied and pg_cron is active:

| Job | Schedule | Action |
|-----|----------|--------|
| `partition-advance-events` | 25th of month 02:00 UTC | Creates partitions for next 2 months |
| `partition-retention-drop` | Daily 03:30 UTC | Drops partitions older than 90 days |

To check recent job runs:

```sql
SELECT jobname, runid, status, start_time, end_time, return_message
FROM cron.job_run_details
ORDER BY start_time DESC
LIMIT 20;
```

To manually trigger partition advance (e.g., after first deploy):

```sql
SELECT create_monthly_partition('events'::regclass,
       (date_trunc('month', now()) + interval '1 month')::date);
-- repeat for dialer_call_events and automation_executions
```

To manually drop old partitions:

```sql
SELECT drop_old_partitions('events', 90);
SELECT drop_old_partitions('dialer_call_events', 90);
SELECT drop_old_partitions('automation_executions', 90);
```
