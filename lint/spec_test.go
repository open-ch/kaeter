package lint

import (
	"os"
	"path"
	"testing"

	"github.com/open-ch/kaeter/modules"

	"github.com/stretchr/testify/assert"
)

func TestFindSpecFile(t *testing.T) {
	tests := []struct {
		name             string
		expectedSpecFile string
		filesInModule    []string
		valid            bool
	}{
		{
			name:             "Returns spec file when one is found",
			expectedSpecFile: "something.spec",
			filesInModule:    []string{"something.spec"},
			valid:            true,
		},
		{
			name:             "Fails if no spec file exists",
			expectedSpecFile: "",
			filesInModule:    []string{"CHANGES", "CHANGELOG.md"},
			valid:            false,
		},
		{
			name:             "Returns first match if there are multiple",
			expectedSpecFile: "something1.spec",
			filesInModule:    []string{"something", "something1.spec", "something2.spec"},
			valid:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modulePath := t.TempDir()

			for _, file := range tt.filesInModule {
				err := os.WriteFile(path.Join(modulePath, file), []byte(""), 0644)
				assert.NoError(t, err)
			}

			specFile, err := findSpecFile(modulePath)

			if tt.valid {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedSpecFile, specFile)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestCheckSpecChangelog(t *testing.T) {
	versionsWithINIT := createMockVersions(t, []string{"1.0.0-1", "0.0.0"})
	versionsWithINIT[1].CommitID = modules.InitRef
	tests := []struct {
		name     string
		spec     string
		versions *modules.Versions
		valid    bool
	}{
		{
			name: "Pass if spec contains released versions",
			spec: `Name: testing-spec
Version: 1.0.0
%changelog
* Fri Aug 1 2042 author - 1.0.0-2
- FIX: Fixes the output to always be 42
* Fri Aug 1 2042 author - 1.0.0-1
- TRIVIAL: Initial version release
`,
			versions: &modules.Versions{ReleasedVersions: createMockVersions(t, []string{"1.0.0-1", "1.0.0-2"})},
			valid:    true,
		},
		{
			name: "Pass if spec contains released versions ignoring INIT version",
			spec: `Name: testing-spec
Version: 1.0.0
%changelog
* Fri Aug 1 2042 author - 1.0.0-1
- TRIVIAL: Initial version release
`,
			versions: &modules.Versions{ReleasedVersions: versionsWithINIT},
			valid:    true,
		},
		{
			name: "Pass for release with multiple authors and emails",
			spec: `Name: testing-spec
Version: 1.0.0
%changelog
* Fri Aug 1 2042 aut1 <aut1@example.com>, aut2 <aut2@example.com> - 1.0.0-1
- TRIVIAL: Initial version release
`,
			versions: &modules.Versions{ReleasedVersions: createMockVersions(t, []string{"1.0.0-1"})},
			valid:    true,
		},
		{
			name: "Pass for release with authors having emails containing dashes",
			spec: `Name: testing-spec
Version: 1.42.0
%changelog
* Fri Aug 1 2042 John Doe <jdoe@example-example.com>, aut2 <aut2@ex-am-ple.com> - 1.42.0-1
- TRIVIAL: testing emails containing a dash
`,
			versions: &modules.Versions{ReleasedVersions: createMockVersions(t, []string{"1.42.0-1"})},
			valid:    true,
		},
		{
			name: "Fail if kaeter version doesn't include -release",
			spec: `Name: testing-spec
Version: 1.0.0
%changelog
* Fri Aug 1 2042 author - 1.0.0-1
- TRIVIAL: Initial version release
`,
			versions: &modules.Versions{ReleasedVersions: createMockVersions(t, []string{"1.0.0"})},
			valid:    false,
		},
		{
			name: "Fail if spec contains invalid version formatting",
			spec: `Name: testing-spec
Version: 1.0.0
%changelog
* 01.08.2042 author 1.0.0-1
- TRIVIAL: Initial version release
`,
			versions: &modules.Versions{ReleasedVersions: createMockVersions(t, []string{"1.0.0-1"})},
			valid:    false,
		},
		{
			name: "Pass if spec ok without releases",
			spec: `Name: testing-spec
Version: 1.0.0
%changelog
`,
			versions: &modules.Versions{},
			valid:    true,
		},
		{
			name: "Fails if spec file doesn't have released versions",
			spec: `Name: testing-spec
Version: 1.0.0
%changelog
`,
			versions: &modules.Versions{ReleasedVersions: createMockVersions(t, []string{"1.0.0-1"})},
			valid:    false,
		},
		{
			name:     "Fails if spec file has no %changelog section",
			spec:     `# Empty spec`,
			versions: &modules.Versions{},
			valid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			specFile, err := os.CreateTemp(t.TempDir(), "test.spec")
			assert.NoError(t, err)
			_, err = specFile.WriteString(tt.spec)
			assert.NoError(t, err)
			specPath := specFile.Name()
			t.Logf("tmp specPath: %s (comment out the defer os.Remove to keep file after the tests)", specPath)

			err = checkSpecChangelog(specPath, tt.versions)

			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
