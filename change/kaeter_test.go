package change

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-ch/kaeter//mocks"
	"github.com/open-ch/kaeter//modules"
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
