package modules

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestVersionsParsing(t *testing.T) {
	var tests = []struct {
		name           string
		versionsYAML   []byte
		expectedModule KaeterModule
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
			expectedModule: KaeterModule{ModuleID: "ch.open.tools:kaeter-ci", ModulePath: "module", ModuleType: "Makefile"},
		},
		{
			name:           "Expect invalid versions.yaml to fail with error",
			versionsYAML:   []byte(`]]clearly__ not yaml [[`),
			expectedError:  true,
			expectedModule: KaeterModule{ModulePath: "module"},
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
			expectedModule: KaeterModule{
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
		testModulePath := filepath.Join(testFolderPath, tc.expectedModule.ModulePath)
		err := os.Mkdir(testModulePath, 0755)
		assert.NoError(t, err)

		createMockFile(t, testModulePath, "versions.yaml", tc.versionsYAML)

		module, err := readKaeterModuleInfo(filepath.Join(testModulePath, "versions.yaml"), testFolderPath)

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
