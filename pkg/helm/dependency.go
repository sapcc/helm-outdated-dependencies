package helm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Masterminds/semver"
	yamlv3 "gopkg.in/yaml.v3"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/getter"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/repo"
)

const (
	requirementsName  = "requirements.yaml"
	chartMetadataName = "Chart.yaml"
)

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

	reqs, err := chartutil.LoadRequirements(c)
	if err != nil {
		return nil, err
	}

	// Ignore local dependencies referenced by file://.. as they there's always just a single version available.
	var deps []*chartutil.Dependency
	for _, d := range reqs.Dependencies {
		if !strings.HasPrefix(d.Repository, "file://") {
			deps = append(deps, d)
		}
	}
	reqs.Dependencies = deps
	return reqs, nil
}

// ListOutdatedDependencies returns a list of outdated dependencies of the given chart.
func ListOutdatedDependencies(chartPath string, helmSettings *helm_env.EnvSettings, repositoryFilter []string) ([]*Result, error) {
	chartDeps, err := LoadDependencies(chartPath)
	if err != nil {
		if err == chartutil.ErrRequirementsNotFound {
			fmt.Printf("Chart %v has no requirements.\n", chartPath)
			return nil, err
		}
		return nil, err
	}

	// Consider only dependencies in the given repositories.
	chartDeps = filterDependenciesByRepository(chartDeps, repositoryFilter)

	if err = parallelRepoUpdate(chartDeps, helmSettings); err != nil {
		return nil, err
	}

	var res []*Result
	for _, dep := range chartDeps.Dependencies {
		depVersion, err := semver.NewVersion(dep.Version)
		if err != nil {
			fmt.Printf("Error creating semVersion for dependency %s: %s", dep.Name, err.Error())
			continue
		}

		latestVersion, err := findLatestVersionOfDependency(dep, helmSettings)
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
func UpdateDependencies(chartPath string, reqsToUpdate []*Result, indent int) error {
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

	return writeRequirements(chartPath, reqs, indent)
}

// IncrementChart version increments the patch version of the Chart.
func IncrementChartVersion(chartPath string) error {
	c, err := chartutil.Load(chartPath)
	if err != nil {
		return err
	}

	chartVersion, err := getChartVersion(c)
	if err != nil {
		return err
	}

	v := chartVersion.IncPatch()
	c.Metadata.Version = v.String()
	return writeChartMetadata(chartPath, c.Metadata)
}

// findLatestVersionOfDependency returns the latest version of the given dependency in the repository.
func findLatestVersionOfDependency(dep *chartutil.Dependency, helmSettings *helm_env.EnvSettings) (*semver.Version, error) {
	// Read the index file for the repository to get chart information and return chart URL
	repoIndex, err := repo.LoadIndexFile(helmSettings.Home.CacheIndex(normalizeRepoName(dep.Repository)))
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

func writeChartMetadata(chartPath string, c *chart.Metadata) error {
	data, err := toYamlWithIndent(c, 0)
	if err != nil {
		return err
	}

	absPath, err := filepath.Abs(path.Join(chartPath, chartMetadataName))
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

func writeRequirements(chartPath string, reqs *chartutil.Requirements, indent int) error {
	data, err := toYamlWithIndent(reqs, indent)
	if err != nil {
		return err
	}

	absPath, err := filepath.Abs(path.Join(chartPath, requirementsName))
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

func toYamlWithIndent(in interface{}, indent int) ([]byte, error) {
	// Unfortunately chartutil.Requirements, charts.Chart structs only have the JSON anchors, but not the YAML ones.
	// So we have to take the JSON detour.
	jsonData, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}

	var jsonObj interface{}
	if err := yamlv3.Unmarshal(jsonData, &jsonObj); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	enc := yamlv3.NewEncoder(&buf)
	defer enc.Close()
	enc.SetIndent(indent)
	err = enc.Encode(jsonObj)
	return buf.Bytes(), err
}

func getChartVersion(c *chart.Chart) (*semver.Version, error) {
	m := c.GetMetadata()
	if m == nil {
		return nil, errors.New("chart has no metdata")
	}

	v := m.GetVersion()
	if v == "" {
		return nil, errors.New("chart has no version")
	}

	return semver.NewVersion(v)
}

func filterDependenciesByRepository(reqs *chartutil.Requirements, repositoryFilter []string) *chartutil.Requirements {
	var filteredDeps []*chartutil.Dependency
	if repositoryFilter != nil && len(repositoryFilter) > 0 {
		for _, dep := range reqs.Dependencies {
			if stringSliceContains(repositoryFilter, dep.Repository) {
				filteredDeps = append(filteredDeps, dep)
			}
		}
	} else {
		filteredDeps = reqs.Dependencies
	}
	reqs.Dependencies = filteredDeps
	return reqs
}

func parallelRepoUpdate(chartDeps *chartutil.Requirements, helmSettings *helm_env.EnvSettings) error {
	var repos []string
	for _, dep := range chartDeps.Dependencies {
		if !stringSliceContains(repos, dep.Repository) {
			repos = append(repos, dep.Repository)
		}
	}

	var wg sync.WaitGroup
	for _, c := range repos {
		tmpRepo := &repo.Entry{
			Name: normalizeRepoName(c),
			URL:  c,
		}

		r, err := repo.NewChartRepository(tmpRepo, getter.All(*helmSettings))
		if err != nil {
			return err
		}

		wg.Add(1)
		go func(r *repo.ChartRepository) {
			if err := r.DownloadIndexFile(helmSettings.Home.CacheIndex(tmpRepo.Name)); err != nil {
				fmt.Printf("unable to get an update from the %q chart repository (%s):\n\t%s\n", r.Config.Name, r.Config.URL, err)
			} else {
				fmt.Printf("successfully got an update from the %q chart repository\n", r.Config.URL)
			}
			wg.Done()
		}(r)
	}
	wg.Wait()
	return nil
}

func stringSliceContains(stringSlice []string, searchString string) bool {
	for _, s := range stringSlice {
		if strings.Contains(s, searchString) || strings.Contains(searchString, s) {
			return true
		}
	}
	return false
}

func normalizeRepoName(repoURL string) string {
	name := strings.TrimLeft(repoURL, "https://")
	name = strings.TrimSuffix(name, "/")
	name = strings.ReplaceAll(name, "/", "-")
	return strings.ReplaceAll(name, ".", "-")
}
