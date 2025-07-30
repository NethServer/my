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
        error "You must be on the main branch to deploy. Current branch: $current_branch"
    fi
}

# Get the latest git tag
get_latest_tag() {
    local latest_tag
    latest_tag=$(git describe --tags --abbrev=0 2>/dev/null)

    if [ -z "$latest_tag" ]; then
        error "No git tags found. Create a release first using './release.sh patch|minor|major'"
    fi

    echo "$latest_tag"
}

# Verify Docker images exist
verify_docker_images() {
    local version=$1

    info "Verifying Docker images exist for version $version..."

    for image in backend collect frontend proxy; do
        local image_url="ghcr.io/nethserver/my/$image:$version"
        info "Checking: $image_url"

        if docker manifest inspect "$image_url" >/dev/null 2>&1; then
            success "$image image found"
        else
            error "Image not found: $image_url. Please ensure the release workflow has completed successfully."
        fi
    done

    success "All Docker images verified successfully"
}

# Update render.yaml with new image tags
update_render_yaml() {
    local version=$1

    info "Updating render.yaml with version $version..."

    # Create backup
    cp render.yaml render.yaml.backup

    # Update backend image tag
    sed -i.tmp 's|ghcr\.io/nethserver/my/backend:v[0-9]*\.[0-9]*\.[0-9]*|ghcr.io/nethserver/my/backend:'"$version"'|g' render.yaml

    # Update collect image tag
    sed -i.tmp 's|ghcr\.io/nethserver/my/collect:v[0-9]*\.[0-9]*\.[0-9]*|ghcr.io/nethserver/my/collect:'"$version"'|g' render.yaml

    # Update frontend image tag
    sed -i.tmp 's|ghcr\.io/nethserver/my/frontend:v[0-9]*\.[0-9]*\.[0-9]*|ghcr.io/nethserver/my/frontend:'"$version"'|g' render.yaml

    # Update proxy image tag
    sed -i.tmp 's|ghcr\.io/nethserver/my/proxy:v[0-9]*\.[0-9]*\.[0-9]*|ghcr.io/nethserver/my/proxy:'"$version"'|g' render.yaml

    # Remove sed backup file
    rm -f render.yaml.tmp

    # Show changes
    info "Updated render.yaml with the following image references:"
    grep "url: ghcr.io/nethserver/my/" render.yaml | sed 's/^/  /'

    success "render.yaml updated successfully"
}

# Commit and push changes
commit_and_push() {
    local version=$1
    local git_user_name
    local git_user_email

    # Get git user info
    git_user_name=$(git config user.name)
    git_user_email=$(git config user.email)

    if [ -z "$git_user_name" ] || [ -z "$git_user_email" ]; then
        error "Git user name and email must be configured. Run: git config --global user.name 'Your Name' && git config --global user.email 'your@email.com'"
    fi

    info "Committing changes as $git_user_name <$git_user_email>..."

    # Check if there are changes to commit
    if git diff --quiet render.yaml; then
        warning "No changes to commit in render.yaml"
        return
    fi

    # Add and commit changes
    git add render.yaml
    git commit -m "deploy: update render.yaml to use $version images"

    # Push to remote
    info "Pushing to remote..."
    git push origin main

    success "Changes committed and pushed to main branch"
}

# Clean up backup file
cleanup() {
    if [ -f "render.yaml.backup" ]; then
        rm -f render.yaml.backup
        info "Cleaned up backup file"
    fi
}

# Show usage
usage() {
    echo "Usage: $0 [--skip-verify]"
    echo ""
    echo "Deploy the latest git tag to production by updating render.yaml"
    echo ""
    echo "Options:"
    echo "  --skip-verify    Skip Docker image verification (faster but less safe)"
    echo ""
    echo "This script will:"
    echo "  1. Get the latest git tag"
    echo "  2. Show the tag and ask for confirmation"
    echo "  3. Verify Docker images exist (unless --skip-verify)"
    echo "  4. Update render.yaml with new image tags"
    echo "  5. Commit and push changes"
    echo "  6. Render will automatically deploy the updated services"
    exit 1
}

# Main function
main() {
    local skip_verify=false

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --skip-verify)
                skip_verify=true
                shift
                ;;
            -h|--help)
                usage
                ;;
            *)
                echo "Unknown option: $1"
                usage
                ;;
        esac
    done

    info "Starting deployment process..."

    # Pre-flight checks
    check_git_status
    check_main_branch

    # Get latest tag
    latest_tag=$(get_latest_tag)

    info "Latest git tag: $latest_tag"

    # Confirm with user
    echo ""
    read -p "Do you want to deploy $latest_tag to production? [y/N] " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        warning "Deployment cancelled"
        exit 0
    fi

    # Set trap for cleanup
    trap cleanup EXIT

    # Verify images exist (unless skipped)
    if [ "$skip_verify" = false ]; then
        verify_docker_images "$latest_tag"
    else
        warning "Skipping Docker image verification"
    fi

    # Update render.yaml
    update_render_yaml "$latest_tag"

    # Commit and push
    commit_and_push "$latest_tag"

    success "Deployment initiated successfully!"
    success "render.yaml updated with $latest_tag images and pushed to main"
    success "Render will now automatically deploy the updated services"

    # Show useful links
    echo ""
    info "Monitor deployment progress at:"
    info "- Render Dashboard: https://dashboard.render.com"
    info "- Production URL: https://my.nethesis.it"

    echo ""
    info "Deployment Summary:"
    info "- Backend: ghcr.io/nethserver/my/backend:$latest_tag"
    info "- Collect: ghcr.io/nethserver/my/collect:$latest_tag"
    info "- Frontend: ghcr.io/nethserver/my/frontend:$latest_tag"
    info "- Proxy: ghcr.io/nethserver/my/proxy:$latest_tag"
}

# Run main function
main "$@"