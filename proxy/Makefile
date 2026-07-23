CONTAINER_NAME = my-nginx
IMAGE = docker.io/library/nginx:alpine
INTERNAL_PORT = 8443
CONF = $(shell pwd)/nginx-dev.conf
CERT = $(shell pwd)/my.localtest.me+1.pem
KEY = $(shell pwd)/my.localtest.me+1-key.pem
FWD_PID = /tmp/my-nginx-fwd.pid

.PHONY: dev-setup dev-up dev-down dev-restart dev-logs dev-status

## Generate TLS certificates with mkcert (one-time setup)
dev-setup:
	@if [ -f $(CERT) ] && [ -f $(KEY) ]; then \
		echo "TLS certificates already exist"; \
	else \
		if ! command -v mkcert >/dev/null 2>&1; then \
			echo "Error: mkcert is not installed."; \
			echo "Install it with: brew install mkcert (macOS) or see https://github.com/FiloSottile/mkcert"; \
			exit 1; \
		fi; \
		echo "Installing local CA (if needed)..."; \
		mkcert -install 2>/dev/null || true; \
		echo "Generating TLS certificates..."; \
		cd $(shell pwd) && mkcert "my.localtest.me" "*.support.my.localtest.me"; \
		echo "Certificates generated: $(CERT) and $(KEY)"; \
	fi

## Start nginx dev proxy on port 443 (requires sudo for port forwarding)
dev-up: dev-setup
	@if podman ps --format '{{.Names}}' | grep -q '^$(CONTAINER_NAME)$$'; then \
		echo "$(CONTAINER_NAME) is already running"; \
	else \
		podman run -d --rm \
			--name $(CONTAINER_NAME) \
			-p $(INTERNAL_PORT):443 \
			-v $(CONF):/etc/nginx/nginx.conf:ro,z \
			-v $(CERT):/etc/nginx/certs/cert.pem:ro,z \
			-v $(KEY):/etc/nginx/certs/key.pem:ro,z \
			$(IMAGE) && \
		echo "$(CONTAINER_NAME) started on port $(INTERNAL_PORT)"; \
	fi
	@if [ -f $(FWD_PID) ] && sudo kill -0 $$(cat $(FWD_PID)) 2>/dev/null; then \
		echo "port 443 -> $(INTERNAL_PORT) forwarding is already active"; \
	else \
		echo "Starting port 443 -> $(INTERNAL_PORT) forwarding (requires sudo)..."; \
		sudo python3 $(shell pwd)/port-forward.py $(INTERNAL_PORT) 443 & \
		echo $$! > $(FWD_PID); \
		sleep 0.5; \
		echo "port 443 -> $(INTERNAL_PORT) forwarding active"; \
	fi

## Stop nginx dev proxy
dev-down:
	@if [ -f $(FWD_PID) ]; then \
		sudo kill $$(cat $(FWD_PID)) 2>/dev/null; \
		rm -f $(FWD_PID); \
		echo "port 443 forwarding stopped"; \
	fi
	@podman stop $(CONTAINER_NAME) 2>/dev/null && \
		echo "$(CONTAINER_NAME) stopped" || \
		echo "$(CONTAINER_NAME) is not running"

## Restart nginx dev proxy
dev-restart: dev-down dev-up

## Show nginx dev proxy logs
dev-logs:
	@podman logs -f $(CONTAINER_NAME)

## Show nginx dev proxy status
dev-status:
	@podman ps --filter name=$(CONTAINER_NAME) --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'
	@if [ -f $(FWD_PID) ] && sudo kill -0 $$(cat $(FWD_PID)) 2>/dev/null; then \
		echo "port 443 -> $(INTERNAL_PORT) forwarding: active (pid: $$(cat $(FWD_PID)))"; \
	else \
		echo "port 443 -> $(INTERNAL_PORT) forwarding: inactive"; \
	fi
