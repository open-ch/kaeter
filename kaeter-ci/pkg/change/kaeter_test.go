package change

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var dummyMakefile = []byte(".PHONY: snapshot release\nsnapshot:\n\t@echo Testing snapshot\nrelease:\n\t@echo Testing release")

func TestCheckMakefileTypeForChanges(t *testing.T) {
	var tests = []struct {
		name            string
		module          kaeterModule
		allTouchedFiles []string
		info            Information
		makefile        []byte
		expectedModules map[string]kaeterModule
	}{
		{
			name:            "Expected no module changes detected",
			module:          kaeterModule{ModuleID: "ch.open.test:unit", ModulePath: "module", ModuleType: "Makefile"},
			allTouchedFiles: []string{"folder/blah.md"},
			info:            Information{},
			makefile:        dummyMakefile,
			expectedModules: map[string]kaeterModule{},
		},
		{
			name:            "Expected bazel target with changes detected",
			module:          kaeterModule{ModuleID: "ch.open.test:unit", ModulePath: "module", ModuleType: "Makefile"},
			allTouchedFiles: []string{},
			info:            Information{Bazel: BazelChange{Targets: []string{"//unit:test"}}},
			makefile:        []byte(".PHONY: snapshot release\nsnapshot:\n\tbazel run //unit:test\nrelease:\n\t@echo Testing release"),
			expectedModules: map[string]kaeterModule{"ch.open.test:unit": kaeterModule{ModuleID: "ch.open.test:unit", ModulePath: "module", ModuleType: "Makefile"}},
		},
		{
			name:            "Expected matching path file changes detected",
			module:          kaeterModule{ModuleID: "ch.open.test:unit", ModulePath: "module", ModuleType: "Makefile"},
			allTouchedFiles: []string{"module/blah.md"},
			info:            Information{},
			makefile:        dummyMakefile,
			expectedModules: map[string]kaeterModule{"ch.open.test:unit": kaeterModule{ModuleID: "ch.open.test:unit", ModulePath: "module", ModuleType: "Makefile"}},
		},
	}

	for _, tc := range tests {
		testFolderPath := createTmpFolder(t)
		testModulePath := filepath.Join(testFolderPath, tc.module.ModulePath)
		defer os.RemoveAll(testFolderPath)
		err := os.Mkdir(testModulePath, 0755)
		assert.NoError(t, err)
		detector := New(logrus.InfoLevel, testFolderPath, "commit1", "commit2")
		kc := KaeterChange{Modules: map[string]kaeterModule{}}
		createMockFile(t, testModulePath, tc.module.ModuleType, tc.makefile)

		detector.checkMakefileTypeForChanges(&tc.module, &kc, &tc.info, tc.allTouchedFiles)

		assert.Equal(t, tc.expectedModules, kc.Modules, tc.name)
	}

}

func TestBazelTargetParsing(t *testing.T) {
	d := New(logrus.InfoLevel, ".", "commit1", "commit2")

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
		detector := New(logrus.InfoLevel, ".", "commit1", "commit2")
		testFolder := createTmpFolder(t)
		defer os.RemoveAll(testFolder)
		createMockFile(t, testFolder, tc.makefileExtension, tc.makefileContent)
		testTarget := "snapshot"

		commandsList := detector.listMakeCommands(testFolder, testTarget)

		assert.Equal(t, tc.expectedCommands, commandsList, "Failed to read commands from Makefile")
	}
}

func TestVersionsParsing(t *testing.T) {
	var tests = []struct {
		name           string
		versionsYAML   []byte
		expectedModule kaeterModule
		expectedError  bool
	}{
		{
			name: "Expect valid versions.yaml to be parsed",
			versionsYAML: []byte(`id: ch.open.tools:kaeter-ci
type: Makefile
versioning: SemVer
versions:
    0.0.0: 1970-01-01T00:00:00Z|INIT
`),
			expectedModule: kaeterModule{ModuleID: "ch.open.tools:kaeter-ci", ModulePath: "module", ModuleType: "Makefile"},
		},
		{
			name:           "Expect invalid versions.yaml to fail with error",
			versionsYAML:   []byte(`]]clearly__ not yaml [[`),
			expectedError:  true,
			expectedModule: kaeterModule{ModulePath: "module"},
		},
		{
			name: "Expect annotations to be parsed when available",
			versionsYAML: []byte(`id: ch.open.osix.pkg:OSAGhello
type: Makefile
versioning: SemVer
metadata:
    annotations:
        SCRUBBED-URL"true"
        SCRUBBED-URLqueue=osrp-dev
versions:
    0.0.0: 1970-01-01T00:00:00Z|INIT
`),
			expectedModule: kaeterModule{
				ModuleID:    "ch.open.osix.pkg:OSAGhello",
				ModulePath:  "module",
				ModuleType:  "Makefile",
				Annotations: map[string]string{"SCRUBBED-URL: "true", "SCRUBBED-URL: "queue=osrp-dev"},
			},
		},
	}

	for _, tc := range tests {
		testFolderPath := createTmpFolder(t)
		defer os.RemoveAll(testFolderPath)
		detector := New(logrus.InfoLevel, testFolderPath, "commit1", "commit2")
		testModulePath := filepath.Join(testFolderPath, tc.expectedModule.ModulePath)
		err := os.Mkdir(testModulePath, 0755)
		assert.NoError(t, err)
		createMockFile(t, testModulePath, "versions.yaml", tc.versionsYAML)

		module, err := detector.readKaeterModuleInfo(filepath.Join(testModulePath, "versions.yaml"))

		if tc.expectedError {
			assert.Error(t, err, tc.name)
		} else {
			assert.NoError(t, err, tc.name)
			assert.Equal(t, tc.expectedModule, module, tc.name)
		}

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
