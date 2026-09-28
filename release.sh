#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
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
    echo -e "${RED}❌ $1${NC}" >&2
    exit 1
}

# Check if git repo is clean (only tracked files)
check_git_status() {
    # Check for modified, staged, or deleted tracked files
    if [ -n "$(git status --porcelain | grep -E '^[MADR]')" ]; then
        error "Git working directory has uncommitted changes to tracked files. Please commit or stash your changes."
    fi

    # Show untracked files as info (but don't block)
    untracked=$(git status --porcelain | grep '^??' || true)
    if [ -n "$untracked" ]; then
        warning "Untracked files present (will be ignored):"
        echo "$untracked" | sed 's/^?? /  - /'
        echo ""
    fi
}

# Check if we're on main branch
check_main_branch() {
    current_branch=$(git branch --show-current)
    if [ "$current_branch" != "main" ]; then
        error "You must be on the main branch to create a release. Current branch: $current_branch"
    fi
}

# Check code formatting
check_formatting() {
    local component=$1

    info "Checking code formatting for $component..."

    cd "$component"
    if [[ "$component" =~ ^(backend|sync|collect)$ ]]; then
        # Go formatting check
        local unformatted=$(gofmt -s -l . | wc -l)
        if [ "$unformatted" -gt 0 ]; then
            error "Code is not formatted properly in $component. Run 'make fmt' to fix it."
        fi
    elif [ "$component" = "frontend" ]; then
        # Frontend formatting check
        if ! npm run format; then
            error "Code is not formatted properly in $component. Run 'npm run format-fix' to fix it."
        fi
    elif [[ "$component" =~ ^(proxy)$ ]]; then
        success "Skipping code formatting for $component (no formatting configured)"
    fi
    cd ..
    success "Code formatting OK for $component"
}

# Run linting
run_linting() {
    local component=$1

    info "Running linting for $component..."

    cd "$component"
    if [ "$component" = "backend" ] || [ "$component" = "collect" ]; then
        # Check if golangci-lint is available
        if command -v golangci-lint >/dev/null 2>&1; then
            if ! golangci-lint run; then
                error "Linting failed for $component"
            fi
        else
            warning "golangci-lint not found, skipping linting for $component"
            warning "Install with: https://golangci-lint.run/usage/install/"
        fi
    elif [ "$component" = "frontend" ]; then
        # Run frontend linting and type checking
        if ! npm run lint; then
            error "Linting failed for $component"
        fi
        if ! npm run type-check; then
            error "Type checking failed for $component"
        fi
    elif [[ "$component" =~ ^(proxy)$ ]]; then
        success "Skipping linting for $component (no linting configured)"
    else
        if ! make lint; then
            error "Linting failed for $component"
        fi
    fi
    cd ..
    success "Linting passed for $component"
}

# Run tests
run_tests() {
    local component=$1

    info "Running tests for $component..."

    cd "$component"
    if [ "$component" = "backend" ] || [ "$component" = "collect" ]; then
        if ! go test ./...; then
            error "Tests failed for $component"
        fi
    elif [ "$component" = "frontend" ]; then
        # Run frontend tests
        if ! npm run test:unit -- --run; then
            error "Tests failed for $component"
        fi
    elif [[ "$component" =~ ^(proxy)$ ]]; then
        success "Skipping tests for $component (no test suite)"
    else
        if ! make test; then
            error "Tests failed for $component"
        fi
    fi
    cd ..
    success "Tests passed for $component"
}

# The release must be the commit CI tested: the end-to-end verdicts below are
# read from GitHub Actions for this exact commit, and the push at the end
# must be a fast-forward.
check_up_to_date() {
    info "Checking main is in sync with origin/main..."

    if ! git fetch --quiet origin main; then
        error "Could not fetch origin/main"
    fi

    local local_sha remote_sha
    local_sha=$(git rev-parse HEAD)
    remote_sha=$(git rev-parse origin/main)
    if [ "$local_sha" = "$remote_sha" ]; then
        success "main is at origin/main (${local_sha:0:8})"
        return
    fi

    if git merge-base --is-ancestor HEAD origin/main; then
        error "Local main is behind origin/main. Run 'git pull' first."
    elif git merge-base --is-ancestor origin/main HEAD; then
        error "Local main has commits not on origin/main. Push them and let CI run on them first."
    else
        error "Local main and origin/main have diverged. Reconcile them first."
    fi
}

# Latest run of a workflow on main for a commit, as "id status conclusion url",
# or nothing when there is none.
ci_run_for() {
    local workflow=$1 sha=$2
    gh run list --workflow "$workflow" --branch main --commit "$sha" --limit 1 \
        --json databaseId,status,conclusion,url \
        -q '.[] | "\(.databaseId) \(.status) \(.conclusion) \(.url)"' \
        || error "Could not read the $workflow runs with gh. Check 'gh auth status'."
}

