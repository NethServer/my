#!/bin/sh

# Default PORT if not set
PORT=${PORT:-9009}
export PORT

# Default join members for single-node testing (same host)
MIMIR_JOIN_MEMBER1=${MIMIR_JOIN_MEMBER1:-localhost}
MIMIR_JOIN_MEMBER2=${MIMIR_JOIN_MEMBER2:-localhost}
export MIMIR_JOIN_MEMBER1
export MIMIR_JOIN_MEMBER2

echo "==> Expanding Mimir config..."
envsubst < /etc/mimir/my.yaml.template > /tmp/mimir-config.yaml

echo "==> Starting Mimir on port ${PORT}..."
echo "    Join members: ${MIMIR_JOIN_MEMBER1}:7946, ${MIMIR_JOIN_MEMBER2}:7946"
exec /bin/mimir --config.file=/tmp/mimir-config.yaml
