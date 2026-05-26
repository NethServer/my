#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
MIGRATION_DIR="$SCRIPT_DIR"

CONTAINER_ENGINE=""
POSTGRES_IMAGE="postgres:16-alpine"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info()    { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $1"; }

detect_container_engine() {
    if command -v docker >/dev/null 2>&1; then
        CONTAINER_ENGINE="docker"
    elif command -v podman >/dev/null 2>&1; then
        CONTAINER_ENGINE="podman"
    else
        log_error "Docker or Podman not found"
        exit 1
    fi
}

check_database_url() {
    if [ -z "$DATABASE_URL" ]; then
        log_error "DATABASE_URL not set"
        exit 1
    fi
}

find_migration_file() {
    local f
    f=$(find "$MIGRATION_DIR" -maxdepth 1 -type f -name "${1}_*.sql" ! -name "*_rollback.sql" | sort | head -1)
    echo "$f"
}

find_rollback_file() {
    local f
    f=$(find "$MIGRATION_DIR" -maxdepth 1 -type f -name "${1}_*_rollback.sql" | sort | head -1)
    echo "$f"
}

create_migrations_table() {
    run_psql_raw "
    CREATE TABLE IF NOT EXISTS schema_migrations (
        migration_number VARCHAR(10) PRIMARY KEY,
        applied_at TIMESTAMPTZ DEFAULT now(),
        description TEXT,
        checksum VARCHAR(64)
    );" >/dev/null
}

run_psql_raw() {
    $CONTAINER_ENGINE run --rm -i \
        --network=host \
        "$POSTGRES_IMAGE" \
        psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -c "$1"
}

run_psql_file() {
    set +e
    $CONTAINER_ENGINE run --rm -i \
        --network=host \
        "$POSTGRES_IMAGE" \
        psql "$DATABASE_URL" -v ON_ERROR_STOP=1 2>&1 | grep -v "NOTICE:"

    PSQL_RC=${PIPESTATUS[0]}
    set -e

    if [ "$PSQL_RC" -ne 0 ]; then
        exit $PSQL_RC
    fi
}

get_file_checksum() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | cut -d' ' -f1
    else
        shasum -a 256 "$1" | cut -d' ' -f1
    fi
}

check_migration_status() {
    create_migrations_table

    COUNT=$($CONTAINER_ENGINE run --rm -i \
        --network=host \
        "$POSTGRES_IMAGE" \
        psql "$DATABASE_URL" -t -A -c \
        "SELECT COUNT(*) FROM schema_migrations WHERE migration_number='$1';")

    [ "$COUNT" = "1" ]
}

get_stored_checksum() {
    $CONTAINER_ENGINE run --rm -i \
        --network=host \
        "$POSTGRES_IMAGE" \
        psql "$DATABASE_URL" -t -A -c \
        "SELECT COALESCE(checksum, '') FROM schema_migrations WHERE migration_number='$1';"
}

# Refuse to no-op if the on-disk file's checksum differs from what was recorded
# at apply time. Editing an applied migration silently lets the new content
# slip past CREATE TABLE IF NOT EXISTS and similar idempotent guards.
report_checksum_drift() {
    local mig="$1" file="$2"
    local stored actual
    stored=$(get_stored_checksum "$mig")
    actual=$(get_file_checksum "$file")

    if [ -z "$stored" ]; then
        log_warning "Migration $mig has no recorded checksum (legacy row)"
        return 0
    fi

    if [ "$stored" = "$actual" ]; then
        return 0
    fi

    log_error "Migration $mig drift: file changed after apply"
    log_error "  recorded: $stored"
    log_error "  on-disk:  $actual"
    log_error "Editing an applied migration is unsafe. Add a new migration with"
    log_error "the schema change, or roll this one back first."
    return 1
}

