CONTAINER_NAME = backup-local-garage
COMPOSE_FILE = $(shell pwd)/docker-compose.local.yml
S3_ENDPOINT = http://localhost:13900
ADMIN_ENDPOINT = http://localhost:13903
BUCKET = my-backups-dev
ACCESS_KEY = backup-local-key
SECRET_KEY = backup-local-secret-backup-local-secret-0000000000
REPO_ROOT = $(shell cd ../.. && pwd)
BACKEND_ENV = $(REPO_ROOT)/backend/.env
COLLECT_ENV = $(REPO_ROOT)/collect/.env

.PHONY: dev-setup dev-up dev-down dev-restart dev-logs dev-status dev-ready \
        dev-shell dev-objects dev-reset test-roundtrip

## Inject BACKUP_S3_* env vars into backend/.env and collect/.env
dev-setup:
	@echo "Configuring backend/.env..."
	@if [ ! -f $(BACKEND_ENV) ]; then \
		echo "  skipped: $(BACKEND_ENV) not found (run backend/make dev-setup first)"; \
	elif grep -q '^BACKUP_S3_ENDPOINT=' $(BACKEND_ENV); then \
		echo "  already configured (BACKUP_S3_ENDPOINT present)"; \
	else \
		printf '\n# ===========================================\n# BACKUP STORAGE (local Garage)\n# ===========================================\nBACKUP_S3_ENDPOINT=%s\nBACKUP_S3_REGION=garage\nBACKUP_S3_BUCKET=%s\nBACKUP_S3_ACCESS_KEY=%s\nBACKUP_S3_SECRET_KEY=%s\nBACKUP_S3_USE_PATH_STYLE=true\n' \
			'$(S3_ENDPOINT)' '$(BUCKET)' '$(ACCESS_KEY)' '$(SECRET_KEY)' >> $(BACKEND_ENV); \
		echo "  added BACKUP_S3_* entries"; \
	fi
	@echo "Configuring collect/.env..."
	@if [ ! -f $(COLLECT_ENV) ]; then \
		echo "  skipped: $(COLLECT_ENV) not found (run collect/make dev-setup first)"; \
	elif grep -q '^BACKUP_S3_ENDPOINT=' $(COLLECT_ENV); then \
		echo "  already configured (BACKUP_S3_ENDPOINT present)"; \
	else \
		printf '\n# ===========================================\n# BACKUP STORAGE (local Garage)\n# ===========================================\nBACKUP_S3_ENDPOINT=%s\nBACKUP_S3_REGION=garage\nBACKUP_S3_BUCKET=%s\nBACKUP_S3_ACCESS_KEY=%s\nBACKUP_S3_SECRET_KEY=%s\nBACKUP_S3_USE_PATH_STYLE=true\n' \
			'$(S3_ENDPOINT)' '$(BUCKET)' '$(ACCESS_KEY)' '$(SECRET_KEY)' >> $(COLLECT_ENV); \
		echo "  added BACKUP_S3_* entries"; \
	fi
	@echo "Done. Restart backend and collect to pick up the new env vars."

## Start local Garage (S3-compatible object storage for dev)
dev-up:
	@if podman ps --format '{{.Names}}' | grep -q '^$(CONTAINER_NAME)$$'; then \
		echo "$(CONTAINER_NAME) is already running"; \
	else \
		echo "Starting Garage (single node, local volume)..."; \
		podman-compose -f $(COMPOSE_FILE) up -d; \
		echo "Waiting for Garage to be ready..."; \
		for i in 1 2 3 4 5 6 7 8 9 10 11 12; do \
			if podman exec $(CONTAINER_NAME) /garage -c /etc/garage/garage.toml status >/dev/null 2>&1; then \
				echo "$(CONTAINER_NAME) ready on $(S3_ENDPOINT)"; \
				exit 0; \
			fi; \
			sleep 1; \
		done; \
		echo "Warning: $(CONTAINER_NAME) did not become ready within 12s"; \
		exit 1; \
	fi

## Stop local Garage instance
dev-down:
	@podman-compose -f $(COMPOSE_FILE) down 2>/dev/null && \
		echo "$(CONTAINER_NAME) stopped" || \
		echo "$(CONTAINER_NAME) is not running"

## Restart local Garage instance
dev-restart: dev-down dev-up

## Show Garage container logs (follow)
dev-logs:
	@podman logs -f $(CONTAINER_NAME)

## Show Garage container status and cluster health
dev-status:
	@podman ps --filter name=$(CONTAINER_NAME) --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'
	@if podman exec $(CONTAINER_NAME) /garage -c /etc/garage/garage.toml status >/dev/null 2>&1; then \
		echo "Garage cluster check: OK"; \
	else \
		echo "Garage cluster check: FAILED"; \
	fi

## Check Garage admin health endpoint
dev-ready:
	@curl -sf $(ADMIN_ENDPOINT)/health >/dev/null && echo "Garage admin OK" || (echo "Garage admin not ready"; exit 1)

## Open an interactive garage CLI shell inside the container
dev-shell:
	@podman exec -it $(CONTAINER_NAME) /bin/sh

## List objects currently stored in the backup bucket
dev-objects:
	@podman exec $(CONTAINER_NAME) /garage -c /etc/garage/garage.toml \
		bucket info $(BUCKET) || true

## Wipe the local Garage volumes (destructive — all objects are lost)
dev-reset:
	@podman-compose -f $(COMPOSE_FILE) down -v
	@echo "Garage volumes removed. Next 'dev-up' starts with an empty cluster."

## Run the end-to-end appliance simulation against the local stack
test-roundtrip:
	@./test-roundtrip.sh
