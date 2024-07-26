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
# Drop colors from script output
: ${NOCOLOR:=}

CYAN="\e[36m"
YELLOW="\e[33m"
ENDCOLOR="\e[0m"

[[ -z $NOCOLOR ]] || {
  CYAN=""
  YELLOW=""
  ENDCOLOR=""
}

# Require SHA to work
[[ -n $GITHUB_SHA ]] || {
  echo "Missing GITHUB_SHA and cannot generate" >&2
  exit 1
}

# Get engine from parameters
ENGINE=${1:-}
[[ -n $ENGINE ]] || {
  echo "Usage: $0 <engine>" >&2
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

for df in $(find ./pkg/backupengine/${ENGINE}/ -name 'Dockerfile*'); do
  img_suffix=${ENGINE}

  img_ver=$(basename "${df}" | cut -d . -f 2)
  [[ $img_ver == Dockerfile ]] || img_suffix="${ENGINE}-${img_ver}"

  echo -e "${CYAN}Building '${controller_images[0]}-${img_suffix}-cache' from '${df}'...${ENDCOLOR}" >&2

  # Build once for the first image found with extra string added
  docker build \
    --build-arg "CONTROLLER_IMAGE=${controller_images[0]}" \
    -f "${df}" \
    -t "${controller_images[0]}-${img_suffix}-cache" \
    ./pkg/backupengine/${ENGINE}/

  # Tag for all collected image names and push
  for controller_image in "${controller_images[@]}"; do
    echo -e "${CYAN}Pushing '${controller_image}-${img_suffix}'...${ENDCOLOR}" >&2
    docker tag "${controller_images[0]}-${img_suffix}-cache" "${controller_image}-${img_suffix}"
    docker push "${controller_image}-${img_suffix}"
  done

  # Untag cache image
  docker rmi "${controller_images[0]}-${img_suffix}-cache"
done
