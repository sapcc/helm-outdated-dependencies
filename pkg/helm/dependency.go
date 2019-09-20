package helm

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/Masterminds/semver"
	"github.com/ghodss/yaml"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/getter"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/repo"
)

const requirementsName = "requirements.yaml"

// Result ...
type Result struct {
	*chartutil.Dependency

	LatestVersion *semver.Version
}

// GetHelmHome returns the HELM_HOME path.
func GetHelmHome() helmpath.Home {
	home := helm_env.DefaultHelmHome
	if h := os.Getenv("HELM_HOME"); h != "" {
		home = h
	}
	return helmpath.Home(home)
}

// LoadDependencies loads the dependencies of the given chart.
func LoadDependencies(chartPath string) (*chartutil.Requirements, error) {
	c, err := chartutil.Load(chartPath)
	if err != nil {
		return nil, err
	}

	return chartutil.LoadRequirements(c)
}

// ListOutdatedDependencies returns a list of outdated dependencies of the given chart.
func ListOutdatedDependencies(chartPath string, helmSettings *helm_env.EnvSettings) ([]*Result, error) {
	chartDeps, err := LoadDependencies(chartPath)
	if err != nil {
		if err == chartutil.ErrRequirementsNotFound {
			fmt.Printf("Chart %v has no requirements.\n", chartPath)
			return nil, err
		}
		return nil, err
	}

	var res []*Result
	for _, dep := range chartDeps.Dependencies {
		depVersion, err := semver.NewVersion(dep.Version)
		if err != nil {
			fmt.Printf("Error creating semVersion for dependency %s: %s", dep.Name, err.Error())
			continue
		}

		latestVersion, err := FindLatestVersionOfDependency(dep, helmSettings)
		if err != nil {
			fmt.Printf("Error getting latest version of %s: %s\n", dep.Name, err.Error())
			continue
		}

		if depVersion.LessThan(latestVersion) {
			res = append(res, &Result{
				Dependency:    dep,
				LatestVersion: latestVersion,
			})
		}
	}

	return res, nil
}

// UpdateDependencies updates the dependencies of the given chart.
func UpdateDependencies(chartPath string, reqsToUpdate []*Result) error {
	c, err := chartutil.Load(chartPath)
	if err != nil {
		return err
	}

	reqs, err := chartutil.LoadRequirements(c)
	if err != nil {
		return err
	}

	for _, newDep := range reqsToUpdate {
		for _, oldDep := range reqs.Dependencies {
			if newDep.Name == oldDep.Name && newDep.Repository == newDep.Repository {
				oldDep.Version = newDep.LatestVersion.String()
			}
		}
	}

	return writeRequirements(chartPath, reqs)
}

// FindLatestVersionOfDependency returns the latest version of the given dependency in the repository.
func FindLatestVersionOfDependency(dep *chartutil.Dependency, helmSettings *helm_env.EnvSettings) (*semver.Version, error) {
	// Download and write the index file to a temporary location
	tempIndexFile, err := ioutil.TempFile("", "tmp-repo-file")
	if err != nil {
		return nil, fmt.Errorf("cannot write index file for repository requested")
	}
	defer os.Remove(tempIndexFile.Name())

	c := repo.Entry{URL: dep.Repository}
	r, err := repo.NewChartRepository(&c, getter.All(*helmSettings))
	if err != nil {
		return nil, err
	}
	if err := r.DownloadIndexFile(tempIndexFile.Name()); err != nil {
		return nil, fmt.Errorf("can't reach repository %s: %s", dep.Repository, err.Error())
	}

	// Read the index file for the repository to get chart information and return chart URL
	repoIndex, err := repo.LoadIndexFile(tempIndexFile.Name())
	if err != nil {
		return nil, err
	}

	// With no version given the highest one is returned.
	cv, err := repoIndex.Get(dep.Name, "")
	if err != nil {
		return nil, err
	}

	return semver.NewVersion(cv.Version)
}

func writeRequirements(chartPath string, reqs *chartutil.Requirements) error {
	// Unfortunately chartutil.Requirements only has the JSON omitempty annotations, but not the YAML ones.
	// So we have to take the JSON detour.
	data, err := json.Marshal(reqs)
	if err != nil {
		return err
	}

	data, err = yaml.JSONToYAML(data)
	if err != nil {
		return err
	}

	requirementsPath := path.Join(chartPath, requirementsName)
	absPath, err := filepath.Abs(requirementsPath)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(absPath, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteAt(data, 0)
	return err
}
