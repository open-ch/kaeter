package change

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)


func TestHelmChartFolderMatch(t *testing.T) {
	d := New(logrus.InfoLevel, ".", "commit1", "commit2")
	charts := []string{"path/to/chart/1/", "path/to/chart/2/"}

	files := []string{"path/to/chart/1/other/File.asdf", "path/to/chart/1/folder/asdf"}
	result := d.matchFilesAndCharts(files, charts)
	assert.Equal(t, []string{"path/to/chart/1/"}, result)

	files = []string{"path/to/chart/1/file1"}
	result = d.matchFilesAndCharts(files, charts)
	assert.Equal(t, []string{"path/to/chart/1/"}, result)

	files = []string{"path/to/chart/1/file1", "path/to/chart/2/file1"}
	result = d.matchFilesAndCharts(files, charts)
	assert.Equal(t, []string{"path/to/chart/1/", "path/to/chart/2/"}, result)
}
