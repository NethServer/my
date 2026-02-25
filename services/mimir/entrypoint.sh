#!/bin/sh

# Default PORT if not set
PORT=${PORT:-9009}
export PORT

echo "==> Expanding Mimir config..."
envsubst < /etc/mimir/my.yaml.template > /tmp/mimir-config.yaml

echo "==> Starting Mimir with alertmanager only for alert support on port ${PORT}..."
exec /bin/mimir -config.file=/tmp/mimir-config.yaml
