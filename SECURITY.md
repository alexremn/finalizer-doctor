# Security Policy

## Reporting a vulnerability

Please report security vulnerabilities **privately** via GitHub Security
Advisories: <https://github.com/alexremn/finalizer-doctor/security/advisories/new>.

Do **not** open a public issue for a suspected vulnerability.

We aim to acknowledge a report within 5 business days and to provide a remediation
timeline after triage.

## Scope

finalizer-doctor mutates Kubernetes objects (clearing finalizers, deleting
orphans) only behind an explicit `--apply` gate. Reports of particular interest:

- Any path that mutates the cluster without satisfying the safety gate.
- Any path that renders a `DEAD` / "safe to clear" verdict without hard proof, or
  that bypasses the cannot-probe / live-signal vetoes.
- Any path that bypasses the per-action re-verify or the `resourceVersion`
  precondition.

## Supported versions

The latest released minor version receives security fixes.
