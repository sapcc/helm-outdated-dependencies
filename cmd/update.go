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
		chartPath      string
		helmSettings   *helm_env.EnvSettings
		maxColumnWidth uint
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
		maxColumnWidth: 60,
		helmSettings: &helm_env.EnvSettings{
			Home: helm.GetHelmHome(),
		},
	}

	cmd := &cobra.Command{
		Use:          "update",
		Long:         listLongUsage,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
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

	return cmd
}

func (u *updateCmd) update() error {
	outdatedDeps, err := helm.ListOutdatedDependencies(u.chartPath, u.helmSettings)
	if err != nil {
		return err
	}

	fmt.Println(u.formatResults(outdatedDeps))

	return helm.UpdateDependencies(u.chartPath, outdatedDeps)
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
