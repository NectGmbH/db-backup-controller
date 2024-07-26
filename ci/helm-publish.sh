#!/usr/bin/env bash
set -euo pipefail

function log() {
  echo "[$(date +%H:%M:%S)] $@" >&2
}

[[ -n ${GITHUB_REPOSITORY_OWNER:-} ]] || {
  echo "FATAL: Not executed in Github Actions." >&2
  exit 1
}

log "Patching chart version..."
version=$(git describe --tags --always)
if [[ $version =~ ^v ]]; then
  version=${version##v}
else
  version="0.0.0-${version}"
fi

yq -i -P \
  ".version = \"${version}\", .appVersion = \"${version}\"" \
  ./charts/db-backup-controller/Chart.yaml

log "Linting chart before publish..."
helm lint ./charts/db-backup-controller

log "Packaging chart..."
helm package ./charts/db-backup-controller

log "Pushing chart..."
repo="ghcr.io/${GITHUB_REPOSITORY_OWNER,,}"
helm push db-backup-controller-${version}.tgz oci://${repo}
