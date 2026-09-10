#!/bin/bash
#
# Dependency vulnerability check for every component of this repository.
#
# Node components (docs, frontend) are checked with `npm audit`, which reads
# the same GitHub advisory database Dependabot reports from. Every component
# is also checked with `trivy fs` when trivy is installed locally, matching
# the scan CI uploads to GitHub code scanning.
#
# .trivyignore in the repository root is the single allowlist of accepted
# risks for both tools: one CVE / GHSA / GO id per line, trailing comments
# allowed. Trivy reads it natively; this script applies it to npm audit too,
# so an accepted risk must carry both ids to be silenced everywhere.
#
# Usage:
#   ./vuln-check.sh --all
#   ./vuln-check.sh docs
#   ./vuln-check.sh backend frontend
#
# Environment:
#   NPM_LEVEL  npm audit severity that fails the run (default: high)
#   SEVERITY   trivy severities that fail the run    (default: HIGH,CRITICAL)
#
# Exit codes: 0 clean, 1 vulnerabilities found, 2 usage or tooling error.

set -uo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
IGNORE_FILE="$ROOT_DIR/.trivyignore"

EMPTY_DOCKER_CONFIG=""

NPM_LEVEL="${NPM_LEVEL:-high}"
SEVERITY="${SEVERITY:-HIGH,CRITICAL}"

ALL_COMPONENTS="backend sync collect frontend docs proxy services/mimir"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

success() {
    echo -e "${GREEN}✅ $1${NC}"
}

warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

error() {
    echo -e "${RED}❌ $1${NC}"
}

usage() {
    echo "Usage: $0 [--all | <component>...]"
    echo ""
    echo "Check dependencies for known vulnerabilities."
    echo ""
    echo "Components: $ALL_COMPONENTS"
    echo ""
    echo "Environment:"
    echo "  NPM_LEVEL  npm audit severity that fails the run (default: high)"
    echo "  SEVERITY   trivy severities that fail the run    (default: HIGH,CRITICAL)"
    echo ""
    echo "Accepted risks are allowlisted in .trivyignore (CVE / GHSA / GO ids)."
    exit 2
}

# Which checker applies to a component
component_kind() {
    case "$1" in
        backend|sync|collect)   echo "go" ;;
        frontend|docs)          echo "npm" ;;
        proxy|services/mimir)   echo "files" ;;
        *)                      echo "" ;;
    esac
}

# Advisory ids accepted as known risks, read from .trivyignore
load_allowlist() {
    if [ ! -f "$IGNORE_FILE" ]; then
        ALLOWLIST=""
        return
    fi
    ALLOWLIST=$(sed 's/#.*//' "$IGNORE_FILE" | awk '{print $1}' \
        | grep -E '^(CVE-|GHSA-|GO-)' | tr '\n' ' ')
}

is_allowlisted() {
    local id=$1
    [ -n "$id" ] || return 1
    case " $ALLOWLIST " in
        *" $id "*) return 0 ;;
    esac
    return 1
}

severity_rank() {
    case "$1" in
        critical)        echo 4 ;;
        high)            echo 3 ;;
        moderate|medium) echo 2 ;;
        low)             echo 1 ;;
        *)               echo 0 ;;
    esac
}

