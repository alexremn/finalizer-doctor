# finalizer-doctor

[![release](https://img.shields.io/github/v/release/alexremn/finalizer-doctor?sort=semver)](https://github.com/alexremn/finalizer-doctor/releases)
[![CI](https://github.com/alexremn/finalizer-doctor/actions/workflows/ci.yml/badge.svg)](https://github.com/alexremn/finalizer-doctor/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/alexremn/finalizer-doctor)](https://goreportcard.com/report/github.com/alexremn/finalizer-doctor)
[![License](https://img.shields.io/github/license/alexremn/finalizer-doctor)](LICENSE)

A safe stuck-`Terminating` doctor for Kubernetes. It pinpoints the exact finalizer
and the dead controller / APIService blocking deletion, cleans truly-orphaned
resources first, and clears finalizers only as a gated last resort.

`kubectl finalizer-doctor` (alias: `kubectl fid`).

> ## ⚠️ This is a last resort — read this first
>
> Clearing a finalizer is **irreversible** and can orphan real infrastructure.
> finalizer-doctor exists to make that decision *safely*, not to make it *easy*:
>
> - **Dry-run by default.** It changes nothing unless you pass `--apply`.
> - **It refuses to guess.** A finalizer is only "safe to clear" when there is
>   *hard proof* the owning controller is gone. Anything it cannot read, or any
>   sign the controller is merely slow, blocks the verdict.
> - **It re-verifies before every irreversible action** and binds the `--apply`
>   confirmation to the exact state it showed you.
>
> If you just want the namespace gone and don't care what breaks, this is not the
> tool for you — a three-line snippet will do that, dangerously.

## Why

Namespaces (and other objects) hang in `Terminating` because a finalizer's
controller crashed or was uninstalled. The usual fixes — editing out finalizers,
`--grace-period=0 --force`, blindly emptying `spec.finalizers` — orphan real
infrastructure because nobody distinguishes *"controller gone, safe to clear"*
from *"controller just slow."* finalizer-doctor makes that distinction explicit
and evidence-based.

## Install

### Homebrew

```bash
brew install alexremn/tap/finalizer-doctor
```

This installs both `kubectl-finalizer_doctor` and `kubectl-fid` on your `PATH`, so
`kubectl finalizer-doctor` and `kubectl fid` work immediately.

### Standalone binaries

Download the archive for your platform from the
[releases page](https://github.com/alexremn/finalizer-doctor/releases) and place
both `kubectl-finalizer_doctor` and `kubectl-fid` on your `PATH`. `kubectl` will
then expose them as `kubectl finalizer-doctor` and `kubectl fid`.

### krew

```bash
kubectl krew install finalizer-doctor
```

## Usage

```bash
# Diagnose a stuck namespace (dry-run, human output)
kubectl finalizer-doctor ns/my-namespace

# Any finalizer-blocked resource
kubectl finalizer-doctor widgets.example.com/foo -n team-a

# Machine-readable output for CI
kubectl finalizer-doctor ns/my-namespace --output json

# Scan the whole cluster for stuck objects (read-only)
kubectl finalizer-doctor --all

# Apply (mutates) — requires the proof-bound digest the dry-run printed
kubectl finalizer-doctor ns/my-namespace --apply --confirm=<digest>
```

`--apply` cannot be combined with `--all`.

### Verdict modes

- `--verdict strict` (default): DEAD only on hard proof, with vetoes for live or
  unreadable signals. Time alone never yields DEAD.
- `--verdict score`: a transparent confidence readout that still requires a hard
  signal and passes the same vetoes — at least as safe as strict.

### Exit codes

| Code | Meaning |
|------|---------|
| `0` | nothing stuck, or apply cleared cleanly |
| `1` | operational error or invalid invocation (e.g. `--apply` with `--all`) |
| `2` | stuck object(s) found (report-mode signal for CI) |
| `3` | refused (SLOW/UNKNOWN, gate not satisfied, blocker, re-verify changed) |

## Safety model

The guardrails between a verdict and a mutation:

- **Dry-run by default** — nothing changes without `--apply`.
- **Evidence-based verdict** — a finalizer is only `DEAD` ("safe to clear") on hard
  proof the owner is gone; a live signal, an unreadable liveness source, or time
  alone never yields `DEAD`.
- **Proof-bound confirmation** — `--apply` (non-interactive) requires a `--confirm`
  digest that the dry-run prints, binding the token to the exact target, finalizer
  set, verdict, and `resourceVersion`.
- **Per-action re-verify** — the verdict is re-checked immediately before every
  irreversible action; a recovered controller aborts the run.
- **Ordered remediation** — clean true orphans first, refuse if a `failurePolicy=Fail`
  webhook with dead backing would reject the patch, then clear the finalizer last.
- **Targeted patch** — removes only the dead finalizer; namespaces use the
  `/finalize` subresource with a `resourceVersion` precondition; the namespace
  `spec` and `metadata` finalizers are gated jointly.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Security reports: [SECURITY.md](SECURITY.md).

## License

[Apache-2.0](LICENSE).
