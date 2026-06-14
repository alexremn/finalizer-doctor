package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/alexremn/finalizer-doctor/internal/cluster"
)

// Execute builds the root command, runs it, and returns the process exit code.
func Execute() int {
	var code int
	root := newRootCmd(&code)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		if code == 0 {
			code = 1
		}
	}
	return code
}

func newRootCmd(code *int) *cobra.Command {
	o := Options{}
	var kubeconfig, kcontext string

	cmd := &cobra.Command{
		Use:           "finalizer-doctor <target>",
		Short:         "Safely diagnose and clear finalizers on stuck-Terminating resources",
		Args:          cobra.ArbitraryArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !o.All {
				if len(args) != 1 {
					*code = 1
					return &InvalidInvocation{Msg: "exactly one <target> is required (or use --all)"}
				}
				o.Target = args[0]
			}
			client, err := buildClient(kubeconfig, kcontext)
			if err != nil {
				*code = 1
				return err
			}
			o.Interactive = isTTY()

			// Interactive apply: show the fresh dry-run, then require the typed name.
			if o.Apply && o.Interactive {
				dry := o
				dry.Apply = false
				dry.Interactive = false
				if dout, _, derr := Run(cmd.Context(), client, dry); derr == nil {
					fmt.Fprint(cmd.OutOrStdout(), dout)
				}
				fmt.Fprint(cmd.ErrOrStderr(), "Type the resource name to confirm: ")
				line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
				o.TypedName = strings.TrimSpace(line)
			}

			out, c, err := Run(cmd.Context(), client, o)
			*code = c
			fmt.Fprint(cmd.OutOrStdout(), out)
			return err
		},
	}

	f := cmd.Flags()
	f.BoolVar(&o.All, "all", false, "scan the whole cluster for stuck objects (read-only)")
	f.BoolVar(&o.Apply, "apply", false, "enable mutation (default off -> dry-run/explain)")
	f.StringVar(&o.Confirm, "confirm", "", "proof-bound digest from the dry-run (required for non-interactive apply)")
	f.StringVar(&o.Verdict, "verdict", "strict", "verdict strategy: strict|score")
	f.StringVar(&o.Output, "output", "human", "output format: human|json")
	f.StringVarP(&o.Namespace, "namespace", "n", "", "namespace for namespaced targets")
	f.StringVar(&kubeconfig, "kubeconfig", "", "path to the kubeconfig file")
	f.StringVar(&kcontext, "context", "", "kube context to use")

	cmd.AddCommand(newVersionCmd())
	return cmd
}

func buildClient(kubeconfig, kcontext string) (cluster.ClusterClient, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfig != "" {
		rules.ExplicitPath = kubeconfig
	}
	overrides := &clientcmd.ConfigOverrides{}
	if kcontext != "" {
		overrides.CurrentContext = kcontext
	}
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("kube config: %w", err)
	}
	return cluster.NewFromConfig(cfg)
}

func isTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
