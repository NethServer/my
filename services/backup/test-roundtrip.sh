#!/usr/bin/env bash
# End-to-end simulation of the NS8/NethSecurity backup upload flow against
# the locally running collect endpoint. Mirrors what send-cluster-backup /
# send-backup do on a real appliance: build a GPG-encrypted archive and
# upload it with HTTP Basic auth.
#
# Prerequisites:
#   1. services/backup/docker-compose.local.yml up (Garage + bootstrap)
#   2. collect running with BACKUP_S3_* pointing at localhost:13900
#   3. A registered system in the `my` database whose credentials are
#      exported below as SYSTEM_KEY / SYSTEM_SECRET
#   4. gpg and curl available on the host
#
# Usage:
#   export SYSTEM_KEY="my_sys_XXXXXXXXXXXXXXXX"
#   export SYSTEM_SECRET="my_XXXXXXXXXXXXXXXXXXXX.YYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYY"
#   export COLLECT_URL="http://localhost:8081"   # adjust if collect listens elsewhere
#   ./services/backup/test-roundtrip.sh

set -euo pipefail

: "${SYSTEM_KEY:?export SYSTEM_KEY before running}"
: "${SYSTEM_SECRET:?export SYSTEM_SECRET before running}"
: "${COLLECT_URL:=http://localhost:8081}"
: "${BACKUP_PASSPHRASE:=local-dev-passphrase}"

workdir="$(mktemp -d -t my-backup-sim-XXXXXX)"
trap 'rm -rf "$workdir"' EXIT

dump="$workdir/dump.json"
gz="$workdir/dump.json.gz"
gpg_file="$workdir/dump.json.gz.gpg"

echo "==> Writing a fake cluster dump to $dump"
cat > "$dump" <<'JSON'
{
  "simulation": true,
  "timestamp_unix": 0,
  "nodes": [
    {"id": 1, "hostname": "leader.example.test"},
    {"id": 2, "hostname": "worker1.example.test"}
  ],
  "subscription": {"provider": "nsent", "system_id": "sim"},
  "note": "this mimics dump.json produced by ns8-core/cluster/bin/cluster-backup"
}
JSON

echo "==> Compressing with gzip (same flags as cluster-backup)"
gzip -n -f "$dump"

echo "==> Encrypting with GPG symmetric AES-256 (same flags as appliances)"
gpg --batch --yes -c --pinentry-mode loopback \
    --cipher-algo AES256 \
    --passphrase "$BACKUP_PASSPHRASE" \
    "$gz"

size=$(stat -f%z "$gpg_file" 2>/dev/null || stat -c%s "$gpg_file")
sha=$(shasum -a 256 "$gpg_file" | awk '{print $1}')
echo "==> Encrypted payload: $gpg_file ($size bytes, sha256=$sha)"

echo "==> POST $COLLECT_URL/api/systems/backups with BasicAuth($SYSTEM_KEY:***)"
response=$(mktemp)
http_code=$(
  curl --silent --output "$response" --write-out '%{http_code}' \
       --user "$SYSTEM_KEY:$SYSTEM_SECRET" \
       --header "X-Filename: dump.json.gz.gpg" \
       --header "X-System-Version: simulated-ns8-3.0.0" \
       --header "Content-Type: application/octet-stream" \
       --data-binary "@$gpg_file" \
       "$COLLECT_URL/api/systems/backups"
)
echo "    HTTP $http_code"
cat "$response"
echo
rm "$response"

if [[ "$http_code" != "201" ]]; then
  echo "!! upload did not return 201; aborting inspection" >&2
  exit 1
fi

echo
echo "==> Listing objects in Garage via the bootstrap container"
podman exec backup-local-garage /garage -c /etc/garage/garage.toml \
    bucket info my-backups-dev || true

echo
echo "==> Listing backup objects with awscli (path-style)"
AWS_ACCESS_KEY_ID=backup-local-key \
AWS_SECRET_ACCESS_KEY=backup-local-secret-backup-local-secret-0000000000 \
aws --endpoint-url http://localhost:13900 \
    --region garage \
    s3 ls s3://my-backups-dev/ --recursive || \
  echo "(aws cli not installed — skip this check)"

echo
echo "Done. If everything looks right the GPG blob is now in Garage under"
echo "<org_id>/<system_id>/<uuidv7>.<ext>."
