package change

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-ch/kaeter/mocks"
	"github.com/open-ch/kaeter/modules"
)

var dummyMakefile = ".PHONY: snapshot release\nsnapshot:\n\t@echo Testing snapshot\nrelease:\n\t@echo Testing release"

func TestCheckModuleForChanges(t *testing.T) {
	var tests = []struct {
		name            string
		module          modules.KaeterModule
		allTouchedFiles []string
		makefile        string
		expectedModules map[string]modules.KaeterModule
	}{
		{
			name:            "Expected no module changes detected",
			module:          modules.KaeterModule{ModuleID: "ch.open.test:unit", ModulePath: "module", ModuleType: "Makefile"},
			allTouchedFiles: []string{"folder/blah.md"},
			makefile:        dummyMakefile,
			expectedModules: map[string]modules.KaeterModule{},
		},
		{
			name:            "Expected matching path file changes detected",
			module:          modules.KaeterModule{ModuleID: "ch.open.test:unit", ModulePath: "module", ModuleType: "Makefile"},
			allTouchedFiles: []string{"module/blah.md"},
			makefile:        dummyMakefile,
			expectedModules: map[string]modules.KaeterModule{"ch.open.test:unit": {ModuleID: "ch.open.test:unit", ModulePath: "module", ModuleType: "Makefile"}},
		},
		{
			name:            "Expect root module with changes is detected",
			module:          modules.KaeterModule{ModuleID: "ch.open.test:unit", ModulePath: ".", ModuleType: "Makefile"},
			allTouchedFiles: []string{"blah.md"},
			makefile:        dummyMakefile,
			expectedModules: map[string]modules.KaeterModule{"ch.open.test:unit": {ModuleID: "ch.open.test:unit", ModulePath: ".", ModuleType: "Makefile"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolderPath := mocks.CreateMockRepo(t)
			defer os.RemoveAll(testFolderPath)
			t.Logf("Temp folder: %s\n(disable `defer os.RemoveAll(testFolder)` to keep for debugging)\n", testFolderPath)
			testModulePath := mocks.AddSubDirKaeterMock(t, testFolderPath, tc.module.ModulePath, mocks.EmptyVersionsYAML)
			kaeterModules, err := modules.GetKaeterModules(testFolderPath)
			assert.NoError(t, err)
			t.Logf("mock modules: %s", kaeterModules)
			detector := &Detector{
				RootPath:      testFolderPath,
				KaeterModules: kaeterModules,
			}
			kc := KaeterChange{Modules: map[string]modules.KaeterModule{}}
			mocks.CreateMockFile(t, testModulePath, tc.module.ModuleType, tc.makefile)

			err = detector.checkModuleForChanges(&tc.module, &kc, tc.allTouchedFiles)

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedModules, kc.Modules, tc.name)
		})
	}
}

func TestModuleDependencies(t *testing.T) {
	var tests = []struct {
		name            string
		modules         []modules.KaeterModule
		allTouchedFiles []string
		makefile        string
		expectedModules map[string]modules.KaeterModule
	}{
		{
			name:            "Expected only one module if no change in dependencies",
			modules:         []modules.KaeterModule{{ModuleID: "ch.open.test:module", ModulePath: "module", ModuleType: "Makefile"}, {ModuleID: "ch.open.test:module2", ModulePath: "module2", ModuleType: "Makefile", Dependencies: []string{"module"}}},
			allTouchedFiles: []string{"module2/blah.md"},
			makefile:        dummyMakefile,
			expectedModules: map[string]modules.KaeterModule{"ch.open.test:module2": {ModuleID: "ch.open.test:module2", ModulePath: "module2", ModuleType: "Makefile", Dependencies: []string{"module"}}},
		},
		{
			name:            "Expected 2 modules if change in dependencies",
			modules:         []modules.KaeterModule{{ModuleID: "ch.open.test:module", ModulePath: "module", ModuleType: "Makefile"}, {ModuleID: "ch.open.test:module2", ModulePath: "module2", ModuleType: "Makefile", Dependencies: []string{"module"}}},
			allTouchedFiles: []string{"module/blah.md"},
			makefile:        dummyMakefile,
			expectedModules: map[string]modules.KaeterModule{
				"ch.open.test:module":  {ModuleID: "ch.open.test:module", ModulePath: "module", ModuleType: "Makefile"},
				"ch.open.test:module2": {ModuleID: "ch.open.test:module2", ModulePath: "module2", ModuleType: "Makefile", Dependencies: []string{"module"}},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolderPath := mocks.CreateMockRepo(t)
			defer os.RemoveAll(testFolderPath)
			t.Logf("Temp folder: %s\n(disable `defer os.RemoveAll(testFolder)` to keep for debugging)\n", testFolderPath)
			// Create all modules
			for _, module := range tc.modules {
				testModulePath := mocks.AddSubDirKaeterMock(t, testFolderPath, module.ModulePath, mocks.EmptyVersionsYAML)
				mocks.CreateMockFile(t, testModulePath, module.ModuleType, tc.makefile)
			}
			kc := KaeterChange{Modules: map[string]modules.KaeterModule{}}
			kaeterModules, err := modules.GetKaeterModules(testFolderPath)
			assert.NoError(t, err)
			t.Logf("mock modules: %s", kaeterModules)
			detector := &Detector{
				RootPath:      testFolderPath,
				KaeterModules: kaeterModules,
			}
			for _, module := range tc.modules {
				err = detector.checkModuleForChanges(&module, &kc, tc.allTouchedFiles)
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expectedModules, kc.Modules, tc.name)
		})
	}
}
