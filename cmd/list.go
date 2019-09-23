package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/gosuri/uitable"
	"github.com/sapcc/helm-outdated-dependencies/pkg/helm"
	"github.com/spf13/cobra"
	helm_env "k8s.io/helm/pkg/helm/environment"
)

var listLongUsage = `
Helm plugin to manage outdated dependencies of a Helm chart.

Examples:
  $ helm outdated-dependencies list
  $ helm outdated-dependencies list <chartPath>
`

type (
	listCmd struct {
		maxColumnWidth uint
		chartPath      string
		repositories   []string
		helmSettings   *helm_env.EnvSettings
	}
)

func newListOutdatedDependenciesCmd() *cobra.Command {
	l := &listCmd{
		helmSettings: &helm_env.EnvSettings{
			Home: helm.GetHelmHome(),
		},
		maxColumnWidth: 60,
		repositories:   []string{},
	}

	cmd := &cobra.Command{
		Use:          "list",
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
			l.chartPath = path

			if maxColumnWidth, err := cmd.Flags().GetUint("max-column-width"); err == nil {
				l.maxColumnWidth = maxColumnWidth
			}

			if repositories, err := cmd.Flags().GetStringSlice("repositories"); err == nil {
				l.repositories = repositories
			}

			return l.list()
		},
	}

	addCommonFlags(cmd)

	return cmd
}

func (l *listCmd) list() error {
	outdatedDeps, err := helm.ListOutdatedDependencies(l.chartPath, l.helmSettings, l.repositories)
	if err != nil {
		return err
	}

	fmt.Println(l.formatResults(outdatedDeps))
	return nil
}

func (l *listCmd) formatResults(results []*helm.Result) string {
	if len(results) == 0 {
		return "All charts up to date."
	}
	table := uitable.New()
	table.MaxColWidth = l.maxColumnWidth
	table.AddRow("The following dependencies are outdated:")
	table.AddRow("NAME", "VERSION", "LATEST_VERSION", "REPOSITORY")
	for _, r := range results {
		table.AddRow(r.Name, r.Version, r.LatestVersion, r.Repository)
	}
	return table.String()
}
