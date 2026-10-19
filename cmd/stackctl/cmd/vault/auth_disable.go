package vault

import (
	"github.com/spf13/cobra"
)

// NewAuthDisableCmd creates the "auth disable" subcommand.
func NewAuthDisableCmd() *cobra.Command {
	return NewAuthDisableCmdFunc()
}

// NewAuthDisableCmdFunc is a function variable for creating the auth disable command.
var NewAuthDisableCmdFunc = func() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "disable [path...]",
		Short: "Disable one or more auth methods",
		Long: `Disable auth methods at the given paths.

Without a path in a terminal, check the auth methods from a list (the
built-in token/ method is not offered). One confirmation is asked for the
whole batch unless --yes is set or stdin is not a terminal.

Examples:
  stackctl vault auth disable
  stackctl vault auth disable approle
  stackctl vault auth disable approle k8s-vps-01 --yes`,
		Args:         cobra.ArbitraryArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthDisable(cmd, args, yes)
		},
	}

	addYesFlag(cmd, &yes)
	return cmd
}

func runAuthDisable(cmd *cobra.Command, args []string, yes bool) error {
	paths, err := resolveNames(args, pickSpec{
		required: "auth method path",
		title:    "Select the auth methods to disable",
		empty:    "no auth methods to disable",
		list:     listDisableableAuthMethods,
	})
	if err != nil {
		return err
	}
	if err := confirmBatch(cmd, yes, "Disable auth methods", paths); err != nil {
		return err
	}
	return runBulk(bulkSpec{
		title:    "Disabled auth methods",
		column:   "PATH",
		okStatus: "disabled",
		action:   AuthMethodClient.Disable,
	}, paths)
}

func listDisableableAuthMethods() ([]string, error) {
	auths, err := AuthMethodClient.List()
	if err != nil {
		return nil, err
	}
	return withoutExcluded(mapKeys(auths), systemMounts), nil
}
