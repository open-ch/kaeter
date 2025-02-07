package inventory

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-ch/kaeter/modules"
)

func Test_RedFromFile(t *testing.T) {
	file, err := os.CreateTemp("", "readTest.json")
	assert.NoError(t, err)
	defer os.Remove(file.Name())
	tmpName := file.Name()

	inv := mockInventoryObject(t)
	var stringJSON string
	stringJSON, err = inv.ToJSON()
	assert.NoError(t, err)
	fmt.Fprintln(file, stringJSON)
	err = file.Close()
	assert.NoError(t, err)

	var i *Inventory
	i, err = ReadFromFile(tmpName)
	assert.NoError(t, err)
	assert.Equal(t, inv, i)
}

func Test_ReadFromBytes(t *testing.T) {
	var tests = []struct {
		name     string
		inBytes  []byte
		expected *Inventory
		mustFail bool
	}{
		{
			name:     "happy path",
			inBytes:  []byte(mockFullJSON(t)),
			expected: mockInventoryObject(t),
			mustFail: false,
		},
		{
			name: "empty",
			inBytes: []byte(`{
    "Modules": null,
    "RepoRoot": "a/repo/root"
}`),
			expected: &Inventory{
				Lookup: map[string]modules.KaeterModule{},
				ModuleInventory: ModuleInventory{
					Modules:  nil,
					RepoRoot: "a/repo/root",
				},
			},
			mustFail: false,
		},
		{
			name:    "empty JSON",
			inBytes: []byte(`{}`),
			expected: &Inventory{
				Lookup: map[string]modules.KaeterModule{},
				ModuleInventory: ModuleInventory{
					Modules:  nil,
					RepoRoot: "",
				},
			},
			mustFail: false,
		},
		{
			name:     "empty string fails",
			inBytes:  []byte(``),
			expected: nil,
			mustFail: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := readFromBytes(tc.inBytes)
			if tc.mustFail {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func Test_Read(t *testing.T) {
	var tests = []struct {
		name     string
		reader   io.Reader
		expected *Inventory
		mustFail bool
	}{
		{
			name:     "happy path",
			reader:   strings.NewReader(mockFullJSON(t)),
			expected: mockInventoryObject(t),
			mustFail: false,
		},
		{
			name: "empty",
			reader: strings.NewReader(`{
    "Modules": null,
    "RepoRoot": "a/repo/root"
}`),
			expected: &Inventory{
				Lookup: map[string]modules.KaeterModule{},
				ModuleInventory: ModuleInventory{
					Modules:  nil,
					RepoRoot: "a/repo/root",
				},
			},
			mustFail: false,
		},
		{
			name:   "empty JSON",
			reader: strings.NewReader(`{}`),
			expected: &Inventory{
				Lookup: map[string]modules.KaeterModule{},
				ModuleInventory: ModuleInventory{
					Modules:  nil,
					RepoRoot: "",
				},
			},
			mustFail: false,
		},
		{
			name:     "empty string fails",
			reader:   strings.NewReader(``),
			expected: nil,
			mustFail: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Read(tc.reader)
			if tc.mustFail {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestInventory_ToJSON(t *testing.T) {
	var tests = []struct {
		name      string
		expected  string
		inventory *Inventory
		mustFail  bool
	}{
		{
			name:      "happy path",
			expected:  mockFullJSON(t),
			inventory: mockInventoryObject(t),
			mustFail:  false,
		},
		{
			name: "empty",
			expected: `{
    "Modules": null,
    "RepoRoot": "b/path"
}`,
			inventory: &Inventory{
				Lookup: map[string]modules.KaeterModule{},
				ModuleInventory: ModuleInventory{
					Modules:  nil,
					RepoRoot: "b/path",
				},
			},
			mustFail: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.inventory.ToJSON()
			if tc.mustFail {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestInventory_GetModule(t *testing.T) {
	inv := mockInventoryObject(t)
	var tests = []struct {
		name     string
		expected *modules.KaeterModule
		inputID  string
		mustFail bool
	}{
		{
			name: "module1",
			expected: &modules.KaeterModule{
				ModuleID:     "example.com:module1",
				ModulePath:   "happiness/this/way",
				ModuleType:   "nice",
				Annotations:  nil,
				Dependencies: nil,
			},
			inputID:  "example.com:module1",
			mustFail: false,
		},
		{
			name: "module2",
			expected: &modules.KaeterModule{
				ModuleID:   "example.com:module2",
				ModulePath: "that/way",
				ModuleType: "nice",
				Annotations: map[string]string{
					"example.com/annotation1": "annotation1",
				},
			},
			inputID:  "example.com:module2",
			mustFail: false,
		},
		{
			name:     "module does not exist",
			expected: nil,
			inputID:  "example.com:module4",
			mustFail: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := inv.GetModule(tc.inputID)
			if tc.mustFail {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestInventory_GetPathForModule(t *testing.T) {
	inv := mockInventoryObject(t)
	var tests = []struct {
		name     string
		expected string
		inputID  string
		mustFail bool
	}{
		{
			name:     "module1",
			expected: "happiness/this/way",
			inputID:  "example.com:module1",
			mustFail: false,
		},
		{
			name:     "module does not exist",
			expected: "",
			inputID:  "example.com:module4",
			mustFail: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := inv.GetPathForModule(tc.inputID)
			if tc.mustFail {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestInventory_GetAnnotationsForModule(t *testing.T) {
	inv := mockInventoryObject(t)
	var tests = []struct {
		name     string
		expected map[string]string
		inputID  string
		mustFail bool
	}{
		{
			name:     "empty",
			expected: nil,
			inputID:  "example.com:module1",
			mustFail: false,
		},
		{
			name: "one annotation",
			expected: map[string]string{
				"example.com/annotation1": "annotation1",
			},
			inputID:  "example.com:module2",
			mustFail: false,
		},
		{
			name: "two annotations",
			expected: map[string]string{
				"example.com/annotation1": "no",
				"example.com/annotation2": "yes",
			},
			inputID:  "example.com:module3",
			mustFail: false,
		},
		{
			name:     "module does not exist",
			expected: nil,
			inputID:  "example.com:module4",
			mustFail: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := inv.GetAnnotationsForModule(tc.inputID)
			if tc.mustFail {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestBuildInventory(t *testing.T) {
	repoRoot := "a/repo/root"
	modulesInput := mockModuleList(t)
	expectedInventory := mockInventoryObject(t)

	var tests = []struct {
		name        string
		modulesList []modules.KaeterModule
		expected    *Inventory
		mustFail    bool
	}{
		{
			name:        "happy path",
			modulesList: modulesInput,
			expected:    expectedInventory,
			mustFail:    false,
		},
		{
			name:        "list is sorted",
			modulesList: shuffleHelper(t, modulesInput),
			expected:    expectedInventory,
			mustFail:    false,
		},
		{
			name:        "nil",
			modulesList: nil,
			expected: &Inventory{
				Lookup: map[string]modules.KaeterModule{},
				ModuleInventory: ModuleInventory{
					Modules:  nil,
					RepoRoot: repoRoot,
				},
			},
			mustFail: false,
		},
		{
			name:        "empty",
			modulesList: []modules.KaeterModule{},
			expected: &Inventory{
				Lookup: map[string]modules.KaeterModule{},
				ModuleInventory: ModuleInventory{
					Modules:  nil,
					RepoRoot: repoRoot,
				},
			},
			mustFail: false,
		},
		{
			name:        "duplicates",
			modulesList: append(modulesInput, modulesInput...),
			expected:    expectedInventory,
			mustFail:    true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := buildInventory(repoRoot, tc.modulesList, idCmp)
			if tc.mustFail {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestIdCmp(t *testing.T) {
	mockA := modules.KaeterModule{
		ModuleID: "a",
	}
	mockB := modules.KaeterModule{
		ModuleID: "b",
	}
	var tests = []struct {
		name     string
		a        modules.KaeterModule
		b        modules.KaeterModule
		expected int
	}{
		{name: "a before b",
			a:        mockA,
			b:        mockB,
			expected: -1,
		},
		{name: "b before a",
			a:        mockB,
			b:        mockA,
			expected: 1,
		},
		{name: "same a",
			a:        mockA,
			b:        mockA,
			expected: 0,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := idCmp(tc.a, tc.b)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func mockModuleList(t *testing.T) []modules.KaeterModule {
	t.Helper()
	return []modules.KaeterModule{
		{
			ModuleID:   "example.com:module1",
			ModulePath: "happiness/this/way",
			ModuleType: "nice",
		},
		{
			ModuleID:   "example.com:module2",
			ModulePath: "that/way",
			ModuleType: "nice",
			Annotations: map[string]string{
				"example.com/annotation1": "annotation1",
			},
		},
		{
			ModuleID:   "example.com:module3",
			ModulePath: "this/is/the/way",
			ModuleType: "nice",
			Annotations: map[string]string{
				"example.com/annotation1": "no",
				"example.com/annotation2": "yes",
			},
			AutoRelease: "123",
		},
	}
}

func mockInventoryObject(t *testing.T) *Inventory {
	t.Helper()
	repoRoot := "a/repo/root"
	modulesInput := mockModuleList(t)
	return &Inventory{
		Lookup: map[string]modules.KaeterModule{
			"example.com:module1": {
				ModuleID:   "example.com:module1",
				ModulePath: "happiness/this/way",
				ModuleType: "nice",
			},
			"example.com:module2": {
				ModuleID:   "example.com:module2",
				ModulePath: "that/way",
				ModuleType: "nice",
				Annotations: map[string]string{
					"example.com/annotation1": "annotation1",
				},
			},
			"example.com:module3": {
				ModuleID:   "example.com:module3",
				ModulePath: "this/is/the/way",
				ModuleType: "nice",
				Annotations: map[string]string{
					"example.com/annotation1": "no",
					"example.com/annotation2": "yes",
				},
				AutoRelease: "123",
			},
		},
		ModuleInventory: ModuleInventory{
			RepoRoot: repoRoot,
			Modules:  modulesInput,
		},
	}
}

func shuffleHelper(t *testing.T, modulesList []modules.KaeterModule) []modules.KaeterModule {
	t.Helper()
	for i := range modulesList {
		j := rand.Intn(i + 1)
		modulesList[i], modulesList[j] = modulesList[j], modulesList[i]
	}
	return modulesList
}

func mockFullJSON(t *testing.T) string {
	t.Helper()
	stringJSON := `{
    "Modules": [
        {
            "id": "example.com:module1",
            "path": "happiness/this/way",
            "type": "nice"
        },
        {
            "id": "example.com:module2",
            "path": "that/way",
            "type": "nice",
            "annotations": {
                "example.com/annotation1": "annotation1"
            }
        },
        {
            "id": "example.com:module3",
            "path": "this/is/the/way",
            "type": "nice",
            "annotations": {
                "example.com/annotation1": "no",
                "example.com/annotation2": "yes"
            },
            "autoRelease": "123"
        }
    ],
    "RepoRoot": "a/repo/root"
}`
	return stringJSON
}
