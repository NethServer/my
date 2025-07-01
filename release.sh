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
    echo -e "${RED}❌ $1${NC}"
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
    local unformatted=$(gofmt -s -l . | wc -l)
    if [ "$unformatted" -gt 0 ]; then
        error "Code is not formatted properly in $component. Run 'make fmt' to fix it."
    fi
    cd ..
    success "Code formatting OK for $component"
}

# Run linting
run_linting() {
    local component=$1
    info "Running linting for $component..."

    cd "$component"
    if [ "$component" = "backend" ]; then
        # Check if golangci-lint is available
        if command -v golangci-lint >/dev/null 2>&1; then
            if ! golangci-lint run; then
                error "Linting failed for $component"
            fi
        else
            warning "golangci-lint not found, skipping linting for $component"
            warning "Install with: https://golangci-lint.run/usage/install/"
        fi
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
    if [ "$component" = "backend" ]; then
        if ! go test ./...; then
            error "Tests failed for $component"
        fi
    else
        if ! make test; then
            error "Tests failed for $component"
        fi
    fi
    cd ..
    success "Tests passed for $component"
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
        .components.sync = $version
    ' version.json > version.json.tmp && mv version.json.tmp version.json
}

# Show usage
usage() {
    echo "Usage: $0 [patch|minor|major]"
    echo ""
    echo "Bump version, commit, tag and push for release"
    echo ""
    echo "Options:"
    echo "  patch    Bump patch version (0.0.1 -> 0.0.2)"
    echo "  minor    Bump minor version (0.0.1 -> 0.1.0)"
    echo "  major    Bump major version (0.0.1 -> 1.0.0)"
    echo ""
    echo "Examples:"
    echo "  $0 patch   # For bug fixes"
    echo "  $0 minor   # For new features"
    echo "  $0 major   # For breaking changes"
    exit 1
}

# Main function
main() {
    # Check for jq dependency
    if ! command -v jq &> /dev/null; then
        error "jq is required but not installed. Install it with: brew install jq"
    fi

    # Parse arguments
    if [ $# -eq 0 ]; then
        usage
    fi

    bump_type=$1

    if [[ ! "$bump_type" =~ ^(patch|minor|major)$ ]]; then
        usage
    fi

    info "Starting release process..."

    # Pre-flight checks
    check_git_status
    check_main_branch

    # Quality checks
    info "Running quality checks..."
    check_formatting "backend"
    check_formatting "sync"
    run_linting "backend"
    run_linting "sync"
    run_tests "backend"
    run_tests "sync"
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

    # Commit changes
    info "Creating commit..."
    git add version.json
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