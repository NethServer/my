#!/bin/sh
set -eu

: "${GARAGE_RPC_SECRET:?must be a 32-byte hex string (openssl rand -hex 32)}"
: "${GARAGE_ADMIN_TOKEN:?must be set to a random secret}"

export GARAGE_RPC_SECRET GARAGE_ADMIN_TOKEN

mkdir -p /var/lib/garage/meta /var/lib/garage/data

envsubst < /etc/garage/garage.toml.template > /etc/garage/garage.toml

exec /garage -c /etc/garage/garage.toml server