# True when every file a commit changes is one e2e-main.yml's paths-ignore
# skips ('**.md', 'docs/**'), i.e. a commit that never gets a fullstack run.
is_docs_only_commit() {
    local changed
    changed=$(git diff-tree --no-commit-id --name-only -r -m "$1")
    [ -n "$changed" ] && ! printf '%s\n' "$changed" | grep -qvE '(\.md$|^docs/)'
}

# Waits for a run to complete and requires it to have succeeded.
CI_WAIT_TIMEOUT=${CI_WAIT_TIMEOUT:-2700}
CI_POLL_INTERVAL=30
wait_for_ci_run() {
    local workflow=$1 label=$2 sha=$3
    local deadline=$(( $(date +%s) + CI_WAIT_TIMEOUT ))
    local run id status conclusion url

    while true; do
        run=$(ci_run_for "$workflow" "$sha")
        [ -n "$run" ] || error "No $label run for ${sha:0:8} on main."
        read -r id status conclusion url <<< "$run"

        [ "$status" = "completed" ] && break
        if [ "$(date +%s)" -ge "$deadline" ]; then
            error "The $label run for ${sha:0:8} is still $status after $((CI_WAIT_TIMEOUT / 60)) minutes: $url"
        fi
        info "The $label run for ${sha:0:8} is $status, waiting: $url"
        sleep "$CI_POLL_INTERVAL"
    done

    if [ "$conclusion" != "success" ]; then
        error "The $label run for ${sha:0:8} ended with '$conclusion': $url"
    fi
    CI_RUN_ID=$id
    CI_RUN_URL=$url
}

# The fullstack suite runs in CI on every push to main except docs-only ones,
# so the verdict for HEAD is that of the nearest commit that has a run, as
# long as everything after it is docs-only.
check_fullstack_run() {
    local sha
    sha=$(git rev-parse HEAD)
    info "Checking the CI fullstack run for ${sha:0:8}..."

    local candidate run skipped=0
    for candidate in $(git rev-list --first-parent --max-count=50 HEAD); do
        run=$(ci_run_for e2e-main.yml "$candidate")
        if [ -n "$run" ]; then
            wait_for_ci_run e2e-main.yml "fullstack" "$candidate"
            if [ "$skipped" -gt 0 ]; then
                success "CI fullstack run passed on ${candidate:0:8}, followed by $skipped docs-only commit(s): $CI_RUN_URL"
            else
                success "CI fullstack run passed: $CI_RUN_URL"
            fi
            return
        fi
        if ! is_docs_only_commit "$candidate"; then
            error "No CI fullstack run for ${candidate:0:8}, which changes more than docs. Run the E2E - Full Stack workflow on main, or pass --skip-tests."
        fi
        skipped=$((skipped + 1))
    done
    error "No CI fullstack run in the last 50 commits of main."
}

# The smoke suite targets the deployed QA environment rather than any
# checkout: e2e-smoke.yml runs it on every push to main once QA serves that
# commit. It runs for every commit, docs-only ones included.
check_smoke_run() {
    local sha
    sha=$(git rev-parse HEAD)
    info "Checking the QA smoke run for ${sha:0:8}..."

    local run
    run=$(ci_run_for e2e-smoke.yml "$sha")
    if [ -z "$run" ]; then
        error "No QA smoke run for ${sha:0:8}. Run the E2E - QA Smoke workflow on main, or pass --skip-tests."
    fi
    wait_for_ci_run e2e-smoke.yml "QA smoke" "$sha"

    # The workflow succeeds without testing anything when QA never came up
    # with this commit: it warns and skips the suite rather than failing.
    local ran
    ran=$(gh run view "$CI_RUN_ID" --json jobs \
        -q '[.jobs[].steps[] | select(.name == "Run the smoke suite") | .conclusion] | first // ""')
    if [ "$ran" != "success" ]; then
        error "The QA smoke run for ${sha:0:8} passed without running the suite (QA did not serve this commit in time): $CI_RUN_URL"
    fi
    success "QA smoke run passed: $CI_RUN_URL"
}

# Check the documentation site (Docusaurus)
run_docs_checks() {
    info "Running checks for docs..."

    cd docs
    if [ ! -d node_modules ]; then
        info "Installing documentation dependencies..."
        if ! make install; then
            error "Dependency install failed for docs"
        fi
    fi
    if ! make type-check; then
        error "Type checking failed for docs"
    fi
    # Docusaurus fails the build on broken links
    if ! make build; then
        error "Documentation build failed for docs"
    fi
    cd ..
    success "Checks passed for docs"
}

# Check every component for known dependency vulnerabilities
run_vulnerability_checks() {
    info "Checking dependencies for known vulnerabilities..."

    if ! ./vuln-check.sh --all; then
        error "Known vulnerabilities found. Bump the dependency (or add a package.json \"overrides\" entry), or record the accepted risk with its reason in .trivyignore."
    fi

    success "No known vulnerabilities above threshold"
}

