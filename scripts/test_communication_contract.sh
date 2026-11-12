#!/usr/bin/env bash
# Apply unchanged Laravel migration sources only to an isolated local cluster.
# The caller must start a disposable server and supply its loopback port.
set -euo pipefail
: "${COMMUNICATION_TEST_PORT:?set the disposable PostgreSQL port}"
: "${COMMUNICATION_TEST_USER:?set the disposable cluster administrator}"
: "${WEB3CODEX_REFERENCE_ROOT:?set the read-only Laravel project path}"
if [[ ! "$COMMUNICATION_TEST_PORT" =~ ^[0-9]+$ ]] || [[ "$COMMUNICATION_TEST_PORT" == 5432 ]]; then
  echo 'Use a dedicated disposable PostgreSQL port, not 5432.' >&2
  exit 1
fi
migration_dir="$WEB3CODEX_REFERENCE_ROOT/database/migrations/codex_zones/sql"
registry_sql="$migration_dir/2026_10_20_235826_communications.up.sql"
configuration_sql="$migration_dir/2026_10_21_200000_communication_configuration.up.sql"
coverage_sql="$migration_dir/2026_10_21_220000_communication_observation_coverage.up.sql"
channels_sql="$migration_dir/2026_10_27_000000_communication_channels.up.sql"
activation_sql="$migration_dir/2026_10_28_000000_communication_channel_activation.up.sql"
identity_sql="$migration_dir/2026_10_31_000000_communication_channel_identity.up.sql"
[[ -f "$registry_sql" && -f "$configuration_sql" && -f "$coverage_sql" && -f "$channels_sql" && -f "$activation_sql" && -f "$identity_sql" ]]
psql_args=(-X -v ON_ERROR_STOP=1 -h 127.0.0.1 -p "$COMMUNICATION_TEST_PORT" -U "$COMMUNICATION_TEST_USER")
# Fail if these already exist. This script never drops or replaces a database.
psql "${psql_args[@]}" postgres <<'SQL'
CREATE ROLE codex_zones_admin NOLOGIN;
CREATE ROLE codex_zones_runtime NOLOGIN;
CREATE DATABASE codexsecure_configuration_test OWNER codex_zones_admin;
SQL
# SQL equivalent of the PageMeta Blueprint migration (the first of four migrations).
# Production startup never executes this disposable bootstrap.
psql "${psql_args[@]}" codexsecure_configuration_test \
  -c 'SET ROLE codex_zones_admin' \
  -c "CREATE TABLE public.pagemeta_packages (uuid uuid PRIMARY KEY DEFAULT gen_random_uuid(), payload jsonb NOT NULL CHECK (jsonb_typeof(payload)='object'), hash varchar(128) NOT NULL CHECK (hash ~ '^[0-9a-f]{128}$'), tiu numeric(20,6) NOT NULL CHECK (tiu>0), created_at timestamptz(6) NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at timestamptz(6) NOT NULL DEFAULT CURRENT_TIMESTAMP); CREATE INDEX pagemeta_packages_hash_index ON public.pagemeta_packages(hash)" \
  -f "$registry_sql" -f "$configuration_sql"
psql "${psql_args[@]}" codexsecure_configuration_test \
  -f "$(dirname "$0")/../internal/communications/testdata/legacy_replay.sql" \
  -c 'SET ROLE codex_zones_admin' -f "$coverage_sql"
# Communication channels (C1/C2A), applied unchanged in one transaction as Laravel's migrator does.
psql "${psql_args[@]}" -1 codexsecure_configuration_test -c 'SET ROLE codex_zones_admin' -f "$channels_sql"
# Channel activation (C3A), likewise unchanged and in its own transaction.
psql "${psql_args[@]}" -1 codexsecure_configuration_test -c 'SET ROLE codex_zones_admin' -f "$activation_sql"
# Channel identity (C11-H: Site, Channel and User UUIDs), likewise unchanged and in its own transaction.
psql "${psql_args[@]}" -1 codexsecure_configuration_test -c 'SET ROLE codex_zones_admin' -f "$identity_sql"
export COMMUNICATION_TEST_DSN="host=127.0.0.1 port=$COMMUNICATION_TEST_PORT dbname=codexsecure_configuration_test user=$COMMUNICATION_TEST_USER sslmode=disable"
"${GO_BIN:-go}" test -race -tags integration ./internal/communications ./internal/channels ./internal/handlers -v
