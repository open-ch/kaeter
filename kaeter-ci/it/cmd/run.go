package cmd

import (
	"testing"

	"github.com/open-ch/go-libs/gitshell"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

var (
	// Points to the module to be checked
	repoRoot   string
	kaeterPath string
	baseCommit string
)

func init() {
	runCmd := &cobra.Command{
		Use:   "kaeter-ci-test",
		Short: "kaeter-ci-test performs integration test on kaeter-ci.",
		Run: func(cmd *cobra.Command, args []string) {
			testing.Main(
				nil,
				[]testing.InternalTest{
					{"E2ETestSuite", TestE2ETestSuite},
				},
				nil, nil,
			)
		},
	}

	runCmd.PersistentFlags().StringVarP(&repoRoot, "repoRoot", "p", ".", "The path to the testing repo")
	runCmd.PersistentFlags().StringVarP(&kaeterPath, "kaeterPath", "k", "kaeter-ci", "The path to the kaeter executable")
	runCmd.PersistentFlags().StringVarP(&baseCommit, "baseCommit", "c", "HEAD", "The current base commit")
	rootCmd.AddCommand(runCmd)
}

type e2eTestSuite struct {
	suite.Suite
	repoRoot   string
	kaeterPath string
	baseCommit string
}

// TestE2ETestSuite starts the IT suite for kaeter-ci
func TestE2ETestSuite(t *testing.T) {
	suite.Run(t, &e2eTestSuite{})
}

func (s *e2eTestSuite) SetupSuite() {
	s.repoRoot = repoRoot
	s.kaeterPath = kaeterPath
	s.baseCommit = baseCommit
}

func (s *e2eTestSuite) SetupTest() {
	_, err := gitshell.GitReset(s.repoRoot, s.baseCommit)
	s.NoError(err)
}
