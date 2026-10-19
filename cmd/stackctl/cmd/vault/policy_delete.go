package vault

import (
	"github.com/spf13/cobra"
)

// NewPolicyDeleteCmd creates the "policy delete" subcommand.
func NewPolicyDeleteCmd() *cobra.Command {
	return NewPolicyDeleteCmdFunc()
}

// NewPolicyDeleteCmdFunc is a function variable for creating the policy delete command.
var NewPolicyDeleteCmdFunc = func() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete [name...]",
		Short: "Delete one or more policies",
		Long: `Remove policies from Vault.

Without a name in a terminal, check the policies from a list (the built-in
"root" and "default" policies are not offered). One confirmation is asked
for the whole batch unless --yes is set or stdin is not a terminal.

Examples:
  stackctl vault policy delete
  stackctl vault policy delete my-app-read
  stackctl vault policy delete my-app-read my-app-write --yes`,
		Args:         cobra.ArbitraryArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPolicyDelete(cmd, args, yes)
		},
	}

	addYesFlag(cmd, &yes)
	return cmd
}

func runPolicyDelete(cmd *cobra.Command, args []string, yes bool) error {
	names, err := resolveNames(args, pickSpec{
		required: "policy name",
		title:    "Select the policies to delete",
		empty:    "no deletable policies found",
		list:     listDeletablePolicies,
	})
	if err != nil {
		return err
	}
	if err := confirmBatch(cmd, yes, "Delete policies", names); err != nil {
		return err
	}
	return runBulk(bulkSpec{
		title:    "Deleted policies",
		column:   "NAME",
		okStatus: "deleted",
		action:   PolicyClient.Delete,
	}, names)
}

func listDeletablePolicies() ([]string, error) {
	policies, err := PolicyClient.List()
	if err != nil {
		return nil, err
	}
	return withoutExcluded(policies, builtinPolicies), nil
}
