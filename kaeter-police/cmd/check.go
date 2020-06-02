package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/open-ch/go-libs/fsutils"
	"github.com/spf13/cobra"
)

func init() {

	checkCmd := &cobra.Command{
		Use:   "check",
		Short: "Basic quality checks for the specified module.",
		Long: `Check that the specified module meets some basic quality requirements:

    For every kaeter-managed package (which has a versions.yml file) the following is checked:
     - the existence of README.md
     - the existence of CHANGELOG.md`,
		Run: func(cmd *cobra.Command, args []string) {
			err := runCheck()
			if err != nil {
				logger.Errorf("Check failed: %s", err)
				os.Exit(1)
			}
		},
	}

	rootCmd.AddCommand(checkCmd)
}

func runCheck() error {
	root, err := fsutils.SearchClosestParentContaining(rootPath, ".git")
	if err != nil {
		return err
	}

	allVersionsFiles, err := fsutils.SearchByFileName(root, versionsFile)
	if err != nil {
		return err
	}

	for _, absVersionFilePath := range allVersionsFiles {
		absModulePath := filepath.Dir(absVersionFilePath)
		if err := checkExistence(readmeFile, absModulePath); err != nil {
			return fmt.Errorf("README existence check failed: %s", err.Error())
		}

		if err := checkExistence(changelogFile, absModulePath); err != nil {
			return fmt.Errorf("CHANGELOG existence check failed: %s", err.Error())
		}
	}

	return nil
}

func checkExistence(file string, absModulePath string) error {
	info, err := os.Stat(absModulePath)
	if err != nil {
		return fmt.Errorf("Error in getting FileInfo about '%s': %s", absModulePath, err.Error())
	}

	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", info.Name())
	}
	absFilePath := filepath.Join(absModulePath, file)

	_, err = os.Stat(absFilePath)
	if err != nil {
		return fmt.Errorf("Error in getting FileInfo about '%s': %s", file, err.Error())
	}

	return nil
}
