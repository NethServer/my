#!/bin/sh

echo '==> Environment Variables Debug:'
echo "PORT=$PORT"
echo "RENDER_SERVICE_NAME=$RENDER_SERVICE_NAME"
echo "IS_PULL_REQUEST=$IS_PULL_REQUEST"
echo "Original BACKEND_SERVICE_NAME=$BACKEND_SERVICE_NAME"
echo "Original COLLECT_SERVICE_NAME=$COLLECT_SERVICE_NAME"
echo "Original FRONTEND_SERVICE_NAME=$FRONTEND_SERVICE_NAME"

# Check if this is a PR preview environment
if [ "$IS_PULL_REQUEST" = "true" ]; then
    echo '==> PR Preview detected, adjusting service names...'
    
    # Extract PR suffix from RENDER_SERVICE_NAME (everything after "my-proxy-qa")
    PR_SUFFIX=$(echo "$RENDER_SERVICE_NAME" | sed 's/^my-proxy-qa//')
    echo "Extracted PR suffix: $PR_SUFFIX"
    
    # Apply PR suffix to all service names
    export BACKEND_SERVICE_NAME="${BACKEND_SERVICE_NAME}${PR_SUFFIX}"
    export COLLECT_SERVICE_NAME="${COLLECT_SERVICE_NAME}${PR_SUFFIX}"
    export FRONTEND_SERVICE_NAME="${FRONTEND_SERVICE_NAME}${PR_SUFFIX}"
    
    echo "Adjusted BACKEND_SERVICE_NAME=$BACKEND_SERVICE_NAME"
    echo "Adjusted COLLECT_SERVICE_NAME=$COLLECT_SERVICE_NAME"
    echo "Adjusted FRONTEND_SERVICE_NAME=$FRONTEND_SERVICE_NAME"
else
    echo '==> Not a PR preview, using original service names'
fi

# Extract DNS resolver and search domain from /etc/resolv.conf
# nginx resolver does not honor search domains, so service names must be FQDNs
RESOLVER=$(awk '/^nameserver/ {print $2; exit}' /etc/resolv.conf)
export RESOLVER="${RESOLVER:-8.8.8.8}"

SEARCH_DOMAIN=$(awk '/^search/ {print $2; exit}' /etc/resolv.conf)
if [ -n "$SEARCH_DOMAIN" ]; then
    export BACKEND_SERVICE_NAME="${BACKEND_SERVICE_NAME}.${SEARCH_DOMAIN}"
    export COLLECT_SERVICE_NAME="${COLLECT_SERVICE_NAME}.${SEARCH_DOMAIN}"
    export FRONTEND_SERVICE_NAME="${FRONTEND_SERVICE_NAME}.${SEARCH_DOMAIN}"
fi

echo "DNS resolver: $RESOLVER"
echo "Search domain: ${SEARCH_DOMAIN:-none}"

echo '==> Substituting nginx config...'
envsubst '$PORT $BACKEND_SERVICE_NAME $COLLECT_SERVICE_NAME $FRONTEND_SERVICE_NAME $RESOLVER' < /etc/nginx/nginx.conf > /tmp/nginx.conf

echo '==> Generated upstream URLs:'
grep -E 'set.*upstream' /tmp/nginx.conf || true

echo '==> Starting nginx...'
exec nginx -c /tmp/nginx.conf -g 'daemon off;'