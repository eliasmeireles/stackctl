package kubeconfig

import (
	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/kubeconfig"
)

// NewRemoveCmd creates the remove subcommand.
func NewRemoveCmd() *cobra.Command {
	return newRemoveCmdFunc()
}

var newRemoveCmdFunc = func() *cobra.Command {
	var yes bool
	c := &cobra.Command{
		Use:     "remove [context-name...]",
		Aliases: []string{"delete", "rm"},
		Short:   "Remove contexts (and orphan clusters/users) from kubeconfig",
		Long: `Remove one or more contexts from the local kubeconfig. Clusters and users
no longer referenced by any context are removed too.

Without a context name in a terminal, check the contexts from a list.
In a terminal the batch is confirmed once (y/N) unless --yes is given.
Without a terminal (scripts, pipes) no confirmation is asked.

Every context is processed even when one fails; the command exits with an
error listing the failures.

Examples:
  stackctl kubeconfig remove
  stackctl kubeconfig remove home-lab
  stackctl kubeconfig remove home-lab dev-user@homelab --yes`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(cmd, args, yes)
		},
		ValidArgsFunction: completeLocalContexts,
	}
	c.Flags().BoolVarP(&yes, "yes", "y", false, "Skip the confirmation prompt")
	return c
}

func runRemove(cmd *cobra.Command, args []string, yes bool) error {
	if err := requireNamesOrTerminal(args, "context name"); err != nil {
		return err
	}
	names, err := namesOrPick(args, "Select the contexts to remove", "no local contexts found", localContextNames)
	if err != nil {
		return err
	}
	if !yes && isInteractiveTerminal() {
		if err := confirmNames(cmd, "Remove", names); err != nil {
			return err
		}
	}

	path := kubeconfig.GetPath()
	return batch{
		title: "Removed contexts",
		done:  "deleted",
		apply: func(name string) error { return kubeconfig.RemoveConfig(path, name) },
	}.run(names)
}

// completeLocalContexts completes local context names not yet given.
func completeLocalContexts(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	names, err := localContextNames()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return without(names, args), cobra.ShellCompDirectiveNoFileComp
}

func without(names, exclude []string) []string {
	skip := make(map[string]bool, len(exclude))
	for _, e := range exclude {
		skip[e] = true
	}
	out := make([]string, 0, len(names))
	for _, n := range names {
		if !skip[n] {
			out = append(out, n)
		}
	}
	return out
}