# baseline_if_schema_present: if schema_migrations is empty but the schema is
# clearly already in place (any non-bookkeeping table exists in public), mark
# every on-disk migration as applied without running it. This is how we keep
# the runner idempotent against the backend's database.Init() boot-time
# schema.sql apply — schema.sql already represents the cumulative head state,
# so re-running migrations against it is both redundant and dangerous (some
# migrations are not idempotent; e.g. 010 CREATE OR REPLACE VIEW fails when
# 012's MATERIALIZED VIEW already sits on the same name).
#
# ┌─────────────────────────────────────────────────────────────────────────┐
# │ INVARIANT: schema.sql MUST contain every CREATE TABLE / INDEX / VIEW    │
# │ that on-disk migrations produce. If you add a new migration without     │
# │ folding its effect into schema.sql, a fresh-init flow                   │
# │ (`make dev-up` → `make run` → `make db-migrate`) will silently mark     │
# │ your new migration as applied without ever creating the table — and     │
# │ the bug only shows up at runtime when a query hits the missing object.  │
# │ See backend/database/migrations/README.md.                              │
# └─────────────────────────────────────────────────────────────────────────┘
#
# Safe against partial migrations: only fires when schema_migrations has zero
# rows. As soon as anything is recorded (including by this function itself)
# subsequent calls no-op — adding a new migration to an existing DB runs it
# normally.
baseline_if_schema_present() {
    create_migrations_table

    local rowcount tablecount
    rowcount=$($CONTAINER_ENGINE run --rm -i \
        --network=host \
        "$POSTGRES_IMAGE" \
        psql "$DATABASE_URL" -t -A -c \
        "SELECT COUNT(*) FROM schema_migrations;")
    [ "$rowcount" = "0" ] || return 0

    # Count tables in the public schema other than schema_migrations itself.
    # >0 means schema.sql (or an earlier setup) already created the cumulative
    # schema; <=0 means a truly empty database where every migration must run.
    tablecount=$($CONTAINER_ENGINE run --rm -i \
        --network=host \
        "$POSTGRES_IMAGE" \
        psql "$DATABASE_URL" -t -A -c \
        "SELECT COUNT(*) FROM information_schema.tables
         WHERE table_schema = 'public' AND table_name <> 'schema_migrations';")
    [ "$tablecount" -gt 0 ] || return 0

    log_info "Schema present but schema_migrations is empty — baselining from on-disk migrations"

    # Redirect stdin to /dev/null inside the loop body: `podman run -i` (used
    # by run_psql_raw) consumes the pipe's stdin, so without this redirect the
    # loop reads only the first migration and the rest of the list is eaten by
    # podman. The classic stdin-leak-in-subprocess-inside-while-read bug.
    local mig file desc checksum
    while read -r mig; do
        [ -z "$mig" ] && continue
        file=$(find_migration_file "$mig")
        [ -z "$file" ] && continue
        desc=$(basename "$file" .sql | sed "s/^${mig}_//")
        checksum=$(get_file_checksum "$file")
        run_psql_raw "
            INSERT INTO schema_migrations (migration_number, description, checksum)
            VALUES ('$mig', '$desc', '$checksum')
            ON CONFLICT (migration_number) DO NOTHING;
        " </dev/null >/dev/null
    done < <(list_migration_files)

    log_success "Baselined schema_migrations from existing schema"
}

apply_migration() {
    MIG="$1"
    FILE=$(find_migration_file "$MIG")

    if [ -z "$FILE" ]; then
        log_error "Migration file for $MIG not found"
        exit 1
    fi

    # Detect "fresh from schema.sql" — backend's database.Init() applies the
    # baseline schema.sql which already contains every table in its
    # post-migration form, but schema_migrations is left empty. Without this
    # step we'd then try to replay 001..N against the already-populated schema
    # and fail on non-idempotent statements (e.g. 010's CREATE OR REPLACE VIEW
    # unified_organizations on top of 012's MATERIALIZED VIEW). Baseline the
    # bookkeeping once; subsequent ticks see "already applied" and no-op.
    baseline_if_schema_present

    if check_migration_status "$MIG"; then
        if ! report_checksum_drift "$MIG" "$FILE"; then
            exit 2
        fi
        log_warning "Migration $MIG already applied"
        return 0
    fi

    # Refuse to replay a migration when later siblings are already recorded.
    # Replaying is what triggers errors like "X is not a view" when a follow-up
    # migration has changed the underlying object's type.
    if ! detect_gaps; then
        exit 3
    fi

    CHECKSUM=$(get_file_checksum "$FILE" || true)
    DESC=$(basename "$FILE" .sql | sed "s/^${MIG}_//")

    log_info "Applying migration $MIG"

    {
        echo "BEGIN;"
        echo "SELECT pg_advisory_xact_lock(987654);"
        cat "$FILE"
        echo "INSERT INTO schema_migrations (migration_number, description, checksum)"
        echo "VALUES ('$MIG', '$DESC', '$CHECKSUM')"
        echo "ON CONFLICT (migration_number) DO NOTHING;"
        echo "COMMIT;"
    } | run_psql_file

    log_success "Migration $MIG applied"
}

rollback_migration() {
    MIG="$1"
    FILE=$(find_rollback_file "$MIG")

    if [ -z "$FILE" ]; then
        log_error "Rollback file for $MIG not found"
        exit 1
    fi

    if ! check_migration_status "$MIG"; then
        log_warning "Migration $MIG not applied"
        return 0
    fi

    log_info "Rolling back migration $MIG"

    {
        echo "BEGIN;"
        echo "SELECT pg_advisory_xact_lock(987654);"
        cat "$FILE"
        echo "DELETE FROM schema_migrations WHERE migration_number='$MIG';"
        echo "COMMIT;"
    } | run_psql_file

    log_success "Migration $MIG rolled back"
}

list_applied_migrations() {
    create_migrations_table
    $CONTAINER_ENGINE run --rm -i \
        --network=host \
        "$POSTGRES_IMAGE" \
        psql "$DATABASE_URL" -t -A \
        -c "SELECT migration_number FROM schema_migrations ORDER BY migration_number;" \
        | tr -d ' '
}

list_migration_files() {
    find "$MIGRATION_DIR" -maxdepth 1 -type f -name '*.sql' ! -name '*_rollback.sql' \
        | sed -E 's@.*/([0-9]+)_.*\.sql$@\1@' \
        | sort -u
}

# Detect numbers that have a file on disk and a newer applied sibling but no
# schema_migrations row. That state means apply will replay an already-applied
# migration and fail (e.g. CREATE OR REPLACE VIEW over a now-MATERIALIZED view).
detect_gaps() {
    local applied_file gaps_file max_applied
    applied_file=$(mktemp)
    gaps_file=$(mktemp)

    list_applied_migrations >"$applied_file"

    if [ ! -s "$applied_file" ]; then
        rm -f "$applied_file" "$gaps_file"
        return 0
    fi

    max_applied=$(tail -n1 "$applied_file")

    while read -r mig; do
        [ -z "$mig" ] && continue
        if [ "$(printf '%s\n' "$mig" "$max_applied" | sort | head -n1)" = "$mig" ] \
           && [ "$mig" != "$max_applied" ] \
           && ! grep -qx "$mig" "$applied_file"; then
            echo "$mig" >>"$gaps_file"
        fi
    done < <(list_migration_files)

    if [ -s "$gaps_file" ]; then
        local gap_list
        gap_list=$(tr '\n' ' ' <"$gaps_file" | sed 's/ $//')
        log_error "schema_migrations is inconsistent — applied rows skip: $gap_list"
        log_error "  but later migrations are recorded as applied (max=$max_applied)."
        log_error ""
        log_error "Either these were applied out-of-band (mark them) or never ran"
        log_error "(investigate before doing anything)."
        log_error ""
        log_error "If the schema is actually up to date, mark them as applied:"
        log_error "  ./run_migration.sh repair $gap_list"
        log_error "  (or from the backend dir: make db-repair MIGRATIONS=\"$gap_list\")"
        rm -f "$applied_file" "$gaps_file"
        return 1
    fi

    rm -f "$applied_file" "$gaps_file"
    return 0
}

repair_migrations() {
    if [ $# -eq 0 ]; then
        log_error "repair requires at least one migration number"
        log_error "Usage: $0 repair <num> [<num>...]"
        exit 1
    fi

    create_migrations_table

    local mig file desc checksum
    for mig in "$@"; do
        file=$(find_migration_file "$mig")
        if [ -z "$file" ]; then
            log_error "Migration file for $mig not found — refusing to repair"
            exit 1
        fi

        if check_migration_status "$mig"; then
            log_warning "Migration $mig is already recorded — skipping"
            continue
        fi

        desc=$(basename "$file" .sql | sed "s/^${mig}_//")
        checksum=$(get_file_checksum "$file")

        log_info "Marking migration $mig as applied (description=$desc)"
        run_psql_raw "
            INSERT INTO schema_migrations (migration_number, description, checksum)
            VALUES ('$mig', '$desc', '$checksum')
            ON CONFLICT (migration_number) DO UPDATE SET checksum = EXCLUDED.checksum;
        " >/dev/null
        log_success "Migration $mig marked as applied"
    done
}

verify_all() {
    create_migrations_table

    local mig stored file actual
    local ok=0 drift=0 missing_file=0 no_checksum=0

    while IFS='|' read -r mig stored; do
        mig="${mig// /}"
        stored="${stored// /}"
        [ -z "$mig" ] && continue

        file=$(find_migration_file "$mig")
        if [ -z "$file" ]; then
            log_warning "Migration $mig: applied but file not present (renamed?)"
            missing_file=$((missing_file + 1))
            continue
        fi

        if [ -z "$stored" ]; then
            log_warning "Migration $mig: applied but no checksum recorded"
            no_checksum=$((no_checksum + 1))
            continue
        fi

        actual=$(get_file_checksum "$file")
        if [ "$stored" = "$actual" ]; then
            ok=$((ok + 1))
        else
            log_error "Migration $mig: DRIFT (file edited after apply)"
            log_error "  recorded: $stored"
            log_error "  on-disk:  $actual"
            drift=$((drift + 1))
        fi
    done < <($CONTAINER_ENGINE run --rm -i \
        --network=host \
        "$POSTGRES_IMAGE" \
        psql "$DATABASE_URL" -t -A -F'|' \
        -c "SELECT migration_number, COALESCE(checksum,'') FROM schema_migrations ORDER BY migration_number;")

    log_info "verify: ok=$ok drift=$drift missing_file=$missing_file no_checksum=$no_checksum"
    [ "$drift" = "0" ]
}

show_usage() {
    echo "Usage: $0 <migration_number> {apply|rollback|status}"
    echo "       $0 verify"
    echo "       $0 repair <num> [<num>...]"
}

main() {
    detect_container_engine
    check_database_url

    if [ $# -ge 1 ] && [ "$1" = "verify" ]; then
        verify_all
        exit $?
    fi

    if [ $# -ge 1 ] && [ "$1" = "repair" ]; then
        shift
        repair_migrations "$@"
        exit $?
    fi

    if [ $# -ne 2 ]; then
        show_usage
        exit 1
    fi

    MIG="$1"
    ACTION="$2"

    case "$ACTION" in
        apply)    apply_migration "$MIG" ;;
        rollback) rollback_migration "$MIG" ;;
        status)
            if check_migration_status "$MIG"; then
                log_success "Migration $MIG is applied"
                exit 0
            else
                log_info "Migration $MIG is NOT applied"
                exit 1
            fi
            ;;
        *)
            show_usage
            exit 1
            ;;
    esac
}

main "$@"