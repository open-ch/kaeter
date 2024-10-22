package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/open-ch/kaeter/inventory"
)

func getInventorizeCommand() *cobra.Command {
	createInventoryCmd := &cobra.Command{
		Use:     "inventorize",
		Short:   "Create an inventory of kaeter modules in given path and print to STDOUT",
		Long:    `This command extracts all kaeter modules and returns them as list to STDOUT.`,
		PreRunE: validateAllPathFlags,
		RunE: func(_ *cobra.Command, _ []string) error {
			// this command does not need "path" information, we "just" need repoRoot.
			// however, current implementation of evaluating repoRoot requires the path flag.
			// that's why we call validateAllPathFlags in PreRunE.
			repositoryPath := viper.GetString("repoRoot")
			inv, err := inventory.InventorizeRepo(repositoryPath)
			if err != nil {
				return fmt.Errorf("failed to create module inventory: %w", err)
			}

			var stringJSON string
			stringJSON, err = inv.ToJSON()
			if err != nil {
				return fmt.Errorf("could not marshal kaeter modules to JSON: %w", err)
			}
			fmt.Println(stringJSON)
			return nil
		},
	}

	return createInventoryCmd
}
