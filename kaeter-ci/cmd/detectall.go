package cmd

import (
	kaeterChange "github.com/open-ch/kaeter/kaeter-ci/pkg/change"
	"github.com/open-ch/kaeter/kaeter-ci/pkg/modules"

	"github.com/spf13/cobra"
)

func getDetectAllCommand() *cobra.Command {
	var changesFile string
	var modulesFile string
	var currentCommit string
	var previousCommit string
	var pullRequest kaeterChange.PullRequest

	detectCmd := &cobra.Command{
		Use:   "detect-all",
		Short: "Combined command to detect all modules and detect changes",
		Long: `This command extracts all the kaeter modules and runs change detection.
It will out put both a modules.json and a changeset.json file, similar to running
"kaeter-ci modules" and "kaeter-ci check" but it avoids running the module detection twice.`,
		Run: func(cmd *cobra.Command, args []string) {
			kaeterModules, err := modules.GetKaeterModules(repoPath)
			if err != nil {
				logger.Fatalf("failed to detect kaeter modules: %s", err)
			}
			err = saveModulesToFile(kaeterModules, modulesFile)
			if err != nil {
				logger.Fatalf("Modules detection failed: %s", err)
			} else {
				logger.Infof("Modules found saved to %s", modulesFile)
			}

			detector := &kaeterChange.Detector{
				Logger:         logger,
				RootPath:       repoPath,
				PreviousCommit: previousCommit,
				CurrentCommit:  currentCommit,
				KaeterModules:  kaeterModules,
				PullRequest:    &pullRequest,
			}
			err = runChangeDetection(detector, changesFile)
			if err != nil {
				logger.Fatalf("Change detection failed: %s", err)
			}
		},
	}

	flags := detectCmd.Flags()
	flags.StringVar(&previousCommit, "previous-commit", "HEAD~1", "The previous commit")
	flags.StringVar(&currentCommit, "latest-commit", "HEAD", "The current commit")
	flags.StringVar(&pullRequest.Title, "pr-title", "", "Optional: if a pull request is open, the title")
	flags.StringVar(&pullRequest.Body, "pr-body", "", "Optional: if a pull request is open, the body")
	flags.StringVar(&changesFile, "changes-output", "./changeset.json", "The path to the file containing the change information")
	flags.StringVar(&modulesFile, "modules-output", "./modules.json", "The path to the file containing the module information")

	return detectCmd
}
