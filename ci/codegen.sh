#!/usr/bin/env bash
set -euo pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
CODEGEN_PKG=${SCRIPT_ROOT}/ci/code-generator
PKG=$(awk '/module /{ print $2 }' ${SCRIPT_ROOT}/go.mod)

export GOBIN=$(mktemp -d)
trap "rm -rf ${GOBIN}" EXIT

source "${CODEGEN_PKG}/kube_codegen.sh"

kube::codegen::gen_helpers \
  --boilerplate "${SCRIPT_ROOT}/hack/boilerplate.go.txt" \
  ./pkg

kube::codegen::gen_client \
  --boilerplate "${SCRIPT_ROOT}/hack/boilerplate.go.txt" \
  --output-dir "./pkg/generated" \
  --output-pkg "${PKG}/pkg/generated" \
  --with-watch \
  ./pkg
