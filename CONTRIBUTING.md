# Contributing to finalizer-doctor

Thanks for considering a contribution.

## Development

```bash
make test        # go test -race ./...
make cover       # coverage (target: >= 80% overall, ~100% for internal/verdict)
make lint        # golangci-lint
make build       # builds kubectl-finalizer_doctor and kubectl-fid into ./bin
make integration # envtest-backed adapter tests (needs setup-envtest)
make e2e         # kind-based end-to-end (needs a kind cluster)
```

This project is built with **Go 1.23** and a pure-pipeline architecture: the
domain logic in `internal/` is pure and table-tested; only `internal/cluster`
touches the Kubernetes API.

## Expectations

- **Test-driven.** New behavior comes with tests. The verdict and safety logic is
  the trust surface — cover it thoroughly.
- **Conventional Commits** for messages (`feat:`, `fix:`, `docs:`, `test:`,
  `chore:`, `refactor:`, `ci:`).
- Keep files small and focused; follow the existing package boundaries.
- Run `make test lint` before opening a PR.

## Verdict & safety changes

The verdict engine (`internal/verdict`) and the safety/apply logic
(`internal/apply`, `internal/plan`, `internal/webhook`) are the trust surface.
Changes there must be covered by tests and clearly described in the PR.

## Reporting security issues

Please do not open public issues for vulnerabilities — see [SECURITY.md](SECURITY.md).
