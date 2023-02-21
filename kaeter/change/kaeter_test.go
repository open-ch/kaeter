package change

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/open-ch/kaeter/kaeter/modules"
	"github.com/open-ch/kaeter/kaeter/pkg/mocks"
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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolderPath := mocks.CreateMockRepo(t)
			defer os.RemoveAll(testFolderPath)
			testModulePath := mocks.AddSubDirKaeterMock(t, testFolderPath, tc.module.ModulePath, mocks.EmptyVersionsYAML)
			kaeterModules, err := modules.GetKaeterModules(testFolderPath)
			assert.NoError(t, err)
			detector := &Detector{
				Logger:        logrus.New(),
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

func TestBazelTargetParsing(t *testing.T) {
	d := &Detector{
		Logger:   logrus.New(),
		RootPath: ".",
	}
	packageName := "//test/package"
	makeOutputs := []string{
		"# Check all containers can be built",
		"bazel build :release-bundle",
		"bazel query \"kind(container_push, deps(:release-bundle))\" | xargs -t -L1 bazel run --define DOCKER_TAG=snapshot",
	}
	result := d.extractBazelTargetsFromStrings(packageName, makeOutputs)
	assert.Equal(t, []string{"//test/package:release-bundle"}, result)

	packageName = "//web-mc"
	makeOutputs = []string{
		"echo building version \"snapshot\"",
		"bazel run --define DOCKER_TAG=\"snapshot\" --stamp //web-mc:publish_artifactory --verbose_failures",
		"bazel run --define DOCKER_TAG=\"snapshot\" --stamp //web-mc:publish --verbose_failures",
	}
	result = d.extractBazelTargetsFromStrings(packageName, makeOutputs)
	assert.Equal(t, []string{"//web-mc:publish", "//web-mc:publish_artifactory"}, result)
}

func TestListMakeCommands(t *testing.T) {
	var tests = []struct {
		makefileExtension string
		makefileContent   string
		expectedCommands  []string
	}{
		{
			makefileExtension: "Makefile",
			makefileContent:   ".PHONY: snapshot\nsnapshot:\n\t@echo Testing snapshot target",
			expectedCommands:  []string{"echo Testing snapshot target", ""},
		},
		{
			makefileExtension: "Makefile.kaeter",
			makefileContent:   ".PHONY: snapshot\nsnapshot:\n\t@echo Testing snapshot target",
			expectedCommands:  []string{"echo Testing snapshot target", ""},
		},
	}

	for _, tc := range tests {
		t.Run(tc.makefileExtension, func(t *testing.T) {
			detector := &Detector{
				Logger:   logrus.New(),
				RootPath: ".",
			}
			testFolder := mocks.CreateTmpFolder(t)
			defer os.RemoveAll(testFolder)
			mocks.CreateMockFile(t, testFolder, tc.makefileExtension, tc.makefileContent)
			testTarget := "snapshot"

			commandsList, err := detector.listMakeCommands(testFolder, testTarget)

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedCommands, commandsList, "Failed to read commands from Makefile")
		})
	}
}
