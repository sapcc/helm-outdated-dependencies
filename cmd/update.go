package cmd

import (
	"fmt"
	"github.com/gosuri/uitable"
	"github.com/sapcc/helm-outdated-dependencies/pkg/helm"
	"github.com/spf13/cobra"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"path/filepath"
)

type (
	updateCmd struct {
		chartPath               string
		helmSettings            *helm_env.EnvSettings
		maxColumnWidth          uint
		indent                  int
		isIncrementChartVersion bool
		repositories []string
	}
)

var updateLongUsage = `
Helm plugin to manage outdated dependencies of a Helm chart.

Examples:
  $ helm outdated-dependencies update 
  $ helm outdated-dependencies update <chartPath>
`

func newUpdateOutdatedDependenciesCmd() *cobra.Command {
	u := &updateCmd{
		helmSettings: &helm_env.EnvSettings{
			Home: helm.GetHelmHome(),
		},
		maxColumnWidth: 60,
		repositories: []string{},
	}

	cmd := &cobra.Command{
		Use:          "update",
		Long:         listLongUsage,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if maxColumnWidth, err := cmd.Flags().GetUint("max-column-width"); err == nil {
				u.maxColumnWidth = maxColumnWidth
			}

			if repositories, err := cmd.Flags().GetStringSlice("repositories"); err == nil {
				u.repositories = repositories
			}

			path := "."
			if len(args) > 0 {
				path = args[0]
			}

			path, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			u.chartPath = path
			return u.update()
		},
	}

	addCommonFlags(cmd)
	cmd.Flags().BoolVarP(&u.isIncrementChartVersion, "increment-chart-version", "", false, "Increment the version of the Helm chart if requirements are updated.")
	cmd.Flags().IntVarP(&u.indent, "indent", "", 4, "Indent to use when writing the requirements.yaml .")

	return cmd
}

func (u *updateCmd) update() error {
	outdatedDeps, err := helm.ListOutdatedDependencies(u.chartPath, u.helmSettings, u.repositories)
	if err != nil {
		return err
	}

	if len(outdatedDeps) == 0 {
		fmt.Println("All charts up-to-date.")
		return nil
	}

	fmt.Println(u.formatResults(outdatedDeps))

	if u.isIncrementChartVersion {
		if err = helm.IncrementChartVersion(u.chartPath); err != nil {
			return err
		}
	}

	return helm.UpdateDependencies(u.chartPath, outdatedDeps, u.indent)
}

func (u *updateCmd) formatResults(results []*helm.Result) string {
	if len(results) == 0 {
		return "All charts up to date."
	}
	table := uitable.New()
	table.MaxColWidth = u.maxColumnWidth
	table.AddRow("Updating the following dependencies to their latest version:")
	table.AddRow("NAME", "VERSION", "LATEST_VERSION", "REPOSITORY")
	for _, r := range results {
		table.AddRow(r.Name, r.Version, r.LatestVersion, r.Repository)
	}
	return table.String()
}
