#!/bin/sh
# Isolated disposable deployment, recreation and database-plus-media recovery.
set -eu
image=${1:-quiz-postgres-migration:local}
root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
tmp=$(mktemp -d)
project="quizsmoke$(date +%s)$$"
restore_project="${project}restore"
export COMPOSE_FILE="$tmp/compose.yml"
export SMOKE_IMAGE="$image" SMOKE_INIT="$root/deploy/postgres/10-create-app.sh"
cleanup() {
  result=$?
  trap - EXIT HUP INT TERM
  COMPOSE_PROJECT_NAME="$project" docker compose down -v --remove-orphans >/dev/null 2>&1 || true
  COMPOSE_PROJECT_NAME="$restore_project" docker compose down -v --remove-orphans >/dev/null 2>&1 || true
  rm -rf "$tmp"
  exit "$result"
}
trap cleanup EXIT HUP INT TERM
cat > "$COMPOSE_FILE" <<'YAML'
services:
  postgres:
    image: postgres:18.6
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: smoke-admin
      POSTGRES_DB: postgres
      APP_DB_NAME: quiz_dev
      APP_DB_USER: quiz_app
      APP_DB_PASSWORD: smoke-app
    volumes:
      - db:/var/lib/postgresql
      - ${SMOKE_INIT}:/docker-entrypoint-initdb.d/10-create-app.sh:ro
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -h 127.0.0.1 -U postgres -d postgres"]
      interval: 2s
      timeout: 3s
      retries: 30
  app:
    image: ${SMOKE_IMAGE}
    environment:
      DATABASE_URL: postgresql://quiz_app:smoke-app@postgres:5432/quiz_dev?sslmode=disable
      MEDIA_ROOT: /app/data/media
      APP_ENV: development
      AUTH_MODE: dev
      PUBLIC_BASE_URL: http://localhost:8081
      DEV_ADMIN_PASSWORD: smoke-password
    volumes:
      - media:/app/data/media
    depends_on:
      postgres:
        condition: service_healthy
volumes:
  db:
  media:
YAML
export COMPOSE_PROJECT_NAME="$project"
wait_ready() {
  for attempt in $(seq 1 60); do
    if docker compose exec -T app wget -q -O /dev/null http://127.0.0.1:8080/health/ready; then return 0; fi
    sleep 1
  done
  docker compose logs >&2
  return 1
}
docker compose up -d --wait
wait_ready
docker compose exec -T postgres psql -U postgres -d quiz_dev -v ON_ERROR_STOP=1 -c "INSERT INTO events (id,pin,name,created_at) VALUES ('d9aba17d-d55f-4cd6-8b28-aaf57ea3bad0','123456','smoke-event',now())"
docker compose exec -T app sh -c 'printf smoke-media > /app/data/media/smoke-file'
docker compose up -d --force-recreate app
wait_ready
"$root/scripts/backup.sh" "$tmp/backup"
export COMPOSE_PROJECT_NAME="$restore_project"
"$root/scripts/restore-empty.sh" "$tmp/backup"
wait_ready
count=$(docker compose exec -T postgres psql -U postgres -d quiz_dev -Atc "SELECT count(*) FROM events WHERE name='smoke-event'")
[ "$count" = 1 ]
[ "$(docker compose exec -T app cat /app/data/media/smoke-file)" = smoke-media ]
docker compose stop postgres
if docker compose exec -T app wget -q -O /dev/null http://127.0.0.1:8080/health/ready; then
  printf 'Readiness succeeded while PostgreSQL was stopped\n' >&2; exit 1
fi
docker compose start postgres
wait_ready
printf 'Fresh PostgreSQL deployment, recreation, readiness and restore passed.\n'
