#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
HACK_PKG=${HACK_PKG:-$(
  cd "${SCRIPT_ROOT}"
  ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator
)}
GO_PKG="github.com/eumel8/otc-rds-operator/pkg"

bash "${HACK_PKG}"/generate-groups.sh "all" \
  ${GO_PKG}/rds/v1alpha1/apis \
  ${GO_PKG} \
  rds:v1alpha1 \
  --go-header-file "${SCRIPT_ROOT}"/hack/boilerplate.go.txt
