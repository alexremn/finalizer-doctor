#!/usr/bin/env bash
# Downloads the envtest control-plane binaries (kube-apiserver + etcd) and prints
# the path to export as KUBEBUILDER_ASSETS. Usage:
#   export KUBEBUILDER_ASSETS="$(hack/setup-envtest.sh)"
#   go test -tags integration ./internal/cluster/...
set -euo pipefail
cd "$(dirname "$0")/.."
go tool -modfile=tools/go.mod setup-envtest use -p path
