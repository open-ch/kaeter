package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/open-ch/kaeter/kaeter/change"
	"github.com/open-ch/kaeter/kaeter/modules"
)

// Result holds the structure of the JSON output generated by the command
type Result struct {
	Modules map[string]modules.KaeterModule
}

func getCIDetectChangesCommand() *cobra.Command {
	var changesFile string
	var modulesFile string
	var currentCommit string
	var previousCommit string
	var pullRequest change.PullRequest
	var skipModulesDetection bool
	var skipChangesDetection bool

	detectCmd := &cobra.Command{
		Use:   "detect-changes",
		Short: "Combined command to detect all modules and changes",
		Long: `This command extracts all the kaeter modules and runs change detection.
It will out put both a modules.json and a changeset.json file.

Previously called "kaeter-ci detect-all".
`,
		Run: func(cmd *cobra.Command, args []string) {
			// TODO use viper to get root to avoid global
			kaeterModules, err := modules.GetKaeterModules(repoRoot)
			if err != nil {
				logger.Fatalf("failed to detect kaeter modules: %s", err)
			}
			if !skipModulesDetection {
				err = saveModulesToFile(kaeterModules, modulesFile)
				if err != nil {
					logger.Fatalf("Modules detection failed: %s", err)
				} else {
					logger.Infof("Modules found saved to %s", modulesFile)
				}

				if skipChangesDetection {
					return
				}
			}

			detector := &change.Detector{
				Logger:         logger,
				RootPath:       repoRoot,
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

	flags.BoolVar(&skipModulesDetection, "changes-only", false, "Skips saving the modules.json file")
	flags.BoolVar(&skipChangesDetection, "modules-only", false, "Skips change detection and saving changeset.json")
	detectCmd.MarkFlagsMutuallyExclusive("changes-only", "modules-only")

	return detectCmd
}

func runChangeDetection(detector *change.Detector, outputFile string) error {
	if !filepath.IsAbs(outputFile) {
		outputFile = filepath.Join(repoRoot, outputFile)
	}

	changeset, err := detector.Check()
	if err != nil {
		return err
	}

	changesetJSON, err := json.MarshalIndent(changeset, "", "    ")
	if err != nil {
		return err
	}
	detector.Logger.Infoln(string(changesetJSON))
	return os.WriteFile(outputFile, changesetJSON, 0600)
}

func saveModulesToFile(kaeterModules []modules.KaeterModule, outputFile string) error {
	result := new(Result)
	result.Modules = make(map[string]modules.KaeterModule)

	for _, module := range kaeterModules {
		result.Modules[module.ModuleID] = module
	}

	resultJSON, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		return err
	}

	if !filepath.IsAbs(outputFile) {
		outputFile = filepath.Join(repoRoot, outputFile)
	}

	return os.WriteFile(outputFile, resultJSON, 0600)
}
