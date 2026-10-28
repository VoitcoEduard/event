#!/bin/sh
# Restore into a new, empty database and media volume; never overwrite data.
set -eu
if [ "$#" -ne 1 ]; then
  printf 'Usage: COMPOSE_FILE=... %s BACKUP_DIRECTORY\n' "$0" >&2
  exit 2
fi
backup_dir=$(CDPATH= cd -- "$1" && pwd)
for f in app-image.txt schema-version.txt database.dump media.tar; do
  [ -s "$backup_dir/$f" ] || { printf 'Missing backup file %s\n' "$f" >&2; exit 1; }
done
app_id=$(docker compose ps --all -q app)
if [ -n "$app_id" ] && [ "$(docker inspect --format '{{.State.Running}}' "$app_id")" = true ]; then
  printf 'Stop the application before restoring.\n' >&2; exit 1
fi
docker compose up -d postgres >&2
ready=0
for attempt in $(seq 1 60); do
  if docker compose exec -T postgres pg_isready -h 127.0.0.1 -U postgres -d postgres >/dev/null 2>&1; then ready=1; break; fi
  sleep 1
done
[ "$ready" = 1 ] || { printf 'PostgreSQL did not become ready\n' >&2; exit 1; }
if [ -z "$app_id" ]; then docker compose create app >&2; app_id=$(docker compose ps --all -q app); fi
actual_image=$(docker inspect --format '{{.Image}}' "$app_id")
expected_image=$(cat "$backup_dir/app-image.txt")
[ "$actual_image" = "$expected_image" ] || { printf 'Application image does not match backup\n' >&2; exit 1; }
table_count=$(docker compose exec -T postgres sh -c \
  'psql -U "$POSTGRES_USER" -d "$APP_DB_NAME" -Atc "SELECT count(*) FROM information_schema.tables WHERE table_schema = '\''public'\''"')
[ "$table_count" = 0 ] || { printf 'Database is not empty; refusing restore\n' >&2; exit 1; }
docker compose run --rm --no-deps -T --entrypoint sh app -c \
  'test -z "$(find /app/data/media -mindepth 1 -print -quit)"' || \
  { printf 'Media volume is not empty; refusing restore\n' >&2; exit 1; }

docker compose exec -T postgres sh -c \
  'exec pg_restore --exit-on-error --no-owner --no-privileges --role="$APP_DB_USER" -U "$POSTGRES_USER" -d "$APP_DB_NAME"' < "$backup_dir/database.dump"
docker compose run --rm --no-deps -T --entrypoint tar app \
  -C /app/data/media -xf - < "$backup_dir/media.tar"
restored_version=$(docker compose exec -T postgres sh -c \
  'psql -U "$POSTGRES_USER" -d "$APP_DB_NAME" -Atc "SELECT version_id FROM goose_db_version WHERE is_applied ORDER BY id DESC LIMIT 1"')
[ "$restored_version" = "$(cat "$backup_dir/schema-version.txt")" ] || { printf 'Restored schema version differs from backup\n' >&2; exit 1; }
docker compose start app >&2
printf 'Restore complete; check /health/ready and verify event and media data.\n'
