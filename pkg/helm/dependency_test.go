package helm

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/helm/pkg/chartutil"
	"os"
	"path"
	"testing"
)

func newRequirements() *chartutil.Requirements {
	return &chartutil.Requirements{
		Dependencies: []*chartutil.Dependency{
			{
				Name: "testdependency",
				Version: "v0.0.1",
				Repository: "https://repo.evil.corp",
			},
			{
				Name: "testdepdendency1",
				Version: "v0.0.2",
				Repository: "https://repo.evil.corp",
			},
		},
	}
}

func TestWriteRequirements(t *testing.T) {
	dir, err := os.Getwd()
	require.NoError(t, err, "there must be no error getting the current path")
	chartPath := path.Join(dir, "fixtures")
	_, err = os.Create(path.Join(chartPath, requirementsName))
	require.NoError(t, err, "there must be no error creating the requirements.yaml")

	err = writeRequirements(chartPath, newRequirements())
	assert.NoError(t, err, "there should be no error writing the chart requirements")
}
