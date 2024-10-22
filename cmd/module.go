package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/open-ch/kaeter/inventory"
	"github.com/open-ch/kaeter/modules"
)

func getModuleCommand() *cobra.Command {
	var inventoryPath string
	var annotationFlag bool
	var requestPathFlag bool

	createInventoryCmd := &cobra.Command{
		Use:     "module",
		Short:   "Get kaeter module information as JSON",
		Args:    cobra.MatchAll(cobra.ExactArgs(1)),
		Long:    `Get kaeter module information as JSON either from the filesystem or inventory file`,
		PreRunE: validateAllPathFlags,
		RunE: func(_ *cobra.Command, args []string) error {
			// this command does not need "path" information, we "just" need repoRoot.
			// however, current implementation of evaluating repoRoot requires the path flag.
			// that's why we call validateAllPathFlags in PreRunE.
			repositoryPath := viper.GetString("repoRoot")
			inv, err := getInventory(repositoryPath, inventoryPath)
			if err != nil {
				return fmt.Errorf("failed to get module inventory: %w", err)
			}
			moduleID := args[0]

			if annotationFlag {
				var m map[string]string
				m, err = inv.GetAnnotationsForModule(moduleID)
				if err != nil {
					return fmt.Errorf("could not get annotations for module: %w", err)
				}
				var s string
				s, err = encodeAsJSONString(m)
				if err != nil {
					return err
				}
				fmt.Println(s)
				return nil
			}
			if requestPathFlag {
				var s string
				s, err = inv.GetPathForModule(moduleID)
				if err != nil {
					return fmt.Errorf("could not get path for module: %w", err)
				}
				fmt.Println(s)
				return nil
			}
			var module *modules.KaeterModule
			module, err = inv.GetModule(moduleID)
			if err != nil {
				return fmt.Errorf("could not get module for ID: %w", err)
			}

			var s string
			s, err = encodeAsJSONString(module)
			if err != nil {
				return err
			}
			fmt.Println(s)
			return nil
		},
	}
	createInventoryCmd.Flags().StringVarP(&inventoryPath, "inventory", "", "", "path to inventory file")
	createInventoryCmd.Flags().BoolVarP(&annotationFlag, "annotations", "", false, "get annotations only (JSON)")
	createInventoryCmd.Flags().BoolVarP(&requestPathFlag, "get-path", "", false, "get ModulePath (string)")
	createInventoryCmd.MarkFlagsMutuallyExclusive("annotations", "get-path")
	// ideally, we would only use path or inventory, but we don't for now
	// createInventoryCmd.MarkFlagsOneRequired("path","inventory")
	// createInventoryCmd.MarkFlagsMutuallyExclusive("path","inventory")
	return createInventoryCmd
}

func getInventory(repo, invPath string) (*inventory.Inventory, error) {
	if invPath != "" {
		return inventory.ReadFromFile(invPath)
	}
	return inventory.InventorizeRepo(repo)
}

func encodeAsJSONString(v any) (string, error) {
	bytes, err := json.MarshalIndent(v, "", "    ")
	if err != nil {
		return "", fmt.Errorf("failed to JSON encode module: %w", err)
	}
	s := string(bytes)
	return s, nil
}
