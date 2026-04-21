CONTAINER_NAME = my-mimir
COMPOSE_FILE = $(shell pwd)/docker-compose.local.yml
MIMIR_URL = http://localhost:9009
REPO_ROOT = $(shell cd ../.. && pwd)
BACKEND_ENV = $(REPO_ROOT)/backend/.env
COLLECT_ENV = $(REPO_ROOT)/collect/.env
WEBHOOK_TOKEN = local-test-secret
# host.containers.internal works for podman; docker users may need host.docker.internal
WEBHOOK_URL = http://host.containers.internal:8081/api/alert_history

.PHONY: dev-setup dev-up dev-down dev-restart dev-logs dev-status dev-ready

## Inject Mimir and alerting webhook env vars into backend/.env and collect/.env
dev-setup:
	@echo "Configuring backend/.env..."
	@if [ ! -f $(BACKEND_ENV) ]; then \
		echo "  skipped: $(BACKEND_ENV) not found (run backend/make dev-setup first)"; \
	elif grep -q '^MIMIR_URL=' $(BACKEND_ENV); then \
		echo "  already configured (MIMIR_URL present)"; \
	else \
		printf '\n# ===========================================\n# MIMIR / ALERTING CONFIGURATION\n# ===========================================\nMIMIR_URL=%s\nALERTING_HISTORY_WEBHOOK_URL=%s\nALERTING_HISTORY_WEBHOOK_TOKEN=%s\n' \
			'$(MIMIR_URL)' '$(WEBHOOK_URL)' '$(WEBHOOK_TOKEN)' >> $(BACKEND_ENV); \
		echo "  added MIMIR_URL, ALERTING_HISTORY_WEBHOOK_URL, ALERTING_HISTORY_WEBHOOK_TOKEN"; \
	fi
	@echo "Configuring collect/.env..."
	@if [ ! -f $(COLLECT_ENV) ]; then \
		echo "  skipped: $(COLLECT_ENV) not found (run collect/make dev-setup first)"; \
	elif grep -q '^MIMIR_URL=' $(COLLECT_ENV); then \
		echo "  already configured (MIMIR_URL present)"; \
	else \
		printf '\n# ===========================================\n# MIMIR / ALERTING CONFIGURATION\n# ===========================================\nMIMIR_URL=%s\nALERTING_HISTORY_WEBHOOK_SECRET=%s\n' \
			'$(MIMIR_URL)' '$(WEBHOOK_TOKEN)' >> $(COLLECT_ENV); \
		echo "  added MIMIR_URL, ALERTING_HISTORY_WEBHOOK_SECRET"; \
	fi
	@echo "Done. Restart backend and collect to pick up the new env vars."

## Start Mimir locally with filesystem storage (no S3 required)
dev-up:
	@if podman ps --format '{{.Names}}' | grep -q '^$(CONTAINER_NAME)$$'; then \
		echo "$(CONTAINER_NAME) is already running"; \
	else \
		echo "Starting Mimir (alertmanager target, filesystem storage)..."; \
		podman-compose -f $(COMPOSE_FILE) up -d; \
		echo "Waiting for Mimir to be ready..."; \
		for i in 1 2 3 4 5 6 7 8 9 10; do \
			if curl -sf $(MIMIR_URL)/ready >/dev/null 2>&1; then \
				echo "$(CONTAINER_NAME) ready on $(MIMIR_URL)"; \
				exit 0; \
			fi; \
			sleep 1; \
		done; \
		echo "Warning: $(CONTAINER_NAME) did not become ready within 10s"; \
		exit 1; \
	fi

## Stop Mimir local instance
dev-down:
	@podman-compose -f $(COMPOSE_FILE) down 2>/dev/null && \
		echo "$(CONTAINER_NAME) stopped" || \
		echo "$(CONTAINER_NAME) is not running"

## Restart Mimir local instance
dev-restart: dev-down dev-up

## Show Mimir container logs (follow)
dev-logs:
	@podman logs -f $(CONTAINER_NAME)

## Show Mimir container status
dev-status:
	@podman ps --filter name=$(CONTAINER_NAME) --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'
	@if curl -sf $(MIMIR_URL)/ready >/dev/null 2>&1; then \
		echo "Mimir ready check: ✓ ($(MIMIR_URL)/ready)"; \
	else \
		echo "Mimir ready check: ✗ ($(MIMIR_URL)/ready)"; \
	fi

## Check Mimir readiness endpoint
dev-ready:
	@curl -sf $(MIMIR_URL)/ready && echo "" || (echo "Mimir not ready"; exit 1)
