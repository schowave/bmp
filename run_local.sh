#!/bin/bash
set -e

IMAGE_NAME="bmp"
CONTAINER_NAME="bmp"
PORT=8080
SAVEGAME_DIR="$(pwd)/savegame"

mkdir -p "$SAVEGAME_DIR"

# Stop and remove existing container if running
if podman ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    echo "Stopping existing container..."
    podman stop "$CONTAINER_NAME" 2>/dev/null || true
    podman rm "$CONTAINER_NAME" 2>/dev/null || true
fi

echo "Building container image..."
podman build -t "$IMAGE_NAME" .

echo "Starting container — open http://localhost:${PORT}"
podman run \
    --name "$CONTAINER_NAME" \
    -p "${PORT}:80" \
    -v "${SAVEGAME_DIR}:/savegame" \
    --rm \
    "$IMAGE_NAME"
