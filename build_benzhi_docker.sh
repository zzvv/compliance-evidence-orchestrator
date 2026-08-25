#!/bin/bash
set -e
IMAGE_NAME=${1:-compliance-evidence-orchestrator}
DOCKER_PLATFORM=${2:-linux/amd64}
docker build --platform "$DOCKER_PLATFORM" -f benzhi.Dockerfile -t "$IMAGE_NAME" .
