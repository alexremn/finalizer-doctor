// Package cli wires cobra commands and orchestrates the pipeline.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Build info, set via -ldflags at release time.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print build information",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "finalizer-doctor %s (commit %s, built %s)\n", Version, Commit, Date)
			return err
		},
	}
}
