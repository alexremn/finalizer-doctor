// Command finalizer-doctor is the kubectl plugin entrypoint; it delegates to
// internal/cli. The same binary is also distributed as kubectl-fid.
package main

import (
	"os"

	"github.com/alexremn/finalizer-doctor/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
