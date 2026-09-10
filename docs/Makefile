# Use bash for shell commands (consistent with the other components)
SHELL := /bin/bash

# Dependency vulnerability check shared with every other component.
# Accepted risks are allowlisted in ../.trivyignore.
VULN_CHECK=../vuln-check.sh

# Default target
.PHONY: all
all: install build

# Install dependencies from the lockfile
.PHONY: install
install:
	@echo "Installing documentation dependencies..."
	@npm ci

# Start the documentation dev server (hot reload) on :3000
.PHONY: run
run:
	@echo "Starting documentation dev server..."
	@npm start

# Build the static site. Docusaurus fails the build on broken links,
# so this is the check that catches dead cross-references.
.PHONY: build
build:
	@echo "Building documentation..."
	@npm run build

# Serve the built site locally
.PHONY: serve
serve:
	@npm run serve

# Type-check the Docusaurus config, sidebars and React components
.PHONY: type-check
type-check:
	@echo "Type-checking documentation sources..."
	@npm run typecheck

# Regenerate the i18n translation files
.PHONY: translations
translations:
	@echo "Writing translation files..."
	@npm run write-translations

# Check dependencies for known vulnerabilities (npm audit, plus trivy when
# installed). This is the check that keeps the Dependabot and code scanning
# alerts on docs/package-lock.json from piling up.
.PHONY: audit
audit:
	@$(VULN_CHECK) docs

# Clean build output and the Docusaurus cache
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf build .docusaurus .cache-loader

# Pre-commit checks
.PHONY: pre-commit
pre-commit: type-check build audit
	@echo "All pre-commit checks passed!"

# Show help
.PHONY: help
help:
	@echo "User documentation (Docusaurus) - nethserver.github.io/my"
	@echo ""
	@echo "Available targets:"
	@echo "  audit         - Check dependencies for known vulnerabilities"
	@echo "  build         - Build the static site (fails on broken links)"
	@echo "  clean         - Clean build output and Docusaurus cache"
	@echo "  help          - Show this help"
	@echo "  install       - Install dependencies from the lockfile"
	@echo "  pre-commit    - Run all pre-commit checks (type-check + build + audit)"
	@echo "  run           - Start the dev server with hot reload"
	@echo "  serve         - Serve the built site locally"
	@echo "  translations  - Regenerate the i18n translation files"
	@echo "  type-check    - Type-check config, sidebars and components"
	@echo ""
	@echo "Vulnerabilities:"
	@echo "  Accepted risks are allowlisted in ../.trivyignore (CVE + GHSA id"
	@echo "  plus the reason). Thresholds: NPM_LEVEL (npm audit, default high),"
	@echo "  SEVERITY (trivy, default HIGH,CRITICAL)."
	@echo ""
	@echo "Note: the site is deployed automatically via GitHub Actions"
	@echo ""

# Default goal
.DEFAULT_GOAL := help
