#!/usr/bin/env sh

set -e

IMAGE="my-nethesis-ui:dist"

podman build --tag "$IMAGE" --target dist --force-rm --layers .
container_id=$(podman create my-nethesis-ui:dist /)
trap 'podman rm -f $container_id' EXIT
trap 'podman image rm -f $IMAGE' EXIT
mkdir -p dist
podman export "$container_id" | tar --extract --overwrite --directory dist