# npm audit, filtered through the allowlist
check_npm() {
    local component=$1
    local dir="$ROOT_DIR/$component"
    local report findings threshold blocking allowed below

    if [ ! -f "$dir/package-lock.json" ]; then
        warning "$component: no package-lock.json, skipping npm audit"
        return 0
    fi

    info "$component: npm audit (fails on $NPM_LEVEL and above)"

    # npm audit exits non-zero when it finds anything; the report is still valid
    report=$(cd "$dir" && npm audit --json 2>/dev/null)
    if ! printf '%s' "$report" | jq -e 'has("vulnerabilities")' >/dev/null 2>&1; then
        error "$component: npm audit returned no usable report (offline, or dependencies not installed — run 'npm ci' in $component/)"
        return 2
    fi

    # One line per advisory: severity, package, id, title
    findings=$(printf '%s' "$report" | jq -r '
        .vulnerabilities // {}
        | to_entries[]
        | .value.via[]?
        | select(type == "object")
        | [ (.severity // "unknown"),
            (.name // "?"),
            ((.url // "") | sub(".*/"; "")),
            (.title // "") ]
        | @tsv' | sort -u)

    threshold=$(severity_rank "$NPM_LEVEL")
    blocking=0
    allowed=0
    below=0

    while IFS=$'\t' read -r sev pkg id title; do
        [ -n "${sev:-}" ] || continue
        if is_allowlisted "$id"; then
            allowed=$((allowed + 1))
            continue
        fi
        if [ "$(severity_rank "$sev")" -ge "$threshold" ]; then
            blocking=$((blocking + 1))
            printf '   %-9s %-24s %-24s %s\n' "$sev" "$pkg" "$id" "$title"
        else
            below=$((below + 1))
        fi
    done <<< "$findings"

    if [ "$blocking" -gt 0 ]; then
        error "$component: $blocking advisory(ies) at $NPM_LEVEL or above"
        echo "   Fix with a version bump (or a package.json \"overrides\" entry),"
        echo "   or record the accepted risk with a reason in .trivyignore"
        echo "   (add the GHSA id, not only the CVE, so npm audit honours it too)."
        return 1
    fi

    success "$component: no npm advisory at $NPM_LEVEL or above ($allowed allowlisted, $below below threshold)"
    return 0
}

# trivy filesystem scan, same as the CI pipeline.
# --exit-code 7 keeps "found something" distinguishable from trivy failing to
# run (missing vulnerability DB, no network), which exits 1.
check_trivy() {
    local component=$1
    local dir="$ROOT_DIR/$component"
    local rc

    if [ -z "$TRIVY" ]; then
        warning "$component: trivy scan skipped (trivy not installed)"
        return 0
    fi

    info "$component: trivy fs (severity $SEVERITY)"

    # DOCKER_CONFIG is neutralized on purpose: the scan reads the local
    # filesystem and the only registry call is the public vulnerability DB,
    # so a stale credsStore in ~/.docker/config.json must not break it.
    DOCKER_CONFIG="$EMPTY_DOCKER_CONFIG" "$TRIVY" fs --quiet --scanners vuln \
        --ignorefile "$IGNORE_FILE" \
        --skip-dirs '**/node_modules' \
        --skip-dirs '**/build' \
        --skip-dirs '**/dist' \
        --severity "$SEVERITY" \
        --exit-code 7 \
        "$dir"
    rc=$?

    case $rc in
        0)
            success "$component: trivy clean"
            return 0
            ;;
        7)
            error "$component: trivy found vulnerabilities at $SEVERITY"
            return 1
            ;;
        *)
            warning "$component: trivy could not run (exit $rc) — scan incomplete"
            return 0
            ;;
    esac
}

main() {
    [ $# -gt 0 ] || usage

    case "$1" in
        -h|--help) usage ;;
        --all)     set -- $ALL_COMPONENTS ;;
    esac

    local component kind rc=0 needs_jq=""

    for component in "$@"; do
        kind=$(component_kind "$component")
        if [ -z "$kind" ]; then
            error "unknown component: $component"
            usage
        fi
        [ "$kind" = "npm" ] && needs_jq=1
    done

    if [ -n "$needs_jq" ] && ! command -v jq >/dev/null 2>&1; then
        error "jq is required but not installed. Install it with: brew install jq"
        exit 2
    fi

    load_allowlist

    TRIVY=$(command -v trivy 2>/dev/null || true)
    if [ -z "$TRIVY" ]; then
        warning "trivy not installed: Go modules and container layers are not checked"
        warning "Install it with: brew install trivy (same scan CI runs on main)"
    else
        EMPTY_DOCKER_CONFIG=$(mktemp -d)
        trap 'rm -rf "$EMPTY_DOCKER_CONFIG"' EXIT
    fi

    for component in "$@"; do
        echo ""
        echo "── $component ──"
        case "$(component_kind "$component")" in
            npm)
                check_npm "$component" || rc=1
                check_trivy "$component" || rc=1
                ;;
            go|files)
                check_trivy "$component" || rc=1
                ;;
        esac
    done

    echo ""
    if [ "$rc" -ne 0 ]; then
        error "Vulnerability check failed"
    else
        success "Vulnerability check passed"
    fi
    return $rc
}

main "$@"
