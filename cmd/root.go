package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmdLongUsage = `
Helm plugin to manage outdated dependencies of a Helm chart.

Examples:
  $ helm outdated-dependencies list <pathToChart> 										- Checks if there's a newer version of any dependency available in the specified repository.
  $ helm outdated-dependencies list <pathToChart> --repositories repo1.corp,repo2.corp 	- Checks if there's a newer version of any dependency available only using the given repositories. 

  $ helm outdated-dependencies update <pathToChart> 							- Updates all outdated dependencies to the latest version found in the repository.
  $ helm outdated-dependencies update <pathToChart> --increment-chart-version	- Updates all outdated dependencies to the latest version found in the repository and increments the version of the Helm chart.
`

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:       "outdated-dependencies",
		Long:      rootCmdLongUsage,
		ValidArgs: []string{"chartPath"},
	}

	cmd.AddCommand(
		newListOutdatedDependenciesCmd(),
		newUpdateOutdatedDependenciesCmd(),
	)

	return cmd
}

func addCommonFlags(cmd *cobra.Command) {
	cmd.Flags().IntP("max-column-width", "w", 60, "Max column width to use for tables")
	cmd.Flags().StringSliceP("repositories", "r", []string{}, "Limit search to the given repository URLs. Can also just provide a part of the URL.")
}
