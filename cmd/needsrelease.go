package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/open-ch/kaeter/log"
	"github.com/open-ch/kaeter/modules"
)

func getNeedsReleaseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "needsrelease",
		Hidden:  true, // TODO unhide once implementation is stable, currently WIP
		Aliases: []string{"nr"},
		Short:   "Outputs information for modules detected at the given paths regarding unreleased commits or old releases",
		Long: `This command makes it easy to catch modules that have either unreleased commits
or where not recently released/updated.
Modules will be automatically detected for at all the given paths, folders containing
multiple modules are supported.
Then for each module information about unreleased commits, and last release will be included.
The output will be a json object for each detected module separated by new lines.`,
		PreRunE: validateAllPathFlags,
		RunE: func(_ *cobra.Command, _ []string) error {
			inputSearchPaths := viper.GetStringSlice("path")
			moduleErrorCount := 0
			for _, searchPath := range inputSearchPaths {
				log.Info("Checking for modules in path", "searchPath", searchPath)
				modulesChan, err := modules.GetNeedsReleaseInfoIn(searchPath)
				if err != nil {
					return err // Fail on first path error
				}
				for needsReleaseInfo := range modulesChan {
					if needsReleaseInfo.Error != nil {
						moduleErrorCount++
						log.Error("Module with error", "versionsYamlPath", needsReleaseInfo.ModulePath, "error", needsReleaseInfo.Error)
					}
					printNeedsReleaseInfo(&needsReleaseInfo)
				}
			}
			if moduleErrorCount > 0 {
				return fmt.Errorf("several (%d) module(s) have parsing errors", moduleErrorCount)
			}
			return nil
		},
	}

	return cmd
}

func printNeedsReleaseInfo(needsReleaseInfo *modules.ModuleNeedsReleaseInfo) {
	needsReleaseInfoJSON, err := json.Marshal(needsReleaseInfo)
	if err != nil {
		log.Error("Unable to format module needsrelease data", "module", needsReleaseInfo.ModuleID)
	}
	fmt.Println(string(needsReleaseInfoJSON))
}
