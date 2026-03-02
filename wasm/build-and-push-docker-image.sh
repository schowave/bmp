#!/bin/bash

set -e

IMAGE=schowave/bmp

# Read version from VERSION file (single source of truth)
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
VERSION=$(cat "${SCRIPT_DIR}/../VERSION" | tr -d '[:space:]')

echo "Building WASM image version: wasm-${VERSION}"

# Detect container engine: prefer podman, fall back to docker
if command -v podman &> /dev/null; then
    ENGINE=podman
elif command -v docker &> /dev/null; then
    ENGINE=docker
else
    echo "Error: Neither podman nor docker found. Please install one of them."
    exit 1
fi

# Build the bundle first
cd "$SCRIPT_DIR" && make build

# Ensure we're logged in to Docker Hub
if ! $ENGINE login --get-login docker.io &> /dev/null; then
    echo "Not logged in to Docker Hub. Please log in:"
    $ENGINE login docker.io
fi

echo "Using $ENGINE to build and push image..."

if [ "$ENGINE" = "podman" ]; then
    podman build --platform linux/amd64 -t ${IMAGE}:wasm-${VERSION} "$SCRIPT_DIR"
    podman tag ${IMAGE}:wasm-${VERSION} ${IMAGE}:wasm
    podman push ${IMAGE}:wasm-${VERSION} docker://docker.io/${IMAGE}:wasm-${VERSION}
    podman push ${IMAGE}:wasm docker://docker.io/${IMAGE}:wasm
else
    docker build --tag ${IMAGE}:wasm-${VERSION} --tag ${IMAGE}:wasm "$SCRIPT_DIR"
    docker push ${IMAGE}:wasm-${VERSION}
    docker push ${IMAGE}:wasm
fi

echo "${IMAGE}:wasm-${VERSION} and ${IMAGE}:wasm built and pushed successfully."
