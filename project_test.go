package gosolc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProject_Fixtures(t *testing.T) {
	var fixturesPath = "./fixtures"

	entries, err := os.ReadDir(fixturesPath)
	require.NoError(t, err)

	for _, e := range entries {
		t.Run(e.Name(), func(t *testing.T) {
			testPath := filepath.Join(fixturesPath, e.Name())

			project, err := NewProject(WithContractsDir(testPath), WithRuns(200))
			require.NoError(t, err)

			res, err := project.Compile()
			require.NoError(t, err)

			require.NotEmpty(t, res.Contracts)
		})
	}
}
