package change

import (
	"io/ioutil"
	"os"
	"github.com/open-ch/kaeter/kaeter-ci/pkg/modules"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var dummyMakefile = []byte(".PHONY: snapshot release\nsnapshot:\n\t@echo Testing snapshot\nrelease:\n\t@echo Testing release")

func TestCheckMakefileTypeForChanges(t *testing.T) {
	var tests = []struct {
		name            string
		module          modules.KaeterModule
		allTouchedFiles []string
		info            Information
		makefile        []byte
		expectedModules map[string]modules.KaeterModule
	}{
		{
			name:            "Expected no module changes detected",
			module:          modules.KaeterModule{ModuleID: "ch.open.test:unit", ModulePath: "module", ModuleType: "Makefile"},
			allTouchedFiles: []string{"folder/blah.md"},
			info:            Information{},
			makefile:        dummyMakefile,
			expectedModules: map[string]modules.KaeterModule{},
		},
		{
			name:            "Expected bazel target with changes detected",
			module:          modules.KaeterModule{ModuleID: "ch.open.test:unit", ModulePath: "module", ModuleType: "Makefile"},
			allTouchedFiles: []string{},
			info:            Information{Bazel: BazelChange{Targets: []string{"//unit:test"}}},
			makefile:        []byte(".PHONY: snapshot release\nsnapshot:\n\tbazel run //unit:test\nrelease:\n\t@echo Testing release"),
			expectedModules: map[string]modules.KaeterModule{"ch.open.test:unit": {ModuleID: "ch.open.test:unit", ModulePath: "module", ModuleType: "Makefile"}},
		},
		{
			name:            "Expected matching path file changes detected",
			module:          modules.KaeterModule{ModuleID: "ch.open.test:unit", ModulePath: "module", ModuleType: "Makefile"},
			allTouchedFiles: []string{"module/blah.md"},
			info:            Information{},
			makefile:        dummyMakefile,
			expectedModules: map[string]modules.KaeterModule{"ch.open.test:unit": {ModuleID: "ch.open.test:unit", ModulePath: "module", ModuleType: "Makefile"}},
		},
	}

	for _, tc := range tests {
		testFolderPath := createTmpFolder(t)
		testModulePath := filepath.Join(testFolderPath, tc.module.ModulePath)
		defer os.RemoveAll(testFolderPath)
		err := os.Mkdir(testModulePath, 0755)
		assert.NoError(t, err)
		detector := &Detector{logrus.New(), testFolderPath, "commit1", "commit2"}
		kc := KaeterChange{Modules: map[string]modules.KaeterModule{}}
		createMockFile(t, testModulePath, tc.module.ModuleType, tc.makefile)

		detector.checkMakefileTypeForChanges(&tc.module, &kc, &tc.info, tc.allTouchedFiles)

		assert.Equal(t, tc.expectedModules, kc.Modules, tc.name)
	}

}

func TestBazelTargetParsing(t *testing.T) {
	d := &Detector{logrus.New(), ".", "commit1", "commit2"}
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

func TestMakefileTargetParsing(t *testing.T) {
	var tests = []struct {
		makefileExtension string
		makefileContent   []byte
		expectedCommands  []string
	}{
		{
			makefileExtension: "Makefile",
			makefileContent:   []byte(".PHONY: snapshot\nsnapshot:\n\t@echo Testing snapshot target"),
			expectedCommands:  []string{"echo Testing snapshot target", ""},
		},
		{
			makefileExtension: "Makefile.kaeter",
			makefileContent:   []byte(".PHONY: snapshot\nsnapshot:\n\t@echo Testing snapshot target"),
			expectedCommands:  []string{"echo Testing snapshot target", ""},
		},
	}

	for _, tc := range tests {
		detector := &Detector{logrus.New(), ".", "commit1", "commit2"}
		testFolder := createTmpFolder(t)
		defer os.RemoveAll(testFolder)
		createMockFile(t, testFolder, tc.makefileExtension, tc.makefileContent)
		testTarget := "snapshot"

		commandsList := detector.listMakeCommands(testFolder, testTarget)

		assert.Equal(t, tc.expectedCommands, commandsList, "Failed to read commands from Makefile")
	}
}

func createTmpFolder(t *testing.T) string {
	testFolderPath, err := os.MkdirTemp("", "kaeter-*")
	assert.NoError(t, err)

	return testFolderPath
}

func createMockFile(t *testing.T, tmpPath string, filename string, content []byte) {
	err := ioutil.WriteFile(filepath.Join(tmpPath, filename), content, 0644)
	assert.NoError(t, err)
}
