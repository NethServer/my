#!/bin/bash

# Database Migration Runner Script (Containerized)
# Usage: ./run_migration.sh [migration_number] [action]
#
# Examples:
#   ./run_migration.sh 001 apply    # Apply migration 001
#   ./run_migration.sh 001 rollback # Rollback migration 001
#   ./run_migration.sh 001 status   # Check migration status

set -e  # Exit on any error

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MIGRATION_DIR="$SCRIPT_DIR"

# Container configuration
CONTAINER_ENGINE=""
POSTGRES_IMAGE="postgres:16-alpine"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Detect container engine (Docker or Podman)
detect_container_engine() {
    if command -v docker >/dev/null 2>&1; then
        CONTAINER_ENGINE="docker"
        log_info "Using Docker as container engine"
    elif command -v podman >/dev/null 2>&1; then
        CONTAINER_ENGINE="podman"
        log_info "Using Podman as container engine"
    else
        log_error "Neither Docker nor Podman found. Please install one of them."
        exit 1
    fi
}

# Check if DATABASE_URL is set
check_database_url() {
    if [ -z "$DATABASE_URL" ]; then
        log_error "DATABASE_URL environment variable is not set"
        log_info "Set it like: export DATABASE_URL='postgres://user:password@localhost/dbname'"
        exit 1
    fi
}

# Find migration file by number
find_migration_file() {
    local migration_number=$1
    local pattern="${MIGRATION_DIR}/${migration_number}_*.sql"

    # Find migration file (exclude rollback files)
    local migration_file=$(ls $pattern 2>/dev/null | grep -v "_rollback.sql" | head -1)
    echo "$migration_file"
}

# Find rollback file by number
find_rollback_file() {
    local migration_number=$1
    local pattern="${MIGRATION_DIR}/${migration_number}_*_rollback.sql"

    local rollback_file=$(ls $pattern 2>/dev/null | head -1)
    echo "$rollback_file"
}

# Check if migration files exist
check_migration_files() {
    local migration_number=$1
    local migration_file=$(find_migration_file "$migration_number")
    local rollback_file=$(find_rollback_file "$migration_number")

    if [ -z "$migration_file" ] || [ ! -f "$migration_file" ]; then
        log_error "Migration file not found for migration $migration_number"
        exit 1
    fi

    if [ -z "$rollback_file" ] || [ ! -f "$rollback_file" ]; then
        log_error "Rollback file not found for migration $migration_number"
        exit 1
    fi
}

# Run psql command in container
run_psql() {
    local sql_command="$1"
    local input_file="$2"

    if [ -n "$input_file" ]; then
        # Run with input file - filter only NOTICE messages
        $CONTAINER_ENGINE run --rm -i \
            --network=host \
            -v "$MIGRATION_DIR:/migrations:ro" \
            "$POSTGRES_IMAGE" \
            psql "$DATABASE_URL" -f "/migrations/$(basename "$input_file")" 2>&1 | grep -v "NOTICE:"
    else
        # Run with SQL command - filter only NOTICE messages
        $CONTAINER_ENGINE run --rm -i \
            --network=host \
            "$POSTGRES_IMAGE" \
            psql "$DATABASE_URL" -c "$sql_command" 2>&1 | grep -v "NOTICE:"
    fi
}

# Run psql command in container with output
run_psql_output() {
    local sql_command="$1"
    local flags="$2"

    # Run psql and capture both output and exit code
    local temp_output=$(mktemp)
    if $CONTAINER_ENGINE run --rm -i \
        --network=host \
        "$POSTGRES_IMAGE" \
        psql "$DATABASE_URL" $flags -c "$sql_command" > "$temp_output" 2>&1; then
        # Success: filter NOTICE messages and return output
        grep -v "NOTICE:" "$temp_output"
    else
        # Error: show all output (including errors) and exit with error
        cat "$temp_output"
        rm "$temp_output"
        return 1
    fi
    rm "$temp_output"
}

# Create migrations table if it doesn't exist
create_migrations_table() {
    log_info "Creating migrations table if it doesn't exist..."
    run_psql "CREATE TABLE IF NOT EXISTS schema_migrations (
        migration_number VARCHAR(10) PRIMARY KEY,
        applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
        description TEXT,
        checksum VARCHAR(64)
    );" > /dev/null
    log_success "Migrations table ready"
}

# Calculate file checksum
get_file_checksum() {
    local file=$1
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$file" | cut -d' ' -f1
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$file" | cut -d' ' -f1
    else
        log_warning "No checksum utility found, using file size"
        stat -f%z "$file" 2>/dev/null || stat -c%s "$file"
    fi
}

