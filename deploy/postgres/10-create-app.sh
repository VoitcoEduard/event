#!/bin/sh
set -eu
: "${APP_DB_NAME:?APP_DB_NAME is required}"
: "${APP_DB_USER:?APP_DB_USER is required}"
: "${APP_DB_PASSWORD:?APP_DB_PASSWORD is required}"
# psql variables quote identifiers and passwords as SQL values. The PostgreSQL
# image runs this once, only for an empty database volume.
psql --set ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres \
  --set app_user="$APP_DB_USER" --set app_password="$APP_DB_PASSWORD" --set app_db="$APP_DB_NAME" <<'SQL'
CREATE ROLE :"app_user" LOGIN PASSWORD :'app_password';
SELECT format('CREATE DATABASE %I OWNER %I', :'app_db', :'app_user') \gexec
SQL
