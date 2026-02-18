#!/bin/sh

# Default PORT if not set
PORT=${PORT:-9009}
export PORT

echo "==> Expanding Mimir config..."
envsubst < /etc/mimir/my.yaml.template > /tmp/mimir-config.yaml

echo "==> Starting Mimir alertmanager on port ${PORT}..."
exec /bin/mimir -target=alertmanager -config.file=/tmp/mimir-config.yaml
