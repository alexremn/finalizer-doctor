// Command finalizer-doctor is the kubectl plugin entrypoint; it delegates to
// internal/cli. The same binary is also distributed as kubectl-fid.
package main

import (
	"os"

	// Register client-go auth providers (OIDC, etc.) so the plugin works with
	// the full range of kubeconfig auth methods (krew best practice).
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/alexremn/finalizer-doctor/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
