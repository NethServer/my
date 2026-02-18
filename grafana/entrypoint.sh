#!/bin/sh

echo '==> Expanding Grafana provisioning templates...'

# Expand datasource template
envsubst '${BACKEND_SERVICE_NAME} ${MIMIR_SYSTEM_KEY} ${MIMIR_SYSTEM_SECRET}' \
  < /etc/grafana/provisioning/datasources/mimir.yaml.template \
  > /etc/grafana/provisioning/datasources/mimir.yaml

echo '==> Starting Grafana...'
exec /run.sh
