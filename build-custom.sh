#!/bin/bash
set -e

IMAGE_NAME="ghcr.io/peter5842/sub2api"
TAG="${1:-latest}"

echo "Building custom Sub2API Docker image..."
docker build -t "${IMAGE_NAME}:${TAG}" .

echo "Build complete: ${IMAGE_NAME}:${TAG}"
echo ""
echo "To push to registry:"
echo "  docker push ${IMAGE_NAME}:${TAG}"
echo ""
echo "To test locally:"
echo "  cd deploy && docker-compose -f docker-compose.custom.yml up -d"
