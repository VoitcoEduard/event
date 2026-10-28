#!/bin/sh
# COMPOSE_FILE selects the stack; run this on the Docker host.
set -eu
umask 077
if [ "$#" -ne 1 ]; then
  printf 'Usage: COMPOSE_FILE=... %s NEW_BACKUP_DIRECTORY\n' "$0" >&2
  exit 2
fi
backup_dir=$1
mkdir -m 700 "$backup_dir"
backup_dir=$(CDPATH= cd -- "$backup_dir" && pwd)
app_id=$(docker compose ps --all -q app)
if [ -z "$app_id" ]; then printf 'Application container not found\n' >&2; exit 1; fi
[ "$(printf '%s\n' "$app_id" | wc -l | tr -d ' ')" = 1 ] || { printf 'Expected one application container\n' >&2; exit 1; }
docker inspect --format '{{.Image}}' "$app_id" > "$backup_dir/app-image.txt"
date -u +%Y-%m-%dT%H:%M:%SZ > "$backup_dir/created-at.txt"
was_running=$(docker inspect --format '{{.State.Running}}' "$app_id")
resume_app() {
  result=$?
  trap - EXIT HUP INT TERM
  if [ "$was_running" = true ]; then
    docker compose start app >&2 || result=1
  fi
  exit "$result"
}
trap resume_app EXIT HUP INT TERM
if [ "$was_running" = true ]; then docker compose stop app >&2; fi

docker compose exec -T postgres sh -c \
  'exec pg_dump -U "$POSTGRES_USER" -d "$APP_DB_NAME" --format=custom' > "$backup_dir/database.dump"
docker compose exec -T postgres sh -c \
  'exec psql -U "$POSTGRES_USER" -d "$APP_DB_NAME" -Atc "SELECT version_id FROM goose_db_version WHERE is_applied ORDER BY id DESC LIMIT 1"' > "$backup_dir/schema-version.txt"
docker compose run --rm --no-deps -T --entrypoint tar app \
  -C /app/data/media -cf - . > "$backup_dir/media.tar"
printf 'Backup saved to %s\n' "$backup_dir"
