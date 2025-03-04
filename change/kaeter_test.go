package change

import (
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

	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolderPath, _ := mocks.CreateMockRepo(t)
			testModulePath, _ := mocks.CreateKaeterModule(t, testFolderPath, &mocks.KaeterModuleConfig{
				Path:         tc.module.ModulePath,
				Makefile:     mocks.EmptyMakefileContent,
				VersionsYAML: mocks.EmptyVersionsYAML,
			})
			kaeterModules, err := modules.GetKaeterModules(testFolderPath)
			assert.NoError(t, err)
			t.Logf("mock modules: %v", kaeterModules)
			detector := &Detector{
				RootPath:      testFolderPath,
				KaeterModules: kaeterModules,
			}
			kc := KaeterChange{Modules: map[string]modules.KaeterModule{}}
			mocks.CreateMockFile(t, testModulePath, tc.module.ModuleType, tc.makefile)

			err = detector.checkModuleForChanges(&tests[i].module, &kc, tc.allTouchedFiles)

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
			name: "Expected only one module if no change in dependencies",
			modules: []modules.KaeterModule{
				{ModuleID: "ch.open.test:module", ModulePath: "module", ModuleType: "Makefile"},
				{ModuleID: "ch.open.test:module2", ModulePath: "module2", ModuleType: "Makefile", Dependencies: []string{"module"}},
			},
			allTouchedFiles: []string{"module2/blah.md"},
			makefile:        dummyMakefile,
			expectedModules: map[string]modules.KaeterModule{"ch.open.test:module2": {ModuleID: "ch.open.test:module2", ModulePath: "module2", ModuleType: "Makefile", Dependencies: []string{"module"}}},
		},
		{
			name: "Expected 2 modules if change in dependencies",
			modules: []modules.KaeterModule{
				{ModuleID: "ch.open.test:module", ModulePath: "module", ModuleType: "Makefile"},
				{ModuleID: "ch.open.test:module2", ModulePath: "module2", ModuleType: "Makefile", Dependencies: []string{"module"}},
			},
			allTouchedFiles: []string{"module/blah.md"},
			makefile:        dummyMakefile,
			expectedModules: map[string]modules.KaeterModule{
				"ch.open.test:module":  {ModuleID: "ch.open.test:module", ModulePath: "module", ModuleType: "Makefile"},
				"ch.open.test:module2": {ModuleID: "ch.open.test:module2", ModulePath: "module2", ModuleType: "Makefile", Dependencies: []string{"module"}},
			},
		},
		{
			name: "Expected 2 modules if change in dependencies is a file",
			modules: []modules.KaeterModule{
				{ModuleID: "ch.open.test:module", ModulePath: "module", ModuleType: "Makefile"},
				{ModuleID: "ch.open.test:module2", ModulePath: "module2", ModuleType: "Makefile", Dependencies: []string{"module/blah.md"}},
			},
			allTouchedFiles: []string{"module/blah.md"},
			makefile:        dummyMakefile,
			expectedModules: map[string]modules.KaeterModule{
				"ch.open.test:module":  {ModuleID: "ch.open.test:module", ModulePath: "module", ModuleType: "Makefile"},
				"ch.open.test:module2": {ModuleID: "ch.open.test:module2", ModulePath: "module2", ModuleType: "Makefile", Dependencies: []string{"module/blah.md"}},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolderPath, _ := mocks.CreateMockRepo(t)
			for _, module := range tc.modules {
				testModulePath, _ := mocks.CreateKaeterModule(t, testFolderPath, &mocks.KaeterModuleConfig{
					Path:         module.ModulePath,
					Makefile:     mocks.EmptyMakefileContent,
					VersionsYAML: mocks.GetEmptyVersionsYaml(t, module.ModuleID),
				})
				mocks.CreateMockFile(t, testModulePath, module.ModuleType, tc.makefile)
			}
			for _, file := range tc.allTouchedFiles {
				mocks.CreateMockFile(t, testFolderPath, file, "")
			}
			kc := KaeterChange{Modules: map[string]modules.KaeterModule{}}
			kaeterModules, err := modules.GetKaeterModules(testFolderPath)
			assert.NoError(t, err)
			t.Logf("mock modules: %v", kaeterModules)
			detector := &Detector{
				RootPath:      testFolderPath,
				KaeterModules: kaeterModules,
			}
			for i := range tc.modules {
				err = detector.checkModuleForChanges(&tc.modules[i], &kc, tc.allTouchedFiles)
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expectedModules, kc.Modules, tc.name)
		})
	}
}
