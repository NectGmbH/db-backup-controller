#!/usr/bin/env bash
set -euo pipefail

# Define fallback SHA for non-Github Actions
: ${GITHUB_SHA:=$(git rev-parse HEAD)}
# Allow to override the default image name
# NOTE(kahlers): It ain't pretty but avoids collision with the Helm-
# Chart naming…
: ${IMAGE_BASE:=ghcr.io/nectgmbh/db-backup-controller-image}
# Allow to override the image fully
: ${IMAGE_NAME_OVERRIDE:=}

# Require SHA to work
[[ -n $GITHUB_SHA ]] || {
  echo "Missing GITHUB_SHA and cannot generate" >&2
  exit 1
}

# Assemble version to match helm-publish.sh
version=$(git describe --tags --always)
if [[ $version =~ ^v ]]; then
  version=${version##v}
else
  version="0.0.0-${version}"
fi

# Collect image names to push to
controller_images=()

if [[ -n $IMAGE_NAME_OVERRIDE ]]; then
  controller_images+=("${IMAGE_NAME_OVERRIDE}")
else
  controller_images+=("${IMAGE_BASE}:${version}")
fi

if [[ ${GITHUB_REF:-notgithub} =~ refs/tags/ ]]; then
  # We're on a tag, publish to latest
  controller_images+=("${IMAGE_BASE}:latest")
fi

# Build and push
for controller_image in "${controller_images[@]}"; do
  docker build -t "${controller_image}" .
  docker push "${controller_image}"
done
