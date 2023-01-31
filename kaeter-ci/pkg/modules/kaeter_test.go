package modules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-ch/kaeter/kaeter/pkg/mocks"
)

// TODO test GetKaeterModules

func TestReadKaeterModuleInfo(t *testing.T) {
	var tests = []struct {
		name           string
		versionsYAML   string
		expectedModule KaeterModule
		expectedError  bool
	}{
		{
			name: "Expect valid versions.yaml to be parsed",
			versionsYAML: `id: ch.open.tools:kaeter-ci
type: Makefile
versioning: SemVer
versions:
    0.0.0: 1970-01-01T00:00:00Z|INIT
`,
			expectedModule: KaeterModule{ModuleID: "ch.open.tools:kaeter-ci", ModulePath: "module", ModuleType: "Makefile"},
		},
		{
			name:           "Expect invalid versions.yaml to fail with error",
			versionsYAML:   `]]clearly__ not yaml [[`,
			expectedError:  true,
			expectedModule: KaeterModule{ModulePath: "module"},
		},
		{
			name: "Expect annotations to be parsed when available",
			versionsYAML: `id: ch.open.osix.pkg:OSAGhello
type: Makefile
versioning: SemVer
metadata:
    annotations:
        SCRUBBED-URL "true"
        SCRUBBED-URL queue=osrp-dev
versions:
    0.0.0: 1970-01-01T00:00:00Z|INIT
`,
			expectedModule: KaeterModule{
				ModuleID:    "ch.open.osix.pkg:OSAGhello",
				ModulePath:  "module",
				ModuleType:  "Makefile",
				Annotations: map[string]string{"SCRUBBED-URL": "true", "SCRUBBED-URL": "queue=osrp-dev"},
			},
		},
		{
			name: "Detects auto release version",
			versionsYAML: `id: ch.open.tools:unit-test
type: Makefile
versioning: SemVer
versions:
    0.0.0: 1970-01-01T00:00:00Z|INIT
    1.0.0: 1997-08-29T02:14:00Z|AUTORELEASE
`,
			expectedModule: KaeterModule{
				ModuleID:    "ch.open.tools:unit-test",
				ModulePath:  "module",
				ModuleType:  "Makefile",
				AutoRelease: "1.0.0",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testFolder := mocks.CreateMockRepo(t)
			defer os.RemoveAll(testFolder)
			absModulePath := mocks.AddSubDirKaeterMock(t, testFolder, tc.expectedModule.ModulePath, tc.versionsYAML)

			module, err := readKaeterModuleInfo(filepath.Join(absModulePath, "versions.yaml"), testFolder)

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedModule, module)
			}
		})
	}
}