# Get current version from version.json
get_current_version() {
    if [ ! -f "version.json" ]; then
        error "version.json not found"
    fi

    current_version=$(jq -r '.version' version.json)
    if [ "$current_version" = "null" ]; then
        error "Could not read version from version.json"
    fi
    echo "$current_version"
}

# Bump version based on type (patch, minor, major)
bump_version() {
    local current=$1
    local type=$2

    IFS='.' read -ra ADDR <<< "$current"
    major=${ADDR[0]}
    minor=${ADDR[1]}
    patch=${ADDR[2]}

    case $type in
        "patch")
            patch=$((patch + 1))
            ;;
        "minor")
            minor=$((minor + 1))
            patch=0
            ;;
        "major")
            major=$((major + 1))
            minor=0
            patch=0
            ;;
        *)
            error "Invalid bump type: $type. Use patch, minor, or major"
            ;;
    esac

    echo "$major.$minor.$patch"
}

# Update version.json with new version
update_version_file() {
    local new_version=$1

    jq --arg version "$new_version" '
        .version = $version |
        .components.backend = $version |
        .components.sync = $version |
        .components.collect = $version |
        .components.frontend = $version |
        .components.proxy = $version |
        .components."services/mimir" = $version
    ' version.json > version.json.tmp && mv version.json.tmp version.json
}

# Update individual VERSION files for Go components
update_component_versions() {
    local new_version=$1

    info "Updating component VERSION files..."

    # Update backend VERSION file
    if [ -f "backend/pkg/version/VERSION" ]; then
        echo "$new_version" > "backend/pkg/version/VERSION"
        success "Updated backend/pkg/version/VERSION"
    else
        warning "backend/pkg/version/VERSION not found"
    fi

    # Update collect VERSION file
    if [ -f "collect/pkg/version/VERSION" ]; then
        echo "$new_version" > "collect/pkg/version/VERSION"
        success "Updated collect/pkg/version/VERSION"
    else
        warning "collect/pkg/version/VERSION not found"
    fi

    # Update sync VERSION file
    if [ -f "sync/pkg/version/VERSION" ]; then
        echo "$new_version" > "sync/pkg/version/VERSION"
        success "Updated sync/pkg/version/VERSION"
    else
        warning "sync/pkg/version/VERSION not found"
    fi

    # Update services/mimir VERSION file
    if [ -f "services/mimir/VERSION" ]; then
        echo "$new_version" > "services/mimir/VERSION"
        success "Updated services/mimir/VERSION"
    else
        warning "services/mimir/VERSION not found"
    fi
}

# Update frontend package.json version
update_frontend_version() {
    local new_version=$1

    info "Updating frontend package.json version..."

    if [ -f "frontend/package.json" ]; then
        # Use jq to update the version in package.json
        jq --arg version "$new_version" '.version = $version' frontend/package.json > frontend/package.json.tmp && mv frontend/package.json.tmp frontend/package.json
        success "Updated frontend/package.json to version $new_version"
    else
        warning "frontend/package.json not found"
    fi
}

# Update OpenAPI specification version
update_openapi_version() {
    local new_version=$1

    info "Updating OpenAPI specification version..."

    if [ -f "backend/openapi.yaml" ]; then
        # Use sed to update only the version line in the info block (after title line)
        sed -i.bak '/^info:/,/^[a-zA-Z]/ s/^  version: .*/  version: '"$new_version"'/' backend/openapi.yaml
        rm -f backend/openapi.yaml.bak
        success "Updated backend/openapi.yaml version to $new_version"
    else
        warning "backend/openapi.yaml not found"
    fi
}

# Update documentation version
update_docs_version() {
    local new_version=$1

    info "Updating documentation version..."

    # English docs
    if [ -f "docs/docs/intro.md" ]; then
        sed -i.bak 's/Current version: \*\*[0-9.]*\*\*/Current version: **'"$new_version"'**/' docs/docs/intro.md
        rm -f docs/docs/intro.md.bak
        success "Updated docs/docs/intro.md version to $new_version"
    else
        warning "docs/docs/intro.md not found"
    fi

    # Italian docs
    if [ -f "docs/i18n/it/docusaurus-plugin-content-docs/current/intro.md" ]; then
        sed -i.bak 's/Versione corrente: \*\*[0-9.]*\*\*/Versione corrente: **'"$new_version"'**/' docs/i18n/it/docusaurus-plugin-content-docs/current/intro.md
        rm -f docs/i18n/it/docusaurus-plugin-content-docs/current/intro.md.bak
        success "Updated docs/i18n/it/docusaurus-plugin-content-docs/current/intro.md version to $new_version"
    else
        warning "docs/i18n/it/docusaurus-plugin-content-docs/current/intro.md not found"
    fi
}

