package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmdLongUsage = `
Helm plugin to manage outdated dependencies of a Helm chart.

Examples:
  $ helm outdated-dependencies list   <pathToChart> - Checks if there's a newer version of the dependency available in the specified repository.
  $ helm outdated-dependencies update <pathToChart> - Updates all outdated dependencies to the latest version found in the repository.
`

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "outdated-dependencies",
		Short: "outdeps",
		Long:  rootCmdLongUsage,
	}

	cmd.AddCommand(
		newListOutdatedDependenciesCmd(),
		newUpdateOutdatedDependenciesCmd(),
	)

	return cmd
}
