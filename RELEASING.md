# Releasing

Releases are tag-driven and built by [GoReleaser](https://goreleaser.com) in the
`release` GitHub Actions workflow.

## Cut a release

```bash
git tag v0.1.0
git push origin v0.1.0
```

The workflow cross-compiles two binaries (`kubectl-finalizer_doctor` and
`kubectl-fid`) for linux/darwin/windows × amd64/arm64, publishes a draft GitHub
release with archives + checksums + SBOMs, and updates the Homebrew formula.

## One-time prerequisites

Homebrew distribution publishes a formula to a separate tap repository:

1. Create a public repo **`alexremn/homebrew-tap`** (GoReleaser commits
   `Formula/finalizer-doctor.rb` into it).
2. Create a Personal Access Token with `repo` (or fine-grained contents:write)
   scope on `alexremn/homebrew-tap`, and add it to **this** repo as the secret
   **`HOMEBREW_TAP_GITHUB_TOKEN`**. The default `GITHUB_TOKEN` cannot write to
   another repository, so the formula push needs this PAT.

After the first release, users install with:

```bash
brew install alexremn/tap/finalizer-doctor
```

## Note on the Homebrew formula

We use a GoReleaser **formula** (`brews:`) rather than a Cask because the tool
ships **two** binaries (`kubectl-finalizer_doctor` + `kubectl-fid`) and must work
on both macOS and Linux — a formula installs both cross-platform, whereas Casks
are macOS-only and awkward for multiple binaries. GoReleaser marks `brews:` as
deprecated, so the release workflow pins `goreleaser` to `~> v2` (where `brews:`
remains supported). Revisit when migrating to a future major.

## krew

The `.krew.yaml` manifest enables `kubectl krew install --manifest=.krew.yaml`
directly. krew-index submission is planned post-v1.