# Show usage
usage() {
    echo "Usage: $0 [patch|minor|major] [--skip-tests]"
    echo ""
    echo "Bump version, commit, tag and push for release"
    echo ""
    echo "Options:"
    echo "  patch         Bump patch version (0.0.1 -> 0.0.2)"
    echo "  minor         Bump minor version (0.0.1 -> 0.1.0)"
    echo "  major         Bump major version (0.0.1 -> 1.0.0)"
    echo "  --skip-tests  Skip the unit tests and the CI end-to-end checks"
    echo "                (formatting, linting, docs and vulnerability checks still run)"
    echo ""
    echo "Local main must match origin/main. The E2E - Full Stack and E2E - QA Smoke"
    echo "runs for that commit must pass on GitHub Actions: the script waits for them"
    echo "(up to CI_WAIT_TIMEOUT seconds, default 2700) and reads them with gh."
    echo ""
    echo "Examples:"
    echo "  $0 patch               # For bug fixes"
    echo "  $0 minor               # For new features"
    echo "  $0 major               # For breaking changes"
    echo "  $0 patch --skip-tests  # Skip tests and CI checks"
    exit 1
}

# Main function
main() {
    # Check for jq dependency
    if ! command -v jq &> /dev/null; then
        error "jq is required but not installed. Install it with: brew install jq"
    fi

    # Parse arguments
    bump_type=""
    skip_tests=false
    for arg in "$@"; do
        case $arg in
            patch|minor|major)
                [ -z "$bump_type" ] || usage
                bump_type=$arg
                ;;
            --skip-tests)
                skip_tests=true
                ;;
            *)
                usage
                ;;
        esac
    done

    if [ -z "$bump_type" ]; then
        usage
    fi

    if [ "$skip_tests" = false ] && ! command -v gh &> /dev/null; then
        error "gh is required to read the CI end-to-end runs. Install it from https://cli.github.com, or pass --skip-tests."
    fi

    info "Starting release process..."

    # Pre-flight checks
    check_git_status
    check_main_branch
    check_up_to_date

    # Quality checks
    info "Running quality checks..."
    check_formatting "backend"
    check_formatting "sync"
    check_formatting "collect"
    check_formatting "frontend"
    check_formatting "proxy"
    run_linting "backend"
    run_linting "sync"
    run_linting "collect"
    run_linting "frontend"
    run_linting "proxy"
    if [ "$skip_tests" = true ]; then
        warning "Skipping unit and end-to-end tests (--skip-tests)"
    else
        run_tests "backend"
        run_tests "sync"
        run_tests "collect"
        run_tests "frontend"
        run_tests "proxy"
        check_fullstack_run
        check_smoke_run
    fi
    run_docs_checks
    run_vulnerability_checks
    success "All quality checks passed!"

    # Get current version and calculate new version
    current_version=$(get_current_version)
    new_version=$(bump_version "$current_version" "$bump_type")

    info "Current version: $current_version"
    info "New version: $new_version"

    # Confirm with user
    echo ""
    read -p "Do you want to create release v$new_version? [y/N] " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        warning "Release cancelled"
        exit 0
    fi

    # Update version file
    info "Updating version.json..."
    update_version_file "$new_version"

    # Update component VERSION files
    update_component_versions "$new_version"

    # Update frontend package.json version
    update_frontend_version "$new_version"

    # Update OpenAPI specification version
    update_openapi_version "$new_version"

    # Update documentation version
    update_docs_version "$new_version"

    # Commit changes
    info "Creating commit..."
    release_files=(
        version.json
        backend/pkg/version/VERSION
        collect/pkg/version/VERSION
        sync/pkg/version/VERSION
        services/mimir/VERSION
        frontend/package.json
        frontend/package-lock.json
        backend/openapi.yaml
        docs/docs/intro.md
        docs/i18n/it/docusaurus-plugin-content-docs/current/intro.md
    )
    for f in "${release_files[@]}"; do
        [ -f "$f" ] && git add "$f"
    done
    git commit -m "release: bump version to v$new_version"

    # Create tag
    info "Creating tag v$new_version..."
    git tag -a "v$new_version" -m "Release v$new_version"

    # Push to remote
    info "Pushing to remote..."
    git push origin main
    git push origin "v$new_version"

    success "Release v$new_version created successfully!"
    success "GitHub Actions will now build and publish the release."

    # Show useful links
    echo ""
    info "Useful links:"
    info "- Actions: https://github.com/$(git config remote.origin.url | sed 's/.*://g' | sed 's/.git$//g')/actions"
    info "- Releases: https://github.com/$(git config remote.origin.url | sed 's/.*://g' | sed 's/.git$//g')/releases"
}

# Run main function
main "$@"