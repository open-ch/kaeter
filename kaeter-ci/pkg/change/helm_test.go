package change

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestHelmChartFolderMatch(t *testing.T) {
	testCases := []struct {
		name            string
		inputFiles      []string
		expectedMatches []string
	}{
		{
			name:            "Multiple matches in 1 chart",
			inputFiles:      []string{"path/to/chart/1/other/File.asdf", "path/to/chart/1/folder/asdf"},
			expectedMatches: []string{"path/to/chart/1/"},
		},
		{
			name:            "One match in 1 chart",
			inputFiles:      []string{"path/to/chart/1/file1"},
			expectedMatches: []string{"path/to/chart/1/"},
		},
		{
			name:            "Mutiple matches in multiple charts",
			inputFiles:      []string{"path/to/chart/1/file1", "path/to/chart/2/file1"},
			expectedMatches: []string{"path/to/chart/1/", "path/to/chart/2/"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			chartPaths := []string{"path/to/chart/1/", "path/to/chart/2/"}
			d := &Detector{
				Logger:         logrus.New(),
				RootPath:       "n/a",
				PreviousCommit: "commit1",
				CurrentCommit:  "commit2",
			}

			result := d.matchFilesAndCharts(tc.inputFiles, chartPaths)
			assert.Equal(t, tc.expectedMatches, result)
		})
	}
}
