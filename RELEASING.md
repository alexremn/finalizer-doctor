# Releasing

Releases are tag-driven and built by [GoReleaser](https://goreleaser.com) in the
`release` GitHub Actions workflow.

## Cut a release

```bash
git tag v0.1.0
git push origin v0.1.0
```

The workflow first runs the full CI gate (`ci.yml`: unit, lint, envtest
integration, kind e2e, govulncheck). GoReleaser then cross-compiles two binaries
(`kubectl-finalizer_doctor` and `kubectl-fid`) for linux/darwin/windows ×
amd64/arm64, creates a **draft** GitHub release with archives + checksums +
SBOMs, and updates the Homebrew cask.

## Publish

Review the draft, then publish it:

```bash
gh release edit v0.1.0 --draft=false
```

Publishing triggers the `krew` workflow: krew-release-bot opens the version
bump PR against `kubernetes-sigs/krew-index` from `.krew.yaml`.

## One-time prerequisites

Homebrew distribution publishes a cask to a separate tap repository:

1. A public repo **`alexremn/homebrew-tap`** (GoReleaser commits
   `Casks/finalizer-doctor.rb` into it).
2. A Personal Access Token with `repo` (or fine-grained contents:write) scope on
   `alexremn/homebrew-tap`, stored in **this** repo as the secret
   **`HOMEBREW_TAP_GITHUB_TOKEN`**. The default `GITHUB_TOKEN` cannot write to
   another repository.

Users install with:

```bash
brew install alexremn/tap/finalizer-doctor
kubectl krew install finalizer-doctor
```
