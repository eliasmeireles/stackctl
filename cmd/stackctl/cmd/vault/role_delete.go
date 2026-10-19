package vault

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

// NewRoleDeleteCmd creates the "role delete" subcommand.
func NewRoleDeleteCmd() *cobra.Command {
	return NewRoleDeleteCmdFunc()
}

// NewRoleDeleteCmdFunc is a function variable for creating the role delete command.
var NewRoleDeleteCmdFunc = func() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <auth-mount> [role-name...]",
		Short: "Delete one or more roles",
		Long: `Delete roles under an auth method.

With only the auth mount in a terminal, check the roles of that mount from a
list. One confirmation is asked for the whole batch unless --yes is set or
stdin is not a terminal.

Examples:
  stackctl vault role delete auth/kubernetes
  stackctl vault role delete auth/kubernetes ci-kubeconfig
  stackctl vault role delete auth/approle ci-kubeconfig ci-deploy --yes`,
		Args:         cobra.MinimumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRoleDelete(cmd, args[0], args[1:], yes)
		},
	}

	addYesFlag(cmd, &yes)
	return cmd
}

func runRoleDelete(cmd *cobra.Command, mount string, args []string, yes bool) error {
	roles, err := resolveNames(args, pickSpec{
		required: "role name",
		title:    fmt.Sprintf("Select the roles to delete from %s", mount),
		empty:    fmt.Sprintf("no roles found at %s", mount),
		list:     func() ([]string, error) { return listRoles(mount) },
	})
	if err != nil {
		return err
	}
	if err := confirmBatch(cmd, yes, fmt.Sprintf("Delete roles from %s:", mount), roles); err != nil {
		return err
	}
	return runBulk(bulkSpec{
		title:    fmt.Sprintf("Deleted roles from %s", mount),
		column:   "NAME",
		okStatus: "deleted",
		action:   func(role string) error { return RoleClient.Delete(mount, role) },
	}, roles)
}

func listRoles(mount string) ([]string, error) {
	roles, err := RoleClient.List(mount)
	if err != nil {
		return nil, err
	}
	sort.Strings(roles)
	return roles, nil
}
