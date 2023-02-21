package cmd

import (
	"encoding/json"
	"os"
	"github.com/open-ch/kaeter/kaeter/modules"
	"path/filepath"

	"github.com/spf13/cobra"
)

// Result holds the structure of the JSON output generated by the command
type Result struct {
	Modules map[string]modules.KaeterModule
}

func getModulesCommand() *cobra.Command {
	var outputFile string

	modulesCmd := &cobra.Command{
		Use:   "modules",
		Short: "List all detected modules inside a specified path.",
		Long: `List information of all detected modules inside a specified path.
This includes the following module information:
  - identifier
  - path
  - type
  - annotations`,
		Run: func(cmd *cobra.Command, args []string) {
			kaeterModules, err := modules.GetKaeterModules(repoPath)
			if err != nil {
				logger.Fatalf("kaeter-ci: failed to detect kaeter modules: %s", err)
			}

			err = saveModulesToFile(kaeterModules, outputFile)
			if err != nil {
				logger.Fatalf("kaeter-ci: Modules command failed: %s", err)
			} else {
				logger.Infof("kaeter-ci: Output stored in %s", outputFile)
			}
		},
	}

	modulesCmd.Flags().StringVar(&outputFile, "output", "./modules.json", "The path to the file containing the module information")

	return modulesCmd
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
		outputFile = filepath.Join(repoPath, outputFile)
	}

	return os.WriteFile(outputFile, resultJSON, 0600)
}
