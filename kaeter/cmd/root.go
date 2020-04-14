package cmd

import (
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const versionsFile = "versions.yml"
const makeFile = "Makefile"

var (
	// Points to the module to be released
	modulePath string

	rootCmd = &cobra.Command{
		Use:   "kaeter",
		Short: "kaeter handles the releasing and versioning of your modules within a fat repo.",
		Long: `kaeter offers a standard approach for releasing and versioning arbitrary artifacts. 
Its goal is to provide a 'descriptive release' process, in which developers request the release of given artifacts, 
and upon acceptation of the request, a separate build infrastructure is in charge of carrying out the build.`,
	}

	// Logger...
	logger = log.New()
)

func init() {
	cobra.OnInitialize()

	rootCmd.PersistentFlags().StringVarP(&modulePath, "path", "p", ".",
		"Path to the module for which a release is required.")

	logger.SetFormatter(&log.TextFormatter{
		DisableTimestamp: true,
	})
}

// Execute runs the whole enchilada, baby!
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func openRepoAndWorktree(moduleDir string) (*git.Repository, *git.Worktree, error) {
	repo, err := git.PlainOpenWithOptions(moduleDir, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, nil, err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return nil, nil, err
	}
	return repo, wt, err
}
