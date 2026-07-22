#!/bin/sh
set -eu

psql --set=ON_ERROR_STOP=1 \
  --username "$POSTGRES_USER" \
  --dbname "$POSTGRES_DB" \
  --set=platform_user="$PLATFORM_DB_USER" \
  --set=platform_password="$PLATFORM_DB_PASSWORD" \
  --set=platform_database="$PLATFORM_DB_NAME" \
  --set=newapi_user="$NEWAPI_DB_USER" \
  --set=newapi_password="$NEWAPI_DB_PASSWORD" \
  --set=newapi_database="$NEWAPI_DB_NAME" <<'SQL'
SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', :'platform_user', :'platform_password')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'platform_user') \gexec

SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', :'newapi_user', :'newapi_password')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'newapi_user') \gexec

SELECT format('CREATE DATABASE %I OWNER %I', :'platform_database', :'platform_user')
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = :'platform_database') \gexec

SELECT format('CREATE DATABASE %I OWNER %I', :'newapi_database', :'newapi_user')
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = :'newapi_database') \gexec
SQL
