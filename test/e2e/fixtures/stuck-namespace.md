# e2e fixture: a stuck-Terminating namespace

The `e2e` build-tagged test (`test/e2e/e2e_test.go`) expects a namespace named
`e2e-stuck` left in `Terminating` by a dead controller. Setup (run by the CI e2e
job against a `kind` cluster):

```bash
# 1. Install a CRD + a controller whose finalizer guards its CRs.
kubectl apply -f widget-crd.yaml
kubectl apply -f widget-controller.yaml      # adds finalizer example.com/cleanup

# 2. Create the namespace and a guarded CR inside it.
kubectl create namespace e2e-stuck
kubectl -n e2e-stuck apply -f widget-instance.yaml

# 3. Kill the controller so its finalizer can never be removed.
kubectl delete -f widget-controller.yaml
# (or, to exercise the namespace classic case, delete the backing APIService)

# 4. Delete the namespace — it now hangs Terminating.
kubectl delete namespace e2e-stuck --wait=false

# 5. Build the plugin and run the test.
go build -o test/e2e/kubectl-finalizer_doctor ./cmd/finalizer-doctor
go test -tags e2e ./test/e2e/ -run TestDiagnoseStuckNamespace -v
```

The YAML manifests referenced above (`widget-crd.yaml`, `widget-controller.yaml`,
`widget-instance.yaml`) are intentionally not committed yet; add minimal ones when
wiring the CI e2e job. The test asserts the tool finds the stuck object and emits a
verdict (`DEAD`/`SLOW`/`UNKNOWN`) in JSON.
