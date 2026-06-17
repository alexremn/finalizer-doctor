#!/usr/bin/env bash
# Downloads the envtest control-plane binaries (kube-apiserver + etcd) and prints
# the path to export as KUBEBUILDER_ASSETS. Usage:
#   export KUBEBUILDER_ASSETS="$(hack/setup-envtest.sh)"
#   go test -tags integration ./internal/cluster/...
set -euo pipefail
go run sigs.k8s.io/controller-runtime/tools/setup-envtest@latest use -p path
