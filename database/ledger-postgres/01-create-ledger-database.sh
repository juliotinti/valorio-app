#!/bin/sh
set -eu

psql \
  --set=ON_ERROR_STOP=1 \
  --username "$POSTGRES_USER" \
  --dbname postgres \
  --set=app_user="$LEDGER_APP_USER" \
  --set=app_password="$LEDGER_APP_PASSWORD" <<'SQL'
CREATE ROLE :"app_user"
  WITH LOGIN
  PASSWORD :'app_password'
  NOSUPERUSER
  NOCREATEDB
  NOCREATEROLE
  NOINHERIT
  NOREPLICATION
  NOBYPASSRLS;
CREATE DATABASE ledger_db OWNER :"app_user";
REVOKE ALL ON DATABASE ledger_db FROM PUBLIC;
GRANT CONNECT, TEMPORARY ON DATABASE ledger_db TO :"app_user";
SQL