# Check migration status
check_migration_status() {
    local migration_number=$1

    create_migrations_table || return 1

    local count_output
    if ! count_output=$(run_psql_output "SELECT COUNT(*) FROM schema_migrations WHERE migration_number = '$migration_number';" "-t"); then
        log_error "Failed to check migration status"
        return 1
    fi

    local count=$(echo "$count_output" | xargs)

    # Check if count is a number
    if ! [[ "$count" =~ ^[0-9]+$ ]]; then
        log_error "Invalid response from database: $count"
        return 1
    fi

    if [ "$count" -eq 1 ]; then
        local applied_at_output
        if ! applied_at_output=$(run_psql_output "SELECT applied_at FROM schema_migrations WHERE migration_number = '$migration_number';" "-t"); then
            log_error "Failed to get migration timestamp"
            return 1
        fi
        local applied_at=$(echo "$applied_at_output" | xargs)
        log_success "Migration $migration_number is applied (applied at: $applied_at)"
        return 0
    else
        log_info "Migration $migration_number is not applied"
        return 1
    fi
}

# Apply migration
apply_migration() {
    local migration_number=$1
    local migration_file=$(find_migration_file "$migration_number")

    # Check if already applied
    if check_migration_status "$migration_number" > /dev/null 2>&1; then
        log_warning "Migration $migration_number is already applied"
        return 0
    fi

    log_info "Applying migration $migration_number..."

    # Calculate checksum
    local checksum=$(get_file_checksum "$migration_file")

    # Extract description from filename (e.g., 001_add_applications.sql -> add_applications)
    local description=$(basename "$migration_file" .sql | sed "s/^${migration_number}_//")

    # Create temporary transaction script
    local temp_script=$(mktemp)
    cat > "$temp_script" <<EOF
BEGIN;

-- Apply the migration
\i /migrations/$(basename "$migration_file")

-- Record migration
INSERT INTO schema_migrations (migration_number, description, checksum)
VALUES ('$migration_number', '$description', '$checksum');

COMMIT;
EOF

    # Apply migration in container - filter only NOTICE messages, keep errors
    $CONTAINER_ENGINE run --rm -i \
        --network=host \
        -v "$MIGRATION_DIR:/migrations:ro" \
        -v "$temp_script:/tmp/migration.sql:ro" \
        "$POSTGRES_IMAGE" \
        psql "$DATABASE_URL" -f "/tmp/migration.sql" 2>&1 | grep -v "NOTICE:"

    # Cleanup
    rm "$temp_script"

    log_success "Migration $migration_number applied successfully"
}

# Rollback migration
rollback_migration() {
    local migration_number=$1
    local rollback_file=$(find_rollback_file "$migration_number")

    # Check if applied
    if ! check_migration_status "$migration_number" > /dev/null 2>&1; then
        log_warning "Migration $migration_number is not applied, nothing to rollback"
        return 0
    fi

    log_info "Rolling back migration $migration_number..."

    # Create temporary rollback script
    local temp_script=$(mktemp)
    cat > "$temp_script" <<EOF
BEGIN;

-- Apply the rollback
\i /migrations/$(basename "$rollback_file")

-- Remove migration record
DELETE FROM schema_migrations WHERE migration_number = '$migration_number';

COMMIT;
EOF

    # Apply rollback in container - filter only NOTICE messages, keep errors
    $CONTAINER_ENGINE run --rm -i \
        --network=host \
        -v "$MIGRATION_DIR:/migrations:ro" \
        -v "$temp_script:/tmp/rollback.sql:ro" \
        "$POSTGRES_IMAGE" \
        psql "$DATABASE_URL" -f "/tmp/rollback.sql" 2>&1 | grep -v "NOTICE:"

    # Cleanup
    rm "$temp_script"

    log_success "Migration $migration_number rolled back successfully"
}

# Show usage
show_usage() {
    echo "Database Migration Runner (Containerized)"
    echo ""
    echo "Usage: $0 [migration_number] [action]"
    echo ""
    echo "Actions:"
    echo "  apply      Apply the migration"
    echo "  rollback   Rollback the migration"
    echo "  status     Check migration status"
    echo ""
    echo "Examples:"
    echo "  $0 001 apply      # Apply migration 001"
    echo "  $0 001 rollback   # Rollback migration 001"
    echo "  $0 001 status     # Check if migration 001 is applied"
    echo ""
    echo "Requirements:"
    echo "  - Docker or Podman must be installed"
    echo "  - DATABASE_URL environment variable must be set"
    echo "  - Example: export DATABASE_URL='postgres://user:pass@host:5432/dbname'"
    echo ""
    echo "Note: This script uses containers to run PostgreSQL client commands,"
    echo "      so you don't need to install psql locally."
}

# Main script
main() {
    if [ $# -ne 2 ]; then
        show_usage
        exit 1
    fi

    local migration_number=$1
    local action=$2

    log_info "Database Migration Runner (Containerized) - Migration $migration_number, Action: $action"

    detect_container_engine
    check_database_url
    check_migration_files "$migration_number"

    case $action in
        "apply")
            apply_migration "$migration_number"
            ;;
        "rollback")
            rollback_migration "$migration_number"
            ;;
        "status")
            check_migration_status "$migration_number"
            ;;
        *)
            log_error "Unknown action: $action"
            show_usage
            exit 1
            ;;
    esac
}

# Run main function
main "$@"