package vault

import (
	"github.com/spf13/cobra"
)

// NewEngineDisableCmd creates the "engine disable" subcommand.
func NewEngineDisableCmd() *cobra.Command {
	return NewEngineDisableCmdFunc()
}

// NewEngineDisableCmdFunc is a function variable for creating the engine disable command.
var NewEngineDisableCmdFunc = func() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "disable [path...]",
		Short: "Disable one or more secrets engines",
		Long: `Disable secrets engines at the given paths.

Without a path in a terminal, check the engines from a list (system mounts
such as sys/, identity/ and cubbyhole/ are not offered). One confirmation is
asked for the whole batch unless --yes is set or stdin is not a terminal.

Examples:
  stackctl vault engine disable
  stackctl vault engine disable transit
  stackctl vault engine disable transit pki --yes`,
		Args:         cobra.ArbitraryArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEngineDisable(cmd, args, yes)
		},
	}

	addYesFlag(cmd, &yes)
	return cmd
}

func runEngineDisable(cmd *cobra.Command, args []string, yes bool) error {
	paths, err := resolveNames(args, pickSpec{
		required: "engine path",
		title:    "Select the secrets engines to disable",
		empty:    "no secrets engines to disable",
		list:     listDisableableEngines,
	})
	if err != nil {
		return err
	}
	if err := confirmBatch(cmd, yes, "Disable secrets engines", paths); err != nil {
		return err
	}
	return runBulk(bulkSpec{
		title:    "Disabled secrets engines",
		column:   "PATH",
		okStatus: "disabled",
		action:   EngineClient.Disable,
	}, paths)
}

func listDisableableEngines() ([]string, error) {
	mounts, err := EngineClient.List()
	if err != nil {
		return nil, err
	}
	return withoutExcluded(mapKeys(mounts), systemMounts), nil
}
