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

apply_migration() {
    MIG="$1"
    FILE=$(find_migration_file "$MIG")

    if [ -z "$FILE" ]; then
        log_error "Migration file for $MIG not found"
        exit 1
    fi

    if check_migration_status "$MIG"; then
        log_warning "Migration $MIG already applied"
        return 0
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

show_usage() {
    echo "Usage: $0 <migration_number> {apply|rollback|status}"
}

main() {
    if [ $# -ne 2 ]; then
        show_usage
        exit 1
    fi

    MIG="$1"
    ACTION="$2"

    detect_container_engine
    check_database_url

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