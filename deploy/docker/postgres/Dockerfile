FROM pgvector/pgvector:pg16

# Install pg_cron for automated partition management.
# pg_cron requires shared_preload_libraries=pg_cron in postgresql.conf —
# this is set via the postgres service command in docker-compose.yml.
RUN apt-get update \
    && apt-get install -y --no-install-recommends postgresql-16-cron \
    && rm -rf /var/lib/apt/lists/*
