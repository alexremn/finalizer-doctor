#!/usr/bin/env bash
# Reproducible stuck-namespace demo for finalizer-doctor.
#
# Recreates the canonical real-world cause: a dead aggregated APIService whose
# backing Service is gone. The namespace controller can no longer enumerate that
# group, so `kubectl delete namespace` hangs in Terminating with
# NamespaceDeletionDiscoveryFailure. finalizer-doctor verdicts the `kubernetes`
# finalizer DEAD (dead aggregated API + no CRD + no content) and can clear it.
#
# Requires: a throwaway cluster (kind) and finalizer-doctor on PATH.
# Record with: asciinema rec demo.cast -c './hack/demo.sh'
set -euo pipefail

NS=demo-stuck
APISVC=v1beta1.metrics.example.com

echo "# 1. Register a dead aggregated APIService (backing service does not exist)"
kubectl apply -f - <<'EOF'
apiVersion: apiregistration.k8s.io/v1
kind: APIService
metadata:
  name: v1beta1.metrics.example.com
spec:
  group: metrics.example.com
  version: v1beta1
  groupPriorityMinimum: 100
  versionPriority: 100
  service:
    name: nonexistent
    namespace: kube-system
  insecureSkipTLSVerify: true
EOF

echo "# 2. Create a namespace and request deletion — it will hang"
kubectl create namespace "$NS"
kubectl delete namespace "$NS" --wait=false

echo "# 3. Confirm it is stuck Terminating"
sleep 3
kubectl get namespace "$NS"

echo "# 4. Diagnose (dry-run, nothing changes)"
kubectl finalizer-doctor "ns/$NS"

echo "# 5. Apply, using the proof-bound digest the dry-run printed above"
echo "#    kubectl finalizer-doctor ns/$NS --apply --confirm=<digest>"

echo "# 6. Cleanup the dead APIService so cluster discovery is healthy again"
echo "#    kubectl delete apiservice $APISVC"
