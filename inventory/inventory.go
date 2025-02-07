package inventory

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/open-ch/kaeter/log"
	"github.com/open-ch/kaeter/modules"
)

// ModuleInventory inventory of kaeter modules in a repo
type ModuleInventory struct {
	Modules  []modules.KaeterModule `json:"Modules"` // Modules name is upper case in json because of historical reasons.
	RepoRoot string                 `json:"RepoRoot"`
}

// Inventory holds the data in ModuleInventory and additionally a lookup table
type Inventory struct {
	Lookup          map[string]modules.KaeterModule
	ModuleInventory ModuleInventory
}

// InventorizeRepo finds all kaeter modules in repositoryPath and creates an inventory
func InventorizeRepo(repositoryPath string) (*Inventory, error) {
	kaeterModules, err := modules.GetKaeterModules(repositoryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect kaeter modules: %w", err)
	}
	return buildInventory(repositoryPath, kaeterModules, idCmp)
}

// ReadFromFile reads an inventory of kaeter modules from a file
func ReadFromFile(filePath string) (*Inventory, error) {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return readFromBytes(raw)
}

// Read easy to use interface to read a kaeter inventory
func Read(input io.Reader) (*Inventory, error) {
	raw, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}
	return readFromBytes(raw)
}

func readFromBytes(inBytes []byte) (*Inventory, error) {
	modInv := ModuleInventory{}
	err := json.Unmarshal(inBytes, &modInv)
	if err != nil {
		return nil, err
	}
	// we take the repositoryPath from the file data
	return buildInventory(modInv.RepoRoot, modInv.Modules, idCmp)
}

// ToJSON converts the inventory list to an array of bytes in JSON format
func (i *Inventory) ToJSON() (string, error) {
	bytesJSON, err := json.MarshalIndent(i.ModuleInventory, "", "    ")
	if err != nil {
		return "", fmt.Errorf("could not marshal kaeter modules to JSON: %w", err)
	}
	stringJSON := string(bytesJSON)
	return stringJSON, nil
}

// GetModule returns a kaeter module from the inventory
func (i *Inventory) GetModule(id string) (*modules.KaeterModule, error) {
	m, found := i.Lookup[id]
	if !found {
		return nil, fmt.Errorf("module id does not exists: %s", id)
	}
	return &m, nil
}

// GetPathForModule returns a module path if the module is in the inventory
func (i *Inventory) GetPathForModule(id string) (string, error) {
	m, found := i.Lookup[id]
	if !found {
		return "", fmt.Errorf("module id does not exists: %s", id)
	}
	return m.ModulePath, nil
}

// GetAnnotationsForModule returns a map of all annotations if the module is in the inventory
func (i *Inventory) GetAnnotationsForModule(id string) (map[string]string, error) {
	m, found := i.Lookup[id]
	if !found {
		return nil, fmt.Errorf("module id does not exists: %s", id)
	}
	return m.Annotations, nil
}

// idCmp comparison function to sort kaeter modules
func idCmp(a, b modules.KaeterModule) int { //nolint:gocritic
	return strings.Compare(a.ModuleID, b.ModuleID)
}

// buildInventory takes a list of modules and creates the inventory structs
func buildInventory(repositoryPath string, modulesList []modules.KaeterModule, cmp func(a, b modules.KaeterModule) int) (*Inventory, error) {
	lookup := make(map[string]modules.KaeterModule)
	var unique []modules.KaeterModule
	var duplicates []string //nolint:prealloc

	for _, m := range modulesList {
		old, found := lookup[m.ModuleID]
		if !found {
			lookup[m.ModuleID] = m
			unique = append(unique, m)
			continue
		}
		duplicates = append(duplicates, m.ModuleID)
		log.Warn("duplicate Kaeter ModuleID found! Ignoring pathB...", "moduleID", m.ModuleID, "pathA", old.ModulePath, "pathB", m.ModulePath)
	}
	if duplicates != nil {
		return nil, fmt.Errorf("duplicate module IDs found: %s", strings.Join(duplicates, ", "))
	}
	// now sort the unique list
	slices.SortFunc(unique, cmp)
	modInv := ModuleInventory{
		Modules:  unique,
		RepoRoot: repositoryPath,
	}
	inv := Inventory{
		Lookup:          lookup,
		ModuleInventory: modInv,
	}
	return &inv, nil
}
