package cmd

import (
	"io/ioutil"
	"path"

	"github.com/open-ch/go-libs/gitshell"
)

func (s *e2eTestSuite) TestHelmChartFolderMatch() {
	// Write a change in the Helm chart elasticsearch monitoring
	chartFolder := "libs/charts/osdp/elasticsearch-monitoring/"
	modifiedFile := path.Join(s.repoRoot, chartFolder, "value2.yaml")
	fileContent := []byte("testVariable: value")
	s.NoError(ioutil.WriteFile(modifiedFile, fileContent, 0644))
	_, err := gitshell.GitAdd(s.repoRoot, modifiedFile)
	s.NoError(err)
	_, err = gitshell.GitCommit(s.repoRoot, "WIP")
	s.NoError(err)
	newCommit, err := gitshell.GitResolveRevision(s.repoRoot, "HEAD")
	s.NoError(err)

	// Execute the query
	info, err := executeKaeterCI(s.kaeterPath, s.repoRoot, s.baseCommit, newCommit)
	s.NoError(err)

	// Verify the result
	s.Contains(info.Helm.Charts, chartFolder)
	s.Equal(1, len(info.Helm.Charts))

}

func (s *e2eTestSuite) TestHelmChartMultipleChanges() {
	// Write a change in the Helm chart elasticsearch monitoring
	chartFolder1 := "libs/charts/osdp/elasticsearch-monitoring/"
	modifiedFile1 := path.Join(s.repoRoot, chartFolder1, "value2.yaml")
	fileContent1 := []byte("testVariable: value")
	s.NoError(ioutil.WriteFile(modifiedFile1, fileContent1, 0644))
	_, err := gitshell.GitAdd(s.repoRoot, modifiedFile1)
	s.NoError(err)

	// Write a change in the Helm chart kafdrop
	chartFolder2 := "libs/charts/osdp/kafdrop/"
	modifiedFile2 := path.Join(s.repoRoot, chartFolder2, "value2.yaml")
	fileContent2 := []byte("testVariable: value")
	s.NoError(ioutil.WriteFile(modifiedFile2, fileContent2, 0644))
	_, err = gitshell.GitAdd(s.repoRoot, modifiedFile2)
	s.NoError(err)
	_, err = gitshell.GitCommit(s.repoRoot, "WIP")
	s.NoError(err)
	newCommit, err := gitshell.GitResolveRevision(s.repoRoot, "HEAD")
	s.NoError(err)

	// Execute the query
	info, err := executeKaeterCI(s.kaeterPath, s.repoRoot, s.baseCommit, newCommit)
	s.NoError(err)

	// Verify the result
	s.Contains(info.Helm.Charts, chartFolder1)
	s.Contains(info.Helm.Charts, chartFolder2)
	s.Equal(2, len(info.Helm.Charts))
}
